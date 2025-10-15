package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// TimeRange represents a time window for metrics queries
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
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
	ClusterID        string    `json:"clusterId"`
	PodName          string    `json:"podName"`
	CPUPercentage    float64   `json:"cpuPercentage"`
	MemoryPercentage float64   `json:"memoryPercentage"`
	LastTimestamp    time.Time `json:"lastTimestamp"`
}

// PodStatusMetric represents pod status metrics
type PodStatusMetric struct {
	ClusterID            string `json:"clusterId"`
	NodeName             string `json:"nodeName"`
	PodName              string `json:"podName"`
	PodPhase             string `json:"podPhase"`
	ContainerStatus      string `json:"containerStatus"`
	ContainerReasons     string `json:"containerReasons"`
	RunningContainers    uint64 `json:"runningContainers"`
	NonRunningContainers uint64 `json:"nonRunningContainers"`
	DerivedStatus        string `json:"derivedStatus"`
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

// CollectMetrics gathers all metrics from ClickHouse for a specific time range
func (c *ClickHouseClient) CollectMetrics(timeRange TimeRange) (*ClickHouseMetrics, error) {
	ctx := context.Background()
	metrics := &ClickHouseMetrics{
		LastUpdated: time.Now(),
	}

	// List of pods to monitor
	monitoredPods := []string{
		"linuxmonitor-8d545644d-wv77v",
		"apache-metrics-6d7f45d5d8-vbmcf",
		"mssql-telegraf-pipeline-dcffcd5f6-kqmch",
		"chi-clickhouse-vusmart-0-0-0",
		"chi-clickhouse-vusmart-0-1-0",
		"apache-metrics-6d7f45d5d8-gdzlv",
	}

	// Collect pod resource metrics
	podResourceMetrics, err := c.GetPodResourceMetrics(ctx, monitoredPods, timeRange)
	if err != nil {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Error collecting pod resource metrics: %v", err), "error")
	} else {
		metrics.PodResourceMetrics = podResourceMetrics
	}

	// Collect pod status metrics
	podStatusMetrics, err := c.GetPodStatusMetrics(ctx, monitoredPods, timeRange)
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

// GetPodResourceMetrics fetches resource utilization for specific pods within a time range
func (c *ClickHouseClient) GetPodResourceMetrics(ctx context.Context, pods []string, timeRange TimeRange) ([]PodResourceMetric, error) {
	query := `
        SELECT
            cluster_identifiers AS cluster_id,
            kubernetes_pod_name AS pod_name,
            AVG(kubernetes_pod_cpu_usage_limit_pct) AS avg_cpu_pct,
            AVG(kubernetes_pod_memory_usage_limit_pct) AS avg_memory_pct,
            MAX(timestamp) AS latest_timestamp
        FROM
            vmetrics_kubernetes_kubelet_metrics_view
        WHERE
            type = 'pod'
            AND kubernetes_pod_name IN (?)
            AND timestamp BETWEEN ? AND ?
        GROUP BY
            cluster_identifiers,
            kubernetes_pod_name
        ORDER BY
            latest_timestamp DESC`

	rows, err := c.Client.Query(ctx, query, pods, timeRange.From, timeRange.To)
	if err != nil {
		return nil, fmt.Errorf("error querying pod resource metrics: %v", err)
	}
	defer rows.Close()

	var metrics []PodResourceMetric
	for rows.Next() {
		var m PodResourceMetric
		if err := rows.Scan(&m.ClusterID, &m.PodName, &m.CPUPercentage, &m.MemoryPercentage, &m.LastTimestamp); err != nil {
			return nil, fmt.Errorf("error scanning pod resource metrics: %v", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetPodStatusMetrics fetches status information for specific pods within a time range
func (c *ClickHouseClient) GetPodStatusMetrics(ctx context.Context, pods []string, timeRange TimeRange) ([]PodStatusMetric, error) {
	query := `
        WITH
        pod_latest AS (
        SELECT
            cluster_identifiers,
            kubernetes_namespace,
            kubernetes_pod_name,
            argMax(kubernetes_node_name, timestamp) AS node_name,
            argMax(kubernetes_pod_status_phase, timestamp) AS pod_phase
        FROM vmetrics_kubernetes_kube_state_metrics_view
        WHERE
            type = 'state_pod'
            AND kubernetes_pod_name IN (?)
            AND timestamp BETWEEN ? AND ?
        GROUP BY cluster_identifiers, kubernetes_namespace, kubernetes_pod_name
        ),
        container_latest AS (
        SELECT
            cluster_identifiers,
            kubernetes_namespace,
            kubernetes_pod_name,
            kubernetes_container_name,
            argMax(kubernetes_container_status_phase, timestamp) AS container_phase,
            argMax(kubernetes_container_status_ready, timestamp) AS container_ready,
            argMax(kubernetes_container_status_reason, timestamp) AS container_reason
        FROM vmetrics_kubernetes_kube_state_metrics_view
        WHERE
            type = 'state_container'
            AND kubernetes_pod_name IN (?)
        GROUP BY cluster_identifiers, kubernetes_namespace, kubernetes_pod_name, kubernetes_container_name
        ),
        container_rollup AS (
        SELECT
            cluster_identifiers,
            kubernetes_namespace,
            kubernetes_pod_name,
            count() > 0 AS containers_exist,
            arrayStringConcat(groupArray(concat(kubernetes_container_name, '=', lower(toString(container_phase)))), ', ') AS containers_status,
            arrayStringConcat(arrayFilter(x -> x != '', groupArray(container_reason)), ', ') AS container_reasons,
            any(container_reason) AS first_container_reason,
            sumIf(1, lower(toString(container_phase)) = 'running') AS running_containers,
            sumIf(1, lower(toString(container_phase)) != 'running') AS non_running_containers
        FROM container_latest
        GROUP BY cluster_identifiers, kubernetes_namespace, kubernetes_pod_name
        )
        SELECT
            p.cluster_identifiers,
            p.node_name,
            p.kubernetes_pod_name,
            lower(p.pod_phase),
            coalesce(c.containers_status, ''),
            coalesce(c.container_reasons, ''),
            coalesce(c.running_containers, 0),
            coalesce(c.non_running_containers, 0),
            CASE
                WHEN lower(p.pod_phase) = 'pending' AND NOT coalesce(c.containers_exist, 0)
                THEN 'Pending'
                WHEN c.first_container_reason != ''
                THEN c.first_container_reason
                ELSE lower(p.pod_phase)
            END AS derived_status
        FROM pod_latest p
        LEFT JOIN container_rollup c
            ON  c.cluster_identifiers = p.cluster_identifiers
            AND c.kubernetes_namespace = p.kubernetes_namespace
            AND c.kubernetes_pod_name = p.kubernetes_pod_name`

	rows, err := c.Client.Query(ctx, query, pods, timeRange.From, timeRange.To, pods)
	if err != nil {
		return nil, fmt.Errorf("error querying pod status metrics: %v", err)
	}
	defer rows.Close()

	var metrics []PodStatusMetric
	for rows.Next() {
		var m PodStatusMetric
		if err := rows.Scan(
			&m.ClusterID,
			&m.NodeName,
			&m.PodName,
			&m.PodPhase,
			&m.ContainerStatus,
			&m.ContainerReasons,
			&m.RunningContainers,
			&m.NonRunningContainers,
			&m.DerivedStatus,
		); err != nil {
			return nil, fmt.Errorf("error scanning pod status metrics: %v", err)
		}
		metrics = append(metrics, m)
	}

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
