package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"vuDataSim/src/logger"
)

func HandleAPIGetAllBinaryStatus(w http.ResponseWriter, r *http.Request) {
	response, err := BinaryControl.GetAllBinaryStatuses()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
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
	SendJSONResponse(w, http.StatusOK, apiResponse)
}

func HandleAPIGetBinaryStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	status, err := BinaryControl.GetBinaryStatus(nodeName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get binary status for node %s: %v", nodeName, err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

// handleAPIStartBinary handles POST /api/binary/start/{node}
func HandleAPIStartBinary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
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

	response, err := BinaryControl.StartBinary(nodeName, timeout)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
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
	SendJSONResponse(w, statusCode, apiResponse)
}

// HandleAPIStartAllBinaries handles POST /api/binary/start-all
func HandleAPIStartAllBinaries(w http.ResponseWriter, r *http.Request) {
	// Parse timeout from query parameters (default: 30 seconds)
	timeout := 30
	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	response, err := BinaryControl.StartAllBinaries(timeout)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start all binaries: %v", err),
		})
		return
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}
	SendJSONResponse(w, http.StatusOK, apiResponse)
}

// HandleAPIStopAllBinaries handles POST /api/binary/stop-all
func HandleAPIStopAllBinaries(w http.ResponseWriter, r *http.Request) {
	// Parse timeout from query parameters (default: 30 seconds)
	timeout := 30
	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	response, err := BinaryControl.StopAllBinaries(timeout)
	if err != nil {
		logger.LogError("all", "binary_stop", fmt.Sprintf("Failed to stop all binaries: %v", err))
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop all binaries: %v", err),
		})
		return
	}

	// Verify all binaries are actually stopped
	allStopped := true
	if response.Data != nil {
		bulkData := response.Data
		for _, result := range bulkData.Results {
			if result.Success {
				// Double-check by getting status for this node
				status, statusErr := BinaryControl.GetBinaryStatus(result.NodeName)
				if statusErr != nil || status.Status == "running" {
					allStopped = false
					logger.LogWarning(result.NodeName, "binary_stop", fmt.Sprintf("Binary still running after stop operation: status=%s, error=%v", status.Status, statusErr))
					break
				}
			} else {
				allStopped = false
				logger.LogWarning(result.NodeName, "binary_stop", "Stop operation reported failure")
				break
			}
		}
	}

	if !allStopped {
		logger.LogWarning("all", "binary_stop", "Some binaries still running, attempting second stop operation")
		// Retry stop operation once more
		retryResponse, retryErr := BinaryControl.StopAllBinaries(timeout)
		if retryErr == nil && retryResponse != nil {
			// Verify again after retry
			allStopped = true
			if retryResponse.Data != nil {
				retryBulkData := retryResponse.Data
				for _, result := range retryBulkData.Results {
					if result.Success {
						status, statusErr := BinaryControl.GetBinaryStatus(result.NodeName)
						if statusErr != nil || status.Status == "running" {
							allStopped = false
							logger.LogError(result.NodeName, "binary_stop", fmt.Sprintf("Binary still running after retry: status=%s, error=%v", status.Status, statusErr))
							break
						}
					} else {
						allStopped = false
						logger.LogError(result.NodeName, "binary_stop", "Retry stop operation also failed")
						break
					}
				}
			}
			if allStopped {
				logger.LogSuccess("all", "binary_stop", "All binaries successfully stopped after retry and verified dead")
				response.Message = "completed"
				response.Data = retryResponse.Data // Update with retry results
			} else {
				logger.LogError("all", "binary_stop", "Some binaries still running after retry operation")
				response.Message = "partial failure - some binaries still running"
			}
		} else {
			logger.LogError("all", "binary_stop", fmt.Sprintf("Retry stop operation failed: %v", retryErr))
			response.Message = "stop operation failed with retry"
		}
	} else {
		logger.LogSuccess("all", "binary_stop", "All binaries successfully stopped and verified dead")
		response.Message = "completed"
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}
	SendJSONResponse(w, http.StatusOK, apiResponse)
}

// handleAPIStopBinary handles POST /api/binary/stop/{node}
func HandleAPIStopBinary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["node"]

	if nodeName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
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

	response, err := BinaryControl.StopBinary(nodeName, timeout)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
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
	SendJSONResponse(w, statusCode, apiResponse)
}
