package web

import (
	"Cyber-Jianghu/server/internal/adapters"
	"Cyber-Jianghu/server/internal/interfaces"
	"Cyber-Jianghu/server/internal/storage"
	"context"
	"fmt"
	"log"
	"sync"
)

// LiveService manages live platform connections
type LiveService struct {
	adapter interfaces.LiveAdapter
	connected bool
	platform string
	roomID string
	mu sync.RWMutex
	danmakuParser *adapters.DanmakuParser
	redisStore *storage.RedisStore
}

// NewLiveService creates a new live service
func NewLiveService(platform string) *LiveService {
	return &LiveService{
		platform: platform,
		danmakuParser: adapters.NewDanmakuParser(),
	}
}

// SetRedisStore sets the Redis store for danmaku storage
func (s *LiveService) SetRedisStore(redisStore *storage.RedisStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redisStore = redisStore
}

// ConnectOptions holds connection parameters for live platform
type ConnectRequest struct {
	Platform string `json:"platform"` // "bilibili" or "douyin"
	RoomID   string `json:"room_id"`
	Cookie   string `json:"cookie"`
}

// ConnectResponse holds connection response
type ConnectResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Platform  string `json:"platform"`
	RoomID    string `json:"room_id"`
	Connected bool   `json:"connected"`
}

// StatusResponse holds status response
type StatusResponse struct {
	Connected bool   `json:"connected"`
	Platform  string `json:"platform,omitempty"`
	RoomID    string `json:"room_id,omitempty"`
	ClientCount int  `json:"client_count"`
}

// Connect connects to a live platform
func (s *LiveService) Connect(ctx context.Context, opts *ConnectRequest, hub *DanmakuHub) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return fmt.Errorf("already connected")
	}

	// Create adapter based on platform
	switch opts.Platform {
	case "bilibili":
		s.adapter = adapters.NewBilibiliAdapter()
	case "douyin":
		// TODO: Implement Douyin adapter
		return fmt.Errorf("douyin adapter not implemented yet")
	default:
		return fmt.Errorf("unsupported platform: %s", opts.Platform)
	}

	// Connect to platform
	connectOpts := &interfaces.ConnectOptions{
		RoomID: opts.RoomID,
		Cookie: opts.Cookie,
	}

	if err := s.adapter.Connect(ctx, connectOpts); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	s.connected = true
	s.platform = opts.Platform
	s.roomID = opts.RoomID
	s.adapter.(*adapters.BilibiliAdapter).SetParser(s.danmakuParser)

	// Subscribe to danmaku and forward to hub
	go s.forwardDanmaku(ctx, hub)

	log.Printf("[LiveService] Connected to %s room %s", opts.Platform, opts.RoomID)
	return nil
}

// Disconnect disconnects from the live platform
func (s *LiveService) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	if err := s.adapter.Disconnect(); err != nil {
		log.Printf("[LiveService] Error disconnecting: %v", err)
	}

	s.connected = false
	s.platform = ""
	s.roomID = ""

	log.Printf("[LiveService] Disconnected")
	return nil
}

// IsConnected returns whether the service is connected
func (s *LiveService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// GetStatus returns the current status
func (s *LiveService) GetStatus() *StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &StatusResponse{
		Connected: s.connected,
		Platform:  s.platform,
		RoomID:    s.roomID,
		ClientCount: 0, // Will be set by handler
	}
}

// forwardDanmaku forwards danmaku from adapter to hub
func (s *LiveService) forwardDanmaku(ctx context.Context, hub *DanmakuHub) {
	danmakuChan, err := s.adapter.SubscribeDanmaku(ctx)
	if err != nil {
		log.Printf("[LiveService] Failed to subscribe danmaku: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case danmaku, ok := <-danmakuChan:
			if !ok {
				log.Printf("[LiveService] Danmaku channel closed")
				return
			}

			// Parse danmaku for commands
			parsedCmd := s.danmakuParser.Parse(danmaku)
			if parsedCmd.Type != adapters.CommandNone {
				log.Printf("[LiveService] Parsed command: %+v from %s", parsedCmd, danmaku.Username)
			}

			// Broadcast to all WebSocket clients
			hub.Broadcast(danmaku)

			// Store to Redis (non-blocking)
			s.mu.RLock()
			redisStore := s.redisStore
			s.mu.RUnlock()

			if redisStore != nil {
				go func(d interfaces.Danmaku) {
					if err := redisStore.StoreDanmaku(context.Background(), d); err != nil {
						log.Printf("[LiveService] Failed to store danmaku to Redis: %v", err)
					}
				}(danmaku)
			}
		}
	}
}
