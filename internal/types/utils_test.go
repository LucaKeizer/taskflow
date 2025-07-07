package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateJobID(t *testing.T) {
	id1 := GenerateJobID()
	id2 := GenerateJobID()

	if id1 == id2 {
		t.Error("Expected unique job IDs, got duplicates")
	}

	if len(id1) != 32 {
		t.Errorf("Expected job ID length 32, got %d", len(id1))
	}
}

func TestNewJob(t *testing.T) {
	payload := json.RawMessage(`{"test": "data"}`)
	req := &JobRequest{
		Type:        JobTypeEmail,
		Payload:     payload,
		MaxAttempts: 5,
	}

	job := NewJob(req)

	if job.Type != JobTypeEmail {
		t.Errorf("Expected job type %s, got %s", JobTypeEmail, job.Type)
	}

	if job.MaxAttempts != 5 {
		t.Errorf("Expected max attempts 5, got %d", job.MaxAttempts)
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected status pending, got %s", job.Status)
	}

	if job.Attempts != 0 {
		t.Errorf("Expected 0 attempts, got %d", job.Attempts)
	}

	if job.ID == "" {
		t.Error("Expected non-empty job ID")
	}
}

func TestNewJobWithScheduledTime(t *testing.T) {
	payload := json.RawMessage(`{"test": "data"}`)
	scheduledTime := time.Now().Add(1 * time.Hour)

	req := &JobRequest{
		Type:        JobTypeWebhook,
		Payload:     payload,
		ScheduledAt: &scheduledTime,
	}

	job := NewJob(req)

	if !job.ScheduledAt.Equal(scheduledTime) {
		t.Errorf("Expected scheduled time %v, got %v", scheduledTime, job.ScheduledAt)
	}
}

func TestValidateJobRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *JobRequest
		wantErr bool
	}{
		{
			name: "valid email job",
			request: &JobRequest{
				Type:    JobTypeEmail,
				Payload: json.RawMessage(`{"to": "test@example.com", "subject": "Test", "body": "Test body"}`),
			},
			wantErr: false,
		},
		{
			name: "missing job type",
			request: &JobRequest{
				Payload: json.RawMessage(`{"test": "data"}`),
			},
			wantErr: true,
		},
		{
			name: "missing payload",
			request: &JobRequest{
				Type: JobTypeEmail,
			},
			wantErr: true,
		},
		{
			name: "invalid job type",
			request: &JobRequest{
				Type:    JobType("invalid"),
				Payload: json.RawMessage(`{"test": "data"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid email payload - missing to",
			request: &JobRequest{
				Type:    JobTypeEmail,
				Payload: json.RawMessage(`{"subject": "Test"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid email payload - missing subject",
			request: &JobRequest{
				Type:    JobTypeEmail,
				Payload: json.RawMessage(`{"to": "test@example.com"}`),
			},
			wantErr: true,
		},
		{
			name: "valid webhook job",
			request: &JobRequest{
				Type:    JobTypeWebhook,
				Payload: json.RawMessage(`{"url": "https://example.com/webhook", "method": "POST"}`),
			},
			wantErr: false,
		},
		{
			name: "invalid webhook payload - missing URL",
			request: &JobRequest{
				Type:    JobTypeWebhook,
				Payload: json.RawMessage(`{"method": "POST"}`),
			},
			wantErr: true,
		},
		{
			name: "valid image resize job",
			request: &JobRequest{
				Type:    JobTypeImageResize,
				Payload: json.RawMessage(`{"image_url": "https://example.com/image.jpg", "sizes": [100, 200]}`),
			},
			wantErr: false,
		},
		{
			name: "invalid image resize payload - missing sizes",
			request: &JobRequest{
				Type:    JobTypeImageResize,
				Payload: json.RawMessage(`{"image_url": "https://example.com/image.jpg"}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJobRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJobRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection refused",
			err:      &MockError{msg: "connection refused"},
			expected: true,
		},
		{
			name:     "timeout error",
			err:      &MockError{msg: "timeout occurred"},
			expected: true,
		},
		{
			name:     "network unreachable",
			err:      &MockError{msg: "network is unreachable"},
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      &MockError{msg: "invalid credentials"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// MockError is a simple error implementation for testing
type MockError struct {
	msg string
}

func (e *MockError) Error() string {
	return e.msg
}
