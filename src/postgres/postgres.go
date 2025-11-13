// Package postgres provides PostgreSQL database connection and operations
package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
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
	return Config{
		Host:     "10.96.1.65", // Using IP address for reliable connection
		Port:     5432,
		User:     "Load_Testing_Tool_read_user",
		Password: "StrongPassword123",
		DBName:   "multicore",
	}
}