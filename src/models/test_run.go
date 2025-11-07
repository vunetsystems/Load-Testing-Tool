// Package models defines data structures for the test run tracking process
package models

import "time"

// TestRun represents a test run record
type TestRun struct {
	TestID         string    `json:"test_id"`
	TargetEPS      int       `json:"target_eps"`
	StartTime      time.Time `json:"start_time"`
	StopTime       *time.Time `json:"stop_time,omitempty"`
	O11ySources    []string  `json:"o11y_sources"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	Status         string    `json:"status"` // 'running', 'completed', 'stopped', 'failed'
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