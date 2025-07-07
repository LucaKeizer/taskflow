package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"taskflow/internal/types"
	"time"
)

type DataExportProcessor struct{}

func NewDataExportProcessor() *DataExportProcessor {
	return &DataExportProcessor{}
}

func (d *DataExportProcessor) SupportedJobTypes() []types.JobType {
	return []types.JobType{types.JobTypeDataExport}
}

func (d *DataExportProcessor) ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error) {
	// Parse the data export payload
	var payload types.DataExportPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid data export payload: %w", err)
	}

	log.Printf("Exporting data with query: %s to format: %s", payload.Query, payload.ExportType)

	// Process the export
	result, err := d.processExport(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to process export: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, nil
}

func (d *DataExportProcessor) processExport(ctx context.Context, payload types.DataExportPayload) (*types.DataExportResult, error) {
	// Simulate data fetching time
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Generate mock data based on query
	data := d.generateMockData(payload.Query)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(payload.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export data based on format
	var filePath string
	var err error

	switch payload.ExportType {
	case "csv":
		filePath, err = d.exportCSV(data, payload.OutputPath)
	case "json":
		filePath, err = d.exportJSON(data, payload.OutputPath)
	case "xlsx":
		// For demo purposes, we'll create a CSV and pretend it's Excel
		filePath, err = d.exportCSV(data, payload.OutputPath+".csv")
	default:
		return nil, fmt.Errorf("unsupported export type: %s", payload.ExportType)
	}

	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	result := &types.DataExportResult{
		FilePath: filePath,
		FileSize: fileInfo.Size(),
		RowCount: len(data),
		Format:   payload.ExportType,
	}

	log.Printf("📊 Exported %d rows to %s (%d bytes)", result.RowCount, result.Format, result.FileSize)

	return result, nil
}

func (d *DataExportProcessor) generateMockData(query string) []map[string]interface{} {
	// Generate mock data based on query keywords
	rowCount := 100 + rand.Intn(900) // 100-1000 rows

	var data []map[string]interface{}

	for i := 0; i < rowCount; i++ {
		row := map[string]interface{}{
			"id":         i + 1,
			"name":       fmt.Sprintf("Record %d", i+1),
			"value":      rand.Float64() * 1000,
			"created_at": time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour).Format("2006-01-02"),
			"status":     []string{"active", "inactive", "pending"}[rand.Intn(3)],
		}

		// Add query-specific fields
		if contains(query, "user") {
			row["email"] = fmt.Sprintf("user%d@example.com", i+1)
			row["age"] = 18 + rand.Intn(50)
		}

		if contains(query, "order") {
			row["amount"] = rand.Float64() * 500
			row["product"] = fmt.Sprintf("Product %d", rand.Intn(10)+1)
		}

		data = append(data, row)
	}

	return data
}

func (d *DataExportProcessor) exportCSV(data []map[string]interface{}, outputPath string) (string, error) {
	// Ensure CSV extension
	if filepath.Ext(outputPath) != ".csv" {
		outputPath += ".csv"
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if len(data) == 0 {
		return outputPath, nil
	}

	// Write header
	var headers []string
	for key := range data[0] {
		headers = append(headers, key)
	}
	writer.Write(headers)

	// Write data rows
	for _, row := range data {
		var values []string
		for _, header := range headers {
			values = append(values, fmt.Sprintf("%v", row[header]))
		}
		writer.Write(values)
	}

	return outputPath, nil
}

func (d *DataExportProcessor) exportJSON(data []map[string]interface{}, outputPath string) (string, error) {
	// Ensure JSON extension
	if filepath.Ext(outputPath) != ".json" {
		outputPath += ".json"
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(map[string]interface{}{
		"data":        data,
		"total":       len(data),
		"exported_at": time.Now().Format(time.RFC3339),
	})

	return outputPath, err
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
