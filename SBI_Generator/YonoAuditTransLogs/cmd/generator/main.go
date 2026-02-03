package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"kafka-message-generator/internal/config"
	"kafka-message-generator/internal/generator"
	"kafka-message-generator/internal/metrics"
	"kafka-message-generator/internal/producer"
	"kafka-message-generator/internal/worker"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg)
	defer logger.Sync()

	logger.Info("Starting Kafka Message Generator",
		zap.String("mode", cfg.Execution.Mode),
		zap.Int("eps", cfg.Execution.EPS),
		zap.String("topic", cfg.Kafka.Topic),
	)

	// Initialize session manager
	rotationInterval, _ := cfg.Session.GetRotationInterval()
	sessionMgr := generator.NewSessionManager(cfg.Session.SessionIDPrefix, rotationInterval)

	// Initialize message generator
	msgGen := generator.NewMessageGenerator(cfg, sessionMgr)

	// Initialize metrics collector
	metricsCol := metrics.NewCollector()

	// Initialize Prometheus exporter if enabled
	var promExporter *metrics.PrometheusExporter
	if cfg.Monitoring.EnableMetrics {
		promExporter = metrics.NewPrometheusExporter(metricsCol, cfg.Monitoring.MetricsPort)
		if err := promExporter.Start(); err != nil {
			logger.Error("Failed to start Prometheus exporter", zap.Error(err))
		} else {
			logger.Info("Prometheus metrics available", zap.Int("port", cfg.Monitoring.MetricsPort))
		}
	}

	// Initialize producer pool
	logger.Info("Initializing Kafka producer pool", zap.Int("num_producers", cfg.Kafka.Producer.NumProducers))
	producerPool, err := producer.NewProducerPool(cfg, metricsCol)
	if err != nil {
		logger.Fatal("Failed to create producer pool", zap.Error(err))
	}
	defer producerPool.Close()

	// Initialize worker pool
	logger.Info("Initializing worker pool", zap.Int("workers", cfg.Concurrency.WorkerPoolSize))
	workerPool := worker.NewWorkerPool(cfg, msgGen, producerPool, metricsCol)
	metricsInterval, _ := cfg.Logging.GetMetricsInterval()
	stopMetrics := make(chan struct{})
	go logMetrics(logger, metricsCol, promExporter, metricsInterval, stopMetrics)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker pool
	logger.Info("Starting message generation")
	startTime := time.Now()
	workerPool.Start()

	// Wait for completion or signal
	done := make(chan struct{})
	go func() {
		workerPool.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Message generation completed")
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))
		workerPool.StopWithTimeout(30 * time.Second)
	}

	// Stop metrics logging
	close(stopMetrics)

	// Final statistics
	elapsed := time.Since(startTime)
	stats := metricsCol.GetStats()
	
	logger.Info("Final Statistics",
		zap.Int64("total_generated", stats.TotalGenerated()),
		zap.Int64("error_only", stats.GeneratedError),
		zap.Int64("trans_only", stats.GeneratedTrans),
		zap.Int64("both", stats.GeneratedBoth),
		zap.Int64("access_log", stats.GeneratedAccessLog),
		zap.Int64("eis", stats.GeneratedEIS),
		zap.Int64("sent", stats.Sent),
		zap.Int64("failed", stats.Failed),
		zap.Duration("elapsed", elapsed),
		zap.Float64("avg_eps", float64(stats.Sent)/elapsed.Seconds()),
	)

	// Stop Prometheus exporter
	if promExporter != nil {
		promExporter.Stop()
	}

	logger.Info("Shutdown complete")
}

// initLogger initializes the logger
func initLogger(cfg *config.Config) *zap.Logger {
	var zapConfig zap.Config

	if cfg.Logging.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	switch cfg.Logging.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logger, _ := zapConfig.Build()
	return logger
}

// logMetrics periodically logs metrics
func logMetrics(logger *zap.Logger, collector *metrics.Collector, prom *metrics.PrometheusExporter, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collector.UpdateEPS()
			if prom != nil {
				prom.UpdateMetrics()
			}
			stats := collector.GetStats()
			
			logger.Info("Metrics",
				zap.Int64("generated", stats.TotalGenerated()),
				zap.Int64("sent", stats.Sent),
				zap.Int64("failed", stats.Failed),
				zap.Float64("current_eps", stats.CurrentEPS),
				zap.Duration("uptime", stats.Uptime),
			)
		case <-stop:
			return
		}
	}
}
