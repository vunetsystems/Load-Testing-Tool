package clickhouse

import (
	"context"
	"fmt"
)

// GetPodResourceMetrics fetches resource utilization for specific pods
func (c *ClickHouseClient) GetPodResourceMetrics(pods []string) ([]PodResourceMetric, error) {
	query := `
		SELECT
			cluster_identifiers AS cluster_id,
			kubernetes_pod_name AS pod_name,
			avg(kubernetes_pod_cpu_usage_limit_pct) AS cpu_percentage,
			avg(kubernetes_pod_memory_usage_limit_pct) AS memory_percentage,
			max(timestamp) AS latest_timestamp
		FROM 
			vmetrics_kubernetes_kubelet_metrics_view
		WHERE 
			type = 'pod'
			AND kubernetes_pod_name IN (?)
		GROUP BY
			cluster_identifiers,
			kubernetes_pod_name`

	rows, err := c.Client.Query(context.Background(), query, pods)
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

// GetPodStatusMetrics fetches status information for specific pods
func (c *ClickHouseClient) GetPodStatusMetrics(pods []string) ([]PodStatusMetric, error) {
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

	rows, err := c.Client.Query(context.Background(), query, pods, pods)
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
