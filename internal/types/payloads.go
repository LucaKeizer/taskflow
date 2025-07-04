package types

// EmailPayload represents the data needed for email jobs
type EmailPayload struct {
	To      string            `json:"to"`
	CC      []string          `json:"cc,omitempty"`
	BCC     []string          `json:"bcc,omitempty"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	HTML    bool              `json:"html,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// EmailResult represents the result of an email job
type EmailResult struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}

// ImageResizePayload represents the data needed for image resize jobs
type ImageResizePayload struct {
	ImageURL     string `json:"image_url"`
	Sizes        []int  `json:"sizes"`       // [100, 300, 500] - widths in pixels
	Format       string `json:"format"`      // "jpeg", "png", "webp"
	Quality      int    `json:"quality"`     // 1-100 for JPEG
	OutputPath   string `json:"output_path"` // S3 bucket path or local path
	PreserveMeta bool   `json:"preserve_meta,omitempty"`
}

// ImageResizeResult represents the result of an image resize job
type ImageResizeResult struct {
	OriginalURL string         `json:"original_url"`
	Images      []ResizedImage `json:"images"`
	Metadata    ImageMetadata  `json:"metadata,omitempty"`
}

// ResizedImage represents a single resized image
type ResizedImage struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"` // File size in bytes
	URL    string `json:"url"`  // Final URL where image is stored
}

// ImageMetadata represents metadata extracted from the original image
type ImageMetadata struct {
	OriginalWidth  int    `json:"original_width"`
	OriginalHeight int    `json:"original_height"`
	OriginalSize   int64  `json:"original_size"`
	Format         string `json:"format"`
}

// WebhookPayload represents the data needed for webhook jobs
type WebhookPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"` // GET, POST, PUT, etc.
	Headers map[string]string `json:"headers,omitempty"`
	Data    interface{}       `json:"data,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // Timeout in seconds
}

// WebhookResult represents the result of a webhook job
type WebhookResult struct {
	StatusCode   int               `json:"status_code"`
	ResponseBody string            `json:"response_body,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Duration     int64             `json:"duration_ms"`
}

// DataExportPayload represents the data needed for data export jobs
type DataExportPayload struct {
	ExportType string                 `json:"export_type"` // "csv", "json", "xlsx"
	Query      string                 `json:"query"`       // SQL query or data source
	Format     map[string]interface{} `json:"format,omitempty"`
	OutputPath string                 `json:"output_path"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
}

// DataExportResult represents the result of a data export job
type DataExportResult struct {
	FilePath string `json:"file_path"`
	FileSize int64  `json:"file_size"`
	RowCount int    `json:"row_count"`
	Format   string `json:"format"`
}
