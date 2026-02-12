package adapters

import (
	"Cyber-Jianghu/server/internal/interfaces"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
)

// BilibiliAdapter implements LiveAdapter for Bilibili live platform
type BilibiliAdapter struct {
	conn          *websocket.Conn
	danmakuChan   chan interfaces.Danmaku
	roomID        string
	cookie        string
	connected     atomic.Bool
	mu            sync.Mutex
	cancel        context.CancelFunc
	heartbeatDone chan struct{}
}

// Bilibili message protocol constants
const (
	protocolVersion    = 1
	operationHeartbeat = 2
	operationMessage   = 5
	operationAuth      = 7
	headerLength       = 16
)

// bilibiliMessage represents the Bilibili WebSocket message format
type bilibiliMessage struct {
	PacketLength uint32
	HeaderLength uint16
	ProtocolVer  uint16
	Operation    uint32
	SequenceID   uint32
	Body         []byte
}

// bilibiliAuthResponse represents the auth response
type bilibiliAuthResponse struct {
	Code int `json:"code"`
	Data struct {
		Host   string `json:"host"`
		Port   int    `json:"port"`
		Token  string `json:"token"`
		Status int    `json:"status"`
	} `json:"data"`
	Message string `json:"message"`
}

// NewBilibiliAdapter creates a new Bilibili live adapter
func NewBilibiliAdapter() *BilibiliAdapter {
	return &BilibiliAdapter{
		danmakuChan: make(chan interfaces.Danmaku, 1000),
	}
}

// Connect establishes connection to Bilibili live platform
func (b *BilibiliAdapter) Connect(ctx context.Context, opts *interfaces.ConnectOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.roomID = opts.RoomID
	b.cookie = opts.Cookie
	b.heartbeatDone = make(chan struct{})

	ctx, b.cancel = context.WithCancel(ctx)

	// Get live room info
	host, port, token, err := b.getRoomInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get room info: %w", err)
	}

	// Connect to WebSocket
	wsURL := fmt.Sprintf("wss://%s:%d/sub", host, port)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, http.Header{
		"User-Agent":   {"Mozilla/5.0"},
		"Cookie":      {b.cookie},
		"Origin":      {"https://live.bilibili.com"},
		"Referer":     {fmt.Sprintf("https://live.bilibili.com/%s", b.roomID)},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	b.conn = conn

	// Send auth packet
	if err := b.sendAuth(token); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send auth: %w", err)
	}

	b.connected.Store(true)

	// Start reading messages
	go b.readMessages(ctx)

	// Start heartbeat
	go b.heartbeat(ctx)

	return nil
}

// getRoomInfo retrieves live room connection info
func (b *BilibiliAdapter) getRoomInfo(ctx context.Context) (host string, port int, token string, err error) {
	apiURL := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?id=%s&type=0", b.roomID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", 0, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Cookie", b.cookie)
	req.Header.Set("Referer", fmt.Sprintf("https://live.bilibili.com/%s", b.roomID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	var authResp bilibiliAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", 0, "", err
	}

	if authResp.Code != 0 {
		return "", 0, "", fmt.Errorf("auth failed: %s", authResp.Message)
	}

	return authResp.Data.Host, authResp.Data.Port, authResp.Data.Token, nil
}

// sendAuth sends authentication packet
func (b *BilibiliAdapter) sendAuth(token string) error {
	authJSON := map[string]interface{}{
		"uid":         0,
		"roomid":      b.roomID,
		"protover":    3,
		"platform":    "web",
		"type":        2,
		"key":         token,
	}
	body, _ := json.Marshal(authJSON)
	return b.sendMessage(operationAuth, body)
}

// sendMessage sends a message with Bilibili protocol
func (b *BilibiliAdapter) sendMessage(op uint32, body []byte) error {
	totalLen := headerLength + len(body)

	msg := bilibiliMessage{
		PacketLength: uint32(totalLen),
		HeaderLength: headerLength,
		ProtocolVer:  protocolVersion,
		Operation:    op,
		SequenceID:   1,
		Body:         body,
	}

	buf := make([]byte, totalLen)
	buf[0] = byte(msg.PacketLength >> 24)
	buf[1] = byte(msg.PacketLength >> 16)
	buf[2] = byte(msg.PacketLength >> 8)
	buf[3] = byte(msg.PacketLength)
	buf[4] = byte(msg.HeaderLength >> 8)
	buf[5] = byte(msg.HeaderLength)
	buf[6] = byte(msg.ProtocolVer >> 8)
	buf[7] = byte(msg.ProtocolVer)
	buf[8] = byte(msg.Operation >> 24)
	buf[9] = byte(msg.Operation >> 16)
	buf[10] = byte(msg.Operation >> 8)
	buf[11] = byte(msg.Operation)
	buf[12] = byte(msg.SequenceID >> 24)
	buf[13] = byte(msg.SequenceID >> 16)
	buf[14] = byte(msg.SequenceID >> 8)
	buf[15] = byte(msg.SequenceID)
	copy(buf[headerLength:], msg.Body)

	return b.conn.WriteMessage(websocket.BinaryMessage, buf)
}

// readMessages reads messages from WebSocket
func (b *BilibiliAdapter) readMessages(ctx context.Context) {
	defer b.close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, data, err := b.conn.ReadMessage()
			if err != nil {
				if err != io.EOF && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					// Log error
				}
				return
			}

			b.handleMessage(data)
		}
	}
}

// handleMessage processes incoming messages
func (b *BilibiliAdapter) handleMessage(data []byte) {
	offset := 0
	for offset < len(data) {
		if len(data) < offset+headerLength {
			break
		}

		packetLen := binary.BigEndian.Uint32(data[offset : offset+4])
		headerLen := binary.BigEndian.Uint16(data[offset+4 : offset+6])
		operation := binary.BigEndian.Uint32(data[offset+8 : offset+12])

		if packetLen < headerLength {
			break
		}

		if operation == operationMessage {
			body := data[offset+headerLength : offset+int(packetLen)]
			b.parseDanmaku(body)
		}

		offset += int(packetLen)
	}
}

// parseDanmaku parses danmaku from message body
func (b *BilibiliAdapter) parseDanmaku(body []byte) {
	// Bilibili uses a custom JSON format for danmaku
	// Simplified parser for CMD_DANMU_MSG (0x4001)

	// Skip protocol header if present
	if len(body) > 16 {
		// Check for Bilibili's JSON format
		bodyStr := string(body)
		if bodyStr[0] == '{' {
			var msg struct {
				Cmd string `json:"cmd"`
				Info [][]interface{} `json:"info"`
			}
			if err := json.Unmarshal(body, &msg); err == nil && msg.Cmd == "DANMU_MSG" {
				if len(msg.Info) > 2 {
					// info[0]: danmaku text
					// info[1]: user info [uid, username]
					// info[2]: other info
					danmakuText, _ := msg.Info[1].(string)
					userInfo, _ := msg.Info[2].([]interface{})
					if len(userInfo) > 1 {
						uid := fmt.Sprintf("%v", userInfo[0])
						username, _ := userInfo[1].(string)

						danmaku := interfaces.Danmaku{
							Username:  username,
							UserID:    uid,
							Content:   danmakuText,
							Timestamp: time.Now().Unix(),
							IsVip:     false,
							IsAdmin:   false,
							GiftValue: 0,
						}

						select {
						case b.danmakuChan <- danmaku:
						default:
							// Channel full, drop message
						}
					}
				}
			}
		}
	}
}

// heartbeat sends periodic heartbeat messages
func (b *BilibiliAdapter) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := b.sendMessage(operationHeartbeat, nil); err != nil {
				return
			}
		}
	}
}

// SubscribeDanmaku returns a channel for receiving danmaku messages
func (b *BilibiliAdapter) SubscribeDanmaku(ctx context.Context) (<-chan interfaces.Danmaku, error) {
	if !b.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}
	return b.danmakuChan, nil
}

// SendChat sends a chat message to the live room
func (b *BilibiliAdapter) SendChat(ctx context.Context, msg string) error {
	// Bilibili requires authentication to send messages
	// This would need to be implemented with proper auth
	return fmt.Errorf("not implemented: requires authenticated API")
}

// HealthCheck checks if the connection is still alive
func (b *BilibiliAdapter) HealthCheck(ctx context.Context) error {
	if !b.connected.Load() {
		return fmt.Errorf("not connected")
	}
	return nil
}

// Disconnect closes the connection
func (b *BilibiliAdapter) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected.Load() {
		return nil
	}

	if b.cancel != nil {
		b.cancel()
	}

	if b.conn != nil {
		b.conn.Close()
	}

	b.connected.Store(false)
	close(b.danmakuChan)

	return nil
}

// close internal cleanup
func (b *BilibiliAdapter) close() {
	b.connected.Store(false)
	if b.heartbeatDone != nil {
		close(b.heartbeatDone)
	}
}
