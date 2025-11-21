package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vuDataSim/src/logger"
)

// K6LoginResult represents a single k6 login test result
type K6LoginResult struct {
	Timestamp       time.Time `json:"timestamp"`
	TestID          string    `json:"test_id"`
	NoOfUsers       uint16    `json:"no_of_users"`
	TestName        string    `json:"test_name"`
	P95ResponseTime float64   `json:"p95_response_time"`
}

// GetK6LoginResults fetches k6 login test results based on the specified query
func GetK6LoginResults(ctx context.Context, testID string) ([]K6LoginResult, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT
		    timestamp AS "timestamp",
		    test_id AS "Test ID",
		    vus AS "No of Users",
		    test_name AS "Test Name",
		    quantile(0.9)(avg_response_time) AS "P95 Response time"
		FROM monitoring.k6_login
	`

	whereClause := ""
	conditions := []string{}

	// If test_id is provided, don't apply time filter to show all historical data for that test
	if testID != "" {
		conditions = append(conditions, fmt.Sprintf("test_id = '%s'", testID))
	} else {
		// If no test_id, apply time filter to limit results
		conditions = append(conditions, "timestamp >= now() - toIntervalDay(1)")
	}

	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query += whereClause

	query += `
		GROUP BY
		    timestamp, test_id, vus, test_name
		ORDER BY timestamp;
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 login results: %v", err))
		return nil, fmt.Errorf("failed to query k6 login results: %v", err)
	}
	defer rows.Close()

	var results []K6LoginResult
	for rows.Next() {
		var result K6LoginResult
		err := rows.Scan(
			&result.Timestamp,
			&result.TestID,
			&result.NoOfUsers,
			&result.TestName,
			&result.P95ResponseTime,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan k6 login result row: %v", err))
			continue
		}
		results = append(results, result)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d k6 login results", len(results)), "info")
	return results, nil
}
