package web

import (
	"Cyber-Jianghu/server/internal/interfaces"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *DanmakuHub
	mu     sync.Mutex
	closed bool
}

// DanmakuHub manages WebSocket connections and broadcasts danmaku messages
type DanmakuHub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan interfaces.Danmaku
	danmakuOut chan []byte
	mu         sync.RWMutex
}

// NewDanmakuHub creates a new danmaku hub
func NewDanmakuHub() *DanmakuHub {
	return &DanmakuHub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		broadcast:  make(chan interfaces.Danmaku, 1000),
		danmakuOut: make(chan []byte, 1000),
	}
}

// Run starts the hub's event loop
func (h *DanmakuHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case danmaku := <-h.broadcast:
			h.broadcastDanmaku(danmaku)
		}
	}
}

// registerClient adds a new client to the hub
func (h *DanmakuHub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	log.Printf("[Hub] Client connected: %s (total: %d)", client.ID, len(h.clients))

	// Start the client's write pump
	go client.writePump()
}

// unregisterClient removes a client from the hub
func (h *DanmakuHub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		log.Printf("[Hub] Client disconnected: %s (total: %d)", client.ID, len(h.clients))
	}
}

// broadcastDanmaku sends a danmaku message to all connected clients
func (h *DanmakuHub) broadcastDanmaku(danmaku interfaces.Danmaku) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Serialize danmaku to JSON
	data, err := json.Marshal(map[string]interface{}{
		"type":    "danmaku",
		"data":    danmaku,
		"time":    time.Now().Unix(),
		"sentAt":  danmaku.Timestamp,
	})
	if err != nil {
		log.Printf("[Hub] Failed to marshal danmaku: %v", err)
		return
	}

	// Send to all clients
	sentCount := 0
	for _, client := range h.clients {
		select {
		case client.Send <- data:
			sentCount++
		default:
			// Client send buffer full, skip
			log.Printf("[Hub] Client send buffer full: %s", client.ID)
		}
	}

	log.Printf("[Hub] Broadcast danmaku to %d clients", sentCount)
}

// Broadcast sends a danmaku message to all connected clients (public method)
func (h *DanmakuHub) Broadcast(danmaku interfaces.Danmaku) {
	select {
	case h.broadcast <- danmaku:
	default:
		log.Printf("[Hub] Broadcast channel full, dropping danmaku")
	}
}

// GetClientCount returns the number of connected clients
func (h *DanmakuHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			if !ok {
				// Hub closed the channel
				c.closed = true
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Client] Error writing to %s: %v", c.ID, err)
				c.closed = true
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

		case <-ticker.C:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}

			// Send ping
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[Client] Error sending ping to %s: %v", c.ID, err)
				c.closed = true
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	c.Conn.Close()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Client] Unexpected close from %s: %v", c.ID, err)
			}
			break
		}
	}
}
