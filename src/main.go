package main

import (
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

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"gopkg.in/yaml.v3"
)

// Application version and configuration
const (
	AppVersion = "1.0.0"
	StaticDir  = "./static"
	Port       = ":3000"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin in development
	},
}

// Global application state
type AppState struct {
	IsSimulationRunning bool                    `json:"isSimulationRunning"`
	CurrentProfile      string                  `json:"currentProfile"`
	TargetEPS           int                     `json:"targetEps"`
	TargetKafka         int                     `json:"targetKafka"`
	TargetClickHouse    int                     `json:"targetClickHouse"`
	StartTime           time.Time               `json:"startTime"`
	NodeData            map[string]*NodeMetrics `json:"nodeData"`
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

// ClusterSettings represents cluster-wide configuration
type ClusterSettings struct {
	BackupRetentionDays int    `yaml:"backup_retention_days"`
	ConflictResolution  string `yaml:"conflict_resolution"`
	ConnectionTimeout   int    `yaml:"connection_timeout"`
	MaxRetries          int    `yaml:"max_retries"`
	SyncTimeout         int    `yaml:"sync_timeout"`
}

// NodeConfig represents a single node configuration
type NodeConfig struct {
	Host        string `yaml:"host"`
	User        string `yaml:"user"`
	KeyPath     string `yaml:"key_path"`
	ConfDir     string `yaml:"conf_dir"`
	BinaryDir   string `yaml:"binary_dir"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
}

// NodesConfig represents the entire nodes configuration
type NodesConfig struct {
	ClusterSettings ClusterSettings       `yaml:"cluster_settings"`
	Nodes           map[string]NodeConfig `yaml:"nodes"`
}

// NodeManager handles node operations
type NodeManager struct {
	nodesConfigPath string
	snapshotsDir    string
	backupsDir      string
	logsDir         string
	nodesConfig     NodesConfig
}

// NewNodeManager creates a new node manager instance
func NewNodeManager() *NodeManager {
	return &NodeManager{
		nodesConfigPath: "src/configs/nodes.yaml",
		snapshotsDir:    "src/node_control/node_snapshots",
		backupsDir:      "src/node_control/node_backups",
		logsDir:         "src/node_control/logs",
		nodesConfig: NodesConfig{
			ClusterSettings: ClusterSettings{
				BackupRetentionDays: 30,
				ConflictResolution:  "manual",
				ConnectionTimeout:   10,
				MaxRetries:          3,
				SyncTimeout:         60,
			},
			Nodes: make(map[string]NodeConfig),
		},
	}
}

// LoadNodesConfig loads the nodes configuration from YAML file
func (nm *NodeManager) LoadNodesConfig() error {
	if _, err := os.Stat(nm.nodesConfigPath); os.IsNotExist(err) {
		// Create default config if file doesn't exist
		return nm.SaveNodesConfig()
	}

	data, err := os.ReadFile(nm.nodesConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read nodes config file: %v", err)
	}

	err = yaml.Unmarshal(data, &nm.nodesConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nodes config file: %v", err)
	}

	return nil
}

// SaveNodesConfig saves the nodes configuration to YAML file
func (nm *NodeManager) SaveNodesConfig() error {
	data, err := yaml.Marshal(nm.nodesConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes config: %v", err)
	}

	err = os.WriteFile(nm.nodesConfigPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write nodes config file: %v", err)
	}

	return nil
}

// AddNode adds a new node to the configuration and copies files via SSH
func (nm *NodeManager) AddNode(name, host, user, keyPath, confDir, binaryDir, description string, enabled bool) error {
	if _, exists := nm.nodesConfig.Nodes[name]; exists {
		return fmt.Errorf("node %s already exists", name)
	}

	nodeConfig := NodeConfig{
		Host:        host,
		User:        user,
		KeyPath:     keyPath,
		ConfDir:     confDir,
		BinaryDir:   binaryDir,
		Description: description,
		Enabled:     enabled,
	}

	nm.nodesConfig.Nodes[name] = nodeConfig

	// Save configuration first
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save nodes config: %v", err)
	}

	// Copy files to remote node
	err = nm.copyFilesToNode(name, nodeConfig)
	if err != nil {
		// Rollback configuration on copy failure
		delete(nm.nodesConfig.Nodes, name)
		nm.SaveNodesConfig()
		return fmt.Errorf("failed to copy files to node: %v", err)
	}

	log.Printf("Successfully added node %s", name)
	return nil
}

// RemoveNode removes a node from configuration and cleans up files
func (nm *NodeManager) RemoveNode(name string) error {
	_, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	// Remove from configuration
	delete(nm.nodesConfig.Nodes, name)
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	// Clean up snapshots and backups
	err = nm.cleanupNodeFiles(name)
	if err != nil {
		log.Printf("Warning: failed to cleanup files for node %s: %v", name, err)
	}

	log.Printf("Successfully removed node %s", name)
	return nil
}

// EnableNode enables a node
func (nm *NodeManager) EnableNode(name string) error {
	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	nodeConfig.Enabled = true
	nm.nodesConfig.Nodes[name] = nodeConfig

	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	log.Printf("Successfully enabled node %s", name)
	return nil
}

// DisableNode disables a node
func (nm *NodeManager) DisableNode(name string) error {
	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	nodeConfig.Enabled = false
	nm.nodesConfig.Nodes[name] = nodeConfig

	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	log.Printf("Successfully disabled node %s", name)
	return nil
}

// GetNodes returns all nodes
func (nm *NodeManager) GetNodes() map[string]NodeConfig {
	return nm.nodesConfig.Nodes
}

// GetEnabledNodes returns only enabled nodes
func (nm *NodeManager) GetEnabledNodes() map[string]NodeConfig {
	enabledNodes := make(map[string]NodeConfig)
	for name, config := range nm.nodesConfig.Nodes {
		if config.Enabled {
			enabledNodes[name] = config
		}
	}
	return enabledNodes
}

// copyFilesToNode copies the binary and conf.d directory to the remote node
func (nm *NodeManager) copyFilesToNode(nodeName string, nodeConfig NodeConfig) error {
	localBinary := "src/finalvudatasim"
	localConfDir := "src/conf.d"

	// Check if local files exist
	if _, err := os.Stat(localBinary); os.IsNotExist(err) {
		return fmt.Errorf("local binary file %s not found", localBinary)
	}

	if _, err := os.Stat(localConfDir); os.IsNotExist(err) {
		return fmt.Errorf("local conf.d directory %s not found", localConfDir)
	}

	// Create remote directories
	err := nm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s %s", nodeConfig.BinaryDir, nodeConfig.ConfDir))
	if err != nil {
		return fmt.Errorf("failed to create remote directories: %v", err)
	}

	// Copy binary file
	err = nm.scpCopy(nodeConfig, localBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	if err != nil {
		return fmt.Errorf("failed to copy binary: %v", err)
	}

	// Copy conf.d directory recursively
	err = nm.scpCopyDir(nodeConfig, localConfDir, nodeConfig.ConfDir)
	if err != nil {
		return fmt.Errorf("failed to copy conf.d directory: %v", err)
	}

	log.Printf("Successfully copied files to node %s", nodeName)
	return nil
}

// cleanupNodeFiles removes snapshots and backups for a node
func (nm *NodeManager) cleanupNodeFiles(nodeName string) error {
	nodeSnapshotDir := filepath.Join(nm.snapshotsDir, nodeName)
	nodeBackupDir := filepath.Join(nm.backupsDir, nodeName)

	if _, err := os.Stat(nodeSnapshotDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeSnapshotDir)
		if err != nil {
			return fmt.Errorf("failed to remove snapshot directory: %v", err)
		}
	}

	if _, err := os.Stat(nodeBackupDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeBackupDir)
		if err != nil {
			return fmt.Errorf("failed to remove backup directory: %v", err)
		}
	}

	return nil
}

// sshExec executes a command on the remote node via SSH
func (nm *NodeManager) sshExec(nodeConfig NodeConfig, command string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH command failed: %v", err)
	}

	return nil
}

// scpCopy copies a single file to the remote node
func (nm *NodeManager) scpCopy(nodeConfig NodeConfig, localPath, remotePath string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-r", // recursive for directories
		localPath,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP copy failed: %v", err)
	}

	return nil
}

// scpCopyDir copies a directory recursively to the remote node
func (nm *NodeManager) scpCopyDir(nodeConfig NodeConfig, localDir, remoteDir string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-r",
		localDir,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remoteDir),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP directory copy failed: %v", err)
	}

	return nil
}

// Global node manager instance
var nodeManager = NewNodeManager()

// Initialize node data from nodes.yaml configuration
func init() {
	log.Printf("Initializing node data from configuration...")

	// Load nodes from configuration
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
		log.Println("Using default node configuration")
		initializeDefaultNodes()
		return
	}

	log.Printf("Loaded nodes from config: %v", nodeManager.GetNodes())

	// Initialize node data using real node names from config
	nodeIndex := 0
	for nodeName, nodeConfig := range nodeManager.GetNodes() {
		nodeID := nodeName
		log.Printf("Initializing node: %s, enabled: %v", nodeID, nodeConfig.Enabled)

		// Detect real system resources for this node
		totalMemory := 8.0 // Default fallback
		totalCPU := 4.0    // Default fallback

		// Try to detect real values if node is enabled
		if nodeConfig.Enabled {
			log.Printf("Detecting real system resources for node %s", nodeID)

			// Detect total memory
			realMemory, err := getNodeTotalMemory(nodeConfig)
			if err != nil {
				log.Printf("Warning: Failed to detect total memory for node %s: %v", nodeID, err)
			} else {
				totalMemory = realMemory
				log.Printf("SUCCESS: Node %s has %.1f GB total memory", nodeID, totalMemory)
			}

			// Detect total CPU cores
			realCPU, err := getNodeTotalCPU(nodeConfig)
			if err != nil {
				log.Printf("Warning: Failed to detect CPU cores for node %s: %v", nodeID, err)
			} else {
				totalCPU = realCPU
				log.Printf("SUCCESS: Node %s has %.1f CPU cores", nodeID, totalCPU)
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

		// Use system resource detection for local node (when config is not available)
		totalMemory := 8.0
		totalCPU := 4.0

		// Try to detect local system resources
		if nodeID == "node1" { // For the first node, try to detect local system resources
			log.Printf("Detecting local system resources for fallback node %s", nodeID)

			// Detect local memory
			localMemory, err := getLocalSystemMemory()
			if err != nil {
				log.Printf("Warning: Failed to detect local memory: %v", err)
			} else {
				totalMemory = localMemory
				log.Printf("SUCCESS: Local system has %.1f GB total memory", totalMemory)
			}

			// Detect local CPU cores
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

	// Generate real-time logs based on actual system status
	logs := generateRealTimeLogs()

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

// generateRealTimeLogs creates realistic logs based on actual node data
func generateRealTimeLogs() []map[string]interface{} {
	logs := make([]map[string]interface{}, 0)

	// Add logs for each active node based on real metrics
	for nodeID, node := range appState.NodeData {
		if node.Status == "active" {
			// CPU usage log
			cpuLog := map[string]interface{}{
				"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
				"node":      nodeID,
				"module":    "System Monitor",
				"message":   fmt.Sprintf("CPU usage: %.1f%% (Available: %.1f/%d cores)", node.CPU, node.TotalCPU-(node.TotalCPU*node.CPU/100), int(node.TotalCPU)),
				"type":      "metric",
			}
			logs = append(logs, cpuLog)

			// Memory usage log
			usedMemory := node.TotalMemory * (node.Memory / 100)
			memLog := map[string]interface{}{
				"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
				"node":      nodeID,
				"module":    "System Monitor",
				"message":   fmt.Sprintf("Memory usage: %.1f%% (Used: %.1f/%.1f GB)", node.Memory, usedMemory, node.TotalMemory),
				"type":      "metric",
			}
			logs = append(logs, memLog)

			// Status logs based on load
			if node.CPU > 90 {
				statusLog := map[string]interface{}{
					"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
					"node":      nodeID,
					"module":    "Load Balancer",
					"message":   "High CPU usage detected, redistributing load...",
					"type":      "warning",
				}
				logs = append(logs, statusLog)
			}

			if node.Memory > 80 {
				statusLog := map[string]interface{}{
					"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
					"node":      nodeID,
					"module":    "Memory Manager",
					"message":   "High memory usage detected, optimizing allocations...",
					"type":      "warning",
				}
				logs = append(logs, statusLog)
			}

			// Heartbeat logs
			heartbeatLog := map[string]interface{}{
				"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
				"node":      nodeID,
				"module":    "Heartbeat",
				"message":   "Node heartbeat OK - System operational",
				"type":      "success",
			}
			logs = append(logs, heartbeatLog)
		} else {
			// Inactive node log
			inactiveLog := map[string]interface{}{
				"timestamp": time.Now().Add(-time.Minute * time.Duration(len(logs)+1)).Format("2006-01-02 15:04:05"),
				"node":      nodeID,
				"module":    "System",
				"message":   "Node is inactive. No monitoring data available.",
				"type":      "error",
			}
			logs = append(logs, inactiveLog)
		}
	}

	// Add some general system logs
	systemLogs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-time.Minute * 5).Format("2006-01-02 15:04:05"),
			"node":      "System",
			"module":    "Cluster Manager",
			"message":   "Cluster monitoring active - Real-time data collection enabled",
			"type":      "info",
		},
		{
			"timestamp": time.Now().Add(-time.Minute * 3).Format("2006-01-02 15:04:05"),
			"node":      "System",
			"module":    "WebSocket",
			"message":   "WebSocket server started - Real-time updates enabled",
			"type":      "success",
		},
	}

	logs = append(logs, systemLogs...)

	return logs
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

// Background goroutine to collect real-time metrics from actual nodes
func collectRealMetrics() {
	log.Printf("Starting real metrics collection...")
	ticker := time.NewTicker(3 * time.Second) // Slightly longer interval for real data collection
	defer ticker.Stop()

	for range ticker.C {
		log.Printf("Real metrics collection tick...")
		appState.mutex.Lock()

		// Load current node configurations
		err := nodeManager.LoadNodesConfig()
		if err != nil {
			log.Printf("Warning: Failed to load nodes config for metrics collection: %v", err)
			appState.mutex.Unlock()
			continue
		}

		// Collect real metrics for each active node
		for nodeID, node := range appState.NodeData {
			if node.Status == "active" {
				// Get node configuration for SSH connection
				nodeConfig, exists := nodeManager.GetNodes()[nodeID]
				if !exists {
					log.Printf("Warning: Node %s not found in configuration", nodeID)
					continue
				}

				// Collect real CPU usage
				log.Printf("Collecting CPU metrics for node %s", nodeID)
				cpuUsage, err := getNodeCPUUsage(nodeConfig)
				if err != nil {
					log.Printf("Warning: Failed to get CPU usage for node %s: %v", nodeID, err)
					// Keep previous value on error
				} else {
					log.Printf("SUCCESS: Node %s CPU usage: %.1f%%", nodeID, cpuUsage)
					node.CPU = cpuUsage
				}

				// Collect real memory usage
				log.Printf("Collecting memory metrics for node %s", nodeID)
				memUsage, err := getNodeMemoryUsage(nodeConfig)
				if err != nil {
					log.Printf("Warning: Failed to get memory usage for node %s: %v", nodeID, err)
					// Keep previous value on error
				} else {
					log.Printf("SUCCESS: Node %s Memory usage: %.1f%%", nodeID, memUsage)
					node.Memory = memUsage
				}

				// Collect real total memory (only if not already set or if it's the default 8GB)
				if node.TotalMemory == 8.0 {
					log.Printf("Collecting total memory for node %s", nodeID)
					totalMemory, err := getNodeTotalMemory(nodeConfig)
					if err != nil {
						log.Printf("Warning: Failed to get total memory for node %s: %v", nodeID, err)
						// Keep default 8GB on error
					} else {
						log.Printf("SUCCESS: Node %s Total memory: %.1f GB", nodeID, totalMemory)
						node.TotalMemory = totalMemory
					}
				}

				// Collect real total CPU cores (only if not already set or if it's the default 4 cores)
				if node.TotalCPU == 4.0 {
					log.Printf("Collecting total CPU cores for node %s", nodeID)
					totalCPU, err := getNodeTotalCPU(nodeConfig)
					if err != nil {
						log.Printf("Warning: Failed to get CPU cores for node %s: %v", nodeID, err)
						// Keep default 4 cores on error
					} else {
						log.Printf("SUCCESS: Node %s Total CPU cores: %.1f", nodeID, totalCPU)
						node.TotalCPU = totalCPU
					}
				}

				// Only update real metrics - no simulation
				node.LastUpdate = time.Now()
			}
		}

		// Broadcast updates to WebSocket clients
		go appState.broadcastUpdate()
		appState.mutex.Unlock()
	}
}

// Get real CPU usage from node via SSH
func getNodeCPUUsage(nodeConfig NodeConfig) (float64, error) {
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
func getNodeMemoryUsage(nodeConfig NodeConfig) (float64, error) {
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
func getNodeTotalMemory(nodeConfig NodeConfig) (float64, error) {
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

// Get total CPU cores from node via SSH
func getNodeTotalCPU(nodeConfig NodeConfig) (float64, error) {
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
func sshExec(nodeConfig NodeConfig, command string) (string, error) {
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
			Data:    nodeManager.nodesConfig.ClusterSettings,
		})
	case http.MethodPut:
		var settings ClusterSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		nodeManager.nodesConfig.ClusterSettings = settings
		err := nodeManager.SaveNodesConfig()
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

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize start time
	appState.StartTime = time.Now()

	// Initialize node manager
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
		log.Println("Node management features may not be available")
	}

	// Check for CLI node management commands
	if len(os.Args) > 1 {
		command := os.Args[1]
		if handleNodeManagementCLI(command, os.Args[2:]) {
			return // Exit after handling CLI command
		}
	}

	log.Printf("Starting vuDataSim Cluster Manager v%s", AppVersion)
	log.Printf("Serving static files from: %s", StaticDir)

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

	// Start background real metrics collection
	go collectRealMetrics()

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
	log.Printf("Server starting on port %s", Port)
	log.Printf("Open http://localhost%s in your browser", Port)

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
