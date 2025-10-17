package node_control

/*
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func handleDeleteNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
	err := nm.RemoveNode(nodeName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s deleted successfully", nodeName),
	})
}



*/

// sendJSONResponse sends a JSON response

/*
func handleCreateNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
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
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	err := nm.AddNode(nodeName, nodeData.Host, nodeData.User, nodeData.KeyPath,
		nodeData.ConfDir, nodeData.BinaryDir, nodeData.Description, nodeData.Enabled)

	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s created successfully", nodeName),
	})
}

func handleUpdateNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
	var nodeData struct {
		Enabled *bool `json:"enabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	if nodeData.Enabled != nil {
		if *nodeData.Enabled {
			err := nm.EnableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		} else {
			err := nm.DisableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		}
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s updated successfully", nodeName),
	})
}

/*
// handleAPINodes handles GET /api/nodes (list all nodes)
func handleAPINodes(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	nodes := nm.GetNodes()
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

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    nodeList,
	})
}

// handleAPINodeActions handles POST/PUT/DELETE /api/nodes/{name}
func handleAPINodeActions(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	// Extract node name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	nodeName := parts[0]

	switch r.Method {
	case http.MethodPost:
		handleCreateNode(w, r, nm, nodeName)
	case http.MethodPut:
		handleUpdateNode(w, r, nm, nodeName)
	case http.MethodDelete:
		handleDeleteNode(w, r, nm, nodeName)
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// handleAPIClusterSettings handles GET/PUT /api/cluster-settings
func handleAPIClusterSettings(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nm.nodesConfig.ClusterSettings,
		})
	case http.MethodPut:
		var settings ClusterSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		nm.nodesConfig.ClusterSettings = settings
		err := nm.SaveNodesConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Cluster settings updated successfully",
		})
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// handleAPIAppConfig handles GET/PUT /api/config
func handleAPIAppConfig(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nm.appConfig,
		})
	case http.MethodPut:
		var config AppConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		nm.appConfig = config
		err := nm.SaveAppConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Application configuration updated successfully",
		})
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

*/
