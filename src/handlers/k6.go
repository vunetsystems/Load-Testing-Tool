package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	"vuDataSim/src/database"
	"vuDataSim/src/logger"
	"vuDataSim/src/models"
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
	LastUpdated       time.Time `json:"lastUpdated,omitempty"`
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

// K6ScriptParams represents parameters for K6 script execution
type K6ScriptParams struct {
	TimeRangesCsv string `json:"timeRangesCsv"`
	VUs           int    `json:"vus"`
	Duration      string `json:"duration"`
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
		logger.LogError("System", "k6", fmt.Sprintf("Failed to read K6 config file: %v", err))
		return
	}

	var config K6Config
	if err := json.Unmarshal(data, &config); err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to parse K6 config file: %v", err))
		return
	}

	h.mutex.Lock()
	h.config = config
	h.status.CurrentUserCount = config.GlobalUserCount
	h.mutex.Unlock()

	logger.LogSuccess("System", "k6", "K6 configuration loaded successfully")
}

// saveConfig saves current K6 configuration to file
func (h *K6Handler) saveConfig() {
	configPath := "src/k6_config.json"
	data, err := json.MarshalIndent(h.config, "", "  ")
	if err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to marshal K6 config: %v", err))
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to write K6 config file: %v", err))
		return
	}

	logger.LogSuccess("System", "k6", "K6 configuration saved successfully")
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
			logger.LogError("System", "k6", fmt.Sprintf("Failed to kill K6 process: %v", err))
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
	logger.LogWithNode("System", "k6", "Validating K6 configuration parameters", "info")

	if config.GlobalUserCount < 1 || config.GlobalUserCount > 1000 {
		logger.LogWarning("System", "k6", fmt.Sprintf("Invalid global user count: %d (must be between 1 and 1000)", config.GlobalUserCount))
		return fmt.Errorf("global user count must be between 1 and 1000")
	}

	if config.TestDuration == "" {
		logger.LogWarning("System", "k6", "Test duration is required but not provided")
		return fmt.Errorf("test duration is required")
	}

	if config.RampUpDuration < 1 {
		logger.LogWarning("System", "k6", fmt.Sprintf("Invalid ramp up duration: %d (must be at least 1 second)", config.RampUpDuration))
		return fmt.Errorf("ramp up duration must be at least 1 second")
	}

	if config.MaxDuration < 1 {
		logger.LogWarning("System", "k6", fmt.Sprintf("Invalid max duration: %d (must be at least 1 second)", config.MaxDuration))
		return fmt.Errorf("max duration must be at least 1 second")
	}

	if config.IntervalBetweenTests < 0 {
		logger.LogWarning("System", "k6", fmt.Sprintf("Invalid interval between tests: %d (cannot be negative)", config.IntervalBetweenTests))
		return fmt.Errorf("interval between tests cannot be negative")
	}

	if len(config.EnabledScripts) == 0 {
		logger.LogWarning("System", "k6", "No scripts enabled - at least one script must be enabled")
		return fmt.Errorf("at least one script must be enabled")
	}

	logger.LogSuccess("System", "k6", "K6 configuration validation passed")
	return nil
}

// generateK6Script generates a dynamic K6 script based on current configuration
func (h *K6Handler) generateK6Script() (string, error) {
	logger.LogWithNode("System", "k6", "Generating dynamic K6 script based on current configuration", "info")

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
		logger.LogError("System", "k6", fmt.Sprintf("Failed to write dynamic script to %s: %v", scriptPath, err))
		return "", fmt.Errorf("failed to write dynamic script: %v", err)
	}

	logger.LogSuccess("System", "k6", fmt.Sprintf("Dynamic K6 script generated successfully at %s", scriptPath))
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
		logger.LogError("System", "k6", fmt.Sprintf("K6 script execution failed: %v", err))
	} else {
		logger.LogSuccess("System", "k6", "K6 script execution completed successfully")
	}

	// Log output for debugging
	if len(output) > 0 {
		logger.LogWithNode("System", "k6", fmt.Sprintf("K6 script output: %s", string(output)), "info")
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
// RunCombinedScript handles API requests to start the K6 combined script
func (h *K6Handler) RunCombinedScript(w http.ResponseWriter, r *http.Request) {
    // Decode request
    var params K6ScriptParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        logger.LogError("System", "k6", fmt.Sprintf("Invalid request body: %v", err))
        SendJSONResponse(w, http.StatusBadRequest, APIResponse{
            Success: false,
            Message: fmt.Sprintf("Invalid request body: %v", err),
        })
        return
    }

    // Normalize values
    if params.VUs <= 0 {
        params.VUs = 1
    }
    if params.Duration == "" {
        params.Duration = "60m"
    }

    // Lock only to check/set shared state
    h.mutex.Lock()
    if h.status.IsRunning {
        // Double-check if old process still valid
        processRunning := false
        if h.cmd != nil && h.cmd.Process != nil {
            if err := h.cmd.Process.Signal(syscall.Signal(0)); err == nil {
                processRunning = true
            }
        }

        if processRunning {
            h.mutex.Unlock()
            SendJSONResponse(w, http.StatusConflict, APIResponse{
                Success: false,
                Message: "K6 test is already running",
            })
            return
        }

        // Cleanup stale process info
        if h.cmd != nil && h.cmd.Process != nil {
            _ = h.cmd.Process.Kill()
            _, _ = h.cmd.Process.Wait()
        }
        h.cmd = nil
        h.status.IsRunning = false
        h.status.CurrentScript = ""
    }

    // Mark test as running
    h.status.IsRunning = true
    h.status.CurrentScript = "combined.sh"
    h.status.LastUpdated = time.Now()
    h.mutex.Unlock()

    // Read enabled modules for K6 run tracking
    enabledModules, err := h.readModuleConfig()
    if err != nil {
        logger.LogError("System", "k6", fmt.Sprintf("Failed to read module config: %v", err))
        // Continue execution even if module config fails
    }

    // Create K6 run record - using 0 for iterations and interval since they're not used in phase11.sh
    k6Run, err := database.CreateK6Run(params.TimeRangesCsv, params.VUs, 0, 0, enabledModules)
    if err != nil {
        logger.LogError("System", "k6", fmt.Sprintf("Failed to create K6 run record: %v", err))
        // Continue execution even if database operation fails
    } else {
        logger.LogWithNode("System", "k6", fmt.Sprintf("Created K6 run record with ID: %s", k6Run.TestID), "info")
    }

    // Add config sync before execution
    if enabledModules != nil {
        err = h.updateDashboardConfig(enabledModules)
        if err != nil {
            logger.LogError("System", "k6", fmt.Sprintf("Failed to update dashboard config: %v", err))
            // Handle error appropriately
        }
    }

    logger.LogWithNode("System", "k6", fmt.Sprintf("Starting phase11 script with VUs=%d, Duration=%s, TimeRangesCsv=%s", params.VUs, params.Duration, params.TimeRangesCsv), "info")

    // Check and create users if needed
    userCheckResult := h.checkAndCreateUsers(params.VUs)
    logger.LogWithNode("System", "k6", fmt.Sprintf("User check result: %s", userCheckResult), "info")

    // Run asynchronously with K6 run tracking
    h.executePhase11ScriptWithTracking(params.TimeRangesCsv, params.VUs, params.Duration, k6Run)

    SendJSONResponse(w, http.StatusOK, APIResponse{
        Success: true,
        Message: fmt.Sprintf("K6 test started successfully at %s", time.Now().Format(time.RFC3339)),
        Data: map[string]interface{}{
            "test_id": k6Run.TestID,
        },
    })
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
	  logger.LogError("System", "k6", fmt.Sprintf("Failed to read timeout.txt: %v", err))
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
	  logger.LogError("System", "k6", fmt.Sprintf("User creation script failed: %v, output: %s", err, string(output)))
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

	logger.LogWarning("System", "k6", fmt.Sprintf("User creation may not have completed successfully, output: %s", string(output)))
	return fmt.Sprintf("User creation completed (verification inconclusive)")
}

// executeCombinedScriptWithTracking executes the combined.sh script with given parameters and tracks the K6 run
func (h *K6Handler) executeCombinedScriptWithTracking(timeRange string, vus, iterations, interval int, k6Run *models.K6Run) {
    // Context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
    defer cancel()

    // Prepare the command (kill entire group on cancel)
    cmd := exec.CommandContext(ctx, "bash", "-c",
        fmt.Sprintf("exec ./combined.sh %s %d %d %d", timeRange, vus, iterations, interval))
    cmd.Dir = "/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // ensure we can kill all children
    h.cmd = cmd

    logger.LogWithNode("System", "k6", "About to execute combined.sh", "info")

    start := time.Now()
    output, err := cmd.CombinedOutput()
    duration := time.Since(start)

    logger.LogWithNode("System", "k6", fmt.Sprintf("combined.sh finished in %s", duration.String()), "info")

    if ctx.Err() == context.DeadlineExceeded {
        // kill process group if context timed out
        if cmd.Process != nil {
            _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
            logger.LogError("System", "k6", "Killed combined.sh due to timeout")
        }
    }

    // Prepare log file path
    logDir := "/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/logs"
    logPath := filepath.Join(logDir, fmt.Sprintf("combined_output_%d.log", time.Now().Unix()))
    if err := os.MkdirAll(logDir, 0755); err == nil {
        if writeErr := os.WriteFile(logPath, output, 0644); writeErr == nil {
            logger.LogSuccess("System", "k6", fmt.Sprintf("K6 combined script output saved to %s", logPath))
        } else {
            logger.LogError("System", "k6", fmt.Sprintf("Failed to write combined output: %v", writeErr))
        }
    } else {
        logger.Error().Err(err).Msg("Failed to create logs directory")
    }

    // Update K6 run status based on execution result
    if k6Run != nil {
        if err != nil {
            // Mark as failed if there was an error
            if updateErr := database.UpdateK6RunStatus(k6Run.TestID, "failed"); updateErr != nil {
                logger.LogError("System", "k6", fmt.Sprintf("Failed to update K6 run status to failed: %v", updateErr))
            }
        } else {
            // Mark as completed if successful
            if completeErr := database.CompleteK6Run(k6Run.TestID); completeErr != nil {
                logger.LogError("System", "k6", fmt.Sprintf("Failed to complete K6 run: %v", completeErr))
            } else {
                logger.LogSuccess("System", "k6", fmt.Sprintf("K6 run %s completed successfully", k6Run.TestID))
            }
        }
    }

    if err != nil {
        logger.LogError("System", "k6", fmt.Sprintf("K6 combined script execution failed: %v", err))
    }

    // Update status safely after run
    h.mutex.Lock()
    h.status.IsRunning = false
    h.status.CurrentScript = ""
    h.status.LastUpdated = time.Now()
    h.cmd = nil
    h.mutex.Unlock()

    logger.LogSuccess("System", "k6", "K6 combined script execution completed and state reset")
   }
   
   // executePhase11ScriptWithTracking executes the phase11.sh script with given parameters and tracks the K6 run
   func (h *K6Handler) executePhase11ScriptWithTracking(timeRangesCsv string, vus int, duration string, k6Run *models.K6Run) {
       // Context with timeout (longer timeout for phase11.sh)
       ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
       defer cancel()
   
       // Prepare the command (kill entire group on cancel)
       cmd := exec.CommandContext(ctx, "bash", "-c",
           fmt.Sprintf("exec ./phase11.sh \"%s\" %d %s", timeRangesCsv, vus, duration))
       cmd.Dir = "/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
       cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // ensure we can kill all children
       h.cmd = cmd
   
       logger.LogWithNode("System", "k6", "About to execute phase11.sh", "info")
   
       start := time.Now()
       output, err := cmd.CombinedOutput()
       duration_exec := time.Since(start)
   
       logger.LogWithNode("System", "k6", fmt.Sprintf("phase11.sh finished in %s", duration_exec.String()), "info")
   
       if ctx.Err() == context.DeadlineExceeded {
           // kill process group if context timed out
           if cmd.Process != nil {
               _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
               logger.LogError("System", "k6", "Killed phase11.sh due to timeout")
           }
       }
   
       // Prepare log file path
       logDir := "/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/logs"
       logPath := filepath.Join(logDir, fmt.Sprintf("phase11_output_%d.log", time.Now().Unix()))
       if err := os.MkdirAll(logDir, 0755); err == nil {
           if writeErr := os.WriteFile(logPath, output, 0644); writeErr == nil {
               logger.LogSuccess("System", "k6", fmt.Sprintf("K6 phase11 script output saved to %s", logPath))
           } else {
               logger.LogError("System", "k6", fmt.Sprintf("Failed to write phase11 output: %v", writeErr))
           }
       } else {
           logger.Error().Err(err).Msg("Failed to create logs directory")
       }
   
       // Update K6 run status based on execution result
       if k6Run != nil {
           if err != nil {
               // Mark as failed if there was an error
               if updateErr := database.UpdateK6RunStatus(k6Run.TestID, "failed"); updateErr != nil {
                   logger.LogError("System", "k6", fmt.Sprintf("Failed to update K6 run status to failed: %v", updateErr))
               }
           } else {
               // Mark as completed if successful
               if completeErr := database.CompleteK6Run(k6Run.TestID); completeErr != nil {
                   logger.LogError("System", "k6", fmt.Sprintf("Failed to complete K6 run: %v", completeErr))
               } else {
                   logger.LogSuccess("System", "k6", fmt.Sprintf("K6 run %s completed successfully", k6Run.TestID))
               }
           }
       }
   
       if err != nil {
           logger.LogError("System", "k6", fmt.Sprintf("K6 phase11 script execution failed: %v", err))
       }
   
       // Update status safely after run
       h.mutex.Lock()
       h.status.IsRunning = false
       h.status.CurrentScript = ""
       h.status.LastUpdated = time.Now()
       h.cmd = nil
       h.mutex.Unlock()
   
       logger.LogSuccess("System", "k6", "K6 phase11 script execution completed and state reset")
   }


// readModuleConfig reads the enabled modules from conf.yml
func (h *K6Handler) readModuleConfig() ([]string, error) {
	logger.LogWithNode("System", "k6", "Reading enabled modules from configuration", "info")

	configPath := "src/migrate/conf.d/conf.yml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to read module config from %s: %v", configPath, err))
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
		logger.LogError("System", "k6", fmt.Sprintf("Failed to parse module config YAML: %v", err))
		return nil, fmt.Errorf("failed to parse module config: %v", err)
	}

	var enabledModules []string
	for moduleName, moduleSettings := range moduleConfig.IncludeModuleDirs {
		if moduleSettings.Enabled {
			enabledModules = append(enabledModules, moduleName)
		}
	}

	logger.LogSuccess("System", "k6", fmt.Sprintf("Successfully read %d enabled modules from configuration", len(enabledModules)))
	return enabledModules, nil
}

// updateDashboardConfig updates k6_config.yaml based on enabled modules
func (h *K6Handler) updateDashboardConfig(enabledModules []string) error {
	logger.LogWithNode("System", "k6", fmt.Sprintf("Updating dashboard config for %d enabled modules", len(enabledModules)), "info")

	// Read module-dashboard mapping
	mappingPath := "src/configs/module_dashboard_mapping.yaml"
	mappingData, err := os.ReadFile(mappingPath)
	if err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to read module dashboard mapping from %s: %v", mappingPath, err))
		return fmt.Errorf("failed to read module dashboard mapping: %v", err)
	}

	var mappingConfig ModuleDashboardMapping
	if err := yaml.Unmarshal(mappingData, &mappingConfig); err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to parse module dashboard mapping YAML: %v", err))
		return fmt.Errorf("failed to parse module dashboard mapping: %v", err)
	}

	// Read current k6_config.yaml
	k6ConfigPath := "k6_final/k6_dashboard_name/k6_config.yaml"
	k6Data, err := os.ReadFile(k6ConfigPath)
	if err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to read k6 config from %s: %v", k6ConfigPath, err))
		return fmt.Errorf("failed to read k6 config: %v", err)
	}

	var k6Config K6DashboardConfig
	if err := yaml.Unmarshal(k6Data, &k6Config); err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to parse k6 config YAML: %v", err))
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
		logger.LogError("System", "k6", fmt.Sprintf("Failed to marshal updated k6 config: %v", err))
		return fmt.Errorf("failed to marshal updated k6 config: %v", err)
	}

	if err := os.WriteFile(k6ConfigPath, updatedData, 0644); err != nil {
		logger.LogError("System", "k6", fmt.Sprintf("Failed to write updated k6 config to %s: %v", k6ConfigPath, err))
		return fmt.Errorf("failed to write updated k6 config: %v", err)
	}

	logger.LogSuccess("System", "k6", fmt.Sprintf("Dashboard configuration updated successfully for %d enabled modules", len(enabledModules)))
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

// HandleAPIGetNextK6TestID handles GET /api/k6/next-test-id
func HandleAPIGetNextK6TestID(w http.ResponseWriter, r *http.Request) {
	nextID, err := database.GetNextK6TestID()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to generate next K6 test ID: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"next_test_id": nextID,
		},
		Message: "Next K6 test ID generated successfully",
	})
}