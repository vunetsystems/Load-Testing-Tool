package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/database"
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
	testIDStr := r.URL.Query().Get("test_id")

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

	// Define O11y source mappings (available for both real-time and test-filtered modes)
	o11yToTopics := map[string][]string{
		"MongoDB": {
			"mongo-metrics-input",           // input
			"mongo-metrics", "mongo-top-stats", "mongo-shard-stats", "mongo-col-stats", "mongo-db-stats", // outputs
		},
		"Mssql": {  // ✅ Fixed: matches categories.yaml "Mssql"
			"mssql-telegraf",               // input
			"mssql-memory-clerks", "mssql-database-io", "mssql-net-response", "mssql-hadr-replica",
			"mssql-schedulers", "mssql-requests", "mssql-server-properties", "mssql-performance",
			"mssql-hadr-dbreplica", "mssql-session", "mssql-telegraf-health", "mssql-volume-space",
			"mssql-cpu", "mssql-waitstats", "mssql-cluster", "mssql-recentbackup", // outputs
		},
		"Apache": {
			"apache-metrics-input", "apache-logs-input", // inputs
			"apache-logs", "apache-metrics",             // outputs
		},
		"LinuxMonitor": {  // ✅ Fixed: matches categories.yaml "LinuxMonitor"
			"linux-monitor-input", // input
			"linux-monitor-additional-metrics", "linux-monitor-process-metrics",
			"linux-monitor-resource-metrics", "linux-monitor-storage-metrics", // outputs
		},
		"Azure_Firewall": {
			"azure-firewall-input", // input
			// Add output topics if they exist
		},
		"Azure_Redis_Cache": {
			"azure-redis-cache-input", // input
			// Add output topics if they exist
		},
		"AzureStorageBlob": {
			"vuazure-storage-blob-input", // input
			// Add output topics if they exist
		},
	}

	// Map database names to categories.yaml names for compatibility
	dbNameToCategory := map[string]string{
		"Linux Monitor": "LinuxMonitor",
		"Azure Firewall": "Azure_Firewall",
		"Azure Redis Cache": "Azure_Redis_Cache",
		"Azure Storage Blob": "AzureStorageBlob",
	}

	// Define Kafka topics to monitor - default to all topics (input + output)
	topics := []string{
		// Input topics
		"apache-metrics-input", "apache-logs-input",
		"azure-firewall-input", "azure-redis-cache-input", "vuazure-storage-blob-input",
		"linux-monitor-input", "mongo-metrics-input", "mssql-telegraf",

		// Output topics
		"apache-logs", "apache-metrics",
		"mongo-metrics", "mongo-top-stats", "mongo-shard-stats", "mongo-col-stats", "mongo-db-stats",
		"mssql-memory-clerks", "mssql-database-io", "mssql-net-response", "mssql-hadr-replica",
		"mssql-schedulers", "mssql-requests", "mssql-server-properties", "mssql-performance",
		"mssql-hadr-dbreplica", "mssql-session", "mssql-telegraf-health", "mssql-volume-space",
		"mssql-cpu", "mssql-waitstats", "mssql-cluster", "mssql-recentbackup",
		"linux-monitor-additional-metrics", "linux-monitor-process-metrics",
		"linux-monitor-resource-metrics", "linux-monitor-storage-metrics",
	}

	// If test_id is provided, filter topics based on the test run's O11y sources
	if testIDStr != "" {
		testRun, err := database.GetTestRun(testIDStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or test run not found: %v", err),
			})
			return
		}

		// Use test run's time range if no custom time range provided
		if startStr == "" || endStr == "" {
			if testRun.EndTime != nil {
				timeRange.From = testRun.StartTime
				timeRange.To = *testRun.EndTime
			} else {
				// Test is still running, use start time to now
				timeRange.From = testRun.StartTime
				timeRange.To = time.Now()
			}
		}

		// Filter topics to only include those enabled for this test run
		filteredTopics := []string{}
		for _, source := range testRun.O11ySources {
			// Map database name to category name if needed
			categoryName := source
			if mapped, exists := dbNameToCategory[source]; exists {
				categoryName = mapped
			}

			if topicList, exists := o11yToTopics[categoryName]; exists {
				filteredTopics = append(filteredTopics, topicList...)
			}
		}

		// If no matching topics found, return empty result
		if len(filteredTopics) == 0 {
			SendJSONResponse(w, http.StatusOK, APIResponse{
				Success: true,
				Message: "No Kafka topics found for the selected test run's O11y sources",
				Data:    []clickhouse.KafkaTopicMetric{},
			})
			return
		}

		topics = filteredTopics
	}

	kafkaMetrics, err := clickhouse.GetKafkaTopicMetrics(r.Context(), topics, timeRange)
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
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	testIDStr := r.URL.Query().Get("test_id")

	var timeRange clickhouse.TimeRange
	if startStr == "" || endStr == "" {
		// Default to last 24 hours if no time range provided
		timeRange.To = time.Now()
		timeRange.From = timeRange.To.Add(-24 * time.Hour)
	} else {
		var err error
		timeRange.From, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid start time format: %v", err),
			})
			return
		}
		timeRange.To, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid end time format: %v", err),
			})
			return
		}
	}

	// If test_id is provided, get time range from database
	if testIDStr != "" {
		k6Run, err := database.GetK6Run(testIDStr)
		if err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or K6 run not found: %v", err),
			})
			return
		}

		// Use K6 run's time range if no custom time range provided
		if startStr == "" || endStr == "" {
			if k6Run.EndTime != nil {
				timeRange.From = k6Run.StartTime
				timeRange.To = *k6Run.EndTime
			} else {
				// Test is still running, use start time to now
				timeRange.From = k6Run.StartTime
				timeRange.To = time.Now()
			}
		}
	}

	k6LoginResults, err := clickhouse.GetK6LoginResults(r.Context(), timeRange)
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
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	testIDStr := r.URL.Query().Get("test_id")

	var timeRange clickhouse.TimeRange
	if startStr == "" || endStr == "" {
		// Default to last 24 hours if no time range provided
		timeRange.To = time.Now()
		timeRange.From = timeRange.To.Add(-24 * time.Hour)
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

	// If test_id is provided, get time range from database
	if testIDStr != "" {
		k6Run, err := database.GetK6Run(testIDStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or K6 run not found: %v", err),
			})
			return
		}

		// Use K6 run's time range if no custom time range provided
		if startStr == "" || endStr == "" {
			if k6Run.EndTime != nil {
				timeRange.From = k6Run.StartTime
				timeRange.To = *k6Run.EndTime
			} else {
				// Test is still running, use start time to now
				timeRange.From = k6Run.StartTime
				timeRange.To = time.Now()
			}
		}
	}

	k6DashboardResults, err := clickhouse.GetK6DashboardResults(r.Context(), timeRange)
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
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	testIDStr := r.URL.Query().Get("test_id")

	var timeRange clickhouse.TimeRange
	if startStr == "" || endStr == "" {
		// Default to last 24 hours if no time range provided
		timeRange.To = time.Now()
		timeRange.From = timeRange.To.Add(-24 * time.Hour)
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

	// If test_id is provided, get time range from database
	if testIDStr != "" {
		k6Run, err := database.GetK6Run(testIDStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or K6 run not found: %v", err),
			})
			return
		}

		// Use K6 run's time range if no custom time range provided
		if startStr == "" || endStr == "" {
			if k6Run.EndTime != nil {
				timeRange.From = k6Run.StartTime
				timeRange.To = *k6Run.EndTime
			} else {
				// Test is still running, use start time to now
				timeRange.From = k6Run.StartTime
				timeRange.To = time.Now()
			}
		}
	}

	// Debug: return test data
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 results retrieved successfully",
		Data: "test data",
	})
	return

	k6Results, err := clickhouse.GetK6Results(r.Context(), dashboard, timeRange)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get k6 results: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
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

