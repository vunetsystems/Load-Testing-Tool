package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// K6Result represents a single k6 monitoring result
type K6Result struct {
	Timestamp      time.Time `json:"timestamp"`
	NoOfUsers      uint16    `json:"no_of_users"`
	TimeFilter     string    `json:"time_filter"`
	PanelName      string    `json:"panel_name"`
	DashboardName  string    `json:"dashboard_name"`
	P95ResponseTime float64   `json:"p95_response_time"`
}

// GetK6Results fetches k6 results based on the specified query
func GetK6Results(ctx context.Context) ([]K6Result, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT
		  timestamp AS "Timestamp",
		  vus AS "No of Users",
		  time_range AS "Time Filter",
		  panel_name AS "Panel Name",
		  dashboard_name AS "Dashboard Name",
		  quantile(0.9)(panel_avg_response_time) AS "P95 Response time"
		FROM monitoring.k6_results
		WHERE timestamp >= now() - INTERVAL 1 HOUR
		  AND vus IN (5)
		  AND time_range IN ('6h')
		  AND dashboard_name IN ('LinuxServerInsights')
		GROUP BY timestamp, vus, time_range, panel_name, dashboard_name
		ORDER BY timestamp;
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 results: %v", err))
		return nil, fmt.Errorf("failed to query k6 results: %v", err)
	}
	defer rows.Close()

	var results []K6Result
	for rows.Next() {
		var result K6Result
		err := rows.Scan(
			&result.Timestamp,
			&result.NoOfUsers,
			&result.TimeFilter,
			&result.PanelName,
			&result.DashboardName,
			&result.P95ResponseTime,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan k6 result row: %v", err))
			continue
		}
		results = append(results, result)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d k6 results", len(results)), "info")
	return results, nil
}