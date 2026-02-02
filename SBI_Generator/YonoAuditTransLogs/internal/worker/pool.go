package worker

import (
	"context"
	"sync"
	"time"

	"kafka-message-generator/internal/config"
	"kafka-message-generator/internal/generator"
	"kafka-message-generator/internal/metrics"
	"kafka-message-generator/internal/producer"
)

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	config        *config.Config
	msgGenerator  *generator.MessageGenerator
	producerPool  *producer.ProducerPool
	rateLimiter   *RateLimiter
	metricsCol    *metrics.Collector
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	cfg *config.Config,
	msgGen *generator.MessageGenerator,
	prodPool *producer.ProducerPool,
	metricsCol *metrics.Collector,
) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		config:       cfg,
		msgGenerator: msgGen,
		producerPool: prodPool,
		rateLimiter:  NewRateLimiter(cfg.Execution.EPS, cfg.Concurrency.RateLimiterBurst),
		metricsCol:   metricsCol,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	// Determine total messages to generate
	var totalMessages int64
	var duration time.Duration
	
	if wp.config.Execution.Mode == "count" {
		totalMessages = int64(wp.config.Execution.Count)
	} else {
		dur, _ := wp.config.Execution.GetDuration()
		duration = dur
	}
	
	// Start workers
	for i := 0; i < wp.config.Concurrency.WorkerPoolSize; i++ {
		wp.wg.Add(1)
		go wp.worker(totalMessages, duration)
	}
}

// worker is the worker goroutine
func (wp *WorkerPool) worker(totalMessages int64, duration time.Duration) {
	defer wp.wg.Done()
	
	var deadline time.Time
	if duration > 0 {
		deadline = time.Now().Add(duration)
	}
	
	messageCount := int64(0)
	
	for {
		// Check if we should stop
		select {
		case <-wp.ctx.Done():
			return
		default:
		}
		
		// Check count-based termination (only in count mode)
		if wp.config.Execution.Mode == "count" && totalMessages > 0 {
			if messageCount >= totalMessages/int64(wp.config.Concurrency.WorkerPoolSize) {
				return
			}
		}
		
		// Check duration-based termination (only in duration mode)
		if wp.config.Execution.Mode == "duration" && duration > 0 && time.Now().After(deadline) {
			return
		}
		
		// Wait for rate limiter
		if err := wp.rateLimiter.Wait(wp.ctx); err != nil {
			return
		}
		
		// Generate message
		startTime := time.Now()
		msg, err := wp.msgGenerator.GenerateMessage()
		if err != nil {
			wp.metricsCol.IncrementFailed()
			continue
		}
		
		// Convert to JSON
		msgBytes, err := msg.ToJSON()
		if err != nil {
			wp.metricsCol.IncrementFailed()
			continue
		}
		
		// Determine message type for metrics
		msgType := "both"
		if msg.YonoAdtError != nil && msg.YonoAdtTrans == nil {
			msgType = "error"
		} else if msg.YonoAdtError == nil && msg.YonoAdtTrans != nil {
			msgType = "trans"
		}
		
		// Extract transaction ID for key (from either error or trans) if enabled
		var key string
		if wp.config.Kafka.Producer.EnableKey {
			if msg.YonoAdtError != nil {
				// Parse to get trnsId - for simplicity, use timestamp
				key = string(msgBytes[:20]) // Use first 20 bytes as key
			} else if msg.YonoAdtTrans != nil {
				key = string(msgBytes[:20])
			}
		}
		
		// Send to Kafka
		wp.producerPool.SendMessage(key, msgBytes)
		
		// Update metrics
		wp.metricsCol.IncrementGenerated(msgType)
		wp.metricsCol.RecordLatency(time.Since(startTime))
		
		messageCount++
	}
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

// Wait waits for all workers to complete naturally
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// StopWithTimeout stops the worker pool with a timeout
func (wp *WorkerPool) StopWithTimeout(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wp.Stop()
		close(done)
	}()
	
	select {
	case <-done:
		// Stopped successfully
	case <-time.After(timeout):
		// Timeout - force stop
	}
}
