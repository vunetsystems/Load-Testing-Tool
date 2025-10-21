package clickhouse

import (
	"context"
	"fmt"
	"os"
	"time"

	"vuDataSim/src/logger"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.yaml.in/yaml/v3"
)

// ClickHouseConfig holds configuration for connection
type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AppConfig holds the entire application configuration
type AppConfig struct {
	ClickHouse     ClickHouseConfig `yaml:"clickhouse"`
	MonitoredPods  []string         `yaml:"monitored_pods"`
	MonitoredNodes []string         `yaml:"monitored_nodes"`
	MonitoringDB   ClickHouseConfig `yaml:"monitoring_db"`
}

// ClickHouseClient wraps the ClickHouse connection and config
type ClickHouseClient struct {
	Client clickhouse.Conn
	Config ClickHouseConfig
}

// NewClickHouseClient initializes and checks the ClickHouse connection
func NewClickHouseClient(config ClickHouseConfig) (*ClickHouseClient, error) {
	logger.LogWithNode("System", "ClickHouse", "Initializing ClickHouse client connection", "info")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to connect to ClickHouse: %v", err))
		return nil, err
	}

	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("ClickHouse ping failed: %v", err))
		return nil, err
	}

	client := &ClickHouseClient{
		Client: conn,
		Config: config,
	}
	logger.LogSuccess("System", "ClickHouse", "ClickHouse client initialized successfully")
	return client, nil
}

// HealthCheck tests the connection
func (ch *ClickHouseClient) HealthCheck() error {
	ctx := context.Background()
	return ch.Client.Ping(ctx)
}

// Close closes the ClickHouse connection
func (ch *ClickHouseClient) Close() error {
	return ch.Client.Close()
}

// Global instances (used in main app; consider dependency injection for tests)
var clickHouseClient *ClickHouseClient
var clickHouseConfig ClickHouseConfig
var monitoringDBClient *ClickHouseClient
var monitoringDBConfig ClickHouseConfig
var monitoredPods []string
var monitoredNodes []string

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config AppConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	clickHouseConfig = config.ClickHouse
	monitoringDBConfig = config.MonitoringDB
	monitoredPods = config.MonitoredPods
	monitoredNodes = config.MonitoredNodes

	logger.LogWithNode("System", "ClickHouse", "Configuration loaded successfully", "info")
	return nil
}

// Initializes and sets global client
func InitClickHouse(configPath string) error {
	// Load configuration first
	err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	client, err := NewClickHouseClient(clickHouseConfig)
	if err != nil {
		return err
	}
	clickHouseClient = client

	// Initialize monitoring DB client if configured
	if monitoringDBConfig.Host != "" {
		monitoringClient, err := NewClickHouseClient(monitoringDBConfig)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to initialize monitoring DB client: %v", err))
		} else {
			monitoringDBClient = monitoringClient
			logger.LogSuccess("System", "ClickHouse", "Monitoring DB client initialized successfully")
		}
	}

	logger.LogSuccess("System", "ClickHouse", "ClickHouse client initialized successfully")
	return nil
}

// Check health status and provide config info
func GetClickHouseHealth() (map[string]interface{}, error) {
	if clickHouseClient == nil {
		return map[string]interface{}{
			"status": "disconnected",
		}, fmt.Errorf("ClickHouse client not initialized")
	}
	err := clickHouseClient.HealthCheck()
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}, err
	}
	return map[string]interface{}{
		"status":       "connected",
		"host":         clickHouseConfig.Host,
		"port":         clickHouseConfig.Port,
		"database":     clickHouseConfig.Database,
		"last_checked": time.Now(),
	}, nil
}

// GetMonitoredPods returns the list of monitored pods
func GetMonitoredPods() []string {
	return monitoredPods
}

// GetMonitoredNodes returns the list of monitored nodes
func GetMonitoredNodes() []string {
	return monitoredNodes
}
