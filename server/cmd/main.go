package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Cyber-Jianghu/server/internal/config"
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

	// Create router
	r := web.NewRouter(cfg, mysqlStore, redisStore)

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
