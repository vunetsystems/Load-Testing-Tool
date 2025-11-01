package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
	"vuDataSim/src/logger"
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
			// Stop node_metrics_api binary before disabling node
			_, err := BinaryControl.StopMetricsBinary(nodeName, 10)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: "Failed to stop node_metrics_api before disabling node: " + err.Error(),
				})
				return
			}
			err = NodeManager.DisableNode(nodeName)
			if err != nil {
				SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
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
	// Stop node_metrics_api binary before removing node
	logger.Info().Str("node", nodeName).Msg("Stopping node_metrics_api binary before node removal")
	_, err := BinaryControl.StopMetricsBinary(nodeName, 10)
	if err != nil {
		logger.Warn().Err(err).Str("node", nodeName).Msg("Failed to stop node_metrics_api binary - continuing with removal")
		// Don't fail the entire operation if metrics binary stop fails
	} else {
		logger.LogSuccess(nodeName, "node_control", "node_metrics_api binary stopped successfully")
	}

	// Stop finalvudatasim binary before removing node
	logger.Info().Str("node", nodeName).Msg("Stopping finalvudatasim binary before node removal")
	_, err = BinaryControl.StopBinary(nodeName, 10)
	if err != nil {
		logger.Warn().Err(err).Str("node", nodeName).Msg("Failed to stop finalvudatasim binary - continuing with removal")
		// Don't fail the entire operation if binary stop fails
	} else {
		logger.LogSuccess(nodeName, "node_control", "finalvudatasim binary stopped successfully")
	}

	// Now remove the node from configuration
	err = NodeManager.RemoveNode(nodeName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s deleted successfully (binaries stopped and node removed from configuration)", nodeName),
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

// func HandleAPIGetClusterMetrics(w http.ResponseWriter, r *http.Request) {
// 	metrics, err := clickhouse.GetClusterNodeMetrics()
// 	if err != nil {
// 		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
// 			Success: false,
// 			Message: fmt.Sprintf("Failed to fetch cluster metrics: %v", err),
// 		})
// 		return
// 	}

// 	SendJSONResponse(w, http.StatusOK, APIResponse{
// 		Success: true,
// 		Message: "Cluster metrics retrieved successfully",
// 		Data:    metrics,
// 	})
// }

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

func HandleAPIAddAllClusterNodes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	logger.Info().Msg("🧩 Starting HandleAPIAddAllClusterNodes request")

	if r.Method != http.MethodPost {
		logger.Warn().Msg("Invalid HTTP method used")
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	logger.Info().Msg("✅ HTTP Method Check passed")

	// Step 2: Fetch all cluster nodes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "get", "nodes", "-o", "wide")
	output, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to execute kubectl command")
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to execute kubectl: %v", err),
		})
		return
	}

	logger.Info().Msg("✅ Successfully fetched kubectl nodes output")

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		logger.Error().Msg("No nodes found in kubectl output")
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "No nodes found in kubectl output",
		})
		return
	}

	nodeLines := lines[1:]
	fetchedNodes := []map[string]string{}
	addedNodes := []string{}

	type NodeResult struct {
		Name    string `json:"name"`
		IP      string `json:"ip"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	nodeResults := []NodeResult{}

	logger.Info().Msg("🔍 Parsing kubectl output for node details")

	// Step 3: Load app config once
	appConfig, err := NodeManager.LoadAppConfig()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load app config")
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load app config: %v", err),
		})
		return
	}

	logger.Info().Msg("✅ Loaded app configuration")

	currentNodeIP := appConfig.Network.CurrentNodeIP

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, line := range nodeLines {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		nodeName := fields[0]
		nodeRole := fields[2]
		nodeIP := fields[5]

		fetchedNodes = append(fetchedNodes, map[string]string{
			"name": nodeName,
			"ip":   nodeIP,
			"role": nodeRole,
		})

		if nodeIP == currentNodeIP {
			logger.Info().Str("node", nodeName).Msg("Skipping current node")
			continue
		}

		addNodeReq := node_control.AddNodeRequest{
			Name:        nodeName,
			Host:        nodeIP,
			User:        appConfig.Network.RemoteUser,
			KeyPath:     appConfig.Paths.RemoteSSHKey,
			ConfDir:     appConfig.Paths.RemoteConfDir,
			BinaryDir:   appConfig.Paths.RemoteBinaryDir,
			Description: nodeRole,
			Enabled:     true,
		}

		wg.Add(1)
		go func(req node_control.AddNodeRequest) {
			defer wg.Done()

			logger.Info().Str("node", req.Name).Msg("🚀 Starting node addition process")

			err := NodeManager.AddNode(req)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				logger.Error().Err(err).Str("node", req.Name).Msg("❌ Failed to add node")
				nodeResults = append(nodeResults, NodeResult{
					Name:    req.Name,
					IP:      req.Host,
					Status:  "failed",
					Message: err.Error(),
				})
				return
			}

			logger.Info().Str("node", req.Name).Msg("✅ Node added and binaries started successfully")

			addedNodes = append(addedNodes, req.Name)
			nodeResults = append(nodeResults, NodeResult{
				Name:    req.Name,
				IP:      req.Host,
				Status:  "success",
				Message: "Node added and started successfully",
			})
		}(addNodeReq)
	}

	wg.Wait()
	duration := time.Since(startTime)

	logger.Info().
		Int("total_fetched", len(fetchedNodes)).
		Int("total_added", len(addedNodes)).
		Dur("duration", duration).
		Msg("🏁 Completed adding all cluster nodes")

	response := map[string]interface{}{
		"fetched_nodes": fetchedNodes,
		"node_status":   nodeResults,
		"added_nodes":   addedNodes,
		"current_node":  currentNodeIP,
		"duration_sec":  duration.Seconds(),
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Processed %d nodes (added %d successfully)", len(fetchedNodes), len(addedNodes)),
		Data:    response,
	})
}
