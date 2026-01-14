// Package kafka_processors implements node JSON metrics summarization for completed test runs
// This module discovers all nodes in the cluster and generates JSON summaries
// of node CPU and memory usage metrics.
package kafka_processors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
)

// NodeStat represents aggregated node resource metrics
type NodeStat struct {
	NodeName      string
	MaxCPUUsed    float64
	MinCPUUsed    float64
	AvgCPUUsed    float64
	TotalCPUCores int64
	MaxMemoryUsed float64
	MinMemoryUsed float64
	AvgMemoryUsed float64
	TotalMemoryGB float64
}

// NodeCPUStatJSON represents a single node CPU metric data point from ClickHouse for JSON processing
type NodeCPUStatJSON struct {
	NodeName           string
	MaxCPUUsedPercent  float64
	MinCPUUsedPercent  float64
	AvgCPUUsedPercent  float64
	TotalCapacityCores int64
	AvgUsedCores       float64
}

// NodeMemoryStatJSON represents a single node memory metric data point from ClickHouse for JSON processing
type NodeMemoryStatJSON struct {
	NodeName             string
	MaxMemoryUsedPercent float64
	MinMemoryUsedPercent float64
	AvgMemoryUsedPercent float64
	TotalCapacityGB      float64
}

// GetDistinctNodeNames fetches all unique node names from the kubernetes_node_data table
func GetDistinctNodeNames(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT trimmed
		FROM (
			SELECT trim(BOTH ' \t\n\r\f\v' FROM node_name) AS trimmed
			FROM monitoring.kubernetes_node_data
			WHERE timestamp >= toDateTime('%s')
			AND timestamp <= toDateTime('%s')
		)
		WHERE trimmed != ''
		ORDER BY trimmed ASC

	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var nodeNames []string

	for rows.Next() {
		var nodeName string
		if err := rows.Scan(&nodeName); err != nil {
			logger.LogWarning("System", "NodeDiscovery", fmt.Sprintf("Failed to scan node name: %v", err))
			continue
		}

		nodeName = strings.TrimSpace(nodeName)
		if nodeName == "" {
			logger.LogWarning("System", "NodeDiscovery", "Skipped empty or invalid node name")
			continue
		}

		nodeNames = append(nodeNames, nodeName)
	}

	if len(nodeNames) == 0 {
		logger.LogWithNode("System", "NodeDiscovery", "No nodes found in the specified time range", "warn")
	} else {
		logger.LogSuccess("System", "NodeDiscovery", fmt.Sprintf("Discovered %d distinct nodes", len(nodeNames)))
	}

	return nodeNames, nil
}

// FetchNodeCPUMetricsForNodes fetches node CPU metrics from ClickHouse for the specified node names
func FetchNodeCPUMetricsForNodes(chClient *clickhouse.ClickHouseClient, nodeNames []string, start, end time.Time) ([]NodeCPUStatJSON, error) {
	if len(nodeNames) == 0 {
		return nil, fmt.Errorf("no node names provided")
	}

	// Build IN clause for node names
	nodeList := make([]string, len(nodeNames))
	for i, name := range nodeNames {
		nodeList[i] = fmt.Sprintf("'%s'", name)
	}
	nodeStr := strings.Join(nodeList, ",")

	query := fmt.Sprintf(`
		SELECT
		    node_name,
		    ROUND(MAX((total_nanocores / (capacity_cpu_cores * 1000000000)) * 100), 2) AS max_cpu_used_percent,
		    ROUND(MIN((total_nanocores / (capacity_cpu_cores * 1000000000)) * 100), 2) AS min_cpu_used_percent,
		    ROUND(AVG((total_nanocores / (capacity_cpu_cores * 1000000000)) * 100), 2) AS avg_cpu_used_percent,
		    any(capacity_cpu_cores) AS total_capacity_cores,
		    ROUND((AVG((total_nanocores / (capacity_cpu_cores * 1000000000)) * 100) / 100) * any(capacity_cpu_cores), 2) AS avg_used_cores
		FROM
		(
		    SELECT
		        node_name,
		        toStartOfInterval(timestamp, toIntervalMinute(5)) AS ts_bucket,
		        SUM(cpu_usage_nanocores) AS total_nanocores,
		        MAX(capacity_cpu_cores) AS capacity_cpu_cores
		    FROM monitoring.kubernetes_node_data
		    WHERE (timestamp >= toDateTime('%s'))
		      AND (timestamp <= toDateTime('%s'))
		      AND (node_name IN (%s))
		    GROUP BY node_name, ts_bucket
		    HAVING capacity_cpu_cores > 0
		) AS node_usage
		GROUP BY node_name
		ORDER BY node_name ASC
	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"), nodeStr)

	logger.LogWithNode("System", "NodeCPUProcessor", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []NodeCPUStatJSON

	for rows.Next() {
		var stat NodeCPUStatJSON
		if err := rows.Scan(
			&stat.NodeName,
			&stat.MaxCPUUsedPercent,
			&stat.MinCPUUsedPercent,
			&stat.AvgCPUUsedPercent,
			&stat.TotalCapacityCores,
			&stat.AvgUsedCores,
		); err != nil {
			logger.LogWarning("System", "NodeCPUProcessor", fmt.Sprintf("Failed to scan node CPU row: %v", err))
			continue
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "NodeCPUProcessor", "No node CPU data found for given time range", "warn")
		logger.LogWithNode("System", "NodeCPUProcessor", fmt.Sprintf("Query executed for nodes: %v", nodeNames), "debug")
	} else {
		logger.LogSuccess("System", "NodeCPUProcessor", fmt.Sprintf("Fetched node CPU stats for %d nodes", len(stats)))
		// Log the actual results for debugging
		for _, stat := range stats {
			logger.LogWithNode("System", "NodeCPUProcessor", fmt.Sprintf("Node: %s, CPU%%: Min=%.2f, Avg=%.2f, Max=%.2f, Cores=%d, AvgUsed=%.2f",
				stat.NodeName, stat.MinCPUUsedPercent, stat.AvgCPUUsedPercent, stat.MaxCPUUsedPercent,
				stat.TotalCapacityCores, stat.AvgUsedCores), "debug")
		}
	}

	return stats, nil
}

// FetchNodeMemoryMetricsForNodes fetches node memory metrics from ClickHouse for the specified node names
func FetchNodeMemoryMetricsForNodes(chClient *clickhouse.ClickHouseClient, nodeNames []string, start, end time.Time) ([]NodeMemoryStatJSON, error) {
	if len(nodeNames) == 0 {
		return nil, fmt.Errorf("no node names provided")
	}

	// Build IN clause for node names
	nodeList := make([]string, len(nodeNames))
	for i, name := range nodeNames {
		nodeList[i] = fmt.Sprintf("'%s'", name)
	}
	nodeStr := strings.Join(nodeList, ",")

	query := fmt.Sprintf(`
		SELECT
		    node_name,
		    ROUND(MAX(memory_used_percent), 2) AS max_memory_used_percent,
		    ROUND(MIN(memory_used_percent), 2) AS min_memory_used_percent,
		    ROUND(AVG(memory_used_percent), 2) AS avg_memory_used_percent,
		    ROUND(any(total_capacity_gb), 2) AS total_capacity_gb
		FROM
		(
		    SELECT
		        node_name,
		        ROUND(
		            (SUM(memory_usage_bytes) /
		             NULLIF(SUM(memory_usage_bytes) + SUM(memory_available_bytes), 0)) * 100,
		            2
		        ) AS memory_used_percent,
		        (MAX(capacity_memory_bytes) / 1024 / 1024 / 1024) AS total_capacity_gb
		    FROM monitoring.kubernetes_node_data
		    WHERE (timestamp >= toDateTime('%s'))
		      AND (timestamp <= toDateTime('%s'))
		      AND (node_name IN (%s))
		    GROUP BY
		        toStartOfInterval(timestamp, toIntervalMinute(5)),
		        node_name
		) AS node_usage
		GROUP BY node_name
		ORDER BY node_name ASC
	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"), nodeStr)

	logger.LogWithNode("System", "NodeMemoryProcessor", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []NodeMemoryStatJSON

	for rows.Next() {
		var stat NodeMemoryStatJSON
		if err := rows.Scan(
			&stat.NodeName,
			&stat.MaxMemoryUsedPercent,
			&stat.MinMemoryUsedPercent,
			&stat.AvgMemoryUsedPercent,
			&stat.TotalCapacityGB,
		); err != nil {
			logger.LogWarning("System", "NodeMemoryProcessor", fmt.Sprintf("Failed to scan node memory row: %v", err))
			continue
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "NodeMemoryProcessor", "No node memory data found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "NodeMemoryProcessor", fmt.Sprintf("Fetched node memory stats for %d nodes", len(stats)))
	}

	return stats, nil
}

// combineCPUAndMemoryStats combines CPU and memory stats into NodeStat structs
func combineCPUAndMemoryStats(cpuStats []NodeCPUStatJSON, memoryStats []NodeMemoryStatJSON) []NodeStat {
	// Create maps for easy lookup
	cpuMap := make(map[string]NodeCPUStatJSON)
	for _, stat := range cpuStats {
		cpuMap[stat.NodeName] = stat
	}

	memoryMap := make(map[string]NodeMemoryStatJSON)
	for _, stat := range memoryStats {
		memoryMap[stat.NodeName] = stat
	}

	var stats []NodeStat
	// Use all unique node names from both
	nodeSet := make(map[string]bool)
	for _, stat := range cpuStats {
		nodeSet[stat.NodeName] = true
	}
	for _, stat := range memoryStats {
		nodeSet[stat.NodeName] = true
	}

	for nodeName := range nodeSet {
		stat := NodeStat{NodeName: nodeName}

		if cpuStat, exists := cpuMap[nodeName]; exists {
			stat.MaxCPUUsed = cpuStat.MaxCPUUsedPercent
			stat.MinCPUUsed = cpuStat.MinCPUUsedPercent
			stat.AvgCPUUsed = cpuStat.AvgCPUUsedPercent
			stat.TotalCPUCores = cpuStat.TotalCapacityCores
		}

		if memoryStat, exists := memoryMap[nodeName]; exists {
			stat.MaxMemoryUsed = memoryStat.MaxMemoryUsedPercent
			stat.MinMemoryUsed = memoryStat.MinMemoryUsedPercent
			stat.AvgMemoryUsed = memoryStat.AvgMemoryUsedPercent
			stat.TotalMemoryGB = memoryStat.TotalCapacityGB
		}

		stats = append(stats, stat)
	}

	return stats
}

// ComputeNodeJSONMetrics computes nodes_cpu and nodes_memory as JSON-serializable maps
func ComputeNodeJSONMetrics(stats []NodeStat) (map[string]interface{}, map[string]interface{}, bool) {
	if len(stats) == 0 {
		return nil, nil, false
	}

	nodesCPU := make(map[string]interface{})
	nodesMemory := make(map[string]interface{})

	for _, s := range stats {
		// CPU metrics
		nodesCPU[s.NodeName] = map[string]interface{}{
			"allocated": float64(s.TotalCPUCores),
			"min":       s.MinCPUUsed,
			"avg":       s.AvgCPUUsed,
			"max":       s.MaxCPUUsed,
		}

		// Memory metrics
		nodesMemory[s.NodeName] = map[string]interface{}{
			"allocated": s.TotalMemoryGB,
			"min":       s.MinMemoryUsed,
			"avg":       s.AvgMemoryUsed,
			"max":       s.MaxMemoryUsed,
		}
	}

	return nodesCPU, nodesMemory, true
}

// ProcessNodeResourceSummaryJSON processes node resource metrics and returns JSON columns
func ProcessNodeResourceSummaryJSON(chClient *clickhouse.ClickHouseClient, start, end time.Time) (map[string]interface{}, map[string]interface{}, bool, error) {
	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Starting node resource processing for time range %s to %s", start, end), "info")

	// Step 1: Get distinct node names
	nodeNames, err := GetDistinctNodeNames(chClient, start, end)
	if err != nil {
		logger.LogError("System", "NodeProcessor", fmt.Sprintf("Failed to get distinct node names: %v", err))
		return nil, nil, false, fmt.Errorf("failed to get distinct node names: %w", err)
	}

	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Found %d distinct nodes: %v", len(nodeNames), nodeNames), "info")

	if len(nodeNames) == 0 {
		logger.LogWithNode("System", "NodeProcessor", "No nodes found in cluster", "warn")
		return nil, nil, false, nil
	}

	// Step 2: Fetch CPU metrics
	logger.LogWithNode("System", "NodeProcessor", "Fetching CPU metrics", "info")
	cpuStats, err := FetchNodeCPUMetricsForNodes(chClient, nodeNames, start, end)
	if err != nil {
		logger.LogError("System", "NodeProcessor", fmt.Sprintf("Failed to fetch node CPU metrics: %v", err))
		return nil, nil, false, fmt.Errorf("failed to fetch node CPU metrics: %w", err)
	}

	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Fetched CPU stats for %d nodes", len(cpuStats)), "info")

	// Step 3: Fetch memory metrics
	logger.LogWithNode("System", "NodeProcessor", "Fetching memory metrics", "info")
	memoryStats, err := FetchNodeMemoryMetricsForNodes(chClient, nodeNames, start, end)
	if err != nil {
		logger.LogError("System", "NodeProcessor", fmt.Sprintf("Failed to fetch node memory metrics: %v", err))
		return nil, nil, false, fmt.Errorf("failed to fetch node memory metrics: %w", err)
	}

	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Fetched memory stats for %d nodes", len(memoryStats)), "info")

	// Step 4: Combine into NodeStat structs
	stats := combineCPUAndMemoryStats(cpuStats, memoryStats)
	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Combined into %d NodeStat structs", len(stats)), "info")

	// Step 5: Compute JSON metrics
	nodesCPU, nodesMemory, found := ComputeNodeJSONMetrics(stats)
	logger.LogWithNode("System", "NodeProcessor", fmt.Sprintf("Computed JSON metrics - CPU keys: %d, Memory keys: %d, found: %t", len(nodesCPU), len(nodesMemory), found), "info")

	if !found {
		logger.LogWithNode("System", "NodeProcessor", "No node metrics found", "warn")
		return nil, nil, false, nil
	}

	return nodesCPU, nodesMemory, true, nil
}
