// Package models defines data structures for the test run tracking process
package models

import (
	"database/sql"
	"time"
)

// TestRun represents a test run record
type TestRun struct {
	TestID                        string          `json:"test_id"`
	TestName                      string          `json:"test_name,omitempty"`
	TargetEPS                     int             `json:"target_eps"`
	StartTime                     time.Time       `json:"start_time"`
	EndTime                       *time.Time      `json:"end_time,omitempty"`
	O11ySources                   []string        `json:"o11y_sources"`
	TimeoutSeconds         int             `json:"timeout_seconds"`
	Status                 string          `json:"status"` // 'running', 'completed', 'stopped', 'failed'
	MinInputMsgsPerSec     sql.NullFloat64 `json:"min_input_msgs_per_sec,omitempty"`
	AvgInputMsgsPerSec     sql.NullFloat64 `json:"avg_input_msgs_per_sec,omitempty"`
	MaxInputMsgsPerSec     sql.NullFloat64 `json:"max_input_msgs_per_sec,omitempty"`
	MinOutputMsgsPerSec    sql.NullFloat64 `json:"min_output_msgs_per_sec,omitempty"`
	AvgOutputMsgsPerSec    sql.NullFloat64 `json:"avg_output_msgs_per_sec,omitempty"`
	MaxOutputMsgsPerSec    sql.NullFloat64 `json:"max_output_msgs_per_sec,omitempty"`
	MinLag                 sql.NullFloat64 `json:"min_lag,omitempty"`
	AvgLag                 sql.NullFloat64 `json:"avg_lag,omitempty"`
	MaxLag                 sql.NullFloat64 `json:"max_lag,omitempty"`
	KafkaSummaryGenerated  sql.NullBool    `json:"kafka_summary_generated,omitempty"`
	ProcessRateSummary     string          `json:"process_rate_summary,omitempty"` // JSON summary of process rates per o11y source
	IngestionSummary       string          `json:"ingestion_summary,omitempty"`    // JSON summary of hyperscale ingestion table-wise EPS data
	PipelineInfo           string          `json:"pipeline_info,omitempty"`        // JSON mapping of o11y sources to pipeline details
	TraefikMemAllocated    sql.NullFloat64 `json:"traefik_mem_allocated,omitempty"`
	PodsCpu                string          `json:"pods_cpu,omitempty"`             // JSON string for pod CPU metrics
	PodsMemory             string          `json:"pods_memory,omitempty"`          // JSON string for pod memory metrics
	PodRestarts            string          `json:"pod_restarts,omitempty"`         // JSON string for pod restart metrics
}

// TestRunStartRequest represents the request payload for starting a test run
type TestRunStartRequest struct {
	TestName       string   `json:"test_name,omitempty"`
	TargetEPS      int      `json:"target_eps"`
	O11ySources    []string `json:"o11y_sources"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

// TestRunStartResponse represents the response for starting a test run
type TestRunStartResponse struct {
	Success bool    `json:"success"`
	Data    TestRun `json:"data"`
	Error   string  `json:"error,omitempty"`
}

// TestRunStopResponse represents the response for stopping a test run
type TestRunStopResponse struct {
	Success bool    `json:"success"`
	Data    TestRun `json:"data"`
	Error   string  `json:"error,omitempty"`
}

// TestRunListResponse represents the response for listing test runs
type TestRunListResponse struct {
	Success bool       `json:"success"`
	Data    []*TestRun `json:"data"`
	Error   string     `json:"error,omitempty"`
}

// TestRunDetailResponse represents the response for getting a specific test run
type TestRunDetailResponse struct {
	Success bool    `json:"success"`
	Data    TestRun `json:"data"`
	Error   string  `json:"error,omitempty"`
}

// NextTestIDResponse represents the response for getting the next test ID
type NextTestIDResponse struct {
	Success bool           `json:"success"`
	Data    NextTestIDData `json:"data"`
	Error   string         `json:"error,omitempty"`
}

// NextTestIDData contains the next test ID information
type NextTestIDData struct {
	NextTestID string `json:"next_test_id"`
}
