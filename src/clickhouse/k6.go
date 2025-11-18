package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// K6Result represents a single k6 monitoring result
type K6Result struct {
	Timestamp       time.Time `json:"timestamp"`
	TestID          string    `json:"test_id"`
	NoOfUsers       uint16    `json:"no_of_users"`
	TimeFilter      string    `json:"time_filter"`
	PanelName       string    `json:"panel_name"`
	DashboardName   string    `json:"dashboard_name"`
	PanelStatus     uint16    `json:"panel_status"`
	P95ResponseTime float64   `json:"p95_response_time"`
}

// GetK6Results fetches k6 results based on the specified query
func GetK6Results(ctx context.Context, dashboard string, testID string) ([]K6Result, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
SELECT
	   timestamp AS "Timestamp",
	   test_id AS "Test ID",
	   vus AS "No of Users",
	   time_range AS "Time Filter",
	   panel_name AS "Panel Name",
	   dashboard_name AS "Dashboard Name",
	   panel_status AS "Panel Status",
	   quantile(0.9)(panel_avg_response_time) AS "P95 Response Time"
FROM monitoring.k6_results
WHERE timestamp >= now() - INTERVAL 12 HOUR
	`

	// Add test_id filter if provided
	if testID != "" {
		query += fmt.Sprintf(" AND test_id = '%s'", testID)
	}

	// Add dashboard filter if specified
	if dashboard != "" && dashboard != "all" {
		query += fmt.Sprintf(" AND dashboard_name IN ('%s')", dashboard)
	}

	query += `
GROUP BY
	   timestamp,
	   test_id,
	   vus,
	   time_range,
	   panel_name,
	   dashboard_name,
	   panel_status
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
			&result.TestID,
			&result.NoOfUsers,
			&result.TimeFilter,
			&result.PanelName,
			&result.DashboardName,
			&result.PanelStatus,
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

// GetK6TestIDs fetches distinct test_id values from k6 tables
func GetK6TestIDs(ctx context.Context) ([]string, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT DISTINCT test_id
		FROM monitoring.k6_results
		WHERE test_id != ''
		UNION
		SELECT DISTINCT test_id
		FROM monitoring.k6_login
		WHERE test_id != ''
		ORDER BY test_id
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 test IDs: %v", err))
		return nil, fmt.Errorf("failed to query k6 test IDs: %v", err)
	}
	defer rows.Close()

	var testIDs []string
	for rows.Next() {
		var testID string
		err := rows.Scan(&testID)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan test_id row: %v", err))
			continue
		}
		testIDs = append(testIDs, testID)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d distinct k6 test IDs", len(testIDs)), "info")
	return testIDs, nil
}
