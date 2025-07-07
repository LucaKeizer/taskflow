package queue

import (
	"context"
	"encoding/json"
	"taskflow/internal/types"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// BenchmarkConcurrentEnqueue tests concurrent job enqueueing
func BenchmarkConcurrentEnqueue(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	client.FlushDB(context.Background())
	queue := &RedisQueue{client: client}
	ctx := context.Background()

	payload := types.WebhookPayload{
		URL:    "https://httpbin.org/post",
		Method: "POST",
		Data:   map[string]interface{}{"test": "data"},
	}
	payloadJSON, _ := json.Marshal(payload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			job := &types.Job{
				ID:          types.GenerateJobID(),
				Type:        types.JobTypeWebhook,
				Payload:     payloadJSON,
				Status:      types.JobStatusPending,
				MaxAttempts: 3,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				ScheduledAt: time.Now(),
			}

			if err := queue.EnqueueJob(ctx, job); err != nil {
				b.Fatalf("Failed to enqueue job: %v", err)
			}
		}
	})
}

// BenchmarkGetStats tests statistics retrieval performance
func BenchmarkGetStats(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	queue := &RedisQueue{client: client}
	ctx := context.Background()

	// Pre-populate some stats
	client.HSet(ctx, StatsKey, map[string]interface{}{
		"total":      1000,
		"pending":    50,
		"processing": 10,
		"completed":  900,
		"failed":     40,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queue.GetStats(ctx)
		if err != nil {
			b.Fatalf("Failed to get stats: %v", err)
		}
	}
}

// BenchmarkEnqueueJob tests job enqueueing performance
func BenchmarkEnqueueJob(b *testing.B) {
	// Setup Redis client for benchmarking
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use different DB for benchmarks
	})
	defer client.Close()

	// Clear test data
	client.FlushDB(context.Background())

	queue := &RedisQueue{client: client}
	ctx := context.Background()

	// Create test job
	payload := types.EmailPayload{
		To:      "benchmark@example.com",
		Subject: "Benchmark Test",
		Body:    "This is a benchmark test email",
	}
	payloadJSON, _ := json.Marshal(payload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			job := &types.Job{
				ID:          types.GenerateJobID(),
				Type:        types.JobTypeEmail,
				Payload:     payloadJSON,
				Status:      types.JobStatusPending,
				MaxAttempts: 3,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				ScheduledAt: time.Now(),
			}

			if err := queue.EnqueueJob(ctx, job); err != nil {
				b.Fatalf("Failed to enqueue job: %v", err)
			}
		}
	})
}

// BenchmarkDequeueJob tests job dequeueing performance
func BenchmarkDequeueJob(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	client.FlushDB(context.Background())
	queue := &RedisQueue{client: client}
	ctx := context.Background()

	// Pre-populate queue with jobs
	payload := types.EmailPayload{
		To:      "benchmark@example.com",
		Subject: "Benchmark Test",
		Body:    "This is a benchmark test email",
	}
	payloadJSON, _ := json.Marshal(payload)

	for i := 0; i < b.N; i++ {
		job := &types.Job{
			ID:          types.GenerateJobID(),
			Type:        types.JobTypeEmail,
			Payload:     payloadJSON,
			Status:      types.JobStatusPending,
			MaxAttempts: 3,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ScheduledAt: time.Now(),
		}
		queue.EnqueueJob(ctx, job)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			job, err := queue.DequeueJob(ctx, "bench-worker", 1*time.Second)
			if err != nil {
				b.Fatalf("Failed to dequeue job: %v", err)
			}
			if job == nil {
				b.Fatal("Expected job, got nil")
			}
		}
	})
}

// BenchmarkJobProcessingCycle tests complete job lifecycle
func BenchmarkJobProcessingCycle(b *testing.B) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	client.FlushDB(context.Background())
	queue := &RedisQueue{client: client}
	ctx := context.Background()

	payload := types.EmailPayload{
		To:      "benchmark@example.com",
		Subject: "Benchmark Test",
		Body:    "This is a benchmark test email",
	}
	payloadJSON, _ := json.Marshal(payload)

	result := types.EmailResult{
		MessageID: "msg_12345",
		SentAt:    time.Now().Format(time.RFC3339),
	}
	resultJSON, _ := json.Marshal(result)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create and enqueue job
		job := &types.Job{
			ID:          types.GenerateJobID(),
			Type:        types.JobTypeEmail,
			Payload:     payloadJSON,
			Status:      types.JobStatusPending,
			MaxAttempts: 3,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ScheduledAt: time.Now(),
		}

		// Enqueue
		if err := queue.EnqueueJob(ctx, job); err != nil {
			b.Fatalf("Failed to enqueue job: %v", err)
		}

		// Dequeue
		dequeuedJob, err := queue.DequeueJob(ctx, "bench-worker", 1*time.Second)
		if err != nil {
			b.Fatalf("Failed to dequeue job: %v", err)
		}

		// Complete
		if err := queue.CompleteJob(ctx, dequeuedJob.ID, resultJSON); err != nil {
			b.Fatalf("Failed to complete job: %v", err)
		}
	}
}

//
