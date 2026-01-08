// Package kafka_processors implements consumer lag summarization for completed test runs
// This module processes test runs and generates consumer lag summaries
// based on Kafka consumer lag metrics for input topics of o11y sources.
package kafka_processors

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// ConsumerLagStat represents a single consumer lag data point from ClickHouse
type ConsumerLagStat struct {
	RecordsLag float64
}

// fetchConsumerLagMetrics fetches consumer lag metrics from ClickHouse for given topics and time range
// Queries monitoring.kafka_consumer_Fetch_Manager_LagMetrics and returns all lag values > 0
func fetchConsumerLagMetrics(chClient *clickhouse.ClickHouseClient, topics []string, start, end time.Time) ([]float64, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// Build the IN clause for topics
	topicList := make([]string, len(topics))
	for i, t := range topics {
		topicList[i] = fmt.Sprintf("'%s'", t)
	}
	topicStr := strings.Join(topicList, ",")

	query := fmt.Sprintf(`
		SELECT "records-lag"
		FROM monitoring.kafka_consumer_Fetch_Manager_LagMetrics
		WHERE 
		--"records-lag" > 0
		--AND 
		  topic IN (%s)
		  AND timestamp >= toDateTime('%s')
		  AND timestamp <= toDateTime('%s')
	`, topicStr, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var lagValues []float64

	for rows.Next() {
		var lag float64
		if err := rows.Scan(&lag); err != nil {
			logger.LogWarning("System", "ConsumerLagProcessor", fmt.Sprintf("Failed to scan lag row: %v", err))
			continue
		}
		lagValues = append(lagValues, lag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(lagValues) == 0 {
		logger.LogWithNode("System", "ConsumerLagProcessor", "No consumer lag data found for given topics/time range", "warn")
	} else {
		logger.LogSuccess("System", "ConsumerLagProcessor", fmt.Sprintf("Fetched %d consumer lag values", len(lagValues)))
	}

	return lagValues, nil
}

// computeLagStats computes min, avg, max from lag values
func computeLagStats(lagValues []float64) (minLag, avgLag, maxLag float64) {
	if len(lagValues) == 0 {
		return 0.0, 0.0, 0.0
	}

	minLag = math.MaxFloat64
	maxLag = -math.MaxFloat64
	sum := 0.0

	for _, lag := range lagValues {
		if lag < minLag {
			minLag = lag
		}
		if lag > maxLag {
			maxLag = lag
		}
		sum += lag
	}

	avgLag = sum / float64(len(lagValues))
	return minLag, avgLag, maxLag
}

// ProcessConsumerLagSummary fetches and computes consumer lag statistics for a test run
func ProcessConsumerLagSummary(chClient *clickhouse.ClickHouseClient, topics []string, startTime, endTime time.Time) (minLag, avgLag, maxLag float64, err error) {
	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Starting consumer lag summary for topics: %v", topics), "info")

	lagValues, err := fetchConsumerLagMetrics(chClient, topics, startTime, endTime)
	if err != nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("failed to fetch consumer lag metrics: %w", err)
	}

	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Fetched %d lag values: %v", len(lagValues), lagValues), "debug")

	minLag, avgLag, maxLag = computeLagStats(lagValues)

	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Computed stats: min=%.2f, avg=%.2f, max=%.2f", minLag, avgLag, maxLag), "info")
	logger.LogSuccess("System", "ConsumerLagProcessor", "Successfully processed consumer lag summary")
	return minLag, avgLag, maxLag, nil
}

// ProcessConsumerLagSummaryPerSource computes lag statistics for each source individually
func ProcessConsumerLagSummaryPerSource(chClient *clickhouse.ClickHouseClient, o11ySources []string, cfg TopicsConfig, startTime, endTime time.Time) (map[string]map[string]float64, error) {
	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Processing lag for sources: %v", o11ySources), "info")
	lagPerSource := make(map[string]map[string]float64)

	for _, src := range o11ySources {
		logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Processing source: %s", src), "info")

		// Find input topics for this source
		var inputTopics []string
		for _, s := range cfg.Sources {
			logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Checking config source: %s", s.Name), "debug")
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				for _, inputTopic := range s.InputTopic {
					inputTopics = append(inputTopics, inputTopic.Name)
				}
				break
			}
		}

		logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Found topics for %s: %v", src, inputTopics), "info")

		// Use existing ProcessConsumerLagSummary function
		minLag, avgLag, maxLag, err := ProcessConsumerLagSummary(chClient, inputTopics, startTime, endTime)
		if err != nil {
			logger.LogWarning("System", "ConsumerLagProcessor", fmt.Sprintf("Failed to process lag for source %s: %v", src, err))
			continue
		}

		logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Lag stats for %s: min=%.2f, avg=%.2f, max=%.2f", src, minLag, avgLag, maxLag), "info")

		lagPerSource[src] = map[string]float64{
			"min": minLag,
			"avg": avgLag,
			"max": maxLag,
		}
	}

	logger.LogWithNode("System", "ConsumerLagProcessor", fmt.Sprintf("Final lag per source: %+v", lagPerSource), "info")
	return lagPerSource, nil
}

// LoadTopicsConfigForLag loads the YAML configuration for lag processing
func LoadTopicsConfigForLag() (TopicsConfig, error) {
	yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
	if err != nil {
		return TopicsConfig{}, fmt.Errorf("failed to read topics_tables.yaml: %w", err)
	}
	var cfg TopicsConfig
	err = yaml.Unmarshal(yamlData, &cfg)
	if err != nil {
		return TopicsConfig{}, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return cfg, nil
}
