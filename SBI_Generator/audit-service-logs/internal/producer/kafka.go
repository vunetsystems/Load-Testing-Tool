package producer

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"audit-service-logs-generator/internal/config"
)

type KafkaProducer struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewKafkaProducer(cfg *config.Config, broker string) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_6_0_0
	
	// Optimized settings for throughput
	saramaConfig.Producer.Return.Successes = false
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.Flush.Frequency = 10 * time.Millisecond
	saramaConfig.Producer.Flush.Messages = 100

	saramaConfig.Net.TLS.Enable = true
	saramaConfig.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	producer, err := sarama.NewAsyncProducer([]string{broker}, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	kp := &KafkaProducer{
		producer: producer,
		topic:    cfg.Kafka.Topic,
	}

	// Start error handler
	go kp.handleErrors()

	return kp, nil
}

func (kp *KafkaProducer) SendMessage(value []byte) {
	msg := &sarama.ProducerMessage{
		Topic: kp.topic,
		Value: sarama.ByteEncoder(value),
	}
	kp.producer.Input() <- msg
}

func (kp *KafkaProducer) handleErrors() {
	for err := range kp.producer.Errors() {
		log.Printf("ERROR: Kafka producer error: %v", err)
	}
}

func (kp *KafkaProducer) Close() error {
	return kp.producer.Close()
}
