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
	Restarts    int64       `json:"restarts"`
	LastSeen    time.Time `json:"last_seen"`
}

// GetPodMonitoringData fetches pod monitoring data for a specific namespace
func GetPodMonitoringData(ctx context.Context, namespace string) ([]PodMonitoringData, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	query := `
		WITH
		    -- Latest pod condition/phase/state/status
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
		        WHERE timestamp >= (now() - toIntervalMinute(15))
		        GROUP BY namespace, pod_name
		    ),

		    -- Readiness summary per pod
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
		        WHERE timestamp >= (now() - toIntervalMinute(15))
		        GROUP BY namespace, pod_name
		    )

		SELECT
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status AS ready,
		    l.latest_condition AS status,
		    ROUND((quantileExact(0.95)(k.cpu_usage_nanocores) / 1000000000) * 100, 2) AS cpu_usage,
		    ROUND((quantileExact(0.95)(k.memory_usage_bytes) * 100.) / NULLIF(max(k.resource_limits_memory_bytes), 0), 2) AS memory_usage,
		    max(k.restarts_total) AS restarts,
		    max(k.timestamp) AS last_seen
		FROM monitoring.kubernetes_pod_container_data AS k
		LEFT JOIN readiness_summary AS r ON k.namespace = r.namespace AND k.pod_name = r.pod_name
		LEFT JOIN latest_status AS l ON k.namespace = l.namespace AND k.pod_name = l.pod_name
		WHERE (k.timestamp >= (now() - toIntervalMinute(15)))
		  AND (k.namespace = ?)
		GROUP BY
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status,
		    l.latest_condition
		ORDER BY last_seen DESC
		;
	`

	rows, err := clickHouseClient.Client.Query(ctx, query, namespace)
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

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pods for namespace %s", len(pods), namespace), "info")
	return pods, nil
}

// GetPodMonitoringDataAllNamespaces fetches pod monitoring data for all namespaces
func GetPodMonitoringDataAllNamespaces(ctx context.Context) ([]PodMonitoringData, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
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
		        WHERE timestamp >= (now() - toIntervalMinute(15))
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
		        WHERE timestamp >= (now() - toIntervalMinute(15))
		        GROUP BY pod_name
		    )

		SELECT
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status AS ready,
		    l.latest_condition AS status,
		    ROUND((quantileExact(0.95)(k.cpu_usage_nanocores) / 1000000000) * 100, 2) AS cpu_usage,
		    ROUND((quantileExact(0.95)(k.memory_usage_bytes) * 100.) / NULLIF(max(k.resource_limits_memory_bytes), 0), 2) AS memory_usage,
		    max(k.restarts_total) AS restarts,
		    max(k.timestamp) AS last_seen
		FROM monitoring.kubernetes_pod_container_data AS k
		LEFT JOIN readiness_summary AS r ON k.pod_name = r.pod_name
		LEFT JOIN latest_status AS l ON k.pod_name = l.pod_name
		WHERE k.timestamp >= (now() - toIntervalMinute(15))
		GROUP BY
		    k.namespace,
		    k.pod_name,
		    k.node_name,
		    r.ready_status,
		    l.latest_condition
		ORDER BY last_seen DESC
		;
	`

	rows, err := clickHouseClient.Client.Query(ctx, query)
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

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pods from all namespaces", len(pods)), "info")
	return pods, nil
}
