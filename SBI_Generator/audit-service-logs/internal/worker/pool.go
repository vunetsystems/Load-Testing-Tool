package worker

import (
	"context"
	"log"
	"sync"

	"audit-service-logs-generator/internal/config"
	"audit-service-logs-generator/internal/generator"
	"audit-service-logs-generator/internal/metrics"
	"audit-service-logs-generator/internal/producer"
)

type WorkerPool struct {
	config       *config.Config
	generator    *generator.MessageGenerator
	producer     *producer.KafkaProducer
	rateLimiter  *RateLimiter
	metrics      *metrics.Collector
	wg           sync.WaitGroup
}

func NewWorkerPool(cfg *config.Config, gen *generator.MessageGenerator, prod *producer.KafkaProducer, met *metrics.Collector) *WorkerPool {
	return &WorkerPool{
		config:      cfg,
		generator:   gen,
		producer:    prod,
		rateLimiter: NewRateLimiter(cfg.Execution.TargetEPS),
		metrics:     met,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	log.Printf("Starting %d workers...", wp.config.Execution.Workers)
	
	for i := 0; i < wp.config.Execution.Workers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx)
	}
}

func (wp *WorkerPool) worker(ctx context.Context) {
	defer wp.wg.Done()

	for {
		for serviceId, svc := range wp.config.Services {
			// Wait for rate limiter permission for EACH message
			if err := wp.rateLimiter.Wait(ctx); err != nil {
				return // Context cancelled
			}

			wp.metrics.IncrementGenerated()

			msgBytes, err := wp.generator.GenerateMessage(serviceId, svc.ApiUrl)
			if err != nil {
				log.Printf("ERROR: Failed to generate message: %v", err)
				wp.metrics.IncrementFailed()
				continue
			}

			wp.producer.SendMessage(msgBytes)
			wp.metrics.IncrementSent()
		}
	}
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
