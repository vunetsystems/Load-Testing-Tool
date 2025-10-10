package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
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

// SystemMetrics represents the complete metrics payload
type SystemMetrics struct {
	NodeID    string    `json:"nodeId"`
	Timestamp time.Time `json:"timestamp"`
	System    struct {
		CPU    CPUInfo    `json:"cpu"`
		Memory MemoryInfo `json:"memory"`
		Uptime int64      `json:"uptime_seconds"`
	} `json:"system"`
}

// CPUInfo represents CPU metrics
type CPUInfo struct {
	UsedPercent float64 `json:"used_percent"`
	Cores       int     `json:"cores"`
	Load1M      float64 `json:"load_1m"`
}

// MemoryInfo represents memory metrics
type MemoryInfo struct {
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	TotalGB     float64 `json:"total_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// MetricsCollector handles system metrics collection
type MetricsCollector struct {
	currentMetrics SystemMetrics
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

	mc := &MetricsCollector{
		nodeID: nodeID,
	}

	// Initialize total CPU and memory
	mc.initializeSystemInfo()

	return mc
}

// initializeSystemInfo gets static system information
func (mc *MetricsCollector) initializeSystemInfo() {
	mc.currentMetrics.System.CPU.Cores = getCPUCores()
	mc.currentMetrics.System.Memory.TotalGB = getTotalMemoryGB()
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

	// Update timestamp
	mc.currentMetrics.Timestamp = time.Now()

	// Collect CPU metrics
	cpuUsage := getCPUUsagePercent()
	load1M := getLoadAverage1M()
	mc.currentMetrics.System.CPU.UsedPercent = cpuUsage
	mc.currentMetrics.System.CPU.Load1M = load1M

	// Collect memory metrics
	usedGB, availableGB := getMemoryUsageGB()
	mc.currentMetrics.System.Memory.UsedGB = usedGB
	mc.currentMetrics.System.Memory.AvailableGB = availableGB
	mc.currentMetrics.System.Memory.UsedPercent = (usedGB / mc.currentMetrics.System.Memory.TotalGB) * 100

	// Collect uptime
	mc.currentMetrics.System.Uptime = getSystemUptimeSeconds()
}

// GetCurrentMetrics returns the current metrics (thread-safe)
func (mc *MetricsCollector) GetCurrentMetrics() SystemMetrics {
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
	// Ensure node ID is set
	metrics.NodeID = mc.nodeID

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
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

// getCPUCores returns the total number of CPU cores
func getCPUCores() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Printf("Error reading /proc/cpuinfo: %v", err)
		return 4 // fallback
	}

	count := 0
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}

	if count == 0 {
		return 4 // fallback
	}

	return count
}

// getCPUUsagePercent returns current CPU usage as percentage
func getCPUUsagePercent() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		log.Printf("Error reading /proc/stat: %v", err)
		return 0
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 1 {
		return 0
	}

	// Parse first line: cpu user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(lines[0])
	if len(fields) < 8 {
		return 0
	}

	// Convert string values to integers
	values := make([]float64, 8)
	for i := 1; i < len(fields) && i <= 8; i++ {
		if val, err := strconv.ParseFloat(fields[i], 64); err == nil {
			values[i-1] = val
		}
	}

	// Calculate total time
	total := values[0] + values[1] + values[2] + values[3] + values[4] + values[5] + values[6] + values[7]

	// Calculate idle time (idle + iowait)
	idle := values[3] + values[4]

	// CPU usage = (total - idle) / total * 100
	if total == 0 {
		return 0
	}

	return ((total - idle) / total) * 100
}

// getLoadAverage1M returns the 1-minute load average
func getLoadAverage1M() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		log.Printf("Error reading /proc/loadavg: %v", err)
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}

	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		log.Printf("Error parsing load average: %v", err)
		return 0
	}

	return load
}

// getTotalMemoryGB returns total memory in GB
func getTotalMemoryGB() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		log.Printf("Error reading /proc/meminfo: %v", err)
		return 8.0 // fallback 8GB
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
					return kb / 1024 / 1024 // Convert KB to GB
				}
			}
		}
	}

	return 8.0 // fallback
}

// getMemoryUsageGB returns used and available memory in GB
func getMemoryUsageGB() (float64, float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		log.Printf("Error reading /proc/meminfo: %v", err)
		return 0, 8.0 // fallback
	}

	var memTotalKB, memAvailableKB float64

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memTotalKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memAvailableKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
	}

	if memTotalKB == 0 {
		return 0, 8.0 // fallback
	}

	usedKB := memTotalKB - memAvailableKB
	usedGB := usedKB / 1024 / 1024
	availableGB := memAvailableKB / 1024 / 1024

	return usedGB, availableGB
}

// getSystemUptimeSeconds returns system uptime in seconds
func getSystemUptimeSeconds() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		log.Printf("Error reading /proc/uptime: %v", err)
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		log.Printf("Error parsing uptime: %v", err)
		return 0
	}

	return int64(uptime)
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

func main() {
	// Parse command line flags
	portFlag := flag.String("port", "", "Port to listen on")
	flag.Parse()

	// Get configuration from command line flag, then environment variable, then default
	port := *portFlag
	if port == "" {
		port = os.Getenv("METRICS_PORT")
	}
	if port == "" {
		port = DefaultPort
	}

	nodeID := getNodeIDFromEnv()

	log.Printf("Starting Node Metrics API server...")
	log.Printf("Node ID: %s", nodeID)
	log.Printf("Port: %s", port)

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
	log.Printf("Server listening on port %s", port)
	log.Printf("Metrics endpoint: http://localhost:%s/api/system/metrics", port)
	log.Printf("Health endpoint: http://localhost:%s/api/system/health", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
