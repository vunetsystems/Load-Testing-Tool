package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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

// ClusterSettings represents cluster-wide configuration
type ClusterSettings struct {
	BackupRetentionDays int    `yaml:"backup_retention_days"`
	ConflictResolution  string `yaml:"conflict_resolution"`
	ConnectionTimeout   int    `yaml:"connection_timeout"`
	MaxRetries          int    `yaml:"max_retries"`
	SyncTimeout         int    `yaml:"sync_timeout"`
}

// BinaryControl manages binary operations across nodes
type BinaryControl struct {
	nodesConfigPath string
	nodesConfig     NodesConfig
}

// BinaryStatus represents the status of a binary on a node
type BinaryStatus struct {
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"` // running, stopped, not_found, error
	PID         int    `json:"pid"`
	StartTime   string `json:"startTime"`
	ProcessInfo string `json:"processInfo"`
	LastChecked string `json:"lastChecked"`
}

// BinaryControlResponse represents the response from binary control operations
type BinaryControlResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewBinaryControl creates a new binary control instance
func NewBinaryControl() *BinaryControl {
	return &BinaryControl{
		nodesConfigPath: "src/configs/nodes.yaml",
		nodesConfig: NodesConfig{
			Nodes: make(map[string]NodeConfig),
		},
	}
}

// LoadNodesConfig loads the nodes configuration from YAML file
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

// GetEnabledNodes returns only enabled nodes
func (bc *BinaryControl) GetEnabledNodes() map[string]NodeConfig {
	enabledNodes := make(map[string]NodeConfig)
	for name, config := range bc.nodesConfig.Nodes {
		if config.Enabled {
			enabledNodes[name] = config
		}
	}
	return enabledNodes
}

// StartBinary starts the binary on a specific node
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

	// Check if binary is already running
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

	// Use nohup to run in background and redirect output
	startCmd := fmt.Sprintf("cd %s && nohup ", nodeConfig.BinaryDir, binaryPath)

	err = bc.sshExec(nodeConfig, startCmd)
	if err != nil {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start binary on node %s: %v", nodeName, err),
		}, err
	}

	// Wait a moment for the process to start
	time.Sleep(2 * time.Second)

	// Get the updated status
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

// StopBinary stops the binary on a specific node
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

	// Get current status to find PID
	status, err := bc.GetBinaryStatus(nodeName)
	if err != nil || status.Status != "running" {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Binary is not running on node %s", nodeName),
		}, fmt.Errorf("binary not running on node %s", nodeName)
	}

	// Stop the binary using kill command
	log.Printf("Stopping binary on node %s (PID: %d)", nodeName, status.PID)

	// First try graceful termination
	killCmd := fmt.Sprintf("kill %d", status.PID)
	err = bc.sshExec(nodeConfig, killCmd)
	if err != nil {
		// If graceful termination fails, try force kill
		log.Printf("Graceful termination failed, trying force kill on node %s", nodeName)
		killCmd = fmt.Sprintf("kill -9 %d", status.PID)
		err = bc.sshExec(nodeConfig, killCmd)
		if err != nil {
			return &BinaryControlResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to stop binary on node %s: %v", nodeName, err),
			}, err
		}
	}

	// Wait a moment for the process to stop
	time.Sleep(2 * time.Second)

	// Verify the process has stopped
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

// GetBinaryStatus gets the status of the binary on a specific node
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
	// Look for the finalvudatasim process
	checkCmd := "pgrep -f finalvudatasim"
	output, err := bc.sshExecWithOutput(nodeConfig, checkCmd)
	if err != nil || strings.TrimSpace(output) == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Get process details
	pids := strings.Split(strings.TrimSpace(output), "\n")
	if len(pids) == 0 || pids[0] == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Get the first PID (main process)
	pid, err := strconv.Atoi(pids[0])
	if err != nil {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "error",
			ProcessInfo: fmt.Sprintf("Failed to parse PID: %v", err),
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, err
	}

	// Get process start time
	startTimeCmd := fmt.Sprintf("ps -p %d -o lstart=", pid)
	startTimeOutput, err := bc.sshExecWithOutput(nodeConfig, startTimeCmd)
	if err != nil {
		startTimeOutput = "Unknown"
	} else {
		startTimeOutput = strings.TrimSpace(startTimeOutput)
	}

	// Get detailed process info
	processInfoCmd := fmt.Sprintf("ps -p %d -o pid,ppid,pcpu,pmem,etime,comm", pid)
	processInfo, err := bc.sshExecWithOutput(nodeConfig, processInfoCmd)
	if err != nil {
		processInfo = fmt.Sprintf("PID: %d", pid)
	}

	return &BinaryStatus{
		NodeName:    nodeName,
		Status:      "running",
		PID:         pid,
		StartTime:   startTimeOutput,
		ProcessInfo: strings.TrimSpace(processInfo),
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// GetAllBinaryStatuses gets the status of binaries on all enabled nodes
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
			// Add error status
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

// sshExec executes a command on the remote node via SSH
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

// sshExecWithOutput executes a command on the remote node and returns the output
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

// Initialize binary control
func init() {
	err := binaryControl.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config for binary control: %v", err)
	}
}
