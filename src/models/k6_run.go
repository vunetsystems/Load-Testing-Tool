// Package models defines data structures for the K6 run tracking process
package models

import "time"

// K6Run represents a K6 test run record
type K6Run struct {
	TestID         string    `json:"test_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	TimeRange      string    `json:"time_range"`
	VUs            int       `json:"vus"`
	Iterations     int       `json:"iterations"`
	Interval       int       `json:"interval"`
	O11ySources    []string  `json:"o11y_sources"`
	Status         string    `json:"status"` // 'running', 'completed', 'stopped', 'failed'
}

// K6RunStartRequest represents the request payload for starting a K6 test run
type K6RunStartRequest struct {
	TimeRange   string   `json:"timeRange"`
	VUs         int      `json:"vus"`
	Iterations  int      `json:"iterations"`
	Interval    int      `json:"interval"`
	O11ySources []string `json:"o11y_sources"`
}

// K6RunStartResponse represents the response for starting a K6 test run
type K6RunStartResponse struct {
	Success bool   `json:"success"`
	Data    K6Run  `json:"data"`
	Error   string `json:"error,omitempty"`
}

// K6RunStopResponse represents the response for stopping a K6 test run
type K6RunStopResponse struct {
	Success bool   `json:"success"`
	Data    K6Run  `json:"data"`
	Error   string `json:"error,omitempty"`
}

// K6RunListResponse represents the response for listing K6 test runs
type K6RunListResponse struct {
	Success bool     `json:"success"`
	Data    []*K6Run `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// K6RunDetailResponse represents the response for getting a specific K6 test run
type K6RunDetailResponse struct {
	Success bool   `json:"success"`
	Data    K6Run  `json:"data"`
	Error   string `json:"error,omitempty"`
}

// NextK6TestIDResponse represents the response for getting the next K6 test ID
type NextK6TestIDResponse struct {
	Success     bool             `json:"success"`
	Data        NextK6TestIDData `json:"data"`
	Error       string           `json:"error,omitempty"`
}

// NextK6TestIDData contains the next K6 test ID information
type NextK6TestIDData struct {
	NextTestID string `json:"next_test_id"`
}