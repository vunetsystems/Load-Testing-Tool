package bin_control

import (
	"fmt"
	"log"
	"net/http"
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

	binaryPath := fmt.Sprintf("%s/node_metrics_api", node.BinaryDir)
	log.Printf("Starting node_metrics_api on node %s: %s", nodeName, binaryPath)

	// Check if already running
	output, err := bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err == nil && output != "" {
		return response(false, fmt.Sprintf("node_metrics_api already running on node %s", nodeName)), fmt.Errorf("metrics binary already running")
	}

	// First ensure binary exists and is executable
	checkCmd := fmt.Sprintf("test -x %s && echo 'Binary exists and is executable'", binaryPath)
	if output, err := bc.sshExecWithOutput(node, checkCmd); err != nil {
		return response(false, fmt.Sprintf("node_metrics_api binary not found or not executable on node %s: %v", nodeName, err)), err
	} else {
		log.Printf("Binary check result: %s", output)
	}

	// Start metrics binary with proper logging
	log.Printf("Starting binary with command: cd %s && ./node_metrics_api --port 8086", node.BinaryDir)
	startCmd := fmt.Sprintf("cd %s && ./node_metrics_api --port 8086 > metrics_api.log 2>&1", node.BinaryDir)
	if err := bc.sshExec(node, startCmd); err != nil {
		// Get error logs if startup failed
		logOutput, _ := bc.sshExecWithOutput(node, fmt.Sprintf("cd %s && cat metrics_api.log 2>/dev/null || echo 'No log file found'", node.BinaryDir))
		return response(false, fmt.Sprintf("Failed to start node_metrics_api on node %s: %v. Startup log: %s", nodeName, err, logOutput)), err
	}

	log.Printf("Binary start command sent, waiting for startup...")
	time.Sleep(3 * time.Second)

	// Check if binary is actually running
	output, err = bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err != nil || output == "" {
		// Get startup error logs
		logOutput, _ := bc.sshExecWithOutput(node, fmt.Sprintf("cd %s && cat metrics_api.log 2>/dev/null || echo 'No error log available'", node.BinaryDir))
		return response(false, fmt.Sprintf("node_metrics_api failed to start on node %s. Process check failed: %v, Startup log: %s", nodeName, err, logOutput)), fmt.Errorf("binary startup failed")
	}

	log.Printf("Binary process found running, performing health check...")

	// Verify the binary is actually responding on port 8086
	time.Sleep(2 * time.Second)
	healthURL := fmt.Sprintf("http://%s:8086/api/system/health", node.Host)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		logOutput, _ := bc.sshExecWithOutput(node, fmt.Sprintf("cd %s && cat metrics_api.log", node.BinaryDir))
		return response(false, fmt.Sprintf("node_metrics_api not responding on node %s. Health check failed: %v, Log: %s", nodeName, err, logOutput)), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logOutput, _ := bc.sshExecWithOutput(node, fmt.Sprintf("cd %s && cat metrics_api.log", node.BinaryDir))
		return response(false, fmt.Sprintf("node_metrics_api health check failed on node %s. Status: %d, Log: %s", nodeName, resp.StatusCode, logOutput)), fmt.Errorf("health check failed")
	}

	log.Printf("node_metrics_api successfully started and verified on node %s", nodeName)

	data := map[string]interface{}{
		"nodeName":     nodeName,
		"action":       "start_metrics",
		"timeout":      timeout,
		"binaryPath":   binaryPath,
		"status":       "running",
		"healthCheck":  "passed",
		"port":         8086,
		"healthUrl":    healthURL,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("node_metrics_api started successfully on node %s", nodeName),
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

	log.Printf("Stopping node_metrics_api on node %s", nodeName)

	// Check if binary is actually running
	output, err := bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err != nil || output == "" {
		return response(false, fmt.Sprintf("node_metrics_api not running on node %s", nodeName)), fmt.Errorf("metrics binary not running")
	}

	log.Printf("Found running node_metrics_api processes: %s", output)

	// Kill all matching processes
	pids := strings.Split(output, "\n")
	stoppedCount := 0
	for _, pidStr := range pids {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			log.Printf("Warning: Invalid PID %s: %v", pidStr, err)
			continue
		}

		// Try graceful kill first
		killCmd := fmt.Sprintf("kill %d", pid)
		if err := bc.sshExec(node, killCmd); err != nil {
			log.Printf("Graceful kill failed for PID %d, trying force kill", pid)
			// Force kill if graceful fails
			killCmd = fmt.Sprintf("kill -9 %d", pid)
			if err := bc.sshExec(node, killCmd); err != nil {
				log.Printf("Warning: Failed to kill process %d: %v", pid, err)
			}
		} else {
			stoppedCount++
			log.Printf("Successfully stopped process %d", pid)
		}
	}

	log.Printf("Waiting for processes to stop...")
	time.Sleep(3 * time.Second)

	// Verify all processes are stopped
	output, err = bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	status := "running"
	if err != nil || output == "" {
		status = "stopped"
		log.Printf("node_metrics_api successfully stopped on node %s", nodeName)
	} else {
		log.Printf("Warning: Some node_metrics_api processes may still be running: %s", output)
	}

	data := map[string]interface{}{
		"nodeName":     nodeName,
		"action":       "stop_metrics",
		"timeout":      timeout,
		"status":       status,
		"stoppedCount": stoppedCount,
		"remaining":    output,
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("node_metrics_api stop operation completed on node %s (stopped %d processes)", nodeName, stoppedCount),
		Data:    data,
	}, nil
}

// DebugMetricsBinary provides detailed debugging information for the metrics binary on a node
func (bc *BinaryControl) DebugMetricsBinary(nodeName string) (*BinaryControlResponse, error) {
	node, ok := bc.nodesConfig.Nodes[nodeName]
	if !ok {
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}

	binaryPath := fmt.Sprintf("%s/node_metrics_api", node.BinaryDir)
	debugInfo := make(map[string]interface{})

	// 1. Check if binary file exists and is executable
	log.Printf("=== DEBUG INFO FOR NODE %s ===", nodeName)

	// Check binary file
	fileCheck, err := bc.sshExecWithOutput(node, fmt.Sprintf("ls -la %s", binaryPath))
	debugInfo["binary_file_info"] = fileCheck
	if err != nil {
		debugInfo["binary_exists"] = false
		debugInfo["binary_error"] = err.Error()
	} else {
		debugInfo["binary_exists"] = true
	}

	// Check if executable
	execCheck, err := bc.sshExecWithOutput(node, fmt.Sprintf("test -x %s && echo 'executable' || echo 'not executable'", binaryPath))
	debugInfo["is_executable"] = strings.TrimSpace(execCheck) == "executable"

	// 2. Check running processes
	processes, err := bc.sshExecWithOutput(node, "pgrep -f node_metrics_api")
	if err != nil {
		debugInfo["processes_running"] = false
		debugInfo["process_error"] = err.Error()
	} else {
		debugInfo["processes_running"] = true
		debugInfo["process_list"] = strings.Split(processes, "\n")
	}

	// 3. Check port availability
	portCheck, err := bc.sshExecWithOutput(node, "netstat -tlnp 2>/dev/null | grep :8086 || ss -tlnp 2>/dev/null | grep :8086 || echo 'port check command not available'")
	debugInfo["port_8086_status"] = portCheck
	if err != nil {
		debugInfo["port_check_error"] = err.Error()
	}

	// 4. Check for error logs
	logContent, err := bc.sshExecWithOutput(node, fmt.Sprintf("cd %s && cat metrics_api.log 2>/dev/null || echo 'No log file found'", node.BinaryDir))
	debugInfo["startup_log"] = logContent
	if err != nil {
		debugInfo["log_read_error"] = err.Error()
	}

	// 5. Try to start binary manually and capture immediate output
	log.Printf("Attempting manual start for debugging...")
	manualStartCmd := fmt.Sprintf("cd %s && timeout 10s ./node_metrics_api --port 8086 2>&1 || echo 'Manual start failed or timed out'", node.BinaryDir)
	manualOutput, err := bc.sshExecWithOutput(node, manualStartCmd)
	debugInfo["manual_start_output"] = manualOutput
	if err != nil {
		debugInfo["manual_start_error"] = err.Error()
	}

	// 6. Check system resources
	diskSpace, _ := bc.sshExecWithOutput(node, "df -h .")
	memory, _ := bc.sshExecWithOutput(node, "free -h")
	debugInfo["disk_space"] = diskSpace
	debugInfo["memory_info"] = memory

	log.Printf("=== DEBUG INFO COLLECTION COMPLETE ===")

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Debug information collected for node %s", nodeName),
		Data:    debugInfo,
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

	output, err := bc.sshExecWithOutput(node, "pgrep -f './finalvudatasim'")
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

	processInfo, err := bc.sshExecWithOutput(node, fmt.Sprintf("ps -p %d -o pid,ppid,pcpu,pmem,etime,cmd", pid))
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
