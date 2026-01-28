package producer

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"kafka-message-generator/internal/config"
)

// KafkaProducer wraps a Sarama async producer
type KafkaProducer struct {
	producer sarama.AsyncProducer
	topic    string
	errors   chan error
	successes chan *sarama.ProducerMessage
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer(cfg *config.Config) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	
	// Set producer configuration
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(cfg.Kafka.Producer.RequiredAcks)
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	
	// Set compression
	switch cfg.Kafka.Producer.Compression {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}
	
	saramaConfig.Producer.MaxMessageBytes = cfg.Kafka.Producer.MaxMessageBytes
	
	// Set flush settings
	flushFreq, err := cfg.Kafka.Producer.GetFlushFrequency()
	if err != nil {
		return nil, fmt.Errorf("invalid flush frequency: %w", err)
	}
	saramaConfig.Producer.Flush.Frequency = flushFreq
	saramaConfig.Producer.Flush.Messages = cfg.Kafka.Producer.FlushMessages
	
	// Set retry settings
	retryBackoff, err := cfg.Kafka.Producer.GetRetryBackoff()
	if err != nil {
		return nil, fmt.Errorf("invalid retry backoff: %w", err)
	}
	saramaConfig.Producer.Retry.Max = cfg.Kafka.Producer.RetryMax
	saramaConfig.Producer.Retry.Backoff = retryBackoff
	
	// Set idempotent
	saramaConfig.Producer.Idempotent = cfg.Kafka.Producer.Idempotent
	if cfg.Kafka.Producer.Idempotent {
		saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
		saramaConfig.Producer.Retry.Max = 5
		saramaConfig.Net.MaxOpenRequests = 1
	}
	
	// Set connection timeout
	timeout, err := cfg.Kafka.Connection.GetTimeout()
	if err != nil {
		return nil, fmt.Errorf("invalid connection timeout: %w", err)
	}
	saramaConfig.Net.DialTimeout = timeout
	saramaConfig.Net.ReadTimeout = timeout
	saramaConfig.Net.WriteTimeout = timeout
	
	// Set keep alive
	keepAlive, err := cfg.Kafka.Connection.GetKeepAlive()
	if err != nil {
		return nil, fmt.Errorf("invalid keep alive: %w", err)
	}
	saramaConfig.Net.KeepAlive = keepAlive
	
	// Set TLS configuration
	if cfg.Kafka.Connection.EnableTLS {
		saramaConfig.Net.TLS.Enable = true
		if cfg.Kafka.Connection.InsecureSkipVerify {
			saramaConfig.Net.TLS.Config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}
	
	// Set version
	saramaConfig.Version = sarama.V2_6_0_0
	
	// Create producer
	producer, err := sarama.NewAsyncProducer(cfg.Kafka.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	
	kp := &KafkaProducer{
		producer:  producer,
		topic:     cfg.Kafka.Topic,
		errors:    make(chan error, 100),
		successes: make(chan *sarama.ProducerMessage, 100),
	}
	
	// Start error and success handlers
	go kp.handleErrors()
	go kp.handleSuccesses()
	
	return kp, nil
}

// SendMessage sends a message to Kafka
func (kp *KafkaProducer) SendMessage(key string, value []byte) {
	msg := &sarama.ProducerMessage{
		Topic:     kp.topic,
		Value:     sarama.ByteEncoder(value),
		Timestamp: time.Now(),
	}

	if key != "" {
		msg.Key = sarama.StringEncoder(key)
	}

	kp.producer.Input() <- msg
}

// handleErrors handles producer errors
func (kp *KafkaProducer) handleErrors() {
	for err := range kp.producer.Errors() {
		select {
		case kp.errors <- err.Err:
		default:
			// Error channel full, drop error
		}
	}
}

// handleSuccesses handles producer successes
func (kp *KafkaProducer) handleSuccesses() {
	for msg := range kp.producer.Successes() {
		select {
		case kp.successes <- msg:
		default:
			// Success channel full, drop success
		}
	}
}

// Errors returns the error channel
func (kp *KafkaProducer) Errors() <-chan error {
	return kp.errors
}

// Successes returns the success channel
func (kp *KafkaProducer) Successes() <-chan *sarama.ProducerMessage {
	return kp.successes
}

// Close closes the producer
func (kp *KafkaProducer) Close() error {
	return kp.producer.Close()
}
