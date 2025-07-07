#!/bin/bash

# TaskFlow Job Testing Script
# This script creates sample jobs to test the TaskFlow system

API_URL="${API_URL:-http://localhost:8080/api/v1}"

echo "Testing TaskFlow Jobs"
echo "API URL: $API_URL"
echo

# Function to create a job and track it
create_and_track_job() {
    local job_data="$1"
    local job_name="$2"
    
    echo "Creating $job_name job..."
    
    response=$(curl -s -X POST "$API_URL/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_data")
    
    job_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$job_id" ]; then
        echo "Failed to create job"
        echo "Response: $response"
        return 1
    fi
    
    echo "Created job: $job_id"
    
    # Track job progress
    echo "Tracking job progress..."
    for i in {1..10}; do
        status_response=$(curl -s "$API_URL/jobs/$job_id")
        status=$(echo "$status_response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        
        echo "   Status: $status"
        
        if [ "$status" = "completed" ] || [ "$status" = "failed" ]; then
            break
        fi
        
        sleep 2
    done
    
    echo "📋 Final job details:"
    curl -s "$API_URL/jobs/$job_id" | python3 -m json.tool || echo "$status_response"
    echo
}

# Test 1: Email Job
echo "=== Test 1: Email Job ==="
email_job='{
    "type": "email",
    "payload": {
        "to": "test@example.com",
        "subject": "TaskFlow Test Email",
        "body": "This is a test email from TaskFlow distributed task queue system."
    }
}'
create_and_track_job "$email_job" "Email"

# Test 2: Image Resize Job
echo "=== Test 2: Image Resize Job ==="
image_job='{
    "type": "image_resize",
    "payload": {
        "image_url": "https://picsum.photos/1920/1080",
        "sizes": [100, 300, 500],
        "format": "webp",
        "output_path": "/data/exports/test_image"
    }
}'
create_and_track_job "$image_job" "Image Resize"

# Test 3: Webhook Job
echo "=== Test 3: Webhook Job ==="
webhook_job='{
    "type": "webhook",
    "payload": {
        "url": "https://httpbin.org/post",
        "method": "POST",
        "data": {
            "event": "test_webhook",
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
            "source": "taskflow"
        }
    }
}'
create_and_track_job "$webhook_job" "Webhook"

# Test 4: Data Export Job
echo "=== Test 4: Data Export Job ==="
export_job='{
    "type": "data_export",
    "payload": {
        "export_type": "csv",
        "query": "SELECT * FROM users WHERE created_at > now() - interval '\''1 week'\''",
        "output_path": "/data/exports/test_export"
    }
}'
create_and_track_job "$export_job" "Data Export"

# Get overall statistics
echo "=== System Statistics ==="
curl -s "$API_URL/stats" | python3 -m json.tool
echo

# Get worker information
echo "=== Active Workers ==="
curl -s "$API_URL/workers" | python3 -m json.tool
echo

# List recent jobs
echo "=== Recent Jobs ==="
curl -s "$API_URL/jobs?page=1&page_size=5" | python3 -m json.tool

echo "🎉 Job testing complete!"