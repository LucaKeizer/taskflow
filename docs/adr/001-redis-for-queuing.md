# ADR-001: Redis for Job Queuing

## Status
Accepted

## Context
TaskFlow needs a reliable, fast queuing system to handle job distribution between the API server and worker processes. The system must support:

- High throughput (thousands of jobs per minute)
- Atomic operations for job state transitions
- Blocking operations for efficient worker polling
- Persistence for job reliability
- Low latency job pickup

## Decision
We will use Redis as our primary job queue implementation.

## Rationale

### Why Redis?

**Performance:**
- In-memory operations provide sub-millisecond latency
- Supports 100k+ operations per second on modest hardware
- Blocking operations (BRPOPLPUSH) eliminate polling overhead

**Reliability:**
- Atomic operations prevent race conditions during job state transitions
- BRPOPLPUSH provides atomic move from pending to processing queue
- Configurable persistence (RDB + AOF) ensures job durability

**Operational Simplicity:**
- Single dependency vs. complex message brokers
- Well-understood operations and monitoring
- Excellent Go client libraries

**Feature Match:**
- List operations perfect for FIFO job queues
- Hash operations ideal for job metadata storage
- Pub/Sub capabilities for future real-time notifications

### Alternatives Considered

**PostgreSQL Queues:**
- Pros: Single storage system, ACID guarantees
- Cons: Higher latency, polling required, lock contention at scale

**RabbitMQ:**
- Pros: Dedicated message broker, advanced routing
- Cons: Additional operational complexity, heavier resource usage

**Apache Kafka:**
- Pros: High throughput, built for streaming
- Cons: Overkill for job queuing, complex operations

**AWS SQS:**
- Pros: Managed service, infinite scale
- Cons: Vendor lock-in, higher latency, eventually consistent

## Implementation Details

### Queue Structure
```
taskflow:jobs:pending     -> List of job IDs (FIFO)
taskflow:jobs:processing  -> List of job IDs currently being processed
taskflow:job:{id}         -> Hash containing job details
```

### Key Operations
- **Enqueue**: LPUSH to pending queue + SET job data
- **Dequeue**: BRPOPLPUSH pending to processing + GET job data
- **Complete**: LREM from processing + UPDATE job data
- **Fail**: LREM from processing + LPUSH to pending (retry) or mark failed

### Atomicity Guarantees
- Job transitions use Redis pipelines for atomic multi-operation updates
- Worker failure recovery handled by processing queue cleanup
- Exponential backoff for failed jobs

## Consequences

### Positive
- Very high performance and low latency
- Simple operational model
- Built-in clustering support for horizontal scaling
- Rich ecosystem of monitoring tools

### Negative
- Additional dependency beyond PostgreSQL
- Memory usage grows with queue depth
- Requires Redis-specific expertise for advanced tuning

### Risks and Mitigations
- **Memory exhaustion**: Implement job TTL and queue depth monitoring
- **Single point of failure**: Use Redis Sentinel or Cluster mode
- **Data loss**: Enable persistence with appropriate sync policies

## Monitoring and Observability
- Queue depth metrics via Redis INFO
- Job processing rates via custom metrics
- Memory usage and connection pool monitoring
- Alert on queue depth growth and processing delays

## Future Considerations
- Evaluate Redis Streams for more advanced job routing
- Consider Redis Cluster for horizontal scaling beyond single instance
- Potential migration to cloud-managed Redis for operational simplicity

## References
- [Redis Lists Documentation](https://redis.io/docs/data-types/lists/)
- [Redis Persistence Guide](https://redis.io/docs/manual/persistence/)
- [BRPOPLPUSH Command](https://redis.io/commands/brpoplpush/)