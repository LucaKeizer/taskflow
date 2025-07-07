package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"taskflow/internal/types"
	"time"
)

type ImageResizeProcessor struct{}

func NewImageResizeProcessor() *ImageResizeProcessor {
	return &ImageResizeProcessor{}
}

func (i *ImageResizeProcessor) SupportedJobTypes() []types.JobType {
	return []types.JobType{types.JobTypeImageResize}
}

func (i *ImageResizeProcessor) ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error) {
	// Parse the image resize payload
	var payload types.ImageResizePayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid image resize payload: %w", err)
	}

	log.Printf("Resizing image %s to sizes: %v", payload.ImageURL, payload.Sizes)

	// Simulate image processing
	result, err := i.processImage(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, nil
}

func (i *ImageResizeProcessor) processImage(ctx context.Context, payload types.ImageResizePayload) (*types.ImageResizeResult, error) {
	// Simulate download time
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate getting original image metadata
	originalWidth := 1920
	originalHeight := 1080
	originalSize := int64(2500000) // 2.5MB

	metadata := types.ImageMetadata{
		OriginalWidth:  originalWidth,
		OriginalHeight: originalHeight,
		OriginalSize:   originalSize,
		Format:         "JPEG",
	}

	var resizedImages []types.ResizedImage

	// Process each requested size
	for _, width := range payload.Sizes {
		// Calculate proportional height
		height := (width * originalHeight) / originalWidth

		// Simulate processing time based on image size
		processingTime := time.Duration(width/100) * time.Millisecond
		select {
		case <-time.After(processingTime):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Simulate file size (smaller images = smaller files)
		sizeRatio := float64(width) / float64(originalWidth)
		newSize := int64(float64(originalSize) * sizeRatio * sizeRatio)

		// Generate mock URL for resized image
		format := payload.Format
		if format == "" {
			format = "jpeg"
		}

		url := fmt.Sprintf("%s/resized_%dx%d.%s",
			payload.OutputPath, width, height, format)

		resizedImage := types.ResizedImage{
			Width:  width,
			Height: height,
			Size:   newSize,
			URL:    url,
		}

		resizedImages = append(resizedImages, resizedImage)

		log.Printf("📸 Resized image to %dx%d (%d bytes)", width, height, newSize)
	}

	result := &types.ImageResizeResult{
		OriginalURL: payload.ImageURL,
		Images:      resizedImages,
		Metadata:    metadata,
	}

	return result, nil
}
