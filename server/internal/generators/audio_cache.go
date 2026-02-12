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

// AudioCacheEntry represents a cached audio entry
type AudioCacheEntry struct {
	Key           string                 `json:"key"`
	FilePath      string                 `json:"file_path"`
	AudioData     []byte                 `json:"-"`
	Base64Data    string                 `json:"base64_data,omitempty"`
	Text          string                 `json:"text"`
	VoiceID       string                 `json:"voice_id"`
	Options       *TTSOptions            `json:"options"`
	CreatedAt     time.Time              `json:"created_at"`
	LastAccessed  time.Time              `json:"last_accessed"`
	AccessCount   int                    `json:"access_count"`
	FileSize      int64                  `json:"file_size"`
	Duration      float64                `json:"duration"` // Duration in seconds
	Format        string                 `json:"format"`    // wav, mp3, etc.
	SampleRate    int                    `json:"sample_rate"`
	Hits         int                    `json:"hits"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AudioCache manages audio caching
type AudioCache struct {
	entries     map[string]*AudioCacheEntry
	directory   string
	maxEntries  int
	ttl         time.Duration
	mu          sync.RWMutex
	stats       *AudioCacheStats
}

// AudioCacheStats holds statistics about cache performance
type AudioCacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	TotalEntries int    `json:"total_entries"`
	TotalSize    int64  `json:"total_size"`
	TotalDuration float64 `json:"total_duration"`
}

// NewAudioCache creates a new audio cache
func NewAudioCache(directory string, maxEntries int, ttl time.Duration) *AudioCache {
	return &AudioCache{
		entries:    make(map[string]*AudioCacheEntry),
		directory:  directory,
		maxEntries: maxEntries,
		ttl:         ttl,
		stats:       &AudioCacheStats{},
	}
}

// Initialize loads existing cache entries from disk
func (c *AudioCache) Initialize(ctx context.Context) error {
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

		var cacheEntry AudioCacheEntry
		if err := json.Unmarshal(metaData, &cacheEntry); err != nil {
			continue
		}

		// Check if expired
		if !c.ttl.IsZero() && time.Since(cacheEntry.CreatedAt) > c.ttl {
			// Clean up expired entry
			_ = os.Remove(filepath.Join(c.directory, entry.Name()))
			_ = os.Remove(metaPath)
			continue
		}

		// Load audio data if needed
		audioPath := filepath.Join(c.directory, entry.Name())
		if info, err := entry.Info(); err == nil {
			cacheEntry.FileSize = info.Size()
		}

		// Update stats
		c.entries[cacheEntry.Key] = &cacheEntry
		c.stats.TotalEntries++
		c.stats.TotalSize += cacheEntry.FileSize
		c.stats.TotalDuration += cacheEntry.Duration
	}

	return nil
}

// Get retrieves an audio from cache
func (c *AudioCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		c.stats.Misses++
		c.updateHitRate()
		return nil, fmt.Errorf("cache miss: %s", key)
	}

	// Check if expired
	if !c.ttl.IsZero() && time.Since(entry.CreatedAt) > c.ttl {
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
	if len(entry.AudioData) > 0 {
		data, err := os.ReadFile(entry.FilePath)
		if err != nil {
			c.stats.Misses++
			c.updateHitRate()
			return nil, fmt.Errorf("failed to read cached file: %w", err)
		}
		return data, nil
	}

	return entry.AudioData, nil
}

// Put stores an audio in cache
func (c *AudioCache) Put(ctx context.Context, key string, data []byte, text string, voiceID string, opts *TTSOptions, format string, duration float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate filename with extension
	ext := format
	if ext == "" {
		ext = "wav"
	}
	filename := fmt.Sprintf("%s.%s", key, ext)
	filePath := filepath.Join(c.directory, filename)

	// Write audio data
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write audio: %w", err)
	}

	// Create cache entry
	now := time.Now()
	entry := &AudioCacheEntry{
		Key:          key,
		FilePath:     filePath,
		AudioData:    data,
		Text:         text,
		VoiceID:      voiceID,
		Options:      opts,
		CreatedAt:    now,
		LastAccessed: now,
		AccessCount:   0,
		FileSize:     int64(len(data)),
		Duration:     duration,
		Format:       format,
		Hits:         0,
		Metadata:     make(map[string]interface{}),
	}

	// Add metadata from options if available
	if opts != nil {
		entry.Metadata["speed"] = opts.Speed
		entry.Metadata["language"] = opts.Language
		entry.Metadata["tone"] = opts.Tone
	}
	entry.Metadata["voice_id"] = voiceID

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
	c.stats.TotalDuration += duration

	// Check if we need to evict entries
	if len(c.entries) > c.maxEntries {
		c.evictOldest()
	}

	return nil
}

// Check checks if an entry exists in cache
func (c *AudioCache) Check(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return false
	}

	// Check if expired
	if !c.ttl.IsZero() && time.Since(entry.CreatedAt) > c.ttl {
		return false
	}

	return true
}

// Invalidate removes an entry from cache
func (c *AudioCache) Invalidate(key string) error {
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
	c.stats.TotalDuration -= entry.Duration

	return nil
}

// Clear removes all entries from cache
func (c *AudioCache) Clear(ctx context.Context) error {
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
	c.entries = make(map[string]*AudioCacheEntry)
	c.stats.TotalEntries = 0
	c.stats.TotalSize = 0
	c.stats.TotalDuration = 0
	c.stats.Hits = 0
	c.stats.Misses = 0
	c.stats.HitRate = 0

	return nil
}

// GetStats returns cache statistics
func (c *AudioCache) GetStats() *AudioCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statsCopy := *c.stats
	return &statsCopy
}

// evictOldest removes the oldest entries to make room
func (c *AudioCache) evictOldest() {
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
func (c *AudioCache) updateHitRate() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.Hits) / float64(total)
	}
}

// GenerateCacheKey generates a cache key from text and options
func GenerateAudioCacheKey(text string, voiceID string, opts *TTSOptions) string {
	// Create a canonical representation of the request
	data := fmt.Sprintf("%s|%s|%f|%s",
		text,
		voiceID,
		opts.Speed,
		opts.Tone,
	)

	// Generate MD5 hash
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CleanExpired removes expired entries from cache
func (c *AudioCache) CleanExpired(ctx context.Context) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl.IsZero() {
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
			c.stats.TotalDuration -= entry.Duration
			count++
		}
	}

	return count
}

// ConvertAudioFormat converts audio between formats
// This is a placeholder - actual conversion would use FFmpeg or similar
func ConvertAudioFormat(input []byte, inputFormat, outputFormat string) ([]byte, error) {
	if inputFormat == outputFormat {
		return input, nil
	}

	// Placeholder for audio conversion
	// In production, this would call FFmpeg or similar tool
	return nil, fmt.Errorf("audio conversion not yet implemented: %s -> %s", inputFormat, outputFormat)
}

// GetAudioDuration estimates audio duration from file size
// This is a rough estimate based on typical audio settings
func EstimateAudioDuration(fileSize int64, format string, sampleRate int) float64 {
	// Rough estimates:
	// WAV at 16kHz, 16-bit: ~32KB per second
	// WAV at 24kHz, 16-bit: ~48KB per second
	// MP3: varies widely

	bytesPerSecond := int64(32000) // Default estimate for 16kHz WAV
	if sampleRate == 24000 {
		bytesPerSecond = 48000
	}
	if format == "mp3" {
		bytesPerSecond = bytesPerSecond / 10 // MP3 is roughly 10x smaller
	}

	return float64(fileSize) / float64(bytesPerSecond)
}

// GetCacheKeys returns all cache keys
func (c *AudioCache) GetCacheKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}

	return keys
}

// GetEntry returns detailed information about a cache entry
func (c *AudioCache) GetEntry(key string) (*AudioCacheEntry, error) {
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
