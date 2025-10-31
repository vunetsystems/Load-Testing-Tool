package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	"vuDataSim/src/logger"
)

// K6Config represents the K6 load testing configuration
type K6Config struct {
	GlobalUserCount      int      `json:"globalUserCount"`
	TestDuration         string   `json:"testDuration"` // e.g., "6h", "15m"
	RampUpDuration       int      `json:"rampUpDuration"` // seconds
	MaxDuration          int      `json:"maxDuration"` // seconds
	EnabledScripts       []string `json:"enabledScripts"`
	IntervalBetweenTests int      `json:"intervalBetweenTests"` // seconds
}

// K6Status represents the current K6 execution status
type K6Status struct {
	IsRunning         bool      `json:"isRunning"`
	CurrentScript     string    `json:"currentScript,omitempty"`
	StartTime         time.Time `json:"startTime,omitempty"`
	CurrentUserCount  int       `json:"currentUserCount"`
	CompletedScripts  []string  `json:"completedScripts"`
	FailedScripts     []string  `json:"failedScripts"`
	LastError         string    `json:"lastError,omitempty"`
}

// ModuleDashboardMapping represents the mapping between modules and dashboards
type ModuleDashboardMapping struct {
	ModuleDashboardMapping map[string]struct {
		DashboardName string `yaml:"dashboard_name"`
		DashboardSlug string `yaml:"dashboard_slug"`
	} `yaml:"module_dashboard_mapping"`
}

// K6DashboardConfig represents the structure of k6_config.yaml
type K6DashboardConfig struct {
	BaseURLs struct {
		DashboardAPI string `yaml:"dashboard_api"`
		Panel        string `yaml:"panel"`
	} `yaml:"base_urls"`
	Dashboards []struct {
		ID       string `yaml:"id"`
		Name     string `yaml:"name"`
		Slug     string `yaml:"slug"`
		Enabled  bool   `yaml:"enabled"`
	} `yaml:"dashboards"`
}

// K6Handler manages K6 load testing operations
type K6Handler struct {
	config     K6Config
	status     K6Status
	mutex      sync.RWMutex
	cmd        *exec.Cmd
}

// Global K6 handler instance
var K6Manager = NewK6Handler()

// NewK6Handler creates a new K6Handler instance
func NewK6Handler() *K6Handler {
	handler := &K6Handler{
		config: K6Config{
			GlobalUserCount:      10,
			TestDuration:         "6h",
			RampUpDuration:       10,
			MaxDuration:          10,
			EnabledScripts:       []string{"overall-1.sh"},
			IntervalBetweenTests: 300, // 5 minutes
		},
		status: K6Status{
			IsRunning:        false,
			CurrentUserCount: 10,
			CompletedScripts: []string{},
			FailedScripts:    []string{},
		},
	}

	// Load configuration from file if it exists
	handler.loadConfig()

	return handler
}

// loadConfig loads K6 configuration from file
func (h *K6Handler) loadConfig() {
	configPath := "src/k6_config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Save default config if file doesn't exist
		h.saveConfig()
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to read K6 config file")
		return
	}

	var config K6Config
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to parse K6 config file")
		return
	}

	h.mutex.Lock()
	h.config = config
	h.status.CurrentUserCount = config.GlobalUserCount
	h.mutex.Unlock()

	logger.Info().Str("module", "k6").Msg("K6 configuration loaded successfully")
}

// saveConfig saves current K6 configuration to file
func (h *K6Handler) saveConfig() {
	configPath := "src/k6_config.json"
	data, err := json.MarshalIndent(h.config, "", "  ")
	if err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to marshal K6 config")
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to write K6 config file")
		return
	}

	logger.Info().Str("module", "k6").Msg("K6 configuration saved successfully")
}

// GetK6Config handles GET /api/k6/config
func (h *K6Handler) GetK6Config(w http.ResponseWriter, r *http.Request) {
	h.mutex.RLock()
	config := h.config
	h.mutex.RUnlock()

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    config,
		Message: "K6 configuration retrieved successfully",
	})
}

// UpdateK6Config handles PUT /api/k6/config
func (h *K6Handler) UpdateK6Config(w http.ResponseWriter, r *http.Request) {
	var newConfig K6Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	// Validate configuration
	if err := h.validateConfig(newConfig); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	h.mutex.Lock()
	h.config = newConfig
	h.status.CurrentUserCount = newConfig.GlobalUserCount
	h.mutex.Unlock()

	// Save configuration to file
	h.saveConfig()

	// Broadcast update
	go AppState.BroadcastUpdate()

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    newConfig,
		Message: "K6 configuration updated successfully",
	})

	logger.LogWithNode("System", "k6", fmt.Sprintf("K6 configuration updated: %d users, %s duration", newConfig.GlobalUserCount, newConfig.TestDuration), "info")
}

// GetK6Status handles GET /api/k6/status
func (h *K6Handler) GetK6Status(w http.ResponseWriter, r *http.Request) {
	h.mutex.RLock()
	status := h.status
	h.mutex.RUnlock()

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
		Message: "K6 status retrieved successfully",
	})
}

// StartK6Test handles POST /api/k6/start
func (h *K6Handler) StartK6Test(w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.status.IsRunning {
		SendJSONResponse(w, http.StatusConflict, APIResponse{
			Success: false,
			Message: "K6 test is already running",
		})
		return
	}

	// Generate dynamic script with current configuration
	scriptPath, err := h.generateK6Script()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to generate K6 script: %v", err),
		})
		return
	}

	// Start K6 execution in background
	go h.executeK6Script(scriptPath)

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 test started successfully",
		Data: map[string]interface{}{
			"scriptPath": scriptPath,
			"userCount":  h.config.GlobalUserCount,
			"duration":   h.config.TestDuration,
		},
	})

	logger.LogWithNode("System", "k6", fmt.Sprintf("K6 test started: %d users, %s duration", h.config.GlobalUserCount, h.config.TestDuration), "info")
}

// StopK6Test handles POST /api/k6/stop
func (h *K6Handler) StopK6Test(w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.status.IsRunning {
		SendJSONResponse(w, http.StatusConflict, APIResponse{
			Success: false,
			Message: "No K6 test is currently running",
		})
		return
	}

	// Stop the running K6 process
	if h.cmd != nil && h.cmd.Process != nil {
		if err := h.cmd.Process.Kill(); err != nil {
			logger.Error().Err(err).Str("module", "k6").Msg("Failed to kill K6 process")
		}
	}

	h.status.IsRunning = false
	h.status.LastError = ""

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 test stopped successfully",
	})

	logger.LogWithNode("System", "k6", "K6 test stopped", "info")
}

// validateConfig validates the K6 configuration parameters
func (h *K6Handler) validateConfig(config K6Config) error {
	if config.GlobalUserCount < 1 || config.GlobalUserCount > 1000 {
		return fmt.Errorf("global user count must be between 1 and 1000")
	}

	if config.TestDuration == "" {
		return fmt.Errorf("test duration is required")
	}

	if config.RampUpDuration < 1 {
		return fmt.Errorf("ramp up duration must be at least 1 second")
	}

	if config.MaxDuration < 1 {
		return fmt.Errorf("max duration must be at least 1 second")
	}

	if config.IntervalBetweenTests < 0 {
		return fmt.Errorf("interval between tests cannot be negative")
	}

	if len(config.EnabledScripts) == 0 {
		return fmt.Errorf("at least one script must be enabled")
	}

	return nil
}

// generateK6Script generates a dynamic K6 script based on current configuration
func (h *K6Handler) generateK6Script() (string, error) {
	template := `#!/bin/bash

# Auto-generated K6 script
# Generated at: %s
# Global User Count: %d
# Test Duration: %s

echo "Starting K6 load test with %d users for %s duration"
echo "Generated at: %s"
echo "Working directory: $(pwd)"

# Execute K6 scripts with configured parameters
%s

echo "K6 load test completed"
`

	// Generate script execution commands for each enabled script
	var scriptCommands string
	for _, script := range h.config.EnabledScripts {
		// Map script names to their full paths in k6_dashboard_name subdirectories
		var scriptPath string
		switch script {
		case "overall-1.sh":
			scriptPath = "k6_dashboard_name/linux-mssql-dashboard/overall-1.sh"
		case "traces.sh":
			scriptPath = "k6_dashboard_name/traces/overall-1.sh"
		case "login.sh":
			scriptPath = "k6_dashboard_name/login/overall.sh"
		case "reports.sh":
			scriptPath = "k6_dashboard_name/reports/overall.sh"
		case "log_analytics.sh":
			scriptPath = "k6_dashboard_name/log_analytics/overall-1.sh"
		default:
			scriptPath = script // fallback to direct path
		}

		scriptCmd := fmt.Sprintf("./%s %s %d %d %d\n",
			scriptPath,
			h.config.TestDuration,
			h.config.GlobalUserCount,
			h.config.RampUpDuration,
			h.config.MaxDuration)
		scriptCommands += scriptCmd
	}

	// Generate the complete script
	generatedScript := fmt.Sprintf(template,
		time.Now().Format("2006-01-02 15:04:05"),
		h.config.GlobalUserCount,
		h.config.TestDuration,
		time.Now().Format("2006-01-02 15:04:05"),
		scriptCommands)

	// Write to temporary file
	scriptPath := "/tmp/k6_dynamic_script.sh"
	if err := os.WriteFile(scriptPath, []byte(generatedScript), 0755); err != nil {
		return "", fmt.Errorf("failed to write dynamic script: %v", err)
	}

	return scriptPath, nil
}

// executeK6Script executes the generated K6 script
func (h *K6Handler) executeK6Script(scriptPath string) {
	h.mutex.Lock()
	h.status.IsRunning = true
	h.status.StartTime = time.Now()
	h.status.CurrentScript = scriptPath
	h.status.LastError = ""
	h.cmd = nil
	h.mutex.Unlock()

	// Broadcast initial status
	go AppState.BroadcastUpdate()

	defer func() {
		h.mutex.Lock()
		h.status.IsRunning = false
		h.status.CurrentScript = ""
		h.mutex.Unlock()

		// Broadcast final status
		go AppState.BroadcastUpdate()

		// Clean up temporary script
		os.Remove(scriptPath)
	}()

	logger.LogWithNode("System", "k6", fmt.Sprintf("Starting K6 script execution: %s", scriptPath), "info")

	// Execute the script
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Dir = "k6_final" // Working directory

	// Set up process for potential cancellation
	h.mutex.Lock()
	h.cmd = cmd
	h.mutex.Unlock()

	output, err := cmd.CombinedOutput()

	h.mutex.Lock()
	if err != nil {
		h.status.LastError = err.Error()
		logger.Error().Err(err).Str("module", "k6").Msg("K6 script execution failed")
	} else {
		logger.Info().Str("module", "k6").Msg("K6 script execution completed successfully")
	}

	// Log output for debugging
	if len(output) > 0 {
		logger.Info().Str("module", "k6").Str("output", string(output)).Msg("K6 script output")
	}
	h.mutex.Unlock()
}

// ResetK6Config handles POST /api/k6/config/reset
func (h *K6Handler) ResetK6Config(w http.ResponseWriter, r *http.Request) {
	defaultConfig := K6Config{
		GlobalUserCount:      10,
		TestDuration:         "6h",
		RampUpDuration:       10,
		MaxDuration:          10,
		EnabledScripts:       []string{"overall-1.sh"},
		IntervalBetweenTests: 300,
	}

	h.mutex.Lock()
	h.config = defaultConfig
	h.status.CurrentUserCount = defaultConfig.GlobalUserCount
	h.mutex.Unlock()

	// Save default configuration
	h.saveConfig()

	// Broadcast update
	go AppState.BroadcastUpdate()

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    defaultConfig,
		Message: "K6 configuration reset to defaults",
	})

	logger.LogWithNode("System", "k6", "K6 configuration reset to defaults", "info")
}

// RunCombinedScript handles POST /api/k6/run-combined
func (h *K6Handler) RunCombinedScript(w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Check if status indicates running and verify if process is actually running
	if h.status.IsRunning {
		// Check if the process is actually still running
		if h.cmd != nil && h.cmd.Process != nil {
			if err := h.cmd.Process.Signal(syscall.Signal(0)); err == nil {
				// Process is still running
				SendJSONResponse(w, http.StatusConflict, APIResponse{
					Success: false,
					Message: "K6 test is already running",
				})
				return
			}
		}
		// Process is not running, reset status
		h.status.IsRunning = false
		h.status.LastError = ""
		logger.Warn().Str("module", "k6").Msg("Detected stale running status, resetting")
	}

	// Check for stuck execution (running for more than 2 hours)
	if h.status.IsRunning && time.Since(h.status.StartTime) > 2*time.Hour {
		logger.Warn().Str("module", "k6").Msg("Detected stuck execution, resetting status")
		h.status.IsRunning = false
		h.status.LastError = "Execution was stuck and has been reset"
		if h.cmd != nil && h.cmd.Process != nil {
			h.cmd.Process.Kill()
		}
		h.cmd = nil
	}

	// Parse request body for parameters
	var params struct {
		TimeRange  string `json:"timeRange"`
		VUs        int    `json:"vus"`
		Iterations int    `json:"iterations"`
		Interval   int    `json:"interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	// Validate parameters
	if params.TimeRange == "" {
		params.TimeRange = "15m"
	}
	if params.VUs <= 0 {
		params.VUs = 10
	}
	if params.Iterations <= 0 {
		params.Iterations = 5
	}
	if params.Interval <= 0 {
		params.Interval = 5
	}

	// Read module configuration and update dashboard config synchronously
	enabledModules, err := h.readModuleConfig()
	if err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to read module configuration")
		// Continue with execution even if config read fails
	} else {
		logger.Info().Str("module", "k6").Int("enabled_modules", len(enabledModules)).Msg("Module configuration read successfully")
	}

	if enabledModules != nil {
		err = h.updateDashboardConfig(enabledModules)
		if err != nil {
			logger.Error().Err(err).Str("module", "k6").Msg("Failed to update dashboard configuration")
			// Continue with execution even if update fails
		} else {
			logger.Info().Str("module", "k6").Msg("Dashboard configuration updated successfully")
		}
	}

	// Start execution in background
	go h.executeCombinedScript(params.TimeRange, params.VUs, params.Iterations, params.Interval)

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "K6 combined script started successfully",
		Data: map[string]interface{}{
			"timeRange":  params.TimeRange,
			"vus":        params.VUs,
			"iterations": params.Iterations,
			"interval":   params.Interval,
		},
	})

	logger.LogWithNode("System", "k6", fmt.Sprintf("K6 combined script started: timeRange=%s, vus=%d, iterations=%d, interval=%d",
		params.TimeRange, params.VUs, params.Iterations, params.Interval), "info")
}

// updateStatus sends real-time status updates via WebSocket
func (h *K6Handler) updateStatus(message string) {
	h.mutex.Lock()
	h.status.CurrentScript = message
	h.mutex.Unlock()

	// Broadcast update
	go AppState.BroadcastUpdate()

	logger.LogWithNode("System", "k6", fmt.Sprintf("Status update: %s", message), "info")
}

// checkAndCreateUsers checks user validity and creates users if needed
func (h *K6Handler) checkAndCreateUsers(vus int) string {
	timeoutPath := "k6_final/user_creation_k6/timeout.txt"

	// Read timeout.txt file
	data, err := os.ReadFile(timeoutPath)
	if err != nil {
		logger.Error().Err(err).Str("module", "k6").Msg("Failed to read timeout.txt")
		return "Failed to read user timeout file"
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var lastTimestamp time.Time
	var lastUserCount int

	// Parse timestamp from first line
	if len(lines) > 0 {
		if strings.Contains(lines[0], "Script started at:") {
			timestampStr := strings.TrimPrefix(lines[0], "Script started at: ")
			if parsed, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(timestampStr)); err == nil {
				lastTimestamp = parsed
			}
		}
	}

	// Parse user count from second line
	if len(lines) > 1 {
		if strings.Contains(lines[1], "Target number of users:") {
			countStr := strings.TrimPrefix(lines[1], "Target number of users: ")
			if count, err := strconv.Atoi(strings.TrimSpace(countStr)); err == nil {
				lastUserCount = count
			}
		}
	}

	now := time.Now()
	timeDiff := now.Sub(lastTimestamp)
	needsRefresh := timeDiff.Hours() >= 24 || lastUserCount < vus

	if !needsRefresh {
		return fmt.Sprintf("Users still valid (created %s, %d users available)", lastTimestamp.Format("2006-01-02 15:04:05"), lastUserCount)
	}

	// Need to create users
	h.updateStatus("🔧 Creating new users...")

	cmd := exec.Command("python3", "./user_cookies.py", fmt.Sprintf("%d", vus))
	cmd.Dir = "k6_final/user_creation_k6"

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().Err(err).Str("module", "k6").Str("output", string(output)).Msg("User creation script failed")
		return fmt.Sprintf("User creation failed: %v", err)
	}

	// Check if user creation was successful by reading the updated timeout.txt
	data, err = os.ReadFile(timeoutPath)
	if err != nil {
		return "Failed to verify user creation success"
	}

	content = string(data)
	if strings.Contains(content, fmt.Sprintf("🎉 All %d/%d users created successfully", vus, vus)) {
		return fmt.Sprintf("Successfully created %d users", vus)
	}

	logger.Warn().Str("module", "k6").Str("output", string(output)).Msg("User creation may not have completed successfully")
	return fmt.Sprintf("User creation completed (verification inconclusive)")
}

// executeCombinedScript executes the combined.sh script with given parameters
func (h *K6Handler) executeCombinedScript(timeRange string, vus, iterations, interval int) {
	h.mutex.Lock()
	h.status.IsRunning = true
	h.status.StartTime = time.Now()
	h.status.CurrentScript = "Initializing..."
	h.status.LastError = ""
	h.cmd = nil
	h.mutex.Unlock()

	// Broadcast initial status
	go AppState.BroadcastUpdate()

	defer func() {
		if r := recover(); r != nil {
			h.mutex.Lock()
			h.status.IsRunning = false
			h.status.LastError = fmt.Sprintf("Panic recovered: %v", r)
			h.status.CurrentScript = ""
			h.mutex.Unlock()
			logger.Error().Interface("panic", r).Str("module", "k6").Msg("Panic recovered in executeCombinedScript")
			go AppState.BroadcastUpdate()
		} else {
			h.mutex.Lock()
			h.status.IsRunning = false
			h.status.CurrentScript = ""
			h.mutex.Unlock()
			// Broadcast final status
			go AppState.BroadcastUpdate()
		}
	}()

	// STEP 1: User Validation and Creation
	h.updateStatus("🔍 Checking user validity and count...")
	userCheckResult := h.checkAndCreateUsers(vus)
	logger.LogWithNode("System", "k6", fmt.Sprintf("User validation completed: %s", userCheckResult), "info")
	h.updateStatus(fmt.Sprintf("✅ User check completed: %s", userCheckResult))

	// STEP 2: Execute Combined Script
	h.updateStatus("🚀 Starting combined K6 test execution...")

	// Create context with timeout (2 hours)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./combined.sh", timeRange, fmt.Sprintf("%d", vus), fmt.Sprintf("%d", iterations), fmt.Sprintf("%d", interval))
	cmd.Dir = "k6_final/k6_dashboard_name/linux-mssql-dashboard" // Working directory

	// Set up process for potential cancellation
	h.mutex.Lock()
	h.cmd = cmd
	h.mutex.Unlock()

	output, err := cmd.CombinedOutput()

	// Check if context was cancelled due to timeout
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("script execution timed out after 2 hours")
		logger.Error().Str("module", "k6").Msg("Combined script execution timed out")
	}

	h.mutex.Lock()
	if err != nil {
		h.status.LastError = err.Error()
		logger.Error().Err(err).Str("module", "k6").Msg("Combined script execution failed")
		h.updateStatus("❌ K6 test execution failed")
	} else {
		// Check if script completed successfully
		outputStr := string(output)
		if strings.Contains(outputStr, "✅ Dashboard data inserted into ClickHouse.") &&
		   strings.Contains(outputStr, "✅ Login data inserted into ClickHouse.") {
			logger.LogWithNode("System", "k6", "Combined script execution completed successfully", "success")
			h.updateStatus("✅ K6 test execution completed successfully")
		} else {
			logger.LogWithNode("System", "k6", "Combined script execution completed with potential issues", "warning")
			h.updateStatus("⚠️ K6 test execution completed with warnings")
		}
	}

	// Log output for debugging
	if len(output) > 0 {
		logger.LogWithNode("System", "k6", fmt.Sprintf("Combined script output: %s", string(output)), "info")
	}
	h.mutex.Unlock()
}

// readModuleConfig reads the enabled modules from conf.yml
func (h *K6Handler) readModuleConfig() ([]string, error) {
	configPath := "src/migrate/conf.d/conf.yml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read module config: %v", err)
	}

	var moduleConfig struct {
		DataGenerationTime struct {
			Type string `yaml:"type"`
		} `yaml:"data_generation_time"`
		IncludeModuleDirs map[string]struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"include_module_dirs"`
		Logging   interface{} `yaml:"logging"`
		Output    interface{} `yaml:"output"`
	}

	if err := yaml.Unmarshal(data, &moduleConfig); err != nil {
		return nil, fmt.Errorf("failed to parse module config: %v", err)
	}

	var enabledModules []string
	for moduleName, moduleSettings := range moduleConfig.IncludeModuleDirs {
		if moduleSettings.Enabled {
			enabledModules = append(enabledModules, moduleName)
		}
	}

	return enabledModules, nil
}

// updateDashboardConfig updates k6_config.yaml based on enabled modules
func (h *K6Handler) updateDashboardConfig(enabledModules []string) error {
	// Read module-dashboard mapping
	mappingPath := "src/configs/module_dashboard_mapping.yaml"
	mappingData, err := os.ReadFile(mappingPath)
	if err != nil {
		return fmt.Errorf("failed to read module dashboard mapping: %v", err)
	}

	var mappingConfig ModuleDashboardMapping
	if err := yaml.Unmarshal(mappingData, &mappingConfig); err != nil {
		return fmt.Errorf("failed to parse module dashboard mapping: %v", err)
	}

	// Read current k6_config.yaml
	k6ConfigPath := "k6_final/k6_dashboard_name/k6_config.yaml"
	k6Data, err := os.ReadFile(k6ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read k6 config: %v", err)
	}

	var k6Config K6DashboardConfig
	if err := yaml.Unmarshal(k6Data, &k6Config); err != nil {
		return fmt.Errorf("failed to parse k6 config: %v", err)
	}

	// Create a set of enabled module names for quick lookup
	enabledModuleSet := make(map[string]bool)
	for _, module := range enabledModules {
		enabledModuleSet[module] = true
	}

	// Update dashboard enabled status based on module availability
	for i := range k6Config.Dashboards {
		dashboard := &k6Config.Dashboards[i]
		// Check if any module maps to this dashboard
		enabled := false
		for moduleName, mapping := range mappingConfig.ModuleDashboardMapping {
			if mapping.DashboardSlug == dashboard.Slug && enabledModuleSet[moduleName] {
				enabled = true
				break
			}
		}
		dashboard.Enabled = enabled
	}

	// Write updated config back to file
	updatedData, err := yaml.Marshal(&k6Config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated k6 config: %v", err)
	}

	if err := os.WriteFile(k6ConfigPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated k6 config: %v", err)
	}

	return nil
}

// GetK6Logs handles GET /api/k6/logs
func (h *K6Handler) GetK6Logs(w http.ResponseWriter, r *http.Request) {
	// For now, return a simple message since we don't have persistent log storage
	// In a production system, you might want to implement log file reading
	logs := map[string]interface{}{
		"message": "K6 logs are streamed to main application logs",
		"status":  h.status,
		"note":    "Check main application logs for K6 execution details",
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    logs,
		Message: "K6 logs retrieved successfully",
	})
}

// Wrapper functions for API endpoints (following existing pattern)
func HandleAPIGetK6Config(w http.ResponseWriter, r *http.Request) {
	K6Manager.GetK6Config(w, r)
}

func HandleAPIUpdateK6Config(w http.ResponseWriter, r *http.Request) {
	K6Manager.UpdateK6Config(w, r)
}

func HandleAPIGetK6Status(w http.ResponseWriter, r *http.Request) {
	K6Manager.GetK6Status(w, r)
}

func HandleAPIStartK6Test(w http.ResponseWriter, r *http.Request) {
	K6Manager.StartK6Test(w, r)
}

func HandleAPIStopK6Test(w http.ResponseWriter, r *http.Request) {
	K6Manager.StopK6Test(w, r)
}

func HandleAPIResetK6Config(w http.ResponseWriter, r *http.Request) {
	K6Manager.ResetK6Config(w, r)
}

func HandleAPIGetK6Logs(w http.ResponseWriter, r *http.Request) {
	K6Manager.GetK6Logs(w, r)
}

func HandleAPIRunCombinedScript(w http.ResponseWriter, r *http.Request) {
	K6Manager.RunCombinedScript(w, r)
}