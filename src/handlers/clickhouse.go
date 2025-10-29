package handlers

import (
	"fmt"
	"net/http"
	"time"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
)

func HandleAPIGetClickHouseMetrics(w http.ResponseWriter, r *http.Request) {
	// Get time range from query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var timeRange clickhouse.TimeRange
	if startStr == "" || endStr == "" {
		// Default to last 5 minutes if no time range provided
		timeRange.To = time.Now()
		timeRange.From = timeRange.To.Add(-5 * time.Minute)
	} else {
		var err error
		timeRange.From, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid start time format: %v", err),
			})
			return
		}
		timeRange.To, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid end time format: %v", err),
			})
			return
		}
	}

	metrics, err := clickhouse.CollectClickHouseMetrics(timeRange)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to collect ClickHouse metrics: %v", err),
		})
		return
	}

	// Log the metrics before sending
	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Sending metrics response: %+v", metrics), "info")

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse metrics retrieved successfully",
		Data:    metrics,
	})
}

// handleAPIClickHouseHealth handles GET /api/clickhouse/health
func HandleAPIClickHouseHealth(w http.ResponseWriter, r *http.Request) {
	healthData, err := clickhouse.GetClickHouseHealth()
	if err != nil {
		SendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: fmt.Sprintf("ClickHouse health check failed: %v", err),
			Data:    healthData,
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse is healthy",
		Data:    healthData,
	})
}


// HandleAPIGetKafkaTopicMetrics handles GET /api/clickhouse/kafka-topics
func HandleAPIGetKafkaTopicMetrics(w http.ResponseWriter, r *http.Request) {
	// Get time range from query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var timeRange clickhouse.TimeRange
	if startStr == "" || endStr == "" {
		// Default to last 5 minutes if no time range provided
		timeRange.To = time.Now()
		timeRange.From = timeRange.To.Add(-5 * time.Minute)
	} else {
		var err error
		timeRange.From, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid start time format: %v", err),
			})
			return
		}
		timeRange.To, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid end time format: %v", err),
			})
			return
		}
	}

	// Define Kafka topics to monitor
	topics := []string{
		"apache-metrics-input",
		"azure-firewall-input",
		"azure-redis-cache-input",
		"vuazure-storage-blob-input",
		"linux-monitor-input",
		"mongo-metrics-input",
		"mssql-telegraf",
	}

	kafkaMetrics, err := clickhouse.GetKafkaTopicMetrics(r.Context(), topics)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get Kafka topic metrics: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Kafka topic metrics retrieved successfully",
		Data:    kafkaMetrics,
	})
}

// HandleAPIGetPodMonitoring handles GET /api/clickhouse/pod-monitoring
func HandleAPIGetPodMonitoring(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")

	var podData []clickhouse.PodMonitoringData
	var err error

	if namespace == "" || namespace == "All Namespaces" {
		// Fetch from all namespaces when no namespace specified or "All Namespaces" selected
		podData, err = clickhouse.GetPodMonitoringDataAllNamespaces(r.Context())
	} else {
		// Fetch from specific namespace
		podData, err = clickhouse.GetPodMonitoringData(r.Context(), namespace)
	}

	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get pod monitoring data: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pod monitoring data retrieved successfully",
		Data:    podData,
	})
}
// HandleAPIGetK6DashboardResults handles GET /api/clickhouse/k6-dashboard-results
func HandleAPIGetK6DashboardResults(w http.ResponseWriter, r *http.Request) {
	k6DashboardResults, err := clickhouse.GetK6DashboardResults(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 dashboard results: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 dashboard results retrieved successfully",
		Data:    k6DashboardResults,
	})
}
// HandleAPIGetK6Results handles GET /api/clickhouse/k6-results
func HandleAPIGetK6Results(w http.ResponseWriter, r *http.Request) {
	dashboard := r.URL.Query().Get("dashboard")

	k6Results, err := clickhouse.GetK6Results(r.Context(), dashboard)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 results: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 results retrieved successfully",
		Data:    k6Results,
	})
}

