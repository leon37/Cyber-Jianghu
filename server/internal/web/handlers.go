package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"Cyber-Jianghu/server/internal/config"
	"Cyber-Jianghu/server/internal/engine"
	"Cyber-Jianghu/server/internal/generators"
	"Cyber-Jianghu/server/internal/infra"
	"Cyber-Jianghu/server/internal/storage"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Handlers struct {
	config         *config.Config
	hub            *DanmakuHub
	liveService    *LiveService
	redisStore     *storage.RedisStore
	comfyuiManager *infra.ComfyUIManager
}

func NewHandlers(cfg *config.Config, hub *DanmakuHub, redisStore *storage.RedisStore, comfyuiManager *infra.ComfyUIManager) *Handlers {
	return &Handlers{
		config:         cfg,
		hub:            hub,
		liveService:    nil,
		redisStore:     redisStore,
		comfyuiManager: comfyuiManager,
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
	// Serve index.html from client directory
	indexPath := filepath.Join("..", "client", "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "index.html not found"})
		return
	}
	http.ServeFile(w, r, indexPath)
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

func NewRouter(cfg *config.Config, storyEngine interface{}, redis interface{}, comfyuiManager *infra.ComfyUIManager) *chi.Mux {
	r := chi.NewRouter()

	// Request logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("REQUEST: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// CORS middleware
	r.Use(corsMiddleware)

	// Type assertion for redis store
	var redisStore *storage.RedisStore
	if redis != nil {
		redisStore = redis.(*storage.RedisStore)
	}
	handlers := NewHandlers(cfg, nil, redisStore, comfyuiManager)

	// Type assertion for story engine
	var storyHandlers *StoryHandlers
	var comfyClient *generators.ComfyUIClient

	if storyEngine != nil {
		// Create ComfyUI client
		comfyClient = generators.NewComfyUIClient()

		// Get cache directory
		baseDir, _ := filepath.Abs(filepath.Join(os.Getenv("USERPROFILE"), "cyber-jianghu"))
		imageCacheDir := filepath.Join(baseDir, "image_cache")
		_ = os.MkdirAll(imageCacheDir, 0755)

		storyHandlers = NewStoryHandlers(storyEngine.(*engine.StoryEngine), comfyClient, imageCacheDir)
	}

	// Static file server for client assets
	// The server runs from server/ directory, but client files are in ../client/static
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "..", "client", "static"))
	FileServer := http.StripPrefix("/static/", http.FileServer(filesDir))

	// Public routes
	r.Get("/", handlers.Home)
	r.Get("/health", handlers.HealthCheck)
	r.Mount("/static", FileServer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Story endpoints (demo mode)
		if storyHandlers != nil {
			r.Route("/story", func(r chi.Router) {
				r.Post("/create", storyHandlers.CreateStory)
				r.Post("/continue", storyHandlers.ContinueStory)
				r.Post("/select", storyHandlers.SelectOption)
				r.Get("/{story_id}", storyHandlers.GetStoryStatus)
			})
			// Audio endpoints
			r.Post("/audio/generate", storyHandlers.GenerateAudio)
			// Image endpoints
			r.Post("/image/generate", storyHandlers.GenerateImage)
			// Voice endpoints
			r.Get("/voice/list", storyHandlers.GetVoices)
			r.Post("/voice/default", storyHandlers.SetDefaultVoice)
		}

		// Live endpoints (Phase 2 - completed)
		r.Route("/live", func(r chi.Router) {
			r.Post("/connect", handlers.ConnectLive)
			r.Post("/disconnect", handlers.DisconnectLive)
			r.Get("/status", handlers.GetLiveStatus)
			r.Get("/danmaku", handlers.GetDanmakuStream)
		})

		// Generate endpoints (placeholders for Phase 4-5)
		r.Route("/generate", func(r chi.Router) {
			r.Post("/image", handlers.GenerateImage)
			r.Post("/audio", handlers.GenerateAudio)
		})

		// Lora endpoints (placeholder for Phase 4)
		r.Route("/lora", func(r chi.Router) {
			r.Get("/", handlers.GetLoraModels)
			r.Post("/{id}/enable", handlers.EnableLora)
			r.Post("/{id}/disable", handlers.DisableLora)
		})

		// Voice endpoints (placeholder for Phase 5)
		r.Route("/voice", func(r chi.Router) {
			r.Get("/", handlers.GetVoices)
			r.Post("/{id}/set-default", handlers.SetDefaultVoice)
		})

		// ComfyUI Management endpoints
		r.Route("/comfyui", func(r chi.Router) {
			r.Get("/status", handlers.GetComfyUIStatus)
			r.Post("/start", handlers.StartComfyUI)
			r.Post("/stop", handlers.StopComfyUI)
			r.Post("/restart", handlers.RestartComfyUI)
		})
	})

	return r
}

// Live endpoints
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

	// TODO: Connect to live platform
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Live connection not yet implemented"})
}

func (h *Handlers) DisconnectLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GetLiveStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.liveService == nil || h.liveService.GetStatus() == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Live service not initialized"})
		return
	}

	status := h.liveService.GetStatus()
	status.ClientCount = 0
	if h.hub != nil {
		status.ClientCount = h.hub.GetClientCount()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (h *Handlers) GetDanmakuStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.hub == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Hub not initialized"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// Generate unique client ID
	clientID := generateClientID()

	// Create client
	client := &Client{
		ID:     clientID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
		closed: false,
	}

	// Register client with hub
	h.hub.register <- client

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type": "connected",
		"id": clientID,
		"msg": "Connected to danmaku stream",
		"time": time.Now().Unix(),
	}
	welcomeData, _ := json.Marshal(welcomeMsg)
	select {
	case client.Send <- welcomeData:
	default:
	}

	// Start client read pump
	go client.readPump()

	w.WriteHeader(http.StatusSwitchingProtocols)
}

// Generate endpoints
func (h *Handlers) GenerateImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Use ComfyUI directly or story endpoints"})
}

func (h *Handlers) GenerateAudio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Use GPT-SoVITS directly or story endpoints"})
}

// Lora endpoints
func (h *Handlers) GetLoraModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Use LoRA registry directly or story endpoints"})
}

func (h *Handlers) EnableLora(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) DisableLora(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

// Voice endpoints
func (h *Handlers) GetVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) SetDefaultVoice(w http.ResponseWriter, r *http.Request) {
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

// ConnectRequest represents a live platform connection request
type ConnectRequest struct {
	Platform string `json:"platform"`
	RoomID   string `json:"room_id"`
	Cookie   string `json:"cookie,omitempty"`
}

// LiveStatus represents live connection status
type LiveStatus struct {
	Connected bool   `json:"connected"`
	Platform  string `json:"platform,omitempty"`
	RoomID    string `json:"room_id,omitempty"`
	ClientCount int    `json:"client_count"`
}

// ComfyUI Status Response
type ComfyUIStatusResponse struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

// GetComfyUIStatus returns the current status of ComfyUI
func (h *Handlers) GetComfyUIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.comfyuiManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "ComfyUI manager not initialized"})
		return
	}

	status := string(h.comfyuiManager.GetStatus())
	response := ComfyUIStatusResponse{
		Status: status,
	}

	if h.comfyuiManager.IsReady() {
		response.URL = h.comfyuiManager.GetURL()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StartComfyUI starts the ComfyUI service
func (h *Handlers) StartComfyUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.comfyuiManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "ComfyUI manager not initialized"})
		return
	}

	// Check if already running
	if h.comfyuiManager.IsReady() {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "ComfyUI is already running",
			"status":  "running",
			"url":     h.comfyuiManager.GetURL(),
		})
		return
	}

	// Start ComfyUI
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.comfyuiManager.Start(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to start ComfyUI: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "ComfyUI is starting...",
		"status":  "starting",
	})
}

// StopComfyUI stops the ComfyUI service
func (h *Handlers) StopComfyUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.comfyuiManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "ComfyUI manager not initialized"})
		return
	}

	// Check if already stopped
	if h.comfyuiManager.GetStatus() == infra.ComfyUIStatusStopped {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "ComfyUI is already stopped",
			"status":  "stopped",
		})
		return
	}

	// Stop ComfyUI
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.comfyuiManager.Stop(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to stop ComfyUI: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "ComfyUI stopped successfully",
		"status":  "stopped",
	})
}

// RestartComfyUI restarts the ComfyUI service
func (h *Handlers) RestartComfyUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.comfyuiManager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "ComfyUI manager not initialized"})
		return
	}

	// Restart ComfyUI
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if err := h.comfyuiManager.Restart(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to restart ComfyUI: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "ComfyUI is restarting...",
		"status":  "starting",
	})
}
