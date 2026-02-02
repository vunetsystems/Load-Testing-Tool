package handlers

import (
	"encoding/json"
	"net/http"

	"vuDataSim/src/logger"
	"vuDataSim/src/traces"
)

// HandleAPIStartTracesSimulation handles POST /api/traces/start
func HandleAPIStartTracesSimulation(w http.ResponseWriter, r *http.Request) {
	var req traces.TracesStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.TestName == "" {
		response := APIResponse{
			Success: false,
			Message: "Test name is required",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.EPS <= 0 {
		response := APIResponse{
			Success: false,
			Message: "EPS must be greater than 0",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Minutes <= 0 {
		response := APIResponse{
			Success: false,
			Message: "Minutes must be greater than 0",
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start the simulation
	err := traces.Simulator.Start(req.TestName, req.EPS, req.Minutes)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: err.Error(),
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	logger.Info().
		Str("test_name", req.TestName).
		Int("eps", req.EPS).
		Int("minutes", req.Minutes).
		Msg("Traces simulation started via API")

	response := APIResponse{
		Success: true,
		Message: "Traces simulation started successfully",
		Data: map[string]interface{}{
			"test_name": req.TestName,
			"eps":       req.EPS,
			"minutes":   req.Minutes,
		},
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)
}

// HandleAPIStopTracesSimulation handles POST /api/traces/stop
func HandleAPIStopTracesSimulation(w http.ResponseWriter, r *http.Request) {
	err := traces.Simulator.Stop()
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: err.Error(),
		}
		w.Header().Set(ContentTypeHeader, ApplicationJSON)
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	logger.Info().Msg("Traces simulation stopped via API")

	response := APIResponse{
		Success: true,
		Message: "Traces simulation stopped successfully",
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)
}

// HandleAPIGetTracesStatus handles GET /api/traces/status
func HandleAPIGetTracesStatus(w http.ResponseWriter, r *http.Request) {
	status := traces.Simulator.Status()

	response := APIResponse{
		Success: true,
		Message: status.Message,
		Data:    status,
	}

	w.Header().Set(ContentTypeHeader, ApplicationJSON)
	json.NewEncoder(w).Encode(response)
}
