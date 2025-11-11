// Package database provides database migration operations for test run tracking
package database

import (
	"fmt"
)

// RunMigrations creates the necessary database tables
func RunMigrations() error {
	// Create test_runs table with updated schema
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS test_runs (
		test_id TEXT PRIMARY KEY,
		test_name TEXT,
		target_eps INTEGER NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME NULL,
		o11y_sources TEXT NOT NULL, -- JSON array of source names
		timeout_seconds INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running', -- 'running', 'completed', 'stopped', 'failed'
		total_input_msgs BIGINT DEFAULT 0,
		total_output_msgs BIGINT DEFAULT 0,
		avg_input_msgs_per_sec REAL DEFAULT 0.0,
		avg_output_msgs_per_sec REAL DEFAULT 0.0,
		peak_input_msgs_per_sec REAL DEFAULT 0.0,
		peak_output_msgs_per_sec REAL DEFAULT 0.0,
		min_input_msgs_per_sec REAL DEFAULT 0.0,
		min_output_msgs_per_sec REAL DEFAULT 0.0,
		data_loss_pct REAL DEFAULT 0.0,
		lag_ms_avg REAL DEFAULT 0.0,
		lag_ms_max REAL DEFAULT 0.0,
		anomaly_detected BOOLEAN DEFAULT FALSE,
		anomaly_score_overall REAL DEFAULT 0.0,
		anomaly_details TEXT,
		o11y_sources_summary TEXT,
		kafka_summary_generated BOOLEAN DEFAULT FALSE
	);`

	_, err := DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create test_runs table: %w", err)
	}

	// Run schema updates for existing tables
	err = runSchemaUpdates()
	if err != nil {
		return fmt.Errorf("failed to run schema updates: %w", err)
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

	// Create k6_runs table
	createK6RunsTableSQL := `
	CREATE TABLE IF NOT EXISTS k6_runs (
		test_id TEXT PRIMARY KEY,
		start_time DATETIME NOT NULL,
		end_time DATETIME NULL,
		time_range TEXT NOT NULL,
		vus INTEGER NOT NULL,
		iterations INTEGER NOT NULL,
		interval INTEGER NOT NULL,
		o11y_sources TEXT NOT NULL, -- JSON array of source names
		status TEXT NOT NULL DEFAULT 'running' -- 'running', 'completed', 'stopped', 'failed'
	);`

	_, err = DB.Exec(createK6RunsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create k6_runs table: %w", err)
	}

	// Create index on k6_runs status for faster queries
	k6IndexSQL := `CREATE INDEX IF NOT EXISTS idx_k6_runs_status ON k6_runs(status);`
	_, err = DB.Exec(k6IndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create k6_runs status index: %w", err)
	}

	// Create index on k6_runs start_time for ordering
	k6TimeIndexSQL := `CREATE INDEX IF NOT EXISTS idx_k6_runs_start_time ON k6_runs(start_time DESC);`
	_, err = DB.Exec(k6TimeIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create k6_runs start_time index: %w", err)
	}

	return nil
}

// runSchemaUpdates applies schema changes to existing tables
func runSchemaUpdates() error {
	// Rename stop_time to end_time if it exists
	_, err := DB.Exec(`ALTER TABLE test_runs RENAME COLUMN stop_time TO end_time;`)
	if err != nil {
		// Column might not exist or already renamed, continue
	}

	// Define columns to add with their definitions
	columnsToAdd := map[string]string{
		"test_name":                  "TEXT",
		"total_input_msgs":           "BIGINT DEFAULT 0",
		"total_output_msgs":          "BIGINT DEFAULT 0",
		"avg_input_msgs_per_sec":     "REAL DEFAULT 0.0",
		"avg_output_msgs_per_sec":    "REAL DEFAULT 0.0",
		"peak_input_msgs_per_sec":    "REAL DEFAULT 0.0",
		"peak_output_msgs_per_sec":   "REAL DEFAULT 0.0",
		"min_input_msgs_per_sec":     "REAL DEFAULT 0.0",
		"min_output_msgs_per_sec":    "REAL DEFAULT 0.0",
		"data_loss_pct":              "REAL DEFAULT 0.0",
		"lag_ms_avg":                 "REAL DEFAULT 0.0",
		"lag_ms_max":                 "REAL DEFAULT 0.0",
		"anomaly_detected":           "BOOLEAN DEFAULT FALSE",
		"anomaly_score_overall":      "REAL DEFAULT 0.0",
		"anomaly_details":            "TEXT",
		"o11y_sources_summary":       "TEXT",
		"kafka_summary_generated":    "BOOLEAN DEFAULT FALSE",
	}

	for name, def := range columnsToAdd {
		if !columnExists("test_runs", name) {
			alterSQL := fmt.Sprintf("ALTER TABLE test_runs ADD COLUMN %s %s;", name, def)
			_, err := DB.Exec(alterSQL)
			if err != nil {
				return fmt.Errorf("failed to add column %s: %w", name, err)
			}
		}
	}

	return nil
}

// columnExists checks if a column exists in the specified table
func columnExists(table, column string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := DB.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt_value interface{}
		var pk int
		err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk)
		if err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}