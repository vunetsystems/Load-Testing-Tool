// Package kafka_processors implements node memory usage summarization for completed test runs
// This module processes test runs and generates node memory usage summaries
// based on kubernetes node data for Kafka and ClickHouse nodes.
package kafka_processors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
)

// NodeMemoryStat represents a single node memory metric data point from ClickHouse
type NodeMemoryStat struct {
	NodeLabel           string
	MaxMemoryUsedPercent float64
	MinMemoryUsedPercent float64
	AvgMemoryUsedPercent float64
}

// NodeMemoryResult represents the computed memory stats for all nodes
type NodeMemoryResult struct {
	Kafka1Min float64
	Kafka1Avg float64
	Kafka1Max float64
	Kafka2Min float64
	Kafka2Avg float64
	Kafka2Max float64
	Kafka3Min float64
	Kafka3Avg float64
	Kafka3Max float64
	Ch1Min    float64
	Ch1Avg    float64
	Ch1Max    float64
	Ch2Min    float64
	Ch2Avg    float64
	Ch2Max    float64
}

// fetchNodeMemoryMetrics fetches node memory metrics from ClickHouse for the specified time range
// This queries monitoring.kubernetes_node_data and computes memory usage percentages
func fetchNodeMemoryMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]NodeMemoryStat, error) {
	// Define the fixed node mappings
	nodeMappings := map[string]string{
		"e2e-82-181": "kafka_node_1",
		"e2e-82-234": "kafka_node_2",
		"e2e-83-134": "kafka_node_3",
		"e2e-83-184": "clickhouse_node_1",
		"e2e-83-212": "clickhouse_node_2",
	}

	// Build the IN clause for node names
	var nodeNames []string
	for node := range nodeMappings {
		nodeNames = append(nodeNames, fmt.Sprintf("'%s'", node))
	}
	nodeStr := fmt.Sprintf("(%s)", strings.Join(nodeNames, ","))

	// Build the CASE statement for node labels
	var caseStatements []string
	for node, label := range nodeMappings {
		caseStatements = append(caseStatements, fmt.Sprintf("WHEN node_name = '%s' THEN '%s'", node, label))
	}
	caseStr := strings.Join(caseStatements, "\n        ")

	query := fmt.Sprintf(`
		SELECT
		    CASE
		        %s
		        ELSE node_name
		    END AS node_label,
		    ROUND(MAX(memory_used_percent), 2) AS max_memory_used_percent,
		    ROUND(MIN(memory_used_percent), 2) AS min_memory_used_percent,
		    ROUND(AVG(memory_used_percent), 2) AS avg_memory_used_percent
		FROM
		(
		    SELECT
		        b.node_name AS node_name,
		        ROUND((SUM(b.memory_usage_bytes) / (SUM(b.memory_usage_bytes) + SUM(b.memory_available_bytes))) * 100, 2) AS memory_used_percent
		    FROM monitoring.kubernetes_node_data AS b
		    WHERE (b.timestamp >= toDateTime('%s'))
		      AND (b.timestamp <= toDateTime('%s'))
		      AND (b.node_name IN %s)
		    GROUP BY
		        toStartOfInterval(b.timestamp, toIntervalMinute(5)),
		        b.node_name
		) AS node_usage
		GROUP BY node_label
		ORDER BY node_label ASC
	`, caseStr, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"), nodeStr)

	logger.LogWithNode("System", "NodeMemoryProcessor", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []NodeMemoryStat

	for rows.Next() {
		var stat NodeMemoryStat

		if err := rows.Scan(&stat.NodeLabel, &stat.MaxMemoryUsedPercent, &stat.MinMemoryUsedPercent, &stat.AvgMemoryUsedPercent); err != nil {
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

// computeNodeMemoryStats computes the final memory statistics mapped to database columns
func computeNodeMemoryStats(stats []NodeMemoryStat) NodeMemoryResult {
	result := NodeMemoryResult{} // All fields default to 0.0

	for _, stat := range stats {
		switch stat.NodeLabel {
		case "kafka_node_1":
			result.Kafka1Min = stat.MinMemoryUsedPercent
			result.Kafka1Max = stat.MaxMemoryUsedPercent
			result.Kafka1Avg = stat.AvgMemoryUsedPercent
		case "kafka_node_2":
			result.Kafka2Min = stat.MinMemoryUsedPercent
			result.Kafka2Max = stat.MaxMemoryUsedPercent
			result.Kafka2Avg = stat.AvgMemoryUsedPercent
		case "kafka_node_3":
			result.Kafka3Min = stat.MinMemoryUsedPercent
			result.Kafka3Max = stat.MaxMemoryUsedPercent
			result.Kafka3Avg = stat.AvgMemoryUsedPercent
		case "clickhouse_node_1":
			result.Ch1Min = stat.MinMemoryUsedPercent
			result.Ch1Max = stat.MaxMemoryUsedPercent
			result.Ch1Avg = stat.AvgMemoryUsedPercent
		case "clickhouse_node_2":
			result.Ch2Min = stat.MinMemoryUsedPercent
			result.Ch2Max = stat.MaxMemoryUsedPercent
			result.Ch2Avg = stat.AvgMemoryUsedPercent
		}
	}

	return result
}

// ProcessNodeMemorySummary fetches and computes node memory statistics for a test run
func ProcessNodeMemorySummary(chClient *clickhouse.ClickHouseClient, startTime, endTime time.Time) (NodeMemoryResult, error) {
	logger.LogWithNode("System", "NodeMemoryProcessor", "Starting node memory summary processing", "info")

	stats, err := fetchNodeMemoryMetrics(chClient, startTime, endTime)
	if err != nil {
		return NodeMemoryResult{}, fmt.Errorf("failed to fetch node memory metrics: %w", err)
	}

	result := computeNodeMemoryStats(stats)

	logger.LogSuccess("System", "NodeMemoryProcessor", "Successfully processed node memory summary")
	return result, nil
}