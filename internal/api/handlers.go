package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"taskflow/internal/queue"
	"taskflow/internal/storage"
	"taskflow/internal/types"

	"github.com/gorilla/mux"
)

type Server struct {
	queue   *queue.RedisQueue
	storage *storage.PostgresStorage
	router  *mux.Router
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type ListJobsResponse struct {
	Jobs       []types.Job `json:"jobs"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func NewServer(queue *queue.RedisQueue, storage *storage.PostgresStorage) *Server {
	s := &Server{
		queue:   queue,
		storage: storage,
		router:  mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Job management
	api.HandleFunc("/jobs", s.createJob).Methods("POST")
	api.HandleFunc("/jobs", s.listJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.getJob).Methods("GET")
	api.HandleFunc("/jobs/{id}/cancel", s.cancelJob).Methods("POST")

	// Statistics and monitoring
	api.HandleFunc("/stats", s.getStats).Methods("GET")
	api.HandleFunc("/workers", s.getWorkers).Methods("GET")
	api.HandleFunc("/health", s.healthCheck).Methods("GET")

	// Add CORS middleware
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// createJob handles POST /api/v1/jobs
func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req types.JobRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON payload", err.Error())
		return
	}

	// Validate the request
	if err := types.ValidateJobRequest(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid job request", err.Error())
		return
	}

	// Create the job
	job := types.NewJob(&req)

	// Store in database
	if err := s.storage.CreateJob(r.Context(), job); err != nil {
		log.Printf("Failed to store job in database: %v", err)
		s.sendError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to create job", "")
		return
	}

	// Enqueue for processing
	if err := s.queue.EnqueueJob(r.Context(), job); err != nil {
		log.Printf("Failed to enqueue job: %v", err)
		s.sendError(w, http.StatusInternalServerError, "QUEUE_ERROR", "Failed to enqueue job", "")
		return
	}

	// Return success response
	response := types.JobResponse{
		Job:     job,
		Message: "Job created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getJob handles GET /api/v1/jobs/{id}
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		s.sendError(w, http.StatusBadRequest, "MISSING_ID", "Job ID is required", "")
		return
	}

	// Try to get from queue first (for real-time status)
	job, err := s.queue.GetJob(r.Context(), jobID)
	if err != nil {
		// If not in queue, try database (for historical jobs)
		job, err = s.storage.GetJob(r.Context(), jobID)
		if err != nil {
			s.sendError(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found", "")
			return
		}
	}

	response := types.JobResponse{Job: job}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listJobs handles GET /api/v1/jobs
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := r.URL.Query().Get("status")
	jobType := r.URL.Query().Get("type")

	// Get jobs from database
	jobs, total, err := s.storage.ListJobs(r.Context(), page, pageSize, status, jobType)
	if err != nil {
		log.Printf("Failed to list jobs: %v", err)
		s.sendError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to retrieve jobs", "")
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := ListJobsResponse{
		Jobs:       jobs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// cancelJob handles POST /api/v1/jobs/{id}/cancel
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		s.sendError(w, http.StatusBadRequest, "MISSING_ID", "Job ID is required", "")
		return
	}

	// Get the job
	job, err := s.queue.GetJob(r.Context(), jobID)
	if err != nil {
		job, err = s.storage.GetJob(r.Context(), jobID)
		if err != nil {
			s.sendError(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found", "")
			return
		}
	}

	// Check if job can be cancelled
	if job.Status == types.JobStatusCompleted || job.Status == types.JobStatusFailed {
		s.sendError(w, http.StatusBadRequest, "CANNOT_CANCEL", "Job cannot be cancelled", fmt.Sprintf("Job is already %s", job.Status))
		return
	}

	// Cancel the job (mark as failed with cancellation message)
	err = s.queue.FailJob(r.Context(), jobID, "Job cancelled by user")
	if err != nil {
		log.Printf("Failed to cancel job: %v", err)
		s.sendError(w, http.StatusInternalServerError, "CANCEL_ERROR", "Failed to cancel job", "")
		return
	}

	// Update in database as well
	job.Status = types.JobStatusFailed
	job.Error = "Job cancelled by user"
	s.storage.UpdateJob(r.Context(), job)

	response := types.JobResponse{
		Job:     job,
		Message: "Job cancelled successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getStats handles GET /api/v1/stats
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.queue.GetStats(r.Context())
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		s.sendError(w, http.StatusInternalServerError, "STATS_ERROR", "Failed to retrieve statistics", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// getWorkers handles GET /api/v1/workers
func (s *Server) getWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := s.storage.GetWorkers(r.Context())
	if err != nil {
		log.Printf("Failed to get workers: %v", err)
		s.sendError(w, http.StatusInternalServerError, "WORKERS_ERROR", "Failed to retrieve workers", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workers": workers,
		"count":   len(workers),
	})
}

// healthCheck handles GET /api/v1/health
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":  "healthy",
		"service": "taskflow-api",
	}

	// Check Redis connection
	if err := s.queue.Ping(r.Context()); err != nil {
		health["status"] = "unhealthy"
		health["redis_error"] = err.Error()
	} else {
		health["redis"] = "connected"
	}

	// Check database connection
	if err := s.storage.Ping(r.Context()); err != nil {
		health["status"] = "unhealthy"
		health["database_error"] = err.Error()
	} else {
		health["database"] = "connected"
	}

	status := http.StatusOK
	if health["status"] == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(health)
}

// sendError sends a structured error response
func (s *Server) sendError(w http.ResponseWriter, statusCode int, code, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	json.NewEncoder(w).Encode(errorResp)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
