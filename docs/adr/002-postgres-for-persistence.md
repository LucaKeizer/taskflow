# ADR-002: PostgreSQL for Job Persistence

## Status
Accepted

## Context
TaskFlow requires persistent storage for job history, audit trails, and analytics. While Redis handles the active job queue efficiently, we need a durable storage system for:

- Completed job results and metadata
- Failed job details and error messages
- Worker registration and health status
- Historical analytics and reporting
- Audit trails for compliance

## Decision
We will use PostgreSQL as our persistent storage layer alongside Redis queuing.

## Rationale

### Why PostgreSQL?

**Data Integrity:**
- ACID transactions ensure data consistency
- Strong typing prevents data corruption
- Foreign key constraints maintain referential integrity
- Mature backup and recovery ecosystem

**Query Capabilities:**
- Rich SQL support for complex analytics queries
- Full-text search for job payload searching
- JSON/JSONB support for flexible payload storage
- Excellent indexing for performance optimization

**Operational Maturity:**
- Battle-tested in production environments
- Extensive monitoring and tooling ecosystem
- Well-understood performance characteristics
- Strong community and documentation

**Go Integration:**
- Excellent driver support (lib/pq, pgx)
- Strong ORM options if needed
- Connection pooling libraries
- Migration tool ecosystem

### Alternatives Considered

**MongoDB:**
- Pros: Native JSON storage, flexible schema
- Cons: Eventually consistent, less mature for transactional workloads

**MySQL:**
- Pros: Widespread adoption, good performance
- Cons: Less advanced JSON support, weaker consistency guarantees

**SQLite:**
- Pros: Zero configuration, embedded
- Cons: No concurrent writes, not suitable for distributed systems

**Time Series Databases (InfluxDB):**
- Pros: Optimized for metrics and analytics
- Cons: Limited transactional support, overkill for job metadata

## Implementation Details

### Schema Design
```sql
-- Jobs table for all job records
CREATE TABLE jobs (
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
);

-- Workers table for worker registration
CREATE TABLE workers (
    id VARCHAR(255) PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL,
    job_types JSONB NOT NULL,
    current_job VARCHAR(255),
    metadata JSONB
);
```

### Index Strategy
```sql
-- Performance indexes
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at);
CREATE INDEX idx_workers_status ON workers(status);
CREATE INDEX idx_workers_last_seen ON workers(last_seen);

-- JSON indexes for payload queries
CREATE INDEX idx_jobs_payload_gin ON jobs USING GIN (payload);
```

### Connection Management
- Connection pooling with configurable limits
- Prepared statements for common queries
- Read replicas for analytics queries (future)
- Connection health monitoring

## Data Flow

### Job Lifecycle
1. **Creation**: Job inserted into PostgreSQL with status 'pending'
2. **Queuing**: Job ID pushed to Redis queue
3. **Processing**: Job status updated to 'processing' in PostgreSQL
4. **Completion**: Result stored in PostgreSQL, removed from Redis
5. **Analysis**: Historical data queried from PostgreSQL

### Consistency Model
- Redis is source of truth for active jobs
- PostgreSQL is source of truth for job history
- Eventual consistency between systems with reconciliation

## Consequences

### Positive
- Durable storage for compliance and auditing
- Rich query capabilities for analytics
- Mature tooling for backup, monitoring, and maintenance
- Strong consistency guarantees for critical data

### Negative
- Additional operational complexity (two storage systems)
- Potential consistency issues between Redis and PostgreSQL
- Higher resource usage compared to single storage system

### Risks and Mitigations
- **Inconsistency**: Implement reconciliation jobs to detect and fix mismatches
- **Performance**: Use appropriate indexes and query optimization
- **Storage growth**: Implement job retention policies and archiving

## Performance Considerations

### Write Optimization
- Batch inserts for high-volume job creation
- Asynchronous updates where consistency allows
- Prepared statements for repeated operations

### Read Optimization
- Indexes on commonly queried columns
- Pagination for large result sets
- Read replicas for analytics workloads

### Maintenance
- Regular VACUUM and ANALYZE operations
- Index maintenance and monitoring
- Partition large tables by date if needed

## Monitoring and Observability
- Connection pool metrics
- Query performance monitoring
- Storage usage and growth trends
- Replication lag (when applicable)
- Lock contention detection

## Future Considerations
- Read replicas for scaling analytics queries
- Partitioning for very large job volumes
- Archival strategy for old job data
- Potential migration to managed PostgreSQL services

## Migration Strategy
- Schema versioning with migration tools
- Backward-compatible changes where possible
- Rolling deployments with database compatibility
- Data migration testing and rollback procedures

## References
- [PostgreSQL JSON Functions](https://www.postgresql.org/docs/current/functions-json.html)
- [PostgreSQL Performance Tips](https://www.postgresql.org/docs/current/performance-tips.html)
- [Go PostgreSQL Best Practices](https://github.com/lib/pq)