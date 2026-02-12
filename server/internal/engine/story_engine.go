package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"Cyber-Jianghu/server/internal/interfaces"
	"Cyber-Jianghu/server/internal/prompts"
	"Cyber-Jianghu/server/internal/rag"
)

// StoryState represents the current state of the story
type StoryState struct {
	CurrentNode   string                 `json:"current_node"`
	CurrentScene  string                 `json:"current_scene"`
	PreviousText  string                 `json:"previous_text"`
	Summary       string                 `json:"summary"`
	Protagonist   string                 `json:"protagonist"`
	NPCs          string                 `json:"npcs"`
	Genre          string                 `json:"genre"`
	Tone           string                 `json:"tone"`
	Style          string                 `json:"style"`
	Options        []StoryOption         `json:"options"`
	Custom         map[string]interface{} `json:"custom"`
}

// StoryOption represents a player choice option
type StoryOption struct {
	ID          string                 `json:"id"`
	Text        string                 `json:"text"`
	Description string                 `json:"description"`
	NextNode    string                 `json:"next_node,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StoryResponse represents the response from story generation
type StoryResponse struct {
	Text           string                 `json:"text"`
	Scene          string                 `json:"scene"`
	Options        []StoryOption         `json:"options"`
	NextNode      string                 `json:"next_node,omitempty"`
	VisualPrompt   string                 `json:"visual_prompt,omitempty"`
	AudioPrompt    string                 `json:"audio_prompt,omitempty"`
	RelatedMemories []rag.Memory          `json:"related_memories,omitempty"`
}

// StoryEngine manages story generation and state
type StoryEngine struct {
	glm5Client    *GLM5Client
	embedService  *EmbeddingService
	memoryStore   *rag.MemoryStore
	promptEngine *prompts.TemplateEngine

	state        map[string]*StoryState
	mu           sync.RWMutex

	storyModel   string // GLM-5 model for story generation
	imageModel   string // Model for image prompt generation
}

// NewStoryEngine creates a new story engine
func NewStoryEngine(
	apiKey string,
	qdrantClient *rag.QdrantClient,
) *StoryEngine {
	glm5Client := NewGLM5Client(apiKey)
	embedService := NewEmbeddingService(apiKey)
	memoryStore := rag.NewMemoryStore(qdrantClient, embedService)
	promptEngine := prompts.NewTemplateEngine()

	// Initialize default templates
	_ = promptEngine.InitializeDefaultTemplates()

	return &StoryEngine{
		glm5Client:    glm5Client,
		embedService:  embedService,
		memoryStore:   memoryStore,
		promptEngine: promptEngine,
		state:        make(map[string]*StoryState),
		storyModel:   "glm-4",
		imageModel:   "embedding-3",
	}
}

// CreateStory creates a new story with initial settings
func (e *StoryEngine) CreateStory(ctx context.Context, storyID string, settings map[string]interface{}) (*StoryState, error) {
	// Extract settings
	protagonist, _ := settings["protagonist"].(string)
	genre, _ := settings["genre"].(string)
	tone, _ := settings["tone"].(string)
	style, _ := settings["style"].(string)

	// Default values
	if genre == "" {
		genre = "武侠"
	}
	if tone == "" {
		tone = "史诗"
	}
	if style == "" {
		style = "古典"
	}

	// Create initial state
	state := &StoryState{
		CurrentNode:  "start",
		CurrentScene: "故事开始",
		PreviousText: "",
		Summary:       fmt.Sprintf("%s主角 %s的故事开始了", genre, protagonist),
		Protagonist:   protagonist,
		NPCs:          "",
		Genre:          genre,
		Tone:           tone,
		Style:          style,
		Options:        []StoryOption{},
		Custom:         make(map[string]interface{}),
	}

	// Store state
	e.mu.Lock()
	e.state[storyID] = state
	e.mu.Unlock()

	// Generate initial story segment
	response, err := e.GenerateStorySegment(ctx, storyID, "", rag.Memory{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial story: %w", err)
	}

	// Update state with response
	state.CurrentScene = response.Scene
	state.PreviousText = response.Text
	state.Options = response.Options

	// Store initial memory
	initialMemory := &rag.Memory{
		ID:        rag.BuildMemoryID(rag.MemoryTypeStoryState, storyID),
		Type:      rag.MemoryTypeStoryState,
		Content:   state.Summary,
		Timestamp: time.Now().Unix(),
		StoryID:   storyID,
		Metadata:  map[string]interface{}{
			"genre":     genre,
			"protagonist": protagonist,
		},
	}

	// Use goroutine to avoid blocking
	go func() {
		_ = e.memoryStore.StoreMemory(context.Background(), initialMemory)
	}()

	return state, nil
}

// GenerateStorySegment generates a story segment based on player action
func (e *StoryEngine) GenerateStorySegment(
	ctx context.Context,
	storyID string,
	playerAction string,
	inputMemory rag.Memory,
) (*StoryResponse, error) {
	// Get current state
	state, err := e.GetStoryState(storyID)
	if err != nil {
		return nil, err
	}

	// Search for related memories
	relatedMemories, err := e.memoryStore.SearchRelatedMemories(
		ctx,
		playerAction,
		10, // Limit to 10 memories
		[]rag.MemoryType{rag.MemoryTypePlayerAction, rag.MemoryTypeDecision, rag.MemoryTypeNPC},
	)
	if err != nil {
		// Continue even if memory search fails
		relatedMemories = []*rag.Memory{}
	}

	// Search for related decisions
	relatedDecisions, err := e.memoryStore.SearchRecentDecisions(ctx, storyID, 5)
	if err != nil {
		relatedDecisions = []*rag.DecisionMemory{}
	}

	// Build story context
	storyCtx := prompts.BuildStoryContext(
		&interfaces.Story{
			CurrentScene: state.CurrentScene,
			PreviousText: state.PreviousText,
			Summary:       state.Summary,
			Protagonist:   state.Protagonist,
			NPCs:          state.NPCs,
			Genre:          state.Genre,
			Tone:           state.Tone,
			Style:          state.Style,
		},
		interfaces.Danmaku{
			Content: playerAction,
		},
		buildMemoryTexts(relatedMemories),
		buildDecisionTexts(relatedDecisions),
	)

	// Render prompt
	prompt, err := e.promptEngine.Render("story_continuation", storyCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Call GLM-5
	messages := []engine.ChatMessage{
		{Role: "user", Content: prompt},
	}

	req := &engine.ChatRequest{
		Messages:    messages,
		Model:       e.storyModel,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	resp, err := e.glm5Client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate story: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from model")
	}

	generatedText := resp.Choices[0].Message.Content

	// Parse response to extract options
	options := e.parseOptionsFromResponse(generatedText)

	// Generate visual prompt
	imageCtx := &prompts.ImagePromptContext{
		SceneDescription: e.extractSceneDescription(generatedText),
		Style:           state.Style,
		Mood:            state.Tone,
	}
	visualPrompt, _ := e.promptEngine.RenderImagePrompt("image_generation", imageCtx)

	// Update state
	e.mu.Lock()
	if currentState, ok := e.state[storyID]; ok {
		currentState.PreviousText = generatedText
		currentState.Options = options
	}
	e.mu.Unlock()

	// Store player action memory
	if playerAction != "" {
		actionMemory := &rag.Memory{
			ID:        rag.BuildMemoryID(rag.MemoryTypePlayerAction, storyID),
			Type:      rag.MemoryTypePlayerAction,
			Content:   playerAction,
			Timestamp: time.Now().Unix(),
			StoryID:   storyID,
			Metadata:  map[string]interface{}{
				"current_node": state.CurrentNode,
			},
		}
		_ = e.memoryStore.StoreMemory(ctx, actionMemory)
	}

	return &StoryResponse{
		Text:            generatedText,
		Scene:           e.extractSceneDescription(generatedText),
		Options:         options,
		VisualPrompt:    visualPrompt,
		RelatedMemories: relatedMemories,
	}, nil
}

// ApplyOption applies a player choice option
func (e *StoryEngine) ApplyOption(ctx context.Context, storyID, optionID string, choiceText string) (*StoryResponse, error) {
	// Store decision memory
	decision := &rag.DecisionMemory{
		Memory: rag.Memory{
			ID:        rag.BuildMemoryID(rag.MemoryTypeDecision, storyID),
			Type:      rag.MemoryTypeDecision,
			Content:   fmt.Sprintf("选择了选项 %s: %s", optionID, choiceText),
			Timestamp: time.Now().Unix(),
			StoryID:   storyID,
			Metadata:  map[string]interface{}{
				"option_id":   optionID,
				"choice_text": choiceText,
			},
		},
		OptionID:   optionID,
		ChoiceText: choiceText,
	}

	_ = e.memoryStore.StoreDecision(ctx, decision)

	// Generate next story segment
	return e.GenerateStorySegment(ctx, storyID, choiceText, decision.Memory)
}

// GetStoryState retrieves the current state of a story
func (e *StoryEngine) GetStoryState(storyID string) (*StoryState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.state[storyID]
	if !ok {
		return nil, fmt.Errorf("story not found: %s", storyID)
	}

	// Return a copy to avoid concurrent modification
	stateCopy := *state
	stateCopy.Options = append([]StoryOption{}, state.Options...)
	if state.Custom != nil {
		stateCopy.Custom = make(map[string]interface{})
		for k, v := range state.Custom {
			stateCopy.Custom[k] = v
		}
	}

	return &stateCopy, nil
}

// GetActiveStories returns list of active story IDs
func (e *StoryEngine) GetActiveStories() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stories := make([]string, 0, len(e.state))
	for id := range e.state {
		stories = append(stories, id)
	}
	return stories
}

// EndStory ends a story and optionally saves it
func (e *StoryEngine) EndStory(ctx context.Context, storyID string, save bool) error {
	state, err := e.GetStoryState(storyID)
	if err != nil {
		return err
	}

	// Store final state as memory
	finalMemory := &rag.Memory{
		ID:        rag.BuildMemoryID(rag.MemoryTypeStoryState, storyID),
		Type:      rag.MemoryTypeStoryState,
		Content:   fmt.Sprintf("故事结束。最后状态: %s", state.CurrentScene),
		Timestamp: time.Now().Unix(),
		StoryID:   storyID,
		Metadata:  map[string]interface{}{
			"final": true,
		},
	}
	_ = e.memoryStore.StoreMemory(ctx, finalMemory)

	// Remove from active states
	e.mu.Lock()
	delete(e.state, storyID)
	e.mu.Unlock()

	return nil
}

// parseOptionsFromResponse extracts options from generated text
func (e *StoryEngine) parseOptionsFromResponse(text string) []StoryOption {
	options := []StoryOption{}

	// Simple parsing - look for numbered options
	// In production, this would be more sophisticated
	optionPatterns := [][]string{
		{"1.", "2.", "3."},
		{"A.", "B.", "C."},
		{"一、", "二、", "三、"},
	}

	for _, patterns := range optionPatterns {
		foundAll := true
		for i, pattern := range patterns {
			if !containsSubstring(text, pattern) {
				foundAll = false
				break
			}

			// Extract option text (simplified)
			optionText := e.extractOptionText(text, pattern)
			if optionText != "" {
				options = append(options, StoryOption{
					ID:          fmt.Sprintf("%d", i+1),
					Text:        pattern + optionText,
					Description: optionText,
				})
			}
		}
		if foundAll && len(options) >= 2 {
			break
		}
	}

	// If no options found, provide default
	if len(options) == 0 {
		options = []StoryOption{
			{ID: "1", Text: "继续前进", Description: "继续探索当前场景"},
			{ID: "2", Text: "观察周围", Description: "仔细观察环境细节"},
			{ID: "3", Text: "询问NPC", Description: "与附近的人交谈"},
		}
	}

	return options
}

// extractSceneDescription extracts scene description from generated text
func (e *StoryEngine) extractSceneDescription(text string) string {
	// Simplified - just take first sentence or paragraph
	// In production, this would use NLP
	lines := splitLines(text, "\n")
	if len(lines) > 0 {
		// Return first substantial line
		for _, line := range lines {
			if len(line) > 10 {
				return line
			}
		}
		return lines[0]
	}
	return text
}

// extractOptionText extracts option text from response
func (e *StoryEngine) extractOptionText(text, prefix string) string {
	// Find prefix and extract text after it
	// Simplified implementation
	return ""
}

// Helper functions
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitLines(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func buildMemoryTexts(memories []*rag.Memory) []string {
	texts := make([]string, len(memories))
	for i, mem := range memories {
		texts[i] = fmt.Sprintf("[%s] %s", mem.Type, mem.Content)
	}
	return texts
}

func buildDecisionTexts(decisions []*rag.DecisionMemory) []string {
	texts := make([]string, len(decisions))
	for i, dec := range decisions {
		texts[i] = fmt.Sprintf("[决策] %s: %s", dec.OptionID, dec.ChoiceText)
	}
	return texts
}
