package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"vuDataSim/src/node_control"

	"github.com/gorilla/mux"
)

func GetDashboardData(w http.ResponseWriter, r *http.Request) {
	AppState.Mutex.RLock()
	defer AppState.Mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data:    AppState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	AppState.Mutex.RLock()
	defer AppState.Mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"version":   AppVersion,
			"timestamp": time.Now(),
			"uptime":    time.Since(AppState.StartTime).String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var metrics node_control.NodeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	AppState.Mutex.Lock()
	defer AppState.Mutex.Unlock()

	if node, exists := AppState.NodeData[nodeID]; exists {
		node.EPS = metrics.EPS
		node.KafkaLoad = metrics.KafkaLoad
		node.CHLoad = metrics.CHLoad
		node.CPU = metrics.CPU
		node.Memory = metrics.Memory
		node.LastUpdate = time.Now()

		response := APIResponse{
			Success: true,
			Message: fmt.Sprintf("Node %s metrics updated successfully", nodeID),
			Data:    node,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// Broadcast update
		// Broadcast update
		go AppState.BroadcastUpdate()
	} else {
		response := APIResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s not found", nodeID),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	}
}
