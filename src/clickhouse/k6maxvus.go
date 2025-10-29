package clickhouse

import (
	"context"
	"fmt"
	"vuDataSim/src/logger"
)

// K6MaxVusResult represents the maximum virtual users result
type K6MaxVusResult struct {
	MaxVus uint16 `json:"max_vus"`
}

// GetK6MaxVus fetches the maximum virtual users from k6 results
func GetK6MaxVus(ctx context.Context) (*K6MaxVusResult, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
		SELECT max(vus)
		FROM monitoring.k6_results;
	`

	var result K6MaxVusResult
	err := monitoringDBClient.Client.QueryRow(ctx, query).Scan(&result.MaxVus)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query k6 max vus: %v", err))
		return nil, fmt.Errorf("failed to query k6 max vus: %v", err)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched max vus: %d", result.MaxVus), "info")
	return &result, nil
}