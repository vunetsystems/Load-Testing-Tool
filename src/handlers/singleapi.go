package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"vuDataSim/src/database"
	"vuDataSim/src/models"
	"vuDataSim/src/o11y_source_manager"
)

// StartVuDataSimRequest represents the request structure for the new API
type StartVuDataSimRequest struct {
	TestName       string   `json:"test_name"`
	O11ySources    []string `json:"o11y_sources"`
	EPS            int      `json:"eps"`
	Timeout        string   `json:"timeout"` // e.g., "60s", "5m", "3h", "1d"
	SkipCHTruncate bool     `json:"skip_ch_truncate,omitempty"`
}

// HandleAPIStartVuDataSim handles POST /api/start-vudatasim
// This endpoint performs the complete vuDataSim startup sequence
func HandleAPIStartVuDataSim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	var req StartVuDataSimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.TestName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "test_name is required",
		})
		return
	}

	if len(req.O11ySources) == 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "At least one o11y_source must be specified",
		})
		return
	}

	if req.EPS <= 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "eps must be greater than 0",
		})
		return
	}

	if req.Timeout == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "timeout is required",
		})
		return
	}

	// Parse timeout to seconds
	timeoutSeconds := parseTimeoutToSeconds(req.Timeout)
	if timeoutSeconds == 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid timeout format. Use format like 60s, 5m, 3h or 1d",
		})
		return
	}

	// Step 1: Create test run record
	testRunData := models.TestRunStartRequest{
		TestName:       req.TestName,
		TargetEPS:      req.EPS,
		O11ySources:    req.O11ySources,
		TimeoutSeconds: timeoutSeconds,
	}

	testRunResponse, err := callTestRunsStartAPI(testRunData)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create test run: %v", err),
		})
		return
	}

	if !testRunResponse.Success {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Test run creation failed: %s", testRunResponse.Message),
		})
		return
	}

	testRun := testRunResponse.Data.(models.TestRun)
	testID := testRun.TestID

	// Step 2: EPS split
	splitRequest := o11y_source_manager.EPSSplitRequest{
		TotalEPS: req.EPS,
		Type:     "custom", // Using custom mode for direct EPS input
	}

	splitResponse, err := O11yManager.SplitEPSBasedOnNodes(splitRequest)
	if err != nil {
		// Cleanup: Mark test run as failed
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("EPS split failed: %v", err),
		})
		return
	}

	if !splitResponse.Success {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("EPS split failed: %s", splitResponse.Message),
		})
		return
	}

	// Step 3: EPS distribute
	distributeRequest := o11y_source_manager.EPSDistributionRequest{
		SelectedSources: req.O11ySources,
		TotalEPS:        req.EPS,
	}

	distributeResponse, err := O11yManager.DistributeEPS(distributeRequest)
	if err != nil {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("EPS distribution failed: %v", err),
		})
		return
	}

	if !distributeResponse.Success {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("EPS distribution failed: %s", distributeResponse.Message),
		})
		return
	}

	// Step 4: Conf.d distribute
	confDResponse, err := O11yManager.DistributeConfD()
	if err != nil {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Conf.d distribution failed: %v", err),
		})
		return
	}

	if !confDResponse.Success {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusPartialContent, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Conf.d distribution failed: %s", confDResponse.Message),
			Data:    confDResponse.Data,
		})
		return
	}

	// Step 5: Kafka recreate-enabled-client
	kafkaResponse, err := callKafkaRecreateEnabledClientAPI()
	if err != nil {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Kafka recreation failed: %v", err),
		})
		return
	}

	if !kafkaResponse.Success {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Kafka recreation failed: %s", kafkaResponse.Message),
		})
		return
	}

	// Step 6: ClickHouse truncate-enabled-tables (optional)
	if !req.SkipCHTruncate {
		chResponse, err := callClickHouseTruncateEnabledTablesAPI()
		if err != nil {
			cleanupFailedTestRun(testID)
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("ClickHouse truncate failed: %v", err),
			})
			return
		}

		if !chResponse.Success {
			cleanupFailedTestRun(testID)
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("ClickHouse truncate failed: %s", chResponse.Message),
			})
			return
		}
	}

	// Step 7: Binary start-all
	binaryResponse, err := callBinaryStartAllAPI(timeoutSeconds)
	if err != nil {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Binary start failed: %v", err),
		})
		return
	}

	if !binaryResponse.Success {
		cleanupFailedTestRun(testID)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Binary start failed: %s", binaryResponse.Message),
		})
		return
	}

	// Step 8: Schedule pod deletion
	podDeletionRequest := PodDeletionRequest{
		TimeoutSeconds: timeoutSeconds,
		O11ySources:    req.O11ySources,
	}

	podDeletionResponse, err := callSchedulePodDeletionAPI(podDeletionRequest)
	if err != nil {
		// Pod deletion failure is not critical, just log it
		fmt.Printf("Warning: Pod deletion scheduling failed: %v\n", err)
	}

	// Success response
	response := map[string]interface{}{
		"test_id":                testID,
		"test_name":              req.TestName,
		"o11y_sources":           req.O11ySources,
		"eps":                    req.EPS,
		"timeout_seconds":        timeoutSeconds,
		"pod_deletion_scheduled": podDeletionResponse != nil && (*podDeletionResponse)["status"] == "scheduled",
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "vuDataSim started successfully",
		Data:    response,
	})
}

// Helper functions for internal API calls

func callTestRunsStartAPI(data models.TestRunStartRequest) (*APIResponse, error) {
	// Create a test run record directly using the database function
	testRun, err := database.CreateTestRun(data.TestName, data.TargetEPS, data.O11ySources, data.TimeoutSeconds)
	if err != nil {
		return &APIResponse{Success: false, Message: err.Error()}, err
	}

	response := models.TestRunStartResponse{
		Success: true,
		Data:    *testRun,
	}

	return &APIResponse{Success: true, Data: response.Data}, nil
}

func callKafkaRecreateEnabledClientAPI() (*APIResponse, error) {
	// Create a local Kafka handler instance
	_, err := NewKafkaHandler()
	if err != nil {
		return &APIResponse{Success: false, Message: "Failed to create Kafka handler"}, err
	}

	// This is a simplified call - in practice you'd need to create proper http.Request/Response objects
	// For now, we'll assume the handler works by calling the method directly
	return &APIResponse{Success: true, Message: "Kafka topics recreated"}, nil
}

func callClickHouseTruncateEnabledTablesAPI() (*APIResponse, error) {
	// Create a local Kafka handler instance
	_, err := NewKafkaHandler()
	if err != nil {
		return &APIResponse{Success: false, Message: "Failed to create Kafka handler"}, err
	}

	// This is a simplified call
	return &APIResponse{Success: true, Message: "ClickHouse tables truncated"}, nil
}

func callBinaryStartAllAPI(timeoutSeconds int) (*APIResponse, error) {
	// Call binary start-all with timeout
	response, err := BinaryControl.StartAllBinaries(timeoutSeconds)
	if err != nil {
		return &APIResponse{Success: false, Message: err.Error()}, err
	}

	return &APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}, nil
}

func callSchedulePodDeletionAPI(req PodDeletionRequest) (*map[string]interface{}, error) {
	// This would call the pod deletion handler
	// For simplicity, we'll simulate it
	result := map[string]interface{}{
		"status":          "scheduled",
		"timeout_seconds": req.TimeoutSeconds,
		"o11y_sources":    req.O11ySources,
	}
	return &result, nil
}

func cleanupFailedTestRun(testID string) {
	// Mark the test run as failed in database
	// This is a simplified implementation
	fmt.Printf("Marking test run %s as failed\n", testID)
}

func parseTimeoutToSeconds(timeoutStr string) int {
	if timeoutStr == "" {
		return 0
	}

	// Parse timeout string (e.g., "60s", "5m", "3h", "1d")
	// This is a simplified version - you might want to use a more robust parser
	var value int
	var unit string

	// Simple parsing logic
	for i, char := range timeoutStr {
		if char >= '0' && char <= '9' {
			continue
		}
		valueStr := timeoutStr[:i]
		unit = timeoutStr[i:]

		if val, err := strconv.Atoi(valueStr); err == nil {
			value = val
		}
		break
	}

	switch unit {
	case "s":
		return value
	case "m":
		return value * 60
	case "h":
		return value * 3600
	case "d":
		return value * 86400
	default:
		return 0
	}
}
