// Package kafka_processors implements node CPU usage summarization for completed test runs
// This module processes test runs and generates node CPU usage summaries
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

// NodeCPUStat represents a single node CPU metric data point from ClickHouse
type NodeCPUStat struct {
	NodeLabel           string
	MaxCPUUsedPercent float64
	MinCPUUsedPercent float64
	AvgCPUUsedPercent float64
}

// NodeCPUResult represents the computed CPU stats for all nodes
type NodeCPUResult struct {
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

// fetchNodeCPUMetrics fetches node CPU metrics from ClickHouse for the specified time range
// This queries monitoring.kubernetes_node_data and computes CPU usage percentages
func fetchNodeCPUMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]NodeCPUStat, error) {
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
		    ROUND(MAX(cpu_used_percent), 2) AS max_cpu_used_percent,
		    ROUND(MIN(cpu_used_percent), 2) AS min_cpu_used_percent,
		    ROUND(AVG(cpu_used_percent), 2) AS avg_cpu_used_percent
		FROM
		(
		    SELECT
		        b.node_name AS node_name,
		        ROUND(
		            (SUM(b.cpu_usage_nanocores) / (SUM(b.capacity_cpu_cores) * 1000000000)) * 100,
		            2
		        ) AS cpu_used_percent
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

	logger.LogWithNode("System", "NodeCPUProcessor", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []NodeCPUStat

	for rows.Next() {
		var stat NodeCPUStat

		if err := rows.Scan(&stat.NodeLabel, &stat.MaxCPUUsedPercent, &stat.MinCPUUsedPercent, &stat.AvgCPUUsedPercent); err != nil {
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
	} else {
		logger.LogSuccess("System", "NodeCPUProcessor", fmt.Sprintf("Fetched node CPU stats for %d nodes", len(stats)))
	}

	return stats, nil
}

// computeNodeCPUStats computes the final CPU statistics mapped to database columns
func computeNodeCPUStats(stats []NodeCPUStat) NodeCPUResult {
	result := NodeCPUResult{} // All fields default to 0.0

	for _, stat := range stats {
		switch stat.NodeLabel {
		case "kafka_node_1":
			result.Kafka1Min = stat.MinCPUUsedPercent
			result.Kafka1Max = stat.MaxCPUUsedPercent
			result.Kafka1Avg = stat.AvgCPUUsedPercent
		case "kafka_node_2":
			result.Kafka2Min = stat.MinCPUUsedPercent
			result.Kafka2Max = stat.MaxCPUUsedPercent
			result.Kafka2Avg = stat.AvgCPUUsedPercent
		case "kafka_node_3":
			result.Kafka3Min = stat.MinCPUUsedPercent
			result.Kafka3Max = stat.MaxCPUUsedPercent
			result.Kafka3Avg = stat.AvgCPUUsedPercent
		case "clickhouse_node_1":
			result.Ch1Min = stat.MinCPUUsedPercent
			result.Ch1Max = stat.MaxCPUUsedPercent
			result.Ch1Avg = stat.AvgCPUUsedPercent
		case "clickhouse_node_2":
			result.Ch2Min = stat.MinCPUUsedPercent
			result.Ch2Max = stat.MaxCPUUsedPercent
			result.Ch2Avg = stat.AvgCPUUsedPercent
		}
	}

	return result
}

// ProcessNodeCPUSummary fetches and computes node CPU statistics for a test run
func ProcessNodeCPUSummary(chClient *clickhouse.ClickHouseClient, startTime, endTime time.Time) (NodeCPUResult, error) {
	logger.LogWithNode("System", "NodeCPUProcessor", "Starting node CPU summary processing", "info")

	stats, err := fetchNodeCPUMetrics(chClient, startTime, endTime)
	if err != nil {
		return NodeCPUResult{}, fmt.Errorf("failed to fetch node CPU metrics: %w", err)
	}

	result := computeNodeCPUStats(stats)

	logger.LogSuccess("System", "NodeCPUProcessor", "Successfully processed node CPU summary")
	return result, nil
}