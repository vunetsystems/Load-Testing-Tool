package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"
	"vuDataSim/src/o11y_source_manager"

	"github.com/gorilla/mux"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SimulationConfig represents simulation configuration
type SimulationConfig struct {
	Profile          string `json:"profile"`
	TargetEPS        int    `json:"targetEps"`
	TargetKafka      int    `json:"targetKafka"`
	TargetClickHouse int    `json:"targetClickHouse"`
}

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// API endpoint to get current state
func getDashboardData(w http.ResponseWriter, r *http.Request) {
	appState.mutex.RLock()
	defer appState.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to start simulation
func startSimulation(w http.ResponseWriter, r *http.Request) {
	var config SimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if appState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "Simulation is already running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate configuration
	if config.TargetEPS < 1 || config.TargetEPS > 100000 {
		response := APIResponse{
			Success: false,
			Message: "Target EPS must be between 1 and 100,000",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update state
	appState.IsSimulationRunning = true
	appState.CurrentProfile = config.Profile
	appState.TargetEPS = config.TargetEPS
	appState.TargetKafka = config.TargetKafka
	appState.TargetClickHouse = config.TargetClickHouse
	appState.StartTime = time.Now()

	response := APIResponse{
		Success: true,
		Message: "Simulation started successfully",
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go appState.broadcastUpdate()

	logger.LogWithNode("System", "Simulation", fmt.Sprintf("Simulation started with profile: %s, Target EPS: %d", config.Profile, config.TargetEPS), "info")
}

// API endpoint to stop simulation
func stopSimulation(w http.ResponseWriter, r *http.Request) {
	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if !appState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "No simulation is currently running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	appState.IsSimulationRunning = false

	response := APIResponse{
		Success: true,
		Message: "Simulation stopped successfully",
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go appState.broadcastUpdate()

	logger.LogWithNode("System", "Simulation", "Simulation stopped", "info")
}

// API endpoint to sync configuration
func syncConfiguration(w http.ResponseWriter, r *http.Request) {
	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	// In a real implementation, this would sync with external configuration sources
	response := APIResponse{
		Success: true,
		Message: "Configuration synced successfully",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   AppVersion,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.LogWithNode("System", "Config", "Configuration synced", "info")
}

// API endpoint to get logs
func getLogs(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	nodeFilter := r.URL.Query().Get("node")
	moduleFilter := r.URL.Query().Get("module")
	limit := r.URL.Query().Get("limit")

	// Parse limit
	limitNum := 50 // default
	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 && parsed <= 1000 {
			limitNum = parsed
		}
	}

	// Read logs from the log file
	logs := readLogsFromFile()

	// Apply filters
	filteredLogs := logs
	if nodeFilter != "" && nodeFilter != "All Nodes" {
		filtered := make([]map[string]interface{}, 0)
		for _, log := range logs {
			if log["node"] == nodeFilter {
				filtered = append(filtered, log)
			}
		}
		filteredLogs = filtered
	}

	if moduleFilter != "" && moduleFilter != "All Modules" {
		filtered := make([]map[string]interface{}, 0)
		for _, log := range filteredLogs {
			if log["module"] == moduleFilter {
				filtered = append(filtered, log)
			}
		}
		filteredLogs = filtered
	}

	// Apply limit
	if len(filteredLogs) > limitNum {
		filteredLogs = filteredLogs[:limitNum]
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"logs":  filteredLogs,
			"total": len(filteredLogs),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to update node metrics (for simulation)
func updateNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var metrics node_control.NodeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if node, exists := appState.NodeData[nodeID]; exists {
		node.EPS = metrics.EPS
		node.KafkaLoad = metrics.KafkaLoad
		node.CHLoad = metrics.CHLoad
		node.CPU = metrics.CPU
		node.Memory = metrics.Memory
		node.LastUpdate = time.Now()

		response := APIResponse{
			Success: true,
			Message: fmt.Sprintf("Node %s metrics updated successfully", nodeID),
			Data:    node,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// Broadcast update
		go appState.broadcastUpdate()
	} else {
		response := APIResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s not found", nodeID),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	}
}

// Health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	appState.mutex.RLock()
	defer appState.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"version":   AppVersion,
			"timestamp": time.Now(),
			"uptime":    time.Since(appState.StartTime).String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Node Management API Handlers

// handleAPINodes handles GET /api/nodes (list all nodes)
func handleAPINodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	nodes := nodeManager.GetNodes()
	nodeList := make([]map[string]interface{}, 0)

	for name, config := range nodes {
		status := "Disabled"
		if config.Enabled {
			status = "Enabled"
		}

		nodeList = append(nodeList, map[string]interface{}{
			"name":        name,
			"host":        config.Host,
			"user":        config.User,
			"status":      status,
			"description": config.Description,
			"binary_dir":  config.BinaryDir,
			"conf_dir":    config.ConfDir,
			"enabled":     config.Enabled,
		})
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    nodeList,
	})
}

// handleAPINodeActions handles POST/PUT/DELETE /api/nodes/{name}
func handleAPINodeActions(w http.ResponseWriter, r *http.Request) {
	// Extract node name from URL path
	vars := mux.Vars(r)
	nodeName := vars["name"]

	if nodeName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	switch r.Method {
	case http.MethodPost:
		handleCreateNode(w, r, nodeName)
	case http.MethodPut:
		handleUpdateNode(w, r, nodeName)
	case http.MethodDelete:
		handleDeleteNode(w, r, nodeName)
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// handleCreateNode handles POST /api/nodes/{name}
func handleCreateNode(w http.ResponseWriter, r *http.Request, nodeName string) {
	var nodeData struct {
		Host        string `json:"host"`
		User        string `json:"user"`
		KeyPath     string `json:"key_path"`
		ConfDir     string `json:"conf_dir"`
		BinaryDir   string `json:"binary_dir"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	err := nodeManager.AddNode(nodeName, nodeData.Host, nodeData.User, nodeData.KeyPath,
		nodeData.ConfDir, nodeData.BinaryDir, nodeData.Description, nodeData.Enabled)

	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s created successfully", nodeName),
	})
}

// handleUpdateNode handles PUT /api/nodes/{name}
func handleUpdateNode(w http.ResponseWriter, r *http.Request, nodeName string) {
	var nodeData struct {
		Enabled *bool `json:"enabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	if nodeData.Enabled != nil {
		if *nodeData.Enabled {
			err := nodeManager.EnableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		} else {
			err := nodeManager.DisableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		}
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s updated successfully", nodeName),
	})
}

// handleDeleteNode handles DELETE /api/nodes/{name}
func handleDeleteNode(w http.ResponseWriter, r *http.Request, nodeName string) {
	err := nodeManager.RemoveNode(nodeName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s deleted successfully", nodeName),
	})
}

// handleAPIClusterSettings handles GET/PUT /api/cluster-settings
func handleAPIClusterSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nodeManager.GetClusterSettings(),
		})
	case http.MethodPut:
		var settings node_control.ClusterSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		err := nodeManager.UpdateClusterSettings(settings)
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Cluster settings updated successfully",
		})
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// Binary Control API Handlers

// handleAPIGetAllBinaryStatus handles GET /api/binary/status
func handleAPIGetAllBinaryStatus(w http.ResponseWriter, r *http.Request) {
	response, err := binaryControl.GetAllBinaryStatuses()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get binary statuses: %v", err),
		})
		return
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}
	sendJSONResponse(w, http.StatusOK, apiResponse)
}

// handleAPIGetBinaryStatus handles GET /api/binary/status/{node}
func handleAPIGetBinaryStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	status, err := binaryControl.GetBinaryStatus(nodeName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get binary status for node %s: %v", nodeName, err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// handleAPIStartBinary handles POST /api/binary/start/{node}
func handleAPIStartBinary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	// Parse timeout from query parameters (default: 30 seconds)
	timeout := 30
	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	response, err := binaryControl.StartBinary(nodeName, timeout)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start binary on node %s: %v", nodeName, err),
		})
		return
	}

	statusCode := http.StatusOK
	if response.Data != nil {
		if data, ok := response.Data.(map[string]interface{}); ok {
			if _, hasWarning := data["warning"]; hasWarning {
				statusCode = http.StatusAccepted // 202 for warnings
			}
		}
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}
	sendJSONResponse(w, statusCode, apiResponse)
}

// handleAPIStopBinary handles POST /api/binary/stop/{node}
func handleAPIStopBinary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	// Parse timeout from query parameters (default: 30 seconds)
	timeout := 30
	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	response, err := binaryControl.StopBinary(nodeName, timeout)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop binary on node %s: %v", nodeName, err),
		})
		return
	}

	statusCode := http.StatusOK
	if response.Data != nil {
		if data, ok := response.Data.(map[string]interface{}); ok {
			if _, hasWarning := data["warning"]; hasWarning {
				statusCode = http.StatusAccepted // 202 for warnings
			}
		}
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}
	sendJSONResponse(w, statusCode, apiResponse)
}

// SSH Status Types
type SSHStatus struct {
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"` // "connected", "disconnected", "error", "disabled"
	Message     string `json:"message"`
	LastChecked string `json:"lastChecked"`
}

// handleAPIGetSSHStatus handles GET /api/ssh/status
func handleAPIGetSSHStatus(w http.ResponseWriter, r *http.Request) {
	enabledNodes := nodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "No enabled nodes found",
			Data:    []SSHStatus{},
		})
		return
	}

	var allStatuses []SSHStatus
	for nodeName, nodeConfig := range enabledNodes {
		status := checkSSHConnectivity(nodeName, nodeConfig)
		allStatuses = append(allStatuses, status)
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved SSH status for %d nodes", len(allStatuses)),
		Data:    allStatuses,
	})
}

// ClickHouse API Handlers

// handleAPIGetClickHouseMetrics handles GET /api/clickhouse/metrics
func handleAPIGetClickHouseMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := clickhouse.CollectClickHouseMetrics()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to collect ClickHouse metrics: %v", err),
		})
		return
	}

	// Log the metrics before sending
	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Sending metrics response: %+v", metrics), "info")

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse metrics retrieved successfully",
		Data:    metrics,
	})
}

// handleAPIClickHouseHealth handles GET /api/clickhouse/health
func handleAPIClickHouseHealth(w http.ResponseWriter, r *http.Request) {
	healthData, err := clickhouse.GetClickHouseHealth()
	if err != nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: fmt.Sprintf("ClickHouse health check failed: %v", err),
			Data:    healthData,
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse is healthy",
		Data:    healthData,
	})
}

// O11y Source Manager API Handlers

// handleAPIGetO11ySources handles GET /api/o11y/sources
func handleAPIGetO11ySources(w http.ResponseWriter, r *http.Request) {
	// Initialize o11y manager if not already done
	if len(o11yManager.GetMaxEPSConfig()) == 0 {
		err := o11yManager.LoadMaxEPSConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to load max EPS config: %v", err),
			})
			return
		}
	}

	// Also load main config to ensure it's up to date
	err := o11yManager.LoadMainConfig()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load main config: %v", err),
		})
		return
	}

	sources := o11yManager.GetAvailableSources()
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    sources,
		Message: fmt.Sprintf("Retrieved %d available o11y sources", len(sources)),
	})
}

// handleAPIGetEnabledO11ySources handles GET /api/o11y/sources/enabled
func handleAPIGetEnabledO11ySources(w http.ResponseWriter, r *http.Request) {
	// Ensure o11y manager is initialized
	// Available sources are loaded dynamically when needed

	sources := o11yManager.GetEnabledSources()
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    sources,
	})
}

// handleAPIGetO11ySourceDetails handles GET /api/o11y/sources/{source}
func handleAPIGetO11ySourceDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	details, err := o11yManager.GetSourceDetails(sourceName)
	if err != nil {
		sendJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Source not found: %s", sourceName),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    details,
	})
}

// handleAPIDistributeEPS handles POST /api/o11y/eps/distribute
func handleAPIDistributeEPS(w http.ResponseWriter, r *http.Request) {
	var request o11y_source_manager.EPSDistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	response, err := o11yManager.DistributeEPS(request)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusBadRequest
	}

	sendJSONResponse(w, statusCode, APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	})
}

// handleAPIGetCurrentEPS handles GET /api/o11y/eps/current
func handleAPIGetCurrentEPS(w http.ResponseWriter, r *http.Request) {
	// Available sources are loaded dynamically when needed

	currentEPS := o11yManager.CalculateCurrentEPS()
	breakdown := o11yManager.GetSourceEPSBreakdown()

	data := map[string]interface{}{
		"totalEPS":  currentEPS,
		"breakdown": breakdown,
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// handleAPIEnableO11ySource handles POST /api/o11y/sources/{source}/enable
func handleAPIEnableO11ySource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	err := o11yManager.EnableSource(sourceName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Source %s enabled successfully", sourceName),
	})
}

// handleAPIDisableO11ySource handles POST /api/o11y/sources/{source}/disable
func handleAPIDisableO11ySource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	err := o11yManager.DisableSource(sourceName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Source %s disabled successfully", sourceName),
	})
}

// handleAPIGetMaxEPSConfig handles GET /api/o11y/max-eps
func handleAPIGetMaxEPSConfig(w http.ResponseWriter, r *http.Request) {
	// Ensure o11y manager is initialized
	if len(o11yManager.GetMaxEPSConfig()) == 0 {
		err := o11yManager.LoadMaxEPSConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to load max EPS config: %v", err),
			})
			return
		}
	}

	maxEPSConfig := o11yManager.GetMaxEPSConfig()
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    maxEPSConfig,
	})
}

// handleAPIDistributeConfD handles POST /api/o11y/confd/distribute
func handleAPIDistributeConfD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	// Distribute conf.d to all enabled nodes
	response, err := o11yManager.DistributeConfD()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to distribute conf.d: %v", err),
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusPartialContent // 206 for partial success
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}

	// Add distribution details to response data
	if apiResponse.Data == nil {
		apiResponse.Data = make(map[string]interface{})
	}
	apiResponse.Data.(map[string]interface{})["distribution"] = response.Distribution

	sendJSONResponse(w, statusCode, apiResponse)
}

// Log parsing functions

// readLogsFromFile reads and parses logs from the zerolog file
func readLogsFromFile() []map[string]interface{} {
	logFilePath := "logs/vuDataSim.log"
	file, err := os.Open(logFilePath)
	if err != nil {
		// If log file doesn't exist yet, return empty slice
		return []map[string]interface{}{}
	}
	defer file.Close()

	var logs []map[string]interface{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue // Skip malformed lines
		}

		// Convert zerolog format to frontend format
		frontendLog := map[string]interface{}{
			"timestamp": parseZerologTimestamp(logEntry["time"]),
			"node":      getLogField(logEntry, "node", "System"),
			"module":    getLogField(logEntry, "module", "System"),
			"message":   getLogField(logEntry, "message", ""),
			"type":      getLogType(logEntry),
		}

		logs = append(logs, frontendLog)
	}

	// Reverse to show newest first
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs
}

// parseZerologTimestamp parses the timestamp from zerolog format
func parseZerologTimestamp(timeInterface interface{}) string {
	if timeStr, ok := timeInterface.(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return time.Now().Format("2006-01-02 15:04:05")
}

// getLogField safely extracts a field from the log entry
func getLogField(entry map[string]interface{}, field, defaultValue string) string {
	if value, ok := entry[field]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getLogType determines the log type based on zerolog level
func getLogType(entry map[string]interface{}) string {
	if level, ok := entry["level"]; ok {
		switch level {
		case "error":
			return "error"
		case "warn":
			return "warning"
		case "info":
			return "info"
		case "debug":
			return "info"
		}
	}

	// Check for type field if set by our logging functions
	if logType, ok := entry["type"]; ok {
		if str, ok := logType.(string); ok {
			return str
		}
	}

	return "info"
}
