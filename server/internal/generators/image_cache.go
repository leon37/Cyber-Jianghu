package generators

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached image
type CacheEntry struct {
	Key           string                 `json:"key"`
	FilePath      string                 `json:"file_path"`
	ImageData     []byte                 `json:"-"`
	Base64Data    string                 `json:"base64_data,omitempty"`
	Prompt        string                 `json:"prompt"`
	Options       *GenerateOptions        `json:"options"`
	CreatedAt     time.Time              `json:"created_at"`
	LastAccessed  time.Time              `json:"last_accessed"`
	AccessCount   int                    `json:"access_count"`
	FileSize      int64                  `json:"file_size"`
	Hits         int                    `json:"hits"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ImageCache manages image caching
type ImageCache struct {
	entries    map[string]*CacheEntry
	directory  string
	maxEntries int
	ttl         time.Duration
	mu          sync.RWMutex
	stats       *CacheStats
}

// CacheStats holds statistics about cache performance
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	TotalEntries int    `json:"total_entries"`
	TotalSize    int64  `json:"total_size"`
}

// NewImageCache creates a new image cache
func NewImageCache(directory string, maxEntries int, ttl time.Duration) *ImageCache {
	return &ImageCache{
		entries:    make(map[string]*CacheEntry),
		directory:  directory,
		maxEntries: maxEntries,
		ttl:         ttl,
		stats:       &CacheStats{},
	}
}

// Initialize loads existing cache entries from disk
func (c *ImageCache) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure directory exists
	if _, err := os.Stat(c.directory); os.IsNotExist(err) {
		if err := os.MkdirAll(c.directory, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		return nil
	}

	// Scan directory and load entries
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check for metadata file
		metaPath := filepath.Join(c.directory, entry.Name()+".meta")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue
		}

		// Load metadata
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(metaData, &cacheEntry); err != nil {
			continue
		}

		// Check if expired
		if c.ttl != 0 && time.Since(cacheEntry.CreatedAt) > c.ttl {
			// Clean up expired entry
			_ = os.Remove(filepath.Join(c.directory, entry.Name()))
			_ = os.Remove(metaPath)
			continue
		}

		// Load image data if needed
		if info, err := entry.Info(); err == nil {
			cacheEntry.FileSize = info.Size()
		}

		c.entries[cacheEntry.Key] = &cacheEntry
		c.stats.TotalEntries++
		c.stats.TotalSize += cacheEntry.FileSize
	}

	return nil
}

// Get retrieves an image from cache
func (c *ImageCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		c.stats.Misses++
		c.updateHitRate()
		return nil, fmt.Errorf("cache miss: %s", key)
	}

	// Check if expired
	if c.ttl > 0 && time.Since(entry.CreatedAt) > c.ttl {
		delete(c.entries, key)
		c.stats.Misses++
		c.updateHitRate()
		// Clean up files
		_ = os.Remove(entry.FilePath)
		_ = os.Remove(entry.FilePath + ".meta")
		return nil, fmt.Errorf("cache entry expired")
	}

	// Update access info
	entry.LastAccessed = time.Now()
	entry.AccessCount++
	entry.Hits++

	c.stats.Hits++
	c.updateHitRate()

	// Return cached data
	if len(entry.ImageData) > 0 {
		data, err := os.ReadFile(entry.FilePath)
		if err != nil {
			c.stats.Misses++
			c.updateHitRate()
			return nil, fmt.Errorf("failed to read cached file: %w", err)
		}
		return data, nil
	}

	return entry.ImageData, nil
}

// Put stores an image in cache
func (c *ImageCache) Put(ctx context.Context, key string, data []byte, prompt string, opts *GenerateOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate filename
	filename := fmt.Sprintf("%s.png", key)
	filePath := filepath.Join(c.directory, filename)

	// Write image data
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	// Create cache entry
	now := time.Now()
	entry := &CacheEntry{
		Key:          key,
		FilePath:     filePath,
		ImageData:    data,
		Prompt:       prompt,
		Options:      opts,
		CreatedAt:    now,
		LastAccessed: now,
		AccessCount:   0,
		FileSize:     int64(len(data)),
		Hits:         0,
		Metadata:     make(map[string]interface{}),
	}

	// Add metadata from options if available
	if opts != nil {
		entry.Metadata["width"] = opts.Width
		entry.Metadata["height"] = opts.Height
		entry.Metadata["model"] = opts.Model
		if opts.Lora != "" {
			entry.Metadata["lora"] = opts.Lora
		}
	}

	// Write metadata
	metaPath := filePath + ".meta"
	metaData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Add to cache
	c.entries[key] = entry
	c.stats.TotalEntries++
	c.stats.TotalSize += int64(len(data))

	// Check if we need to evict entries
	if len(c.entries) > c.maxEntries {
		c.evictOldest()
	}

	return nil
}

// Check checks if an entry exists in cache
func (c *ImageCache) Check(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return false
	}

	// Check if expired
	if c.ttl > 0 && time.Since(entry.CreatedAt) > c.ttl {
		return false
	}

	return true
}

// Invalidate removes an entry from cache
func (c *ImageCache) Invalidate(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	// Delete files
	if entry.FilePath != "" {
		_ = os.Remove(entry.FilePath)
		_ = os.Remove(entry.FilePath + ".meta")
	}

	// Remove from cache
	delete(c.entries, key)
	c.stats.TotalEntries--
	c.stats.TotalSize -= entry.FileSize

	return nil
}

// Clear removes all entries from cache
func (c *ImageCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Delete all files
	for _, entry := range c.entries {
		if entry.FilePath != "" {
			_ = os.Remove(entry.FilePath)
			_ = os.Remove(entry.FilePath + ".meta")
		}
	}

	// Clear cache
	c.entries = make(map[string]*CacheEntry)
	c.stats.TotalEntries = 0
	c.stats.TotalSize = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
	c.stats.HitRate = 0

	return nil
}

// GetStats returns cache statistics
func (c *ImageCache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statsCopy := *c.stats
	return &statsCopy
}

// evictOldest removes the oldest entries to make room
func (c *ImageCache) evictOldest() {
	// Find oldest entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestTime.IsZero() || entry.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccessed
		}
	}

	if oldestKey != "" {
		_ = c.Invalidate(oldestKey)
	}
}

// updateHitRate recalculates the hit rate
func (c *ImageCache) updateHitRate() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total)
	}
}

// GenerateCacheKey generates a cache key from prompt and options
func GenerateCacheKey(prompt string, opts *GenerateOptions) string {
	// Create a canonical representation of the request
	data := fmt.Sprintf("%s|%dx%d|%d|%f|%s|%s|%s",
		prompt,
		opts.Width, opts.Height,
		opts.Steps,
		opts.CFGScale,
		opts.Model,
		opts.Lora,
		opts.LoraStrength,
	)

	// Generate MD5 hash
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CleanExpired removes expired entries from cache
func (c *ImageCache) CleanExpired(ctx context.Context) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl == 0 {
		return 0
	}

	count := 0
	now := time.Now()

	for key, entry := range c.entries {
		if now.Sub(entry.CreatedAt) > c.ttl {
			// Delete files
			if entry.FilePath != "" {
				_ = os.Remove(entry.FilePath)
				_ = os.Remove(entry.FilePath + ".meta")
			}

			// Remove from cache
			delete(c.entries, key)
			c.stats.TotalEntries--
			c.stats.TotalSize -= entry.FileSize
			count++
		}
	}

	return count
}

// GetCacheKeys returns all cache keys
func (c *ImageCache) GetCacheKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}

	return keys
}

// GetEntry returns detailed information about a cache entry
func (c *ImageCache) GetEntry(key string) (*CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, fmt.Errorf("entry not found: %s", key)
	}

	// Return a copy
	entryCopy := *entry
	if entry.Metadata != nil {
		entryCopy.Metadata = make(map[string]interface{})
		for k, v := range entry.Metadata {
			entryCopy.Metadata[k] = v
		}
	}

	return &entryCopy, nil
}
