package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Application version and configuration
const (
	AppVersion = "1.0.0"
	StaticDir  = "./static"
	Port       = ":3000"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin in development
	},
}

// Global application state
type AppState struct {
	IsSimulationRunning bool                    `json:"isSimulationRunning"`
	CurrentProfile      string                  `json:"currentProfile"`
	TargetEPS           int                     `json:"targetEps"`
	TargetKafka         int                     `json:"targetKafka"`
	TargetClickHouse    int                     `json:"targetClickHouse"`
	StartTime           time.Time               `json:"startTime"`
	NodeData            map[string]*NodeMetrics `json:"nodeData"`
	mutex               sync.RWMutex
	clients             map[*websocket.Conn]bool
	broadcast           chan []byte
}

// NodeMetrics represents metrics for a single node
type NodeMetrics struct {
	NodeID     string    `json:"nodeId"`
	Status     string    `json:"status"`
	EPS        int       `json:"eps"`
	KafkaLoad  int       `json:"kafkaLoad"`
	CHLoad     int       `json:"chLoad"`
	CPU        float64   `json:"cpu"`
	Memory     float64   `json:"memory"`
	LastUpdate time.Time `json:"lastUpdate"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SimulationConfig represents simulation configuration
type SimulationConfig struct {
	Profile          string `json:"profile"`
	TargetEPS        int    `json:"targetEps"`
	TargetKafka      int    `json:"targetKafka"`
	TargetClickHouse int    `json:"targetClickHouse"`
}

// Global application state instance
var appState = &AppState{
	IsSimulationRunning: false,
	CurrentProfile:      "medium",
	TargetEPS:           10000,
	TargetKafka:         5000,
	TargetClickHouse:    2000,
	NodeData:            make(map[string]*NodeMetrics),
	clients:             make(map[*websocket.Conn]bool),
	broadcast:           make(chan []byte, 256),
}

// Initialize node data
func init() {
	nodes := []string{"node1", "node2", "node3", "node4", "node5"}

	for _, nodeID := range nodes {
		status := "active"
		if nodeID == "node4" {
			status = "inactive"
		}

		appState.NodeData[nodeID] = &NodeMetrics{
			NodeID:     nodeID,
			Status:     status,
			EPS:        0,
			KafkaLoad:  0,
			CHLoad:     0,
			CPU:        0,
			Memory:     0,
			LastUpdate: time.Now(),
		}
	}

	// Set initial data for active nodes
	appState.NodeData["node1"].EPS = 9800
	appState.NodeData["node1"].KafkaLoad = 4900
	appState.NodeData["node1"].CHLoad = 1950
	appState.NodeData["node1"].CPU = 65
	appState.NodeData["node1"].Memory = 70

	appState.NodeData["node2"].EPS = 9900
	appState.NodeData["node2"].KafkaLoad = 4950
	appState.NodeData["node2"].CHLoad = 1980
	appState.NodeData["node2"].CPU = 70
	appState.NodeData["node2"].Memory = 75

	appState.NodeData["node3"].EPS = 10100
	appState.NodeData["node3"].KafkaLoad = 5050
	appState.NodeData["node3"].CHLoad = 2020
	appState.NodeData["node3"].CPU = 60
	appState.NodeData["node3"].Memory = 65

	appState.NodeData["node5"].EPS = 10000
	appState.NodeData["node5"].KafkaLoad = 5000
	appState.NodeData["node5"].CHLoad = 2000
	appState.NodeData["node5"].CPU = 75
	appState.NodeData["node5"].Memory = 80
}

// WebSocket handler for real-time updates
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	appState.mutex.Lock()
	appState.clients[conn] = true
	appState.mutex.Unlock()

	// Send initial state
	initialState, _ := json.Marshal(appState)
	conn.WriteMessage(websocket.TextMessage, initialState)

	// Listen for client messages
	for {
		var msg []byte
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		log.Printf("Received WebSocket message: %s", msg)
	}

	// Unregister client
	appState.mutex.Lock()
	delete(appState.clients, conn)
	appState.mutex.Unlock()
}

// Broadcast updates to all WebSocket clients
func (state *AppState) broadcastUpdate() {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	state.mutex.RLock()
	clients := make([]*websocket.Conn, 0, len(state.clients))
	for client := range state.clients {
		clients = append(clients, client)
	}
	state.mutex.RUnlock()

	for _, client := range clients {
		go func(c *websocket.Conn) {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				state.mutex.Lock()
				delete(state.clients, c)
				state.mutex.Unlock()
				c.Close()
			}
		}(client)
	}
}

// API endpoint to get current state
func getDashboardData(w http.ResponseWriter, r *http.Request) {
	appState.mutex.RLock()
	defer appState.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to start simulation
func startSimulation(w http.ResponseWriter, r *http.Request) {
	var config SimulationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if appState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "Simulation is already running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate configuration
	if config.TargetEPS < 1 || config.TargetEPS > 100000 {
		response := APIResponse{
			Success: false,
			Message: "Target EPS must be between 1 and 100,000",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update state
	appState.IsSimulationRunning = true
	appState.CurrentProfile = config.Profile
	appState.TargetEPS = config.TargetEPS
	appState.TargetKafka = config.TargetKafka
	appState.TargetClickHouse = config.TargetClickHouse
	appState.StartTime = time.Now()

	response := APIResponse{
		Success: true,
		Message: "Simulation started successfully",
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go appState.broadcastUpdate()

	log.Printf("Simulation started with profile: %s, Target EPS: %d", config.Profile, config.TargetEPS)
}

// API endpoint to stop simulation
func stopSimulation(w http.ResponseWriter, r *http.Request) {
	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if !appState.IsSimulationRunning {
		response := APIResponse{
			Success: false,
			Message: "No simulation is currently running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	appState.IsSimulationRunning = false

	response := APIResponse{
		Success: true,
		Message: "Simulation stopped successfully",
		Data:    appState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Broadcast update
	go appState.broadcastUpdate()

	log.Println("Simulation stopped")
}

// API endpoint to sync configuration
func syncConfiguration(w http.ResponseWriter, r *http.Request) {
	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	// In a real implementation, this would sync with external configuration sources
	response := APIResponse{
		Success: true,
		Message: "Configuration synced successfully",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
			"version":   AppVersion,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Println("Configuration synced")
}

// API endpoint to get logs
func getLogs(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	nodeFilter := r.URL.Query().Get("node")
	moduleFilter := r.URL.Query().Get("module")
	limit := r.URL.Query().Get("limit")

	// Parse limit
	limitNum := 50 // default
	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 && parsed <= 1000 {
			limitNum = parsed
		}
	}

	// Mock log data - in real implementation, this would come from a logging system
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-time.Minute * 30).Format("2006-01-02 15:04:05"),
			"node":      "Node 1",
			"module":    "Module A",
			"message":   "Starting simulation...",
			"type":      "info",
		},
		{
			"timestamp": time.Now().Add(-time.Minute * 25).Format("2006-01-02 15:04:05"),
			"node":      "Node 1",
			"module":    "Module A",
			"message":   "Simulation running...",
			"type":      "info",
		},
		{
			"timestamp": time.Now().Add(-time.Minute * 20).Format("2006-01-02 15:04:05"),
			"node":      "Node 2",
			"module":    "Module B",
			"message":   "Initializing...",
			"type":      "warning",
		},
	}

	// Apply filters
	filteredLogs := logs
	if nodeFilter != "" && nodeFilter != "All Nodes" {
		filtered := make([]map[string]interface{}, 0)
		for _, log := range logs {
			if log["node"] == nodeFilter {
				filtered = append(filtered, log)
			}
		}
		filteredLogs = filtered
	}

	if moduleFilter != "" && moduleFilter != "All Modules" {
		filtered := make([]map[string]interface{}, 0)
		for _, log := range filteredLogs {
			if log["module"] == moduleFilter {
				filtered = append(filtered, log)
			}
		}
		filteredLogs = filtered
	}

	// Apply limit
	if len(filteredLogs) > limitNum {
		filteredLogs = filteredLogs[:limitNum]
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"logs":  filteredLogs,
			"total": len(filteredLogs),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to update node metrics (for simulation)
func updateNodeMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeId"]

	var metrics NodeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	appState.mutex.Lock()
	defer appState.mutex.Unlock()

	if node, exists := appState.NodeData[nodeID]; exists {
		node.EPS = metrics.EPS
		node.KafkaLoad = metrics.KafkaLoad
		node.CHLoad = metrics.CHLoad
		node.CPU = metrics.CPU
		node.Memory = metrics.Memory
		node.LastUpdate = time.Now()

		response := APIResponse{
			Success: true,
			Message: fmt.Sprintf("Node %s metrics updated successfully", nodeID),
			Data:    node,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// Broadcast update
		go appState.broadcastUpdate()
	} else {
		response := APIResponse{
			Success: false,
			Message: fmt.Sprintf("Node %s not found", nodeID),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
	}
}

// Health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	appState.mutex.RLock()
	defer appState.mutex.RUnlock()

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"version":   AppVersion,
			"timestamp": time.Now(),
			"uptime":    time.Since(appState.StartTime).String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Middleware for CORS
func corsMiddleware(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(next)
}

// Background goroutine to simulate real-time metrics updates
func simulateMetrics() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		appState.mutex.Lock()
		if appState.IsSimulationRunning {
			// Simulate metric variations for active nodes
			for _, node := range appState.NodeData {
				if node.Status == "active" {
					// Add small random variations to simulate real-time changes
					variation := (time.Now().UnixNano() % 200) - 100 // -100 to +100
					node.EPS = max(0, node.EPS+int(variation))

					kafkaVariation := int(variation / 2)
					node.KafkaLoad = max(0, node.KafkaLoad+kafkaVariation)

					chVariation := int(variation / 5)
					node.CHLoad = max(0, node.CHLoad+chVariation)

					// Update CPU and memory with smaller variations
					cpuVariation := (time.Now().UnixNano() % 10) - 5 // -5 to +5
					node.CPU = maxFloat(0, minFloat(100, node.CPU+float64(cpuVariation)))

					memoryVariation := (time.Now().UnixNano() % 10) - 5
					node.Memory = maxFloat(0, minFloat(100, node.Memory+float64(memoryVariation)))

					node.LastUpdate = time.Now()
				}
			}

			// Broadcast updates to WebSocket clients
			go appState.broadcastUpdate()
		}
		appState.mutex.Unlock()
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Serve static files
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path
	if r.URL.Path == "/" {
		http.ServeFile(w, r, StaticDir+"/index.html")
		return
	}

	// Serve other static files
	staticPath := StaticDir + r.URL.Path
	http.ServeFile(w, r, staticPath)
}

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize start time
	appState.StartTime = time.Now()

	log.Printf("Starting vuDataSim Cluster Manager v%s", AppVersion)
	log.Printf("Serving static files from: %s", StaticDir)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Static file serving
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(StaticDir+"/"))))
	router.HandleFunc("/", serveStatic)

	// WebSocket endpoint
	router.HandleFunc("/ws", handleWebSocket)

	// API endpoints
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/dashboard", getDashboardData).Methods("GET")
	api.HandleFunc("/simulation/start", startSimulation).Methods("POST")
	api.HandleFunc("/simulation/stop", stopSimulation).Methods("POST")
	api.HandleFunc("/config/sync", syncConfiguration).Methods("POST")
	api.HandleFunc("/logs", getLogs).Methods("GET")
	api.HandleFunc("/nodes/{nodeId}/metrics", updateNodeMetrics).Methods("PUT")
	api.HandleFunc("/health", healthCheck).Methods("GET")

	// Start background simulation
	go simulateMetrics()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")

		appState.mutex.Lock()
		appState.IsSimulationRunning = false
		appState.mutex.Unlock()

		os.Exit(0)
	}()

	// Start server
	log.Printf("Server starting on port %s", Port)
	log.Printf("Open http://localhost%s in your browser", Port)

	srv := &http.Server{
		Addr:         Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
