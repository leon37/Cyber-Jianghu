package rag

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"Cyber-Jianghu/server/internal/engine"
)

const (
	// GLM embedding models
	embeddingV3       = "embedding-3"
	embeddingV2      = "embedding-2"
	cacheTTL         = 24 * time.Hour
	embeddingDim     = 1024 // GLM-5 embedding dimension
)

// EmbeddingCache stores cached embeddings
type EmbeddingCache struct {
	cache map[string]*CachedEmbedding
	mu    sync.RWMutex
}

// CachedEmbedding holds a cached embedding with expiration
type CachedEmbedding struct {
	Vector    []float64
	CreatedAt time.Time
}

// EmbeddingService handles text embedding generation and caching
type EmbeddingService struct {
	client    *engine.GLM5Client
	cache     *EmbeddingCache
	model     string
	batchSize int
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(apiKey string) *EmbeddingService {
	return &EmbeddingService{
		client:    engine.NewGLM5Client(apiKey),
		cache:     &EmbeddingCache{cache: make(map[string]*CachedEmbedding)},
		model:     embeddingV3,
		batchSize: 100, // GLM-5 supports batch processing
	}
}

// SetModel sets the embedding model to use
func (s *EmbeddingService) SetModel(model string) {
	s.model = model
}

// Embed generates embedding for a single text
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float64, error) {
	// Check cache first
	if vec, ok := s.getFromCache(text); ok {
		return vec, nil
	}

	// Generate embedding
	vectors, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Check cache for all texts
	cachedVectors := make([][]float64, len(texts))
	uncachedIndices := make([]int, 0, len(texts))
	uncachedTexts := make([]string, 0, len(texts))

	for i, text := range texts {
		if vec, ok := s.getFromCache(text); ok {
			cachedVectors[i] = vec
		} else {
			uncachedIndices = append(uncachedIndices, i)
			uncachedTexts = append(uncachedTexts, text)
		}
	}

	// If all texts are cached, return
	if len(uncachedTexts) == 0 {
		return cachedVectors, nil
	}

	// Batch process uncached texts
	newVectors, err := s.embedBatchUncached(ctx, uncachedTexts)
	if err != nil {
		return nil, err
	}

	// Fill in the cached vectors
	for i, idx := range uncachedIndices {
		cachedVectors[idx] = newVectors[i]
		// Cache the result
		s.cache.Put(uncachedTexts[i], newVectors[i])
	}

	return cachedVectors, nil
}

// embedBatchUncached performs actual embedding API call
func (s *EmbeddingService) embedBatchUncached(ctx context.Context, texts []string) ([][]float64, error) {
	// Process in batches if needed
	allVectors := make([][]float64, 0, len(texts))

	for i := 0; i < len(texts); i += s.batchSize {
		end := i + s.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		response, err := s.client.CreateEmbedding(ctx, batch, s.model)
		if err != nil {
			return nil, fmt.Errorf("failed to create embeddings: %w", err)
		}

		if response.Error != nil {
			return nil, fmt.Errorf("embedding API error: %s", response.Error.Message)
		}

		// Extract vectors from response
		for _, data := range response.Data {
			// Normalize vector
			normalized := NormalizeVector(data.Embedding)
			allVectors = append(allVectors, normalized)
		}
	}

	return allVectors, nil
}

// getFromCache retrieves embedding from cache
func (s *EmbeddingService) getFromCache(text string) ([]float64, bool) {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	cached, ok := s.cache.cache[text]
	if !ok {
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cached.CreatedAt) > cacheTTL {
		return nil, false
	}

	return cached.Vector, true
}

// Put caches an embedding
func (c *EmbeddingCache) Put(text string, vector []float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[text] = &CachedEmbedding{
		Vector:    vector,
		CreatedAt: time.Now(),
	}
}

// ClearCache clears the embedding cache
func (s *EmbeddingService) ClearCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	s.cache.cache = make(map[string]*CachedEmbedding)
}

// GetCacheSize returns the number of cached embeddings
func (s *EmbeddingService) GetCacheSize() int {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	return len(s.cache.cache)
}

// NormalizeVector normalizes a vector to unit length
func NormalizeVector(vector []float64) []float64 {
	if len(vector) == 0 {
		return vector
	}

	// Calculate norm
	var norm float64
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	// Avoid division by zero
	if norm == 0 {
		return vector
	}

	// Normalize
	normalized := make([]float64, len(vector))
	for i, v := range vector {
		normalized[i] = v / norm
	}

	return normalized
}

// CalculateCosineSimilarity calculates cosine similarity between two vectors
func CalculateCosineSimilarity(v1, v2 []float64) (float64, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vector dimensions don't match: %d vs %d", len(v1), len(v2))
	}

	if len(v1) == 0 {
		return 0, nil
	}

	var dotProduct, norm1, norm2 float64
	for i := range v1 {
		dotProduct += v1[i] * v2[i]
		norm1 += v1[i] * v1[i]
		norm2 += v2[i] * v2[i]
	}

	norm1 = math.Sqrt(norm1)
	norm2 = math.Sqrt(norm2)

	if norm1 == 0 || norm2 == 0 {
		return 0, nil
	}

	return dotProduct / (norm1 * norm2), nil
}

// CalculateEuclideanDistance calculates Euclidean distance between two vectors
func CalculateEuclideanDistance(v1, v2 []float64) (float64, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vector dimensions don't match: %d vs %d", len(v1), len(v2))
	}

	if len(v1) == 0 {
		return 0, nil
	}

	var sum float64
	for i := range v1 {
		diff := v1[i] - v2[i]
		sum += diff * diff
	}

	return math.Sqrt(sum), nil
}

// CalculateDotProduct calculates dot product of two vectors
func CalculateDotProduct(v1, v2 []float64) (float64, error) {
	if len(v1) != len(v2) {
		return 0, fmt.Errorf("vector dimensions don't match: %d vs %d", len(v1), len(v2))
	}

	var dotProduct float64
	for i := range v1 {
		dotProduct += v1[i] * v2[i]
	}

	return dotProduct, nil
}

// IsValidVector checks if a vector is valid (no NaN or Inf values)
func IsValidVector(vector []float64) bool {
	for _, v := range vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// EmbeddingStats holds statistics about the embedding service
type EmbeddingStats struct {
	CacheSize    int
	Model        string
	EmbeddingDim int
	BatchSize    int
}

// GetStats returns statistics about the embedding service
func (s *EmbeddingService) GetStats() *EmbeddingStats {
	return &EmbeddingStats{
		CacheSize:    s.GetCacheSize(),
		Model:        s.model,
		EmbeddingDim: embeddingDim,
		BatchSize:    s.batchSize,
	}
}
