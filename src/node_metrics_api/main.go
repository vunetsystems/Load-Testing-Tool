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
		pidStr := lines[0]
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			metrics.Running = true
			metrics.PID = pid

			// Get process start time
			startTimeOut, _ := exec.Command("ps", "-p", pidStr, "-o", "lstart=").Output()
			metrics.StartTime = strings.TrimSpace(string(startTimeOut))

			// Get CPU and memory usage
			psOut, _ := exec.Command("ps", "-p", pidStr, "-o", "%cpu,rss,cmd").Output()
			psFields := strings.Fields(string(psOut))
			if len(psFields) >= 3 {
				metrics.CPUPercent, _ = strconv.ParseFloat(psFields[0], 64)
				memKB, _ := strconv.ParseFloat(psFields[1], 64)
				metrics.MemMB = memKB / 1024.0
				metrics.Cmdline = strings.Join(psFields[2:], " ")
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

	// Start server
	log.Printf("Server listening on port %s", portStr)
	log.Printf("Metrics endpoint: http://localhost:%s/api/system/metrics", portStr)
	log.Printf("Health endpoint: http://localhost:%s/api/system/health", portStr)

	if err := http.ListenAndServe(":"+portStr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
