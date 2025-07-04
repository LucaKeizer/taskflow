package types

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// GenerateJobID generates a unique job ID
func GenerateJobID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewJob creates a new job from a request
func NewJob(req *JobRequest) *Job {
	now := time.Now()

	job := &Job{
		ID:          GenerateJobID(),
		Type:        req.Type,
		Payload:     req.Payload,
		Status:      JobStatusPending,
		Attempts:    0,
		MaxAttempts: 3, // Default to 3 attempts
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: now,
	}

	// Override max attempts if specified
	if req.MaxAttempts > 0 {
		job.MaxAttempts = req.MaxAttempts
	}

	// Override scheduled time if specified
	if req.ScheduledAt != nil {
		job.ScheduledAt = *req.ScheduledAt
	}

	return job
}

// ValidateJobRequest validates a job request
func ValidateJobRequest(req *JobRequest) error {
	if req.Type == "" {
		return fmt.Errorf("job type is required")
	}

	if len(req.Payload) == 0 {
		return fmt.Errorf("job payload is required")
	}

	// Validate job type
	switch req.Type {
	case JobTypeEmail, JobTypeImageResize, JobTypeWebhook, JobTypeDataExport:
		// Valid job types
	default:
		return fmt.Errorf("invalid job type: %s", req.Type)
	}

	// Validate payload structure based on job type
	return validatePayloadStructure(req.Type, req.Payload)
}

// validatePayloadStructure validates that the payload matches the expected structure for the job type
func validatePayloadStructure(jobType JobType, payload json.RawMessage) error {
	switch jobType {
	case JobTypeEmail:
		var emailPayload EmailPayload
		if err := json.Unmarshal(payload, &emailPayload); err != nil {
			return fmt.Errorf("invalid email payload: %w", err)
		}
		if emailPayload.To == "" {
			return fmt.Errorf("email 'to' field is required")
		}
		if emailPayload.Subject == "" {
			return fmt.Errorf("email 'subject' field is required")
		}

	case JobTypeImageResize:
		var imagePayload ImageResizePayload
		if err := json.Unmarshal(payload, &imagePayload); err != nil {
			return fmt.Errorf("invalid image resize payload: %w", err)
		}
		if imagePayload.ImageURL == "" {
			return fmt.Errorf("image_url is required")
		}
		if len(imagePayload.Sizes) == 0 {
			return fmt.Errorf("at least one size is required")
		}

	case JobTypeWebhook:
		var webhookPayload WebhookPayload
		if err := json.Unmarshal(payload, &webhookPayload); err != nil {
			return fmt.Errorf("invalid webhook payload: %w", err)
		}
		if webhookPayload.URL == "" {
			return fmt.Errorf("webhook URL is required")
		}
		if webhookPayload.Method == "" {
			webhookPayload.Method = "POST" // Default to POST
		}

	case JobTypeDataExport:
		var exportPayload DataExportPayload
		if err := json.Unmarshal(payload, &exportPayload); err != nil {
			return fmt.Errorf("invalid data export payload: %w", err)
		}
		if exportPayload.ExportType == "" {
			return fmt.Errorf("export_type is required")
		}
		if exportPayload.Query == "" {
			return fmt.Errorf("query is required")
		}
	}

	return nil
}

// IsRetryableError determines if an error should trigger a job retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorString := err.Error()

	// Network-related errors are usually retryable
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no route to host",
		"connection reset",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorString, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					hasSubstring(s, substr))))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
