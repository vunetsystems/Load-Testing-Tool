// Package handlers provides HTTP handlers for the test run tracking process
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"vuDataSim/src/database"
	"vuDataSim/src/models"
	"github.com/gorilla/mux"
)

// HandleAPIStartTestRun handles POST /api/test-runs/start
// This endpoint creates a new test run record in the database
func HandleAPIStartTestRun(w http.ResponseWriter, r *http.Request) {
	var req models.TestRunStartRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.TargetEPS <= 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Target EPS must be greater than 0",
		})
		return
	}

	if len(req.O11ySources) == 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "At least one o11y source must be specified",
		})
		return
	}

	if req.TimeoutSeconds <= 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Timeout seconds must be greater than 0",
		})
		return
	}

	// Create test run record
	testRun, err := database.CreateTestRun(req.TargetEPS, req.O11ySources, req.TimeoutSeconds)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create test run: %v", err),
		})
		return
	}

	response := models.TestRunStartResponse{
		Success: true,
		Data:    *testRun,
	}

	SendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    response.Data,
	})
}

// HandleAPIStopTestRun handles PUT /api/test-runs/{id}/stop
// This endpoint updates the stop time for a test run
func HandleAPIStopTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	// Validate UUID format (basic validation)
	if len(testID) != 36 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid test ID format",
		})
		return
	}

	// Stop the test run
	err := database.StopTestRun(testID)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop test run: %v", err),
		})
		return
	}

	// Get updated test run data
	testRun, err := database.GetTestRun(testID)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve updated test run: %v", err),
		})
		return
	}

	response := models.TestRunStopResponse{
		Success: true,
		Data:    *testRun,
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response.Data,
	})
}

// HandleAPIGetTestRuns handles GET /api/test-runs
// This endpoint retrieves all test runs
func HandleAPIGetTestRuns(w http.ResponseWriter, r *http.Request) {
	testRuns, err := database.GetAllTestRuns()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve test runs: %v", err),
		})
		return
	}

	response := models.TestRunListResponse{
		Success: true,
		Data:    testRuns,
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response.Data,
	})
}

// HandleAPIGetTestRun handles GET /api/test-runs/{id}
// This endpoint retrieves a specific test run by ID
func HandleAPIGetTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	// Validate UUID format (basic validation)
	if len(testID) != 36 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid test ID format",
		})
		return
	}

	testRun, err := database.GetTestRun(testID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := fmt.Sprintf("Failed to retrieve test run: %v", err)
		if err.Error() == fmt.Sprintf("test run with ID %s not found", testID) {
			statusCode = http.StatusNotFound
			message = err.Error()
		}
		SendJSONResponse(w, statusCode, APIResponse{
			Success: false,
			Message: message,
		})
		return
	}

	response := models.TestRunDetailResponse{
		Success: true,
		Data:    *testRun,
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response.Data,
	})
}

// HandleAPIGetNextTestID handles GET /api/test-runs/next-id
// This endpoint returns the next test ID that would be assigned
func HandleAPIGetNextTestID(w http.ResponseWriter, r *http.Request) {
	nextID, err := database.GetNextTestID()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get next test ID: %v", err),
		})
		return
	}

	response := models.NextTestIDResponse{
		Success: true,
		Data: models.NextTestIDData{
			NextTestID: nextID,
		},
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    response.Data,
	})
}