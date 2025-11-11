// Package models defines data structures for the test run tracking process
package models

import "time"

// TestRun represents a test run record
type TestRun struct {
	TestID                  string    `json:"test_id"`
	TestName                string    `json:"test_name,omitempty"`
	TargetEPS               int       `json:"target_eps"`
	StartTime               time.Time `json:"start_time"`
	EndTime                 *time.Time `json:"end_time,omitempty"`
	O11ySources             []string  `json:"o11y_sources"`
	TimeoutSeconds          int       `json:"timeout_seconds"`
	Status                  string    `json:"status"` // 'running', 'completed', 'stopped', 'failed'
	TotalInputMsgs          int64     `json:"total_input_msgs,omitempty"`
	TotalOutputMsgs         int64     `json:"total_output_msgs,omitempty"`
	AvgInputMsgsPerSec      float64   `json:"avg_input_msgs_per_sec,omitempty"`
	AvgOutputMsgsPerSec     float64   `json:"avg_output_msgs_per_sec,omitempty"`
	PeakInputMsgsPerSec     float64   `json:"peak_input_msgs_per_sec,omitempty"`
	PeakOutputMsgsPerSec    float64   `json:"peak_output_msgs_per_sec,omitempty"`
	MinInputMsgsPerSec      float64   `json:"min_input_msgs_per_sec,omitempty"`
	MinOutputMsgsPerSec     float64   `json:"min_output_msgs_per_sec,omitempty"`
	DataLossPct             float64   `json:"data_loss_pct,omitempty"`
	LagMsAvg                float64   `json:"lag_ms_avg,omitempty"`
	LagMsMax                float64   `json:"lag_ms_max,omitempty"`
	AnomalyDetected         bool      `json:"anomaly_detected,omitempty"`
	AnomalyScoreOverall     float64   `json:"anomaly_score_overall,omitempty"`
	AnomalyDetails          string    `json:"anomaly_details,omitempty"`
	O11ySourcesSummary      string    `json:"o11y_sources_summary,omitempty"`
	KafkaSummaryGenerated   bool      `json:"kafka_summary_generated,omitempty"`
}

// TestRunStartRequest represents the request payload for starting a test run
type TestRunStartRequest struct {
	TargetEPS      int      `json:"target_eps"`
	O11ySources    []string `json:"o11y_sources"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

// TestRunStartResponse represents the response for starting a test run
type TestRunStartResponse struct {
	Success bool     `json:"success"`
	Data    TestRun  `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// TestRunStopResponse represents the response for stopping a test run
type TestRunStopResponse struct {
	Success bool      `json:"success"`
	Data    TestRun   `json:"data"`
	Error   string    `json:"error,omitempty"`
}

// TestRunListResponse represents the response for listing test runs
type TestRunListResponse struct {
	Success bool       `json:"success"`
	Data    []*TestRun `json:"data"`
	Error   string     `json:"error,omitempty"`
}

// TestRunDetailResponse represents the response for getting a specific test run
type TestRunDetailResponse struct {
	Success bool     `json:"success"`
	Data    TestRun  `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// NextTestIDResponse represents the response for getting the next test ID
type NextTestIDResponse struct {
	Success     bool `json:"success"`
	Data        NextTestIDData `json:"data"`
	Error       string `json:"error,omitempty"`
}

// NextTestIDData contains the next test ID information
type NextTestIDData struct {
	NextTestID string `json:"next_test_id"`
}