package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"vuDataSim/src/o11y_source_manager"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// Category represents a single category configuration
type Category struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Sources     []string `yaml:"sources" json:"sources"`
}

// CategoriesConfig represents the entire categories configuration
type CategoriesConfig struct {
	Categories map[string]Category `yaml:"categories" json:"categories"`
}

// LoadCategoriesConfig loads categories from the YAML file
func LoadCategoriesConfig() (*CategoriesConfig, error) {
	configPath := filepath.Join("src", "configs", "categories.yaml")

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read categories config file: %v", err)
	}

	var config CategoriesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse categories config YAML: %v", err)
	}

	return &config, nil
}

func HandleAPIGetO11ySources(w http.ResponseWriter, r *http.Request) {
	// Initialize o11y manager if not already done
	if len(O11yManager.GetMaxEPSConfig()) == 0 {
		err := O11yManager.LoadMaxEPSConfig()
		if err != nil {
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to load max EPS config: %v", err),
			})
			return
		}
	}

	// Also load main config to ensure it's up to date
	err := O11yManager.LoadMainConfig()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load main config: %v", err),
		})
		return
	}

	sources := O11yManager.GetAvailableSources()
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    sources,
		Message: fmt.Sprintf("Retrieved %d available o11y sources", len(sources)),
	})
}

// HandleAPIGetEnabledO11ySources Handles GET /api/o11y/sources/enabled
func HandleAPIGetEnabledO11ySources(w http.ResponseWriter, r *http.Request) {
	// Ensure o11y manager is initialized
	// Available sources are loaded dynamically when needed

	sources := O11yManager.GetEnabledSources()
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    sources,
	})
}

// HandleAPIGetO11ySourceDetails Handles GET /api/o11y/sources/{source}
func HandleAPIGetO11ySourceDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	details, err := O11yManager.GetSourceDetails(sourceName)
	if err != nil {
		SendJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Source not found: %s", sourceName),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    details,
	})
}

// HandleAPIDistributeEPS Handles POST /api/o11y/eps/distribute
func HandleAPIDistributeEPS(w http.ResponseWriter, r *http.Request) {
	var request o11y_source_manager.EPSDistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	response, err := O11yManager.DistributeEPS(request)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusBadRequest
	}

	SendJSONResponse(w, statusCode, APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	})
}

// HandleAPIGetCurrentEPS Handles GET /api/o11y/eps/current
func HandleAPIGetCurrentEPS(w http.ResponseWriter, r *http.Request) {
	// Available sources are loaded dynamically when needed

	currentEPS := O11yManager.CalculateCurrentEPS()
	breakdown := O11yManager.GetSourceEPSBreakdown()

	data := map[string]interface{}{
		"totalEPS":  currentEPS,
		"breakdown": breakdown,
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// HandleAPIEnableO11ySource Handles POST /api/o11y/sources/{source}/enable
func HandleAPIEnableO11ySource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	err := O11yManager.EnableSource(sourceName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Source %s enabled successfully", sourceName),
	})
}

// HandleAPIDisableO11ySource Handles POST /api/o11y/sources/{source}/disable
func HandleAPIDisableO11ySource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceName := vars["source"]

	if sourceName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}

	// Available sources are loaded dynamically when needed

	err := O11yManager.DisableSource(sourceName)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Source %s disabled successfully", sourceName),
	})
}

// HandleAPIGetMaxEPSConfig Handles GET /api/o11y/max-eps
func HandleAPIGetMaxEPSConfig(w http.ResponseWriter, r *http.Request) {
	// Ensure o11y manager is initialized
	if len(O11yManager.GetMaxEPSConfig()) == 0 {
		err := O11yManager.LoadMaxEPSConfig()
		if err != nil {
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to load max EPS config: %v", err),
			})
			return
		}
	}

	maxEPSConfig := O11yManager.GetMaxEPSConfig()
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    maxEPSConfig,
	})
}

// HandleAPIDistributeConfD Handles POST /api/o11y/confd/distribute
func HandleAPIDistributeConfD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	// Distribute conf.d to all enabled nodes
	response, err := O11yManager.DistributeConfD()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to distribute conf.d: %v", err),
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusPartialContent // 206 for partial success
	}

	apiResponse := APIResponse{
		Success: response.Success,
		Message: response.Message,
		Data:    response.Data,
	}

	// Add distribution details to response data
	if apiResponse.Data == nil {
		apiResponse.Data = make(map[string]interface{})
	}
	apiResponse.Data.(map[string]interface{})["distribution"] = response.Distribution

	SendJSONResponse(w, statusCode, apiResponse)
}

// HandleAPIGetO11yCategories Handles GET /api/o11y/categories
func HandleAPIGetO11yCategories(w http.ResponseWriter, r *http.Request) {
	// Load categories from YAML file
	config, err := LoadCategoriesConfig()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load categories config: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    config.Categories,
		Message: fmt.Sprintf("Retrieved %d categories", len(config.Categories)),
	})
}
