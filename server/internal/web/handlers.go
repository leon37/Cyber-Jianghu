package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"Cyber-Jianghu/server/internal/config"
	"Cyber-Jianghu/server/internal/storage"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

type Handlers struct {
	config      *config.Config
	hub         *DanmakuHub
	liveService *LiveService
	redisStore  *storage.RedisStore
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

func NewHandlers(cfg *config.Config, hub *DanmakuHub, redisStore *storage.RedisStore) *Handlers {
	return &Handlers{
		config:      cfg,
		hub:         hub,
		liveService: NewLiveService(""), // Platform will be set on connect
		redisStore:  redisStore,
	}
}

func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cyber-jianghu",
	})
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Welcome to Cyber-Jianghu",
		"version": "1.0.0",
	})
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func NewRouter(cfg *config.Config, mysql interface{}, redis interface{}) *chi.Mux {
	r := chi.NewRouter()

	// CORS middleware
	r.Use(corsMiddleware)

	// Create danmaku hub
	hub := NewDanmakuHub()
	go hub.Run()

	// Get redis store from redis interface
	var redisStore *storage.RedisStore
	if rs, ok := redis.(*storage.RedisStore); ok {
		redisStore = rs
	}

	handlers := NewHandlers(cfg, hub, redisStore)

	// Public routes
	r.Get("/", handlers.Home)
	r.Get("/health", handlers.HealthCheck)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Story endpoints
		r.Route("/story", func(r chi.Router) {
			r.Get("/", handlers.GetStories)
			r.Post("/", handlers.CreateStory)
			r.Get("/{id}", handlers.GetStory)
			r.Put("/{id}", handlers.UpdateStory)
			r.Delete("/{id}", handlers.DeleteStory)
		})

		// Live endpoints
		r.Route("/live", func(r chi.Router) {
			r.Post("/connect", handlers.ConnectLive)
			r.Post("/disconnect", handlers.DisconnectLive)
			r.Get("/status", handlers.GetLiveStatus)
			r.Get("/danmaku", handlers.GetDanmakuStream)
		})

		// Generate endpoints
		r.Route("/generate", func(r chi.Router) {
			r.Post("/image", handlers.GenerateImage)
			r.Post("/audio", handlers.GenerateAudio)
			r.Post("/story", handlers.GenerateStory)
		})

		// Memory endpoints
		r.Route("/memory", func(r chi.Router) {
			r.Post("/", handlers.StoreMemory)
			r.Get("/search", handlers.SearchMemories)
			r.Delete("/{id}", handlers.DeleteMemory)
		})
	})

	return r
}

// Placeholder handlers for implementation in later phases
func (h *Handlers) GetStories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

func (h *Handlers) CreateStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GetStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) UpdateStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) DeleteStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GetLiveStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := h.liveService.GetStatus()
	status.ClientCount = h.hub.GetClientCount()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (h *Handlers) ConnectLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.Platform == "" || req.RoomID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "platform and room_id are required",
		})
		return
	}

	// Check if already connected
	if h.liveService.IsConnected() {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Already connected to a live room",
		})
		return
	}

	// Create new live service for this platform
	h.liveService = NewLiveService(req.Platform)
	h.liveService.SetRedisStore(h.redisStore)

	// Connect to platform
	ctx := r.Context()
	if err := h.liveService.Connect(ctx, &req, h.hub); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Send success response
	resp := ConnectResponse{
		Success:   true,
		Message:   fmt.Sprintf("Connected to %s room %s", req.Platform, req.RoomID),
		Platform:  req.Platform,
		RoomID:    req.RoomID,
		Connected: true,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	log.Printf("[Handlers] Connected to %s room %s", req.Platform, req.RoomID)
}

func (h *Handlers) DisconnectLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if connected
	if !h.liveService.IsConnected() {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Not connected to any live room",
		})
		return
	}

	// Disconnect
	if err := h.liveService.Disconnect(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Send success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Disconnected from live room",
	})

	log.Printf("[Handlers] Disconnected from live room")
}

func (h *Handlers) GetDanmakuStream(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}

	// Generate unique client ID
	clientID := generateClientID()

	// Create client
	client := &Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  h.hub,
	}

	// Register client with hub
	h.hub.register <- client

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type": "connected",
		"id":   clientID,
		"msg":  "Connected to danmaku stream",
		"time": time.Now().Unix(),
	}
	welcomeData, _ := json.Marshal(welcomeMsg)
	select {
	case client.Send <- welcomeData:
	default:
	}

	// Start client read pump
	go client.readPump()

	log.Printf("[WS] Client %s connected (total: %d)", clientID, h.hub.GetClientCount())
}

func (h *Handlers) GenerateImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GenerateAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GenerateStory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) StoreMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) SearchMemories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

// generateClientID generates a unique client ID
func generateClientID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:16]
	}
	return hex.EncodeToString(b)
}
