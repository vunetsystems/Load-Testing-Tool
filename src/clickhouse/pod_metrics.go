package clickhouse

import (
	"context"
	"fmt"
	"os"
	"time"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// TimeRange represents a time window for metrics queries
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ClickHouseMetrics represents aggregated metrics from ClickHouse
type ClickHouseMetrics struct {
	KafkaProducerMetrics []KafkaProducerMetric `json:"kafkaProducerMetrics,omitempty"`
	KafkaTopicMetrics    []KafkaTopicMetric    `json:"kafkaTopicMetrics,omitempty"`
	SystemMetrics        []SystemMetric        `json:"systemMetrics,omitempty"`
	DatabaseMetrics      []DatabaseMetric      `json:"databaseMetrics,omitempty"`
	ContainerMetrics     []ContainerMetric     `json:"containerMetrics,omitempty"`
	LastUpdated          time.Time             `json:"lastUpdated"`
}

// KafkaProducerMetric represents Kafka producer metrics
type KafkaProducerMetric struct {
	Timestamp        time.Time `json:"timestamp"`
	ClientID         string    `json:"clientId"`
	Topic            string    `json:"topic"`
	RecordSendTotal  float64   `json:"recordSendTotal"`
	RecordSendRate   float64   `json:"recordSendRate"`
	ByteTotal        float64   `json:"byteTotal"`
	ByteRate         float64   `json:"byteRate"`
	RecordErrorTotal float64   `json:"recordErrorTotal"`
	RecordErrorRate  float64   `json:"recordErrorRate"`
	CompressionRate  float64   `json:"compressionRate"`
}

// SystemMetric represents system-level metrics
type SystemMetric struct {
	Timestamp   time.Time `json:"timestamp"`
	Host        string    `json:"host"`
	CPUUsage    float64   `json:"cpuUsage"`
	MemoryUsage float64   `json:"memoryUsage"`
	DiskUsage   float64   `json:"diskUsage"`
	NetworkRX   float64   `json:"networkRx"`
	NetworkTX   float64   `json:"networkTx"`
}

// DatabaseMetric represents database performance metrics
type DatabaseMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	Database      string    `json:"database"`
	Table         string    `json:"table"`
	QueryCount    int64     `json:"queryCount"`
	QueryDuration float64   `json:"queryDuration"`
	ErrorCount    int64     `json:"errorCount"`
}

// ContainerMetric represents container/Kubernetes metrics
type ContainerMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	Namespace     string    `json:"namespace"`
	PodName       string    `json:"podName"`
	ContainerName string    `json:"containerName"`
	CPUUsage      float64   `json:"cpuUsage"`
	MemoryUsage   float64   `json:"memoryUsage"`
	Status        string    `json:"status"`
}

// KafkaTopicMetric represents Kafka topic metrics (Messages In Per Sec by Topic)
type KafkaTopicMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	Topic         string    `json:"topic"`
	OneMinuteRate float64   `json:"oneMinuteRate"`
	Count         float64   `json:"count"`
	IsInput       bool      `json:"isInput"` // Whether this is an input topic (for frontend display)
}

// getKafkaProducerMetrics retrieves latest Kafka producer metrics
func (ch *ClickHouseClient) getKafkaProducerMetrics(ctx context.Context, limit int) ([]KafkaProducerMetric, error) {
	query := `
        SELECT
            timestamp,
            "client-id",
            topic,
            "record-send-total",
            "record-send-rate",
            "byte-total",
            "byte-rate",
            "record-error-total",
            "record-error-rate",
            "compression-rate"
        FROM kafka_producer_Producer_Topic_Metrics_data
        WHERE timestamp >= now() - INTERVAL 5 MINUTE
        ORDER BY timestamp DESC
        LIMIT ?
    `

	rows, err := ch.Client.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query Kafka producer metrics: %v", err)
	}
	defer rows.Close()

	var metrics []KafkaProducerMetric
	for rows.Next() {
		var metric KafkaProducerMetric
		err := rows.Scan(
			&metric.Timestamp,
			&metric.ClientID,
			&metric.Topic,
			&metric.RecordSendTotal,
			&metric.RecordSendRate,
			&metric.ByteTotal,
			&metric.ByteRate,
			&metric.RecordErrorTotal,
			&metric.RecordErrorRate,
			&metric.CompressionRate,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan Kafka metric row: %v", err))
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getSystemMetrics retrieves latest system metrics
func (ch *ClickHouseClient) getSystemMetrics(ctx context.Context, limit int) ([]SystemMetric, error) {
	query := `
        SELECT
            timestamp,
            host,
            usage_user as cpu_usage,
            usage_percent as memory_usage,
            usage_percent as disk_usage,
            rx_bytes as network_rx,
            tx_bytes as network_tx
        FROM system
        WHERE timestamp >= now() - INTERVAL 5 MINUTE
        ORDER BY timestamp DESC
        LIMIT ?
    `

	rows, err := ch.Client.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query system metrics: %v", err)
	}
	defer rows.Close()

	var metrics []SystemMetric
	for rows.Next() {
		var metric SystemMetric
		err := rows.Scan(
			&metric.Timestamp,
			&metric.Host,
			&metric.CPUUsage,
			&metric.MemoryUsage,
			&metric.DiskUsage,
			&metric.NetworkRX,
			&metric.NetworkTX,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan system metric row: %v", err))
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getDatabaseMetrics retrieves latest database metrics
func (ch *ClickHouseClient) getDatabaseMetrics(ctx context.Context, limit int) ([]DatabaseMetric, error) {
	query := `
        SELECT
            timestamp,
            database,
            table,
            query_count,
            query_duration_ms as query_duration,
            error_count
        FROM clickhouse_query_log
        WHERE timestamp >= now() - INTERVAL 5 MINUTE
            AND type = 'QueryFinish'
        ORDER BY timestamp DESC
        LIMIT ?
    `

	rows, err := ch.Client.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query database metrics: %v", err)
	}
	defer rows.Close()

	var metrics []DatabaseMetric
	for rows.Next() {
		var metric DatabaseMetric
		err := rows.Scan(
			&metric.Timestamp,
			&metric.Database,
			&metric.Table,
			&metric.QueryCount,
			&metric.QueryDuration,
			&metric.ErrorCount,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan database metric row: %v", err))
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getContainerMetrics retrieves latest container metrics
func (ch *ClickHouseClient) getContainerMetrics(ctx context.Context, limit int) ([]ContainerMetric, error) {
	query := `
        SELECT
            timestamp,
            namespace,
            pod_name,
            container_name,
            cpu_usage_percent as cpu_usage,
            memory_usage_percent as memory_usage,
            status
        FROM kubernetes_pod_container
        WHERE timestamp >= now() - INTERVAL 5 MINUTE
        ORDER BY timestamp DESC
        LIMIT ?
    `

	rows, err := ch.Client.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query container metrics: %v", err)
	}
	defer rows.Close()

	var metrics []ContainerMetric
	for rows.Next() {
		var metric ContainerMetric
		err := rows.Scan(
			&metric.Timestamp,
			&metric.Namespace,
			&metric.PodName,
			&metric.ContainerName,
			&metric.CPUUsage,
			&metric.MemoryUsage,
			&metric.Status,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan container metric row: %v", err))
			continue
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetKafkaTopicMetrics fetches Messages In Per Sec (OneMinuteRate) by Topic for specific topics from monitoring DB
// func GetKafkaTopicMetrics(ctx context.Context, topics []string) ([]KafkaTopicMetric, error) {
// 	if monitoringDBClient == nil {
// 		return nil, fmt.Errorf("monitoring DB client not initialized")
// 	}

// 	brokers := []string{
// 		"http://kafka-cluster-cp-kafka-0.broker-headless.vsmaps:8778/jolokia",
// 		"http://kafka-cluster-cp-kafka-1.broker-headless.vsmaps:8778/jolokia",
// 		"http://kafka-cluster-cp-kafka-2.broker-headless.vsmaps:8778/jolokia",
// 	}

// 	query := `
// 		SELECT
// 			t.topic AS metric,
// 			t.timestamp AS timestamp,
// 			sum(t.OneMinuteRate) AS OneMinuteRate
// 		FROM kafka_Broker_Topic_Metrics AS t
// 		INNER JOIN (
// 			SELECT
// 				topic,
// 				max(timestamp) AS latest_ts
// 			FROM kafka_Broker_Topic_Metrics
// 			WHERE
// 				name = 'MessagesInPerSec'
// 				AND timestamp >= now() - INTERVAL 10 MINUTE
// 			GROUP BY topic
// 		) AS latest
// 		ON t.topic = latest.topic AND t.timestamp = latest.latest_ts
// 		WHERE
// 			t.name = 'MessagesInPerSec'
// 			AND t.topic IN (?)
// 		GROUP BY
// 			t.topic,
// 			t.timestamp
// 		ORDER BY
// 			t.timestamp DESC
// 	`

// 	rows, err := monitoringDBClient.Client.Query(ctx, query, brokers, brokers, topics)
// 	if err != nil {
// 		return nil, fmt.Errorf("error querying Kafka topic metrics: %v", err)
// 	}
// 	defer rows.Close()

// 	var metrics []KafkaTopicMetric
// 	for rows.Next() {
// 		var m KafkaTopicMetric
// 		if err := rows.Scan(&m.Topic, &m.Timestamp, &m.OneMinuteRate); err != nil {
// 			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan Kafka topic metric row: %v", err))
// 			continue
// 		}
// 		metrics = append(metrics, m)
// 	}

// 	return metrics, nil
// }

// TopicConfig represents the structure of topics_tables.yaml
type TopicConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig represents a single O11y source in topics_tables.yaml
type SourceConfig struct {
	Name        string      `yaml:"name"`
	InputTopic  []TopicItem `yaml:"inputTopic"`
	OutputTopic []TopicItem `yaml:"outputTopic"`
}

// TopicItem represents a topic in the config
type TopicItem struct {
	Name string `yaml:"name"`
}

// loadTopicConfig loads the topic configuration from YAML file
func loadTopicConfig() (*TopicConfig, error) {
	configPath := "src/configs/topics_tables.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics config file: %v", err)
	}

	var config TopicConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse topics config: %v", err)
	}

	return &config, nil
}

// getInputTopicsFromConfig reads the topics_tables.yaml config and returns a set of all input topic names
func getInputTopicsFromConfig() map[string]bool {
	inputTopicSet := make(map[string]bool)

	config, err := loadTopicConfig()
	if err != nil {
		// If config fails, log error and return empty map
		logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to load topic config for input topic detection: %v", err))
		return inputTopicSet
	}

	// Extract all input topics from config
	for _, source := range config.Sources {
		for _, inputTopic := range source.InputTopic {
			inputTopicSet[inputTopic.Name] = true
		}
	}

	return inputTopicSet
}

func GetKafkaTopicMetrics(ctx context.Context, topics []string, timeRange TimeRange) ([]KafkaTopicMetric, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("monitoring DB client not initialized")
	}

	// Ensure topics list isn’t empty
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Querying Kafka metrics for %d topics from %s to %s", len(topics), timeRange.From, timeRange.To), "debug")
	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Topics: %v", topics), "debug")

	// Separate input topics (consumer metrics) from output topics (producer metrics)
	// Use config file to determine which topics are input vs output
	inputTopicSet := getInputTopicsFromConfig()

	var inputTopics []string
	var outputTopics []string
	for _, topic := range topics {
		if inputTopicSet[topic] {
			inputTopics = append(inputTopics, topic)
		} else {
			outputTopics = append(outputTopics, topic)
		}
	}

	var metrics []KafkaTopicMetric

	// Query input topics from consumer metrics table (MessagesInPerSec)
	if len(inputTopics) > 0 {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Querying %d input topics from consumer metrics", len(inputTopics)), "debug")

		inputTopicsList := ""
		for i, topic := range inputTopics {
			if i > 0 {
				inputTopicsList += ", "
			}
			inputTopicsList += "'" + topic + "'"
		}

		query := fmt.Sprintf(`
		SELECT
			topic,
			timestamp,
			OneMinuteRate
		FROM kafka_Broker_Topic_Metrics
		WHERE
			name = 'MessagesInPerSec'
			AND timestamp >= ?
			AND timestamp <= ?
			AND topic IN (%s)
		ORDER BY
			topic,
			timestamp ASC
		`, inputTopicsList)

		rows, err := monitoringDBClient.Client.Query(ctx, query, timeRange.From, timeRange.To)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to query input topic metrics: %v", err))
			// Continue to process output topics even if input fails
		} else {
			defer rows.Close()
			for rows.Next() {
				var m KafkaTopicMetric
				if err := rows.Scan(&m.Topic, &m.Timestamp, &m.OneMinuteRate); err != nil {
					logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan input topic row: %v", err))
					continue
				}
				m.Count = 0
				m.IsInput = true // Mark as input topic
				metrics = append(metrics, m)
			}
			if err := rows.Err(); err != nil {
				logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Error reading input topic rows: %v", err))
			}
		}

		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("✓ Retrieved input topic data"), "debug")
	}

	// Query output topics from producer metrics table
	// Output topics don't have MessagesInPerSec (consumer metrics), so we use producer send rate
	if len(outputTopics) > 0 {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Querying %d output topics from producer metrics", len(outputTopics)), "debug")

		outputTopicsList := ""
		for i, topic := range outputTopics {
			if i > 0 {
				outputTopicsList += ", "
			}
			outputTopicsList += "'" + topic + "'"
		}

		// Query producer metrics for output topics
		query := fmt.Sprintf(`
		SELECT
			topic,
			timestamp,
			"record-send-rate" as OneMinuteRate
		FROM kafka_producer_Producer_Topic_Metrics_data
		WHERE
			timestamp >= ?
			AND timestamp <= ?
			AND topic IN (%s)
		ORDER BY
			topic,
			timestamp ASC
		`, outputTopicsList)

		rows, err := monitoringDBClient.Client.Query(ctx, query, timeRange.From, timeRange.To)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to query output topic producer metrics (will show zeros): %v", err))
			// If producer metrics not available, add zero-value metrics so topics appear in output
			for _, topic := range outputTopics {
				m := KafkaTopicMetric{
					Topic:         topic,
					Timestamp:     timeRange.From,
					OneMinuteRate: 0, // No data available for output topics
					Count:         0,
				}
				metrics = append(metrics, m)
			}
		} else {
			defer rows.Close()
			outputMetricsCount := 0
			for rows.Next() {
				var m KafkaTopicMetric
				if err := rows.Scan(&m.Topic, &m.Timestamp, &m.OneMinuteRate); err != nil {
					logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan output topic row: %v", err))
					continue
				}
				m.Count = 0
				m.IsInput = false // Mark as output topic
				metrics = append(metrics, m)
				outputMetricsCount++
			}
			if err := rows.Err(); err != nil {
				logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Error reading output topic rows: %v", err))
			}

			// If no data found for output topics, add zero-value entries
			if outputMetricsCount == 0 {
				logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("⚠ No producer data found for output topics - adding zero-value entries"), "warning")
				for _, topic := range outputTopics {
					m := KafkaTopicMetric{
						Topic:         topic,
						Timestamp:     timeRange.From,
						OneMinuteRate: 0,
						Count:         0,
						IsInput:       false, // Mark as output topic
					}
					metrics = append(metrics, m)
				}
			}

			logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("✓ Retrieved output topic data"), "debug")
		}
	}

	// Note: individual query loops already check rows.Err() where appropriate.

	// Note: individual query loops already check rows.Err() where appropriate.

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("✓ Total retrieved %d Kafka metric data points", len(metrics)), "debug")

	if len(metrics) == 0 {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("⚠ No data points found for any requested topics in time range"), "warning")
	}

	return metrics, nil
}

// CollectMetrics gathers all metrics from ClickHouse for a specific time range
func (c *ClickHouseClient) CollectMetrics(timeRange TimeRange) (*ClickHouseMetrics, error) {
	ctx := context.Background()
	metrics := &ClickHouseMetrics{
		LastUpdated: time.Now(),
	}

	// List of pods to monitor (loaded from config)

	// Removed pod-related metrics collection

	// Collect Kafka topic metrics for specific topics
	kafkaTopics := []string{
		"apache-metrics-input",
		"azure-firewall-input",
		"azure-redis-cache-input",
		"vuazure-storage-blob-input",
		"linux-monitor-input",
		"mongo-metrics-input",
		"mssql-telegraf",
	}
	// Use default time range (last 5 minutes) for general metrics collection
	defaultTimeRange := TimeRange{
		From: time.Now().Add(-5 * time.Minute),
		To:   time.Now(),
	}
	kafkaTopicMetrics, err := GetKafkaTopicMetrics(ctx, kafkaTopics, defaultTimeRange)
	if err != nil {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Error collecting Kafka topic metrics: %v", err), "error")
	} else {
		metrics.KafkaTopicMetrics = kafkaTopicMetrics
	}

	// Temporarily disabled Kafka metrics
	/*
	   var kafkaMetrics []KafkaProducerMetric
	   kafkaMetrics, err = c.getKafkaProducerMetrics(ctx, 100)
	   if err != nil {
	       logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Error collecting Kafka metrics: %v", err), "error")
	   } else {
	       metrics.KafkaProducerMetrics = kafkaMetrics
	   }
	*/

	// Collect Kafka producer metrics
	/*kafkaMetrics, err := ch.getKafkaProducerMetrics(ctx, 100)
	  if err != nil {
	      logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to collect Kafka metrics: %v", err))
	  } else {
	      metrics.KafkaProducerMetrics = kafkaMetrics
	      logger.LogSuccess("System", "ClickHouse", fmt.Sprintf("Collected %d Kafka producer metrics", len(kafkaMetrics)))
	  }*/

	// Comment out other metrics collection for now - focus on Kafka producer metrics
	/*
	   // Collect system metrics
	   systemMetrics, err := ch.getSystemMetrics(ctx, 100)
	   if err != nil {
	       logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to collect system metrics: %v", err))
	   } else {
	       metrics.SystemMetrics = systemMetrics
	       logger.LogSuccess("System", "ClickHouse", fmt.Sprintf("Collected %d system metrics", len(systemMetrics)))
	   }

	   // Collect database metrics
	   dbMetrics, err := ch.getDatabaseMetrics(ctx, 100)
	   if err != nil {
	       logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to collect database metrics: %v", err))
	   } else {
	       metrics.DatabaseMetrics = dbMetrics
	       logger.LogSuccess("System", "ClickHouse", fmt.Sprintf("Collected %d database metrics", len(dbMetrics)))
	   }

	   // Collect container metrics
	   containerMetrics, err := ch.getContainerMetrics(ctx, 100)
	   if err != nil {
	       logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to collect container metrics: %v", err))
	   } else {
	       metrics.ContainerMetrics = containerMetrics
	       logger.LogSuccess("System", "ClickHouse", fmt.Sprintf("Collected %d container metrics", len(containerMetrics)))
	   }
	*/

	return metrics, nil
}

// collectClickHouseMetrics collects all metrics from ClickHouse for a specific time range
func CollectClickHouseMetrics(timeRange TimeRange) (*ClickHouseMetrics, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	metrics, err := clickHouseClient.CollectMetrics(timeRange)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Error collecting metrics: %v", err))
		return nil, err
	}

	// Debug log the collected metrics

	return metrics, nil
}
