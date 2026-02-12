package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Cyber-Jianghu/server/internal/engine"
	"Cyber-Jianghu/server/internal/interfaces"
)

// MemoryType represents the type of memory
type MemoryType string

const (
	MemoryTypePlayerAction MemoryType = "player_action"  // 玩家行为
	MemoryTypeStoryState  MemoryType = "story_state"   // 故事状态
	MemoryTypeNPC        MemoryType = "npc"           // NPC 交互
	MemoryTypeDecision   MemoryType = "decision"      // 玩家决策
)

// Memory represents a stored memory
type Memory struct {
	ID        string                 `json:"id"`
	Type      MemoryType            `json:"type"`
	Content   string                 `json:"content"`
	Timestamp int64                  `json:"timestamp"`
	StoryID   string                 `json:"story_id"`
	Metadata  map[string]interface{} `json:"metadata"`
	Vector    []float64               `json:"-"`
}

// DecisionMemory represents a player decision
type DecisionMemory struct {
	Memory
	OptionID   string `json:"option_id"`
	ChoiceText string `json:"choice_text"`
	Reason      string `json:"reason"`
}

// MemoryStore manages story memories with vector search
type MemoryStore struct {
	qdrantClient *QdrantClient
	embedding    *engine.EmbeddingService
	collection   string
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(qdrant *QdrantClient, embedding *engine.EmbeddingService) *MemoryStore {
	return &MemoryStore{
		qdrantClient: qdrant,
		embedding:    embedding,
		collection:   memoryCollectionName,
	}
}

// StoreMemory stores a memory with its embedding
func (s *MemoryStore) StoreMemory(ctx context.Context, memory *Memory) error {
	// Generate embedding
	vector, err := s.embedding.Embed(ctx, memory.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Prepare payload
	payload := map[string]interface{}{
		"type":      string(memory.Type),
		"content":   memory.Content,
		"timestamp": memory.Timestamp,
		"story_id":  memory.StoryID,
	}

	// Add metadata to payload
	for k, v := range memory.Metadata {
		payload[k] = v
	}

	// Create point
	point := &Point{
		ID:      memory.ID,
		Vector:  vector,
		Payload: payload,
	}

	// Store in Qdrant
	return s.qdrantClient.InsertPoint(ctx, s.collection, point)
}

// StoreDecision stores a player decision
func (s *MemoryStore) StoreDecision(ctx context.Context, decision *DecisionMemory) error {
	decision.Type = MemoryTypeDecision
	return s.StoreMemory(ctx, &decision.Memory)
}

// SearchRelatedMemories searches for memories related to a query
func (s *MemoryStore) SearchRelatedMemories(ctx context.Context, query string, limit int, memoryTypes []MemoryType) ([]*Memory, error) {
	// Generate query embedding
	queryVector, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Build search options
	opts := &SearchOptions{
		Limit:       limit,
		WithPayload: true,
		ScoreThreshold: 0.7, // Only return highly similar results
	}

	// Search
	results, err := s.qdrantClient.Search(ctx, s.collection, queryVector, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	// Convert to memories
	memories := make([]*Memory, 0, len(results))
	for _, result := range results {
		memory, err := s.resultToMemory(result)
		if err != nil {
			continue // Skip invalid results
		}

		// Filter by memory type if specified
		if len(memoryTypes) > 0 {
			found := false
			for _, mt := range memoryTypes {
				if memory.Type == mt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		memories = append(memories, memory)
	}

	return memories, nil
}

// SearchRecentDecisions searches for recent player decisions
func (s *MemoryStore) SearchRecentDecisions(ctx context.Context, storyID string, limit int) ([]*DecisionMemory, error) {
	queryVector, err := s.embedding.Embed(ctx, "player decision")
	if err != nil {
		return nil, err
	}

	opts := &SearchOptions{
		Limit:       limit,
		WithPayload: true,
	}

	// Add filter for story_id
	if storyID != "" {
		opts.Filter = &Filter{
			Must: []Condition{
				{
					Key:   "story_id",
					Match: storyID,
					Op:    "match",
				},
			},
		}
	}

	results, err := s.qdrantClient.Search(ctx, s.collection, queryVector, opts)
	if err != nil {
		return nil, err
	}

	decisions := make([]*DecisionMemory, 0, len(results))
	for _, result := range results {
		mem, err := s.resultToMemory(result)
		if err != nil {
			continue
		}

		if mem.Type == MemoryTypeDecision {
			decision := &DecisionMemory{
				Memory: *mem,
			}
			if optID, ok := mem.Metadata["option_id"].(string); ok {
				decision.OptionID = optID
			}
			if choiceText, ok := mem.Metadata["choice_text"].(string); ok {
				decision.ChoiceText = choiceText
			}
			if reason, ok := mem.Metadata["reason"].(string); ok {
				decision.Reason = reason
			}
			decisions = append(decisions, decision)
		}
	}

	return decisions, nil
}

// GetMemoriesByType retrieves memories by type
func (s *MemoryStore) GetMemoriesByType(ctx context.Context, memoryType MemoryType, storyID string, limit int) ([]*Memory, error) {
	// Generate a dummy query vector
	queryVector, err := s.embedding.Embed(ctx, string(memoryType))
	if err != nil {
		return nil, err
	}

	opts := &SearchOptions{
		Limit:       limit,
		WithPayload: true,
	}

	// Add filter for type
	conditions := []Condition{
		{
			Key:   "type",
			Match: string(memoryType),
			Op:    "match",
		},
	}

	// Add filter for story_id if specified
	if storyID != "" {
		conditions = append(conditions, Condition{
			Key:   "story_id",
			Match: storyID,
			Op:    "match",
		})
	}

	opts.Filter = &Filter{Must: conditions}

	results, err := s.qdrantClient.Search(ctx, s.collection, queryVector, opts)
	if err != nil {
		return nil, err
	}

	memories := make([]*Memory, 0, len(results))
	for _, result := range results {
		mem, err := s.resultToMemory(result)
		if err != nil {
			continue
		}
		if mem.Type == memoryType {
			memories = append(memories, mem)
		}
	}

	return memories, nil
}

// DeleteMemoriesByStory deletes all memories for a story
func (s *MemoryStore) DeleteMemoriesByStory(ctx context.Context, storyID string) error {
	// Note: Qdrant doesn't support filtering in delete directly
	// We would need to first search for IDs, then delete
	// For now, this is a simplified implementation
	return nil
}

// resultToMemory converts search result to Memory
func (s *MemoryStore) resultToMemory(result *SearchResult) (*Memory, error) {
	memType, ok := result.Payload["type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid memory type")
	}

	content, ok := result.Payload["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid memory content")
	}

	storyID, _ := result.Payload["story_id"].(string)
	timestamp, _ := result.Payload["timestamp"].(float64)

	return &Memory{
		ID:        result.ID,
		Type:      MemoryType(memType),
		Content:   content,
		Timestamp: int64(timestamp),
		StoryID:   storyID,
		Metadata:  result.Payload,
	}, nil
}

// MemoryStats holds statistics about stored memories
type MemoryStats struct {
	TotalCount      int64                    `json:"total_count"`
	ByType          map[MemoryType]int64    `json:"by_type"`
	RecentCount     int                      `json:"recent_count"`
}

// GetStats returns statistics about stored memories
func (s *MemoryStore) GetStats(ctx context.Context) (*MemoryStats, error) {
	// Get collection info
	info, err := s.qdrantClient.GetCollectionInfo(ctx, s.collection)
	if err != nil {
		return nil, err
	}

	return &MemoryStats{
		TotalCount:  int64(info.PointCount),
		ByType:      make(map[MemoryType]int64),
		RecentCount: 0, // Would need a timestamp filter
	}, nil
}

// BuildMemoryID generates a unique memory ID
func BuildMemoryID(memoryType MemoryType, storyID string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%s_%d", memoryType, storyID, timestamp)
}

// BuildContextSummary builds a summary of related memories
func (s *MemoryStore) BuildContextSummary(memories []*Memory, maxMemories int) string {
	if len(memories) == 0 {
		return "（无相关记忆）"
	}

	// Limit number of memories
	if len(memories) > maxMemories {
		memories = memories[:maxMemories]
	}

	var summary strings.Builder
	summary.WriteString("## 相关记忆与决策\n\n")

	for i, mem := range memories {
		summary.WriteString(fmt.Sprintf("%d. ", i+1))

		// Add type-specific formatting
		switch mem.Type {
		case MemoryTypePlayerAction:
			summary.WriteString(fmt.Sprintf("玩家行为: %s", mem.Content))
		case MemoryTypeStoryState:
			summary.WriteString(fmt.Sprintf("故事状态: %s", mem.Content))
		case MemoryTypeNPC:
			summary.WriteString(fmt.Sprintf("NPC交互: %s", mem.Content))
		case MemoryTypeDecision:
			summary.WriteString(fmt.Sprintf("玩家决策: %s", mem.Content))
		}

		// Add timestamp
		if mem.Timestamp > 0 {
			t := time.Unix(mem.Timestamp, 0)
			summary.WriteString(fmt.Sprintf(" (%s)", t.Format("15:04")))
		}

		summary.WriteString("\n")
	}

	return summary.String()
}
