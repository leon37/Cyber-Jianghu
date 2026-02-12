package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LoRAModel represents a LoRA model
type LoRAModel struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Path          string                 `json:"path"`
	Type          string                 `json:"type"` // "character", "style", "scene"
	CharacterName string                 `json:"character_name,omitempty"`
	Style         string                 `json:"style,omitempty"`
	Description   string                 `json:"description"`
	Strength      float64                `json:"strength"` // Default strength
	FileSize      int64                  `json:"file_size"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Enabled       bool                   `json:"enabled"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// LoRARegistry manages LoRA models
type LoRARegistry struct {
	models    map[string]*LoRAModel
	directory string
	mu        sync.RWMutex
}

// NewLoRARegistry creates a new LoRA registry
func NewLoRARegistry(directory string) *LoRARegistry {
	return &LoRARegistry{
		models:    make(map[string]*LoRAModel),
		directory: directory,
	}
}

// LoadModels scans the directory and loads LoRA models
func (r *LoRARegistry) LoadModels(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure directory exists
	if _, err := os.Stat(r.directory); os.IsNotExist(err) {
		// Create directory
		if err := os.MkdirAll(r.directory, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil
	}

	// Scan directory
	entries, err := os.ReadDir(r.directory)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Clear existing models
	r.models = make(map[string]*LoRAModel)

	// Load models
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check file extension
		ext := filepath.Ext(entry.Name())
		if ext != ".safetensors" {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Create model entry
		model := &LoRAModel{
			ID:       generateModelID(entry.Name()),
			Name:     entry.Name()[:len(entry.Name())-len(ext)],
			Path:     filepath.Join(r.directory, entry.Name()),
			Type:     inferModelType(entry.Name()),
			Strength: 0.8,
			FileSize: info.Size(),
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
			Enabled:   true,
			Metadata:  make(map[string]interface{}),
		}

		// Add metadata from filename if available
		model.Metadata = parseMetadataFromFilename(entry.Name())

		r.models[model.ID] = model
	}

	return nil
}

// RegisterModel registers a new LoRA model
func (r *LoRARegistry) RegisterModel(model *LoRAModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already exists
	if _, exists := r.models[model.ID]; exists {
		return fmt.Errorf("model already exists: %s", model.ID)
	}

	// Set timestamps
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	r.models[model.ID] = model

	return nil
}

// GetModel retrieves a model by ID
func (r *LoRARegistry) GetModel(id string) (*LoRAModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", id)
	}

	// Return a copy
	modelCopy := *model
	if model.Metadata != nil {
		modelCopy.Metadata = make(map[string]interface{})
		for k, v := range model.Metadata {
			modelCopy.Metadata[k] = v
		}
	}

	return &modelCopy, nil
}

// GetModelByName retrieves a model by name
func (r *LoRARegistry) GetModelByName(name string) (*LoRAModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, model := range r.models {
		if model.Name == name {
			modelCopy := *model
			return &modelCopy, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", name)
}

// ListModels returns all models
func (r *LoRARegistry) ListModels() []*LoRAModel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*LoRAModel, 0, len(r.models))
	for _, model := range r.models {
		modelCopy := *model
		models = append(models, &modelCopy)
	}

	return models
}

// ListModelsByType returns models filtered by type
func (r *LoRARegistry) ListModelsByType(loraType string) []*LoRAModel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*LoRAModel, 0)
	for _, model := range r.models {
		if model.Type == loraType {
			modelCopy := *model
			models = append(models, &modelCopy)
		}
	}

	return models
}

// ListEnabledModels returns only enabled models
func (r *LoRARegistry) ListEnabledModels() []*LoRAModel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*LoRAModel, 0)
	for _, model := range r.models {
		if model.Enabled {
			modelCopy := *model
			models = append(models, &modelCopy)
		}
	}

	return models
}

// EnableModel enables a model
func (r *LoRARegistry) EnableModel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, ok := r.models[id]
	if !ok {
		return fmt.Errorf("model not found: %s", id)
	}

	model.Enabled = true
	model.UpdatedAt = time.Now()

	return nil
}

// DisableModel disables a model
func (r *LoRARegistry) DisableModel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, ok := r.models[id]
	if !ok {
		return fmt.Errorf("model not found: %s", id)
	}

	model.Enabled = false
	model.UpdatedAt = time.Now()

	return nil
}

// UpdateModelStrength updates the strength of a model
func (r *LoRARegistry) UpdateModelStrength(id string, strength float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, ok := r.models[id]
	if !ok {
		return fmt.Errorf("model not found: %s", id)
	}

	// Clamp strength to [0.0, 1.0]
	if strength < 0.0 {
		strength = 0.0
	}
	if strength > 1.0 {
		strength = 1.0
	}

	model.Strength = strength
	model.UpdatedAt = time.Now()

	return nil
}

// DeleteModel deletes a model
func (r *LoRARegistry) DeleteModel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, ok := r.models[id]
	if !ok {
		return fmt.Errorf("model not found: %s", id)
	}

	// Delete file
	if err := os.Remove(model.Path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	delete(r.models, id)

	return nil
}

// GetCharacterModel returns the LoRA model for a specific character
func (r *LoRARegistry) GetCharacterModel(characterName string) (*LoRAModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, model := range r.models {
		if model.Type == "character" && model.CharacterName == characterName {
			modelCopy := *model
			return &modelCopy, nil
		}
	}

	return nil, fmt.Errorf("character model not found: %s", characterName)
}

// ModelExists checks if a model exists
func (r *LoRARegistry) ModelExists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.models[id]
	return exists
}

// GetStats returns statistics about registered models
func (r *LoRARegistry) GetStats() *LoRAStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &LoRAStats{
		TotalCount:  len(r.models),
		EnabledCount: 0,
		ByType:       make(map[string]int),
	}

	for _, model := range r.models {
		if model.Enabled {
			stats.EnabledCount++
		}
		stats.ByType[model.Type]++
	}

	return stats
}

// LoRAStats holds statistics about LoRA models
type LoRAStats struct {
	TotalCount   int              `json:"total_count"`
	EnabledCount int              `json:"enabled_count"`
	ByType       map[string]int   `json:"by_type"`
	TotalSize    int64            `json:"total_size"`
}

// GetTotalSize returns the total size of all models
func (r *LoRARegistry) GetTotalSize() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total int64
	for _, model := range r.models {
		total += model.FileSize
	}

	return total
}

// Helper functions
func generateModelID(filename string) string {
	// Use filename as ID, removing extension
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)]
}

func inferModelType(filename string) string {
	lower := filename

	// Character models usually contain character names
	// Style models usually have "style" in the name
	// Scene models usually have "scene" in the name

	if contains(lower, "character") || contains(lower, "char") {
		return "character"
	}

	if contains(lower, "style") || contains(lower, "artstyle") {
		return "style"
	}

	if contains(lower, "scene") || contains(lower, "background") {
		return "scene"
	}

	// Default to style
	return "style"
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func parseMetadataFromFilename(filename string) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Parse metadata from filename pattern
	// Example: "character_lishan_lora_v1.safetensors"
	// Extract character name, version, etc.

	// Simplified implementation
	lower := filename

	if contains(lower, "lishan") {
		metadata["character_name"] = "李山"
	}

	if contains(lower, "wuxia") {
		metadata["style"] = "武侠"
	}

	if contains(lower, "cyberpunk") {
		metadata["style"] = "赛博朋克"
	}

	return metadata
}
