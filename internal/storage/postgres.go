package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"taskflow/internal/types"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(databaseURL string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PostgresStorage{db: db}

	// Initialize database schema
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// migrate creates the necessary database tables
func (p *PostgresStorage) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id VARCHAR(255) PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			payload JSONB NOT NULL,
			status VARCHAR(20) NOT NULL,
			result JSONB,
			error TEXT,
			attempts INTEGER DEFAULT 0,
			max_attempts INTEGER DEFAULT 3,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
			scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE,
			completed_at TIMESTAMP WITH TIME ZONE,
			worker_id VARCHAR(255)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_scheduled_at ON jobs(scheduled_at)`,
		`CREATE TABLE IF NOT EXISTS workers (
			id VARCHAR(255) PRIMARY KEY,
			status VARCHAR(20) NOT NULL,
			last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
			job_types JSONB NOT NULL,
			current_job VARCHAR(255),
			metadata JSONB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status)`,
		`CREATE INDEX IF NOT EXISTS idx_workers_last_seen ON workers(last_seen)`,
	}

	for _, query := range queries {
		if _, err := p.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	return nil
}

// CreateJob inserts a new job into the database
func (p *PostgresStorage) CreateJob(ctx context.Context, job *types.Job) error {
	query := `
		INSERT INTO jobs (
			id, type, payload, status, result, error, attempts, max_attempts,
			created_at, updated_at, scheduled_at, started_at, completed_at, worker_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Type, job.Payload, job.Status, job.Result, job.Error,
		job.Attempts, job.MaxAttempts, job.CreatedAt, job.UpdatedAt,
		job.ScheduledAt, job.StartedAt, job.CompletedAt, job.WorkerID,
	)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (p *PostgresStorage) GetJob(ctx context.Context, jobID string) (*types.Job, error) {
	query := `
		SELECT id, type, payload, status, result, error, attempts, max_attempts,
			   created_at, updated_at, scheduled_at, started_at, completed_at, worker_id
		FROM jobs WHERE id = $1
	`

	var job types.Job
	var result, payload sql.NullString
	var startedAt, completedAt sql.NullTime
	var workerID sql.NullString

	err := p.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID, &job.Type, &payload, &job.Status, &result, &job.Error,
		&job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt,
		&job.ScheduledAt, &startedAt, &completedAt, &workerID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	// Handle nullable fields
	if payload.Valid {
		job.Payload = json.RawMessage(payload.String)
	}
	if result.Valid {
		job.Result = json.RawMessage(result.String)
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}

	return &job, nil
}

// UpdateJob updates a job in the database
func (p *PostgresStorage) UpdateJob(ctx context.Context, job *types.Job) error {
	query := `
		UPDATE jobs SET
			status = $2, result = $3, error = $4, attempts = $5,
			updated_at = $6, started_at = $7, completed_at = $8, worker_id = $9
		WHERE id = $1
	`

	_, err := p.db.ExecContext(ctx, query,
		job.ID, job.Status, job.Result, job.Error, job.Attempts,
		job.UpdatedAt, job.StartedAt, job.CompletedAt, job.WorkerID,
	)

	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// ListJobs retrieves jobs with pagination and filtering
func (p *PostgresStorage) ListJobs(ctx context.Context, page, pageSize int, status, jobType string) ([]types.Job, int, error) {
	// Build the WHERE clause
	var whereConditions []string
	var args []interface{}
	argIndex := 1

	if status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if jobType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, jobType)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs %s", whereClause)
	var total int
	err := p.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	// Get jobs with pagination
	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, type, payload, status, result, error, attempts, max_attempts,
			   created_at, updated_at, scheduled_at, started_at, completed_at, worker_id
		FROM jobs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := p.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []types.Job
	for rows.Next() {
		var job types.Job
		var result, payload sql.NullString
		var startedAt, completedAt sql.NullTime
		var workerID sql.NullString

		err := rows.Scan(
			&job.ID, &job.Type, &payload, &job.Status, &result, &job.Error,
			&job.Attempts, &job.MaxAttempts, &job.CreatedAt, &job.UpdatedAt,
			&job.ScheduledAt, &startedAt, &completedAt, &workerID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan job: %w", err)
		}

		// Handle nullable fields
		if payload.Valid {
			job.Payload = json.RawMessage(payload.String)
		}
		if result.Valid {
			job.Result = json.RawMessage(result.String)
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if workerID.Valid {
			job.WorkerID = workerID.String
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating jobs: %w", err)
	}

	return jobs, total, nil
}

// RegisterWorker registers or updates a worker
func (p *PostgresStorage) RegisterWorker(ctx context.Context, worker *types.Worker) error {
	jobTypesJSON, err := json.Marshal(worker.JobTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal job types: %w", err)
	}

	query := `
		INSERT INTO workers (id, status, last_seen, job_types, current_job)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen,
			job_types = EXCLUDED.job_types,
			current_job = EXCLUDED.current_job
	`

	_, err = p.db.ExecContext(ctx, query,
		worker.ID, worker.Status, worker.LastSeen, jobTypesJSON, worker.CurrentJob,
	)

	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	return nil
}

// GetWorkers retrieves all active workers
func (p *PostgresStorage) GetWorkers(ctx context.Context) ([]types.Worker, error) {
	// Consider workers active if they've been seen in the last 5 minutes
	query := `
		SELECT id, status, last_seen, job_types, current_job
		FROM workers
		WHERE last_seen > $1
		ORDER BY last_seen DESC
	`

	cutoff := time.Now().Add(-5 * time.Minute)
	rows, err := p.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query workers: %w", err)
	}
	defer rows.Close()

	var workers []types.Worker
	for rows.Next() {
		var worker types.Worker
		var jobTypesJSON string
		var currentJob sql.NullString

		err := rows.Scan(
			&worker.ID, &worker.Status, &worker.LastSeen, &jobTypesJSON, &currentJob,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}

		// Parse job types
		if err := json.Unmarshal([]byte(jobTypesJSON), &worker.JobTypes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal job types: %w", err)
		}

		if currentJob.Valid {
			worker.CurrentJob = currentJob.String
		}

		workers = append(workers, worker)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workers: %w", err)
	}

	return workers, nil
}
