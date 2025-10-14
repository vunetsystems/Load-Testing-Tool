package clickhouse

import (
	"context"
	"fmt"
	"time"

	"vuDataSim/src/logger"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseConfig holds configuration for connection
type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
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
var clickHouseConfig = ClickHouseConfig{
	Host:     "10.32.3.50",
	Port:     9000,
	Database: "vusmart",
	Username: "monitoring_read",
	Password: "StrongP@assword123",
}

// Initializes and sets global client
func InitClickHouse() error {
	client, err := NewClickHouseClient(clickHouseConfig)
	if err != nil {
		return err
	}
	clickHouseClient = client
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
