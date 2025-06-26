package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"flugo.com/logger"
)

type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Attempts  int                    `json:"attempts"`
	MaxRetry  int                    `json:"max_retry"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Status    JobStatus              `json:"status"`
	Error     string                 `json:"error,omitempty"`
}

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
)

type JobHandler func(job *Job) error

type Queue struct {
	name     string
	jobs     chan *Job
	handlers map[string]JobHandler
	workers  int
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	stats    *QueueStats
}

type QueueStats struct {
	Processed int64 `json:"processed"`
	Failed    int64 `json:"failed"`
	Retried   int64 `json:"retried"`
	Active    int64 `json:"active"`
}

var DefaultQueue *Queue

func Init(workers int) {
	DefaultQueue = NewQueue("default", workers)
	DefaultQueue.Start()
}

func NewQueue(name string, workers int) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	return &Queue{
		name:     name,
		jobs:     make(chan *Job, 1000),
		handlers: make(map[string]JobHandler),
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
		stats:    &QueueStats{},
	}
}

func (q *Queue) RegisterHandler(jobType string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

func (q *Queue) Start() {
	for i := 0; i < q.workers; i++ {
		go q.worker(i)
	}
	logger.Info("Queue '%s' started with %d workers", q.name, q.workers)
}

func (q *Queue) Stop() {
	q.cancel()
	close(q.jobs)
	logger.Info("Queue '%s' stopped", q.name)
}

func (q *Queue) worker(id int) {
	logger.Debug("Worker %d started", id)

	for {
		select {
		case job := <-q.jobs:
			if job == nil {
				logger.Debug("Worker %d stopped", id)
				return
			}
			q.processJob(job, id)

		case <-q.ctx.Done():
			logger.Debug("Worker %d stopped due to context cancellation", id)
			return
		}
	}
}

func (q *Queue) processJob(job *Job, workerID int) {
	q.mu.Lock()
	q.stats.Active++
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		q.stats.Active--
		q.mu.Unlock()
	}()

	logger.Debug("Worker %d processing job %s (type: %s)", workerID, job.ID, job.Type)

	job.Status = StatusProcessing
	job.UpdatedAt = time.Now()
	job.Attempts++

	q.mu.RLock()
	handler, exists := q.handlers[job.Type]
	q.mu.RUnlock()

	if !exists {
		job.Status = StatusFailed
		job.Error = fmt.Sprintf("no handler registered for job type: %s", job.Type)
		logger.Error("No handler for job type %s", job.Type)
		q.mu.Lock()
		q.stats.Failed++
		q.mu.Unlock()
		return
	}

	err := handler(job)
	if err != nil {
		job.Error = err.Error()

		if job.Attempts < job.MaxRetry {
			job.Status = StatusRetrying
			logger.Warn("Job %s failed, retrying (%d/%d): %v", job.ID, job.Attempts, job.MaxRetry, err)

			// Retry with exponential backoff
			delay := time.Duration(job.Attempts*job.Attempts) * time.Second
			time.Sleep(delay)

			select {
			case q.jobs <- job:
				q.mu.Lock()
				q.stats.Retried++
				q.mu.Unlock()
			default:
				logger.Error("Failed to requeue job %s: queue is full", job.ID)
				job.Status = StatusFailed
				q.mu.Lock()
				q.stats.Failed++
				q.mu.Unlock()
			}
		} else {
			job.Status = StatusFailed
			logger.Error("Job %s failed permanently after %d attempts: %v", job.ID, job.Attempts, err)
			q.mu.Lock()
			q.stats.Failed++
			q.mu.Unlock()
		}
	} else {
		job.Status = StatusCompleted
		job.UpdatedAt = time.Now()
		logger.Info("Job %s completed successfully", job.ID)
		q.mu.Lock()
		q.stats.Processed++
		q.mu.Unlock()
	}
}

func (q *Queue) Push(jobType string, payload map[string]interface{}, maxRetry int) error {
	job := &Job{
		ID:        generateJobID(),
		Type:      jobType,
		Payload:   payload,
		MaxRetry:  maxRetry,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	select {
	case q.jobs <- job:
		logger.Debug("Job %s queued (type: %s)", job.ID, job.Type)
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

func (q *Queue) PushDelay(jobType string, payload map[string]interface{}, maxRetry int, delay time.Duration) error {
	go func() {
		time.Sleep(delay)
		q.Push(jobType, payload, maxRetry)
	}()

	logger.Debug("Delayed job %s scheduled (type: %s, delay: %v)", generateJobID(), jobType, delay)
	return nil
}

func (q *Queue) GetStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return &QueueStats{
		Processed: q.stats.Processed,
		Failed:    q.stats.Failed,
		Retried:   q.stats.Retried,
		Active:    q.stats.Active,
	}
}

func (q *Queue) Size() int {
	return len(q.jobs)
}

func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// Helper functions
func RegisterHandler(jobType string, handler JobHandler) {
	if DefaultQueue != nil {
		DefaultQueue.RegisterHandler(jobType, handler)
	}
}

func Push(jobType string, payload map[string]interface{}) error {
	return PushWithRetry(jobType, payload, 3)
}

func PushWithRetry(jobType string, payload map[string]interface{}, maxRetry int) error {
	if DefaultQueue == nil {
		return fmt.Errorf("queue not initialized")
	}
	return DefaultQueue.Push(jobType, payload, maxRetry)
}

func PushDelay(jobType string, payload map[string]interface{}, delay time.Duration) error {
	if DefaultQueue == nil {
		return fmt.Errorf("queue not initialized")
	}
	return DefaultQueue.PushDelay(jobType, payload, 3, delay)
}

func GetStats() *QueueStats {
	if DefaultQueue == nil {
		return &QueueStats{}
	}
	return DefaultQueue.GetStats()
}

// Built-in job handlers
func init() {
	RegisterHandler("send_email", func(job *Job) error {
		to, _ := job.Payload["to"].(string)
		subject, _ := job.Payload["subject"].(string)
		_, _ = job.Payload["body"].(string)

		if to == "" || subject == "" {
			return fmt.Errorf("missing required email parameters")
		}

		logger.Info("Sending email to %s: %s", to, subject)
		time.Sleep(100 * time.Millisecond) // Simulate email sending

		return nil
	})

	RegisterHandler("image_process", func(job *Job) error {
		imagePath, _ := job.Payload["image_path"].(string)
		operation, _ := job.Payload["operation"].(string)

		if imagePath == "" {
			return fmt.Errorf("image_path is required")
		}

		logger.Info("Processing image %s with operation %s", imagePath, operation)
		time.Sleep(500 * time.Millisecond) // Simulate image processing

		return nil
	})

	RegisterHandler("data_export", func(job *Job) error {
		format, _ := job.Payload["format"].(string)
		userID, _ := job.Payload["user_id"].(float64)

		logger.Info("Exporting data for user %d in format %s", int(userID), format)
		time.Sleep(2 * time.Second) // Simulate data export

		return nil
	})

	RegisterHandler("webhook_call", func(job *Job) error {
		url, _ := job.Payload["url"].(string)
		data, _ := job.Payload["data"].(map[string]interface{})

		if url == "" {
			return fmt.Errorf("webhook URL is required")
		}

		dataBytes, _ := json.Marshal(data)
		logger.Info("Calling webhook %s with data: %s", url, string(dataBytes))
		time.Sleep(200 * time.Millisecond) // Simulate webhook call

		return nil
	})

	RegisterHandler("notification", func(job *Job) error {
		userID, _ := job.Payload["user_id"].(float64)
		message, _ := job.Payload["message"].(string)
		channel, _ := job.Payload["channel"].(string)

		logger.Info("Sending %s notification to user %d: %s", channel, int(userID), message)
		time.Sleep(100 * time.Millisecond) // Simulate notification sending

		return nil
	})
}

func SendEmailAsync(to, subject, body string) error {
	return Push("send_email", map[string]interface{}{
		"to":      to,
		"subject": subject,
		"body":    body,
	})
}

func ProcessImageAsync(imagePath, operation string) error {
	return Push("image_process", map[string]interface{}{
		"image_path": imagePath,
		"operation":  operation,
	})
}

func ExportDataAsync(userID int, format string) error {
	return Push("data_export", map[string]interface{}{
		"user_id": userID,
		"format":  format,
	})
}

func CallWebhookAsync(url string, data map[string]interface{}) error {
	return Push("webhook_call", map[string]interface{}{
		"url":  url,
		"data": data,
	})
}

func SendNotificationAsync(userID int, message, channel string) error {
	return Push("notification", map[string]interface{}{
		"user_id": userID,
		"message": message,
		"channel": channel,
	})
}
