package handlers

import (
	"encoding/json"
	"net/http"
	"errors"
	"time"
	"vuDataSim/src/kafka_ch_reset"
	"vuDataSim/src/logger"
)

type PodDeletionRequest struct {
	TimeoutSeconds int      `json:"timeout_seconds"`
	O11ySources    []string `json:"o11y_sources"`
}

func SchedulePodDeletionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PodDeletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.TimeoutSeconds <= 0 || len(req.O11ySources) == 0 {
		http.Error(w, "Invalid timeout or o11y sources", http.StatusBadRequest)
		return
	}

	km, err := kafka_ch_reset.NewKafkaManager("src/configs/topics_tables.yaml")  // Adjust path as needed
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create KafkaManager")
		http.Error(w, "Failed to initialize manager", http.StatusInternalServerError)
		return
	}
	scheduledTime, err := km.SchedulePodDeletion(req.TimeoutSeconds, req.O11ySources)
	if err != nil {
		if errors.Is(err, kafka_ch_reset.ErrNoPipelinesFound) {
			// Return success message, but no scheduling happened
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "no_pipelines_found",
				"message": "No pipelines found in YAML for given sources, nothing to schedule.",
				"o11y_sources": req.O11ySources,
			})
			return
		}

		logger.Error().Err(err).Msg("Failed to schedule pod deletion")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "scheduled",
		"scheduled_time": scheduledTime.Format(time.RFC3339),
		"timeout_seconds": req.TimeoutSeconds,
		"o11y_sources":   req.O11ySources,
	})
}