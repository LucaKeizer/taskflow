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

	"taskflow/internal/api"
	"taskflow/internal/queue"
	"taskflow/internal/storage"
)

func main() {
	// Configuration from environment variables
	config := getConfig()

	log.Printf("Starting TaskFlow API Server...")
	log.Printf("Server will listen on %s", config.ServerAddr)
	log.Printf("Redis: %s, Database: %s", config.RedisAddr, config.DatabaseURL)

	// Initialize Redis queue
	redisQueue := queue.NewRedisQueue(config.RedisAddr, config.RedisPassword, config.RedisDB)
	defer redisQueue.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisQueue.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✓ Connected to Redis")

	// Initialize PostgreSQL storage
	postgresStorage, err := storage.NewPostgresStorage(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer postgresStorage.Close()
	log.Println("✓ Connected to PostgreSQL")

	// Initialize API server
	server := api.NewServer(redisQueue, postgresStorage)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         config.ServerAddr,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 TaskFlow API Server listening on %s", config.ServerAddr)
		log.Printf("📊 Health check: http://%s/api/v1/health", config.ServerAddr)
		log.Printf("📋 API docs will be available at: http://%s/api/v1/", config.ServerAddr)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}

type Config struct {
	ServerAddr    string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	DatabaseURL   string
}

func getConfig() *Config {
	config := &Config{
		ServerAddr:    getEnv("SERVER_ADDR", ":8080"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://taskflow:taskflow@localhost/taskflow?sslmode=disable"),
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example usage information
func init() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println(`TaskFlow API Server

Environment Variables:
  SERVER_ADDR      Server address (default: :8080)
  REDIS_ADDR       Redis address (default: localhost:6379)
  REDIS_PASSWORD   Redis password (default: empty)
  DATABASE_URL     PostgreSQL connection string
                   (default: postgres://taskflow:taskflow@localhost/taskflow?sslmode=disable)

Example API Usage:

1. Create a job:
   curl -X POST http://localhost:8080/api/v1/jobs \
     -H "Content-Type: application/json" \
     -d '{
       "type": "email",
       "payload": {
         "to": "user@example.com",
         "subject": "Test Email",
         "body": "This is a test email from TaskFlow"
       }
     }'

2. Check job status:
   curl http://localhost:8080/api/v1/jobs/{job_id}

3. List jobs:
   curl http://localhost:8080/api/v1/jobs?status=completed&page=1&page_size=10

4. Get statistics:
   curl http://localhost:8080/api/v1/stats

5. Health check:
   curl http://localhost:8080/api/v1/health
`)
		os.Exit(0)
	}
}
