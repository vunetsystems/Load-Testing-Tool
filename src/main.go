package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"vuDataSim/src/bin_control"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/handlers"
	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"

	"github.com/gorilla/mux"
)

var kafkaHandler = handlers.NewKafkaHandler()

func init() {
	// Initialize node data using the node_control package
	node_control.InitNodeData(handlers.NodeManager, handlers.AppState)

	// Initialize binary control with loaded config
	handlers.BinaryControl = bin_control.NewBinaryControl()
	err := handlers.BinaryControl.LoadNodesConfig()
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
	handlers.AppState.StartTime = time.Now()

	// Initialize node manager
	err := handlers.NodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load nodes config")
		logger.Warn().Msg("Node management features may not be available")
	}

	// Initialize o11y source manager
	err = handlers.O11yManager.LoadMaxEPSConfig()
	if err != nil {
		log.Printf("Warning: Failed to load max EPS config: %v", err)
		log.Println("O11y source management features may not be available")
	}

	// Main config is loaded dynamically when needed

	// Source configs are loaded dynamically when needed

	// Check for CLI node management commands

	logger.Info().Str("version", handlers.AppVersion).Msg("Starting vuDataSim Cluster Manager")
	logger.Info().Str("static_dir", handlers.StaticDir).Msg("Serving static files")

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

		http.ServeFile(w, r, handlers.StaticDir+"/"+r.URL.Path)
	})))
	router.HandleFunc("/", serveStatic)

	// WebSocket endpoint
	router.HandleFunc("/ws", handleWebSocket)

	// API endpoints
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/dashboard", handlers.GetDashboardData).Methods("GET")
	api.HandleFunc("/simulation/start", handlers.StartSimulation).Methods("POST")
	api.HandleFunc("/simulation/stop", handlers.StopSimulation).Methods("POST")
	api.HandleFunc("/config/sync", handlers.SyncConfiguration).Methods("POST")
	api.HandleFunc("/logs", handlers.GetLogs).Methods("GET")
	api.HandleFunc("/nodes/{nodeId}/metrics", handlers.UpdateNodeMetrics).Methods("PUT")
	api.HandleFunc("/health", handlers.HealthCheck).Methods("GET`")
	api.HandleFunc("/dashboard", handlers.GetDashboardData).Methods("GET")
	// Cluster metrics API endpoint
	api.HandleFunc("/cluster/metrics", handlers.HandleAPIGetClusterMetrics).Methods("GET")
	// Metrics with time range endpoint
	api.HandleFunc("/metrics", handlers.GetMetrics).Methods("GET")

	// Node management API endpoints
	api.HandleFunc("/nodes", handlers.HandleAPINodes).Methods("GET")
	api.HandleFunc("/nodes/{name}", handlers.HandleAPINodeActions).Methods("POST", "PUT", "DELETE")
	api.HandleFunc("/nodes/{name}/debug", handlers.HandleAPIDebugMetricsBinary).Methods("GET")
	api.HandleFunc("/cluster-settings", handlers.HandleAPIClusterSettings).Methods("GET", "PUT")

	// Binary control API endpoints
	api.HandleFunc("/binary/status", handlers.HandleAPIGetAllBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/status/{node}", handlers.HandleAPIGetBinaryStatus).Methods("GET")
	api.HandleFunc("/binary/start/{node}", handlers.HandleAPIStartBinary).Methods("POST")
	api.HandleFunc("/binary/stop/{node}", handlers.HandleAPIStopBinary).Methods("POST")

	// O11y Source Manager API endpoints
	api.HandleFunc("/o11y/sources", handlers.HandleAPIGetO11ySources).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}", handlers.HandleAPIGetO11ySourceDetails).Methods("GET")
	api.HandleFunc("/o11y/eps/distribute", handlers.HandleAPIDistributeEPS).Methods("POST")
	api.HandleFunc("/o11y/eps/current", handlers.HandleAPIGetCurrentEPS).Methods("GET")
	api.HandleFunc("/o11y/sources/{source}/enable", handlers.HandleAPIEnableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/sources/{source}/disable", handlers.HandleAPIDisableO11ySource).Methods("POST")
	api.HandleFunc("/o11y/max-eps", handlers.HandleAPIGetMaxEPSConfig).Methods("GET")
	api.HandleFunc("/o11y/confd/distribute", handlers.HandleAPIDistributeConfD).Methods("POST")
	// SSH status API endpoint
	api.HandleFunc("/ssh/status", handlers.HandleAPIGetSSHStatus).Methods("GET")
	// ClickHouse metrics API endpoints
	api.HandleFunc("/clickhouse/metrics", handlers.HandleAPIGetClickHouseMetrics).Methods("GET")
	api.HandleFunc("/clickhouse/health", handlers.HandleAPIClickHouseHealth).Methods("GET")
	api.HandleFunc("/clickhouse/kafka-topics", handlers.HandleAPIGetKafkaTopicMetrics).Methods("GET")
	api.HandleFunc("/clickhouse/pod-metrics", handlers.HandleAPIGetPodMetrics).Methods("GET")

	// Kubernetes API endpoints
	api.HandleFunc("/kubernetes/pods", handlers.HandleAPIGetKubernetesPods).Methods("GET")

	// Kafka and ClickHouse Reset API endpoints
	api.HandleFunc("/kafka/topics", kafkaHandler.GetTopics).Methods("GET")
	api.HandleFunc("/kafka/recreate", kafkaHandler.RecreateTopicsForO11ySources).Methods("POST")
	api.HandleFunc("/kafka/status", kafkaHandler.GetTopicStatus).Methods("GET")
	api.HandleFunc("/kafka/describe/{topic}", kafkaHandler.DescribeTopic).Methods("GET")
	api.HandleFunc("/kafka/delete/{topic}", kafkaHandler.DeleteTopic).Methods("DELETE")
	api.HandleFunc("/kafka/create", kafkaHandler.CreateTopic).Methods("POST")
	api.HandleFunc("/clickhouse/truncate", kafkaHandler.TruncateClickHouseTables).Methods("POST")
	api.HandleFunc("/clickhouse/tables", kafkaHandler.GetClickHouseTableNames).Methods("GET")

	// K6 Load Testing API endpoints
	api.HandleFunc("/k6/config", handlers.HandleAPIGetK6Config).Methods("GET")
	api.HandleFunc("/k6/config", handlers.HandleAPIUpdateK6Config).Methods("PUT")
	api.HandleFunc("/k6/config/reset", handlers.HandleAPIResetK6Config).Methods("POST")
	api.HandleFunc("/k6/status", handlers.HandleAPIGetK6Status).Methods("GET")
	api.HandleFunc("/k6/start", handlers.HandleAPIStartK6Test).Methods("POST")
	api.HandleFunc("/k6/stop", handlers.HandleAPIStopK6Test).Methods("POST")
	api.HandleFunc("/k6/logs", handlers.HandleAPIGetK6Logs).Methods("GET")

	// Proxy endpoint for node metrics API
	api.HandleFunc("/proxy/metrics", handlers.HandleProxyMetrics).Methods("GET")

	// Process metrics endpoint - collects finalvudatasim metrics directly via SSH
	api.HandleFunc("/process/metrics", handlers.HandleAPIGetProcessMetrics).Methods("GET")

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

		handlers.AppState.IsSimulationRunning = false
		handlers.AppState.Mutex.Unlock()

		os.Exit(0)
	}()

	// Start server
	logger.Info().Str("port", handlers.Port).Msg("Server starting")
	logger.Info().Str("url", "http://"+handlers.Port).Msg("Open in browser")

	srv := &http.Server{
		Addr:         handlers.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
