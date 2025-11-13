// Package kafka_processors implements pipeline information processing for completed test runs
// This module fetches pipeline details from PostgreSQL for o11y sources used in test runs
package kafka_processors

import (
	"database/sql"
	"encoding/json"
	"fmt"
	// "os"
	"strings"
	"time"

	"vuDataSim/src/logger"
	"vuDataSim/src/postgres"

	// "gopkg.in/yaml.v3"
)

// PipelineData represents the pipeline information for a single o11y source
type PipelineData struct {
	Name     string `json:"name"`
	Threads  int    `json:"threads"`
	Instances int   `json:"instances"`
}


// GetPipelineMapping creates a mapping from o11y source names to pipeline names
func GetPipelineMapping(config *TopicsConfig) map[string]string {
	mapping := make(map[string]string)

	for _, source := range config.Sources {
		// Normalize source name for matching
		normalizedName := strings.ToLower(strings.ReplaceAll(source.Name, " ", ""))
		if len(source.Pipeline) > 0 {
			mapping[normalizedName] = source.Pipeline[0]
		}
	}

	return mapping
}

// FetchPipelineData fetches pipeline information from PostgreSQL for the given o11y sources and timestamp
func FetchPipelineData(o11ySources []string, endTime time.Time) (map[string]PipelineData, error) {
	logger.LogWithNode("System", "PipelineProcessor", fmt.Sprintf("Fetching pipeline data for %d o11y sources", len(o11ySources)), "info")

	// Load topics configuration
	config, err := LoadTopicsConfig("src/configs/topics_tables.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load topics config: %w", err)
	}

	// Create pipeline mapping
	pipelineMapping := GetPipelineMapping(config)

	// Initialize PostgreSQL client
	pgConfig := postgres.GetDefaultConfig()
	pgClient, err := postgres.NewClient(pgConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}
	defer pgClient.Close()

	pipelineInfo := make(map[string]PipelineData)

	// Process each o11y source
	for _, o11ySource := range o11ySources {
		// Normalize o11y source name
		normalizedSource := strings.ToLower(strings.ReplaceAll(o11ySource, " ", ""))

		// Get pipeline name from mapping
		pipelineName, exists := pipelineMapping[normalizedSource]
		if !exists {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("No pipeline mapping found for o11y source: %s", o11ySource))
			continue
		}

		// Query PostgreSQL for pipeline data
		// Get the latest record for the pipeline (ignore timestamp constraints)
		query := `
			SELECT instances, threads_per_instance, name
			FROM vusoft_vusoftetlpipeline
			WHERE name = $1
			ORDER BY COALESCE(updated_at, '1900-01-01'::timestamp) DESC, id DESC
			LIMIT 1
		`

		var instances, threads int
		var name string

		err := pgClient.DB.QueryRow(query, pipelineName).Scan(&instances, &threads, &name)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("No pipeline data found for %s at or before %s", pipelineName, endTime))
			} else {
				logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Failed to query pipeline data for %s: %v", pipelineName, err))
			}
			continue
		}

		pipelineInfo[o11ySource] = PipelineData{
			Name:      name,
			Threads:   threads,
			Instances: instances,
		}

		logger.LogWithNode("System", "PipelineProcessor", fmt.Sprintf("Fetched pipeline data for %s: %s (threads: %d, instances: %d)", o11ySource, name, threads, instances), "debug")
	}

	logger.LogSuccess("System", "PipelineProcessor", fmt.Sprintf("Successfully fetched pipeline data for %d/%d o11y sources", len(pipelineInfo), len(o11ySources)))
	return pipelineInfo, nil
}

// ProcessPipelineInfo fetches and returns pipeline information as JSON string for storage
func ProcessPipelineInfo(o11ySources []string, endTime time.Time) (string, error) {
	pipelineInfo, err := FetchPipelineData(o11ySources, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to fetch pipeline data: %w", err)
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(pipelineInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pipeline info to JSON: %w", err)
	}

	return string(jsonData), nil
}

// BackfillPipelineInfo populates pipeline_info for existing test runs that don't have it
func BackfillPipelineInfo(db *sql.DB) error {
	logger.LogWithNode("System", "PipelineProcessor", "Starting pipeline info backfill", "info")

	// Query test runs that are completed but don't have pipeline_info
	rows, err := db.Query(`
		SELECT test_id, o11y_sources, end_time
		FROM test_runs
		WHERE status = 'completed'
		AND (pipeline_info IS NULL OR pipeline_info = '')
	`)
	if err != nil {
		return fmt.Errorf("failed to query test runs for backfill: %w", err)
	}
	defer rows.Close()

	var testRuns []struct {
		TestID      string
		O11ySources string
		EndTime     *time.Time
	}

	for rows.Next() {
		var tr struct {
			TestID      string
			O11ySources string
			EndTime     *time.Time
		}
		err := rows.Scan(&tr.TestID, &tr.O11ySources, &tr.EndTime)
		if err != nil {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Failed to scan test run: %v", err))
			continue
		}
		testRuns = append(testRuns, tr)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading test runs: %w", err)
	}

	logger.LogWithNode("System", "PipelineProcessor", fmt.Sprintf("Found %d test runs to backfill", len(testRuns)), "info")

	successCount := 0
	for _, tr := range testRuns {
		if tr.EndTime == nil {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Test run %s has no end time, skipping", tr.TestID))
			continue
		}

		var o11ySources []string
		if err := json.Unmarshal([]byte(tr.O11ySources), &o11ySources); err != nil {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Failed to parse o11y sources for test %s: %v", tr.TestID, err))
			continue
		}

		pipelineInfoJSON, err := ProcessPipelineInfo(o11ySources, *tr.EndTime)
		if err != nil {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Failed to process pipeline info for test %s: %v", tr.TestID, err))
			continue
		}

		// Update the test run
		_, err = db.Exec(`
			UPDATE test_runs
			SET pipeline_info = ?
			WHERE test_id = ?
		`, pipelineInfoJSON, tr.TestID)
		if err != nil {
			logger.LogWarning("System", "PipelineProcessor", fmt.Sprintf("Failed to update test run %s: %v", tr.TestID, err))
			continue
		}

		successCount++
		logger.LogWithNode("System", "PipelineProcessor", fmt.Sprintf("Backfilled pipeline info for test %s", tr.TestID), "debug")
	}

	logger.LogSuccess("System", "PipelineProcessor", fmt.Sprintf("Successfully backfilled pipeline info for %d/%d test runs", successCount, len(testRuns)))
	return nil
}