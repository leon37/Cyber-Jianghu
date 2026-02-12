package storage

import (
	"Cyber-Jianghu/server/internal/config"
	"Cyber-Jianghu/server/internal/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(cfg config.RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) GetClient() *redis.Client {
	return s.client
}

// Helper methods for common operations
func (s *RedisStore) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *RedisStore) Del(ctx context.Context, keys ...string) error {
	return s.client.Del(ctx, keys...).Err()
}

func (s *RedisStore) Exists(ctx context.Context, keys ...string) (int64, error) {
	return s.client.Exists(ctx, keys...).Result()
}

func (s *RedisStore) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return s.client.Expire(ctx, key, expiration).Err()
}

// Danmaku storage methods
const (
	danmakuListKey        = "danmaku:list"
	danmakuMaxListSize    = 10000     // Maximum number of danmaku to keep in the list
	danmakuDedupKey       = "danmaku:dedup"
	danmakuDedupTTL       = 5 * time.Minute
	danmakuListTTL        = 24 * time.Hour
)

// StoreDanmaku stores a danmaku message in Redis
func (s *RedisStore) StoreDanmaku(ctx context.Context, danmaku interfaces.Danmaku) error {
	// Deduplication check using content hash
	dedupKey := fmt.Sprintf("%s:%s:%s:%d", danmakuDedupKey, danmaku.UserID, danmaku.Content, danmaku.Timestamp)
	exists, err := s.Exists(ctx, dedupKey)
	if err != nil {
		return fmt.Errorf("failed to check dedup: %w", err)
	}
	if exists > 0 {
		return nil // Duplicate, skip
	}

	// Marshal danmaku to JSON
	data, err := json.Marshal(danmaku)
	if err != nil {
		return fmt.Errorf("failed to marshal danmaku: %w", err)
	}

	// Store in list (LPUSH to add to the front)
	if err := s.client.LPush(ctx, danmakuListKey, data).Err(); err != nil {
		return fmt.Errorf("failed to store danmaku in list: %w", err)
	}

	// Trim list to max size
	if err := s.client.LTrim(ctx, danmakuListKey, 0, int64(danmakuMaxListSize-1)).Err(); err != nil {
		return fmt.Errorf("failed to trim danmaku list: %w", err)
	}

	// Set dedup key
	if err := s.Set(ctx, dedupKey, "1", danmakuDedupTTL); err != nil {
		return fmt.Errorf("failed to set dedup key: %w", err)
	}

	// Set list TTL
	if err := s.client.Expire(ctx, danmakuListKey, danmakuListTTL).Err(); err != nil {
		// Non-critical error, log but don't fail
		fmt.Printf("[RedisStore] Warning: failed to set list TTL: %v", err)
	}

	return nil
}

// GetRecentDanmaku retrieves recent danmaku from Redis
func (s *RedisStore) GetRecentDanmaku(ctx context.Context, limit int64) ([]interfaces.Danmaku, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // Default limit
	}

	// Get danmaku from list
	results, err := s.client.LRange(ctx, danmakuListKey, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get danmaku from list: %w", err)
	}

	// Unmarshal danmaku
	danmakus := make([]interfaces.Danmaku, 0, len(results))
	for _, result := range results {
		var danmaku interfaces.Danmaku
		if err := json.Unmarshal([]byte(result), &danmaku); err != nil {
			continue // Skip invalid entries
		}
		danmakus = append(danmakus, danmaku)
	}

	return danmakus, nil
}

// GetDanmakuCount returns the number of danmaku in the list
func (s *RedisStore) GetDanmakuCount(ctx context.Context) (int64, error) {
	return s.client.LLen(ctx, danmakuListKey).Result()
}

// ClearDanmaku clears all danmaku from Redis
func (s *RedisStore) ClearDanmaku(ctx context.Context) error {
	return s.Del(ctx, danmakuListKey)
}
