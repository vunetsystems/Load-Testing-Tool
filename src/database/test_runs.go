// Package database provides CRUD operations for test runs in the test run tracking process
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vuDataSim/src/bin_control"
	"vuDataSim/src/models"

	"github.com/google/uuid"
)

// CreateTestRun creates a new test run record
func CreateTestRun(testName string, targetEPS int, o11ySources []string, timeoutSeconds int) (*models.TestRun, error) {
	// Generate a new UUID for the test ID
	testID := uuid.New().String()

	// Convert o11y sources to JSON
	sourcesJSON, err := json.Marshal(o11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal o11y sources: %w", err)
	}

	startTime := time.Now().UTC()

	query := `
		INSERT INTO test_runs (test_id, test_name, target_eps, start_time, o11y_sources, timeout_seconds, status)
		VALUES (?, ?, ?, ?, ?, ?, 'running')`

	_, err = DB.Exec(query, testID, testName, targetEPS, startTime, string(sourcesJSON), timeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to insert test run: %w", err)
	}

	return &models.TestRun{
		TestID:         testID,
		TestName:       testName,
		TargetEPS:      targetEPS,
		StartTime:      startTime,
		O11ySources:    o11ySources,
		TimeoutSeconds: timeoutSeconds,
		Status:         "running",
	}, nil
}

// StopTestRun updates the stop time for a test run
func StopTestRun(testID string) error {
	stopTime := time.Now().UTC()

	query := `UPDATE test_runs SET end_time = ?, status = 'stopped' WHERE test_id = ?`
	_, err := DB.Exec(query, stopTime, testID)
	if err != nil {
		return fmt.Errorf("failed to update test run stop time: %w", err)
	}

	return nil
}

// GetTestRun retrieves a specific test run by ID
func GetTestRun(testID string) (*models.TestRun, error) {
	query := `
		SELECT test_id, test_name, target_eps, start_time, end_time, o11y_sources, timeout_seconds, status,
			   min_input_msgs_per_sec, avg_input_msgs_per_sec, max_input_msgs_per_sec,
			   min_output_msgs_per_sec, avg_output_msgs_per_sec, max_output_msgs_per_sec,
			   min_lag, avg_lag, max_lag,
			   kafka_summary_generated,
			   process_rate_summary, ingestion_summary,
			   pods_cpu, pods_memory, pod_restarts, topic_metrics_json
		FROM test_runs WHERE test_id = ?`

	var testRun models.TestRun
	var sourcesJSON string
	var startTimeStr string
	var endTimeStr sql.NullString
	var processRateSummary sql.NullString // Process rate summary JSON
	var ingestionSummary sql.NullString   // Ingestion summary JSON
	var topicMetricsJSON sql.NullString   // Topic metrics JSON

	err := DB.QueryRow(query, testID).Scan(
		&testRun.TestID,
		&testRun.TestName,
		&testRun.TargetEPS,
		&startTimeStr,
		&endTimeStr,
		&sourcesJSON,
		&testRun.TimeoutSeconds,
		&testRun.Status,
		&testRun.MinInputMsgsPerSec,
		&testRun.AvgInputMsgsPerSec,
		&testRun.MaxInputMsgsPerSec,
		&testRun.MinOutputMsgsPerSec,
		&testRun.AvgOutputMsgsPerSec,
		&testRun.MaxOutputMsgsPerSec,
		&testRun.MinLag,
		&testRun.AvgLag,
		&testRun.MaxLag,
		&testRun.KafkaSummaryGenerated,
		&processRateSummary,
		&ingestionSummary,
		&testRun.PodsCpu,
		&testRun.PodsMemory,
		&testRun.PodRestarts,
		&topicMetricsJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test run with ID %s not found", testID)
		}
		return nil, fmt.Errorf("failed to query test run: %w", err)
	}

	// Parse start_time
	testRun.StartTime, err = time.Parse(time.RFC3339Nano, startTimeStr)
	if err != nil {
		// Try alternative format if RFC3339Nano fails
		testRun.StartTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start_time '%s': %w", startTimeStr, err)
		}
	}

	if endTimeStr.Valid {
		endTime, err := time.Parse(time.RFC3339Nano, endTimeStr.String)
		if err != nil {
			// Try alternative format if RFC3339Nano fails
			endTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", endTimeStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse end_time '%s': %w", endTimeStr.String, err)
			}
		}
		testRun.EndTime = &endTime
	}
	if processRateSummary.Valid {
		testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
	}
	if ingestionSummary.Valid {
		testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
	}
	if topicMetricsJSON.Valid {
		testRun.TopicMetricsJSON = topicMetricsJSON.String // Assign topic metrics JSON
	}

	// Parse JSON sources
	err = json.Unmarshal([]byte(sourcesJSON), &testRun.O11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal o11y sources: %w", err)
	}

	return &testRun, nil
}

// GetAllTestRuns retrieves all test runs ordered by start time descending
func GetAllTestRuns() ([]*models.TestRun, error) {
	query := `
		SELECT test_id, test_name, target_eps, start_time, end_time, o11y_sources, timeout_seconds, status,
			   min_input_msgs_per_sec, avg_input_msgs_per_sec, max_input_msgs_per_sec,
			   min_output_msgs_per_sec, avg_output_msgs_per_sec, max_output_msgs_per_sec,
			   min_lag, avg_lag, max_lag,
			   kafka_summary_generated,
			   process_rate_summary, ingestion_summary,
			   pods_cpu, pods_memory, pod_restarts, topic_metrics_json
		FROM test_runs ORDER BY start_time DESC`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query test runs: %w", err)
	}
	defer rows.Close()

	var testRuns []*models.TestRun
	for rows.Next() {
		var testRun models.TestRun
		var sourcesJSON string
		var startTimeStr string
		var endTimeStr sql.NullString
		var processRateSummary sql.NullString // Process rate summary JSON
		var ingestionSummary sql.NullString   // Ingestion summary JSON
		var podsCpu sql.NullString
		var podsMemory sql.NullString
		var podRestarts sql.NullString
		var topicMetricsJSON sql.NullString // Topic metrics JSON

		err := rows.Scan(
			&testRun.TestID,
			&testRun.TestName,
			&testRun.TargetEPS,
			&startTimeStr,
			&endTimeStr,
			&sourcesJSON,
			&testRun.TimeoutSeconds,
			&testRun.Status,
			&testRun.MinInputMsgsPerSec,
			&testRun.AvgInputMsgsPerSec,
			&testRun.MaxInputMsgsPerSec,
			&testRun.MinOutputMsgsPerSec,
			&testRun.AvgOutputMsgsPerSec,
			&testRun.MaxOutputMsgsPerSec,
			&testRun.MinLag,
			&testRun.AvgLag,
			&testRun.MaxLag,
			&testRun.KafkaSummaryGenerated,
			&processRateSummary,
			&ingestionSummary,
			&podsCpu,
			&podsMemory,
			&podRestarts,
			&topicMetricsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		// Parse start_time
		testRun.StartTime, err = time.Parse(time.RFC3339Nano, startTimeStr)
		if err != nil {
			// Try alternative format if RFC3339Nano fails
			testRun.StartTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", startTimeStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse start_time '%s': %w", startTimeStr, err)
			}
		}

		if endTimeStr.Valid {
			endTime, err := time.Parse(time.RFC3339Nano, endTimeStr.String)
			if err != nil {
				// Try alternative format if RFC3339Nano fails
				endTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", endTimeStr.String)
				if err != nil {
					return nil, fmt.Errorf("failed to parse end_time '%s': %w", endTimeStr.String, err)
				}
			}
			testRun.EndTime = &endTime
		}
		if processRateSummary.Valid {
			testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
		}
		if ingestionSummary.Valid {
			testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
		}
		if podsCpu.Valid {
			testRun.PodsCpu = podsCpu.String
		}
		if podsMemory.Valid {
			testRun.PodsMemory = podsMemory.String
		}
		if podRestarts.Valid {
			testRun.PodRestarts = podRestarts.String
		}
		if topicMetricsJSON.Valid {
			testRun.TopicMetricsJSON = topicMetricsJSON.String
		}

		// Parse JSON sources
		err = json.Unmarshal([]byte(sourcesJSON), &testRun.O11ySources)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal o11y sources: %w", err)
		}

		testRuns = append(testRuns, &testRun)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return testRuns, nil
}

// GetNextTestID returns a new UUID that would be assigned to the next test run
func GetNextTestID() (string, error) {
	// Generate a new UUID for preview
	nextID := uuid.New().String()
	return nextID, nil
}

// UpdateTestRunStatus updates the status of a test run
func UpdateTestRunStatus(testID string, status string) error {
	query := `UPDATE test_runs SET status = ? WHERE test_id = ?`
	_, err := DB.Exec(query, status, testID)
	if err != nil {
		return fmt.Errorf("failed to update test run status: %w", err)
	}

	return nil
}

// CompleteTimedOutTestRuns checks for running test runs that have exceeded their timeout
// or where no binaries are running, and marks them as completed
func CompleteTimedOutTestRuns(bc *bin_control.BinaryControl) error {
	currentTime := time.Now().UTC()

	// Check if any binaries are running
	statusResp, err := bc.GetAllBinaryStatuses()
	if err != nil {
		return fmt.Errorf("failed to get binary statuses: %w", err)
	}

	anyBinaryRunning := false
	if statusResp.Success && statusResp.Data != nil {
		statuses, ok := statusResp.Data.([]bin_control.BinaryStatus)
		if ok {
			for _, status := range statuses {
				if status.Status == "running" {
					anyBinaryRunning = true
					break
				}
			}
		}
	}

	var query string
	var args []interface{}

	if !anyBinaryRunning {
		// No binaries running - complete ALL running test runs that have been running for at least 30 seconds
		query = `UPDATE test_runs SET end_time = ?, status = 'completed' WHERE status = 'running' AND datetime(start_time, '+30 seconds') <= ?`
		args = []interface{}{currentTime, currentTime}
	} else {
		// Binaries running - complete only timed-out runs
		query = `
			UPDATE test_runs
			SET end_time = ?, status = 'completed'
			WHERE status = 'running'
			AND datetime(start_time, '+' || timeout_seconds || ' seconds') <= ?
		`
		args = []interface{}{currentTime, currentTime}
	}

	_, err = DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to complete test runs: %w", err)
	}

	return nil
}

// UpdateTestRunPodMetrics updates the pods_cpu, pods_memory, and pod_restarts JSON fields for a test run
func UpdateTestRunPodMetrics(testID string, podsCpu, podsMemory, podRestarts string) error {
	query := `UPDATE test_runs SET pods_cpu = ?, pods_memory = ?, pod_restarts = ? WHERE test_id = ?`
	_, err := DB.Exec(query, podsCpu, podsMemory, podRestarts, testID)
	if err != nil {
		return fmt.Errorf("failed to update test run pod metrics: %w", err)
	}
	return nil
}

// GetTestRunO11ySources retrieves only the o11y_sources for a test run by ID
func GetTestRunO11ySources(testID string) ([]string, error) {
	query := `SELECT o11y_sources FROM test_runs WHERE test_id = ?`

	var sourcesJSON string
	err := DB.QueryRow(query, testID).Scan(&sourcesJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test run with ID %s not found", testID)
		}
		return nil, fmt.Errorf("failed to query o11y_sources: %w", err)
	}

	var o11ySources []string
	err = json.Unmarshal([]byte(sourcesJSON), &o11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal o11y sources: %w", err)
	}

	return o11ySources, nil
}

// UpdateLagPerSource updates only the lag_per_source column
func UpdateLagPerSource(testID string, lagPerSourceJSON string) error {
	query := `UPDATE test_runs SET lag_per_source = ? WHERE test_id = ?`
	_, err := DB.Exec(query, lagPerSourceJSON, testID)
	if err != nil {
		return fmt.Errorf("failed to update lag_per_source: %w", err)
	}
	return nil
}

// GetLagPerSource retrieves only the lag_per_source data
func GetLagPerSource(testID string) (sql.NullString, error) {
	query := `SELECT lag_per_source FROM test_runs WHERE test_id = ?`
	var lagPerSource sql.NullString
	err := DB.QueryRow(query, testID).Scan(&lagPerSource)
	if err != nil {
		if err == sql.ErrNoRows {
			return lagPerSource, fmt.Errorf("test run with ID %s not found", testID)
		}
		return lagPerSource, fmt.Errorf("failed to query lag_per_source: %w", err)
	}
	return lagPerSource, nil
}
