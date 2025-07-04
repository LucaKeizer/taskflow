package types

import (
	"encoding/json"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
)

// JobType represents different types of jobs we can process
type JobType string

const (
	JobTypeEmail       JobType = "email"
	JobTypeImageResize JobType = "image_resize"
	JobTypeWebhook     JobType = "webhook"
	JobTypeDataExport  JobType = "data_export"
)

// Job represents a task to be processed
type Job struct {
	ID          string          `json:"id" db:"id"`
	Type        JobType         `json:"type" db:"type"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	Status      JobStatus       `json:"status" db:"status"`
	Result      json.RawMessage `json:"result,omitempty" db:"result"`
	Error       string          `json:"error,omitempty" db:"error"`
	Attempts    int             `json:"attempts" db:"attempts"`
	MaxAttempts int             `json:"max_attempts" db:"max_attempts"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	ScheduledAt time.Time       `json:"scheduled_at" db:"scheduled_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	WorkerID    string          `json:"worker_id,omitempty" db:"worker_id"`
}

// JobRequest represents a request to create a new job
type JobRequest struct {
	Type        JobType         `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	MaxAttempts int             `json:"max_attempts,omitempty"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
}

// JobResponse represents the response when creating or querying a job
type JobResponse struct {
	Job     *Job   `json:"job"`
	Message string `json:"message,omitempty"`
}

// Worker represents a worker instance
type Worker struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	LastSeen   time.Time `json:"last_seen"`
	JobTypes   []JobType `json:"job_types"`
	CurrentJob string    `json:"current_job,omitempty"`
}

// JobStats represents statistics about job processing
type JobStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}
