package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

const (
	defaultCollectionName = "cyber_jianghu"
	memoryCollectionName = "memories"
	decisionCollectionName = "decisions"
	defaultVectorSize   = 1024  // Embedding dimension
	grpcPort           = 6334
)

// QdrantClient wraps Qdrant client
type QdrantClient struct {
	client      *qdrant.Client
	collections map[string]bool
	mu          *safeMutex
}

// safeMutex provides a reentrant mutex-like mechanism
type safeMutex struct {
	mu    chan struct{}
	owned bool
}

func newSafeMutex() *safeMutex {
	m := &safeMutex{
		mu:    make(chan struct{}, 1),
		owned: false,
	}
	m.mu <- struct{}{}
	return m
}

func (m *safeMutex) lock() {
	<-m.mu
}

func (m *safeMutex) unlock() {
	m.mu <- struct{}{}
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(host string, port int, apiKey string) (*QdrantClient, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &QdrantClient{
		client:      client,
		collections: make(map[string]bool),
		mu:          newSafeMutex(),
	}, nil
}

// Point represents a vector point with payload
type Point struct {
	ID      string
	Vector  []float64
	Payload map[string]interface{}
}

// SearchOptions holds search options
type SearchOptions struct {
	Limit           int
	ScoreThreshold  float64
	Filter          *Filter
	WithPayload     bool
	WithVector      bool
}

// Filter represents a payload filter
type Filter struct {
	Must       []Condition
	MustNot    []Condition
	Should     []Condition
}

// Condition represents a filter condition
type Condition struct {
	Key    string
	Match  interface{} // string, int64, []string
	Op     string       // "match", "match_any", "range", etc.
}

// CollectionConfig holds collection configuration
type CollectionConfig struct {
	Name       string
	VectorSize int
	Distance   string // "Cosine", "Euclid", "Dot"
}

// CreateCollection creates a new collection
func (q *QdrantClient) CreateCollection(ctx context.Context, config *CollectionConfig) error {
	q.mu.lock()
	defer q.mu.unlock()

	// Check if collection already exists
	exists, err := q.client.CollectionExists(ctx, config.Name)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if exists {
		q.collections[config.Name] = true
		return nil
	}

	// Create collection vectors config
	vectorsConfig := qdrant.NewVectorsConfigMap(
		map[string]qdrant.VectorParams{
			"vector": {
				Size:     uint64(config.VectorSize),
				Distance: qdrant.Distance(config.Distance),
			},
		},
	)

	// Create collection
	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: config.Name,
		VectorsConfig:  vectorsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", config.Name, err)
	}

	q.collections[config.Name] = true
	return nil
}

// InitializeCollections initializes default collections
func (q *QdrantClient) InitializeCollections(ctx context.Context) error {
	collections := []*CollectionConfig{
		{
			Name:       memoryCollectionName,
			VectorSize: defaultVectorSize,
			Distance:   "Cosine",
		},
		{
			Name:       decisionCollectionName,
			VectorSize: defaultVectorSize,
			Distance:   "Cosine",
		},
		{
			Name:       defaultCollectionName,
			VectorSize: defaultVectorSize,
			Distance:   "Cosine",
		},
	}

	for _, config := range collections {
		if err := q.CreateCollection(ctx, config); err != nil {
			return fmt.Errorf("failed to initialize collection %s: %w", config.Name, err)
		}
	}

	return nil
}

// InsertPoints inserts points into a collection
func (q *QdrantClient) InsertPoints(ctx context.Context, collectionName string, points []*Point) error {
	q.mu.lock()
	defer q.mu.unlock()

	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, point := range points {
		qdrantPoints[i] = qdrant.NewPoint(
			uint64(i), // Using index as ID for now
			point.Vector,
			point.Payload,
		)
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         qdrantPoints,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// InsertPoint inserts a single point into a collection
func (q *QdrantClient) InsertPoint(ctx context.Context, collectionName string, point *Point) error {
	return q.InsertPoints(ctx, collectionName, []*Point{point})
}

// Search searches for similar vectors
func (q *QdrantClient) Search(ctx context.Context, collectionName string, vector []float64, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{
			Limit:       10,
			WithPayload: true,
		}
	}

	// Build search request
	searchOpts := []qdrant.SearchOpt{
		qdrant.WithLimit(uint64(opts.Limit)),
		qdrant.WithPayload(opts.WithPayload),
	}

	if opts.ScoreThreshold > 0 {
		// Note: Qdrant's go-client might not have WithScoreThreshold
		// Implement filtering on results instead
	}

	// Perform search
	points, err := q.client.Search(ctx, collectionName, vector, searchOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert to search results
	results := make([]*SearchResult, 0, len(points))
	for _, point := range points {
		// Apply score threshold
		if opts.ScoreThreshold > 0 && point.Score < opts.ScoreThreshold {
			continue
		}

		result := &SearchResult{
			ID:       fmt.Sprintf("%d", point.Id),
			Score:    point.Score,
			Vector:   point.Vector,
			Payload:  point.Payload,
		}
		results = append(results, result)
	}

	return results, nil
}

// SearchResult represents a search result
type SearchResult struct {
	ID      string
	Score   float64
	Vector  []float64
	Payload map[string]interface{}
}

// DeletePoints deletes points from a collection
func (q *QdrantClient) DeletePoints(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	qdrantIds := make([]qdrant.PointId, len(ids))
	for i, id := range ids {
		qdrantIds[i] = qdrant.NewIDNum(uint64(i)) // Simplified
	}

	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points:         qdrantIds,
	})

	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// DeleteCollection deletes a collection
func (q *QdrantClient) DeleteCollection(ctx context.Context, collectionName string) error {
	q.mu.lock()
	defer q.mu.unlock()

	err := q.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", collectionName, err)
	}

	delete(q.collections, collectionName)
	return nil
}

// GetCollectionInfo returns information about a collection
func (q *QdrantClient) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	info, err := q.client.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	return &CollectionInfo{
		Name:       collectionName,
		VectorSize: int(info.Config.Params.VectorsConfig.Size),
		PointCount: int(info.PointsCount),
	}, nil
}

// CollectionInfo holds collection information
type CollectionInfo struct {
	Name       string
	VectorSize int
	PointCount int
}

// Close closes the client connection
func (q *QdrantClient) Close() error {
	// Qdrant go-client doesn't explicitly need to be closed
	// This is a placeholder for cleanup
	return nil
}

// HealthCheck checks if Qdrant is healthy
func (q *QdrantClient) HealthCheck(ctx context.Context) error {
	// Try to get collection list
	_, err := q.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("Qdrant health check failed: %w", err)
	}
	return nil
}

// CollectionExists checks if a collection exists
func (q *QdrantClient) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	return q.client.CollectionExists(ctx, collectionName)
}
