package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"
)

// SSHHandler handles SSH-related HTTP requests
type SSHHandler struct {
	NodeManager *node_control.NodeManager
}

// NewSSHHandler creates a new SSH handler with the given node manager
func NewSSHHandler(NodeManager *node_control.NodeManager) *SSHHandler {
	return &SSHHandler{
		NodeManager: NodeManager,
	}
}

// GetSSHStatus handles GET /api/ssh/status
func (h *SSHHandler) GetSSHStatus(w http.ResponseWriter, r *http.Request) {
	enabledNodes := h.NodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "No enabled nodes found",
			Data:    []SSHStatus{},
		})
		return
	}

	var allStatuses []SSHStatus
	for nodeName, nodeConfig := range enabledNodes {
		status := h.CheckSSHConnectivity(nodeName, nodeConfig)
		allStatuses = append(allStatuses, status)
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved SSH status for %d nodes", len(allStatuses)),
		Data:    allStatuses,
	})
}

// checkSSHConnectivity checks SSH connectivity for a single node
func (h *SSHHandler) CheckSSHConnectivity(nodeName string, nodeConfig node_control.NodeConfig) SSHStatus {
	status := SSHStatus{
		NodeName:    nodeName,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Test SSH connection with a simple command
	testCmd := "echo 'SSH connection test'"
	output, err := h.NodeManager.SSHExecWithOutput(nodeConfig, testCmd)

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

// HandleAPIGetSSHStatus handles GET /api/ssh/status
func HandleAPIGetSSHStatus(w http.ResponseWriter, r *http.Request) {
	// Create SSH handler instance
	sshHandler := NewSSHHandler(NodeManager)

	// Delegate to the SSHHandler's GetSSHStatus method
	sshHandler.GetSSHStatus(w, r)
}
