package generators

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ImageQueue manages image generation requests with queuing
type ImageQueue struct {
	requests chan *QueueRequest
	results  map[string]*QueueResult
	mu       sync.RWMutex
	workerCount int
	maxWorkers  int
}

// QueueRequest represents a queued image generation request
type QueueRequest struct {
	ID        string
	Options   *GenerateOptions
	ResultCh  chan *QueueResult
	CreatedAt time.Time
	Priority  int // Higher = higher priority
}

// QueueResult represents the result of a queued request
type QueueResult struct {
	ID        string
	ImageData []byte
	Error      error
	Duration   time.Duration
}

// NewImageQueue creates a new image generation queue
func NewImageQueue(maxWorkers int) *ImageQueue {
	return &ImageQueue{
		requests:   make(chan *QueueRequest, 100),
		results:    make(map[string]*QueueResult),
		workerCount: 0,
		maxWorkers:  maxWorkers,
	}
}

// Start starts the queue workers
func (q *ImageQueue) Start(ctx context.Context, comfyClient *ComfyUIClient) {
	// Start workers
	for i := 0; i < q.maxWorkers; i++ {
		go q.worker(ctx, comfyClient)
		q.workerCount++
	}

	// Start cleanup goroutine
	go q.cleanup(ctx)
}

// Stop stops the queue
func (q *ImageQueue) Stop() {
	close(q.requests)
}

// worker processes queued requests
func (q *ImageQueue) worker(ctx context.Context, comfyClient *ComfyUIClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-q.requests:
			if !ok {
				return
			}

			// Process request
			startTime := time.Now()
			imageData, err := comfyClient.GenerateImage(ctx, req.Options)
			duration := time.Since(startTime)

			result := &QueueResult{
				ID:        req.ID,
				ImageData: imageData.ImageData,
				Error:      err,
				Duration:   duration,
			}

			// Store result
			q.mu.Lock()
			q.results[req.ID] = result
			q.mu.Unlock()

			// Send to result channel
			select {
			case req.ResultCh <- result:
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
				// Timeout sending result
			}
		}
	}
}

// cleanup removes old results from the queue
func (q *ImageQueue) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.mu.Lock()
			now := time.Now()
			for id, result := range q.results {
				// Remove results older than 10 minutes
				if result.Error == nil && now.Sub(time.Time{}) > 10*time.Minute {
					delete(q.results, id)
				}
			}
			q.mu.Unlock()
		}
	}
}

// Enqueue adds a request to the queue
func (q *ImageQueue) Enqueue(req *QueueRequest) error {
	select {
	case q.requests <- req:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// GetResult retrieves a result by ID
func (q *ImageQueue) GetResult(id string) (*QueueResult, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result, ok := q.results[id]
	return result, ok
}

// GetQueueSize returns the current queue size
func (q *ImageQueue) GetQueueSize() int {
	return len(q.requests)
}

// GetWorkerCount returns the number of active workers
func (q *ImageQueue) GetWorkerCount() int {
	return q.workerCount
}

// EnqueueWithWait enqueues a request and waits for the result
func (q *ImageQueue) EnqueueWithWait(ctx context.Context, req *QueueRequest) (*QueueResult, error) {
	resultCh := make(chan *QueueResult, 1)
	req.ResultCh = resultCh

	if err := q.Enqueue(req); err != nil {
		return nil, err
	}

	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("request timeout")
	}
}
