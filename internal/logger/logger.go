package logger

import (
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with our custom fields and methods
type Logger struct {
	*logrus.Logger
}

// Fields type for structured logging
type Fields map[string]interface{}

var defaultLogger *Logger

// Init initializes the global logger with the specified configuration
func Init(level, format string) *Logger {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set output format
	if strings.ToLower(format) == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Set output destination
	logger.SetOutput(os.Stdout)

	defaultLogger = &Logger{Logger: logger}
	return defaultLogger
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		return Init("info", "json")
	}
	return defaultLogger
}

// SetOutput sets the logger output destination
func (l *Logger) SetOutput(output io.Writer) {
	l.Logger.SetOutput(output)
}

// WithFields creates a new logger entry with structured fields
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithJobID creates a logger entry with job ID field
func (l *Logger) WithJobID(jobID string) *logrus.Entry {
	return l.WithFields(Fields{"job_id": jobID})
}

// WithWorkerID creates a logger entry with worker ID field
func (l *Logger) WithWorkerID(workerID string) *logrus.Entry {
	return l.WithFields(Fields{"worker_id": workerID})
}

// WithJobType creates a logger entry with job type field
func (l *Logger) WithJobType(jobType string) *logrus.Entry {
	return l.WithFields(Fields{"job_type": jobType})
}

// WithRequestID creates a logger entry with request ID field
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithFields(Fields{"request_id": requestID})
}

// WithError creates a logger entry with error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// JobStarted logs when a job starts processing
func (l *Logger) JobStarted(jobID, jobType, workerID string) {
	l.WithFields(Fields{
		"job_id":    jobID,
		"job_type":  jobType,
		"worker_id": workerID,
		"event":     "job_started",
	}).Info("Job processing started")
}

// JobCompleted logs when a job completes successfully
func (l *Logger) JobCompleted(jobID, jobType, workerID string, duration int64) {
	l.WithFields(Fields{
		"job_id":      jobID,
		"job_type":    jobType,
		"worker_id":   workerID,
		"duration_ms": duration,
		"event":       "job_completed",
	}).Info("Job completed successfully")
}

// JobFailed logs when a job fails
func (l *Logger) JobFailed(jobID, jobType, workerID string, attempts int, err error) {
	l.WithFields(Fields{
		"job_id":    jobID,
		"job_type":  jobType,
		"worker_id": workerID,
		"attempts":  attempts,
		"event":     "job_failed",
	}).WithError(err).Error("Job processing failed")
}

// JobRetrying logs when a job is being retried
func (l *Logger) JobRetrying(jobID, jobType string, attempts, maxAttempts int) {
	l.WithFields(Fields{
		"job_id":       jobID,
		"job_type":     jobType,
		"attempts":     attempts,
		"max_attempts": maxAttempts,
		"event":        "job_retrying",
	}).Warn("Job will be retried")
}

// WorkerStarted logs when a worker starts
func (l *Logger) WorkerStarted(workerID string, supportedTypes []string) {
	l.WithFields(Fields{
		"worker_id":       workerID,
		"supported_types": supportedTypes,
		"event":           "worker_started",
	}).Info("Worker started")
}

// WorkerStopped logs when a worker stops
func (l *Logger) WorkerStopped(workerID string, reason string) {
	l.WithFields(Fields{
		"worker_id": workerID,
		"reason":    reason,
		"event":     "worker_stopped",
	}).Info("Worker stopped")
}

// APIRequest logs HTTP API requests
func (l *Logger) APIRequest(method, path, remoteAddr string, statusCode int, duration int64) {
	l.WithFields(Fields{
		"method":      method,
		"path":        path,
		"remote_addr": remoteAddr,
		"status_code": statusCode,
		"duration_ms": duration,
		"event":       "api_request",
	}).Info("API request processed")
}

// SystemStarted logs when the system starts
func (l *Logger) SystemStarted(component, version string) {
	l.WithFields(Fields{
		"component": component,
		"version":   version,
		"event":     "system_started",
	}).Info("System started successfully")
}

// SystemStopping logs when the system is shutting down
func (l *Logger) SystemStopping(component, reason string) {
	l.WithFields(Fields{
		"component": component,
		"reason":    reason,
		"event":     "system_stopping",
	}).Info("System shutting down")
}

// DatabaseConnected logs successful database connection
func (l *Logger) DatabaseConnected(dbType string) {
	l.WithFields(Fields{
		"db_type": dbType,
		"event":   "database_connected",
	}).Info("Database connected successfully")
}

// RedisConnected logs successful Redis connection
func (l *Logger) RedisConnected(addr string) {
	l.WithFields(Fields{
		"redis_addr": addr,
		"event":      "redis_connected",
	}).Info("Redis connected successfully")
}

// Convenience functions that use the default logger

// Debug logs a debug message
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// WithFields creates a logger entry with fields using the default logger
func WithFields(fields Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}
