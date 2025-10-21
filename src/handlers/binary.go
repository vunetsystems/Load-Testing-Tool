package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
