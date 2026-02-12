package models

import (
	"time"

	"gorm.io/gorm"
)

// Story represents a game session/story
type Story struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:255" json:"title"`
	SessionID string         `gorm:"uniqueIndex;size:64" json:"session_id"`
	Status    string         `gorm:"size:32" json:"status"` // "active", "paused", "ended"
	CurrentScene string      `gorm:"size:128" json:"current_scene"`
	JSONContext string       `gorm:"type:text" json:"-"` // Serialized StoryContext
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// StoryDecision represents a user decision in the story
type StoryDecision struct {
	ID        string `gorm:"primaryKey" json:"id"`
	StoryID   string `gorm:"index" json:"story_id"`
	UserID    string `gorm:"size:64" json:"user_id"`
	Username  string `gorm:"size:128" json:"username"`
	Action    string `gorm:"type:text" json:"action"` // "/attack", "/talk", etc.
	Content   string `gorm:"type:text" json:"content"` // Danmaku content
	Selected  bool   `json:"selected"` // Whether this action was executed
	Timestamp int64  `json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

// StoryMemory represents a stored memory point
type StoryMemory struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	StoryID   string    `gorm:"index" json:"story_id"`
	SessionID string    `gorm:"index;size:64" json:"session_id"`
	Type      string    `gorm:"size:32" json:"type"` // "decision", "character", "event"
	Content   string    `gorm:"type:text" json:"content"`
	Metadata  string    `gorm:"type:text" json:"-"` // Serialized metadata
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
