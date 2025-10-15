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
	DefaultPort     = "8085"
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

// SystemMetrics represents basic system metrics
type SystemMetrics struct {
	CPU struct {
		UsedPercent float64 `json:"used_percent"`
		Cores       int     `json:"cores"`
		Load1m      float64 `json:"load_1m"`
	} `json:"cpu"`
	Memory struct {
		UsedGB      float64 `json:"used_gb"`
		AvailableGB float64 `json:"available_gb"`
		TotalGB     float64 `json:"total_gb"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"memory"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}

// MetricsCollector handles system metrics collection
type MetricsCollector struct {
	currentMetrics       FinalVuDataSimMetrics
	currentSystemMetrics SystemMetrics
	mutex                sync.RWMutex
	nodeID               string
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

	// Collect system metrics
	sys := SystemMetrics{}
	// CPU
	cpuOut, _ := exec.Command("nproc").Output()
	cores, _ := strconv.Atoi(strings.TrimSpace(string(cpuOut)))
	sys.CPU.Cores = cores
	loadOut, _ := exec.Command("cat", "/proc/loadavg").Output()
	loadFields := strings.Fields(string(loadOut))
	if len(loadFields) > 0 {
		sys.CPU.Load1m, _ = strconv.ParseFloat(loadFields[0], 64)
	}
	psCpuOut, _ := exec.Command("top", "-bn1").Output()
	for _, line := range strings.Split(string(psCpuOut), "\n") {
		if strings.Contains(line, "Cpu(s)") {
			parts := strings.Split(line, ",")
			if len(parts) > 0 {
				cpuUsedStr := strings.Fields(parts[0])
				if len(cpuUsedStr) > 1 {
					val, _ := strconv.ParseFloat(cpuUsedStr[1], 64)
					sys.CPU.UsedPercent = val
				}
			}
		}
	}
	// Memory
	memOut, _ := exec.Command("free", "-g").Output()
	for _, line := range strings.Split(string(memOut), "\n") {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				total, _ := strconv.ParseFloat(fields[1], 64)
				used, _ := strconv.ParseFloat(fields[2], 64)
				free, _ := strconv.ParseFloat(fields[3], 64)
				sys.Memory.TotalGB = total
				sys.Memory.UsedGB = used
				sys.Memory.AvailableGB = free
				if total > 0 {
					sys.Memory.UsedPercent = (used / total) * 100
				}
			}
		}
	}
	// Uptime
	uptimeOut, _ := exec.Command("cat", "/proc/uptime").Output()
	uptimeFields := strings.Fields(string(uptimeOut))
	if len(uptimeFields) > 0 {
		uptimeSec, _ := strconv.ParseFloat(uptimeFields[0], 64)
		sys.UptimeSeconds = int64(uptimeSec)
	}

	// Store both process and system metrics
	mc.currentMetrics = metrics
	mc.currentSystemMetrics = sys
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

	w.Header().Set("Content-Type", "application/json")

	metrics := mc.GetCurrentMetrics()
	system := mc.currentSystemMetrics

	resp := map[string]interface{}{
		"nodeId":      mc.nodeID,
		"timestamp":   metrics.Timestamp,
		"running":     metrics.Running,
		"pid":         metrics.PID,
		"start_time":  metrics.StartTime,
		"cpu_percent": metrics.CPUPercent,
		"mem_mb":      metrics.MemMB,
		"cmdline":     metrics.Cmdline,
		"system":      system,
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
