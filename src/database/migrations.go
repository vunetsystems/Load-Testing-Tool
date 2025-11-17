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
		avg_input_msgs_per_sec REAL DEFAULT 0.0,
		avg_output_msgs_per_sec REAL DEFAULT 0.0,
		max_input_msgs_per_sec REAL DEFAULT 0.0,
		max_output_msgs_per_sec REAL DEFAULT 0.0,
		min_input_msgs_per_sec REAL DEFAULT 0.0,
		min_output_msgs_per_sec REAL DEFAULT 0.0,
		data_loss_pct REAL DEFAULT 0.0,
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
		test_name TEXT,
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

	// Add test_name column to k6_runs table if it doesn't exist
	if !columnExists("k6_runs", "test_name") {
		alterSQL := "ALTER TABLE k6_runs ADD COLUMN test_name TEXT;"
		_, err = DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add test_name column to k6_runs: %w", err)
		}
	}

	// Add duration column to k6_runs table if it doesn't exist
	if !columnExists("k6_runs", "duration") {
		alterSQL := "ALTER TABLE k6_runs ADD COLUMN duration TEXT;"
		_, err = DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add duration column to k6_runs: %w", err)
		}
	}

	// Add summarised column to k6_runs table if it doesn't exist
	if !columnExists("k6_runs", "summarised") {
		alterSQL := "ALTER TABLE k6_runs ADD COLUMN summarised BOOLEAN DEFAULT FALSE;"
		_, err = DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add summarised column to k6_runs: %w", err)
		}
	}

	// Add k6_summary column to k6_runs table if it doesn't exist
	if !columnExists("k6_runs", "k6_summary") {
		alterSQL := "ALTER TABLE k6_runs ADD COLUMN k6_summary TEXT;"
		_, err = DB.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add k6_summary column to k6_runs: %w", err)
		}
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
		"test_name":                                  "TEXT",
		"avg_input_msgs_per_sec":                     "REAL DEFAULT 0.0",
		"avg_output_msgs_per_sec":                    "REAL DEFAULT 0.0",
		"max_input_msgs_per_sec":                     "REAL DEFAULT 0.0",
		"max_output_msgs_per_sec":                    "REAL DEFAULT 0.0",
		"min_input_msgs_per_sec":                     "REAL DEFAULT 0.0",
		"min_output_msgs_per_sec":                    "REAL DEFAULT 0.0",
		"data_loss_pct":                              "REAL DEFAULT 0.0",
		"kafka_summary_generated":                    "BOOLEAN DEFAULT FALSE",
		"pod_resource_check":                         "BOOLEAN DEFAULT FALSE",
		"pod_metrics":                                "TEXT",
		"process_rate_summary":                       "TEXT", // JSON summary of process rates per o11y source
		"ingestion_summary":                          "TEXT", // JSON summary of hyperscale ingestion table-wise EPS data
		"pipeline_info":                              "TEXT", // JSON mapping of o11y sources to pipeline details
		"kafka_1_node_mem_min":                       "REAL DEFAULT 0.0",
		"kafka_1_node_mem_avg":                       "REAL DEFAULT 0.0",
		"kafka_1_node_mem_max":                       "REAL DEFAULT 0.0",
		"kafka_2_node_mem_min":                       "REAL DEFAULT 0.0",
		"kafka_2_node_mem_avg":                       "REAL DEFAULT 0.0",
		"kafka_2_node_mem_max":                       "REAL DEFAULT 0.0",
		"kafka_3_node_mem_min":                       "REAL DEFAULT 0.0",
		"kafka_3_node_mem_avg":                       "REAL DEFAULT 0.0",
		"kafka_3_node_mem_max":                       "REAL DEFAULT 0.0",
		"ch1_node_mem_min":                           "REAL DEFAULT 0.0",
		"ch1_node_mem_avg":                           "REAL DEFAULT 0.0",
		"ch1_node_mem_max":                           "REAL DEFAULT 0.0",
		"ch2_node_mem_min":                           "REAL DEFAULT 0.0",
		"ch2_node_mem_avg":                           "REAL DEFAULT 0.0",
		"ch2_node_mem_max":                           "REAL DEFAULT 0.0",
		"kafka_1_node_cpu_min":                       "REAL DEFAULT 0.0",
		"kafka_1_node_cpu_avg":                       "REAL DEFAULT 0.0",
		"kafka_1_node_cpu_max":                       "REAL DEFAULT 0.0",
		"kafka_2_node_cpu_min":                       "REAL DEFAULT 0.0",
		"kafka_2_node_cpu_avg":                       "REAL DEFAULT 0.0",
		"kafka_2_node_cpu_max":                       "REAL DEFAULT 0.0",
		"kafka_3_node_cpu_min":                       "REAL DEFAULT 0.0",
		"kafka_3_node_cpu_avg":                       "REAL DEFAULT 0.0",
		"kafka_3_node_cpu_max":                       "REAL DEFAULT 0.0",
		"ch1_node_cpu_min":                           "REAL DEFAULT 0.0",
		"ch1_node_cpu_avg":                           "REAL DEFAULT 0.0",
		"ch1_node_cpu_max":                           "REAL DEFAULT 0.0",
		"ch2_node_cpu_min":                           "REAL DEFAULT 0.0",
		"ch2_node_cpu_avg":                           "REAL DEFAULT 0.0",
		"ch2_node_cpu_max":                           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_cpu_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_cpu_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_cpu_max":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_mem_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_mem_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_mem_max":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_cpu_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_cpu_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_cpu_max":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_mem_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_mem_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_mem_max":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_cpu_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_cpu_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_cpu_max":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_mem_min":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_mem_avg":           "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_mem_max":           "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_cpu_min":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_cpu_avg":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_cpu_max":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_mem_min":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_mem_avg":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_mem_max":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_cpu_min":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_cpu_avg":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_cpu_max":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_mem_min":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_mem_avg":       "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_mem_max":       "REAL DEFAULT 0.0",
		"pipeline_pod_cpu_min":                       "REAL DEFAULT 0.0",
		"pipeline_pod_cpu_avg":                       "REAL DEFAULT 0.0",
		"pipeline_pod_cpu_max":                       "REAL DEFAULT 0.0",
		"pipeline_pod_mem_min":                       "REAL DEFAULT 0.0",
		"pipeline_pod_mem_avg":                       "REAL DEFAULT 0.0",
		"pipeline_pod_mem_max":                       "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_cpu_allocated":     "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_0_mem_allocated":     "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_cpu_allocated":     "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_1_mem_allocated":     "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_cpu_allocated":     "REAL DEFAULT 0.0",
		"kafka_cluster_cp_kafka_2_mem_allocated":     "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_cpu_allocated": "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_0_0_mem_allocated": "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_cpu_allocated": "REAL DEFAULT 0.0",
		"chi_clickhouse_vusmart_0_1_0_mem_allocated": "REAL DEFAULT 0.0",
		"pipeline_pod_cpu_allocated":                 "REAL DEFAULT 0.0",
		"pipeline_pod_mem_allocated":                 "REAL DEFAULT 0.0",
		"traefik_cpu_allocated":                      "REAL DEFAULT 0.0",
		"traefik_mem_allocated":                      "REAL DEFAULT 0.0",
		"min_lag":                                    "REAL DEFAULT 0.0",
		"avg_lag":                                    "REAL DEFAULT 0.0",
		"max_lag":                                    "REAL DEFAULT 0.0",
		"kafka_specs":                                "TEXT", // JSON specs of input/output topic partitions and replication factors
		"kafka_1_node_cpu_total":                     "REAL DEFAULT 0.0",
		"kafka_1_node_mem_total":                     "REAL DEFAULT 0.0",
		"kafka_2_node_cpu_total":                     "REAL DEFAULT 0.0",
		"kafka_2_node_mem_total":                     "REAL DEFAULT 0.0",
		"kafka_3_node_cpu_total":                     "REAL DEFAULT 0.0",
		"kafka_3_node_mem_total":                     "REAL DEFAULT 0.0",
		"ch1_node_cpu_total":                         "REAL DEFAULT 0.0",
		"ch1_node_mem_total":                         "REAL DEFAULT 0.0",
		"ch2_node_cpu_total":                         "REAL DEFAULT 0.0",
		"ch2_node_mem_total":                         "REAL DEFAULT 0.0",
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

	// Rename peak columns to max if they exist and max doesn't exist
	if columnExists("test_runs", "peak_input_msgs_per_sec") && !columnExists("test_runs", "max_input_msgs_per_sec") {
		_, err := DB.Exec("ALTER TABLE test_runs RENAME COLUMN peak_input_msgs_per_sec TO max_input_msgs_per_sec;")
		if err != nil {
			return fmt.Errorf("failed to rename peak_input_msgs_per_sec: %w", err)
		}
	}
	if columnExists("test_runs", "peak_output_msgs_per_sec") && !columnExists("test_runs", "max_output_msgs_per_sec") {
		_, err := DB.Exec("ALTER TABLE test_runs RENAME COLUMN peak_output_msgs_per_sec TO max_output_msgs_per_sec;")
		if err != nil {
			return fmt.Errorf("failed to rename peak_output_msgs_per_sec: %w", err)
		}
	}

	// Drop removed columns if they exist
	columnsToDrop := []string{
		"total_input_msgs",
		"total_output_msgs",
		"lag_ms_avg",
		"lag_ms_max",
		"anomaly_detected",
		"anomaly_score_overall",
		"anomaly_details",
		"o11y_sources_summary",
		"peak_input_msgs_per_sec",
		"peak_output_msgs_per_sec",
	}
	for _, col := range columnsToDrop {
		if columnExists("test_runs", col) {
			dropSQL := fmt.Sprintf("ALTER TABLE test_runs DROP COLUMN %s;", col)
			_, err := DB.Exec(dropSQL)
			if err != nil {
				return fmt.Errorf("failed to drop column %s: %w", col, err)
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
