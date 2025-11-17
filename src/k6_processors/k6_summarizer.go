// Package k6_processors implements K6 summarization for completed test runs
// This module processes test runs that have completed and generates segment-wise start and stop time summaries
// based on the k6_login table data for each test.
package k6_processors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vuDataSim/src/database"
	"vuDataSim/src/logger"
	"vuDataSim/src/models"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// K6Summary represents the computed summary of segment times for a K6 test run
type K6Summary struct {
	TestID           string             `json:"test_id"`
	StartTime        time.Time          `json:"start_time"`
	EndTime          time.Time          `json:"end_time"`
	DurationMinutes  float64            `json:"duration_minutes"`
	SegmentSummaries []K6SegmentSummary `json:"segment_summaries"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

// K6SegmentSummary represents summary for a specific segment
type K6SegmentSummary struct {
	SegmentNumber uint8     `json:"segment_number"`
	SegmentStart  time.Time `json:"segment_start"`
	SegmentEnd    time.Time `json:"segment_end"`
}

// K6Segment represents a single segment data point from ClickHouse
type K6Segment struct {
	Source        string    `json:"source"`
	SegmentNumber uint8     `json:"segment_number"`
	SegmentStart  time.Time `json:"segment_start"`
	SegmentEnd    time.Time `json:"segment_end"`
}

// EnsureK6RunsTableColumns adds the missing columns to k6_runs table if they don't exist
func EnsureK6RunsTableColumns(db *sql.DB) error {
	logger.LogWithNode("System", "K6Summarizer", "Ensuring k6_runs table has required columns for K6 summarization", "info")

	// Check and add summarised column
	if !columnExists(db, "k6_runs", "summarised") {
		_, err := db.Exec("ALTER TABLE k6_runs ADD COLUMN summarised BOOLEAN DEFAULT FALSE")
		if err != nil {
			return fmt.Errorf("failed to add summarised column: %w", err)
		}
	}

	// Check and add k6_summary column
	if !columnExists(db, "k6_runs", "k6_summary") {
		_, err := db.Exec("ALTER TABLE k6_runs ADD COLUMN k6_summary TEXT")
		if err != nil {
			return fmt.Errorf("failed to add k6_summary column: %w", err)
		}
	}

	logger.LogSuccess("System", "K6Summarizer", "K6 runs table columns verified/added successfully")
	return nil
}

// columnExists checks if a column exists in the specified table
func columnExists(db *sql.DB, table, column string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt_value interface{}
		var pk int
		err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk)
		if err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

// QueryK6Segments queries ClickHouse for segment start/end times for a specific test run
func QueryK6Segments(ctx context.Context, chClient clickhouse.Conn, testID string) ([]K6Segment, error) {
	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Querying K6 segments for test run %s", testID), "info")

	query := `
		SELECT
			'k6_login' AS source,
			segment_number,
			MIN(timestamp) AS segment_start,
			MAX(timestamp) AS segment_end
		FROM monitoring.k6_login
		WHERE test_id = ?
		GROUP BY segment_number
		ORDER BY segment_number ASC
	`

	rows, err := chClient.Query(ctx, query, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query K6 segments: %w", err)
	}
	defer rows.Close()

	var segments []K6Segment
	for rows.Next() {
		var segment K6Segment
		err := rows.Scan(&segment.Source, &segment.SegmentNumber, &segment.SegmentStart, &segment.SegmentEnd)
		if err != nil {
			logger.LogWarning("System", "K6Summarizer", fmt.Sprintf("Failed to scan K6 segment row: %v", err))
			continue
		}
		segments = append(segments, segment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading K6 segments rows: %w", err)
	}

	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Retrieved %d K6 segment data points", len(segments)), "info")
	return segments, nil
}

// QueryK6LoginMetrics queries ClickHouse for login metrics for a specific test run
func QueryK6LoginMetrics(ctx context.Context, chClient clickhouse.Conn, testID string) (*models.K6LoginMetrics, error) {
	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Querying K6 login metrics for test run %s", testID), "info")

	// First query for the main metrics
	mainQuery := `
		SELECT
		    -- Attempts
		    countIf(segment_number = 1) AS seg1_attempts,
		    countIf(segment_number = 2) AS seg2_attempts,
		    count() AS overall_attempts,

		    -- Success Rates
		    avgIf(success_rate, segment_number = 1) AS seg1_success_rate,
		    avgIf(success_rate, segment_number = 2) AS seg2_success_rate,
		    avg(success_rate) AS overall_success_rate,

		    -- Avg RT (only 200)
		    avgIf(avg_response_time, (segment_number = 1 AND status_code = 200)) AS seg1_avg_rt,
		    avgIf(avg_response_time, (segment_number = 2 AND status_code = 200)) AS seg2_avg_rt,
		    avgIf(avg_response_time, status_code = 200) AS overall_avg_rt,

		    -- Percentiles
		    quantileIf(0.95)(avg_response_time, (segment_number = 1 AND status_code = 200)) AS seg1_p95_rt,
		    quantileIf(0.95)(avg_response_time, (segment_number = 2 AND status_code = 200)) AS seg2_p95_rt,
		    quantileIf(0.95)(avg_response_time, status_code = 200) AS overall_p95_rt,

		    quantileIf(0.99)(avg_response_time, (segment_number = 1 AND status_code = 200)) AS seg1_p99_rt,
		    quantileIf(0.99)(avg_response_time, (segment_number = 2 AND status_code = 200)) AS seg2_p99_rt,
		    quantileIf(0.99)(avg_response_time, status_code = 200) AS overall_p99_rt,

		    -- High-level errors
		    sumIf(error_4xx, segment_number = 1) AS seg1_4xx,
		    sumIf(error_4xx, segment_number = 2) AS seg2_4xx,
		    sum(error_4xx) AS overall_4xx,

		    sumIf(error_5xx, segment_number = 1) AS seg1_5xx,
		    sumIf(error_5xx, segment_number = 2) AS seg2_5xx,
		    sum(error_5xx) AS overall_5xx,

		    -- Failure counts
		    countIf(status_code != 200 AND segment_number = 1) AS seg1_failures,
		    countIf(status_code != 200 AND segment_number = 2) AS seg2_failures,
		    countIf(status_code != 200) AS overall_failures

		FROM monitoring.k6_login
		WHERE test_id = ?
	`

	var metrics models.K6LoginMetrics

	err := chClient.QueryRow(ctx, mainQuery, testID).Scan(
		&metrics.Seg1Attempts,
		&metrics.Seg2Attempts,
		&metrics.OverallAttempts,
		&metrics.Seg1SuccessRate,
		&metrics.Seg2SuccessRate,
		&metrics.OverallSuccessRate,
		&metrics.Seg1AvgRT,
		&metrics.Seg2AvgRT,
		&metrics.OverallAvgRT,
		&metrics.Seg1P95RT,
		&metrics.Seg2P95RT,
		&metrics.OverallP95RT,
		&metrics.Seg1P99RT,
		&metrics.Seg2P99RT,
		&metrics.OverallP99RT,
		&metrics.Seg14xx,
		&metrics.Seg24xx,
		&metrics.Overall4xx,
		&metrics.Seg15xx,
		&metrics.Seg25xx,
		&metrics.Overall5xx,
		&metrics.Seg1Failures,
		&metrics.Seg2Failures,
		&metrics.OverallFailures,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query K6 login metrics: %w", err)
	}

	// Second query for status code failure map
	statusQuery := `
		SELECT
		    status_code,
		    count() AS failure_count
		FROM monitoring.k6_login
		WHERE test_id = ? AND status_code != 200
		GROUP BY status_code
		ORDER BY status_code
	`

	rows, err := chClient.Query(ctx, statusQuery, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status code failures: %w", err)
	}
	defer rows.Close()

	statusCodeMap := make(map[string]int)
	for rows.Next() {
		var statusCode uint16
		var count uint64
		err := rows.Scan(&statusCode, &count)
		if err != nil {
			logger.LogWarning("System", "K6Summarizer", fmt.Sprintf("Failed to scan status code row: %v", err))
			continue
		}
		statusCodeMap[fmt.Sprintf("%d", statusCode)] = int(count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading status code rows: %w", err)
	}

	metrics.StatusCodeFailureMap = statusCodeMap

	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Retrieved K6 login metrics for test run %s", testID), "info")
	return &metrics, nil
}

// QueryK6DashboardLoadTimes queries ClickHouse for dashboard load times for a specific test run
func QueryK6DashboardLoadTimes(ctx context.Context, chClient clickhouse.Conn, testID string) (*models.K6DashboardLoadTimes, error) {
	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Querying K6 dashboard load times for test run %s", testID), "info")

	query := `
		SELECT
		    dashboard_name,
		    segment_number AS segment,
		    time_range,

		    -- Total loads
		    count() AS total_loads,

		    -- Successful loads
		    countIf(dashboard_status = 200) AS success_loads,

		    -- Failed loads (any non-200)
		    countIf(dashboard_status != 200) AS failed_loads,

		    -- Error type counts
		    sum(dashboard_error_4xx) AS errors_4xx,
		    sum(dashboard_error_5xx) AS errors_5xx,
		    sum(dashboard_connection_error) AS errors_conn,

		    -- Success rate
		    avg(dashboard_success_rate) AS success_rate,

		    -- Avg load time (successful only)
		    avgIf(dashboard_avg_response_time, dashboard_status = 200) AS avg_load_ms,

		    -- P95 load time (successful only)
		    quantileIf(0.95)(dashboard_avg_response_time, dashboard_status = 200) AS p95_load_ms

		FROM monitoring.k6_results
		WHERE panel_id = 0
		  AND test_id = ?

		GROUP BY
		    dashboard_name,
		    segment_number,
		    time_range

		ORDER BY
		    dashboard_name,
		    segment_number,
		    time_range
	`

	rows, err := chClient.Query(ctx, query, testID)
	if err != nil {
		return nil, fmt.Errorf("failed to query K6 dashboard load times: %w", err)
	}
	defer rows.Close()

	var dashboardLoadTimes models.K6DashboardLoadTimes
	for rows.Next() {
		var entry models.K6DashboardLoadTimeEntry
		err := rows.Scan(
			&entry.DashboardName,
			&entry.Segment,
			&entry.TimeRange,
			&entry.TotalLoads,
			&entry.SuccessLoads,
			&entry.FailedLoads,
			&entry.Errors4xx,
			&entry.Errors5xx,
			&entry.ErrorsConn,
			&entry.SuccessRate,
			&entry.AvgLoadMs,
			&entry.P95LoadMs,
		)
		if err != nil {
			logger.LogWarning("System", "K6Summarizer", fmt.Sprintf("Failed to scan dashboard load time row: %v", err))
			continue
		}
		dashboardLoadTimes = append(dashboardLoadTimes, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading dashboard load time rows: %w", err)
	}

	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Retrieved %d dashboard load time entries for test run %s", len(dashboardLoadTimes), testID), "info")
	return &dashboardLoadTimes, nil
}

// ComputeK6Summary computes the summary statistics from raw segment data
func ComputeK6Summary(testID string, startTime, endTime time.Time, segments []K6Segment) *K6Summary {
	duration := endTime.Sub(startTime)
	durationMinutes := duration.Minutes()

	summary := &K6Summary{
		TestID:           testID,
		StartTime:        startTime,
		EndTime:          endTime,
		DurationMinutes:  durationMinutes,
		SegmentSummaries: []K6SegmentSummary{},
		GeneratedAt:      time.Now(),
	}

	// Convert segments to segment summaries
	for _, segment := range segments {
		segmentSummary := K6SegmentSummary{
			SegmentNumber: segment.SegmentNumber,
			SegmentStart:  segment.SegmentStart,
			SegmentEnd:    segment.SegmentEnd,
		}
		summary.SegmentSummaries = append(summary.SegmentSummaries, segmentSummary)
	}

	return summary
}

// GenerateK6SummaryForTestRun generates K6 summary for a single completed test run
func GenerateK6SummaryForTestRun(db *sql.DB, chClient clickhouse.Conn, testID string) error {
	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Starting K6 summary generation for test run %s", testID), "info")

	// Get test run details
	testRun, err := database.GetK6Run(testID)
	if err != nil {
		return fmt.Errorf("failed to get test run %s: %w", testID, err)
	}

	if testRun.Status != "completed" {
		return fmt.Errorf("test run %s is not completed (status: %s)", testID, testRun.Status)
	}

	if testRun.EndTime == nil {
		return fmt.Errorf("test run %s has no end time", testID)
	}

	// Query K6 segments for the test duration
	ctx := context.Background()
	segments, err := QueryK6Segments(ctx, chClient, testID)
	if err != nil {
		return fmt.Errorf("failed to query K6 segments: %w", err)
	}

	// Query K6 login metrics
	metrics, err := QueryK6LoginMetrics(ctx, chClient, testID)
	if err != nil {
		return fmt.Errorf("failed to query K6 login metrics: %w", err)
	}

	// Query K6 dashboard load times
	dashboardLoadTimes, err := QueryK6DashboardLoadTimes(ctx, chClient, testID)
	if err != nil {
		return fmt.Errorf("failed to query K6 dashboard load times: %w", err)
	}

	// Compute summary
	summary := ComputeK6Summary(testID, testRun.StartTime, *testRun.EndTime, segments)

	// Serialize summary to JSON
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Update test run with summary and metrics
	_, err = db.Exec(`
		UPDATE k6_runs
		SET k6_summary = ?, summarised = TRUE
		WHERE test_id = ?
	`, string(summaryJSON), testID)
	if err != nil {
		return fmt.Errorf("failed to update test run with summary: %w", err)
	}

	// Store metrics in database
	err = database.UpdateK6RunMetrics(testID, metrics)
	if err != nil {
		return fmt.Errorf("failed to update test run metrics: %w", err)
	}

	// Store dashboard load times in database
	err = database.UpdateK6RunDashboardLoadTimes(testID, dashboardLoadTimes)
	if err != nil {
		return fmt.Errorf("failed to update test run dashboard load times: %w", err)
	}

	logger.LogSuccess("System", "K6Summarizer", fmt.Sprintf("Generated K6 summary for test run %s", testID))
	return nil
}

// GenerateK6Summaries processes all completed K6 test runs that haven't been summarized yet
func GenerateK6Summaries(db *sql.DB, chClient clickhouse.Conn) error {
	logger.LogWithNode("System", "K6Summarizer", "Starting batch K6 summary generation", "info")

	// Ensure table columns exist
	err := EnsureK6RunsTableColumns(db)
	if err != nil {
		return fmt.Errorf("failed to ensure table columns: %w", err)
	}

	// Get all completed test runs without summaries
	rows, err := db.Query(`
		SELECT test_id
		FROM k6_runs
		WHERE status = 'completed'
		AND (summarised IS NULL OR summarised = FALSE)
		ORDER BY start_time DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query completed test runs: %w", err)
	}
	defer rows.Close()

	var testIDs []string
	for rows.Next() {
		var testID string
		err := rows.Scan(&testID)
		if err != nil {
			logger.LogWarning("System", "K6Summarizer", fmt.Sprintf("Failed to scan test ID: %v", err))
			continue
		}
		testIDs = append(testIDs, testID)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading test IDs: %w", err)
	}

	logger.LogWithNode("System", "K6Summarizer", fmt.Sprintf("Found %d test runs to process", len(testIDs)), "info")

	// Process each test run
	successCount := 0
	for _, testID := range testIDs {
		err := GenerateK6SummaryForTestRun(db, chClient, testID)
		if err != nil {
			logger.LogError("System", "K6Summarizer", fmt.Sprintf("Failed to generate summary for test run %s: %v", testID, err))
			continue
		}
		successCount++
	}

	logger.LogSuccess("System", "K6Summarizer", fmt.Sprintf("Successfully generated summaries for %d/%d test runs", successCount, len(testIDs)))
	return nil
}
