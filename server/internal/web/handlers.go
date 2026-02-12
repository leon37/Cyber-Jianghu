package web

import (
	"encoding/json"
	"net/http"

	"Cyber-Jianghu/server/internal/config"
	"github.com/go-chi/chi"
)

type Handlers struct {
	config *config.Config
}

func NewHandlers(cfg *config.Config) *Handlers {
	return &Handlers{
		config: cfg,
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

	handlers := NewHandlers(cfg)

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
			r.Get("/connect", handlers.ConnectLive)
			r.Post("/disconnect", handlers.DisconnectLive)
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

func (h *Handlers) ConnectLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) DisconnectLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
}

func (h *Handlers) GetDanmakuStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "Not implemented"})
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
