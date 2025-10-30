package clickhouse

import (
	"context"
	"fmt"
	"vuDataSim/src/logger"
)

// K6SuccessRate represents the average success rate from k6 results
type K6SuccessRate struct {
	AvgSuccessRate float64 `json:"avg_success_rate"`
}

// GetK6SuccessRate fetches the average success rate from k6 results
func GetK6SuccessRate(ctx context.Context) (*K6SuccessRate, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT avg(dashboard_success_rate) AS avg_success_rate
		FROM monitoring.k6_results
	`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 success rate: %v", err))
		return nil, fmt.Errorf("failed to query k6 success rate: %v", err)
	}
	defer rows.Close()

	var successRate K6SuccessRate
	if rows.Next() {
		err := rows.Scan(
			&successRate.AvgSuccessRate,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan k6 success rate row: %v", err))
			return nil, fmt.Errorf("failed to scan k6 success rate row: %v", err)
		}
	} else {
		logger.LogWarning("System", "ClickHouse", "No k6 success rate data found")
		return nil, fmt.Errorf("no k6 success rate data found")
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched k6 success rate: %.2f%%", successRate.AvgSuccessRate), "info")
	return &successRate, nil
}