// Package database provides CRUD operations for K6 runs in the K6 run tracking process
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vuDataSim/src/models"

	"github.com/google/uuid"
)

// CreateK6Run creates a new K6 run record
func CreateK6Run(testName, timeRange, duration string, vus, iterations, interval int, o11ySources []string) (*models.K6Run, error) {
	// Generate a new UUID for the test ID
	testID := uuid.New().String()

	// Convert o11y sources to JSON
	sourcesJSON, err := json.Marshal(o11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal o11y sources: %w", err)
	}

	startTime := time.Now().UTC()

	query := `
		INSERT INTO k6_runs (test_id, test_name, start_time, time_range, duration, vus, iterations, interval, o11y_sources, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'running')`

	_, err = DB.Exec(query, testID, testName, startTime, timeRange, duration, vus, iterations, interval, string(sourcesJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to insert K6 run: %w", err)
	}

	return &models.K6Run{
		TestID:      testID,
		TestName:    testName,
		StartTime:   startTime,
		TimeRange:   timeRange,
		Duration:    duration,
		VUs:         vus,
		Iterations:  iterations,
		Interval:    interval,
		O11ySources: o11ySources,
		Status:      "running",
	}, nil
}

// StopK6Run updates the end time for a K6 run
func StopK6Run(testID string) error {
	endTime := time.Now().UTC()

	query := `UPDATE k6_runs SET end_time = ?, status = 'stopped' WHERE test_id = ?`
	_, err := DB.Exec(query, endTime, testID)
	if err != nil {
		return fmt.Errorf("failed to update K6 run end time: %w", err)
	}

	return nil
}

// GetK6Run retrieves a specific K6 run by ID
func GetK6Run(testID string) (*models.K6Run, error) {
	query := `
		SELECT test_id, test_name, start_time, end_time, time_range, duration, vus, iterations, interval, o11y_sources, status, Metrics_Login
		FROM k6_runs WHERE test_id = ?`

	var k6Run models.K6Run
	var sourcesJSON string
	var metricsJSON sql.NullString
	var endTime sql.NullTime
	var duration sql.NullString

	err := DB.QueryRow(query, testID).Scan(
		&k6Run.TestID,
		&k6Run.TestName,
		&k6Run.StartTime,
		&endTime,
		&k6Run.TimeRange,
		&duration,
		&k6Run.VUs,
		&k6Run.Iterations,
		&k6Run.Interval,
		&sourcesJSON,
		&k6Run.Status,
		&metricsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("K6 run with ID %s not found", testID)
		}
		return nil, fmt.Errorf("failed to query K6 run: %w", err)
	}

	if endTime.Valid {
		k6Run.EndTime = &endTime.Time
	}

	if duration.Valid {
		k6Run.Duration = duration.String
	}

	// Parse JSON sources
	err = json.Unmarshal([]byte(sourcesJSON), &k6Run.O11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal o11y sources: %w", err)
	}

	// Parse Metrics_Login JSON if present
	if metricsJSON.Valid && metricsJSON.String != "" {
		var metrics models.K6LoginMetrics
		err = json.Unmarshal([]byte(metricsJSON.String), &metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Metrics_Login: %w", err)
		}
		k6Run.MetricsLogin = &metrics
	}

	return &k6Run, nil
}

// GetAllK6Runs retrieves all K6 runs ordered by start time descending
func GetAllK6Runs() ([]*models.K6Run, error) {
	query := `
		SELECT test_id, test_name, start_time, end_time, time_range, duration, vus, iterations, interval, o11y_sources, status, Metrics_Login
		FROM k6_runs ORDER BY start_time DESC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query K6 runs: %w", err)
	}
	defer rows.Close()

	var k6Runs []*models.K6Run
	for rows.Next() {
		var k6Run models.K6Run
		var sourcesJSON string
		var metricsJSON sql.NullString
		var endTime sql.NullTime
		var duration sql.NullString

		err := rows.Scan(
			&k6Run.TestID,
			&k6Run.TestName,
			&k6Run.StartTime,
			&endTime,
			&k6Run.TimeRange,
			&duration,
			&k6Run.VUs,
			&k6Run.Iterations,
			&k6Run.Interval,
			&sourcesJSON,
			&k6Run.Status,
			&metricsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan K6 run: %w", err)
		}

		if endTime.Valid {
			k6Run.EndTime = &endTime.Time
		}

		if duration.Valid {
			k6Run.Duration = duration.String
		}

		// Parse JSON sources
		err = json.Unmarshal([]byte(sourcesJSON), &k6Run.O11ySources)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal o11y sources: %w", err)
		}

		// Parse Metrics_Login JSON if present
		if metricsJSON.Valid && metricsJSON.String != "" {
			var metrics models.K6LoginMetrics
			err = json.Unmarshal([]byte(metricsJSON.String), &metrics)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal Metrics_Login: %w", err)
			}
			k6Run.MetricsLogin = &metrics
		}

		k6Runs = append(k6Runs, &k6Run)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return k6Runs, nil
}

// GetNextK6TestID returns a new UUID that would be assigned to the next K6 test run
func GetNextK6TestID() (string, error) {
	// Generate a new UUID for preview
	nextID := uuid.New().String()
	return nextID, nil
}

// UpdateK6RunStatus updates the status of a K6 run
func UpdateK6RunStatus(testID string, status string) error {
	query := `UPDATE k6_runs SET status = ? WHERE test_id = ?`
	_, err := DB.Exec(query, status, testID)
	if err != nil {
		return fmt.Errorf("failed to update K6 run status: %w", err)
	}

	return nil
}

// CompleteK6Run marks a K6 run as completed
func CompleteK6Run(testID string) error {
	endTime := time.Now().UTC()

	query := `UPDATE k6_runs SET end_time = ?, status = 'completed' WHERE test_id = ?`
	_, err := DB.Exec(query, endTime, testID)
	if err != nil {
		return fmt.Errorf("failed to complete K6 run: %w", err)
	}

	return nil
}

// UpdateK6RunMetrics updates the Metrics_Login field for a K6 run
func UpdateK6RunMetrics(testID string, metrics *models.K6LoginMetrics) error {
	// Marshal metrics to JSON
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	query := `UPDATE k6_runs SET Metrics_Login = ? WHERE test_id = ?`
	_, err = DB.Exec(query, string(metricsJSON), testID)
	if err != nil {
		return fmt.Errorf("failed to update K6 run metrics: %w", err)
	}

	return nil
}
