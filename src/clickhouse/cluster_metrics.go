package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"vuDataSim/src/logger"
)

// ClusterNodeMetrics represents metrics for a single cluster node
type ClusterNodeMetrics struct {
	CPUCores      float64 `json:"cpu_cores"`
	TotalMemoryGB float64 `json:"total_memory_gb"`
	UsedMemoryGB  float64 `json:"used_memory_gb"`
	Target        string  `json:"target"` // Add this field
}

// ClusterMetricsCache handles caching of cluster metrics
type ClusterMetricsCache struct {
	metrics    map[string]ClusterNodeMetrics
	lastUpdate time.Time
	mutex      sync.RWMutex
}

var clusterMetricsCache = &ClusterMetricsCache{
	metrics:    make(map[string]ClusterNodeMetrics),
	lastUpdate: time.Time{},
}

// GetClusterNodeMetrics fetches node metrics with caching
func GetClusterNodeMetrics() (map[string]ClusterNodeMetrics, error) {
	ctx := context.Background()

	clusterMetricsCache.mutex.RLock()
	// Return cached data if less than 30 seconds old
	if time.Since(clusterMetricsCache.lastUpdate) < 30*time.Second && len(clusterMetricsCache.metrics) > 0 {
		cached := make(map[string]ClusterNodeMetrics)
		for k, v := range clusterMetricsCache.metrics {
			cached[k] = v
		}
		clusterMetricsCache.mutex.RUnlock()
		return cached, nil
	}
	clusterMetricsCache.mutex.RUnlock()

	// Fetch fresh data
	clusterMetricsCache.mutex.Lock()
	defer clusterMetricsCache.mutex.Unlock()

	// Double-check after acquiring write lock
	if time.Since(clusterMetricsCache.lastUpdate) < 30*time.Second && len(clusterMetricsCache.metrics) > 0 {
		cached := make(map[string]ClusterNodeMetrics)
		for k, v := range clusterMetricsCache.metrics {
			cached[k] = v
		}
		return cached, nil
	}

	// Query ClickHouse with optimized aggregations
	query := `
		SELECT
			COALESCE(kubernetes_node_name, target) AS node_name,
			target,
			COALESCE(avg(kubernetes_node_cpu_usage_nanocores), 0) / 1000000000 AS avg_cpu_cores,
			COALESCE(
				max(kubernetes_node_memory_available_bytes + kubernetes_node_memory_workingset_bytes), 
				0
			) / (1024 * 1024 * 1024) AS total_memory_gb,
			COALESCE(
				avg(kubernetes_node_memory_workingset_bytes), 
				0
			) / (1024 * 1024 * 1024) AS avg_used_memory_gb
		FROM vusmart.vmetrics_kubernetes_kubelet_metrics_view
		WHERE timestamp >= now() - INTERVAL 5 MINUTE
			AND type = 'node'
			AND target != ''
			AND target IS NOT NULL
		GROUP BY kubernetes_node_name, target
		HAVING avg_cpu_cores > 0
		ORDER BY node_name;
	`

	rows, err := clickHouseClient.Client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ClickHouse: %v", err)
	}
	defer rows.Close()

	metrics := make(map[string]ClusterNodeMetrics)

	for rows.Next() {
		var nodeName, target string
		var avgCpuCores, totalMemoryGB, avgUsedMemoryGB float64

		err := rows.Scan(&nodeName, &target, &avgCpuCores, &totalMemoryGB, &avgUsedMemoryGB)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan cluster metrics row: %v", err))
			continue
		}

		// Skip if nodeName is empty after COALESCE
		if nodeName == "" {
			logger.LogWarning("System", "ClickHouse", "Skipping row with empty node name")
			continue
		}

		// Ensure reasonable values
		if avgCpuCores < 0 {
			avgCpuCores = 0
		}
		if totalMemoryGB < 1 {
			totalMemoryGB = 8 // fallback to 8GB
		}
		if avgUsedMemoryGB < 0 {
			avgUsedMemoryGB = 0
		}
		if avgUsedMemoryGB > totalMemoryGB {
			avgUsedMemoryGB = totalMemoryGB
		}

		// Use target as node name if nodeName is still empty (fallback)
		if nodeName == "" {
			nodeName = target
		}

		metrics[nodeName] = ClusterNodeMetrics{
			CPUCores:      avgCpuCores,
			TotalMemoryGB: totalMemoryGB,
			UsedMemoryGB:  avgUsedMemoryGB,
			Target:        target,
		}
	}

	// Update cache
	clusterMetricsCache.metrics = metrics
	clusterMetricsCache.lastUpdate = time.Now()

	// Add detailed debug logging
	for nodeName, nodeMetrics := range metrics {
		logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Node %s (target: %s) metrics - CPU: %.2f cores, Memory: %.2f/%.2f GB",
			nodeName, nodeMetrics.Target, nodeMetrics.CPUCores, nodeMetrics.UsedMemoryGB, nodeMetrics.TotalMemoryGB), "debug")
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched metrics for %d nodes", len(metrics)), "info")
	return metrics, nil
}
