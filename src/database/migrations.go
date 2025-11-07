// Package database provides database migration operations for test run tracking
package database

import (
	"fmt"
)

// RunMigrations creates the necessary database tables
func RunMigrations() error {
	// Create test_runs table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS test_runs (
		test_id TEXT PRIMARY KEY,
		target_eps INTEGER NOT NULL,
		start_time DATETIME NOT NULL,
		stop_time DATETIME NULL,
		o11y_sources TEXT NOT NULL, -- JSON array of source names
		timeout_seconds INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running' -- 'running', 'completed', 'stopped', 'failed'
	);`

	_, err := DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create test_runs table: %w", err)
	}

	// Create index on status for faster queries
	indexSQL := `CREATE INDEX IF NOT EXISTS idx_test_runs_status ON test_runs(status);`
	_, err = DB.Exec(indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create status index: %w", err)
	}

	// Create index on start_time for ordering
	timeIndexSQL := `CREATE INDEX IF NOT EXISTS idx_test_runs_start_time ON test_runs(start_time DESC);`
	_, err = DB.Exec(timeIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create start_time index: %w", err)
	}

	return nil
}