// Package postgres provides PostgreSQL database connection and operations
package postgres

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"go.yaml.in/yaml/v3"
)

// Config holds PostgreSQL connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// Client wraps the PostgreSQL database connection
type Client struct {
	DB *sql.DB
}

// PostgresConfig holds the postgres configuration from yaml
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

// AppConfig holds the entire application configuration
type AppConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
}

// NewClient creates a new PostgreSQL client with the given configuration
func NewClient(config Config) (*Client, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{DB: db}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}

// GetDefaultConfig returns the default PostgreSQL configuration for the pipeline database
func GetDefaultConfig() Config {
	data, err := os.ReadFile("src/configs/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("failed to read config file: %v", err))
	}

	var cfg AppConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal config: %v", err))
	}

	return Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBName:   cfg.Postgres.DBName,
	}
}
