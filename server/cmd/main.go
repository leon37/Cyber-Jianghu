package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"Cyber-Jianghu/server/internal/config"
	"Cyber-Jianghu/server/internal/engine"
	"Cyber-Jianghu/server/internal/generators"
	"Cyber-Jianghu/server/internal/infra"
	"Cyber-Jianghu/server/internal/rag"
	"Cyber-Jianghu/server/internal/storage"
	"Cyber-Jianghu/server/internal/web"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage connections
	mysqlStore, err := storage.NewMySQLStore(cfg.Database.MySQL)
	if err != nil {
		log.Printf("Warning: Failed to connect to MySQL: %v", err)
		mysqlStore = nil
	} else {
		defer mysqlStore.Close()
		log.Println("MySQL connected successfully")
	}

	redisStore, err := storage.NewRedisStore(cfg.Database.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		redisStore = nil
	} else {
		defer redisStore.Close()
		log.Println("Redis connected successfully")
	}

	// Initialize AI components
	// Get API key from config or environment
	apiKey := cfg.AI.GLM5.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ZHIPUAI_API_KEY")
	}
	if apiKey == "" {
		log.Println("Warning: No ZhipuAI API key provided. Some features may not work.")
	}

	// Initialize Qdrant client
	var qdrantClient *rag.QdrantClient
	if apiKey != "" {
		qdrantHost := cfg.Database.Qdrant.Host
		if qdrantHost == "" {
			qdrantHost = "localhost"
		}
		qdrantPort := cfg.Database.Qdrant.Port
		if qdrantPort == 0 {
			qdrantPort = 6333
		}
		var err error
		qdrantClient, err = rag.NewQdrantClient(qdrantHost, qdrantPort, cfg.Database.Qdrant.APIKey)
		if err != nil {
			log.Printf("Warning: Failed to connect to Qdrant: %v", err)
		} else {
			log.Println("Qdrant connected successfully")
			// Initialize collections
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := qdrantClient.InitializeCollections(ctx); err != nil {
				log.Printf("Warning: Failed to initialize Qdrant collections: %v", err)
			}
			cancel()
		}
	}

	// Create cache directories
	baseDir := "./data"
	audioCacheDir := filepath.Join(baseDir, "audio_cache")
	_ = os.MkdirAll(audioCacheDir, 0755)

	// Initialize StoryEngine
	var storyEngine *engine.StoryEngine
	if qdrantClient != nil {
		storyEngine = engine.NewStoryEngine(apiKey, qdrantClient, audioCacheDir)
		log.Println("StoryEngine initialized successfully")
	}

	// Initialize AIGC components
	_ = generators.NewComfyUIClient()
	imageCacheDir := filepath.Join("./data", "image_cache")
	_ = os.MkdirAll(imageCacheDir, 0755)
	imageCache := generators.NewImageCache(imageCacheDir, 1000, 24*time.Hour)
	_ = imageCache.Initialize(context.Background())

	loraDir := filepath.Join("./data", "lora_models")
	_ = os.MkdirAll(loraDir, 0755)
	loraRegistry := generators.NewLoRARegistry(loraDir)
	_ = loraRegistry.LoadModels(context.Background())

	// Initialize ComfyUI Manager
	var comfyuiManager *infra.ComfyUIManager
	comfyuiCfg := &infra.ComfyUIManagerConfig{
		Host:     "127.0.0.1",
		Port:     8188,
		ModelsDir: "D:\\ComfyUI",
		UseGPU:   true,
	}
	comfyuiManager = infra.NewComfyUIManager(comfyuiCfg)

	// Create router with story engine integration
	r := web.NewRouter(cfg, storyEngine, redisStore, comfyuiManager)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in background
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
