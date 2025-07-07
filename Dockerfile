# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build applications
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/taskflow-api cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/taskflow-worker cmd/worker/main.go

# API service stage
FROM alpine:latest AS api

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/taskflow-api .

# Create directory for exports
RUN mkdir -p /data/exports

EXPOSE 8080

CMD ["./taskflow-api"]

# Worker service stage  
FROM alpine:latest AS worker

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/taskflow-worker .

# Create directory for exports
RUN mkdir -p /data/exports

CMD ["./taskflow-worker"]