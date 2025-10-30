package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

type PodLogEntry struct {
	PodName       string    `json:"pod_name"`
	Timestamp     time.Time `json:"timestamp"`
	LogLevel      string    `json:"log_level"`
	Message       string    `json:"message"`
	ContainerName string    `json:"container_name"`
}

func GetPodLogs(ctx context.Context, namespace string) ([]PodLogEntry, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT
    pod_name,
    timestamp,
    log_level,
    message,
    container_name
FROM monitoring.vlogs_platform_kubelogs
WHERE namespace_name = 'vsmaps'
  AND timestamp >= now() - toIntervalHour(24)
ORDER BY
    pod_name ASC,
    timestamp DESC
LIMIT 100 BY pod_name;
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query, namespace)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query pod logs: %v", err))
		return nil, fmt.Errorf("failed to query pod logs: %v", err)
	}
	defer rows.Close()

	var logs []PodLogEntry
	for rows.Next() {
		var log PodLogEntry
		err := rows.Scan(
			&log.PodName,
			&log.Timestamp,
			&log.LogLevel,
			&log.Message,
			&log.ContainerName,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan pod log row: %v", err))
			continue
		}
		logs = append(logs, log)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pod log entries for namespace %s", len(logs), namespace), "info")
	return logs, nil
}