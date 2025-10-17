package bin_control

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

type NodesConfig struct {
	ClusterSettings ClusterSettings       `yaml:"cluster_settings"`
	Nodes           map[string]NodeConfig `yaml:"nodes"`
}

type ClusterSettings struct {
	BackupRetentionDays int    `yaml:"backup_retention_days"`
	ConflictResolution  string `yaml:"conflict_resolution"`
	ConnectionTimeout   int    `yaml:"connection_timeout"`
	MaxRetries          int    `yaml:"max_retries"`
	SyncTimeout         int    `yaml:"sync_timeout"`
}

type BinaryControl struct {
	nodesConfigPath string
	nodesConfig     NodesConfig
}

type BinaryStatus struct {
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"` // running, stopped, disabled, error
	PID         int    `json:"pid,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	ProcessInfo string `json:"processInfo,omitempty"`
	LastChecked string `json:"lastChecked"`
}

type BinaryControlResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewBinaryControl() *BinaryControl {
	return &BinaryControl{
		nodesConfigPath: "src/configs/nodes.yaml",
		nodesConfig:     NodesConfig{Nodes: make(map[string]NodeConfig)},
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

	if err := yaml.Unmarshal(data, &bc.nodesConfig); err != nil {
		return fmt.Errorf("failed to parse nodes config file: %v", err)
	}

	return nil
}

func (bc *BinaryControl) GetEnabledNodes() map[string]NodeConfig {
	enabled := make(map[string]NodeConfig)
	for name, node := range bc.nodesConfig.Nodes {
		if node.Enabled {
			enabled[name] = node
		}
	}
	return enabled
}

func (bc *BinaryControl) StartBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}
	if !node.Enabled {
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node %s disabled", nodeName)
	}

	status, err := bc.GetBinaryStatus(nodeName)
	if err == nil && status.Status == "running" {
		return response(false, fmt.Sprintf("Binary already running on node %s (PID %d)", nodeName, status.PID)), fmt.Errorf("binary already running")
	}

	binaryPath := fmt.Sprintf("%s/finalvudatasim", node.BinaryDir)
	log.Printf("Starting binary on node %s: %s", nodeName, binaryPath)

	// Run binary in background using nohup, redirect output
	startCmd := fmt.Sprintf("cd %s && nohup ./finalvudatasim > /dev/null 2>&1 & echo $!", node.BinaryDir)
	pidOut, err := bc.sshExecWithOutput(node, startCmd)
	if err != nil {
		return response(false, fmt.Sprintf("Failed to start binary on node %s: %v", nodeName, err)), err
	}
	pidStr := strings.TrimSpace(pidOut)
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return response(false, fmt.Sprintf("Failed to get PID for started binary on node %s", nodeName)), fmt.Errorf("failed to get PID")
	}

	// Schedule kill after timeout (in seconds)
	if timeout > 0 {
		killCmd := fmt.Sprintf("(sleep %d; kill %d) >/dev/null 2>&1 &", timeout*60, pid) // timeout in minutes
		if err := bc.sshExec(node, killCmd); err != nil {
			log.Printf("Warning: failed to schedule kill for binary on node %s: %v", nodeName, err)
		}
	}

	time.Sleep(2 * time.Second)

	newStatus, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Start command sent to node %s, status check failed: %v", nodeName, err),
			Data:    map[string]string{"warning": "Binary may be starting, status check failed"},
		}, nil
	}

	data := map[string]interface{}{
		"nodeName":   nodeName,
		"action":     "start",
		"timeout":    timeout,
		"binaryPath": binaryPath,
		"status":     newStatus,
		"pid":        pid,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary started successfully on node %s (PID %d) with timeout %d min", nodeName, pid, timeout),
		Data:    data,
	}, nil
}

func (bc *BinaryControl) StopBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}
	if !node.Enabled {
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node %s disabled", nodeName)
	}

	status, err := bc.GetBinaryStatus(nodeName)
	if err != nil || status.Status != "running" {
		return response(false, fmt.Sprintf("Binary not running on node %s", nodeName)), fmt.Errorf("binary not running")
	}

	log.Printf("Stopping binary on node %s (PID: %d)", nodeName, status.PID)

	// Attempt graceful kill; if fails, force kill
	killCmd := fmt.Sprintf("kill %d", status.PID)
	if err := bc.sshExec(node, killCmd); err != nil {
		log.Printf("Graceful kill failed, force killing on node %s", nodeName)
		killCmd = fmt.Sprintf("kill -9 %d", status.PID)
		if err := bc.sshExec(node, killCmd); err != nil {
			return response(false, fmt.Sprintf("Failed to stop binary on node %s: %v", nodeName, err)), err
		}
	}

	time.Sleep(2 * time.Second)

	newStatus, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Stop command sent to node %s, status check failed: %v", nodeName, err),
			Data:    map[string]string{"warning": "Binary may be stopped, status check failed"},
		}, nil
	}

	data := map[string]interface{}{
		"nodeName":    nodeName,
		"action":      "stop",
		"timeout":     timeout,
		"previousPID": status.PID,
		"status":      newStatus,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary stopped successfully on node %s", nodeName),
		Data:    data,
	}, nil
}

func (bc *BinaryControl) StartMetricsBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}
	if !node.Enabled {
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node %s disabled", nodeName)
	}

	// Check if already running
	output, err := bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err == nil && output != "" {
		return response(false, fmt.Sprintf("node_metrics_api already running on node %s", nodeName)), fmt.Errorf("metrics binary already running")
	}

	binaryPath := fmt.Sprintf("%s/node_metrics_api", node.BinaryDir)
	log.Printf("Starting node_metrics_api on node %s: %s", nodeName, binaryPath)

	// Run metrics binary in background on port 8086
	cmd := fmt.Sprintf("cd %s && nohup ./node_metrics_api --port 8086 > /dev/null 2>&1 &", node.BinaryDir)
	if err := bc.sshExec(node, cmd); err != nil {
		return response(false, fmt.Sprintf("Failed to start node_metrics_api on node %s: %v", nodeName, err)), err
	}

	time.Sleep(2 * time.Second)

	output, err = bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	status := "stopped"
	if err == nil && output != "" {
		status = "running"
	}

	data := map[string]interface{}{
		"nodeName":   nodeName,
		"action":     "start_metrics",
		"timeout":    timeout,
		"binaryPath": binaryPath,
		"status":     status,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("node_metrics_api started on node %s", nodeName),
		Data:    data,
	}, nil
}

func (bc *BinaryControl) StopMetricsBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}
	if !node.Enabled {
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node %s disabled", nodeName)
	}

	output, err := bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err != nil || output == "" {
		return response(false, fmt.Sprintf("node_metrics_api not running on node %s", nodeName)), fmt.Errorf("metrics binary not running")
	}

	pids := strings.Split(output, "\n")
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			killCmd := fmt.Sprintf("kill %d", pid)
			_ = bc.sshExec(node, killCmd)
		}
	}

	time.Sleep(2 * time.Second)

	output, err = bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	status := "running"
	if err != nil || output == "" {
		status = "stopped"
	}

	data := map[string]interface{}{
		"nodeName": nodeName,
		"action":   "stop_metrics",
		"timeout":  timeout,
		"status":   status,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("node_metrics_api stopped on node %s", nodeName),
		Data:    data,
	}, nil
}

func (bc *BinaryControl) GetBinaryStatus(nodeName string) (*BinaryStatus, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return nil, fmt.Errorf("node %s not found", nodeName)
	}
	if !node.Enabled {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "disabled",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	output, err := bc.sshExecWithOutput(node, "pgrep -f finalvudatasim")
	if err != nil || output == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	pids := strings.Split(output, "\n")
	pid, err := strconv.Atoi(pids[0])
	if err != nil {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "error",
			ProcessInfo: fmt.Sprintf("Failed to parse PID: %v", err),
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, err
	}

	startTime, err := bc.sshExecWithOutput(node, fmt.Sprintf("ps -p %d -o lstart=", pid))
	if err != nil {
		startTime = "Unknown"
	}

	processInfo, err := bc.sshExecWithOutput(node, fmt.Sprintf("ps -p %d -o pid,ppid,pcpu,pmem,etime,comm", pid))
	if err != nil {
		processInfo = fmt.Sprintf("PID: %d", pid)
	}

	return &BinaryStatus{
		NodeName:    nodeName,
		Status:      "running",
		PID:         pid,
		StartTime:   strings.TrimSpace(startTime),
		ProcessInfo: strings.TrimSpace(processInfo),
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

	var statuses []BinaryStatus
	for nodeName := range enabledNodes {
		status, err := bc.GetBinaryStatus(nodeName)
		if err != nil {
			log.Printf("Failed to get status for node %s: %v", nodeName, err)
			statuses = append(statuses, BinaryStatus{
				NodeName:    nodeName,
				Status:      "error",
				ProcessInfo: fmt.Sprintf("Status check failed: %v", err),
				LastChecked: time.Now().Format("2006-01-02 15:04:05"),
			})
		} else {
			statuses = append(statuses, *status)
		}
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved status for %d nodes", len(statuses)),
		Data:    statuses,
	}, nil
}

func (bc *BinaryControl) sshExec(node NodeConfig, command string) error {
	args := []string{
		"-i", node.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", node.User, node.Host),
		command,
	}
	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (bc *BinaryControl) sshExecWithOutput(node NodeConfig, command string) (string, error) {
	args := []string{
		"-i", node.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", node.User, node.Host),
		command,
	}
	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	return strings.TrimSpace(string(output)), err
}

func response(success bool, message string) *BinaryControlResponse {
	return &BinaryControlResponse{
		Success: success,
		Message: message,
	}
}

// Global instance
var binaryControl = NewBinaryControl()

func init() {
	if err := binaryControl.LoadNodesConfig(); err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
	}
}
