package clickhouse

import (
	"context"
	"fmt"
	"time"

	"vuDataSim/src/logger"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouse configuration
type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ClickHouseMetrics represents aggregated metrics from ClickHouse
type ClickHouseMetrics struct {
	KafkaProducerMetrics []KafkaProducerMetric `json:"kafkaProducerMetrics,omitempty"`
	SystemMetrics        []SystemMetric        `json:"systemMetrics,omitempty"`
	DatabaseMetrics      []DatabaseMetric      `json:"databaseMetrics,omitempty"`
	ContainerMetrics     []ContainerMetric     `json:"containerMetrics,omitempty"`
	PodResourceMetrics   []PodResourceMetric   `json:"podResourceMetrics,omitempty"`
	PodStatusMetrics     []PodStatusMetric     `json:"podStatusMetrics,omitempty"`
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

// PodResourceMetric represents pod resource utilization metrics
type PodResourceMetric struct {
	ClusterID       string    `json:"clusterId"`
	PodName         string    `json:"podName"`
	CPUPercentage   float64   `json:"cpuPercentage"`
	MemoryPercentage float64   `json:"memoryPercentage"`
	LastTimestamp   time.Time `json:"lastTimestamp"`
}

// PodStatusMetric represents pod status metrics
type PodStatusMetric struct {
	ClusterID            string    `json:"clusterId"`
	NodeName             string    `json:"nodeName"`
	PodName              string    `json:"podName"`
	PodPhase             string    `json:"podPhase"`
	ContainerStatus      string    `json:"containerStatus"`
	ContainerReasons     string    `json:"containerReasons"`
	RunningContainers    uint64    `json:"runningContainers"`
	NonRunningContainers uint64    `json:"nonRunningContainers"`
	DerivedStatus        string    `json:"derivedStatus"`
}

// ClickHouseClient wraps the ClickHouse client and configuration
type ClickHouseClient struct {
	Client clickhouse.Conn
	Config ClickHouseConfig
}

// NewClickHouseClient creates a new ClickHouse client instance
func NewClickHouseClient(config ClickHouseConfig) (*ClickHouseClient, error) {
	logger.LogWithNode("System", "ClickHouse", "Initializing ClickHouse client connection", "info")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to connect to ClickHouse: %v", err))
		return nil, err
	}

	// Test the connection
	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("ClickHouse ping failed: %v", err))
		return nil, err
	}

	client := &ClickHouseClient{
		Client: conn,
		Config: config,
	}

	logger.LogSuccess("System", "ClickHouse", "ClickHouse client initialized successfully")
	return client, nil
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

// CollectMetrics gathers all metrics from ClickHouse
func (c *ClickHouseClient) CollectMetrics() (*ClickHouseMetrics, error) {
	var err error
	metrics := &ClickHouseMetrics{
		LastUpdated: time.Now(),
	}

	// List of pods to monitor
	monitoredPods := []string{
		"linuxmonitor-8d545644d-wv77v",
		"apache-metrics-6d7f45d5d8-vbmcf",
		"mssql-telegraf-pipeline-dcffcd5f6-kqmch",
	}

	// Collect pod resource metrics
	podResourceMetrics, err := c.GetPodResourceMetrics(monitoredPods)
	if err != nil {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Error collecting pod resource metrics: %v", err), "error")
	} else {
		metrics.PodResourceMetrics = podResourceMetrics
	}

	// Collect pod status metrics
	podStatusMetrics, err := c.GetPodStatusMetrics(monitoredPods)
	if err != nil {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Error collecting pod status metrics: %v", err), "error")
	} else {
		metrics.PodStatusMetrics = podStatusMetrics
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

// HealthCheck performs a health check on the ClickHouse connection
func (ch *ClickHouseClient) HealthCheck() error {
	ctx := context.Background()
	return ch.Client.Ping(ctx)
}

// Close closes the ClickHouse client connection
func (ch *ClickHouseClient) Close() error {
	return ch.Client.Close()
}
