package config

import (
	"fmt"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate message type
	if c.MessageType != "" && c.MessageType != "json_yono" && c.MessageType != "access_log" && c.MessageType != "joint" {
		return fmt.Errorf("message_type must be 'json_yono', 'access_log' or 'joint', got '%s'", c.MessageType)
	}

	// Validate distribution percentages (only relevant for json_yono or joint)
	if c.MessageType == "json_yono" || c.MessageType == "joint" || c.MessageType == "" {
		total := c.Distribution.ErrorOnlyPercent + c.Distribution.TransOnlyPercent + c.Distribution.BothPercent
		if total != 100 {
			return fmt.Errorf("distribution percentages must sum to 100, got %d", total)
		}
	}

	// Validate execution mode
	if c.Execution.Mode != "duration" && c.Execution.Mode != "count" {
		return fmt.Errorf("execution mode must be 'duration' or 'count', got '%s'", c.Execution.Mode)
	}

	// Validate EPS
	if c.Execution.EPS <= 0 {
		return fmt.Errorf("EPS must be positive, got %d", c.Execution.EPS)
	}

	// Validate duration if mode is duration
	if c.Execution.Mode == "duration" {
		if _, err := c.Execution.GetDuration(); err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
	}

	// Validate count if mode is count
	if c.Execution.Mode == "count" && c.Execution.Count <= 0 {
		return fmt.Errorf("count must be positive when mode is 'count', got %d", c.Execution.Count)
	}

	// Validate session rotation interval
	if _, err := c.Session.GetRotationInterval(); err != nil {
		return fmt.Errorf("invalid session rotation interval: %w", err)
	}

	// Validate Kafka brokers
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker must be specified")
	}

	// Validate Kafka topic
	if c.Kafka.Topic == "" {
		return fmt.Errorf("Kafka topic must be specified")
	}

	// Validate compression codec
	validCompressions := map[string]bool{
		"none":   true,
		"gzip":   true,
		"snappy": true,
		"lz4":    true,
		"zstd":   true,
	}
	if !validCompressions[c.Kafka.Producer.Compression] {
		return fmt.Errorf("invalid compression codec: %s", c.Kafka.Producer.Compression)
	}

	// Validate required acks
	if c.Kafka.Producer.RequiredAcks < -1 || c.Kafka.Producer.RequiredAcks > 1 {
		return fmt.Errorf("required_acks must be -1, 0, or 1, got %d", c.Kafka.Producer.RequiredAcks)
	}

	// Validate user ID mode
	if c.UserIDs.Mode != "fixed" && c.UserIDs.Mode != "range" {
		return fmt.Errorf("user_ids mode must be 'fixed' or 'range', got '%s'", c.UserIDs.Mode)
	}

	// Validate user IDs based on mode
	if c.UserIDs.Mode == "fixed" && len(c.UserIDs.FixedList) == 0 {
		return fmt.Errorf("fixed_list must contain at least one user ID when mode is 'fixed'")
	}

	if c.UserIDs.Mode == "range" {
		if c.UserIDs.RangeMin >= c.UserIDs.RangeMax {
			return fmt.Errorf("range_min must be less than range_max")
		}
	}

	// Validate trace ID format
	if c.IDGeneration.TraceIDFormat != "hex" && c.IDGeneration.TraceIDFormat != "uuid" {
		return fmt.Errorf("trace_id_format must be 'hex' or 'uuid', got '%s'", c.IDGeneration.TraceIDFormat)
	}

	// Validate templates
	if c.MessageType == "access_log" || c.MessageType == "joint" {
		if len(c.AccessLog.PodNames) == 0 {
			return fmt.Errorf("at least one pod name must be specified for access_log")
		}
		if len(c.AccessLog.APIUrls) == 0 {
			return fmt.Errorf("at least one API URL must be specified for access_log")
		}
	}

	if c.MessageType == "json_yono" || c.MessageType == "joint" || c.MessageType == "" {
		if len(c.Templates.Error.ErrorCodes) == 0 {
			return fmt.Errorf("at least one error code must be specified")
		}

		if len(c.Templates.Transaction.CommandIDs) == 0 {
			return fmt.Errorf("at least one command ID must be specified")
		}

		// Validate status weights
		if len(c.Templates.Transaction.Statuses) == 0 {
			return fmt.Errorf("at least one transaction status must be specified")
		}
	}

	if c.MessageType == "access_log" || c.MessageType == "joint" {
		totalAccessLogWeight := 0
		for _, status := range c.AccessLog.HttpStatuses {
			totalAccessLogWeight += status.Weight
		}
		if totalAccessLogWeight == 0 {
			return fmt.Errorf("total access log status weight must be greater than 0")
		}
	}

	if c.MessageType == "json_yono" || c.MessageType == "joint" || c.MessageType == "" {
		totalTransWeight := 0
		for _, status := range c.Templates.Transaction.Statuses {
			totalTransWeight += status.Weight
		}
		if totalTransWeight == 0 {
			return fmt.Errorf("total transaction status weight must be greater than 0")
		}
	}

	// Validate concurrency settings
	if c.Concurrency.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker_pool_size must be positive, got %d", c.Concurrency.WorkerPoolSize)
	}

	if c.Kafka.Producer.NumProducers <= 0 {
		return fmt.Errorf("num_producers must be positive, got %d", c.Kafka.Producer.NumProducers)
	}

	return nil
}
