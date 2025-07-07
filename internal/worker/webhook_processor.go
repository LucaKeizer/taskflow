package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"taskflow/internal/types"
	"time"
)

type WebhookProcessor struct {
	client *http.Client
}

func NewWebhookProcessor() *WebhookProcessor {
	return &WebhookProcessor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (w *WebhookProcessor) SupportedJobTypes() []types.JobType {
	return []types.JobType{types.JobTypeWebhook}
}

func (w *WebhookProcessor) ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error) {
	// Parse the webhook payload
	var payload types.WebhookPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}

	log.Printf("Making webhook call to %s", payload.URL)

	start := time.Now()
	result, err := w.makeWebhookCall(ctx, payload)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("webhook call failed: %w", err)
	}

	result.Duration = duration.Milliseconds()

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, nil
}

func (w *WebhookProcessor) makeWebhookCall(ctx context.Context, payload types.WebhookPayload) (*types.WebhookResult, error) {
	// Prepare request body
	var body io.Reader
	if payload.Data != nil {
		jsonData, err := json.Marshal(payload.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(payload.Method), payload.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if payload.Data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range payload.Headers {
		req.Header.Set(key, value)
	}

	// Set custom timeout if specified
	client := w.client
	if payload.Timeout > 0 {
		client = &http.Client{
			Timeout: time.Duration(payload.Timeout) * time.Second,
		}
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	result := &types.WebhookResult{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(responseBody),
		Headers:      responseHeaders,
	}

	log.Printf("🔗 Webhook call to %s completed with status %d", payload.URL, resp.StatusCode)

	return result, nil
}
