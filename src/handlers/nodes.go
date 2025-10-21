package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/node_control"

	"github.com/gorilla/mux"
)

func HandleAPINodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	nodes := NodeManager.GetNodes()
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

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    nodeList,
	})
}

func HandleAPINodeActions(w http.ResponseWriter, r *http.Request) {
	// Extract node name from URL path
	vars := mux.Vars(r)
	nodeName := vars["name"]

	if nodeName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	switch r.Method {
	case http.MethodPost:
		HandleCreateNode(w, r, nodeName)
	case http.MethodPut:
		HandleUpdateNode(w, r, nodeName)
	case http.MethodDelete:
		HandleDeleteNode(w, r, nodeName)
	default:
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

func HandleCreateNode(w http.ResponseWriter, r *http.Request, nodeName string) {
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
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	// Debug log the received data
	fmt.Printf("DEBUG: Received node data - Host: %s, User: %s, KeyPath: %s, ConfDir: %s, BinaryDir: %s\n",
		nodeData.Host, nodeData.User, nodeData.KeyPath, nodeData.ConfDir, nodeData.BinaryDir)

	addNodeReq := node_control.AddNodeRequest{
		Name:        nodeName,
		Host:        nodeData.Host,
		User:        nodeData.User,
		KeyPath:     nodeData.KeyPath,
		ConfDir:     nodeData.ConfDir,
		BinaryDir:   nodeData.BinaryDir,
		Description: nodeData.Description,
		Enabled:     nodeData.Enabled,
	}

	err := NodeManager.AddNode(addNodeReq)

	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	SendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s created successfully", nodeName),
	})
}

func HandleUpdateNode(w http.ResponseWriter, r *http.Request, nodeName string) {
	var nodeData struct {
		Enabled *bool `json:"enabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	if nodeData.Enabled != nil {
		if *nodeData.Enabled {
			err := NodeManager.EnableNode(nodeName)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
			// Start node_metrics_api binary
			_, err = BinaryControl.StartMetricsBinary(nodeName, 10)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: "Node enabled, but failed to start node_metrics_api: " + err.Error(),
				})
				return
			}
		} else {
			err := NodeManager.DisableNode(nodeName)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
			// Stop node_metrics_api binary
			_, err = BinaryControl.StopMetricsBinary(nodeName, 10)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: "Node disabled, but failed to stop node_metrics_api: " + err.Error(),
				})
				return
			}
		}
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Node updated successfully",
	})
}

func HandleDeleteNode(w http.ResponseWriter, r *http.Request, nodeName string) {
	err := NodeManager.RemoveNode(nodeName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s deleted successfully", nodeName),
	})
}

func HandleAPIClusterSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    NodeManager.GetClusterSettings(),
		})
	case http.MethodPut:
		var settings node_control.ClusterSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		err := NodeManager.UpdateClusterSettings(settings)
		if err != nil {
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Cluster settings updated successfully",
		})
	default:
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

func HandleAPIGetClusterMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := clickhouse.GetClusterNodeMetrics()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch cluster metrics: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Cluster metrics retrieved successfully",
		Data:    metrics,
	})
}

func HandleAPIDebugMetricsBinary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Extract node name from URL path
	vars := mux.Vars(r)
	nodeName := vars["name"]

	if nodeName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	debugInfo, err := BinaryControl.DebugMetricsBinary(nodeName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to debug metrics binary: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Debug information retrieved for node %s", nodeName),
		Data:    debugInfo.Data,
	})
}
