package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"taskflow/internal/types"
	"time"
)

type EmailProcessor struct{}

func NewEmailProcessor() *EmailProcessor {
	return &EmailProcessor{}
}

func (e *EmailProcessor) SupportedJobTypes() []types.JobType {
	return []types.JobType{types.JobTypeEmail}
}

func (e *EmailProcessor) ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error) {
	// Parse the email payload
	var payload types.EmailPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid email payload: %w", err)
	}

	log.Printf("Sending email to %s with subject: %s", payload.To, payload.Subject)

	// Simulate email sending (in real implementation, you'd use SMTP or email service)
	err := e.sendEmail(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	// Create result
	result := types.EmailResult{
		MessageID: fmt.Sprintf("msg_%d", time.Now().Unix()),
		SentAt:    time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, nil
}

// sendEmail simulates sending an email
func (e *EmailProcessor) sendEmail(ctx context.Context, payload types.EmailPayload) error {
	// Simulate processing time
	select {
	case <-time.After(time.Duration(1+len(payload.Body)/100) * time.Second):
		// Email "sent" successfully
		log.Printf("✉️  Email sent to %s", payload.To)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
