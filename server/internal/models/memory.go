package models

import (
	"time"
)

// Memory represents a stored memory point
// This is the database model for StoryMemory
type Memory struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"` // "decision", "character", "event"
	Content   string    `json:"content"`
	VectorID  string    `json:"vector_id"` // Qdrant vector ID
	Metadata  string    `json:"-"` // Serialized metadata
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// GetMetadata returns the deserialized metadata map
func (m *Memory) GetMetadata() map[string]interface{} {
	// TODO: Implement deserialization
	return make(map[string]interface{})
}

// IsExpired checks if the memory has expired
func (m *Memory) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}
