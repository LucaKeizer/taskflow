package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all the Prometheus metrics for TaskFlow
type Metrics struct {
	// Job metrics
	JobsTotal          *prometheus.CounterVec
	JobsProcessingTime *prometheus.HistogramVec
	JobsInQueue        prometheus.Gauge
	JobsProcessing     prometheus.Gauge
	JobRetries         *prometheus.CounterVec

	// Worker metrics
	WorkersActive       prometheus.Gauge
	WorkerJobsProcessed *prometheus.CounterVec

	// API metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
	HTTPRequestsSize *prometheus.HistogramVec

	// System metrics
	QueueDepth   *prometheus.GaugeVec
	SystemUptime prometheus.Gauge
	SystemErrors *prometheus.CounterVec
}

var defaultMetrics *Metrics

// Init initializes the metrics system
func Init() *Metrics {
	metrics := &Metrics{
		// Job metrics
		JobsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "taskflow_jobs_total",
				Help: "Total number of jobs processed by status",
			},
			[]string{"type", "status"},
		),
		JobsProcessingTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "taskflow_job_processing_duration_seconds",
				Help:    "Time spent processing jobs",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"type"},
		),
		JobsInQueue: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "taskflow_jobs_in_queue",
				Help: "Number of jobs currently in queue",
			},
		),
		JobsProcessing: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "taskflow_jobs_processing",
				Help: "Number of jobs currently being processed",
			},
		),
		JobRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "taskflow_job_retries_total",
				Help: "Total number of job retries",
			},
			[]string{"type"},
		),

		// Worker metrics
		WorkersActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "taskflow_workers_active",
				Help: "Number of active workers",
			},
		),
		WorkerJobsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "taskflow_worker_jobs_processed_total",
				Help: "Total number of jobs processed by worker",
			},
			[]string{"worker_id", "type"},
		),

		// API metrics
		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "taskflow_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "taskflow_http_request_duration_seconds",
				Help:    "HTTP request duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestsSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "taskflow_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "endpoint"},
		),

		// System metrics
		QueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "taskflow_queue_depth",
				Help: "Number of items in various queues",
			},
			[]string{"queue_name"},
		),
		SystemUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "taskflow_system_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),
		SystemErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "taskflow_system_errors_total",
				Help: "Total number of system errors",
			},
			[]string{"component", "error_type"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		metrics.JobsTotal,
		metrics.JobsProcessingTime,
		metrics.JobsInQueue,
		metrics.JobsProcessing,
		metrics.JobRetries,
		metrics.WorkersActive,
		metrics.WorkerJobsProcessed,
		metrics.HTTPRequests,
		metrics.HTTPDuration,
		metrics.HTTPRequestsSize,
		metrics.QueueDepth,
		metrics.SystemUptime,
		metrics.SystemErrors,
	)

	defaultMetrics = metrics
	return metrics
}

// GetMetrics returns the default metrics instance
func GetMetrics() *Metrics {
	if defaultMetrics == nil {
		return Init()
	}
	return defaultMetrics
}

// Job metric methods

// IncJobsTotal increments the total jobs counter
func (m *Metrics) IncJobsTotal(jobType, status string) {
	m.JobsTotal.WithLabelValues(jobType, status).Inc()
}

// ObserveJobProcessingTime records job processing time
func (m *Metrics) ObserveJobProcessingTime(jobType string, duration time.Duration) {
	m.JobsProcessingTime.WithLabelValues(jobType).Observe(duration.Seconds())
}

// SetJobsInQueue sets the number of jobs in queue
func (m *Metrics) SetJobsInQueue(count int) {
	m.JobsInQueue.Set(float64(count))
}

// SetJobsProcessing sets the number of jobs being processed
func (m *Metrics) SetJobsProcessing(count int) {
	m.JobsProcessing.Set(float64(count))
}

// IncJobRetries increments the job retries counter
func (m *Metrics) IncJobRetries(jobType string) {
	m.JobRetries.WithLabelValues(jobType).Inc()
}

// Worker metric methods

// SetWorkersActive sets the number of active workers
func (m *Metrics) SetWorkersActive(count int) {
	m.WorkersActive.Set(float64(count))
}

// IncWorkerJobsProcessed increments the worker jobs processed counter
func (m *Metrics) IncWorkerJobsProcessed(workerID, jobType string) {
	m.WorkerJobsProcessed.WithLabelValues(workerID, jobType).Inc()
}

// HTTP metric methods

// IncHTTPRequests increments the HTTP requests counter
func (m *Metrics) IncHTTPRequests(method, endpoint string, statusCode int) {
	m.HTTPRequests.WithLabelValues(method, endpoint, strconv.Itoa(statusCode)).Inc()
}

// ObserveHTTPDuration records HTTP request duration
func (m *Metrics) ObserveHTTPDuration(method, endpoint string, duration time.Duration) {
	m.HTTPDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// ObserveHTTPRequestSize records HTTP request size
func (m *Metrics) ObserveHTTPRequestSize(method, endpoint string, size int64) {
	m.HTTPRequestsSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// System metric methods

// SetQueueDepth sets the depth of a named queue
func (m *Metrics) SetQueueDepth(queueName string, depth int) {
	m.QueueDepth.WithLabelValues(queueName).Set(float64(depth))
}

// SetSystemUptime sets the system uptime
func (m *Metrics) SetSystemUptime(uptime time.Duration) {
	m.SystemUptime.Set(uptime.Seconds())
}

// IncSystemErrors increments the system errors counter
func (m *Metrics) IncSystemErrors(component, errorType string) {
	m.SystemErrors.WithLabelValues(component, errorType).Inc()
}

// Middleware for HTTP metrics collection
type MetricsMiddleware struct {
	metrics *Metrics
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(metrics *Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{metrics: metrics}
}

// Handler returns an HTTP middleware that collects metrics
func (mm *MetricsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Process the request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		endpoint := normalizeEndpoint(r.URL.Path)

		mm.metrics.IncHTTPRequests(r.Method, endpoint, wrapped.statusCode)
		mm.metrics.ObserveHTTPDuration(r.Method, endpoint, duration)
		mm.metrics.ObserveHTTPRequestSize(r.Method, endpoint, r.ContentLength)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// normalizeEndpoint normalizes URL paths for metrics (removes IDs)
func normalizeEndpoint(path string) string {
	// Replace UUIDs and numeric IDs with placeholders
	// This prevents high cardinality in metrics
	if len(path) > 20 {
		// Simple heuristic: if path is long, likely contains ID
		return "/api/v1/jobs/{id}"
	}
	return path
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// Convenience functions using default metrics

// IncJobsTotal increments jobs total using default metrics
func IncJobsTotal(jobType, status string) {
	GetMetrics().IncJobsTotal(jobType, status)
}

// ObserveJobProcessingTime records job processing time using default metrics
func ObserveJobProcessingTime(jobType string, duration time.Duration) {
	GetMetrics().ObserveJobProcessingTime(jobType, duration)
}

// SetJobsInQueue sets jobs in queue using default metrics
func SetJobsInQueue(count int) {
	GetMetrics().SetJobsInQueue(count)
}
