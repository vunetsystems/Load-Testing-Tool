package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"audit-service-logs-generator/internal/config"
	"audit-service-logs-generator/internal/generator"
	"audit-service-logs-generator/internal/metrics"
	"audit-service-logs-generator/internal/producer"
	"audit-service-logs-generator/internal/worker"
)

func main() {
	// Parse command line arguments
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Get Kafka broker address
	broker := cfg.Kafka.PodName + ":" + cfg.Kafka.Port
	
	// If we are running outside cluster, fetch the node name via kubectl
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		nodeName, err := getKafkaNodeName(cfg.Kafka.Namespace, cfg.Kafka.PodName)
		if err == nil {
			broker = nodeName + ":" + cfg.Kafka.Port
			log.Printf("K8s: Using broker %s", broker)
		} else {
			log.Printf("WARN: Failed to resolve Kafka node name via kubectl: %v. Using %s", err, broker)
		}
	}

	// Initialize components
	sessionMgr := generator.NewSessionManager()
	msgGen := generator.NewMessageGenerator(cfg, sessionMgr)
	
	kafkaProducer, err := producer.NewKafkaProducer(cfg, broker)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()
	
	// Start periodic metrics logging (every 10 seconds)
	metricsCollector.StartPeriodicLogging(10 * time.Second)

	workerPool := worker.NewWorkerPool(cfg, msgGen, kafkaProducer, metricsCollector)

	// Context for graceful shutdown
	duration, err := cfg.GetDuration()
	if err != nil {
		log.Fatalf("Invalid duration: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Handle signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Received signal, shutting down...")
		cancel()
	}()

	// Start generation
	log.Printf("Starting generator: EPS=%d, Duration=%v, Workers=%d, Topic=%s", 
		cfg.Execution.TargetEPS, duration, cfg.Execution.Workers, cfg.Kafka.Topic)
		
	startTime := time.Now()
	workerPool.Start(ctx)
	workerPool.Wait()
	
	log.Printf("Done. Execution time: %v", time.Since(startTime))
}

// Helper to get Kafka node name (preserved from original logic)
func getKafkaNodeName(namespace, podName string) (string, error) {
	cmdStr := fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.spec.nodeName}'", podName, namespace)
	cmd := exec.Command("bash", "-c", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	nodeName := string(out)
	if nodeName == "" {
		return "", fmt.Errorf("empty node name returned")
	}
	return nodeName, nil
}
