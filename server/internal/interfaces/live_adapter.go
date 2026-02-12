package interfaces

import "context"

// ConnectOptions holds connection parameters for live platforms
type ConnectOptions struct {
	RoomID string
	Cookie string
}

// Danmaku represents a live chat message
type Danmaku struct {
	Username  string
	UserID    string
	Content   string
	Timestamp int64
	IsVip     bool
	IsAdmin   bool
	GiftValue int // 赠送礼物价值（抖币/金瓜子）
}

// LiveAdapter defines the interface for live streaming platforms
type LiveAdapter interface {
	// Connect establishes connection to the live platform
	Connect(ctx context.Context, opts *ConnectOptions) error

	// SubscribeDanmaku returns a channel for receiving danmaku messages
	SubscribeDanmaku(ctx context.Context) (<-chan Danmaku, error)

	// SendChat sends a chat message to the live room
	SendChat(ctx context.Context, msg string) error

	// HealthCheck checks if the connection is still alive
	HealthCheck(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect() error
}
