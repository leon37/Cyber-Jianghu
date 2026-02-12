package interfaces

import "context"

// MemoryType represents the type of memory
type MemoryType string

const (
	MemoryDecision MemoryType = "decision" // 用户决策
	MemoryCharacter MemoryType = "character" // 角色状态
	MemoryEvent    MemoryType = "event" // 关键事件
)

// Memory represents a stored memory with vector embedding
type Memory struct {
	ID        string
	SessionID string    // 直播会话ID
	Type      MemoryType // "decision" | "character" | "event"
	Content   string    // 记忆文本
	Metadata  map[string]interface{} // 扩展元数据
	Embedding []float64 // 向量表示
	Timestamp int64
}

// VectorStore defines the interface for vector database operations
type VectorStore interface {
	// StoreMemory stores a memory with its embedding
	StoreMemory(ctx context.Context, memory *Memory) error

	// SearchMemories searches for relevant memories by query
	SearchMemories(ctx context.Context, query string, limit int) ([]*Memory, error)

	// SearchMemoriesBySession searches memories within a session
	SearchMemoriesBySession(ctx context.Context, sessionID string, limit int) ([]*Memory, error)

	// UpdateMemory updates memory content and re-embeds
	UpdateMemory(ctx context.Context, memoryID string, updates map[string]interface{}) error

	// DeleteMemory removes a memory
	DeleteMemory(ctx context.Context, memoryID string) error

	// DeleteSessionMemories removes all memories for a session
	DeleteSessionMemories(ctx context.Context, sessionID string) error
}
