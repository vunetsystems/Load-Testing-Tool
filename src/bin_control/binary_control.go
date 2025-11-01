package bin_control

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
	"vuDataSim/src/logger"
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
	configMutex     sync.RWMutex  // Mutex to protect nodesConfig access
	sshSemaphore    chan struct{} // Semaphore to limit concurrent SSH operations
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

type BulkBinaryResult struct {
	NodeName string `json:"nodeName"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PID      int    `json:"pid,omitempty"`
}

type BulkBinaryData struct {
	TotalNodes int                 `json:"totalNodes"`
	Successful int                 `json:"successful"`
	Failed     int                 `json:"failed"`
	Results    []BulkBinaryResult  `json:"results"`
	Logs       []string            `json:"logs"`
}

type BulkBinaryResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Data    *BulkBinaryData  `json:"data,omitempty"`
}

func NewBinaryControl() *BinaryControl {
	return &BinaryControl{
		nodesConfigPath: "src/configs/nodes.yaml",
		nodesConfig:     NodesConfig{Nodes: make(map[string]NodeConfig)},
		configMutex:     sync.RWMutex{},
		sshSemaphore:    make(chan struct{}, 3), // Limit to 3 concurrent SSH operations
	}
}

func (bc *BinaryControl) LoadNodesConfig() error {
	if bc == nil {
		return fmt.Errorf("BinaryControl instance is nil")
	}
	
	// Lock for writing to protect the config
	bc.configMutex.Lock()
	defer bc.configMutex.Unlock()
	
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
	if bc == nil {
		return make(map[string]NodeConfig)
	}
	
	// Lock for reading to protect the config
	bc.configMutex.RLock()
	defer bc.configMutex.RUnlock()
	
	enabled := make(map[string]NodeConfig)
	for name, node := range bc.nodesConfig.Nodes {
		if node.Enabled {
			enabled[name] = node
		}
	}
	return enabled
}

func (bc *BinaryControl) StartBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	log := logger.Logger.With().Str("node", nodeName).Logger()
	log.Info().Msg("Starting binary")

	// ✅ Load config once per call (cheap)
	if err := bc.LoadNodesConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to reload config")
		return response(false, fmt.Sprintf("Failed to reload config: %v", err)), err
	}

	// ✅ Validate node existence and enable state
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()
	if !ok {
		log.Error().Msg("Node not found in config")
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node missing")
	}
	if !node.Enabled {
		log.Warn().Msg("Node is disabled")
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node disabled")
	}

	// ✅ Check if already running
	status, err := bc.GetBinaryStatus(nodeName)
	if err == nil && status.Status == "running" {
		log.Info().Int("pid", status.PID).Msg("Binary already running — treating as success")
		data := map[string]interface{}{
			"nodeName": nodeName,
			"pid":      status.PID,
			"status":   status.Status,
		}
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Binary already running on %s (PID %d)", nodeName, status.PID),
			Data:    data,
		}, nil
	}

	// ✅ Verify binary exists
	binaryPath := fmt.Sprintf("%s/finalvudatasim", node.BinaryDir)
	checkCmd := fmt.Sprintf("ls -la %s", binaryPath)
	if _, err := bc.sshExecWithOutput(node, checkCmd); err != nil {
		log.Error().Err(err).Msg("Binary not found on remote node")
		return response(false, "Binary not found on remote node"), err
	}

	// ✅ Build start command (background-safe)
	startCmd := fmt.Sprintf("cd %s && (nohup ./finalvudatasim >/dev/null 2>&1 &) && exit", node.BinaryDir)
	log.Info().Str("command", startCmd).Msg("Executing start command on remote node")

	err = bc.sshExec(node, startCmd)
	if err != nil {
		// 🔸 Handle timeout gracefully
		if strings.Contains(strings.ToLower(err.Error()), "timed out") ||
			strings.Contains(strings.ToLower(err.Error()), "timeout") {

			log.Warn().Err(err).Msg("SSH timeout detected — verifying if binary actually started")
			// continue to status check
		} else {
			log.Error().Err(err).Msg("SSH start command failed")
			return response(false, fmt.Sprintf("Failed to start binary: SSH error: %v", err)), err
		}
	} else {
		log.Info().Msg("Start command executed successfully — verifying status")
	}

	// ✅ Wait briefly and confirm status
	time.Sleep(2 * time.Second)
	newStatus, statusErr := bc.GetBinaryStatus(nodeName)
	if statusErr != nil {
		log.Warn().Err(statusErr).Msg("Status check failed — binary may still be starting")
	} else if newStatus.Status != "running" {
		log.Warn().Str("status", newStatus.Status).Msg("Binary not yet in running state — may be initializing")
	} else {
		log.Info().Int("pid", newStatus.PID).Msg("Binary confirmed running")
	}

	// ✅ Build response
	data := map[string]interface{}{
		"nodeName": nodeName,
		"status":   "starting",
	}
	if statusErr == nil && newStatus.Status == "running" {
		data["pid"] = newStatus.PID
		data["status"] = newStatus.Status
	}

	// ✅ Optional kill after timeout (only if PID known)
	if timeout > 0 {
		if pid, ok := data["pid"].(int); ok && pid > 0 {
			killCmd := fmt.Sprintf("(sleep %d; kill %d) >/dev/null 2>&1 &", timeout*60, pid)
			_ = bc.sshExec(node, killCmd)
		}
	}

	// ✅ Final response decision
	if statusErr == nil && newStatus.Status == "running" {
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Binary confirmed running on %s (PID %d)", nodeName, newStatus.PID),
			Data:    data,
		}, nil
	}

	// Fallback: start command succeeded, status unknown but not failed
	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary start command executed on %s (pending confirmation)", nodeName),
		Data:    data,
	}, nil
}



func (bc *BinaryControl) StopBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	logger.LogWithNode(nodeName, "binary_stop", "=== STOPPING BINARY ===", "info")
	logger.LogWithNode(nodeName, "binary_stop", "Reloading configuration", "info")

	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		logger.LogError(nodeName, "binary_stop", fmt.Sprintf("Failed to reload config: %v", err))
		return response(false, fmt.Sprintf("Failed to reload config: %v", err)), err
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()

	if !ok {
		logger.LogError(nodeName, "binary_stop", "Node not found in config")
		return response(false, fmt.Sprintf("Node %s not found", nodeName)), fmt.Errorf("node %s missing", nodeName)
	}
	if !node.Enabled {
		logger.LogError(nodeName, "binary_stop", "Node is disabled")
		return response(false, fmt.Sprintf("Node %s is disabled", nodeName)), fmt.Errorf("node %s disabled", nodeName)
	}

	logger.LogWithNode(nodeName, "binary_stop", "Checking current binary status", "info")
	status, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		logger.LogError(nodeName, "binary_stop", fmt.Sprintf("Status check failed: %v", err))
		return response(false, fmt.Sprintf("Failed to check binary status on node %s: %v", nodeName, err)), err
	}
	if status.Status != "running" {
		logger.LogWarning(nodeName, "binary_stop", fmt.Sprintf("Binary not running (status: %s)", status.Status))
		return response(false, fmt.Sprintf("Binary not running on node %s", nodeName)), fmt.Errorf("binary not running")
	}

	// Get detailed status to obtain the actual PID
	logger.LogWithNode(nodeName, "binary_stop", "Getting detailed binary status for PID", "info")
	detailedStatus, err := bc.getDetailedBinaryStatus(nodeName)
	if err != nil {
		logger.LogError(nodeName, "binary_stop", fmt.Sprintf("Failed to get detailed status: %v", err))
		return response(false, fmt.Sprintf("Failed to get binary PID on node %s: %v", nodeName, err)), err
	}
	if detailedStatus.Status != "running" {
		logger.LogWarning(nodeName, "binary_stop", fmt.Sprintf("Binary not running in detailed check (status: %s)", detailedStatus.Status))
		return response(false, fmt.Sprintf("Binary not running on node %s", nodeName)), fmt.Errorf("binary not running")
	}

	logger.LogWithNode(nodeName, "binary_stop", fmt.Sprintf("Binary is running with PID %d", detailedStatus.PID), "info")

	// Attempt graceful kill; if fails, force kill
	killCmd := fmt.Sprintf("kill %d", detailedStatus.PID)
	logger.LogWithNode(nodeName, "binary_stop", fmt.Sprintf("Attempting graceful kill with command: %s", killCmd), "info")
	if err := bc.sshExec(node, killCmd); err != nil {
		logger.LogWarning(nodeName, "binary_stop", "Graceful kill failed, attempting force kill")
		killCmd = fmt.Sprintf("kill -9 %d", detailedStatus.PID)
		logger.LogWithNode(nodeName, "binary_stop", fmt.Sprintf("Force killing with command: %s", killCmd), "info")
		if err := bc.sshExec(node, killCmd); err != nil {
			logger.LogError(nodeName, "binary_stop", fmt.Sprintf("Force kill also failed: %v", err))
			return response(false, fmt.Sprintf("Failed to stop binary on node %s: %v", nodeName, err)), err
		}
	} else {
		logger.LogWithNode(nodeName, "binary_stop", "Graceful kill command executed successfully", "info")
	}

	logger.LogWithNode(nodeName, "binary_stop", "Waiting 5 seconds for process to terminate", "info")
	time.Sleep(5 * time.Second)

	logger.LogWithNode(nodeName, "binary_stop", "Checking binary status after kill attempt", "info")
	newStatus, err := bc.GetBinaryStatus(nodeName)
	if err != nil {
		logger.LogWarning(nodeName, "binary_stop", fmt.Sprintf("Status check failed after kill: %v", err))
		return &BinaryControlResponse{
			Success: true,
			Message: fmt.Sprintf("Stop command sent to node %s, status check failed: %v", nodeName, err),
			Data:    map[string]string{"warning": "Binary may be stopped, status check failed"},
		}, nil
	}

	logger.LogWithNode(nodeName, "binary_stop", fmt.Sprintf("Post-kill status: %s", newStatus.Status), "info")

	// Check if binary was actually stopped
	if newStatus.Status == "running" {
		logger.LogError(nodeName, "binary_stop", fmt.Sprintf("Binary still running after kill attempt (PID: %d)", newStatus.PID))
		return response(false, fmt.Sprintf("Failed to stop binary on node %s - still running after kill attempt", nodeName)), fmt.Errorf("binary still running after kill")
	}

	logger.LogSuccess(nodeName, "binary_stop", "Binary successfully stopped")

	data := map[string]interface{}{
		"nodeName":    nodeName,
		"action":      "stop",
		"timeout":     timeout,
		"previousPID": detailedStatus.PID,
		"status":      newStatus,
	}

	logger.LogWithNode(nodeName, "binary_stop", "Stop operation completed successfully", "info")
	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Binary stopped successfully on node %s", nodeName),
		Data:    data,
	}, nil
}

func (bc *BinaryControl) StartMetricsBinary(nodeName string, timeout int) (*BinaryControlResponse, error) {
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return response(false, fmt.Sprintf("Failed to reload config: %v", err)), err
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()
	
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
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return response(false, fmt.Sprintf("Failed to reload config: %v", err)), err
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()
	
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
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return response(false, fmt.Sprintf("Failed to reload config: %v", err)), err
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()
	
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
	if bc == nil {
		return nil, fmt.Errorf("BinaryControl instance is nil")
	}
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return nil, fmt.Errorf("failed to reload config: %v", err)
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()

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

	// Basic status check - just determine if running or stopped
	output, err := bc.sshExecWithOutput(node, "pgrep -f './finalvudatasim'")
	if err != nil || output == "" {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// If running, return basic status without detailed metrics
	// Detailed metrics will be collected separately only when needed
	return &BinaryStatus{
		NodeName:    nodeName,
		Status:      "running",
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// getDetailedBinaryStatus collects detailed metrics for a running binary
func (bc *BinaryControl) getDetailedBinaryStatus(nodeName string) (*BinaryStatus, error) {
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return nil, fmt.Errorf("failed to reload config: %v", err)
	}

	// Lock for reading to protect the config
	bc.configMutex.RLock()
	node, ok := bc.nodesConfig.Nodes[nodeName]
	bc.configMutex.RUnlock()

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

	// Get PID first
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

	// Validate PID is valid (> 0)
	if pid <= 0 {
		return &BinaryStatus{
			NodeName:    nodeName,
			Status:      "stopped",
			ProcessInfo: "Invalid PID detected (PID <= 0)",
			LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		}, fmt.Errorf("invalid PID %d", pid)
	}

	// Collect detailed metrics only for running processes
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
	if bc == nil {
		return &BinaryControlResponse{
			Success: false,
			Message: "BinaryControl instance is nil",
		}, fmt.Errorf("BinaryControl instance is nil")
	}
	// Reload configuration to ensure we have the latest nodes
	if err := bc.LoadNodesConfig(); err != nil {
		return &BinaryControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to reload config: %v", err),
		}, err
	}

	enabledNodes := bc.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		return &BinaryControlResponse{
			Success: true,
			Message: "No enabled nodes found",
			Data:    []BinaryStatus{},
		}, nil
	}

	// Create a channel to collect results from goroutines
	type statusResult struct {
		nodeName string
		status   *BinaryStatus
		err      error
	}
	resultChan := make(chan statusResult, len(enabledNodes))

	// Launch goroutines to get status for each node
	for nodeName := range enabledNodes {
		go func(nodeName string) {
			// First, get basic status (running/stopped) without detailed metrics
			status, err := bc.GetBinaryStatus(nodeName)
			if err != nil {
				resultChan <- statusResult{nodeName: nodeName, status: status, err: err}
				return
			}

			// Only collect detailed metrics (CPU, memory, process info) if binary is running
			if status.Status == "running" {
				// Re-run status check with detailed metrics collection
				detailedStatus, detailedErr := bc.getDetailedBinaryStatus(nodeName)
				if detailedErr != nil {
					// If detailed collection fails, use basic status
					resultChan <- statusResult{nodeName: nodeName, status: status, err: nil}
				} else {
					resultChan <- statusResult{nodeName: nodeName, status: detailedStatus, err: nil}
				}
			} else {
				// For stopped binaries, return basic status without attempting SSH for details
				resultChan <- statusResult{nodeName: nodeName, status: status, err: nil}
			}
		}(nodeName)
	}

	// Collect results
	var statuses []BinaryStatus
	for i := 0; i < len(enabledNodes); i++ {
		result := <-resultChan
		if result.err != nil {
			log.Printf("Failed to get status for node %s: %v", result.nodeName, result.err)
			statuses = append(statuses, BinaryStatus{
				NodeName:    result.nodeName,
				Status:      "error",
				ProcessInfo: fmt.Sprintf("Status check failed: %v", result.err),
				LastChecked: time.Now().Format("2006-01-02 15:04:05"),
			})
		} else {
			statuses = append(statuses, *result.status)
		}
	}

	return &BinaryControlResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved status for %d nodes", len(statuses)),
		Data:    statuses,
	}, nil
}

func (bc *BinaryControl) StartAllBinaries(timeout int) (*BulkBinaryResponse, error) {
	logger.LogWithNode("all", "bulk_start", "=== BULK START OPERATION STARTED ===", "info")

	// Reload config before starting
	if err := bc.LoadNodesConfig(); err != nil {
		logger.LogError("all", "bulk_start", fmt.Sprintf("Failed to reload config: %v", err))
		return &BulkBinaryResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to reload config: %v", err),
		}, err
	}

	enabledNodes := bc.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		logger.LogError("all", "bulk_start", "No enabled nodes found")
		return &BulkBinaryResponse{
			Success: false,
			Message: "No enabled nodes found",
		}, fmt.Errorf("no enabled nodes found")
	}

	totalNodes := len(enabledNodes)
	logger.LogWithNode("all", "bulk_start", fmt.Sprintf("Launching concurrent start operations on %d nodes", totalNodes), "info")

	resultChan := make(chan BulkBinaryResult, totalNodes)
	var wg sync.WaitGroup

	// Launch goroutines for all enabled nodes
	for nodeName := range enabledNodes {
		wg.Add(1)
		go func(nodeName string) {
			defer wg.Done()
			apiResp, err := bc.StartBinary(nodeName, timeout)

			result := BulkBinaryResult{
				NodeName: nodeName,
				Success:  err == nil && apiResp != nil && apiResp.Success,
				Message:  "",
			}

			if err != nil {
				result.Message = fmt.Sprintf("SSH start command failed: %v", err)
				logger.LogError(nodeName, "bulk_start", fmt.Sprintf("Failed to start binary: %v", err))
			} else {
				result.Message = apiResp.Message
				if apiResp.Success {
					if data, ok := apiResp.Data.(map[string]interface{}); ok {
						if pid, ok := data["pid"].(float64); ok {
							result.PID = int(pid)
						}
					}
					logger.LogSuccess(nodeName, "bulk_start", fmt.Sprintf("Binary started successfully with PID %d", result.PID))
				} else {
					logger.LogWarning(nodeName, "bulk_start", fmt.Sprintf("Start failed: %s", apiResp.Message))
				}
			}

			// Non-blocking send to avoid panic if channel closed
			select {
			case resultChan <- result:
			default:
				logger.LogError(nodeName, "bulk_start", "Failed to send result to channel - closed or full")
			}
		}(nodeName)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make([]BulkBinaryResult, 0, totalNodes)
	var logs []string
	successful := 0

	timeoutChan := time.After(10 * time.Second)
	collectDone := make(chan bool)

	go func() {
		for res := range resultChan {
			results = append(results, res)
			if res.Success {
				successful++
				logs = append(logs, fmt.Sprintf("%s - SUCCESS: %s - %s",
					time.Now().Format("2006-01-02 15:04:05"), res.NodeName, res.Message))
			} else {
				logs = append(logs, fmt.Sprintf("%s - FAILED: %s - %s",
					time.Now().Format("2006-01-02 15:04:05"), res.NodeName, res.Message))
			}
		}
		collectDone <- true
	}()

	select {
	case <-collectDone:
		// All results collected
	case <-timeoutChan:
		logger.LogWarning("all", "bulk_start", "Timeout while collecting binary start results")
	}

message := fmt.Sprintf("Binaries started successfully on %d/%d nodes", successful, totalNodes)
if successful < totalNodes {
	message += fmt.Sprintf(" (%d failed)", totalNodes-successful)
}

logger.LogWithNode("all", "bulk_start", "=== BULK START OPERATION COMPLETED ===", "info")
logger.LogWithNode("all", "bulk_start", message, "info")

	return &BulkBinaryResponse{
		Success: successful > 0,
		Message: message,
		Data: &BulkBinaryData{
			TotalNodes: totalNodes,
			Successful: successful,
			Failed:     totalNodes - successful,
			Results:    results,
			Logs:       logs,
		},
	}, nil
}



func (bc *BinaryControl) StopAllBinaries(timeout int) (*BulkBinaryResponse, error) {
	logger.LogWithNode("all", "bulk_stop", "=== BULK STOP OPERATION STARTED ===", "info")
	logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Stopping binaries on all enabled nodes with timeout %d minutes", timeout), "info")

	// Reload configuration to ensure we have the latest nodes
	logger.LogWithNode("all", "bulk_stop", "Loading nodes configuration...", "info")
	if err := bc.LoadNodesConfig(); err != nil {
		logger.LogError("all", "bulk_stop", fmt.Sprintf("Failed to reload config: %v", err))
		return &BulkBinaryResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to reload config: %v", err),
		}, err
	}
	logger.LogWithNode("all", "bulk_stop", "Successfully loaded nodes configuration", "info")

	enabledNodes := bc.GetEnabledNodes()
	logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Found %d enabled nodes: %v", len(enabledNodes), getNodeNames(enabledNodes)), "info")
	if len(enabledNodes) == 0 {
		logger.LogError("all", "bulk_stop", "No enabled nodes found")
		return &BulkBinaryResponse{
			Success: false,
			Message: "No enabled nodes found",
		}, fmt.Errorf("no enabled nodes found")
	}

	totalNodes := len(enabledNodes)
	logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Starting bulk binary stop operation on %d nodes", totalNodes), "info")

	// Channel to collect results from goroutines
	resultChan := make(chan BulkBinaryResult, totalNodes)

	// Stop binaries concurrently on all nodes
	logger.LogWithNode("all", "bulk_stop", "Launching concurrent stop operations...", "info")
	for nodeName, node := range enabledNodes {
		go func(nodeName string, node NodeConfig) {
			logger.LogWithNode(nodeName, "bulk_stop", "=== STOPPING NODE ===", "info")
			logger.LogWithNode(nodeName, "bulk_stop", "Stopping binary operation", "info")

			response, err := bc.StopBinary(nodeName, timeout)

			result := BulkBinaryResult{
				NodeName: nodeName,
				Success:  response != nil && response.Success,
				Message:  response.Message,
			}

			if err != nil {
				logger.LogError(nodeName, "bulk_stop", fmt.Sprintf("Error: %v", err))
				result.Success = false
				result.Message = fmt.Sprintf("Error: %v", err)
			} else if response.Success {
				logger.LogSuccess(nodeName, "bulk_stop", "Binary stopped successfully")
			} else {
				logger.LogWarning(nodeName, "bulk_stop", fmt.Sprintf("Binary stop failed - %s", response.Message))
			}

			logger.LogWithNode(nodeName, "bulk_stop", "Operation completed", "info")
			resultChan <- result
		}(nodeName, node)
	}

	// Collect results with timeout
	logger.LogWithNode("all", "bulk_stop", "Waiting for all operations to complete...", "info")
	collectionTimeout := time.After(60 * time.Second) // Overall timeout for collection
	collected := 0

	var results []BulkBinaryResult
	var logs []string
	successful := 0
	failed := 0

	for collected < totalNodes {
		select {
		case result := <-resultChan:
			logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Received result for node %s: Success=%v, Message=%s", result.NodeName, result.Success, result.Message), "info")
			results = append(results, result)

			if result.Success {
				successful++
				logs = append(logs, fmt.Sprintf("%s - SUCCESS: %s - %s", time.Now().Format("2006-01-02 15:04:05"), result.NodeName, result.Message))
			} else {
				failed++
				logs = append(logs, fmt.Sprintf("%s - FAILED: %s - %s", time.Now().Format("2006-01-02 15:04:05"), result.NodeName, result.Message))
			}
			collected++
		case <-collectionTimeout:
			logger.LogWarning("all", "bulk_stop", fmt.Sprintf("Timeout while collecting results, collected %d/%d results", collected, totalNodes))
			// Add failed results for uncollected nodes
			for i := collected; i < totalNodes; i++ {
				results = append(results, BulkBinaryResult{
					NodeName: fmt.Sprintf("unknown_%d", i),
					Success:  false,
					Message:  "Timeout while collecting result",
				})
				failed++
				logs = append(logs, fmt.Sprintf("%s - FAILED: unknown_%d - Timeout while collecting result", time.Now().Format("2006-01-02 15:04:05"), i))
			}
			break
		}
	}

	close(resultChan)

	data := &BulkBinaryData{
		TotalNodes: totalNodes,
		Successful: successful,
		Failed:     failed,
		Results:    results,
		Logs:       logs,
	}

	message := fmt.Sprintf("Stopped binaries on %d/%d nodes", successful, totalNodes)
	if failed > 0 {
		message += fmt.Sprintf(" (%d failed)", failed)
	}

	logger.LogWithNode("all", "bulk_stop", "=== BULK STOP OPERATION COMPLETED ===", "info")
	logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Final result: %s", message), "info")
	logger.LogWithNode("all", "bulk_stop", fmt.Sprintf("Successful: %d, Failed: %d", successful, failed), "info")

	return &BulkBinaryResponse{
		Success: successful > 0, // Success if at least one node succeeded
		Message: message,
		Data:    data,
	}, nil
}

func (bc *BinaryControl) sshExec(node NodeConfig, command string) error {
	if bc == nil {
		logger.LogError("binary_control", "ssh_exec", "BinaryControl instance is nil")
		return fmt.Errorf("BinaryControl instance is nil")
	}

	// Acquire semaphore to limit concurrent SSH operations
	bc.sshSemaphore <- struct{}{}
	defer func() { <-bc.sshSemaphore }()

	args := []string{
		"-i", node.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		node.Host,
		command,
	}

	// Timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", args...)
	err := cmd.Run()

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		// ⚠️ Timeout often happens for background (& nohup) commands — treat as possible success
		logger.LogWarning(node.Host, "ssh_exec", fmt.Sprintf("SSH command timed out after 30 seconds (possible background execution): %s", command))
		return fmt.Errorf("SSH command timed out after 30 seconds (possible background execution): %s", command)
	}

	// Handle other errors
	if err != nil {
		// Detect EOF or exit 255 cases (remote hang-up) as soft success for background runs
		errMsg := err.Error()
		if strings.Contains(errMsg, "exit status 255") || strings.Contains(errMsg, "EOF") {
			logger.LogWarning(node.Host, "ssh_exec", fmt.Sprintf("SSH connection closed early (possible background execution): %s", command))
			return fmt.Errorf("SSH connection closed early (possible background execution): %s", command)
		}
		logger.LogError(node.Host, "ssh_exec", fmt.Sprintf("SSH command failed: %v", err))
		return fmt.Errorf("SSH command failed: %v", err)
	}

	logger.LogSuccess(node.Host, "ssh_exec", fmt.Sprintf("SSH command executed successfully: %s", command))
	return nil
}


func (bc *BinaryControl) sshExecWithOutput(node NodeConfig, command string) (string, error) {
	if bc == nil {
		logger.LogError("binary_control", "ssh_exec", "BinaryControl instance is nil")
		return "", fmt.Errorf("BinaryControl instance is nil")
	}

	// Acquire semaphore to limit concurrent SSH operations
	bc.sshSemaphore <- struct{}{}
	defer func() { <-bc.sshSemaphore }()

	args := []string{
		"-i", node.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		node.Host,
		command,
	}

	// Shorter timeout for quick info commands (ls, ps, cat)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		logger.LogError(node.Host, "ssh_exec", fmt.Sprintf("SSH command timed out after 10 seconds: %s", command))
		return "", fmt.Errorf("SSH command timed out after 10 seconds: %s", command)
	}

	// Handle error but still return partial output for diagnostics
	if err != nil {
		outStr := strings.TrimSpace(string(output))
		errMsg := fmt.Sprintf("SSH command failed: %v", err)
		if outStr != "" {
			errMsg += " | output: " + outStr
		}
		logger.LogError(node.Host, "ssh_exec", errMsg)
		return outStr, fmt.Errorf(errMsg)
	}

	logger.LogSuccess(node.Host, "ssh_exec", fmt.Sprintf("SSH command executed successfully: %s", command))
	return strings.TrimSpace(string(output)), nil
}


func response(success bool, message string) *BinaryControlResponse {
	return &BinaryControlResponse{
		Success: success,
		Message: message,
	}
}

// Helper function to get node names for logging
func getNodeNames(nodes map[string]NodeConfig) []string {
	names := make([]string, 0, len(nodes))
	for name := range nodes {
		names = append(names, name)
	}
	return names
}

