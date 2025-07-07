package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"taskflow/internal/queue"
	"taskflow/internal/storage"
	"taskflow/internal/worker"
)

func main() {
	log.Printf("Starting TaskFlow Worker...")

	// Configuration from environment variables
	config := getConfig()

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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create workers
	var workers []*worker.Worker
	var wg sync.WaitGroup

	for i := 0; i < config.WorkerCount; i++ {
		w := worker.NewWorker(redisQueue, postgresStorage)
		workers = append(workers, w)

		wg.Add(1)
		go func(w *worker.Worker) {
			defer wg.Done()
			if err := w.Start(ctx); err != nil && err != context.Canceled {
				log.Printf("Worker %s stopped with error: %v", w.ID, err)
			}
		}(w)

		// Stagger worker startup to avoid thundering herd
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Started %d workers", config.WorkerCount)

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down workers...")

	// Cancel context to signal all workers to stop
	cancel()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers shut down gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Force shutdown after timeout")
	}
}

type Config struct {
	WorkerCount   int
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	DatabaseURL   string
}

func getConfig() *Config {
	config := &Config{
		WorkerCount:   getEnvInt("WORKER_COUNT", 3),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://taskflow:taskflow@localhost/taskflow?sslmode=disable"),
	}

	log.Printf("Configuration:")
	log.Printf("  Workers: %d", config.WorkerCount)
	log.Printf("  Redis: %s", config.RedisAddr)
	log.Printf("  Database: %s", config.DatabaseURL)

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion - in production you'd want better error handling
		if value == "1" {
			return 1
		}
		if value == "2" {
			return 2
		}
		if value == "3" {
			return 3
		}
		if value == "4" {
			return 4
		}
		if value == "5" {
			return 5
		}
		if value == "6" {
			return 6
		}
		if value == "7" {
			return 7
		}
		if value == "8" {
			return 8
		}
		if value == "9" {
			return 9
		}
		if value == "10" {
			return 10
		}
	}
	return defaultValue
}
