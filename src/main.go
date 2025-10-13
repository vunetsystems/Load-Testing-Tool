package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"vuDataSim/src/bin_control"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"

	"vuDataSim/src/o11y_source_manager"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Application version and configuration
const (
	AppVersion = "1.0.0"
	StaticDir  = "./static"
	Port       = ":3000"
)

// ClickHouse configuration (moved to clickhouse package)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin in development
	},
}

// Global application state
type AppState struct {
	IsSimulationRunning bool                          `json:"isSimulationRunning"`
	CurrentProfile      string                        `json:"currentProfile"`
	TargetEPS           int                           `json:"targetEps"`
	TargetKafka         int                           `json:"targetKafka"`
	TargetClickHouse    int                           `json:"targetClickHouse"`
	StartTime           time.Time                     `json:"startTime"`
	NodeData            map[string]*NodeMetrics       `json:"nodeData"`
	ClickHouseMetrics   *clickhouse.ClickHouseMetrics `json:"clickHouseMetrics,omitempty"`
	mutex               sync.RWMutex
	clients             map[*websocket.Conn]bool
	broadcast           chan []byte
}

// NodeMetrics represents metrics for a single node
type NodeMetrics struct {
	NodeID      string    `json:"nodeId"`
	Status      string    `json:"status"`
	EPS         int       `json:"eps"`
	KafkaLoad   int       `json:"kafkaLoad"`
	CHLoad      int       `json:"chLoad"`
	CPU         float64   `json:"cpu"`         // CPU usage percentage (0-100)
	Memory      float64   `json:"memory"`      // Memory usage percentage (0-100)
	TotalCPU    float64   `json:"totalCpu"`    // Total CPU cores available
	TotalMemory float64   `json:"totalMemory"` // Total memory in GB available
	LastUpdate  time.Time `json:"lastUpdate"`
}

// HTTPMetricsResponse represents the response from node metrics API
type HTTPMetricsResponse struct {
	NodeID    string    `json:"nodeId"`
	Timestamp time.Time `json:"timestamp"`
	System    struct {
		CPU    HTTPNodeCPUInfo    `json:"cpu"`
		Memory HTTPNodeMemoryInfo `json:"memory"`
		Uptime int64              `json:"uptime_seconds"`
	} `json:"system"`
}

// HTTPNodeCPUInfo represents CPU metrics from HTTP API
type HTTPNodeCPUInfo struct {
	UsedPercent float64 `json:"used_percent"`
	Cores       int     `json:"cores"`
	Load1M      float64 `json:"load_1m"`
}

// HTTPNodeMemoryInfo represents memory metrics from HTTP API
type HTTPNodeMemoryInfo struct {
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	TotalGB     float64 `json:"total_gb"`
	UsedPercent float64 `json:"used_percent"`
}

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

// ClickHouse data models (moved to clickhouse package)

// Global application state instance
var appState = &AppState{
	IsSimulationRunning: false,
	CurrentProfile:      "medium",
	TargetEPS:           10000,
	TargetKafka:         5000,
	TargetClickHouse:    2000,
	NodeData:            make(map[string]*NodeMetrics),
	clients:             make(map[*websocket.Conn]bool),
	broadcast:           make(chan []byte, 256),
}

// Node Management Structures and Types

// Node control types and instance (moved to node_control package)

// Node control functions (moved to node_control package)

// Node control functions (moved to node_control package)

// Global node manager instance
var nodeManager *node_control.NodeManager

// ClickHouse client and configuration
var clickHouseClient *clickhouse.ClickHouseClient
var clickHouseConfig = clickhouse.ClickHouseConfig{
	Host:     "10.32.3.50", // ClickHouse service ClusterIP
	Port:     9000,
	Database: "monitoring",
	Username: "monitoring_read",
	Password: "StrongP@assword123",
}

// Binary control types and instance (moved to bin_control package)

// Global binary control instance
var binaryControl *bin_control.BinaryControl

// ClickHouse connection and query functions

// initClickHouse initializes the ClickHouse client connection
func initClickHouse() error {
	client, err := clickhouse.NewClickHouseClient(clickHouseConfig)
	if err != nil {
		return err
	}

	clickHouseClient = client
	logger.LogSuccess("System", "ClickHouse", "ClickHouse client initialized successfully")
	return nil
}

// collectClickHouseMetrics collects all metrics from ClickHouse
func collectClickHouseMetrics() (*clickhouse.ClickHouseMetrics, error) {
	if clickHouseClient == nil {
		return nil, fmt.Errorf("ClickHouse client not initialized")
	}

	return clickHouseClient.CollectMetrics()
}

// Global o11y source manager instance
var o11yManager = o11y_source_manager.NewO11ySourceManager()

// Initialize node data from nodes.yaml configuration
func init() {
	log.Printf("Initializing node data from configuration...")

	// Load nodes from configuration
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load nodes config")
		logger.Warn().Msg("Using default node configuration")
		initializeDefaultNodes()
		// Initialize binary control with default config
		binaryControl = bin_control.NewBinaryControl()
		return
	}

	logger.Info().Interface("nodes", nodeManager.GetNodes()).Msg("Loaded nodes from config")

	// Initialize binary control with loaded config
	binaryControl = bin_control.NewBinaryControl()
	err = binaryControl.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config for binary control: %v", err)
	}

	// Initialize node data using real node names from config
	nodeIndex := 0
	for nodeName, nodeConfig := range nodeManager.GetNodes() {
		nodeID := nodeName
		log.Printf("Initializing node: %s, enabled: %v", nodeID, nodeConfig.Enabled)

		// Detect real system resources for this node
		totalMemory := 8.0 // Default fallback
		totalCPU := 4.0    // Default fallback

		// Use HTTP-based metrics for real values if node is enabled
		if nodeConfig.Enabled {
			logger.LogSuccess(nodeID, "System", "Node enabled - will use HTTP metrics collection")

			// Try to get initial metrics via HTTP
			metrics, err := pollNodeMetrics(nodeConfig)
			if err != nil {
				logger.LogWarning(nodeID, "System", fmt.Sprintf("Failed to get initial HTTP metrics: %v", err))
				logger.LogWarning(nodeID, "System", "Will use default values until HTTP metrics are available")
			} else {
				totalMemory = metrics.System.Memory.TotalGB
				totalCPU = float64(metrics.System.CPU.Cores)
				logger.LogSuccess(nodeID, "System", fmt.Sprintf("Initialized with HTTP metrics - CPU: %.1f cores, Memory: %.1f GB",
					totalCPU, totalMemory))
			}
		}

		appState.NodeData[nodeID] = &NodeMetrics{
			NodeID:      nodeID,
			Status:      "active", // Will be updated based on enabled status
			EPS:         0,
			KafkaLoad:   0,
			CHLoad:      0,
			CPU:         0,
			Memory:      0,
			TotalCPU:    totalCPU,
			TotalMemory: totalMemory,
			LastUpdate:  time.Now(),
		}
		nodeIndex++
	}

	// Set initial demo data for active nodes
	setInitialNodeData()
}

// Set initial data for active nodes
func setInitialNodeData() {
	nodeIndex := 0
	for nodeID, node := range appState.NodeData {
		if nodeID == "node4" || nodeIndex >= 3 { // Skip node4 and limit to first 3 active nodes
			node.Status = "inactive"
			continue
		}

		node.Status = "active"

		// Set different initial values for each node
		switch nodeIndex {
		case 0:
			node.EPS = 9800
			node.KafkaLoad = 4900
			node.CHLoad = 1950
			node.CPU = 65
			node.Memory = 70
		case 1:
			node.EPS = 9900
			node.KafkaLoad = 4950
			node.CHLoad = 1980
			node.CPU = 70
			node.Memory = 75
		case 2:
			node.EPS = 10100
			node.KafkaLoad = 5050
			node.CHLoad = 2020
			node.CPU = 60
			node.Memory = 65
		}

		nodeIndex++
	}

	// Set node5 as active if it exists
	if node, exists := appState.NodeData["node5"]; exists {
		node.Status = "active"
		node.EPS = 10000
		node.KafkaLoad = 5000
		node.CHLoad = 2000
		node.CPU = 75
		node.Memory = 80
	}
}

// Fallback function for default nodes when config is not available
func initializeDefaultNodes() {
	nodes := []string{"node1", "node2", "node3", "node4", "node5"}

	for _, nodeID := range nodes {
		status := "active"
		if nodeID == "node4" {
			status = "inactive"
		}

		// Use local system resource detection for local node (when config is not available)
		totalMemory := 8.0
		totalCPU := 4.0

		// Try to detect local system resources using local methods only
		if nodeID == "node1" { // For the first node, try to detect local system resources
			log.Printf("Detecting local system resources for fallback node %s", nodeID)

			// Detect local memory using existing local function
			localMemory, err := getLocalSystemMemory()
			if err != nil {
				log.Printf("Warning: Failed to detect local memory: %v", err)
			} else {
				totalMemory = localMemory
				log.Printf("SUCCESS: Local system has %.1f GB total memory", totalMemory)
			}

			// Detect local CPU cores using existing local function
			localCPU, err := getLocalSystemCPU()
			if err != nil {
				log.Printf("Warning: Failed to detect local CPU cores: %v", err)
			} else {
				totalCPU = localCPU
				log.Printf("SUCCESS: Local system has %.1f CPU cores", totalCPU)
			}
		}

		appState.NodeData[nodeID] = &NodeMetrics{
			NodeID:      nodeID,
			Status:      status,
			EPS:         0,
			KafkaLoad:   0,
			CHLoad:      0,
			CPU:         0,
			Memory:      0,
			TotalCPU:    totalCPU,
			TotalMemory: totalMemory,
			LastUpdate:  time.Now(),
		}
	}

	// Set initial data for active nodes
	appState.NodeData["node1"].EPS = 9800
	appState.NodeData["node1"].KafkaLoad = 4900
	appState.NodeData["node1"].CHLoad = 1950
	appState.NodeData["node1"].CPU = 65
	appState.NodeData["node1"].Memory = 70

	appState.NodeData["node2"].EPS = 9900
	appState.NodeData["node2"].KafkaLoad = 4950
	appState.NodeData["node2"].CHLoad = 1980
	appState.NodeData["node2"].CPU = 70
	appState.NodeData["node2"].Memory = 75

	appState.NodeData["node3"].EPS = 10100
	appState.NodeData["node3"].KafkaLoad = 5050
	appState.NodeData["node3"].CHLoad = 2020
	appState.NodeData["node3"].CPU = 60
	appState.NodeData["node3"].Memory = 65

	appState.NodeData["node5"].EPS = 10000
	appState.NodeData["node5"].KafkaLoad = 5000
	appState.NodeData["node5"].CHLoad = 2000
	appState.NodeData["node5"].CPU = 75
	appState.NodeData["node5"].Memory = 80
}

// WebSocket handler for real-time updates
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	appState.mutex.Lock()
	appState.clients[conn] = true
	appState.mutex.Unlock()

	// Send initial state
	initialState, _ := json.Marshal(appState)
	conn.WriteMessage(websocket.TextMessage, initialState)

	// Listen for client messages
	for {
		var msg []byte
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received WebSocket message: %s", msg)
	}

	// Unregister client
	appState.mutex.Lock()
	delete(appState.clients, conn)
	appState.mutex.Unlock()
}

// Broadcast updates to all WebSocket clients
func (state *AppState) broadcastUpdate() {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	state.mutex.RLock()
	clients := make([]*websocket.Conn, 0, len(state.clients))
	for client := range state.clients {
		clients = append(clients, client)
	}
	state.mutex.RUnlock()

	for _, client := range clients {
		go func(c *websocket.Conn) {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				state.mutex.Lock()
				delete(state.clients, c)
				state.mutex.Unlock()
				c.Close()
			}
		}(client)
	}
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

	log.Printf("Simulation started with profile: %s, Target EPS: %d", config.Profile, config.TargetEPS)
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

	log.Println("Simulation stopped")
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

	log.Println("Configuration synced")
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

// API endpoint to update node metrics (for simulation)
func updateNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var metrics NodeMetrics
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

// Middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Middleware for CORS
func corsMiddleware(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(next)
}

// Background goroutine to collect real-time metrics from nodes via HTTP
func collectRealMetrics() {
	logger.LogWithNode("System", "Metrics", "Starting HTTP-based real metrics collection", "info")
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for range ticker.C {
		logger.LogWithNode("System", "Metrics", "HTTP metrics collection tick", "info")
		appState.mutex.Lock()

		// Load current node configurations
		err := nodeManager.LoadNodesConfig()
		if err != nil {
			logger.LogWarning("System", "Metrics", fmt.Sprintf("Failed to load nodes config for metrics collection: %v", err))
			appState.mutex.Unlock()
			continue
		}

		// Collect real metrics for each active node via HTTP
		for nodeID, node := range appState.NodeData {
			if node.Status == "active" {
				// Get node configuration for HTTP connection
				nodeConfig, exists := nodeManager.GetNodes()[nodeID]
				if !exists {
					logger.LogWarning(nodeID, "Metrics", "Node not found in configuration")
					node.Status = "error"
					continue
				}

				// Poll metrics via HTTP
				logger.LogWithNode(nodeID, "Metrics", "Polling HTTP metrics", "info")
				metrics, err := pollNodeMetrics(nodeConfig)
				if err != nil {
					logger.LogError(nodeID, "Metrics", fmt.Sprintf("Failed to get HTTP metrics: %v", err))
					node.Status = "error"
				} else {
					// Log the received metrics data
					logger.LogMetric(nodeID, "Metrics", fmt.Sprintf("CPU: %.1f%% (%d cores), Memory: %.1f%% (%.1f/%.1f GB), Load: %.2f",
						metrics.System.CPU.UsedPercent, metrics.System.CPU.Cores,
						metrics.System.Memory.UsedPercent, metrics.System.Memory.UsedGB, metrics.System.Memory.TotalGB,
						metrics.System.CPU.Load1M))

					logger.LogSuccess(nodeID, "Metrics", "Metrics collection successful")

					// Update node metrics with HTTP data (maintaining compatibility with frontend)
					node.CPU = metrics.System.CPU.UsedPercent
					node.Memory = metrics.System.Memory.UsedPercent
					node.TotalCPU = float64(metrics.System.CPU.Cores)
					node.TotalMemory = metrics.System.Memory.TotalGB
					node.Status = "active"

					// Update load metrics (keep existing simulation data for now)
					// In a real implementation, you might want to get these from the binary itself
					// For now, we'll maintain the existing demo values
				}

				node.LastUpdate = time.Now()
			}
		}

		// Broadcast updates to WebSocket clients
		go appState.broadcastUpdate()
		appState.mutex.Unlock()
	}
}

// Get real CPU usage from node via SSH
func getNodeCPUUsage(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'vmstat' for more reliable CPU metrics
	cmd := "vmstat 1 2 | tail -1 | awk '{print $13}'"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		// Fallback to top command if vmstat fails
		cmd = "top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'"
		output, err = sshExec(nodeConfig, cmd)
		if err != nil {
			return 0, fmt.Errorf("failed to execute CPU command: %v", err)
		}
	}

	// Parse CPU usage from output (vmstat returns idle %, top returns usage %)
	var cpuUsage float64
	cleanOutput := strings.TrimSpace(output)

	if strings.Contains(cleanOutput, "%") {
		// Contains % symbol, likely from top command
		cpuUsage, err = strconv.ParseFloat(strings.TrimSuffix(cleanOutput, "%"), 64)
		if err != nil {
			// Try regex extraction as fallback
			re := regexp.MustCompile(`\d+\.?\d*`)
			matches := re.FindAllString(cleanOutput, -1)
			if len(matches) > 0 {
				cpuUsage, _ = strconv.ParseFloat(matches[len(matches)-1], 64)
			} else {
				return 0, fmt.Errorf("failed to parse CPU usage from output: %q", cleanOutput)
			}
		}
	} else {
		// No % symbol, likely from vmstat (idle percentage)
		idle, err := strconv.ParseFloat(cleanOutput, 64)
		if err != nil {
			// Try regex extraction as fallback
			re := regexp.MustCompile(`\d+\.?\d*`)
			matches := re.FindAllString(cleanOutput, -1)
			if len(matches) > 0 {
				idle, _ = strconv.ParseFloat(matches[len(matches)-1], 64)
			} else {
				return 0, fmt.Errorf("failed to parse idle CPU from output: %q", cleanOutput)
			}
		}
		cpuUsage = 100 - idle // Convert idle % to usage %
	}

	// Ensure CPU usage is within valid range
	if cpuUsage < 0 {
		cpuUsage = 0
	} else if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage, nil
}

// Get real memory usage from node via SSH
func getNodeMemoryUsage(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'free' command with better parsing
	cmd := "free -b | grep Mem | awk '{printf \"%.2f\", ($3 / $2) * 100}'"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to execute memory command: %v", err)
	}

	// Parse memory usage from output
	memUsage, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory usage: %v", err)
	}

	// Ensure memory usage is within valid range
	if memUsage < 0 {
		memUsage = 0
	} else if memUsage > 100 {
		memUsage = 100
	}

	return memUsage, nil
}

// Get total memory from node via SSH
func getNodeTotalMemory(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'free' command to get total memory in GB
	cmd := "free -b | grep Mem | awk '{printf \"%.2f\", $2 / 1024 / 1024 / 1024}'"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to execute total memory command: %v", err)
	}

	// Parse total memory from output
	totalMemory, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse total memory: %v", err)
	}

	// Ensure we have a reasonable minimum
	if totalMemory < 1 {
		return 8.0, nil // fallback to 8GB if parsing fails
	}

	return totalMemory, nil
}

// pollNodeMetrics performs HTTP GET request to node's metrics endpoint
func pollNodeMetrics(nodeConfig node_control.NodeConfig) (*HTTPMetricsResponse, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Build metrics URL
	metricsURL := fmt.Sprintf("http://%s:%s/api/system/metrics", nodeConfig.Host, port)

	logger.LogWithNode(nodeConfig.Host, "HTTP", fmt.Sprintf("Making GET request to %s", metricsURL), "info")

	// Make HTTP request
	resp, err := client.Get(metricsURL)
	if err != nil {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("Request failed: %v", err))
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	logger.LogWithNode(nodeConfig.Host, "HTTP", fmt.Sprintf("Response status: %d", resp.StatusCode), "info")

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("Bad status: %d %s", resp.StatusCode, resp.Status))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON response
	var metrics HTTPMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("JSON decode failed: %v", err))
		return nil, fmt.Errorf("JSON decode failed: %v", err)
	}

	logger.LogSuccess(nodeConfig.Host, "HTTP", "Metrics response parsed successfully")
	return &metrics, nil
}

// Get total CPU cores from node via SSH (legacy function - kept for compatibility)
func getNodeTotalCPU(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'nproc' command to get CPU count
	cmd := "nproc"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		// Fallback to parsing /proc/cpuinfo
		cmd = "grep -c 'processor' /proc/cpuinfo"
		output, err = sshExec(nodeConfig, cmd)
		if err != nil {
			return 0, fmt.Errorf("failed to execute CPU command: %v", err)
		}
	}

	// Parse CPU cores from output
	cpuCores, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU cores: %v", err)
	}

	// Ensure we have a reasonable minimum
	if cpuCores < 1 {
		return 4.0, nil // fallback to 4 cores if parsing fails
	}

	return cpuCores, nil
}

// Get local system total memory
func getLocalSystemMemory() (float64, error) {
	cmd := exec.Command("free", "-b")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute local free command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				bytes, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse memory bytes: %v", err)
				}
				// Convert bytes to GB
				return float64(bytes) / 1024 / 1024 / 1024, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find memory information in free output")
}

// Get local system total CPU cores
func getLocalSystemCPU() (float64, error) {
	cmd := exec.Command("nproc")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute local nproc command: %v", err)
	}

	cpuCores, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU cores: %v", err)
	}

	if cpuCores < 1 {
		return 4.0, nil // fallback to 4 cores if parsing fails
	}

	return cpuCores, nil
}

// Execute SSH command and return output
func sshExec(nodeConfig node_control.NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR", // Reduce SSH warnings
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)

	// Get stdout and stderr separately
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipes: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start SSH command: %v", err)
	}

	// Read stdout
	stdoutBytes, _ := io.ReadAll(stdout)

	// Read stderr (to capture warnings)
	stderrBytes, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("SSH command failed: %v, stderr: %s", err, string(stderrBytes))
	}

	// Clean the output by removing SSH warnings and connection messages
	output := string(stdoutBytes)
	log.Printf("Raw stdout: %q", output) // Debug log
	output = cleanSSHOutput(output)
	log.Printf("Cleaned stdout: %q", output) // Debug log

	// If output is still empty or contains warnings, try stderr
	if strings.TrimSpace(output) == "" || strings.TrimSpace(output) == "0" {
		output = string(stderrBytes)
		log.Printf("Raw stderr: %q", output) // Debug log
		output = cleanSSHOutput(output)
		log.Printf("Cleaned stderr: %q", output) // Debug log
	}

	return output, nil
}

// Clean SSH output by extracting numeric values using regex
func cleanSSHOutput(output string) string {
	// Use regex to find all numeric values (including decimals) in the output
	re := regexp.MustCompile(`\d+\.?\d*`)
	matches := re.FindAllString(output, -1)

	// Return the last numeric match (should be the command output)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}

	// If no numeric value found, return default
	return "0"
}

// Check if a string is numeric
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Node Management API Handlers

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

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

// CLI Node Management Functions

// handleNodeManagementCLI handles CLI commands for node management
func handleNodeManagementCLI(command string, args []string) bool {
	switch command {
	case "add":
		handleAddNodeCLI(args)
		return true
	case "remove":
		handleRemoveNodeCLI(args)
		return true
	case "enable":
		handleEnableNodeCLI(args)
		return true
	case "disable":
		handleDisableNodeCLI(args)
		return true
	case "list":
		handleListNodesCLI()
		return true
	case "list-enabled":
		handleListEnabledNodesCLI()
		return true
	case "web":
		// Continue to web server mode
		return false
	default:
		return false // Not a node management command
	}
}

func handleAddNodeCLI(args []string) {
	if len(args) < 6 {
		log.Fatal("Usage: vuDataSim-manager add <name> <host> <user> <key_path> <conf_dir> <binary_dir> [description] [enabled]")
	}

	name := args[0]
	host := args[1]
	user := args[2]
	keyPath := args[3]
	confDir := args[4]
	binaryDir := args[5]

	description := ""
	if len(args) > 6 {
		description = strings.Join(args[6:len(args)-1], " ")
	}

	enabled := true
	if len(args) > 7 {
		enabled = args[len(args)-1] == "true"
	}

	err := nodeManager.AddNode(name, host, user, keyPath, confDir, binaryDir, description, enabled)
	if err != nil {
		log.Fatal(err)
	}
}

func handleRemoveNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager remove <name>")
	}

	name := args[0]
	err := nodeManager.RemoveNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleEnableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager enable <name>")
	}

	name := args[0]
	err := nodeManager.EnableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleDisableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager disable <name>")
	}

	name := args[0]
	err := nodeManager.DisableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleListNodesCLI() {
	nodes := nodeManager.GetNodes()
	if len(nodes) == 0 {
		fmt.Println("No nodes configured")
		return
	}

	fmt.Println("Configured Nodes:")
	fmt.Println("================")

	for name, config := range nodes {
		status := "Disabled"
		if config.Enabled {
			status = "Enabled"
		}

		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
	}
}

func handleListEnabledNodesCLI() {
	enabledNodes := nodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		fmt.Println("No enabled nodes")
		return
	}

	fmt.Println("Enabled Nodes:")
	fmt.Println("==============")

	for name, config := range enabledNodes {
		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
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

// handleAPIGetClickHouseMetrics handles GET /api/clickhouse/metrics
func handleAPIGetClickHouseMetrics(w http.ResponseWriter, r *http.Request) {
	if clickHouseClient == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: "ClickHouse client not initialized",
		})
		return
	}

	metrics, err := collectClickHouseMetrics()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to collect ClickHouse metrics: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse metrics retrieved successfully",
		Data:    metrics,
	})
}

// handleAPIClickHouseHealth handles GET /api/clickhouse/health
func handleAPIClickHouseHealth(w http.ResponseWriter, r *http.Request) {
	if clickHouseClient == nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: "ClickHouse client not initialized",
			Data: map[string]interface{}{
				"status": "disconnected",
			},
		})
		return
	}

	err := clickHouseClient.HealthCheck()
	if err != nil {
		sendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Message: fmt.Sprintf("ClickHouse health check failed: %v", err),
			Data: map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			},
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse is healthy",
		Data: map[string]interface{}{
			"status":       "connected",
			"host":         clickHouseConfig.Host,
			"port":         clickHouseConfig.Port,
			"database":     clickHouseConfig.Database,
			"last_checked": time.Now(),
		},
	})
}

// checkSSHConnectivity checks SSH connectivity for a single node
func checkSSHConnectivity(nodeName string, nodeConfig node_control.NodeConfig) SSHStatus {
	status := SSHStatus{
		NodeName:    nodeName,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Test SSH connection with a simple command
	testCmd := "echo 'SSH connection test'"
	output, err := nodeManager.SSHExecWithOutput(nodeConfig, testCmd)

	if err != nil {
		status.Status = "disconnected"
		status.Message = fmt.Sprintf("SSH connection failed: %v", err)
		logger.LogWarning(nodeName, "SSH", fmt.Sprintf("Connection check failed: %v", err))
	} else if strings.TrimSpace(output) == "SSH connection test" {
		status.Status = "connected"
		status.Message = "SSH connection successful"
		logger.LogSuccess(nodeName, "SSH", "Connection check passed")
	} else {
		status.Status = "error"
		status.Message = fmt.Sprintf("Unexpected SSH response: %s", output)
		logger.LogWarning(nodeName, "SSH", fmt.Sprintf("Unexpected response: %s", output))
	}

	return status
}

// Serve static files with proper MIME types
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path
	if r.URL.Path == "/" {
		http.ServeFile(w, r, StaticDir+"/index.html")
		return
	}

	// Serve other static files with proper MIME types
	staticPath := StaticDir + r.URL.Path

	// Set proper MIME types based on file extension
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}

	http.ServeFile(w, r, staticPath)
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

	// Main config is loaded dynamically when needed

	sources := o11yManager.GetAvailableSources()
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    sources,
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

func main() {
	// Initialize logger
	logFilePath := "logs/vuDataSim.log"
	if err := logger.InitLogger(logFilePath); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize start time
	appState.StartTime = time.Now()

	// Initialize node manager
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load nodes config")
		logger.Warn().Msg("Node management features may not be available")
	}

	// Initialize o11y source manager
	err = o11yManager.LoadMaxEPSConfig()
	if err != nil {
		log.Printf("Warning: Failed to load max EPS config: %v", err)
		log.Println("O11y source management features may not be available")
	}

	// Main config is loaded dynamically when needed

	// Source configs are loaded dynamically when needed

	// Check for CLI node management commands
	if len(os.Args) > 1 {
		command := os.Args[1]
		if handleNodeManagementCLI(command, os.Args[2:]) {
			return // Exit after handling CLI command
		}
	}

	logger.Info().Str("version", AppVersion).Msg("Starting vuDataSim Cluster Manager")
	logger.Info().Str("static_dir", StaticDir).Msg("Serving static files")

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Static file serving with proper MIME types
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set proper MIME types for static files
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Content-Type", "text/html")
		}

		http.ServeFile(w, r, StaticDir+"/"+r.URL.Path)
	})))
	router.HandleFunc("/", serveStatic)

	// WebSocket endpoint
	router.HandleFunc("/ws", handleWebSocket)

	// API endpoints
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/dashboard", getDashboardData).Methods("GET")
	api.HandleFunc("/simulation/start", startSimulation).Methods("POST")
	api.HandleFunc("/simulation/stop", stopSimulation).Methods("POST")
	api.HandleFunc("/config/sync", syncConfiguration).Methods("POST")
	api.HandleFunc("/logs", getLogs).Methods("GET")
	api.HandleFunc("/nodes/{nodeId}/metrics", updateNodeMetrics).Methods("PUT")
	api.HandleFunc("/health", healthCheck).Methods("GET")

	// Node management API endpoints
	api.HandleFunc("/nodes", handleAPINodes).Methods("GET")
	api.HandleFunc("/nodes/{name}", handleAPINodeActions).Methods("POST", "PUT", "DELETE")
	api.HandleFunc("/cluster-settings", handleAPIClusterSettings).Methods("GET", "PUT")

	// Binary control API endpoints
	api.HandleFunc("/binary/status", handleAPIGetAllBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/status/{node}", handleAPIGetBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/start/{node}", handleAPIStartBinary).Methods("POST")
	api.HandleFunc("/binary/stop/{node}", handleAPIStopBinary).Methods("POST")

	// O11y Source Manager API endpoints
	api.HandleFunc("/o11y/sources", handleAPIGetO11ySources).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}", handleAPIGetO11ySourceDetails).Methods("GET")
	api.HandleFunc("/o11y/eps/distribute", handleAPIDistributeEPS).Methods("POST")
	api.HandleFunc("/o11y/eps/current", handleAPIGetCurrentEPS).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}/enable", handleAPIEnableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/sources/{source}/disable", handleAPIDisableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/max-eps", handleAPIGetMaxEPSConfig).Methods("GET")
	// SSH status API endpoint
	api.HandleFunc("/ssh/status", handleAPIGetSSHStatus).Methods("GET")
	// ClickHouse metrics API endpoints
	api.HandleFunc("/clickhouse/metrics", handleAPIGetClickHouseMetrics).Methods("GET")
	api.HandleFunc("/clickhouse/health", handleAPIClickHouseHealth).Methods("GET")

	// Initialize ClickHouse client
	if err := initClickHouse(); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize ClickHouse client - metrics will not be available")
	} else {
		logger.Info().Msg("ClickHouse client initialized successfully")
	}

	// Start background real metrics collection

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")

		appState.mutex.Lock()
		appState.IsSimulationRunning = false
		appState.mutex.Unlock()

		os.Exit(0)
	}()

	// Start server
	logger.Info().Str("port", Port).Msg("Server starting")
	logger.Info().Str("url", "http://localhost"+Port).Msg("Open in browser")

	srv := &http.Server{
		Addr:         Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
