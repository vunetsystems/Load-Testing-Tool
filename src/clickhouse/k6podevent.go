package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

type PodEventEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	IncUnixTime   uint64     `json:"inc_unix_time"`
	EventType     string    `json:"event_type"`
	Reason        string    `json:"reason"`
	Message       string    `json:"message"`
	Host          string    `json:"host"`
	NamespaceName string    `json:"namespace_name"`
	PodName       string    `json:"pod_name"`
}

func GetPodEvents(ctx context.Context, namespace string) ([]PodEventEntry, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT
		    timestamp,
		    inc_unix_time,
		    event_type,
		    reason,
		    message,
		    host,
		    namespace_name,
		    pod_name
		FROM monitoring.vlogs_platform_k8s_events
		WHERE namespace_name = 'vsmaps'
		  AND timestamp >= now() - toIntervalHour(24)
		ORDER BY
		    pod_name ASC,
		    timestamp DESC
		LIMIT 100 BY pod_name;
		`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query pod events: %v", err))
		return nil, fmt.Errorf("failed to query pod events: %v", err)
	}
	defer rows.Close()

	var events []PodEventEntry
	for rows.Next() {
		var event PodEventEntry
		err := rows.Scan(
			&event.Timestamp,
			&event.IncUnixTime,
			&event.EventType,
			&event.Reason,
			&event.Message,
			&event.Host,
			&event.NamespaceName,
			&event.PodName,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan pod event row: %v", err))
			continue
		}
		events = append(events, event)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d pod event entries for namespace %s", len(events), namespace), "info")
	return events, nil
}