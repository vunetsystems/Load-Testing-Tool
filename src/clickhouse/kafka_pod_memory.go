package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// KafkaPodMemoryData represents a single memory usage data point for a Kafka pod
type KafkaPodMemoryData struct {
	Timestamp   time.Time `json:"timestamp"`
	MemoryUsage int64     `json:"memory_usage"` // Memory usage in bytes
	PodName     string    `json:"pod_name"`
}

// GetKafkaPodMemoryData fetches Kafka pod memory usage data from the last 6 hours or specified time range
func GetKafkaPodMemoryData(ctx context.Context, timeRange *TimeRange) ([]KafkaPodMemoryData, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	var timeCondition string
	if timeRange != nil {
		timeCondition = fmt.Sprintf("timestamp >= '%s' AND timestamp <= '%s'",
			timeRange.From.Format("2006-01-02 15:04:05"),
			timeRange.To.Format("2006-01-02 15:04:05"))
	} else {
		timeCondition = "timestamp >= now() - toIntervalHour(6)"
	}

	query := fmt.Sprintf(`
		SELECT
			toTimeZone(timestamp, 'Asia/Kolkata') AS "Timestamp (IST)",
			memory_working_set_bytes AS "Memory Usage",
			pod_name
		FROM
			kubernetes_pod_container_data
		WHERE
			%s
			AND pod_name LIKE 'kafka-cluster-cp-kafka-%%'
			AND pod_name NOT LIKE 'kafka-cluster-cp-kafka-connect-%%'
			AND memory_working_set_bytes > 100031104
		ORDER BY "Timestamp (IST)", pod_name
	`, timeCondition)

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query Kafka pod memory data: %v", err))
		return nil, fmt.Errorf("failed to query Kafka pod memory data: %v", err)
	}
	defer rows.Close()

	var results []KafkaPodMemoryData
	for rows.Next() {
		var result KafkaPodMemoryData
		err := rows.Scan(
			&result.Timestamp,
			&result.MemoryUsage,
			&result.PodName,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan Kafka pod memory row: %v", err))
			continue
		}
		results = append(results, result)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d Kafka pod memory data points", len(results)), "info")
	return results, nil
}
