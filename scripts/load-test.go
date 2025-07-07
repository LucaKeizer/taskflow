package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestConfig struct {
	APIUrl     string
	JobCount   int
	Concurrent int
	Duration   time.Duration
	JobTypes   []string
}

type TestResult struct {
	TotalRequests   int64
	SuccessfulJobs  int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	AvgResponseTime time.Duration
	RequestsPerSec  float64
}

type JobRequest struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func main() {
	var (
		apiUrl     = flag.String("url", "http://localhost:8080", "API base URL")
		jobCount   = flag.Int("jobs", 1000, "Number of jobs to create")
		concurrent = flag.Int("concurrent", 50, "Number of concurrent requests")
		duration   = flag.Duration("duration", 0, "Test duration (0 = count-based)")
	)
	flag.Parse()

	config := LoadTestConfig{
		APIUrl:     *apiUrl,
		JobCount:   *jobCount,
		Concurrent: *concurrent,
		Duration:   *duration,
		JobTypes:   []string{"email", "webhook", "image_resize", "data_export"},
	}

	fmt.Printf("🚀 Starting TaskFlow Load Test\n")
	fmt.Printf("API URL: %s\n", config.APIUrl)
	fmt.Printf("Jobs: %d\n", config.JobCount)
	fmt.Printf("Concurrent: %d\n", config.Concurrent)
	if config.Duration > 0 {
		fmt.Printf("Duration: %v\n", config.Duration)
	}
	fmt.Printf("Job Types: %v\n", config.JobTypes)
	fmt.Println()

	// Test API connectivity
	if !testConnectivity(config.APIUrl) {
		fmt.Printf("❌ Cannot connect to API at %s\n", config.APIUrl)
		return
	}

	// Run the load test
	result := runLoadTest(config)

	// Print results
	printResults(result)
}

func testConnectivity(apiUrl string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiUrl + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func runLoadTest(config LoadTestConfig) TestResult {
	var (
		totalRequests   int64
		successfulJobs  int64
		failedRequests  int64
		totalDuration   int64
		minResponseTime int64 = 999999999999 // Large initial value
		maxResponseTime int64
	)

	// Create semaphore to limit concurrency
	semaphore := make(chan struct{}, config.Concurrent)
	var wg sync.WaitGroup

	// Start time
	startTime := time.Now()

	// Duration-based or count-based test
	if config.Duration > 0 {
		// Duration-based test
		endTime := startTime.Add(config.Duration)

		for time.Now().Before(endTime) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				responseTime := makeJobRequest(config.APIUrl, config.JobTypes)
				atomic.AddInt64(&totalRequests, 1)

				if responseTime > 0 {
					atomic.AddInt64(&successfulJobs, 1)
					atomic.AddInt64(&totalDuration, responseTime.Nanoseconds())

					// Update min/max response times
					for {
						current := atomic.LoadInt64(&minResponseTime)
						if responseTime.Nanoseconds() >= current {
							break
						}
						if atomic.CompareAndSwapInt64(&minResponseTime, current, responseTime.Nanoseconds()) {
							break
						}
					}

					for {
						current := atomic.LoadInt64(&maxResponseTime)
						if responseTime.Nanoseconds() <= current {
							break
						}
						if atomic.CompareAndSwapInt64(&maxResponseTime, current, responseTime.Nanoseconds()) {
							break
						}
					}
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}
			}()
		}
	} else {
		// Count-based test
		for i := 0; i < config.JobCount; i++ {
			wg.Add(1)
			go func(jobNum int) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				responseTime := makeJobRequest(config.APIUrl, config.JobTypes)
				atomic.AddInt64(&totalRequests, 1)

				if responseTime > 0 {
					atomic.AddInt64(&successfulJobs, 1)
					atomic.AddInt64(&totalDuration, responseTime.Nanoseconds())

					// Update min/max response times atomically
					for {
						current := atomic.LoadInt64(&minResponseTime)
						if responseTime.Nanoseconds() >= current {
							break
						}
						if atomic.CompareAndSwapInt64(&minResponseTime, current, responseTime.Nanoseconds()) {
							break
						}
					}

					for {
						current := atomic.LoadInt64(&maxResponseTime)
						if responseTime.Nanoseconds() <= current {
							break
						}
						if atomic.CompareAndSwapInt64(&maxResponseTime, current, responseTime.Nanoseconds()) {
							break
						}
					}
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}

				// Progress indicator
				if jobNum%100 == 0 {
					fmt.Printf("Progress: %d/%d jobs submitted\n", jobNum, config.JobCount)
				}
			}(i)
		}
	}

	// Wait for all requests to complete
	wg.Wait()

	actualDuration := time.Since(startTime)

	// Calculate average response time
	var avgResponseTime time.Duration
	if successfulJobs > 0 {
		avgResponseTime = time.Duration(totalDuration / successfulJobs)
	}

	return TestResult{
		TotalRequests:   totalRequests,
		SuccessfulJobs:  successfulJobs,
		FailedRequests:  failedRequests,
		TotalDuration:   actualDuration,
		MinResponseTime: time.Duration(minResponseTime),
		MaxResponseTime: time.Duration(maxResponseTime),
		AvgResponseTime: avgResponseTime,
		RequestsPerSec:  float64(totalRequests) / actualDuration.Seconds(),
	}
}

func makeJobRequest(apiUrl string, jobTypes []string) time.Duration {
	// Randomly select a job type
	jobType := jobTypes[rand.Intn(len(jobTypes))]

	// Create job payload based on type
	var payload interface{}
	switch jobType {
	case "email":
		payload = map[string]interface{}{
			"to":      fmt.Sprintf("loadtest+%d@example.com", rand.Intn(10000)),
			"subject": "Load Test Email",
			"body":    "This is a load test email from TaskFlow",
		}
	case "webhook":
		payload = map[string]interface{}{
			"url":    "https://httpbin.org/post",
			"method": "POST",
			"data": map[string]interface{}{
				"test_id":   rand.Intn(10000),
				"timestamp": time.Now().Unix(),
			},
		}
	case "image_resize":
		sizes := []int{100, 300, 500}
		payload = map[string]interface{}{
			"image_url":   "https://picsum.photos/1920/1080",
			"sizes":       sizes[:rand.Intn(len(sizes))+1],
			"format":      "webp",
			"output_path": fmt.Sprintf("/tmp/loadtest_%d", rand.Intn(10000)),
		}
	case "data_export":
		payload = map[string]interface{}{
			"export_type": "csv",
			"query":       "SELECT * FROM users WHERE id > " + fmt.Sprintf("%d", rand.Intn(1000)),
			"output_path": fmt.Sprintf("/tmp/export_%d", rand.Intn(10000)),
		}
	default:
		payload = map[string]interface{}{
			"test": "data",
		}
	}

	jobRequest := JobRequest{
		Type:    jobType,
		Payload: payload,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(jobRequest)
	if err != nil {
		return 0
	}

	// Make HTTP request
	client := &http.Client{Timeout: 10 * time.Second}

	start := time.Now()
	resp, err := client.Post(
		apiUrl+"/api/v1/jobs",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	responseTime := time.Since(start)

	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return 0
	}

	return responseTime
}

func printResults(result TestResult) {
	fmt.Println()
	fmt.Println("📊 Load Test Results")
	fmt.Printf("Total Requests:    %d\n", result.TotalRequests)
	fmt.Printf("Successful Jobs:   %d\n", result.SuccessfulJobs)
	fmt.Printf("Failed Requests:   %d\n", result.FailedRequests)
	fmt.Printf("Success Rate:      %.2f%%\n", float64(result.SuccessfulJobs)/float64(result.TotalRequests)*100)
	fmt.Println()
	fmt.Printf("Total Duration:    %v\n", result.TotalDuration)
	fmt.Printf("Requests/Second:   %.2f\n", result.RequestsPerSec)
	fmt.Println()
	fmt.Printf("Response Times:\n")
	fmt.Printf("  Min:             %v\n", result.MinResponseTime)
	fmt.Printf("  Max:             %v\n", result.MaxResponseTime)
	fmt.Printf("  Average:         %v\n", result.AvgResponseTime)
	fmt.Println()

	// Performance assessment
	if result.RequestsPerSec > 100 {
		fmt.Println("🎉 Excellent performance!")
	} else if result.RequestsPerSec > 50 {
		fmt.Println("✅ Good performance")
	} else if result.RequestsPerSec > 20 {
		fmt.Println("⚠️  Moderate performance")
	} else {
		fmt.Println("❌ Poor performance")
	}

	successRate := float64(result.SuccessfulJobs) / float64(result.TotalRequests) * 100
	if successRate > 99 {
		fmt.Println("🎯 Excellent reliability!")
	} else if successRate > 95 {
		fmt.Println("✅ Good reliability")
	} else if successRate > 90 {
		fmt.Println("⚠️  Moderate reliability")
	} else {
		fmt.Println("❌ Poor reliability")
	}
}
