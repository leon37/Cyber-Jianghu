package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"Cyber-Jianghu/server/internal/engine"
	"Cyber-Jianghu/server/internal/generators"
	"Cyber-Jianghu/server/internal/rag"
)

// StoryHandlers handles story-related requests
type StoryHandlers struct {
	storyEngine   *engine.StoryEngine
	comfyClient   *generators.ComfyUIClient
	imageCache    *generators.ImageCache
	imageQueue    *generators.ImageQueue
}

// GenerateAudioRequest represents an audio generation request
type GenerateAudioRequest struct {
	Text    string `json:"text"`
	VoiceID string `json:"voice_id,omitempty"`
}

// GenerateAudioResponse represents an audio generation response
type GenerateAudioResponse struct {
	Success     bool   `json:"success"`
	AudioBase64 string `json:"audio_base64,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GenerateImageRequest represents an image generation request
type GenerateImageRequest struct {
	Prompt        string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	Width         int     `json:"width,omitempty"`
	Height        int     `json:"height,omitempty"`
	Steps         int     `json:"steps,omitempty"`
	CFGScale      float64 `json:"cfg_scale,omitempty"`
	Model         string  `json:"model,omitempty"`
}

// GenerateImageResponse represents an image generation response
type GenerateImageResponse struct {
	Success     bool   `json:"success"`
	ImageBase64 string `json:"image_base64,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GetVoicesResponse represents the response for listing voices
type GetVoicesResponse struct {
	Success bool                `json:"success"`
	Voices  []*generators.VoiceModel `json:"voices"`
	Default *generators.VoiceModel    `json:"default"`
	Error   string               `json:"error,omitempty"`
}

// SetDefaultVoiceRequest represents a request to set default voice
type SetDefaultVoiceRequest struct {
	VoiceID string `json:"voice_id"`
}

// NewStoryHandlers creates a new story handlers instance
func NewStoryHandlers(storyEngine *engine.StoryEngine, comfyClient *generators.ComfyUIClient, imageCacheDir string) *StoryHandlers {
	imageCache := generators.NewImageCache(imageCacheDir, 200, 24*time.Hour)
	imageQueue := generators.NewImageQueue(2) // 2 concurrent workers

	return &StoryHandlers{
		storyEngine: storyEngine,
		comfyClient: comfyClient,
		imageCache:  imageCache,
		imageQueue:  imageQueue,
	}
}

// CreateStoryRequest represents a story creation request
type CreateStoryRequest struct {
	Genre       string `json:"genre"`
	Tone        string `json:"tone"`
	Style       string `json:"style"`
	Protagonist string `json:"protagonist"`
}

// CreateStoryResponse represents a story creation response
type CreateStoryResponse struct {
	Success     bool          `json:"success"`
	Message     string        `json:"message,omitempty"`
	Story       *engine.Story `json:"story,omitempty"`
	AudioBase64 string        `json:"audio_base64,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// ContinueStoryRequest represents a story continuation request
type ContinueStoryRequest struct {
	StoryID string `json:"story_id"`
	Action  string `json:"action"`
}

// SelectOptionRequest represents an option selection request
type SelectOptionRequest struct {
	StoryID    string `json:"story_id"`
	OptionID   string `json:"option_id"`
	ChoiceText string `json:"choice_text"`
}

// CreateStory creates a new story
func (h *StoryHandlers) CreateStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CreateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if h.storyEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Story engine not initialized",
		})
		return
	}

	// Create a new story state
	storyID := "demo_story"

	// Initialize story state
	settings := map[string]interface{}{
		"genre":       req.Genre,
		"tone":        req.Tone,
		"style":       req.Style,
		"protagonist": req.Protagonist,
	}

	_, err := h.storyEngine.CreateStory(r.Context(), storyID, settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get the current state for response
	currentState, _ := h.storyEngine.GetStoryState(storyID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CreateStoryResponse{
		Success: true,
		Story: &engine.Story{
			ID:      storyID,
			State:   currentState,
			Content: currentState.PreviousText,
			Options: currentState.Options,
		},
	})
}

// ContinueStory continues a story with an action
func (h *StoryHandlers) ContinueStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ContinueStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if h.storyEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Story engine not initialized",
		})
		return
	}

	// Convert action to rag.Memory format
	inputMemory := rag.Memory{
		ID:        rag.BuildMemoryID(rag.MemoryTypePlayerAction, req.StoryID),
		Type:      rag.MemoryTypePlayerAction,
		Content:   req.Action,
		Timestamp: 0,
		StoryID:   req.StoryID,
		Metadata:  map[string]interface{}{
			"action": req.Action,
		},
	}

	// Generate story segment
	response, err := h.storyEngine.GenerateStorySegment(r.Context(), req.StoryID, req.Action, inputMemory)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get current state
	currentState, _ := h.storyEngine.GetStoryState(req.StoryID)

	// Generate audio for the story text (async to avoid blocking)
	// Audio is cached by the engine, so subsequent requests will be fast
	go func() {
		_, err := h.storyEngine.GenerateAudio(r.Context(), response.Text, "")
		if err != nil {
			log.Printf("Failed to generate audio: %v", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CreateStoryResponse{
		Success: true,
		Story: &engine.Story{
			ID:      req.StoryID,
			State:   currentState,
			Content: response.Text,
			Options: response.Options,
		},
	})
}

// SelectOption applies a selected option to the story
func (h *StoryHandlers) SelectOption(w http.ResponseWriter, r *http.Request) {
	log.Printf("SelectOption called")

	w.Header().Set("Content-Type", "application/json")

	var req SelectOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("SelectOption: Failed to decode request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	log.Printf("SelectOption: story_id=%s, option_id=%s, choice_text=%s", req.StoryID, req.OptionID, req.ChoiceText)

	if h.storyEngine == nil {
		log.Printf("SelectOption: Story engine is nil")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Story engine not initialized",
		})
		return
	}

	// Apply the selected option
	response, err := h.storyEngine.ApplyOption(r.Context(), req.StoryID, req.OptionID, req.ChoiceText)
	if err != nil {
		log.Printf("SelectOption: ApplyOption failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get current state
	currentState, err := h.storyEngine.GetStoryState(req.StoryID)
	if err != nil {
		log.Printf("SelectOption: GetStoryState failed: %v", err)
		// Even if we can't get state, return the response with partial info
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: true,
			Story: &engine.Story{
				ID:      req.StoryID,
				State:   nil,
				Content: response.Text,
				Options: response.Options,
			},
		})
		return
	}

	log.Printf("SelectOption: Success, returning story with %d options", len(response.Options))

	// Generate audio for the story text (async to avoid blocking)
	// Audio is cached by the engine, so subsequent requests will be fast
	go func() {
		_, err := h.storyEngine.GenerateAudio(r.Context(), response.Text, "")
		if err != nil {
			log.Printf("Failed to generate audio: %v", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CreateStoryResponse{
		Success: true,
		Story: &engine.Story{
			ID:      req.StoryID,
			State:   currentState,
			Content: response.Text,
			Options: response.Options,
		},
	})
}

// GetStoryStatus returns the current story status
func (h *StoryHandlers) GetStoryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	storyID := r.URL.Query().Get("story_id")
	if storyID == "" {
		storyID = "demo_story"
	}

	if h.storyEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Story engine not initialized",
		})
		return
	}

	// Get story state
	state, err := h.storyEngine.GetStoryState(storyID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(CreateStoryResponse{
			Success: false,
			Error:   "Story not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CreateStoryResponse{
		Success: true,
		Story: &engine.Story{
			ID:      storyID,
			State:   state,
			Content: state.PreviousText,
			Options: state.Options,
		},
	})
}

// GenerateAudio generates audio for given text
func (h *StoryHandlers) GenerateAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req GenerateAudioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GenerateAudioResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if h.storyEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(GenerateAudioResponse{
			Success: false,
			Error:   "Story engine not initialized",
		})
		return
	}

	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GenerateAudioResponse{
			Success: false,
			Error:   "Text is required",
		})
		return
	}

	// Generate audio
	audioData, err := h.storyEngine.GenerateAudio(r.Context(), req.Text, req.VoiceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GenerateAudioResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Convert to base64
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GenerateAudioResponse{
		Success:     true,
		AudioBase64: audioBase64,
	})
}

// GenerateImage generates an image from prompt
func (h *StoryHandlers) GenerateImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req GenerateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GenerateImageResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if h.comfyClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(GenerateImageResponse{
			Success: false,
			Error:   "ComfyUI client not initialized",
		})
		return
	}

	if req.Prompt == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GenerateImageResponse{
			Success: false,
			Error:   "Prompt is required",
		})
		return
	}

	// Build options
	opts := &generators.GenerateOptions{
		Prompt:        req.Prompt,
		NegativePrompt: req.NegativePrompt,
		Width:         req.Width,
		Height:        req.Height,
		Steps:         req.Steps,
		CFGScale:      req.CFGScale,
		Model:         req.Model,
		SamplerName:   "euler",
		Scheduler:     "normal",
	}

	// Check cache first
	cacheKey := generators.GenerateCacheKey(req.Prompt, opts)
	imageData, err := h.imageCache.Get(r.Context(), cacheKey)
	if err == nil {
		// Cache hit
		imageBase64 := base64.StdEncoding.EncodeToString(imageData)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GenerateImageResponse{
			Success:     true,
			ImageBase64: imageBase64,
		})
		return
	}

	// Cache miss - generate new image
	result, err := h.comfyClient.GenerateImage(r.Context(), opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GenerateImageResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Store in cache (async)
	go func() {
		_ = h.imageCache.Put(context.Background(), cacheKey, result.ImageData, req.Prompt, opts)
	}()

	// Return result
	imageBase64 := base64.StdEncoding.EncodeToString(result.ImageData)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GenerateImageResponse{
		Success:     true,
		ImageBase64: imageBase64,
	})
}

// GetVoices returns all available voices
func (h *StoryHandlers) GetVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get voice registry from story engine
	voices := h.storyEngine.GetAvailableVoices()
	defaultVoice, err := h.storyEngine.GetDefaultVoice()
	if err != nil {
		// No default set, continue
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetVoicesResponse{
		Success: true,
		Voices:  voices,
		Default: defaultVoice,
	})
}

// SetDefaultVoice sets the default voice
func (h *StoryHandlers) SetDefaultVoice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SetDefaultVoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GetVoicesResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if err := h.storyEngine.SetDefaultVoice(req.VoiceID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(GetVoicesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetVoicesResponse{
		Success: true,
	})
}
