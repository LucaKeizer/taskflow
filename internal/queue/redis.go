package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"taskflow/internal/types"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	JobQueueKey        = "taskflow:jobs:pending"
	ProcessingQueueKey = "taskflow:jobs:processing"
	JobKeyPrefix       = "taskflow:job:"
	WorkerKeyPrefix    = "taskflow:worker:"
	StatsKey           = "taskflow:stats"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(addr, password string, db int) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisQueue{
		client: rdb,
	}
}

// Close closes the Redis connection
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// Ping checks if Redis is available
func (r *RedisQueue) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// EnqueueJob adds a job to the pending queue
func (r *RedisQueue) EnqueueJob(ctx context.Context, job *types.Job) error {
	// Store job details
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	jobKey := JobKeyPrefix + job.ID

	// Use a pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Store job data
	pipe.Set(ctx, jobKey, jobData, 24*time.Hour) // Jobs expire after 24 hours

	// Add job ID to pending queue
	pipe.LPush(ctx, JobQueueKey, job.ID)

	// Update stats
	pipe.HIncrBy(ctx, StatsKey, "total", 1)
	pipe.HIncrBy(ctx, StatsKey, "pending", 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// DequeueJob removes and returns a job from the pending queue
// This is a blocking operation that waits for jobs to be available
func (r *RedisQueue) DequeueJob(ctx context.Context, workerID string, timeout time.Duration) (*types.Job, error) {
	// Use BRPOPLPUSH for atomic move from pending to processing
	result := r.client.BRPopLPush(ctx, JobQueueKey, ProcessingQueueKey, timeout)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, nil // No job available (timeout)
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", result.Err())
	}

	jobID := result.Val()

	// Get job details
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		// If we can't get the job, remove it from processing queue
		r.client.LRem(ctx, ProcessingQueueKey, 1, jobID)
		return nil, err
	}

	// Update job status and worker assignment
	job.Status = types.JobStatusProcessing
	job.WorkerID = workerID
	now := time.Now()
	job.StartedAt = &now
	job.UpdatedAt = now

	err = r.UpdateJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Update stats
	pipe := r.client.Pipeline()
	pipe.HIncrBy(ctx, StatsKey, "pending", -1)
	pipe.HIncrBy(ctx, StatsKey, "processing", 1)
	pipe.Exec(ctx)

	return job, nil
}

// GetJob retrieves a job by ID
func (r *RedisQueue) GetJob(ctx context.Context, jobID string) (*types.Job, error) {
	jobKey := JobKeyPrefix + jobID

	result := r.client.Get(ctx, jobKey)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", result.Err())
	}

	var job types.Job
	err := json.Unmarshal([]byte(result.Val()), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates a job's data in Redis
func (r *RedisQueue) UpdateJob(ctx context.Context, job *types.Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	jobKey := JobKeyPrefix + job.ID
	err = r.client.Set(ctx, jobKey, jobData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// CompleteJob marks a job as completed and removes it from processing queue
func (r *RedisQueue) CompleteJob(ctx context.Context, jobID string, result json.RawMessage) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update job status
	job.Status = types.JobStatusCompleted
	job.Result = result
	now := time.Now()
	job.CompletedAt = &now
	job.UpdatedAt = now

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Update job
	jobData, _ := json.Marshal(job)
	jobKey := JobKeyPrefix + job.ID
	pipe.Set(ctx, jobKey, jobData, 24*time.Hour)

	// Remove from processing queue
	pipe.LRem(ctx, ProcessingQueueKey, 1, jobID)

	// Update stats
	pipe.HIncrBy(ctx, StatsKey, "processing", -1)
	pipe.HIncrBy(ctx, StatsKey, "completed", 1)

	_, err = pipe.Exec(ctx)
	return err
}

// FailJob marks a job as failed
func (r *RedisQueue) FailJob(ctx context.Context, jobID string, errorMsg string) error {
	job, err := r.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Attempts++
	job.Error = errorMsg
	job.UpdatedAt = time.Now()

	// Check if we should retry
	if job.Attempts < job.MaxAttempts {
		job.Status = types.JobStatusRetrying
		// Re-queue the job with delay
		return r.requeueJobWithDelay(ctx, job, calculateRetryDelay(job.Attempts))
	} else {
		job.Status = types.JobStatusFailed
		now := time.Now()
		job.CompletedAt = &now
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Update job
	jobData, _ := json.Marshal(job)
	jobKey := JobKeyPrefix + job.ID
	pipe.Set(ctx, jobKey, jobData, 24*time.Hour)

	// Remove from processing queue
	pipe.LRem(ctx, ProcessingQueueKey, 1, jobID)

	// Update stats
	pipe.HIncrBy(ctx, StatsKey, "processing", -1)
	if job.Status == types.JobStatusFailed {
		pipe.HIncrBy(ctx, StatsKey, "failed", 1)
	} else {
		pipe.HIncrBy(ctx, StatsKey, "pending", 1)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetStats returns job processing statistics
func (r *RedisQueue) GetStats(ctx context.Context) (*types.JobStats, error) {
	result := r.client.HGetAll(ctx, StatsKey)
	if result.Err() != nil {
		return nil, result.Err()
	}

	stats := &types.JobStats{}
	data := result.Val()

	if val, ok := data["total"]; ok {
		fmt.Sscanf(val, "%d", &stats.Total)
	}
	if val, ok := data["pending"]; ok {
		fmt.Sscanf(val, "%d", &stats.Pending)
	}
	if val, ok := data["processing"]; ok {
		fmt.Sscanf(val, "%d", &stats.Processing)
	}
	if val, ok := data["completed"]; ok {
		fmt.Sscanf(val, "%d", &stats.Completed)
	}
	if val, ok := data["failed"]; ok {
		fmt.Sscanf(val, "%d", &stats.Failed)
	}

	return stats, nil
}

// requeueJobWithDelay requeues a job after a delay
func (r *RedisQueue) requeueJobWithDelay(ctx context.Context, job *types.Job, delay time.Duration) error {
	job.ScheduledAt = time.Now().Add(delay)

	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}

	jobKey := JobKeyPrefix + job.ID

	// For now, we'll just put it back in the queue immediately
	// In a production system, you'd want a delayed job scheduler
	pipe := r.client.Pipeline()
	pipe.Set(ctx, jobKey, jobData, 24*time.Hour)
	pipe.LPush(ctx, JobQueueKey, job.ID)
	_, err = pipe.Exec(ctx)

	return err
}

// calculateRetryDelay calculates exponential backoff delay
func calculateRetryDelay(attempts int) time.Duration {
	base := time.Second * 5                            // 5 seconds base delay
	delay := base * time.Duration(1<<uint(attempts-1)) // 2^(attempts-1)

	// Cap at 5 minutes
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}

	return delay
}
