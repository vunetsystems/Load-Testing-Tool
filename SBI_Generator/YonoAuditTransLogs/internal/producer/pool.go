package producer

import (
	"fmt"
	"sync"
	"sync/atomic"

	"kafka-message-generator/internal/config"
	"kafka-message-generator/internal/metrics"
)

// ProducerPool manages a pool of Kafka producers
type ProducerPool struct {
	producers  []*KafkaProducer
	metricsCol *metrics.Collector
	current    uint64
	wg         sync.WaitGroup
}

// NewProducerPool creates a new producer pool
func NewProducerPool(cfg *config.Config, metricsCol *metrics.Collector) (*ProducerPool, error) {
	pool := &ProducerPool{
		producers:  make([]*KafkaProducer, cfg.Kafka.Producer.NumProducers),
		metricsCol: metricsCol,
	}

	// Create producers
	for i := 0; i < cfg.Kafka.Producer.NumProducers; i++ {
		producer, err := NewKafkaProducer(cfg)
		if err != nil {
			// Close any already created producers
			pool.Close()
			return nil, fmt.Errorf("failed to create producer %d: %w", i, err)
		}
		pool.producers[i] = producer
	}

	// Start error/success aggregators
	pool.wg.Add(len(pool.producers) * 2)
	for _, producer := range pool.producers {
		go pool.aggregateErrors(producer)
		go pool.aggregateSuccesses(producer)
	}

	return pool, nil
}

// GetProducer returns a producer using round-robin
func (pp *ProducerPool) GetProducer() *KafkaProducer {
	idx := atomic.AddUint64(&pp.current, 1) % uint64(len(pp.producers))
	return pp.producers[idx]
}

// SendMessage sends a message using round-robin producer selection
func (pp *ProducerPool) SendMessage(key string, value []byte) {
	producer := pp.GetProducer()
	producer.SendMessage(key, value)
}

// aggregateErrors aggregates errors from a producer
func (pp *ProducerPool) aggregateErrors(producer *KafkaProducer) {
	defer pp.wg.Done()
	for range producer.Errors() {
		pp.metricsCol.IncrementFailed()
	}
}

// aggregateSuccesses aggregates successes from a producer
func (pp *ProducerPool) aggregateSuccesses(producer *KafkaProducer) {
	defer pp.wg.Done()
	for range producer.Successes() {
		pp.metricsCol.IncrementSent()
	}
}

// Close closes all producers in the pool
func (pp *ProducerPool) Close() error {
	var firstErr error
	for _, producer := range pp.producers {
		if producer != nil {
			if err := producer.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	pp.wg.Wait()
	return firstErr
}
