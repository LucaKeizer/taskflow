package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"taskflow/internal/types"
)

// JobProcessor defines the interface for processing different job types
type JobProcessor interface {
	ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error)
	SupportedJobTypes() []types.JobType
}

// ProcessorRegistry holds all available job processors
type ProcessorRegistry struct {
	processors map[types.JobType]JobProcessor
}

func NewProcessorRegistry() *ProcessorRegistry {
	registry := &ProcessorRegistry{
		processors: make(map[types.JobType]JobProcessor),
	}

	// Register default processors
	registry.RegisterProcessor(NewEmailProcessor())
	registry.RegisterProcessor(NewImageResizeProcessor())
	registry.RegisterProcessor(NewWebhookProcessor())
	registry.RegisterProcessor(NewDataExportProcessor())

	return registry
}

func (r *ProcessorRegistry) RegisterProcessor(processor JobProcessor) {
	for _, jobType := range processor.SupportedJobTypes() {
		r.processors[jobType] = processor
		log.Printf("Registered processor for job type: %s", jobType)
	}
}

func (r *ProcessorRegistry) GetProcessor(jobType types.JobType) (JobProcessor, bool) {
	processor, exists := r.processors[jobType]
	return processor, exists
}

func (r *ProcessorRegistry) GetSupportedJobTypes() []types.JobType {
	var jobTypes []types.JobType
	for jobType := range r.processors {
		jobTypes = append(jobTypes, jobType)
	}
	return jobTypes
}

// ProcessJob processes a job using the appropriate processor
func (r *ProcessorRegistry) ProcessJob(ctx context.Context, job *types.Job) (json.RawMessage, error) {
	processor, exists := r.GetProcessor(job.Type)
	if !exists {
		return nil, fmt.Errorf("no processor found for job type: %s", job.Type)
	}

	log.Printf("Processing job %s of type %s", job.ID, job.Type)

	result, err := processor.ProcessJob(ctx, job)
	if err != nil {
		log.Printf("Job %s failed: %v", job.ID, err)
		return nil, err
	}

	log.Printf("Job %s completed successfully", job.ID)
	return result, nil
}
