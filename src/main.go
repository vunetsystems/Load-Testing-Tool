package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"vuDataSim/src/bin_control"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/kafka_ch_reset"
	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"
	"vuDataSim/src/o11y_source_manager"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Application version and configuration
const (
	AppVersion = "1.0.0"
	StaticDir  = "./static"
	Port       = "164.52.213.158:8086"
)

// Global application state
type AppState struct {
	IsSimulationRunning bool                                 `json:"isSimulationRunning"`
	CurrentProfile      string                               `json:"currentProfile"`
	TargetEPS           int                                  `json:"targetEps"`
	TargetKafka         int                                  `json:"targetKafka"`
	TargetClickHouse    int                                  `json:"targetClickHouse"`
	StartTime           time.Time                            `json:"startTime"`
	NodeData            map[string]*node_control.NodeMetrics `json:"nodeData"`
	ClickHouseMetrics   *clickhouse.ClickHouseMetrics        `json:"clickHouseMetrics,omitempty"`
	mutex               sync.RWMutex
	clients             map[*websocket.Conn]bool
	broadcast           chan []byte
}

// Global application state instance
var appState = &AppState{
	IsSimulationRunning: false,
	CurrentProfile:      "medium",
	TargetEPS:           10000,
	TargetKafka:         5000,
	TargetClickHouse:    2000,
	NodeData:            make(map[string]*node_control.NodeMetrics),
	clients:             make(map[*websocket.Conn]bool),
	broadcast:           make(chan []byte, 256),
}

// Global instances
var nodeManager = node_control.NewNodeManager()
var binaryControl *bin_control.BinaryControl
var o11yManager = o11y_source_manager.NewO11ySourceManager()
var kafkaHandler = kafka_ch_reset.NewKafkaHandler()

// Initialize application
func init() {
	// Initialize node data using the node_control package
	node_control.InitNodeData(nodeManager, appState)

	// Initialize binary control with loaded config
	binaryControl = bin_control.NewBinaryControl()
	err := binaryControl.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config for binary control: %v", err)
	}
}

func main() {
	// Initialize logger
	logFilePath := "logs/vuDataSim.log"
	if err := logger.InitLogger(logFilePath); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize start time
	appState.StartTime = time.Now()

	// Initialize node manager
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load nodes config")
		logger.Warn().Msg("Node management features may not be available")
	}

	// Initialize o11y source manager
	err = o11yManager.LoadMaxEPSConfig()
	if err != nil {
		log.Printf("Warning: Failed to load max EPS config: %v", err)
		log.Println("O11y source management features may not be available")
	}

	// Main config is loaded dynamically when needed

	// Source configs are loaded dynamically when needed

	// Check for CLI node management commands

	logger.Info().Str("version", AppVersion).Msg("Starting vuDataSim Cluster Manager")
	logger.Info().Str("static_dir", StaticDir).Msg("Serving static files")

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Static file serving with proper MIME types
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set proper MIME types for static files
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Content-Type", "text/html")
		}

		http.ServeFile(w, r, StaticDir+"/"+r.URL.Path)
	})))
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
	api.HandleFunc("/dashboard", getDashboardData).Methods("GET")
	// Cluster metrics API endpoint
	api.HandleFunc("/cluster/metrics", handleAPIGetClusterMetrics).Methods("GET")
	// Metrics with time range endpoint
	api.HandleFunc("/metrics", getMetrics).Methods("GET")

	// Node management API endpoints
	api.HandleFunc("/nodes", handleAPINodes).Methods("GET")
	api.HandleFunc("/nodes/{name}", handleAPINodeActions).Methods("POST", "PUT", "DELETE")
	api.HandleFunc("/cluster-settings", handleAPIClusterSettings).Methods("GET", "PUT")

	// Binary control API endpoints
	api.HandleFunc("/binary/status", handleAPIGetAllBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/status/{node}", handleAPIGetBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/start/{node}", handleAPIStartBinary).Methods("POST")
	api.HandleFunc("/binary/stop/{node}", handleAPIStopBinary).Methods("POST")

	// O11y Source Manager API endpoints
	api.HandleFunc("/o11y/sources", handleAPIGetO11ySources).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}", handleAPIGetO11ySourceDetails).Methods("GET")
	api.HandleFunc("/o11y/eps/distribute", handleAPIDistributeEPS).Methods("POST")
	api.HandleFunc("/o11y/eps/current", handleAPIGetCurrentEPS).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}/enable", handleAPIEnableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/sources/{source}/disable", handleAPIDisableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/max-eps", handleAPIGetMaxEPSConfig).Methods("GET")
	api.HandleFunc("/o11y/confd/distribute", handleAPIDistributeConfD).Methods("POST")
	// SSH status API endpoint
	api.HandleFunc("/ssh/status", handleAPIGetSSHStatus).Methods("GET")
	// ClickHouse metrics API endpoints
	api.HandleFunc("/clickhouse/metrics", handleAPIGetClickHouseMetrics).Methods("GET")
	api.HandleFunc("/clickhouse/health", handleAPIClickHouseHealth).Methods("GET")

	// Kafka and ClickHouse Reset API endpoints
	api.HandleFunc("/kafka/topics", kafkaHandler.GetTopics).Methods("GET")
	api.HandleFunc("/kafka/recreate", kafkaHandler.RecreateTopics).Methods("POST")
	api.HandleFunc("/kafka/status", kafkaHandler.GetTopicStatus).Methods("GET")
	api.HandleFunc("/kafka/describe/{topic}", kafkaHandler.DescribeTopic).Methods("GET")
	api.HandleFunc("/kafka/delete/{topic}", kafkaHandler.DeleteTopic).Methods("DELETE")
	api.HandleFunc("/kafka/create", kafkaHandler.CreateTopic).Methods("POST")
	api.HandleFunc("/clickhouse/truncate", kafkaHandler.TruncateClickHouseTables).Methods("POST")

	// Proxy endpoint for node metrics API
	api.HandleFunc("/proxy/metrics", handleProxyMetrics).Methods("GET")

	// Process metrics endpoint - collects finalvudatasim metrics directly via SSH
	api.HandleFunc("/process/metrics", handleAPIGetProcessMetrics).Methods("GET")

	// Initialize ClickHouse client
	if err := clickhouse.InitClickHouse("src/configs/config.yaml"); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize ClickHouse client - metrics will not be available")
	} else {
		logger.Info().Msg("ClickHouse client initialized successfully")
	}

	// Start background real metrics collection

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
	logger.Info().Str("port", Port).Msg("Server starting")
	logger.Info().Str("url", "http://"+Port).Msg("Open in browser")

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

// handleProxyMetrics proxies requests to the node metrics API server
func handleProxyMetrics(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for this endpoint
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Make request to the metrics API server
	resp, err := http.Get("http://216.48.191.10:8086/api/system/metrics")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch metrics from metrics API server")
		http.Error(w, "Failed to fetch metrics", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read metrics response")
		http.Error(w, "Failed to read metrics response", http.StatusInternalServerError)
		return
	}

	// Forward the response to the client
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
