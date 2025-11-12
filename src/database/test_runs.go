// Package database provides CRUD operations for test runs in the test run tracking process
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vuDataSim/src/models"
	"github.com/google/uuid"
)

// CreateTestRun creates a new test run record
func CreateTestRun(targetEPS int, o11ySources []string, timeoutSeconds int) (*models.TestRun, error) {
	// Generate a new UUID for the test ID
	testID := uuid.New().String()

	// Convert o11y sources to JSON
	sourcesJSON, err := json.Marshal(o11ySources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal o11y sources: %w", err)
	}

	startTime := time.Now().UTC()

	query := `
		INSERT INTO test_runs (test_id, target_eps, start_time, o11y_sources, timeout_seconds, status)
		VALUES (?, ?, ?, ?, ?, 'running')`

	_, err = DB.Exec(query, testID, targetEPS, startTime, string(sourcesJSON), timeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to insert test run: %w", err)
	}

	return &models.TestRun{
		TestID:         testID,
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
		SELECT test_id, target_eps, start_time, end_time, o11y_sources, timeout_seconds, status,
			   avg_input_msgs_per_sec, avg_output_msgs_per_sec,
			   max_input_msgs_per_sec, max_output_msgs_per_sec, min_input_msgs_per_sec, min_output_msgs_per_sec,
			   data_loss_pct, kafka_summary_generated, pod_resource_check, pod_metrics,
			   kafka_cluster_cp_kafka_0_cpu_min, kafka_cluster_cp_kafka_0_cpu_avg, kafka_cluster_cp_kafka_0_cpu_max,
			   kafka_cluster_cp_kafka_0_mem_min, kafka_cluster_cp_kafka_0_mem_avg, kafka_cluster_cp_kafka_0_mem_max,
			   kafka_cluster_cp_kafka_1_cpu_min, kafka_cluster_cp_kafka_1_cpu_avg, kafka_cluster_cp_kafka_1_cpu_max,
			   kafka_cluster_cp_kafka_1_mem_min, kafka_cluster_cp_kafka_1_mem_avg, kafka_cluster_cp_kafka_1_mem_max,
			   kafka_cluster_cp_kafka_2_cpu_min, kafka_cluster_cp_kafka_2_cpu_avg, kafka_cluster_cp_kafka_2_cpu_max,
			   kafka_cluster_cp_kafka_2_mem_min, kafka_cluster_cp_kafka_2_mem_avg, kafka_cluster_cp_kafka_2_mem_max,
			   chi_clickhouse_vusmart_0_0_0_cpu_min, chi_clickhouse_vusmart_0_0_0_cpu_avg, chi_clickhouse_vusmart_0_0_0_cpu_max,
			   chi_clickhouse_vusmart_0_0_0_mem_min, chi_clickhouse_vusmart_0_0_0_mem_avg, chi_clickhouse_vusmart_0_0_0_mem_max,
			   chi_clickhouse_vusmart_0_1_0_cpu_min, chi_clickhouse_vusmart_0_1_0_cpu_avg, chi_clickhouse_vusmart_0_1_0_cpu_max,
			   chi_clickhouse_vusmart_0_1_0_mem_min, chi_clickhouse_vusmart_0_1_0_mem_avg, chi_clickhouse_vusmart_0_1_0_mem_max,
			   pipeline_pod_cpu_min, pipeline_pod_cpu_avg, pipeline_pod_cpu_max,
			   pipeline_pod_mem_min, pipeline_pod_mem_avg, pipeline_pod_mem_max,
			   process_rate_summary, ingestion_summary
		FROM test_runs WHERE test_id = ?`

	var testRun models.TestRun
	var sourcesJSON string
	var endTime sql.NullTime
	var podMetrics sql.NullString
	var processRateSummary sql.NullString // New column for process rate summary JSON
	var ingestionSummary sql.NullString   // New ingestion summary column
	var minLag, avgLag, maxLag float64

	err := DB.QueryRow(query, testID).Scan(
		&testRun.TestID,
		&testRun.TargetEPS,
		&testRun.StartTime,
		&endTime,
		&sourcesJSON,
		&testRun.TimeoutSeconds,
		&testRun.Status,
		&testRun.AvgInputMsgsPerSec,
		&testRun.AvgOutputMsgsPerSec,
		&testRun.MaxInputMsgsPerSec,
		&testRun.MaxOutputMsgsPerSec,
		&testRun.MinInputMsgsPerSec,
		&testRun.MinOutputMsgsPerSec,
		&testRun.DataLossPct,
		&testRun.KafkaSummaryGenerated,
		&testRun.PodResourceCheck,
		&podMetrics,
		&testRun.KafkaClusterCpKafka0CpuMin,
		&testRun.KafkaClusterCpKafka0CpuAvg,
		&testRun.KafkaClusterCpKafka0CpuMax,
		&testRun.KafkaClusterCpKafka0MemMin,
		&testRun.KafkaClusterCpKafka0MemAvg,
		&testRun.KafkaClusterCpKafka0MemMax,
		&testRun.KafkaClusterCpKafka1CpuMin,
		&testRun.KafkaClusterCpKafka1CpuAvg,
		&testRun.KafkaClusterCpKafka1CpuMax,
		&testRun.KafkaClusterCpKafka1MemMin,
		&testRun.KafkaClusterCpKafka1MemAvg,
		&testRun.KafkaClusterCpKafka1MemMax,
		&testRun.KafkaClusterCpKafka2CpuMin,
		&testRun.KafkaClusterCpKafka2CpuAvg,
		&testRun.KafkaClusterCpKafka2CpuMax,
		&testRun.KafkaClusterCpKafka2MemMin,
		&testRun.KafkaClusterCpKafka2MemAvg,
		&testRun.KafkaClusterCpKafka2MemMax,
		&testRun.ChiClickhouseVusmart000CpuMin,
		&testRun.ChiClickhouseVusmart000CpuAvg,
		&testRun.ChiClickhouseVusmart000CpuMax,
		&testRun.ChiClickhouseVusmart000MemMin,
		&testRun.ChiClickhouseVusmart000MemAvg,
		&testRun.ChiClickhouseVusmart000MemMax,
		&testRun.ChiClickhouseVusmart010CpuMin,
		&testRun.ChiClickhouseVusmart010CpuAvg,
		&testRun.ChiClickhouseVusmart010CpuMax,
		&testRun.ChiClickhouseVusmart010MemMin,
		&testRun.ChiClickhouseVusmart010MemAvg,
		&testRun.ChiClickhouseVusmart010MemMax,
		&testRun.PipelinePodCpuMin,
		&testRun.PipelinePodCpuAvg,
		&testRun.PipelinePodCpuMax,
		&testRun.PipelinePodMemMin,
		&testRun.PipelinePodMemAvg,
		&testRun.PipelinePodMemMax,
		&processRateSummary, // New process rate summary column
		&ingestionSummary, // New ingestion summary column
		&minLag, &avgLag, &maxLag,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test run with ID %s not found", testID)
		}
		return nil, fmt.Errorf("failed to query test run: %w", err)
	}

	if endTime.Valid {
		testRun.EndTime = &endTime.Time
	}
	if podMetrics.Valid {
		testRun.PodMetrics = podMetrics.String
	}
	if processRateSummary.Valid {
		testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
	}
	if ingestionSummary.Valid {
		testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
	}
	testRun.MinLag = minLag
	testRun.AvgLag = avgLag
	testRun.MaxLag = maxLag

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
		SELECT test_id, target_eps, start_time, end_time, o11y_sources, timeout_seconds, status,
			   avg_input_msgs_per_sec, avg_output_msgs_per_sec,
			   max_input_msgs_per_sec, max_output_msgs_per_sec, min_input_msgs_per_sec, min_output_msgs_per_sec,
			   data_loss_pct, kafka_summary_generated, pod_resource_check, pod_metrics,
			   kafka_cluster_cp_kafka_0_cpu_min, kafka_cluster_cp_kafka_0_cpu_avg, kafka_cluster_cp_kafka_0_cpu_max,
			   kafka_cluster_cp_kafka_0_mem_min, kafka_cluster_cp_kafka_0_mem_avg, kafka_cluster_cp_kafka_0_mem_max,
			   kafka_cluster_cp_kafka_1_cpu_min, kafka_cluster_cp_kafka_1_cpu_avg, kafka_cluster_cp_kafka_1_cpu_max,
			   kafka_cluster_cp_kafka_1_mem_min, kafka_cluster_cp_kafka_1_mem_avg, kafka_cluster_cp_kafka_1_mem_max,
			   kafka_cluster_cp_kafka_2_cpu_min, kafka_cluster_cp_kafka_2_cpu_avg, kafka_cluster_cp_kafka_2_cpu_max,
			   kafka_cluster_cp_kafka_2_mem_min, kafka_cluster_cp_kafka_2_mem_avg, kafka_cluster_cp_kafka_2_mem_max,
			   chi_clickhouse_vusmart_0_0_0_cpu_min, chi_clickhouse_vusmart_0_0_0_cpu_avg, chi_clickhouse_vusmart_0_0_0_cpu_max,
			   chi_clickhouse_vusmart_0_0_0_mem_min, chi_clickhouse_vusmart_0_0_0_mem_avg, chi_clickhouse_vusmart_0_0_0_mem_max,
			   chi_clickhouse_vusmart_0_1_0_cpu_min, chi_clickhouse_vusmart_0_1_0_cpu_avg, chi_clickhouse_vusmart_0_1_0_cpu_max,
			   chi_clickhouse_vusmart_0_1_0_mem_min, chi_clickhouse_vusmart_0_1_0_mem_avg, chi_clickhouse_vusmart_0_1_0_mem_max,
			   pipeline_pod_cpu_min, pipeline_pod_cpu_avg, pipeline_pod_cpu_max,
			   pipeline_pod_mem_min, pipeline_pod_mem_avg, pipeline_pod_mem_max,
			   process_rate_summary, ingestion_summary,
			   min_lag, avg_lag, max_lag
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
		var endTime sql.NullTime
		var podMetrics sql.NullString
		var processRateSummary sql.NullString // New column for process rate summary JSON
		var ingestionSummary sql.NullString // New column for ingestion summary JSON
		var minLag, avgLag, maxLag float64

		err := rows.Scan(
			&testRun.TestID,
			&testRun.TargetEPS,
			&testRun.StartTime,
			&endTime,
			&sourcesJSON,
			&testRun.TimeoutSeconds,
			&testRun.Status,
			&testRun.AvgInputMsgsPerSec,
			&testRun.AvgOutputMsgsPerSec,
			&testRun.MaxInputMsgsPerSec,
			&testRun.MaxOutputMsgsPerSec,
			&testRun.MinInputMsgsPerSec,
			&testRun.MinOutputMsgsPerSec,
			&testRun.DataLossPct,
			&testRun.KafkaSummaryGenerated,
			&testRun.PodResourceCheck,
			&podMetrics,
			&testRun.KafkaClusterCpKafka0CpuMin,
			&testRun.KafkaClusterCpKafka0CpuAvg,
			&testRun.KafkaClusterCpKafka0CpuMax,
			&testRun.KafkaClusterCpKafka0MemMin,
			&testRun.KafkaClusterCpKafka0MemAvg,
			&testRun.KafkaClusterCpKafka0MemMax,
			&testRun.KafkaClusterCpKafka1CpuMin,
			&testRun.KafkaClusterCpKafka1CpuAvg,
			&testRun.KafkaClusterCpKafka1CpuMax,
			&testRun.KafkaClusterCpKafka1MemMin,
			&testRun.KafkaClusterCpKafka1MemAvg,
			&testRun.KafkaClusterCpKafka1MemMax,
			&testRun.KafkaClusterCpKafka2CpuMin,
			&testRun.KafkaClusterCpKafka2CpuAvg,
			&testRun.KafkaClusterCpKafka2CpuMax,
			&testRun.KafkaClusterCpKafka2MemMin,
			&testRun.KafkaClusterCpKafka2MemAvg,
			&testRun.KafkaClusterCpKafka2MemMax,
			&testRun.ChiClickhouseVusmart000CpuMin,
			&testRun.ChiClickhouseVusmart000CpuAvg,
			&testRun.ChiClickhouseVusmart000CpuMax,
			&testRun.ChiClickhouseVusmart000MemMin,
			&testRun.ChiClickhouseVusmart000MemAvg,
			&testRun.ChiClickhouseVusmart000MemMax,
			&testRun.ChiClickhouseVusmart010CpuMin,
			&testRun.ChiClickhouseVusmart010CpuAvg,
			&testRun.ChiClickhouseVusmart010CpuMax,
			&testRun.ChiClickhouseVusmart010MemMin,
			&testRun.ChiClickhouseVusmart010MemAvg,
			&testRun.ChiClickhouseVusmart010MemMax,
			&testRun.PipelinePodCpuMin,
			&testRun.PipelinePodCpuAvg,
			&testRun.PipelinePodCpuMax,
			&testRun.PipelinePodMemMin,
			&testRun.PipelinePodMemAvg,
			&testRun.PipelinePodMemMax,
			&processRateSummary,
			&ingestionSummary, // New ingestion summary column
			&minLag, &avgLag, &maxLag,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		if endTime.Valid {
			testRun.EndTime = &endTime.Time
		}
		if podMetrics.Valid {
			testRun.PodMetrics = podMetrics.String
		}
		if processRateSummary.Valid {
			testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
		}
		if ingestionSummary.Valid {
			testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
		}
		testRun.MinLag = minLag
		testRun.AvgLag = avgLag
		testRun.MaxLag = maxLag

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
// and marks them as completed
func CompleteTimedOutTestRuns() error {
	currentTime := time.Now().UTC()

	query := `
		UPDATE test_runs
		SET end_time = ?, status = 'completed'
		WHERE status = 'running'
		AND datetime(start_time, '+' || timeout_seconds || ' seconds') <= ?
	`

	_, err := DB.Exec(query, currentTime, currentTime)
	if err != nil {
		return fmt.Errorf("failed to complete timed out test runs: %w", err)
	}

	return nil
}