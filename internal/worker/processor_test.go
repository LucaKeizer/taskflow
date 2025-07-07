package worker

import (
	"context"
	"encoding/json"
	"taskflow/internal/types"
	"testing"
)

func TestProcessorRegistry(t *testing.T) {
	registry := NewProcessorRegistry()

	// Test that all expected processors are registered
	expectedTypes := []types.JobType{
		types.JobTypeEmail,
		types.JobTypeImageResize,
		types.JobTypeWebhook,
		types.JobTypeDataExport,
	}

	supportedTypes := registry.GetSupportedJobTypes()

	for _, expectedType := range expectedTypes {
		found := false
		for _, supportedType := range supportedTypes {
			if supportedType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected job type %s to be supported, but it wasn't", expectedType)
		}
	}

	// Test getting processors
	for _, jobType := range expectedTypes {
		processor, exists := registry.GetProcessor(jobType)
		if !exists {
			t.Errorf("Expected processor for job type %s to exist", jobType)
		}
		if processor == nil {
			t.Errorf("Expected non-nil processor for job type %s", jobType)
		}
	}

	// Test getting non-existent processor
	_, exists := registry.GetProcessor(types.JobType("nonexistent"))
	if exists {
		t.Error("Expected non-existent processor to return false")
	}
}

func TestEmailProcessor(t *testing.T) {
	processor := NewEmailProcessor()

	// Test supported job types
	supportedTypes := processor.SupportedJobTypes()
	if len(supportedTypes) != 1 || supportedTypes[0] != types.JobTypeEmail {
		t.Errorf("Expected EmailProcessor to support only email jobs, got %v", supportedTypes)
	}

	// Test valid email job processing
	payload := types.EmailPayload{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test body",
	}

	payloadJSON, _ := json.Marshal(payload)
	job := &types.Job{
		ID:      "test-job-1",
		Type:    types.JobTypeEmail,
		Payload: payloadJSON,
	}

	ctx := context.Background()
	result, err := processor.ProcessJob(ctx, job)

	if err != nil {
		t.Errorf("Expected no error processing valid email job, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from email processing")
	}

	// Verify result structure
	var emailResult types.EmailResult
	if err := json.Unmarshal(result, &emailResult); err != nil {
		t.Errorf("Failed to unmarshal email result: %v", err)
	}

	if emailResult.MessageID == "" {
		t.Error("Expected non-empty message ID in result")
	}

	if emailResult.SentAt == "" {
		t.Error("Expected non-empty sent timestamp in result")
	}
}

func TestWebhookProcessor(t *testing.T) {
	processor := NewWebhookProcessor()

	// Test supported job types
	supportedTypes := processor.SupportedJobTypes()
	if len(supportedTypes) != 1 || supportedTypes[0] != types.JobTypeWebhook {
		t.Errorf("Expected WebhookProcessor to support only webhook jobs, got %v", supportedTypes)
	}

	// Test processing a webhook job (to httpbin.org for testing)
	payload := types.WebhookPayload{
		URL:    "https://httpbin.org/post",
		Method: "POST",
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	payloadJSON, _ := json.Marshal(payload)
	job := &types.Job{
		ID:      "test-webhook-1",
		Type:    types.JobTypeWebhook,
		Payload: payloadJSON,
	}

	ctx := context.Background()
	result, err := processor.ProcessJob(ctx, job)

	if err != nil {
		t.Errorf("Expected no error processing webhook job, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from webhook processing")
	}

	// Verify result structure
	var webhookResult types.WebhookResult
	if err := json.Unmarshal(result, &webhookResult); err != nil {
		t.Errorf("Failed to unmarshal webhook result: %v", err)
	}

	if webhookResult.StatusCode == 0 {
		t.Error("Expected non-zero status code in webhook result")
	}

	if webhookResult.Duration == 0 {
		t.Error("Expected non-zero duration in webhook result")
	}
}

func TestImageResizeProcessor(t *testing.T) {
	processor := NewImageResizeProcessor()

	// Test supported job types
	supportedTypes := processor.SupportedJobTypes()
	if len(supportedTypes) != 1 || supportedTypes[0] != types.JobTypeImageResize {
		t.Errorf("Expected ImageResizeProcessor to support only image resize jobs, got %v", supportedTypes)
	}

	// Test processing an image resize job
	payload := types.ImageResizePayload{
		ImageURL:   "https://picsum.photos/800/600",
		Sizes:      []int{100, 300},
		Format:     "jpeg",
		OutputPath: "/tmp/test",
	}

	payloadJSON, _ := json.Marshal(payload)
	job := &types.Job{
		ID:      "test-image-1",
		Type:    types.JobTypeImageResize,
		Payload: payloadJSON,
	}

	ctx := context.Background()
	result, err := processor.ProcessJob(ctx, job)

	if err != nil {
		t.Errorf("Expected no error processing image resize job, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from image processing")
	}

	// Verify result structure
	var imageResult types.ImageResizeResult
	if err := json.Unmarshal(result, &imageResult); err != nil {
		t.Errorf("Failed to unmarshal image result: %v", err)
	}

	if len(imageResult.Images) != 2 {
		t.Errorf("Expected 2 resized images, got %d", len(imageResult.Images))
	}

	if imageResult.OriginalURL != payload.ImageURL {
		t.Errorf("Expected original URL %s, got %s", payload.ImageURL, imageResult.OriginalURL)
	}
}

func TestDataExportProcessor(t *testing.T) {
	processor := NewDataExportProcessor()

	// Test supported job types
	supportedTypes := processor.SupportedJobTypes()
	if len(supportedTypes) != 1 || supportedTypes[0] != types.JobTypeDataExport {
		t.Errorf("Expected DataExportProcessor to support only data export jobs, got %v", supportedTypes)
	}

	// Test processing a data export job
	payload := types.DataExportPayload{
		ExportType: "csv",
		Query:      "SELECT * FROM users",
		OutputPath: "/tmp/test_export",
	}

	payloadJSON, _ := json.Marshal(payload)
	job := &types.Job{
		ID:      "test-export-1",
		Type:    types.JobTypeDataExport,
		Payload: payloadJSON,
	}

	ctx := context.Background()
	result, err := processor.ProcessJob(ctx, job)

	if err != nil {
		t.Errorf("Expected no error processing data export job, got %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result from data export processing")
	}

	// Verify result structure
	var exportResult types.DataExportResult
	if err := json.Unmarshal(result, &exportResult); err != nil {
		t.Errorf("Failed to unmarshal export result: %v", err)
	}

	if exportResult.FilePath == "" {
		t.Error("Expected non-empty file path in export result")
	}

	if exportResult.RowCount == 0 {
		t.Error("Expected non-zero row count in export result")
	}

	if exportResult.Format != "csv" {
		t.Errorf("Expected format 'csv', got %s", exportResult.Format)
	}
}
