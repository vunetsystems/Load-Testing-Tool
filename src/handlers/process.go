package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vuDataSim/src/node_control"
)

// handleAPIGetProcessMetrics handles GET /api/process/metrics
func HandleAPIGetProcessMetrics(w http.ResponseWriter, r *http.Request) {
	enabledNodes := NodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "No enabled nodes found",
			Data:    []ProcessMetrics{},
		})
		return
	}

	var allMetrics []ProcessMetrics
	for nodeName, nodeConfig := range enabledNodes {
		metrics := CollectProcessMetricsForNode(nodeName, &nodeConfig)
		allMetrics = append(allMetrics, metrics)
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved process metrics for %d nodes", len(allMetrics)),
		Data:    allMetrics,
	})
}

// collectProcessMetricsForNode collects finalvudatasim process metrics for a specific node via SSH
func CollectProcessMetricsForNode(nodeName string, nodeConfig *node_control.NodeConfig) ProcessMetrics {
	metrics := ProcessMetrics{
		NodeID:    nodeName,
		Timestamp: time.Now(),
	}

	// Use SSH to collect process metrics from the remote node
	// Use the same SSH execution method as used in node_manager.go

	// Check if finalvudatasim process is running using SSHExecWithOutput
	output, err := NodeManager.SSHExecWithOutput(*nodeConfig, "pgrep -f finalvudatasim")
	if err != nil || output == "" {
		metrics.Running = false
		return metrics
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		metrics.Running = false
		return metrics
	}

	pidStr := lines[0]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		metrics.Error = fmt.Sprintf("Invalid PID: %s", pidStr)
		return metrics
	}

	metrics.Running = true
	metrics.PID = pid

	// Get process start time
	startTimeOut, err := NodeManager.SSHExecWithOutput(*nodeConfig, fmt.Sprintf("ps -p %s -o lstart=", pidStr))
	if err == nil && startTimeOut != "" {
		metrics.StartTime = strings.TrimSpace(startTimeOut)
	}

	// Get CPU and memory usage
	psOut, err := NodeManager.SSHExecWithOutput(*nodeConfig, fmt.Sprintf("ps -p %s -o %%cpu,rss,cmd", pidStr))
	if err == nil && psOut != "" {
		psFields := strings.Fields(psOut)
		if len(psFields) >= 3 {
			metrics.CPUPercent, _ = strconv.ParseFloat(psFields[0], 64)
			memKB, _ := strconv.ParseFloat(psFields[1], 64)
			metrics.MemMB = memKB / 1024.0
			metrics.Cmdline = strings.Join(psFields[2:], " ")
		}
	}

	return metrics
}
