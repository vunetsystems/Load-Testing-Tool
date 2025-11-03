package handlers

import (
	"fmt"
	"net/http"
	"strconv"
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
// HandleAPIGetK6SuccessRate handles GET /api/clickhouse/k6-success-rate
func HandleAPIGetK6SuccessRate(w http.ResponseWriter, r *http.Request) {
	successRate, err := clickhouse.GetK6SuccessRate(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 success rate: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 success rate retrieved successfully",
		Data:    successRate,
	})
}
// HandleAPIGetPodLogs handles GET /api/clickhouse/pod-logs
func HandleAPIGetPodLogs(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "vsmaps" // default namespace
	}

	logs, err := clickhouse.GetPodLogs(r.Context(), namespace)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get pod logs: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pod logs retrieved successfully",
		Data:    logs,
	})
}

// HandleAPIGetPodEvents handles GET /api/clickhouse/pod-events
func HandleAPIGetPodEvents(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod")

	if namespace == "" {
		namespace = "vsmaps" // default namespace
	}

	var events []clickhouse.PodEventEntry
	var err error

	if podName != "" {
		// Get events for specific pod
		events, err = clickhouse.GetPodEventsForPod(r.Context(), namespace, podName)
	} else {
		// Get events for all pods in namespace
		events, err = clickhouse.GetPodEvents(r.Context(), namespace)
	}

	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get pod events: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pod events retrieved successfully",
		Data:    events,
	})
}
// HandleAPIGetK6MaxVus handles GET /api/clickhouse/k6-max-vus
func HandleAPIGetK6MaxVus(w http.ResponseWriter, r *http.Request) {
	maxVusResult, err := clickhouse.GetK6MaxVus(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 max vus: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 max vus retrieved successfully",
		Data:    maxVusResult,
	})
}
// HandleAPIGetK6LoginResults handles GET /api/clickhouse/k6-login-results
func HandleAPIGetK6LoginResults(w http.ResponseWriter, r *http.Request) {
	k6LoginResults, err := clickhouse.GetK6LoginResults(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 login results: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 login results retrieved successfully",
		Data:    k6LoginResults,
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
// HandleAPIGetKafkaPodMemory handles GET /api/clickhouse/kafka-pod-memory
func HandleAPIGetKafkaPodMemory(w http.ResponseWriter, r *http.Request) {
	data, err := clickhouse.GetKafkaPodMemoryData(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch Kafka pod memory data: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Kafka pod memory data retrieved successfully",
		Data:    data,
	})
}
// HandleAPIGetKafkaNetwork handles GET /api/clickhouse/kafka-network
func HandleAPIGetKafkaNetwork(w http.ResponseWriter, r *http.Request) {
	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	limit := 5 // Default to 5 records as requested

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	data, err := clickhouse.GetKafkaNetworkData(r.Context())
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch Kafka network data: %v", err),
		})
		return
	}

	// Apply pagination
	totalRecords := len(data)
	startIndex := (page - 1) * limit
	endIndex := startIndex + limit

	if startIndex >= totalRecords {
		// Return empty data for out of bounds pages
		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Kafka network data retrieved successfully",
			Data: map[string]interface{}{
				"data":         []clickhouse.KafkaNetworkData{},
				"page":         page,
				"limit":        limit,
				"totalRecords": totalRecords,
				"totalPages":   (totalRecords + limit - 1) / limit,
			},
		})
		return
	}

	if endIndex > totalRecords {
		endIndex = totalRecords
	}

	paginatedData := data[startIndex:endIndex]

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Kafka network data retrieved successfully",
		Data: map[string]interface{}{
			"data":         paginatedData,
			"page":         page,
			"limit":        limit,
			"totalRecords": totalRecords,
			"totalPages":   (totalRecords + limit - 1) / limit,
		},
	})
}

// HandleAPIGetPodTrendData handles GET /api/clickhouse/pod-trend
func HandleAPIGetPodTrendData(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod")
	hoursStr := r.URL.Query().Get("hours")

	if namespace == "" {
		namespace = "vsmaps" // default namespace
	}

	if podName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Pod name is required",
		})
		return
	}

	hours := 24 // default to 24 hours
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // max 1 week
			hours = h
		}
	}

	trendData, err := clickhouse.GetPodTrendData(r.Context(), namespace, podName, hours)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get pod trend data: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pod trend data retrieved successfully",
		Data:    trendData,
	})
}

