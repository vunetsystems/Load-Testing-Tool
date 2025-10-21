package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"vuDataSim/src/logger"
)

const (
	ContentTypeHeader = "Content-Type"
	ApplicationJSON   = "application/json"
)

func StartSimulation(w http.ResponseWriter, r *http.Request) {
	var config SimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	AppState.Mutex.Lock()
	defer AppState.Mutex.Unlock()

	if AppState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "Simulation is already running",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate configuration
	if config.TargetEPS < 1 || config.TargetEPS > 100000 {
		response := APIResponse{
			Success: false,
			Message: "Target EPS must be between 1 and 100,000",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update state
	AppState.IsSimulationRunning = true
	AppState.CurrentProfile = config.Profile
	AppState.TargetEPS = config.TargetEPS
	AppState.TargetKafka = config.TargetKafka
	AppState.TargetClickHouse = config.TargetClickHouse
	AppState.StartTime = time.Now()

	response := APIResponse{
		Success: true,
		Message: "Simulation started successfully",
		Data:    AppState,
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go AppState.BroadcastUpdate()

	logger.LogWithNode("System", "Simulation", fmt.Sprintf("Simulation started with profile: %s, Target EPS: %d", config.Profile, config.TargetEPS), "info")
}

func StopSimulation(w http.ResponseWriter, r *http.Request) {
	AppState.Mutex.Lock()
	defer AppState.Mutex.Unlock()

	if !AppState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "No simulation is currently running",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	AppState.IsSimulationRunning = false

	response := APIResponse{
		Success: true,
		Message: "Simulation stopped successfully",
		Data:    AppState,
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go AppState.BroadcastUpdate()

	logger.LogWithNode("System", "Simulation", "Simulation stopped", "info")
}

func SyncConfiguration(w http.ResponseWriter, r *http.Request) {
	AppState.Mutex.Lock()
	defer AppState.Mutex.Unlock()

	// In a real implementation, this would sync with external configuration sources
	response := APIResponse{
		Success: true,
		Message: "Configuration synced successfully",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   AppVersion,
		},
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)

	logger.LogWithNode("System", "Config", "Configuration synced", "info")
}
