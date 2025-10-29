package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// K6DashboardResult represents a single k6 dashboard monitoring result
type K6DashboardResult struct {
	Timestamp      time.Time `json:"timestamp"`
	NoOfUsers      uint16    `json:"no_of_users"`
	TimeFilter     string    `json:"time_filter"`
	DashboardName  string    `json:"dashboard_name"`
	P95ResponseTime float64   `json:"p95_response_time"`
}

// GetK6DashboardResults fetches k6 dashboard results based on the specified query
func GetK6DashboardResults(ctx context.Context) ([]K6DashboardResult, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT
		    timestamp AS "Timestamp",
		    vus AS "No of Users",
		    time_range AS "Time Filter",
		    dashboard_name AS "Dashboard Name",
		    quantile(0.9)(dashboard_avg_response_time) AS "P95 Response time"
		FROM monitoring.k6_results
		WHERE
		    timestamp >= (now() - INTERVAL 1 HOUR) - INTERVAL 5 HOUR - INTERVAL 30 MINUTE
		    AND vus IN (5)
		    AND time_range IN ('15m')
		GROUP BY
		    timestamp, vus, time_range, dashboard_name
		ORDER BY
		    timestamp;
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 dashboard results: %v", err))
		return nil, fmt.Errorf("failed to query k6 dashboard results: %v", err)
	}
	defer rows.Close()

	var results []K6DashboardResult
	for rows.Next() {
		var result K6DashboardResult
		err := rows.Scan(
			&result.Timestamp,
			&result.NoOfUsers,
			&result.TimeFilter,
			&result.DashboardName,
			&result.P95ResponseTime,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan k6 dashboard result row: %v", err))
			continue
		}
		results = append(results, result)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d k6 dashboard results", len(results)), "info")
	return results, nil
}