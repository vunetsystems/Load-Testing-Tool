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
			   kafka_1_node_cpu_min, kafka_1_node_cpu_avg, kafka_1_node_cpu_max,
			   kafka_1_node_mem_min, kafka_1_node_mem_avg, kafka_1_node_mem_max,
			   kafka_2_node_cpu_min, kafka_2_node_cpu_avg, kafka_2_node_cpu_max,
			   kafka_2_node_mem_min, kafka_2_node_mem_avg, kafka_2_node_mem_max,
			   kafka_3_node_cpu_min, kafka_3_node_cpu_avg, kafka_3_node_cpu_max,
			   kafka_3_node_mem_min, kafka_3_node_mem_avg, kafka_3_node_mem_max,
			   ch1_node_cpu_min, ch1_node_cpu_avg, ch1_node_cpu_max,
			   ch1_node_mem_min, ch1_node_mem_avg, ch1_node_mem_max,
			   ch2_node_cpu_min, ch2_node_cpu_avg, ch2_node_cpu_max,
			   ch2_node_mem_min, ch2_node_mem_avg, ch2_node_mem_max,
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
			   min_input_msgs_per_sec, avg_input_msgs_per_sec, max_input_msgs_per_sec,
			   min_output_msgs_per_sec, avg_output_msgs_per_sec, max_output_msgs_per_sec,
			   min_lag, avg_lag, max_lag,
			   data_loss_pct, kafka_summary_generated, pod_resource_check,
			   process_rate_summary, ingestion_summary
		FROM test_runs WHERE test_id = ?`

	var testRun models.TestRun
	var sourcesJSON string
	var endTime sql.NullTime
	var processRateSummary sql.NullString // Process rate summary JSON
	var ingestionSummary sql.NullString   // Ingestion summary JSON

	err := DB.QueryRow(query, testID).Scan(
		&testRun.TestID,
		&testRun.TestName,
		&testRun.TargetEPS,
		&testRun.StartTime,
		&endTime,
		&sourcesJSON,
		&testRun.TimeoutSeconds,
		&testRun.Status,
		&testRun.Kafka1NodeCpuMin,
		&testRun.Kafka1NodeCpuAvg,
		&testRun.Kafka1NodeCpuMax,
		&testRun.Kafka1NodeMemMin,
		&testRun.Kafka1NodeMemAvg,
		&testRun.Kafka1NodeMemMax,
		&testRun.Kafka2NodeCpuMin,
		&testRun.Kafka2NodeCpuAvg,
		&testRun.Kafka2NodeCpuMax,
		&testRun.Kafka2NodeMemMin,
		&testRun.Kafka2NodeMemAvg,
		&testRun.Kafka2NodeMemMax,
		&testRun.Kafka3NodeCpuMin,
		&testRun.Kafka3NodeCpuAvg,
		&testRun.Kafka3NodeCpuMax,
		&testRun.Kafka3NodeMemMin,
		&testRun.Kafka3NodeMemAvg,
		&testRun.Kafka3NodeMemMax,
		&testRun.Ch1NodeCpuMin,
		&testRun.Ch1NodeCpuAvg,
		&testRun.Ch1NodeCpuMax,
		&testRun.Ch1NodeMemMin,
		&testRun.Ch1NodeMemAvg,
		&testRun.Ch1NodeMemMax,
		&testRun.Ch2NodeCpuMin,
		&testRun.Ch2NodeCpuAvg,
		&testRun.Ch2NodeCpuMax,
		&testRun.Ch2NodeMemMin,
		&testRun.Ch2NodeMemAvg,
		&testRun.Ch2NodeMemMax,
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
		&testRun.MinInputMsgsPerSec,
		&testRun.AvgInputMsgsPerSec,
		&testRun.MaxInputMsgsPerSec,
		&testRun.MinOutputMsgsPerSec,
		&testRun.AvgOutputMsgsPerSec,
		&testRun.MaxOutputMsgsPerSec,
		&testRun.MinLag,
		&testRun.AvgLag,
		&testRun.MaxLag,
		&testRun.DataLossPct,
		&testRun.KafkaSummaryGenerated,
		&testRun.PodResourceCheck,
		&processRateSummary,
		&ingestionSummary,
		&testRun.TraefikCpuAllocated,
		&testRun.TraefikMemAllocated,
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
	if processRateSummary.Valid {
		testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
	}
	if ingestionSummary.Valid {
		testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
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
			   kafka_1_node_cpu_min, kafka_1_node_cpu_avg, kafka_1_node_cpu_max,
			   kafka_1_node_mem_min, kafka_1_node_mem_avg, kafka_1_node_mem_max,
			   kafka_2_node_cpu_min, kafka_2_node_cpu_avg, kafka_2_node_cpu_max,
			   kafka_2_node_mem_min, kafka_2_node_mem_avg, kafka_2_node_mem_max,
			   kafka_3_node_cpu_min, kafka_3_node_cpu_avg, kafka_3_node_cpu_max,
			   kafka_3_node_mem_min, kafka_3_node_mem_avg, kafka_3_node_mem_max,
			   ch1_node_cpu_min, ch1_node_cpu_avg, ch1_node_cpu_max,
			   ch1_node_mem_min, ch1_node_mem_avg, ch1_node_mem_max,
			   ch2_node_cpu_min, ch2_node_cpu_avg, ch2_node_cpu_max,
			   ch2_node_mem_min, ch2_node_mem_avg, ch2_node_mem_max,
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
			   min_input_msgs_per_sec, avg_input_msgs_per_sec, max_input_msgs_per_sec,
			   min_output_msgs_per_sec, avg_output_msgs_per_sec, max_output_msgs_per_sec,
			   min_lag, avg_lag, max_lag,
			   data_loss_pct, kafka_summary_generated, pod_resource_check,
			   process_rate_summary, ingestion_summary, traefik_cpu_allocated, traefik_mem_allocated
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
		var processRateSummary sql.NullString // Process rate summary JSON
		var ingestionSummary sql.NullString // Ingestion summary JSON

		err := rows.Scan(
			&testRun.TestID,
			&testRun.TestName,
			&testRun.TargetEPS,
			&testRun.StartTime,
			&endTime,
			&sourcesJSON,
			&testRun.TimeoutSeconds,
			&testRun.Status,
			&testRun.Kafka1NodeCpuMin,
			&testRun.Kafka1NodeCpuAvg,
			&testRun.Kafka1NodeCpuMax,
			&testRun.Kafka1NodeMemMin,
			&testRun.Kafka1NodeMemAvg,
			&testRun.Kafka1NodeMemMax,
			&testRun.Kafka2NodeCpuMin,
			&testRun.Kafka2NodeCpuAvg,
			&testRun.Kafka2NodeCpuMax,
			&testRun.Kafka2NodeMemMin,
			&testRun.Kafka2NodeMemAvg,
			&testRun.Kafka2NodeMemMax,
			&testRun.Kafka3NodeCpuMin,
			&testRun.Kafka3NodeCpuAvg,
			&testRun.Kafka3NodeCpuMax,
			&testRun.Kafka3NodeMemMin,
			&testRun.Kafka3NodeMemAvg,
			&testRun.Kafka3NodeMemMax,
			&testRun.Ch1NodeCpuMin,
			&testRun.Ch1NodeCpuAvg,
			&testRun.Ch1NodeCpuMax,
			&testRun.Ch1NodeMemMin,
			&testRun.Ch1NodeMemAvg,
			&testRun.Ch1NodeMemMax,
			&testRun.Ch2NodeCpuMin,
			&testRun.Ch2NodeCpuAvg,
			&testRun.Ch2NodeCpuMax,
			&testRun.Ch2NodeMemMin,
			&testRun.Ch2NodeMemAvg,
			&testRun.Ch2NodeMemMax,
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
			&testRun.MinInputMsgsPerSec,
			&testRun.AvgInputMsgsPerSec,
			&testRun.MaxInputMsgsPerSec,
			&testRun.MinOutputMsgsPerSec,
			&testRun.AvgOutputMsgsPerSec,
			&testRun.MaxOutputMsgsPerSec,
			&testRun.MinLag,
			&testRun.AvgLag,
			&testRun.MaxLag,
			&testRun.DataLossPct,
			&testRun.KafkaSummaryGenerated,
			&testRun.PodResourceCheck,
			&processRateSummary,
			&ingestionSummary,
			&testRun.TraefikCpuAllocated,
			&testRun.TraefikMemAllocated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		if endTime.Valid {
			testRun.EndTime = &endTime.Time
		}
		if processRateSummary.Valid {
			testRun.ProcessRateSummary = processRateSummary.String // Assign process rate summary JSON
		}
		if ingestionSummary.Valid {
			testRun.IngestionSummary = ingestionSummary.String // Assign ingestion summary JSON
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