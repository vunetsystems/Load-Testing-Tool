package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/database"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// TopicConfig represents the structure of topics_tables.yaml
type TopicConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

type SourceConfig struct {
	Name             string      `yaml:"name"`
	InputTopic       []TopicItem `yaml:"inputTopic"`
	OutputTopic      []TopicItem `yaml:"outputTopic"`
	ClickHouseTables []string    `yaml:"clickhouseTables,omitempty"`
	Pipeline         []string    `yaml:"pipeline,omitempty"`
}

type TopicItem struct {
	Name string `yaml:"name"`
}

// getTopicConfig loads the topic configuration from YAML file
func getTopicConfig() (*TopicConfig, error) {
	configPath := "src/configs/topics_tables.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics config file: %v", err)
	}

	var config TopicConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse topics config: %v", err)
	}

	return &config, nil
}

// getTopicsForSource returns all input and output topics for a given O11y source
func getTopicsForSource(source string) ([]string, error) {
	config, err := getTopicConfig()
	if err != nil {
		return nil, err
	}

	// Normalize source name for matching
	sourceNormalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(source, "_", "-"), " ", "-"))

	for _, src := range config.Sources {
		srcNameNormalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(src.Name, "_", "-"), " ", "-"))

		if srcNameNormalized == sourceNormalized {
			var topics []string

			// Add input topics
			for _, topic := range src.InputTopic {
				topics = append(topics, topic.Name)
			}

			// Add output topics
			for _, topic := range src.OutputTopic {
				topics = append(topics, topic.Name)
			}

			logger.LogWithNode("System", "Kafka", fmt.Sprintf("Found %d topics for source '%s': %v", len(topics), source, topics), "debug")
			return topics, nil
		}
	}

	logger.LogWithNode("System", "Kafka", fmt.Sprintf("No topics found for source '%s' in config", source), "warning")
	return []string{}, nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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

	// If test_id is provided, filter topics based on the test run's O11y sources using config
	if testIDStr != "" {
		testRun, err := database.GetTestRun(testIDStr)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or test run not found: %v", err),
			})
			return
		}

		// Get topics for each O11y source from config file
		filteredTopics := []string{}
		logger.LogWithNode("System", "Kafka", fmt.Sprintf("Processing test run %s with O11y sources: %v", testIDStr, testRun.O11ySources), "info")

		for _, source := range testRun.O11ySources {
			sourceTopics, err := getTopicsForSource(source)
			if err != nil {
				logger.LogWarning("System", "Kafka", fmt.Sprintf("Failed to get topics for source '%s': %v", source, err))
				continue
			}

			// Add topics to filtered list (avoiding duplicates)
			for _, topic := range sourceTopics {
				if !contains(filteredTopics, topic) {
					filteredTopics = append(filteredTopics, topic)
					logger.LogWithNode("System", "Kafka", fmt.Sprintf("✓ Added topic: %s (from source: %s)", topic, source), "debug")
				}
			}

			logger.LogWithNode("System", "Kafka", fmt.Sprintf("Found %d topics for source '%s'", len(sourceTopics), source), "info")
		}

		logger.LogWithNode("System", "Kafka", fmt.Sprintf("Total filtered topics for test run: %d topics", len(filteredTopics)), "info")

		// If no matching topics found, return empty result
		if len(filteredTopics) == 0 {
			SendJSONResponse(w, http.StatusOK, APIResponse{
				Success: true,
				Message: "No Kafka topics found for the test run's O11y sources in configuration",
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
	testID := r.URL.Query().Get("test_id")

	var podData []clickhouse.PodMonitoringData
	var err error

	// If test_id is provided, get the time range from test_runs
	var timeRange *clickhouse.TimeRange
	if testID != "" {
		testRun, err := database.GetTestRun(testID)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or test run not found: %v", err),
			})
			return
		}

		// Set time range from test run
		timeRange = &clickhouse.TimeRange{
			From: testRun.StartTime,
		}
		if testRun.EndTime != nil {
			timeRange.To = *testRun.EndTime
		} else {
			// If test is still running, use current time
			timeRange.To = time.Now()
		}
	}

	if namespace == "" || namespace == "All Namespaces" {
		// Fetch from all namespaces when no namespace specified or "All Namespaces" selected
		podData, err = clickhouse.GetPodMonitoringDataAllNamespaces(r.Context(), timeRange)
	} else {
		// Fetch from specific namespace
		podData, err = clickhouse.GetPodMonitoringData(r.Context(), namespace, timeRange)
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
	testID := r.URL.Query().Get("test_id")

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

	// If test_id is provided, get the time range from test_runs
	var timeRange *clickhouse.TimeRange
	if testID != "" {
		testRun, err := database.GetTestRun(testID)
		if err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Invalid test_id or test run not found: %v", err),
			})
			return
		}

		// Set time range from test run
		timeRange = &clickhouse.TimeRange{
			From: testRun.StartTime,
		}
		if testRun.EndTime != nil {
			timeRange.To = *testRun.EndTime
		} else {
			// If test is still running, use current time
			timeRange.To = time.Now()
		}
	}

	trendData, err := clickhouse.GetPodTrendData(r.Context(), namespace, podName, hours, timeRange)
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
