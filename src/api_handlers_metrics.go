package main

import (
	"fmt"
	"net/http"
	"time"
	"vuDataSim/src/clickhouse"
)

// MetricsRequest represents a request for metrics with a time range
type MetricsRequest struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// getMetrics handles requests for metrics with a time range
func getMetrics(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		// If no time range is provided, default to last 5 minutes
		endTime := time.Now()
		startTime := endTime.Add(-5 * time.Minute)
		timeRange := clickhouse.TimeRange{
			From: startTime,
			To:   endTime,
		}
		handleMetricsRequest(w, timeRange)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("invalid start time format: %v", err),
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("invalid end time format: %v", err),
		})
		return
	}

	timeRange := clickhouse.TimeRange{
		From: startTime,
		To:   endTime,
	}
	handleMetricsRequest(w, timeRange)
}

func handleMetricsRequest(w http.ResponseWriter, timeRange clickhouse.TimeRange) {
	metrics, err := clickhouse.CollectClickHouseMetrics(timeRange)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("error collecting metrics: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    metrics,
	})
}
