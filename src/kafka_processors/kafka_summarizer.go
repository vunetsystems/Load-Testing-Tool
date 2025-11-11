// Package kafka_processors implements Kafka summarization for completed test runs
// This module processes test runs that have completed and generates Kafka message rate summaries
// based on the o11y sources configured for each test.
// This is an update for Kafka summarization functionality.
package kafka_processors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"vuDataSim/src/database"
	"vuDataSim/src/logger"

	"github.com/ClickHouse/clickhouse-go/v2"
	"gopkg.in/yaml.v3"
)

// KafkaSummary represents the computed summary of Kafka metrics for a test run
type KafkaSummary struct {
	TestID           string                    `json:"test_id"`
	StartTime        time.Time                 `json:"start_time"`
	EndTime          time.Time                 `json:"end_time"`
	DurationMinutes  float64                   `json:"duration_minutes"`
	TotalMessages    int64                     `json:"total_messages"`
	AvgMessageRate   float64                   `json:"avg_message_rate"`
	PeakMessageRate  float64                   `json:"peak_message_rate"`
	SourceSummaries  []KafkaSourceSummary      `json:"source_summaries"`
	GeneratedAt      time.Time                 `json:"generated_at"`
}

// KafkaSourceSummary represents summary for a specific o11y source
type KafkaSourceSummary struct {
	SourceName       string    `json:"source_name"`
	Topics           []string  `json:"topics"`
	TotalMessages    int64     `json:"total_messages"`
	AvgMessageRate   float64   `json:"avg_message_rate"`
	PeakMessageRate  float64   `json:"peak_message_rate"`
}

// TopicConfig represents the configuration for a single o11y source
type TopicConfig struct {
	Name         string   `yaml:"name"`
	InputTopic   []string `yaml:"inputTopic"`
	OutputTopic  []string `yaml:"outputTopic"`
	ClickHouseTables []string `yaml:"clickhouseTables"`
	Pipeline     []string `yaml:"pipeline"`
}

// TopicsConfig represents the complete topics configuration
type TopicsConfig struct {
	Sources []TopicConfig `yaml:"sources"`
}

// KafkaMetric represents a single Kafka metric data point from ClickHouse
type KafkaMetric struct {
	Timestamp     time.Time `json:"timestamp"`
	Topic         string    `json:"topic"`
	OneMinuteRate float64   `json:"one_minute_rate"`
}

// EnsureTestRunsTableColumns adds the missing columns to test_runs table if they don't exist
// This is an update for Kafka summarization functionality
func EnsureTestRunsTableColumns(db *sql.DB) error {
	logger.LogWithNode("System", "KafkaSummarizer", "Ensuring test_runs table has required columns for Kafka summarization", "info")

	// Check and add summary_generated column
	_, err := db.Exec(`
		ALTER TABLE test_runs
		ADD COLUMN IF NOT EXISTS summary_generated BOOLEAN DEFAULT FALSE
	`)
	if err != nil {
		return fmt.Errorf("failed to add summary_generated column: %w", err)
	}

	// Check and add kafka_summary column
	_, err = db.Exec(`
		ALTER TABLE test_runs
		ADD COLUMN IF NOT EXISTS kafka_summary TEXT
	`)
	if err != nil {
		return fmt.Errorf("failed to add kafka_summary column: %w", err)
	}

	logger.LogSuccess("System", "KafkaSummarizer", "Test runs table columns verified/added successfully")
	return nil
}

// LoadTopicsConfig loads the topics configuration from the YAML file
// This is used to map o11y sources to their corresponding Kafka topics
func LoadTopicsConfig(configPath string) (*TopicsConfig, error) {
	logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("Loading topics configuration from %s", configPath), "info")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics config file: %w", err)
	}

	var config TopicsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse topics config: %w", err)
	}

	logger.LogSuccess("System", "KafkaSummarizer", fmt.Sprintf("Loaded configuration for %d sources", len(config.Sources)))
	return &config, nil
}

// GetTopicsForSources returns all topics (input and output) for the given o11y sources
// This is an update for Kafka summarization to resolve topic mappings
func GetTopicsForSources(config *TopicsConfig, sources []string) []string {
	var topics []string
	sourceMap := make(map[string]bool)
	for _, source := range sources {
		sourceMap[source] = true
	}

	for _, source := range config.Sources {
		if sourceMap[source.Name] {
			topics = append(topics, source.InputTopic...)
			topics = append(topics, source.OutputTopic...)
		}
	}

	return topics
}

// QueryKafkaMetrics queries ClickHouse for Kafka metrics within the specified time range
// This is an update for Kafka summarization to fetch metrics data
func QueryKafkaMetrics(ctx context.Context, chClient clickhouse.Conn, topics []string, startTime, endTime time.Time) ([]KafkaMetric, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided for metrics query")
	}

	logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("Querying Kafka metrics for %d topics from %s to %s", len(topics), startTime, endTime), "info")

	query := `
		SELECT
			timestamp,
			topic,
			OneMinuteRate
		FROM kafka_Broker_Topic_Metrics
		WHERE
			name = 'MessagesInPerSec'
			AND topic IN ?
			AND timestamp >= ?
			AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := chClient.Query(ctx, query, topics, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query Kafka metrics: %w", err)
	}
	defer rows.Close()

	var metrics []KafkaMetric
	for rows.Next() {
		var metric KafkaMetric
		err := rows.Scan(&metric.Timestamp, &metric.Topic, &metric.OneMinuteRate)
		if err != nil {
			logger.LogWarning("System", "KafkaSummarizer", fmt.Sprintf("Failed to scan Kafka metric row: %v", err))
			continue
		}
		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading Kafka metrics rows: %w", err)
	}

	logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("Retrieved %d Kafka metric data points", len(metrics)), "info")
	return metrics, nil
}

// ComputeKafkaSummary computes the summary statistics from raw Kafka metrics
// This is an update for Kafka summarization to calculate aggregates
func ComputeKafkaSummary(testID string, startTime, endTime time.Time, metrics []KafkaMetric, config *TopicsConfig, sources []string) *KafkaSummary {
	duration := endTime.Sub(startTime)
	durationMinutes := duration.Minutes()

	summary := &KafkaSummary{
		TestID:          testID,
		StartTime:       startTime,
		EndTime:         endTime,
		DurationMinutes: durationMinutes,
		GeneratedAt:     time.Now(),
		SourceSummaries: []KafkaSourceSummary{},
	}

	// Group metrics by source
	sourceMetrics := make(map[string][]KafkaMetric)
	sourceTopics := make(map[string][]string)

	for _, sourceName := range sources {
		sourceTopics[sourceName] = []string{}
		sourceMetrics[sourceName] = []KafkaMetric{}
	}

	// Map topics to sources
	for _, source := range config.Sources {
		if contains(sources, source.Name) {
			allTopics := append(source.InputTopic, source.OutputTopic...)
			sourceTopics[source.Name] = allTopics
		}
	}

	// Group metrics by source
	for _, metric := range metrics {
		for sourceName, topics := range sourceTopics {
			if contains(topics, metric.Topic) {
				sourceMetrics[sourceName] = append(sourceMetrics[sourceName], metric)
				break
			}
		}
	}

	// Compute per-source summaries
	var totalMessages int64
	var allRates []float64
	var peakRate float64

	for sourceName, sourceMetrics := range sourceMetrics {
		if len(sourceMetrics) == 0 {
			continue
		}

		sourceSummary := KafkaSourceSummary{
			SourceName: sourceName,
			Topics:     sourceTopics[sourceName],
		}

		var sourceTotalMessages int64
		var sourceRates []float64
		var sourcePeakRate float64

		for _, metric := range sourceMetrics {
			// Convert rate per minute to total messages (approximate)
			messages := int64(metric.OneMinuteRate * durationMinutes)
			sourceTotalMessages += messages
			sourceRates = append(sourceRates, metric.OneMinuteRate)

			if metric.OneMinuteRate > sourcePeakRate {
				sourcePeakRate = metric.OneMinuteRate
			}
		}

		sourceSummary.TotalMessages = sourceTotalMessages
		sourceSummary.PeakMessageRate = sourcePeakRate

		if len(sourceRates) > 0 {
			var sum float64
			for _, rate := range sourceRates {
				sum += rate
			}
			sourceSummary.AvgMessageRate = sum / float64(len(sourceRates))
		}

		summary.SourceSummaries = append(summary.SourceSummaries, sourceSummary)
		totalMessages += sourceTotalMessages
		allRates = append(allRates, sourceRates...)
		if sourcePeakRate > peakRate {
			peakRate = sourcePeakRate
		}
	}

	summary.TotalMessages = totalMessages
	summary.PeakMessageRate = peakRate

	if len(allRates) > 0 {
		var sum float64
		for _, rate := range allRates {
			sum += rate
		}
		summary.AvgMessageRate = sum / float64(len(allRates))
	}

	return summary
}

// GenerateKafkaSummaryForTestRun generates Kafka summary for a single completed test run
// This is the main function for Kafka summarization that can be called for integration
// This is an update for Kafka summarization functionality
func GenerateKafkaSummaryForTestRun(db *sql.DB, chClient clickhouse.Conn, configPath string, testID string) error {
	logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("Starting Kafka summary generation for test run %s", testID), "info")

	// Load topics configuration
	config, err := LoadTopicsConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load topics config: %w", err)
	}

	// Get test run details
	testRun, err := database.GetTestRun(testID)
	if err != nil {
		return fmt.Errorf("failed to get test run %s: %w", testID, err)
	}

	if testRun.Status != "completed" {
		return fmt.Errorf("test run %s is not completed (status: %s)", testID, testRun.Status)
	}

	if testRun.EndTime == nil {
		return fmt.Errorf("test run %s has no end time", testID)
	}

	// Get topics for the test's o11y sources
	topics := GetTopicsForSources(config, testRun.O11ySources)
	if len(topics) == 0 {
		logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("No topics found for sources %v, skipping summary", testRun.O11ySources), "info")
		return nil
	}

	// Query Kafka metrics for the test duration
	ctx := context.Background()
	metrics, err := QueryKafkaMetrics(ctx, chClient, topics, testRun.StartTime, *testRun.EndTime)
	if err != nil {
		return fmt.Errorf("failed to query Kafka metrics: %w", err)
	}

	// Compute summary
	summary := ComputeKafkaSummary(testID, testRun.StartTime, *testRun.EndTime, metrics, config, testRun.O11ySources)

	// Serialize summary to JSON
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Update test run with summary
	_, err = db.Exec(`
		UPDATE test_runs
		SET kafka_summary = ?, summary_generated = TRUE
		WHERE test_id = ?
	`, string(summaryJSON), testID)
	if err != nil {
		return fmt.Errorf("failed to update test run with summary: %w", err)
	}

	logger.LogSuccess("System", "KafkaSummarizer", fmt.Sprintf("Generated Kafka summary for test run %s", testID))
	return nil
}

// GenerateKafkaSummaries processes all completed test runs that haven't been summarized yet
// This is the main entry point for batch Kafka summarization
// This is an update for Kafka summarization functionality
func GenerateKafkaSummaries(db *sql.DB, chClient clickhouse.Conn, configPath string) error {
	logger.LogWithNode("System", "KafkaSummarizer", "Starting batch Kafka summary generation", "info")

	// Ensure table columns exist
	err := EnsureTestRunsTableColumns(db)
	if err != nil {
		return fmt.Errorf("failed to ensure table columns: %w", err)
	}

	// Get all completed test runs without summaries
	rows, err := db.Query(`
		SELECT test_id
		FROM test_runs
		WHERE status = 'completed'
		AND (summary_generated IS NULL OR summary_generated = FALSE)
		ORDER BY start_time DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query completed test runs: %w", err)
	}
	defer rows.Close()

	var testIDs []string
	for rows.Next() {
		var testID string
		err := rows.Scan(&testID)
		if err != nil {
			logger.LogWarning("System", "KafkaSummarizer", fmt.Sprintf("Failed to scan test ID: %v", err))
			continue
		}
		testIDs = append(testIDs, testID)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading test IDs: %w", err)
	}

	logger.LogWithNode("System", "KafkaSummarizer", fmt.Sprintf("Found %d test runs to process", len(testIDs)), "info")

	// Process each test run
	successCount := 0
	for _, testID := range testIDs {
		err := GenerateKafkaSummaryForTestRun(db, chClient, configPath, testID)
		if err != nil {
			logger.LogError("System", "KafkaSummarizer", fmt.Sprintf("Failed to generate summary for test run %s: %v", testID, err))
			continue
		}
		successCount++
	}

	logger.LogSuccess("System", "KafkaSummarizer", fmt.Sprintf("Successfully generated summaries for %d/%d test runs", successCount, len(testIDs)))
	return nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}