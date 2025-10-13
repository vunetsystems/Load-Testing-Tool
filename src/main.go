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

	"vuDataSim/src/logger"

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
	MetricsPort int    `yaml:"metrics_port"`
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

	// Ensure Nodes map is initialized
	if nm.nodesConfig.Nodes == nil {
		nm.nodesConfig.Nodes = make(map[string]NodeConfig)
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

	logger.LogSuccess(name, "Node", "Node enabled successfully")

	// Copy files to the node
	err = nm.copyFilesToNode(name, nodeConfig)
	if err != nil {
		logger.LogError(name, "Deployment", fmt.Sprintf("Failed to copy files: %v", err))
		return fmt.Errorf("failed to copy files to node: %v", err)
	}

	// Start the node metrics binary
	err = nm.startNodeMetricsBinary(name, nodeConfig)
	if err != nil {
		logger.LogWarning(name, "Metrics", fmt.Sprintf("Failed to start metrics binary: %v", err))
		// Don't fail the enable operation if metrics binary fails to start
	} else {
		logger.LogSuccess(name, "Metrics", "Node metrics binary started successfully")
	}

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

// copyFilesToNode copies the binaries and conf.d directory to the remote node
func (nm *NodeManager) copyFilesToNode(nodeName string, nodeConfig NodeConfig) error {
	localMainBinary := "src/finalvudatasim"
	localMetricsBinary := "node_metrics_api" // Use the built binary from root directory
	localConfDir := "src/conf.d"

	logger.Debug().
		Str("node", nodeName).
		Str("main_binary", localMainBinary).
		Str("metrics_binary", localMetricsBinary).
		Str("conf_dir", localConfDir).
		Msg("Deployment paths")

	// Check if local files exist
	if _, err := os.Stat(localMainBinary); os.IsNotExist(err) {
		return fmt.Errorf("local main binary file %s not found", localMainBinary)
	}

	if _, err := os.Stat(localMetricsBinary); os.IsNotExist(err) {
		return fmt.Errorf("local metrics binary file %s not found", localMetricsBinary)
	}

	// Build the node_metrics_api binary if it doesn't exist
	if _, err := os.Stat(localMetricsBinary); os.IsNotExist(err) {
		log.Printf("Building node_metrics_api binary for node %s...", nodeName)
		buildCmd := exec.Command("go", "build", "-o", localMetricsBinary, "src/node_metrics_api")
		buildCmd.Dir = "." // Run from project root
		if output, err := buildCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to build node_metrics_api binary: %v, output: %s", err, string(output))
		}
		log.Printf("node_metrics_api binary built successfully for node %s", nodeName)
	}

	if _, err := os.Stat(localConfDir); os.IsNotExist(err) {
		return fmt.Errorf("local conf.d directory %s not found", localConfDir)
	}

	// Create remote directories
	err := nm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s %s", nodeConfig.BinaryDir, nodeConfig.ConfDir))
	if err != nil {
		return fmt.Errorf("failed to create remote directories: %v", err)
	}

	// Copy main binary file
	logger.Info().
		Str("node", nodeName).
		Str("from", localMainBinary).
		Str("to", filepath.Join(nodeConfig.BinaryDir, "finalvudatasim")).
		Msg("Copying main binary")
	err = nm.scpCopy(nodeConfig, localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	if err != nil {
		logger.Error().
			Str("node", nodeName).
			Err(err).
			Msg("Failed to copy main binary")
		return fmt.Errorf("failed to copy main binary: %v", err)
	}
	logger.LogSuccess(nodeName, "Deployment", "Main binary copied successfully")

	// Copy metrics API binary
	logger.Info().
		Str("node", nodeName).
		Str("from", localMetricsBinary).
		Str("to", filepath.Join(nodeConfig.BinaryDir, "node_metrics_api")).
		Msg("Copying metrics binary")
	err = nm.scpCopy(nodeConfig, localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api"))
	if err != nil {
		logger.Error().
			Str("node", nodeName).
			Err(err).
			Msg("Failed to copy metrics binary")
		return fmt.Errorf("failed to copy metrics binary: %v", err)
	}
	logger.LogSuccess(nodeName, "Deployment", "Metrics binary copied successfully")

	// Copy conf.d directory recursively
	logger.Info().
		Str("node", nodeName).
		Str("from", localConfDir).
		Str("to", nodeConfig.ConfDir).
		Msg("Copying conf.d directory")
	err = nm.scpCopyDir(nodeConfig, localConfDir, nodeConfig.ConfDir)
	if err != nil {
		logger.Error().
			Str("node", nodeName).
			Err(err).
			Msg("Failed to copy conf.d directory")
		return fmt.Errorf("failed to copy conf.d directory: %v", err)
	}
	logger.LogSuccess(nodeName, "Deployment", "Conf.d directory copied successfully")

	logger.LogSuccess(nodeName, "Deployment", "Successfully copied all files to node")
	return nil
}

// readMetricsPort reads the port from the metrics.port file on the remote node
func (nm *NodeManager) readMetricsPort(nodeConfig NodeConfig) (string, error) {
	portFilePath := fmt.Sprintf("%s/metrics.port", nodeConfig.BinaryDir)
	cmd := fmt.Sprintf("cat %s", portFilePath)
	portStr, err := nm.sshExecWithOutput(nodeConfig, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to read port file: %v", err)
	}
	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		return "", fmt.Errorf("port file is empty")
	}
	return portStr, nil
}

// startNodeMetricsBinary starts the node metrics API binary on the remote node
func (nm *NodeManager) startNodeMetricsBinary(nodeName string, nodeConfig NodeConfig) error {
	logger.LogWithNode(nodeName, "Metrics", "Starting node metrics binary...", "info")

	// Kill any existing metrics processes
	killCmd := fmt.Sprintf("pkill -f node_metrics_api || true")
	err := nm.sshExec(nodeConfig, killCmd)
	if err != nil {
		logger.LogWarning(nodeName, "Metrics", fmt.Sprintf("Failed to kill existing metrics processes: %v", err))
	}

	// Make sure the binary has execute permissions
	metricsPath := fmt.Sprintf("%s/node_metrics_api", nodeConfig.BinaryDir)
	chmodCmd := fmt.Sprintf("chmod +x %s", metricsPath)
	err = nm.sshExec(nodeConfig, chmodCmd)
	if err != nil {
		logger.LogWarning(nodeName, "Metrics", fmt.Sprintf("Failed to set execute permissions: %v", err))
	}

	// Start the metrics binary in the background (will find available port)
	startCmd := fmt.Sprintf("cd %s && nohup %s > metrics.log 2>&1 & echo $! > metrics.pid", nodeConfig.BinaryDir, metricsPath)
	logger.LogWithNode(nodeName, "Metrics", fmt.Sprintf("Executing start command: %s", startCmd), "info")

	err = nm.sshExec(nodeConfig, startCmd)
	if err != nil {
		logger.LogError(nodeName, "Metrics", fmt.Sprintf("Failed to execute start command: %v", err))
		return fmt.Errorf("failed to start metrics binary: %v", err)
	}

	logger.LogWithNode(nodeName, "Metrics", "Start command executed, waiting for startup...", "info")

	// Wait a moment for the binary to start
	time.Sleep(3 * time.Second)

	// Check if the process is running by checking the PID file
	checkPidCmd := fmt.Sprintf("cat %s/metrics.pid 2>/dev/null || echo 'no pid'", nodeConfig.BinaryDir)
	pidOutput, err := nm.sshExecWithOutput(nodeConfig, checkPidCmd)
	if err != nil {
		logger.LogWarning(nodeName, "Metrics", fmt.Sprintf("Could not check PID file: %v", err))
	} else if strings.TrimSpace(pidOutput) != "no pid" && strings.TrimSpace(pidOutput) != "" {
		logger.LogSuccess(nodeName, "Metrics", fmt.Sprintf("Metrics binary started with PID: %s", strings.TrimSpace(pidOutput)))
	} else {
		logger.LogWarning(nodeName, "Metrics", "PID file not found or empty")
	}

	// Wait for the port file to be created
	time.Sleep(2 * time.Second)

	// Read the port from the metrics.port file
	port, err := nm.readMetricsPort(nodeConfig)
	if err != nil {
		logger.LogError(nodeName, "Metrics", fmt.Sprintf("Failed to read metrics port: %v", err))
		return fmt.Errorf("failed to read metrics port: %v", err)
	}

	logger.LogWithNode(nodeName, "Metrics", fmt.Sprintf("Metrics API is running on port %s", port), "info")

	// Verify the metrics server is running by making a test request
	client := &http.Client{Timeout: 5 * time.Second}
	healthURL := fmt.Sprintf("http://%s:%s/api/system/health", nodeConfig.Host, port)
	logger.LogWithNode(nodeName, "Metrics", fmt.Sprintf("Checking health endpoint: %s", healthURL), "info")

	resp, err := client.Get(healthURL)
	if err != nil {
		logger.LogError(nodeName, "Metrics", fmt.Sprintf("Health check failed: %v", err))
		return fmt.Errorf("metrics server health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.LogError(nodeName, "Metrics", fmt.Sprintf("Health check returned status %d", resp.StatusCode))
		return fmt.Errorf("metrics server returned status %d", resp.StatusCode)
	}

	logger.LogSuccess(nodeName, "Metrics", "Health check passed - metrics binary is running successfully")
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

// sshExecWithOutput executes a command on the remote node via SSH and returns the output
func (nm *NodeManager) sshExecWithOutput(nodeConfig NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
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

	// Capture stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SCP command: %v", err)
	}

	// Read and log stdout
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			logger.LogMetric("System", "SCP", fmt.Sprintf("STDOUT: %s", line))
		}
	}()

	// Read and log stderr
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			logger.LogMetric("System", "SCP", fmt.Sprintf("STDERR: %s", line))
		}
	}()

	if err := cmd.Wait(); err != nil {
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

	// Capture stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SCP command: %v", err)
	}

	// Read and log stdout
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			logger.LogMetric("System", "SCP", fmt.Sprintf("STDOUT: %s", line))
		}
	}()

	// Read and log stderr
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			logger.LogMetric("System", "SCP", fmt.Sprintf("STDERR: %s", line))
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("SCP directory copy failed: %v", err)
	}

	return nil
}

// Global node manager instance
var nodeManager = NewNodeManager()

// Binary control types and instance
type BinaryControlResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type BinaryStatus struct {
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"`
	PID         int    `json:"pid"`
	StartTime   string `json:"startTime"`
	ProcessInfo string `json:"processInfo"`
	LastChecked string `json:"lastChecked"`
}

type BinaryControl struct {
	nodesConfigPath string
	nodesConfig     NodesConfig
}

func NewBinaryControl() *BinaryControl {
	return &BinaryControl{
		nodesConfigPath: "src/configs/nodes.yaml",
		nodesConfig: NodesConfig{
			Nodes: make(map[string]NodeConfig),
		},
	}
}

func (bc *BinaryControl) LoadNodesConfig() error {
	if _, err := os.Stat(bc.nodesConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("nodes config file not found: %s", bc.nodesConfigPath)
	}

	data, err := os.ReadFile(bc.nodesConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read nodes config file: %v", err)
	}

	err = yaml.Unmarshal(data, &bc.nodesConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nodes config file: %v", err)
	}

	return nil
}

func (bc *BinaryControl) GetEnabledNodes() map[string]NodeConfig {
	enabledNodes := make(map[string]NodeConfig)
	for name, config := range bc.nodesConfig.Nodes {
		if config.Enabled {
			enabledNodes[name] = config
		}
	}
	return enabledNodes
}

func (bc *BinaryControl) GetBinaryStatus(nodeName string) (*BinaryStatus, error) {
	nodeConfig, exists := bc.nodesConfig.Nodes[nodeName]
	if !exists {
		return nil, fmt.Errorf("node %s not found in configuration", nodeName)
	}

	if !nodeConfig.Enabled {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "disabled",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Check if the binary process is running
	checkCmd := "pgrep -f finalvudatasim"
	output, err := bc.sshExecWithOutput(nodeConfig, checkCmd)
	if err != nil || strings.TrimSpace(output) == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Parse PID
	pids := strings.Split(strings.TrimSpace(output), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	pid, err := strconv.Atoi(pids[0])
	if err != nil {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "error",
			ProcessInfo: fmt.Sprintf("Failed to parse PID: %v", err),
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, err
	}

	return &BinaryStatus{
		NodeName:    nodeName,
		Status:      "running",
		PID:         pid,
		StartTime:   "Unknown",
		ProcessInfo: fmt.Sprintf("PID: %d", pid),
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (bc *BinaryControl) GetAllBinaryStatuses() (*BinaryControlResponse, error) {
	enabledNodes := bc.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		return &BinaryControlResponse{
			Success: true,
			Message: "No enabled nodes found",
			Data:    []BinaryStatus{},
		}, nil
	}

	var allStatuses []BinaryStatus
	for nodeName := range enabledNodes {
		status, err := bc.GetBinaryStatus(nodeName)
		if err != nil {
			log.Printf("Warning: Failed to get binary status for node %s: %v", nodeName, err)
			allStatuses = append(allStatuses, BinaryStatus{
				NodeName:    nodeName,
				Status:      "error",
				ProcessInfo: fmt.Sprintf("Status check failed: %v", err),
				LastChecked: time.Now().Format("2006-01-02 15:04:05"),
			})
		} else {
			allStatuses = append(allStatuses, *status)
		}
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved status for %d nodes", len(allStatuses)),
		Data:    allStatuses,
	}, nil
}

func (bc *BinaryControl) StartBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	nodeConfig, exists := bc.nodesConfig.Nodes[nodeName]
	if !exists {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s not found in configuration", nodeName),
		}, fmt.Errorf("node %s not found", nodeName)
	}

	if !nodeConfig.Enabled {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s is disabled", nodeName),
		}, fmt.Errorf("node %s is disabled", nodeName)
	}

	// Check if already running
	status, err := bc.GetBinaryStatus(nodeName)
	if err == nil && status.Status == "running" {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Binary is already running on node %s (PID: %d)", nodeName, status.PID),
		}, fmt.Errorf("binary already running on node %s", nodeName)
	}

	// Start the binary
	binaryPath := fmt.Sprintf("%s/finalvudatasim", nodeConfig.BinaryDir)
	log.Printf("Starting binary on node %s: %s", nodeName, binaryPath)

	// Show detailed command in logs
	startCmd := fmt.Sprintf("cd %s && nohup %s > /dev/null 2>&1 &", nodeConfig.BinaryDir, binaryPath)
	log.Printf("Executing start command on node %s: %s", nodeName, startCmd)

	err = bc.sshExec(nodeConfig, startCmd)
	if err != nil {
		log.Printf("SSH execution failed for start command on node %s: %v", nodeName, err)
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start binary on node %s: %v", nodeName, err),
		}, err
	}

	log.Printf("Binary start command executed successfully on node %s", nodeName)

	time.Sleep(2 * time.Second)

	newStatus, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Binary start command sent to node %s, but status check failed: %v", nodeName, err),
			Data:    map[string]interface{}{"warning": "Binary may be starting, but status verification failed"},
		}, nil
	}

	responseData := map[string]interface{}{
		"nodeName":   nodeName,
		"action":     "start",
		"timeout":    timeout,
		"binaryPath": binaryPath,
		"status":     newStatus,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary started successfully on node %s", nodeName),
		Data:    responseData,
	}, nil
}

func (bc *BinaryControl) StopBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	nodeConfig, exists := bc.nodesConfig.Nodes[nodeName]
	if !exists {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s not found in configuration", nodeName),
		}, fmt.Errorf("node %s not found", nodeName)
	}

	if !nodeConfig.Enabled {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s is disabled", nodeName),
		}, fmt.Errorf("node %s is disabled", nodeName)
	}

	// Get current status
	status, err := bc.GetBinaryStatus(nodeName)
	if err != nil || status.Status != "running" {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Binary is not running on node %s", nodeName),
		}, fmt.Errorf("binary not running on node %s", nodeName)
	}

	// Stop the binary
	log.Printf("Stopping binary on node %s (PID: %d)", nodeName, status.PID)
	killCmd := fmt.Sprintf("kill %d", status.PID)
	err = bc.sshExec(nodeConfig, killCmd)
	if err != nil {
		killCmd = fmt.Sprintf("kill -9 %d", status.PID)
		err = bc.sshExec(nodeConfig, killCmd)
		if err != nil {
			return &BinaryControlResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to stop binary on node %s: %v", nodeName, err),
			}, err
		}
	}

	time.Sleep(2 * time.Second)

	newStatus, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Binary stop command sent to node %s, but status verification failed: %v", nodeName, err),
			Data:    map[string]interface{}{"warning": "Binary may be stopped, but status verification failed"},
		}, nil
	}

	responseData := map[string]interface{}{
		"nodeName":    nodeName,
		"action":      "stop",
		"timeout":     timeout,
		"previousPID": status.PID,
		"status":      newStatus,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary stopped successfully on node %s", nodeName),
		Data:    responseData,
	}, nil
}

func (bc *BinaryControl) sshExec(nodeConfig NodeConfig, command string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
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

func (bc *BinaryControl) sshExecWithOutput(nodeConfig NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Global binary control instance
var binaryControl = NewBinaryControl()

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
		binaryControl = NewBinaryControl()
		return
	}

	logger.Info().Interface("nodes", nodeManager.GetNodes()).Msg("Loaded nodes from config")

	// Initialize binary control with loaded config
	binaryControl = NewBinaryControl()
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

// pollNodeMetrics performs HTTP GET request to node's metrics endpoint
func pollNodeMetrics(nodeConfig NodeConfig) (*HTTPMetricsResponse, error) {
	// Read the port from the metrics.port file
	port, err := nodeManager.readMetricsPort(nodeConfig)
	if err != nil {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("Failed to read metrics port: %v", err))
		return nil, fmt.Errorf("failed to read metrics port: %v", err)
	}

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
