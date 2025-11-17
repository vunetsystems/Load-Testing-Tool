// Package models defines data structures for the K6 run tracking process
package models

import "time"

// K6LoginMetrics represents the login metrics computed from k6_login table
type K6LoginMetrics struct {
	// Attempts
	Seg1Attempts    uint64 `json:"seg1_attempts"`
	Seg2Attempts    uint64 `json:"seg2_attempts"`
	OverallAttempts uint64 `json:"overall_attempts"`

	// Success Rates
	Seg1SuccessRate    float64 `json:"seg1_success_rate"`
	Seg2SuccessRate    float64 `json:"seg2_success_rate"`
	OverallSuccessRate float64 `json:"overall_success_rate"`

	// Average Response Time (only 200 status)
	Seg1AvgRT    float64 `json:"seg1_avg_rt"`
	Seg2AvgRT    float64 `json:"seg2_avg_rt"`
	OverallAvgRT float64 `json:"overall_avg_rt"`

	// P95 Percentile
	Seg1P95RT    float64 `json:"seg1_p95_rt"`
	Seg2P95RT    float64 `json:"seg2_p95_rt"`
	OverallP95RT float64 `json:"overall_p95_rt"`

	// P99 Percentile
	Seg1P99RT    float64 `json:"seg1_p99_rt"`
	Seg2P99RT    float64 `json:"seg2_p99_rt"`
	OverallP99RT float64 `json:"overall_p99_rt"`

	// 4xx Errors
	Seg14xx    uint64 `json:"seg1_4xx"`
	Seg24xx    uint64 `json:"seg2_4xx"`
	Overall4xx uint64 `json:"overall_4xx"`

	// 5xx Errors
	Seg15xx    uint64 `json:"seg1_5xx"`
	Seg25xx    uint64 `json:"seg2_5xx"`
	Overall5xx uint64 `json:"overall_5xx"`

	// Failures (status_code != 200)
	Seg1Failures    uint64 `json:"seg1_failures"`
	Seg2Failures    uint64 `json:"seg2_failures"`
	OverallFailures uint64 `json:"overall_failures"`

	// Status Code Failure Breakdown
	StatusCodeFailureMap map[string]int `json:"status_code_failure_map"`
}

// K6DashboardLoadTimeEntry represents a single dashboard load time entry
type K6DashboardLoadTimeEntry struct {
	DashboardName string  `json:"dashboard_name"`
	Segment       uint8   `json:"segment"`
	TimeRange     string  `json:"time_range"`
	TotalLoads    uint64  `json:"total_loads"`
	SuccessLoads  uint64  `json:"success_loads"`
	FailedLoads   uint64  `json:"failed_loads"`
	Errors4xx     uint64  `json:"errors_4xx"`
	Errors5xx     uint64  `json:"errors_5xx"`
	ErrorsConn    uint64  `json:"errors_conn"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLoadMs     float64 `json:"avg_load_ms"`
	P95LoadMs     float64 `json:"p95_load_ms"`
}

// K6DashboardLoadTimes represents dashboard load times metrics
type K6DashboardLoadTimes []K6DashboardLoadTimeEntry

// K6PanelPerformanceEntry represents a single panel performance entry
type K6PanelPerformanceEntry struct {
	Dashboard          string  `json:"dashboard"`
	PanelNameID        string  `json:"panel_name_id"` // "panel_name (panel_id)"
	TotalAttempts      uint64  `json:"total_attempts"`
	FailedAttempts     uint64  `json:"failed_attempts"`
	AvgLoadMs          float64 `json:"avg_load_ms"`
	P95LoadMs          float64 `json:"p95_load_ms"`
	AvgContributionPct float64 `json:"avg_contribution_pct"`
	SuccessRate        float64 `json:"success_rate"`
	Errors4xx          uint64  `json:"errors_4xx"`
	Errors5xx          uint64  `json:"errors_5xx"`
	ErrorsConn         uint64  `json:"errors_conn"`
}

// K6PanelPerformance represents panel performance breakdown metrics
type K6PanelPerformance []K6PanelPerformanceEntry

// K6Run represents a K6 test run record
type K6Run struct {
	TestID                    string                `json:"test_id"`
	TestName                  string                `json:"test_name,omitempty"`
	StartTime                 time.Time             `json:"start_time"`
	EndTime                   *time.Time            `json:"end_time,omitempty"`
	TimeRange                 string                `json:"time_range"`
	Duration                  string                `json:"duration"`
	VUs                       int                   `json:"vus"`
	Iterations                int                   `json:"iterations"`
	Interval                  int                   `json:"interval"`
	O11ySources               []string              `json:"o11y_sources"`
	Status                    string                `json:"status"` // 'running', 'completed', 'stopped', 'failed'
	MetricsLogin              *K6LoginMetrics       `json:"Metrics_Login,omitempty"`
	OverallDashboardLoadTimes *K6DashboardLoadTimes `json:"Overall_Dashboard_Load_Times,omitempty"`
	PanelPerformanceBreakdown *K6PanelPerformance   `json:"Panel_Performance_Breakdown,omitempty"`
}

// K6RunStartRequest represents the request payload for starting a K6 test run
type K6RunStartRequest struct {
	TimeRange   string   `json:"timeRange"`
	Duration    string   `json:"duration"`
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
	Success bool             `json:"success"`
	Data    NextK6TestIDData `json:"data"`
	Error   string           `json:"error,omitempty"`
}

// NextK6TestIDData contains the next K6 test ID information
type NextK6TestIDData struct {
	NextTestID string `json:"next_test_id"`
}
