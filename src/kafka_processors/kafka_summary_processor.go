// Package kafka_processors implements Kafka Input/Output performance metrics summarization
// This is an update for Kafka summarization functionality - new backend script
// that generates summarized Kafka Input/Output performance metrics for completed performance tests.
// The summarization is stored back into the same test_runs table, one row per test ID.
package kafka_processors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
	_ "github.com/mattn/go-sqlite3"
)

// KafkaStat represents a single Kafka metric data point from ClickHouse
// This is an update for Kafka summarization functionality
type KafkaStat struct {
	Timestamp   time.Time
	Topic       string
	MsgsPerSec  float64
}

// ProcessKafkaSummaries processes all completed test runs that haven't been summarized yet
// This is the main entry point for the new Kafka summarization functionality
// This is an update for Kafka summarization functionality
func ProcessKafkaSummaries(db *sql.DB, chClient *clickhouse.ClickHouseClient) error {
	logger.LogWithNode("System", "KafkaSummaryProcessor", "Starting Kafka summary processing for completed tests", "info")

	// 1. Find pending test runs (completed but not summarized)
	row := db.QueryRow(`
		SELECT test_id, start_time, end_time, o11y_sources, target_eps
		FROM test_runs
		WHERE status = 'completed' AND (kafka_summary_generated IS NULL OR kafka_summary_generated = 0)
		ORDER BY start_time ASC
		LIMIT 1;
	`)

	var testID, o11ySourcesStr string
	var startTime, endTime time.Time
	var targetEPS int
	err := row.Scan(&testID, &startTime, &endTime, &o11ySourcesStr, &targetEPS)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.LogWithNode("System", "KafkaSummaryProcessor", "No unsummarized completed tests found", "info")
			return nil
		}
		return fmt.Errorf("failed to query pending test: %w", err)
	}

	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Processing test %s from %s to %s", testID, startTime, endTime), "info")

	// 2. Load the YAML topic definitions
	yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
	if err != nil {
		return fmt.Errorf("failed to read topics_tables.yaml: %w", err)
	}
	var cfg TopicsConfig
	err = yaml.Unmarshal(yamlData, &cfg)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	var o11ySources []string
	json.Unmarshal([]byte(o11ySourcesStr), &o11ySources)

	// 3. Build summary for each o11y source
	summary := make(map[string]interface{})

	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if s.Name == src {
				inputTopic := s.InputTopic[0]
				var outputTopics []string
				for _, ot := range s.OutputTopic {
					outputTopics = append(outputTopics, ot)
				}

				// Fetch input metrics
				inputStats, err := fetchKafkaMetrics(chClient, []string{inputTopic}, startTime, endTime)
				if err != nil {
					logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to fetch input metrics for %s: %v", src, err))
					continue
				}

				// Fetch output metrics
				outputStats, err := fetchKafkaMetrics(chClient, outputTopics, startTime, endTime)
				if err != nil {
					logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to fetch output metrics for %s: %v", src, err))
					continue
				}

				// Compute statistics
				inputSummary := computeKafkaStats(inputStats, inputTopic)
				outputSummary := computeKafkaStatsMulti(outputStats, outputTopics)

				// Calculate output efficiency
				if inputSummary["total_messages"].(float64) > 0 {
					outputSummary["output_efficiency_pct"] = (outputSummary["total_messages"].(float64) / inputSummary["total_messages"].(float64)) * 100
				} else {
					outputSummary["output_efficiency_pct"] = 0.0
				}

				summary[src] = map[string]interface{}{
					"input":  inputSummary,
					"output": outputSummary,
				}
			}
		}
	}

	// 4. Store the summary back to the database
	o11ySummaryJSON, _ := json.Marshal(summary) // Store the same summary in o11y_sources_summary

	// Calculate totals and averages across all sources
	var totalInput, totalOutput int64
	var avgInputRate, avgOutputRate, peakInputRate, peakOutputRate, minInputRate, minOutputRate float64
	minInputRate = math.MaxFloat64
	minOutputRate = math.MaxFloat64
	var anomalyScore float64
	var anomalyDetected bool

	for _, srcSummary := range summary {
		if srcData, ok := srcSummary.(map[string]interface{}); ok {
			if inputData, ok := srcData["input"].(map[string]interface{}); ok {
				totalInput += int64(inputData["total_messages"].(float64))
				avgInputRate += inputData["avg_msgs_per_sec"].(float64)
				if inputData["max_msgs_per_sec"].(float64) > peakInputRate {
					peakInputRate = inputData["max_msgs_per_sec"].(float64)
				}
				if inputData["min_msgs_per_sec"].(float64) < minInputRate {
					minInputRate = inputData["min_msgs_per_sec"].(float64)
				}
				anomalyScore += inputData["anomaly_spikes"].(float64)
			}
			if outputData, ok := srcData["output"].(map[string]interface{}); ok {
				totalOutput += int64(outputData["total_messages"].(float64))
				avgOutputRate += outputData["avg_msgs_per_sec"].(float64)
				if outputData["max_msgs_per_sec"].(float64) > peakOutputRate {
					peakOutputRate = outputData["max_msgs_per_sec"].(float64)
				}
				if outputData["min_msgs_per_sec"].(float64) < minOutputRate {
					minOutputRate = outputData["min_msgs_per_sec"].(float64)
				}
				anomalyScore += outputData["anomaly_spikes"].(float64)
			}
		}
	}

	// Calculate data loss percentage
	var dataLossPct float64
	if totalInput > 0 {
		dataLossPct = ((float64(totalInput - totalOutput)) / float64(totalInput)) * 100
	}

	// Determine if anomaly detected (simple threshold)
	anomalyDetected = anomalyScore > 5.0 // More than 5 total spikes across all sources

	_, err = db.Exec(`
		UPDATE test_runs
		SET total_input_msgs = ?, total_output_msgs = ?,
		    avg_input_msgs_per_sec = ?, avg_output_msgs_per_sec = ?,
		    peak_input_msgs_per_sec = ?, peak_output_msgs_per_sec = ?,
		    min_input_msgs_per_sec = ?, min_output_msgs_per_sec = ?,
		    data_loss_pct = ?, anomaly_detected = ?, anomaly_score_overall = ?,
		    anomaly_details = ?, o11y_sources_summary = ?, kafka_summary_generated = 1
		WHERE test_id = ?;
	`, totalInput, totalOutput, avgInputRate, avgOutputRate,
	   peakInputRate, peakOutputRate, minInputRate, minOutputRate,
	   dataLossPct, anomalyDetected, anomalyScore,
	   "Anomaly detection based on message rate spikes", string(o11ySummaryJSON), testID)
	if err != nil {
		return fmt.Errorf("failed to update test run with summary: %w", err)
	}

	logger.LogSuccess("System", "KafkaSummaryProcessor", fmt.Sprintf("Successfully processed Kafka summary for test %s", testID))
	return nil
}

// fetchKafkaMetrics fetches Kafka metrics from ClickHouse for given topics and time range
// This is an update for Kafka summarization functionality
func fetchKafkaMetrics(chClient *clickhouse.ClickHouseClient, topics []string, start, end time.Time) ([]KafkaStat, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	query := `
		SELECT timestamp, topic, messages_per_sec
		FROM kafka_Broker_Topic_Metrics
		WHERE topic IN (?) AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	ctx := context.Background()
	rows, err := chClient.Client.Query(ctx, query, topics, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query Kafka metrics: %w", err)
	}
	defer rows.Close()

	var stats []KafkaStat
	for rows.Next() {
		var s KafkaStat
		err := rows.Scan(&s.Timestamp, &s.Topic, &s.MsgsPerSec)
		if err != nil {
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to scan metric row: %v", err))
			continue
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading metric rows: %w", err)
	}

	return stats, nil
}

// computeKafkaStats computes statistics for a single topic
// This is an update for Kafka summarization functionality
func computeKafkaStats(stats []KafkaStat, topicName string) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"topic":               topicName,
			"avg_msgs_per_sec":    0.0,
			"max_msgs_per_sec":    0.0,
			"min_msgs_per_sec":    0.0,
			"stdev_msgs_per_sec":  0.0,
			"anomaly_spikes":      0,
			"total_messages":      0.0,
		}
	}

	var sum, min, max float64
	min = math.MaxFloat64
	values := make([]float64, len(stats))

	for i, stat := range stats {
		sum += stat.MsgsPerSec
		if stat.MsgsPerSec < min {
			min = stat.MsgsPerSec
		}
		if stat.MsgsPerSec > max {
			max = stat.MsgsPerSec
		}
		values[i] = stat.MsgsPerSec
	}

	mean := sum / float64(len(stats))

	// Standard deviation
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	stdev := math.Sqrt(variance / float64(len(values)))

	// Count anomalies (beyond ±2σ)
	anomalies := 0
	for _, v := range values {
		if math.Abs(v-mean) > 2*stdev {
			anomalies++
		}
	}

	// Total messages (assuming 1-second intervals)
	totalMessages := sum

	return map[string]interface{}{
		"topic":               topicName,
		"avg_msgs_per_sec":    mean,
		"max_msgs_per_sec":    max,
		"min_msgs_per_sec":    min,
		"stdev_msgs_per_sec":  stdev,
		"anomaly_spikes":      anomalies,
		"total_messages":      totalMessages,
	}
}

// computeKafkaStatsMulti computes statistics for multiple output topics combined
// This is an update for Kafka summarization functionality
func computeKafkaStatsMulti(stats []KafkaStat, topicNames []string) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"total_topics":        len(topicNames),
			"avg_msgs_per_sec":    0.0,
			"max_msgs_per_sec":    0.0,
			"min_msgs_per_sec":    0.0,
			"stdev_msgs_per_sec":  0.0,
			"anomaly_spikes":      0,
			"total_messages":      0.0,
			"output_efficiency_pct": 0.0,
		}
	}

	var sum, min, max float64
	min = math.MaxFloat64
	values := make([]float64, len(stats))

	for i, stat := range stats {
		sum += stat.MsgsPerSec
		if stat.MsgsPerSec < min {
			min = stat.MsgsPerSec
		}
		if stat.MsgsPerSec > max {
			max = stat.MsgsPerSec
		}
		values[i] = stat.MsgsPerSec
	}

	mean := sum / float64(len(stats))

	// Standard deviation
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	stdev := math.Sqrt(variance / float64(len(values)))

	// Count anomalies (beyond ±2σ)
	anomalies := 0
	for _, v := range values {
		if math.Abs(v-mean) > 2*stdev {
			anomalies++
		}
	}

	// Total messages (assuming 1-second intervals)
	totalMessages := sum

	return map[string]interface{}{
		"total_topics":        len(topicNames),
		"avg_msgs_per_sec":    mean,
		"max_msgs_per_sec":    max,
		"min_msgs_per_sec":    min,
		"stdev_msgs_per_sec":  stdev,
		"anomaly_spikes":      anomalies,
		"total_messages":      totalMessages,
		"output_efficiency_pct": 0.0, // Will be calculated by caller
	}
}

// RunKafkaSummaryProcessor runs the Kafka summary processor as a standalone script
// This is an update for Kafka summarization functionality
func RunKafkaSummaryProcessor() {
	// Initialize database connection
	dbPath := "./data/vudatasim.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize ClickHouse client
	err = clickhouse.InitClickHouse("src/configs/config.yaml")
	if err != nil {
		log.Fatal("Failed to initialize ClickHouse:", err)
	}

	chClient := clickhouse.GetClickHouseClient()
	if chClient == nil {
		log.Fatal("ClickHouse client not available")
	}

	// Process summaries
	for {
		err := ProcessKafkaSummaries(db, chClient)
		if err != nil {
			logger.LogError("System", "KafkaSummaryProcessor", fmt.Sprintf("Error processing summaries: %v", err))
		}

		// Wait before checking again
		time.Sleep(30 * time.Second)
	}
}