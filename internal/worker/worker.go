package worker

import (
	"context"
	"fmt"
	"log"
	"taskflow/internal/queue"
	"taskflow/internal/storage"
	"taskflow/internal/types"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	ID             string
	queue          *queue.RedisQueue
	storage        *storage.PostgresStorage
	registry       *ProcessorRegistry
	pollInterval   time.Duration
	shutdown       chan struct{}
	supportedTypes []types.JobType
}

func NewWorker(queue *queue.RedisQueue, storage *storage.PostgresStorage) *Worker {
	registry := NewProcessorRegistry()
	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])

	return &Worker{
		ID:             workerID,
		queue:          queue,
		storage:        storage,
		registry:       registry,
		pollInterval:   5 * time.Second,
		shutdown:       make(chan struct{}),
		supportedTypes: registry.GetSupportedJobTypes(),
	}
}

// Start begins the worker's job processing loop
func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Starting worker %s", w.ID)
	log.Printf("Supported job types: %v", w.supportedTypes)

	// Register worker in database
	if err := w.registerWorker(ctx); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Start heartbeat goroutine
	go w.heartbeat(ctx)

	// Main processing loop
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s shutting down due to context cancellation", w.ID)
			return ctx.Err()
		case <-w.shutdown:
			log.Printf("Worker %s shutting down", w.ID)
			return nil
		default:
			if err := w.processNextJob(ctx); err != nil {
				log.Printf("Error processing job: %v", err)
				// Continue processing other jobs even if one fails
			}
		}
	}
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() {
	close(w.shutdown)
}

// processNextJob fetches and processes the next available job
func (w *Worker) processNextJob(ctx context.Context) error {
	// Try to dequeue a job (with timeout)
	job, err := w.queue.DequeueJob(ctx, w.ID, w.pollInterval)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	// No job available
	if job == nil {
		return nil
	}

	log.Printf("Worker %s processing job %s (type: %s)", w.ID, job.ID, job.Type)

	// Update worker status
	w.updateWorkerStatus(ctx, "processing", job.ID)

	// Process the job
	startTime := time.Now()
	result, err := w.registry.ProcessJob(ctx, job)
	processingDuration := time.Since(startTime)

	if err != nil {
		// Job failed
		log.Printf("Job %s failed after %v: %v", job.ID, processingDuration, err)

		// Check if error is retryable
		if types.IsRetryableError(err) && job.Attempts < job.MaxAttempts {
			log.Printf("Job %s will be retried (attempt %d/%d)", job.ID, job.Attempts+1, job.MaxAttempts)
		}

		if err := w.queue.FailJob(ctx, job.ID, err.Error()); err != nil {
			log.Printf("Failed to mark job as failed: %v", err)
		}

		// Update job in database
		job.Status = types.JobStatusFailed
		job.Error = err.Error()
		job.Attempts++
		now := time.Now()
		job.UpdatedAt = now
		if job.Attempts >= job.MaxAttempts {
			job.CompletedAt = &now
		}
		w.storage.UpdateJob(ctx, job)
	} else {
		// Job succeeded
		log.Printf("Job %s completed successfully in %v", job.ID, processingDuration)

		if err := w.queue.CompleteJob(ctx, job.ID, result); err != nil {
			log.Printf("Failed to mark job as completed: %v", err)
		}

		// Update job in database
		job.Status = types.JobStatusCompleted
		job.Result = result
		now := time.Now()
		job.UpdatedAt = now
		job.CompletedAt = &now
		w.storage.UpdateJob(ctx, job)
	}

	// Update worker status back to idle
	w.updateWorkerStatus(ctx, "idle", "")

	return nil
}

// registerWorker registers this worker in the database
func (w *Worker) registerWorker(ctx context.Context) error {
	worker := &types.Worker{
		ID:       w.ID,
		Status:   "starting",
		LastSeen: time.Now(),
		JobTypes: w.supportedTypes,
	}

	return w.storage.RegisterWorker(ctx, worker)
}

// heartbeat sends periodic heartbeats to indicate the worker is alive
func (w *Worker) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdown:
			return
		case <-ticker.C:
			w.updateWorkerStatus(ctx, "idle", "")
		}
	}
}

// updateWorkerStatus updates the worker's status in the database
func (w *Worker) updateWorkerStatus(ctx context.Context, status, currentJob string) {
	worker := &types.Worker{
		ID:         w.ID,
		Status:     status,
		LastSeen:   time.Now(),
		JobTypes:   w.supportedTypes,
		CurrentJob: currentJob,
	}

	if err := w.storage.RegisterWorker(ctx, worker); err != nil {
		log.Printf("Failed to update worker status: %v", err)
	}
}
