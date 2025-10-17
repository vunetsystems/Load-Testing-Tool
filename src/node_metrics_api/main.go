package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Application configuration
const (
	DefaultPort     = "8086"
	MetricsInterval = 1 * time.Second
)

// FinalVuDataSimMetrics represents metrics for the finalvudatasim process
type FinalVuDataSimMetrics struct {
	Running    bool      `json:"running"`
	PID        int       `json:"pid,omitempty"`
	StartTime  string    `json:"start_time,omitempty"`
	CPUPercent float64   `json:"cpu_percent,omitempty"`
	MemMB      float64   `json:"mem_mb,omitempty"`
	Cmdline    string    `json:"cmdline,omitempty"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
}

// SystemMetrics represents basic system metrics (removed - only process metrics now)

// MetricsCollector handles process metrics collection
type MetricsCollector struct {
	currentMetrics FinalVuDataSimMetrics
	mutex          sync.RWMutex
	nodeID         string
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(nodeID string) *MetricsCollector {
	if nodeID == "" {
		// Generate node ID from hostname if not provided
		hostname, _ := os.Hostname()
		nodeID = hostname
	}
	return &MetricsCollector{nodeID: nodeID}
}

// collectMetrics runs in background to collect system metrics
func (mc *MetricsCollector) collectMetrics() {
	ticker := time.NewTicker(MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		mc.updateMetrics()
	}
}

// updateMetrics collects current system metrics
func (mc *MetricsCollector) updateMetrics() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metrics := FinalVuDataSimMetrics{}
	output, err := exec.Command("pgrep", "-f", "finalvudatasim").Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		// Find the actual finalvudatasim process (not wrapper processes)
		// Since pgrep finds both processes, we need to check each one
		// The actual binary process should be the one with the exact command "./finalvudatasim"
		var actualPid string
		for _, line := range lines {
			pidStr := strings.TrimSpace(line)
			if pidStr != "" {
				// Check if this is the actual binary process
				psCheck, _ := exec.Command("ps", "-p", pidStr, "-o", "cmd=").Output()
				cmdLine := strings.TrimSpace(string(psCheck))
				// Look for processes where the command is exactly "./finalvudatasim"
				if cmdLine == "./finalvudatasim" {
					actualPid = pidStr
					break
				}
			}
		}

		// If we didn't find the exact match, try to find the process with highest CPU usage
		// as a fallback (the actual working process)
		if actualPid == "" {
			var highestPid string
			var highestCpu float64 = 0
			for _, line := range lines {
				pidStr := strings.TrimSpace(line)
				if pidStr != "" {
					psOut, _ := exec.Command("ps", "-p", pidStr, "-o", "pcpu=").Output()
					psLines := strings.Split(strings.TrimSpace(string(psOut)), "\n")
					if len(psLines) >= 2 {
						dataLine := strings.TrimSpace(psLines[1])
						if cpu, err := strconv.ParseFloat(dataLine, 64); err == nil && cpu > highestCpu {
							highestCpu = cpu
							highestPid = pidStr
						}
					}
				}
			}
			if highestPid != "" {
				actualPid = highestPid
			}
		}

		if actualPid != "" {
			pid, err := strconv.Atoi(actualPid)
			if err == nil {
				metrics.Running = true
				metrics.PID = pid

				// Get process start time
				startTimeOut, _ := exec.Command("ps", "-p", actualPid, "-o", "lstart=").Output()
				metrics.StartTime = strings.TrimSpace(string(startTimeOut))

				// Get CPU and memory usage - use more detailed ps command
				psOut, _ := exec.Command("ps", "-p", actualPid, "-o", "pcpu,rss,cmd").Output()
				log.Printf("Raw ps output for PID %s: %q", actualPid, string(psOut))

				psLines := strings.Split(strings.TrimSpace(string(psOut)), "\n")
				log.Printf("ps lines: %v", psLines)

				if len(psLines) >= 2 {
					// Skip header line and get the actual data
					dataLine := psLines[1]
					log.Printf("Data line: %q", dataLine)
					psFields := strings.Fields(dataLine)
					log.Printf("Parsed fields: %v", psFields)

					if len(psFields) >= 3 {
						if cpu, err := strconv.ParseFloat(psFields[0], 64); err == nil {
							metrics.CPUPercent = cpu
							log.Printf("Parsed CPU: %f", cpu)
						}
						if memKB, err := strconv.ParseFloat(psFields[1], 64); err == nil {
							metrics.MemMB = memKB / 1024.0
							log.Printf("Parsed memory: %f KB -> %f MB", memKB, metrics.MemMB)
						}
						metrics.Cmdline = strings.Join(psFields[2:], " ")
						log.Printf("Parsed cmdline: %s", metrics.Cmdline)
					}
				}
			}
		} else {
			metrics.Running = false
			metrics.PID = 0
			metrics.StartTime = ""
			metrics.CPUPercent = 0
			metrics.MemMB = 0
			metrics.Cmdline = ""
		}
	} else {
		metrics.Running = false
		metrics.PID = 0
		metrics.StartTime = ""
		metrics.CPUPercent = 0
		metrics.MemMB = 0
		metrics.Cmdline = ""
	}
	metrics.Timestamp = time.Now()

	// Store only process metrics
	mc.currentMetrics = metrics
}

// GetCurrentMetrics returns the current metrics (thread-safe)
func (mc *MetricsCollector) GetCurrentMetrics() FinalVuDataSimMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.currentMetrics
}

// HTTP handler for /api/system/metrics
func (mc *MetricsCollector) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Add CORS headers to allow requests from main manager
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	metrics := mc.GetCurrentMetrics()

	resp := map[string]interface{}{
		"nodeId":      mc.nodeID,
		"timestamp":   metrics.Timestamp,
		"running":     metrics.Running,
		"pid":         metrics.PID,
		"start_time":  metrics.StartTime,
		"cpu_percent": metrics.CPUPercent,
		"mem_mb":      metrics.MemMB,
		"cmdline":     metrics.Cmdline,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding metrics JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Served metrics for node %s", mc.nodeID)
}

// HTTP handler for /api/system/health
func (mc *MetricsCollector) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Add CORS headers to allow requests from main manager
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "healthy",
		"nodeId":    mc.nodeID,
		"timestamp": time.Now(),
		"uptime":    time.Since(mc.currentMetrics.Timestamp).String(),
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Error encoding health JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// getNodeIDFromEnv gets node ID from environment variable or generates from hostname
func getNodeIDFromEnv() string {
	if nodeID := os.Getenv("NODE_ID"); nodeID != "" {
		return nodeID
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v", err)
		return "unknown-node"
	}

	return hostname
}

// findAvailablePort finds the first available port starting from the default port
func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ { // Try up to 100 ports
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
		// If error is not "address already in use", might be other issue, but for now assume it's occupied
	}
	return 0, fmt.Errorf("no available ports found starting from %d", startPort)
}

func main() {
	// Parse command line flags
	portFlag := flag.String("port", "", "Port to listen on (optional, will find available if not specified)")
	flag.Parse()

	// Determine starting port
	startPortStr := *portFlag
	if startPortStr == "" {
		startPortStr = os.Getenv("METRICS_PORT")
	}
	if startPortStr == "" {
		startPortStr = DefaultPort
	}

	startPort, err := strconv.Atoi(startPortStr)
	if err != nil {
		log.Fatalf("Invalid port: %s", startPortStr)
	}

	// Find available port starting from the specified port
	port, err := findAvailablePort(startPort)
	if err != nil {
		log.Fatalf("Failed to find available port: %v", err)
	}

	portStr := strconv.Itoa(port)

	nodeID := getNodeIDFromEnv()

	log.Printf("Starting Node Metrics API server...")
	log.Printf("Node ID: %s", nodeID)
	log.Printf("Port: %s", portStr)

	// Write the port to a file for the master node to read
	if err := os.WriteFile("metrics.port", []byte(portStr), 0644); err != nil {
		log.Printf("Warning: Failed to write port to file: %v", err)
	}

	// Create metrics collector
	collector := NewMetricsCollector(nodeID)

	// Start background metrics collection
	go collector.collectMetrics()

	// Set up HTTP routes
	http.HandleFunc("/api/system/metrics", collector.handleMetrics)
	http.HandleFunc("/api/system/health", collector.handleHealth)

	// Add health check for root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers to allow requests from main manager
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "Node Metrics API is running",
			"nodeId":  nodeID,
			"version": "1.0.0",
		})
	})

	// Start server - explicitly bind to all interfaces (0.0.0.0)
	log.Printf("Server listening on port %s", portStr)
	log.Printf("Metrics endpoint: http://0.0.0.0:%s/api/system/metrics", portStr)
	log.Printf("Health endpoint: http://0.0.0.0:%s/api/system/health", portStr)

	// Explicitly bind to 0.0.0.0 to ensure IPv4 connectivity
	if err := http.ListenAndServe("0.0.0.0:"+portStr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
