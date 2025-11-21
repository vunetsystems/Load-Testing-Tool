package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// PodMonitoringData represents a single pod's monitoring data
type PodMonitoringData struct {
	Namespace   string    `json:"namespace"`
	PodName     string    `json:"pod_name"`
	NodeName    string    `json:"node_name"`
	Ready       string    `json:"ready"`
	Status      string    `json:"status"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	Restarts    int64     `json:"restarts"`
	LastSeen    time.Time `json:"last_seen"`
}

func GetPodMonitoringData(ctx context.Context, namespace string, timeRange *TimeRange) ([]PodMonitoringData, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	// Default time range if not provided
	if timeRange == nil {
		now := time.Now()
		timeRange = &TimeRange{
			From: now.Add(-15 * time.Minute),
			To:   now,
		}
	}

	query := `
		WITH
		    latest_status AS
		    (
		        SELECT
		            namespace,
		            pod_name,
		            anyLast(
		                COALESCE(
		                    NULLIF(condition, ''),
		                    NULLIF(phase, ''),
		                    NULLIF(state, ''),
		                    NULLIF(status, '')
		                )
		            ) AS latest_condition
		        FROM monitoring.kubernetes_pod_container_data
		        WHERE timestamp >= ? AND timestamp <= ?
		        GROUP BY namespace, pod_name
		    ),

		    readiness_summary AS
		    (
		        SELECT
		            namespace,
		            pod_name,
		            uniqExact(container_name) AS total_containers,
		            uniqExactIf(container_name, readiness = 'ready') AS ready_containers,
		            concat(
		                toString(uniqExactIf(container_name, readiness = 'ready')),
		                '/',
		                toString(uniqExact(container_name))
		            ) AS ready_status
		        FROM monitoring.kubernetes_pod_container_data
		        WHERE timestamp >= ? AND timestamp <= ?
		        GROUP BY namespace, pod_name
		    )

		SELECT
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status AS ready,
		    l.latest_condition AS status,

		    -- ✅ CPU usage % using actual resource limits (millicores)
		    ROUND(
		        (quantileExact(0.95)(k.cpu_usage_nanocores) / 1_000_000) /
		        NULLIF(quantileExact(0.95)(k.resource_limits_millicpu_units), 0) * 100,
		        2
		    ) AS cpu_usage_percent,

		    -- ✅ Memory usage % using actual memory limits (bytes)
		    ROUND(
		        (quantileExact(0.95)(k.memory_usage_bytes) /
		        NULLIF(quantileExact(0.95)(k.resource_limits_memory_bytes), 0)) * 100,
		        2
		    ) AS memory_usage_percent,

		    max(k.restarts_total) AS restarts,
		    max(k.timestamp) AS last_seen
		FROM monitoring.kubernetes_pod_container_data AS k
		LEFT JOIN readiness_summary AS r
		    ON k.namespace = r.namespace AND k.pod_name = r.pod_name
		LEFT JOIN latest_status AS l
		    ON k.namespace = l.namespace AND k.pod_name = l.pod_name
		WHERE (k.timestamp >= ? AND k.timestamp <= ?)
		  AND (k.namespace = ?)
		GROUP BY
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status,
		    l.latest_condition
		ORDER BY last_seen DESC
	`

	rows, err := clickHouseClient.Client.Query(ctx, query, timeRange.From, timeRange.To, timeRange.From, timeRange.To, timeRange.From, timeRange.To, namespace)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query pod monitoring data: %v", err))
		return nil, fmt.Errorf("failed to query pod monitoring data: %v", err)
	}
	defer rows.Close()

	var pods []PodMonitoringData
	for rows.Next() {
		var pod PodMonitoringData
		err := rows.Scan(
			&pod.Namespace,
			&pod.PodName,
			&pod.NodeName,
			&pod.Ready,
			&pod.Status,
			&pod.CPUUsage,
			&pod.MemoryUsage,
			&pod.Restarts,
			&pod.LastSeen,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan pod monitoring row: %v", err))
			continue
		}
		pods = append(pods, pod)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pods for namespace %s in time range %v to %v", len(pods), namespace, timeRange.From, timeRange.To), "info")
	return pods, nil
}

func GetPodMonitoringDataAllNamespaces(ctx context.Context, timeRange *TimeRange) ([]PodMonitoringData, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	// Default time range if not provided
	if timeRange == nil {
		now := time.Now()
		timeRange = &TimeRange{
			From: now.Add(-15 * time.Minute),
			To:   now,
		}
	}

	query := `
		WITH
		    latest_status AS
		    (
		        SELECT
		            pod_name,
		            anyLast(
		                COALESCE(
		                    NULLIF(condition, ''),
		                    NULLIF(phase, ''),
		                    NULLIF(state, ''),
		                    NULLIF(status, '')
		                )
		            ) AS latest_condition
		        FROM monitoring.kubernetes_pod_container_data
		        WHERE timestamp >= ? AND timestamp <= ?
		        GROUP BY pod_name
		    ),

		    readiness_summary AS
		    (
		        SELECT
		            pod_name,
		            uniqExact(container_name) AS total_containers,
		            uniqExactIf(container_name, readiness = 'ready') AS ready_containers,
		            concat(
		                toString(uniqExactIf(container_name, readiness = 'ready')),
		                '/',
		                toString(uniqExact(container_name))
		            ) AS ready_status
		        FROM monitoring.kubernetes_pod_container_data
		        WHERE timestamp >= ? AND timestamp <= ?
		        GROUP BY pod_name
		    )

		SELECT
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status AS ready,
		    l.latest_condition AS status,

		    -- ✅ CPU usage % using actual limits
		    ROUND(
		        (quantileExact(0.95)(k.cpu_usage_nanocores) / 1_000_000) /
		        NULLIF(quantileExact(0.95)(k.resource_limits_millicpu_units), 0) * 100,
		        2
		    ) AS cpu_usage_percent,

		    -- ✅ Memory usage % using actual limits
		    ROUND(
		        (quantileExact(0.95)(k.memory_usage_bytes) /
		        NULLIF(quantileExact(0.95)(k.resource_limits_memory_bytes), 0)) * 100,
		        2
		    ) AS memory_usage_percent,

		    max(k.restarts_total) AS restarts,
		    max(k.timestamp) AS last_seen
		FROM monitoring.kubernetes_pod_container_data AS k
		LEFT JOIN readiness_summary AS r ON k.pod_name = r.pod_name
		LEFT JOIN latest_status AS l ON k.pod_name = l.pod_name
		WHERE k.timestamp >= ? AND k.timestamp <= ?
		GROUP BY
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status,
		    l.latest_condition
		ORDER BY last_seen DESC
	`

	rows, err := clickHouseClient.Client.Query(ctx, query, timeRange.From, timeRange.To, timeRange.From, timeRange.To, timeRange.From, timeRange.To)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query pod monitoring data: %v", err))
		return nil, fmt.Errorf("failed to query pod monitoring data: %v", err)
	}
	defer rows.Close()

	var pods []PodMonitoringData
	for rows.Next() {
		var pod PodMonitoringData
		err := rows.Scan(
			&pod.Namespace,
			&pod.PodName,
			&pod.NodeName,
			&pod.Ready,
			&pod.Status,
			&pod.CPUUsage,
			&pod.MemoryUsage,
			&pod.Restarts,
			&pod.LastSeen,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan pod monitoring row: %v", err))
			continue
		}
		pods = append(pods, pod)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pods from all namespaces in time range %v to %v", len(pods), timeRange.From, timeRange.To), "info")
	return pods, nil
}

// PodTrendData represents time-series data for pod metrics
type PodTrendData struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
}

func GetPodTrendData(ctx context.Context, namespace, podName string, hours int, timeRange *TimeRange) ([]PodTrendData, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	var query string
	var args []interface{}

	if timeRange != nil {
		// Use specific time range with 10-minute buffer
		query = `
			SELECT
			    toStartOfInterval(timestamp, INTERVAL 1 minute) AS time_bucket,

			    ROUND(
			        (avg(cpu_usage_nanocores) / 1_000_000) /
			        NULLIF(avg(resource_limits_millicpu_units), 0) * 100,
			        2
			    ) AS avg_cpu_usage_percent,

			    ROUND(
			        (avg(memory_usage_bytes) /
			        NULLIF(avg(resource_limits_memory_bytes), 0)) * 100,
			        2
			    ) AS avg_memory_usage_percent

			FROM monitoring.kubernetes_pod_container_data
			WHERE timestamp >= ? - INTERVAL 10 minute
			  AND timestamp <= ? + INTERVAL 10 minute
			  AND namespace = ?
			  AND pod_name = ?
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`
		args = []interface{}{timeRange.From, timeRange.To, namespace, podName}
	} else {
		// Use hours-based range with 10-minute buffer
		query = `
			SELECT
			    toStartOfInterval(timestamp, INTERVAL 1 minute) AS time_bucket,

			    ROUND(
			        (avg(cpu_usage_nanocores) / 1_000_000) /
			        NULLIF(avg(resource_limits_millicpu_units), 0) * 100,
			        2
			    ) AS avg_cpu_usage_percent,

			    ROUND(
			        (avg(memory_usage_bytes) /
			        NULLIF(avg(resource_limits_memory_bytes), 0)) * 100,
			        2
			    ) AS avg_memory_usage_percent

			FROM monitoring.kubernetes_pod_container_data
			WHERE timestamp >= now() - toIntervalHour(?) - INTERVAL 10 minute
			  AND timestamp <= now() + INTERVAL 10 minute
			  AND namespace = ?
			  AND pod_name = ?
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`
		args = []interface{}{hours, namespace, podName}
	}

	rows, err := clickHouseClient.Client.Query(ctx, query, args...)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query pod trend data: %v", err))
		return nil, fmt.Errorf("failed to query pod trend data: %v", err)
	}
	defer rows.Close()

	var trendData []PodTrendData
	for rows.Next() {
		var data PodTrendData
		err := rows.Scan(
			&data.Timestamp,
			&data.CPUUsage,
			&data.MemoryUsage,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan pod trend row: %v", err))
			continue
		}
		trendData = append(trendData, data)
	}

	timeRangeDesc := "last hours"
	if timeRange != nil {
		timeRangeDesc = fmt.Sprintf("time range %v to %v (with ±10 min buffer)", timeRange.From, timeRange.To)
	}

	logger.LogWithNode("System", "ClickHouse",
		fmt.Sprintf("Fetched %d trend data points for pod %s/%s in %s", len(trendData), namespace, podName, timeRangeDesc),
		"info",
	)
	return trendData, nil
}
