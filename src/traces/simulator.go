package traces

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"vuDataSim/src/logger"
)

// TracesSimulator manages the traces simulation process
type TracesSimulator struct {
	mu          sync.RWMutex
	running     bool
	testName    string
	eps         int
	minutes     int
	startedAt   time.Time
	cmd         *exec.Cmd
	stopChannel chan struct{}
}

// TracesStatus represents the current status of the traces simulator
type TracesStatus struct {
	Running   bool      `json:"running"`
	TestName  string    `json:"test_name,omitempty"`
	EPS       int       `json:"eps,omitempty"`
	Minutes   int       `json:"minutes,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// TracesStartRequest represents the request to start a traces simulation
type TracesStartRequest struct {
	TestName string `json:"test_name"`
	EPS      int    `json:"eps"`
	Minutes  int    `json:"minutes"`
}

// Global simulator instance
var Simulator = &TracesSimulator{
	stopChannel: make(chan struct{}),
}

// Start begins a new traces simulation
func (ts *TracesSimulator) Start(testName string, eps int, minutes int) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return fmt.Errorf("traces simulation is already running")
	}

	// Validate inputs
	if testName == "" {
		return fmt.Errorf("test name is required")
	}
	if eps <= 0 {
		return fmt.Errorf("EPS must be greater than 0")
	}
	if minutes <= 0 {
		return fmt.Errorf("minutes must be greater than 0")
	}

	// Check if traces_simulator directory exists
	if _, err := os.Stat("traces_simulator"); os.IsNotExist(err) {
		return fmt.Errorf("traces_simulator directory not found")
	}

	// Check if main.go exists
	if _, err := os.Stat("traces_simulator/main.go"); os.IsNotExist(err) {
		return fmt.Errorf("traces_simulator/main.go not found")
	}

	// Build the command
	ts.cmd = exec.Command("go", "run", "main.go",
		fmt.Sprintf("--test-name=%s", testName),
		fmt.Sprintf("--eps=%d", eps),
		fmt.Sprintf("--minutes=%d", minutes),
	)
	ts.cmd.Dir = "traces_simulator"

	// Set process group so we can kill all child processes
	ts.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture output for logging
	ts.cmd.Stdout = os.Stdout
	ts.cmd.Stderr = os.Stderr

	// Start the process
	if err := ts.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start traces simulator: %w", err)
	}

	// Update state
	ts.running = true
	ts.testName = testName
	ts.eps = eps
	ts.minutes = minutes
	ts.startedAt = time.Now()
	ts.stopChannel = make(chan struct{})

	logger.Info().
		Str("test_name", testName).
		Int("eps", eps).
		Int("minutes", minutes).
		Msg("Traces simulation started")

	// Monitor the process in a goroutine
	go ts.monitor()

	return nil
}

// monitor watches the process and updates status when it completes
func (ts *TracesSimulator) monitor() {
	if ts.cmd == nil || ts.cmd.Process == nil {
		return
	}

	// Wait for the process to complete
	err := ts.cmd.Wait()

	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.running = false

	if err != nil {
		logger.Error().Err(err).Msg("Traces simulation completed with error")
	} else {
		logger.Info().
			Str("test_name", ts.testName).
			Int("eps", ts.eps).
			Int("minutes", ts.minutes).
			Msg("Traces simulation completed successfully")
	}

	ts.cmd = nil
}

// Stop terminates the running traces simulation
func (ts *TracesSimulator) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return fmt.Errorf("no traces simulation is running")
	}

	if ts.cmd != nil && ts.cmd.Process != nil {
		// Get the process group ID (same as the process ID since we set Setpgid: true)
		pgid := ts.cmd.Process.Pid

		// Send SIGTERM to the entire process group (negative PID kills the group)
		// This kills go run main.go AND its child process (otel-trace)
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			logger.Warn().Err(err).Msg("SIGTERM to process group failed, trying SIGKILL")
			// If SIGTERM fails, try SIGKILL on the process group
			if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
				return fmt.Errorf("failed to stop traces simulator process group: %w", killErr)
			}
		}
	}

	ts.running = false
	logger.Info().Msg("Traces simulation stopped by user (process group terminated)")

	return nil
}

// Status returns the current status of the traces simulator
func (ts *TracesSimulator) Status() TracesStatus {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	status := TracesStatus{
		Running: ts.running,
	}

	if ts.running {
		status.TestName = ts.testName
		status.EPS = ts.eps
		status.Minutes = ts.minutes
		status.StartedAt = ts.startedAt
		status.Message = "Traces simulation is running"
	} else {
		status.Message = "No traces simulation is running"
	}

	return status
}

// IsRunning returns whether a simulation is currently running
func (ts *TracesSimulator) IsRunning() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.running
}
