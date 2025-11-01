package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"

	"github.com/gorilla/mux"
)

// MetricsRequest represents a request for metrics with a time range
type MetricsRequest struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// getMetrics handles requests for metrics with a time range
func GetMetrics(w http.ResponseWriter, r *http.Request) {
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
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("invalid start time format: %v", err),
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
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
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("error collecting metrics: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    metrics,
	})
}

func HandleProxyMetrics(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for this endpoint
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Extract node name from URL path
	vars := mux.Vars(r)
	nodeName := vars["name"]

	if nodeName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node name is required"})
		return
	}

	// Get node configuration
	nodes := NodeManager.GetNodes()
	nodeConfig, exists := nodes[nodeName]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node not found"})
		return
	}

	// Note: We don't check binary status here because node_metrics_api should run independently
	// of finalvudatasim. System metrics should be available regardless of load testing binary status.

	// Make request to the metrics API server using the node's IP
	url := fmt.Sprintf("http://%s:8086/api/system/metrics", nodeConfig.Host)
	resp, err := http.Get(url)
	if err != nil {
		logger.Error().Err(err).Str("node", nodeName).Str("url", url).Msg("Failed to fetch metrics from metrics API server")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node metrics binary is not running on this VM"})
		return
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		logger.Error().Str("node", nodeName).Str("url", url).Int("status", resp.StatusCode).Msg("Metrics API server returned non-OK status")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node metrics binary is not running on this VM"})
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read metrics response")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read metrics response"})
		return
	}

	// Forward the response to the client
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
