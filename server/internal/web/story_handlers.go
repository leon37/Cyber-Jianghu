package web

import (
	"encoding/json"
	"log"
	"net/http"

	"Cyber-Jianghu/server/internal/engine"
	"Cyber-Jianghu/server/internal/rag"
)

// StoryHandlers handles story-related requests
type StoryHandlers struct {
	storyEngine *engine.StoryEngine
}

// NewStoryHandlers creates a new story handlers instance
func NewStoryHandlers(storyEngine *engine.StoryEngine) *StoryHandlers {
	return &StoryHandlers{
		storyEngine: storyEngine,
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
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Story   *engine.Story `json:"story,omitempty"`
	Error   string        `json:"error,omitempty"`
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
