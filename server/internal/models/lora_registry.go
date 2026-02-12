package models

import (
	"time"
)

// LoRA represents a LoRA model for image generation
type LoRA struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	Style       string    `json:"style"` // "cyberpunk", "ancient", etc.
	Strength    float64   `json:"strength"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LoRAStyle represents predefined style presets
type LoRAStyle struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	LoRANames []string `json:"lora_names"` // Comma-separated LoRA names
	Prompt    string   `json:"prompt"`
	IsActive  bool     `gorm:"default:true" json:"is_active"`
}
