// Package kafka_processors implements Kafka bytes per second metrics summarization
// This module processes test runs and generates bytes per second summaries
// based on kafka_Broker_Topic_Metrics for BytesInPerSec and BytesOutPerSec.
package kafka_processors

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// KafkaBytesResult represents the computed bytes per second metrics
type KafkaBytesResult struct {
	MinInputBytesOutPerSec  float64
	MaxInputBytesOutPerSec  float64
	AvgInputBytesOutPerSec  float64
	MinOutputBytesOutPerSec float64
	MaxOutputBytesOutPerSec float64
	AvgOutputBytesOutPerSec float64
	MinInputBytesInPerSec   float64
	MaxInputBytesInPerSec   float64
	AvgInputBytesInPerSec   float64
	MinOutputBytesInPerSec  float64
	MaxOutputBytesInPerSec  float64
	AvgOutputBytesInPerSec  float64
}

// KafkaBytesGroup represents aggregated bytes metrics for input/output groups
type KafkaBytesGroup struct {
	Group string
	Min   float64
	Max   float64
	Avg   float64
}

// fetchKafkaBytesMetrics fetches bytes per second metrics for given topics and time range
// Queries both BytesOutPerSec and BytesInPerSec, grouping by inputTopic vs outputTopic
func fetchKafkaBytesMetrics(chClient *clickhouse.ClickHouseClient, inputTopic string, outputTopics []string, start, end time.Time) (KafkaBytesResult, error) {
	result := KafkaBytesResult{}

	// Build topic list: input + output
	allTopics := append([]string{inputTopic}, outputTopics...)

	// Build IN clause
	topicList := make([]string, len(allTopics))
	for i, t := range allTopics {
		topicList[i] = fmt.Sprintf("'%s'", t)
	}
	topicStr := strings.Join(topicList, ",")

	// Query for BytesOutPerSec
	bytesOutQuery := fmt.Sprintf(`
		SELECT
		    grp AS group,
		    min(OneMinuteRate) AS min_bytes,
		    max(OneMinuteRate) AS max_bytes,
		    avg(OneMinuteRate) AS avg_bytes
		FROM
		(
		    SELECT
		        OneMinuteRate,
		        if(topic = '%s', 'inputTopic', 'outputTopic') AS grp
		    FROM monitoring.kafka_Broker_Topic_Metrics
		    WHERE
		        timestamp >= toDateTime64('%s', 9)
		        AND timestamp <= toDateTime64('%s', 9)
		        AND name = 'BytesOutPerSec'
		        AND topic IN (%s)
		)
		WHERE OneMinuteRate > 0
		GROUP BY grp
		ORDER BY grp
	`, inputTopic, start.Format("2006-01-02 15:04:05.000000000"), end.Format("2006-01-02 15:04:05.000000000"), topicStr)

	logger.LogWithNode("System", "KafkaBytesProcessor", fmt.Sprintf("Running BytesOutPerSec query:\n%s", bytesOutQuery), "debug")

	rows, err := chClient.Client.Query(context.Background(), bytesOutQuery)
	if err != nil {
		return result, fmt.Errorf("bytes out query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var group KafkaBytesGroup
		err := rows.Scan(&group.Group, &group.Min, &group.Max, &group.Avg)
		if err != nil {
			logger.LogWarning("System", "KafkaBytesProcessor", fmt.Sprintf("Failed to scan bytes out row: %v", err))
			continue
		}

		if group.Group == "inputTopic" {
			result.MinInputBytesOutPerSec = group.Min
			result.MaxInputBytesOutPerSec = group.Max
			result.AvgInputBytesOutPerSec = group.Avg
		} else if group.Group == "outputTopic" {
			result.MinOutputBytesOutPerSec = group.Min
			result.MaxOutputBytesOutPerSec = group.Max
			result.AvgOutputBytesOutPerSec = group.Avg
		}
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("bytes out row iteration error: %w", err)
	}

	// Query for BytesInPerSec
	bytesInQuery := fmt.Sprintf(`
		SELECT
		    grp AS group,
		    min(OneMinuteRate) AS min_bytes,
		    max(OneMinuteRate) AS max_bytes,
		    avg(OneMinuteRate) AS avg_bytes
		FROM
		(
		    SELECT
		        OneMinuteRate,
		        if(topic = '%s', 'inputTopic', 'outputTopic') AS grp
		    FROM monitoring.kafka_Broker_Topic_Metrics
		    WHERE
		        timestamp >= toDateTime64('%s', 9)
		        AND timestamp <= toDateTime64('%s', 9)
		        AND name = 'BytesInPerSec'
		        AND topic IN (%s)
		)
		WHERE OneMinuteRate > 0
		GROUP BY grp
		ORDER BY grp
	`, inputTopic, start.Format("2006-01-02 15:04:05.000000000"), end.Format("2006-01-02 15:04:05.000000000"), topicStr)

	logger.LogWithNode("System", "KafkaBytesProcessor", fmt.Sprintf("Running BytesInPerSec query:\n%s", bytesInQuery), "debug")

	rows, err = chClient.Client.Query(context.Background(), bytesInQuery)
	if err != nil {
		return result, fmt.Errorf("bytes in query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var group KafkaBytesGroup
		err := rows.Scan(&group.Group, &group.Min, &group.Max, &group.Avg)
		if err != nil {
			logger.LogWarning("System", "KafkaBytesProcessor", fmt.Sprintf("Failed to scan bytes in row: %v", err))
			continue
		}

		if group.Group == "inputTopic" {
			result.MinInputBytesInPerSec = group.Min
			result.MaxInputBytesInPerSec = group.Max
			result.AvgInputBytesInPerSec = group.Avg
		} else if group.Group == "outputTopic" {
			result.MinOutputBytesInPerSec = group.Min
			result.MaxOutputBytesInPerSec = group.Max
			result.AvgOutputBytesInPerSec = group.Avg
		}
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("bytes in row iteration error: %w", err)
	}

	logger.LogSuccess("System", "KafkaBytesProcessor", "Fetched Kafka bytes metrics")
	return result, nil
}

// ProcessKafkaBytesSummary processes bytes per second metrics for a test run
func ProcessKafkaBytesSummary(chClient *clickhouse.ClickHouseClient, o11ySources []string, startTime, endTime time.Time) (KafkaBytesResult, error) {
	logger.LogWithNode("System", "KafkaBytesProcessor", "Starting Kafka bytes summary processing", "info")

	result := KafkaBytesResult{}

	// Load topics configuration
	yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
	if err != nil {
		return result, fmt.Errorf("failed to read topics_tables.yaml: %w", err)
	}
	var cfg TopicsConfig
	err = yaml.Unmarshal(yamlData, &cfg)
	if err != nil {
		return result, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Process each o11y source
	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				if len(s.InputTopic) == 0 {
					logger.LogWarning("System", "KafkaBytesProcessor", fmt.Sprintf("No input topic for source %s", s.Name))
					continue
				}

				inputTopic := s.InputTopic[0].Name
				outputTopics := make([]string, len(s.OutputTopic))
				for i, ot := range s.OutputTopic {
					outputTopics[i] = ot.Name
				}

				// Fetch bytes metrics for this source
				sourceResult, err := fetchKafkaBytesMetrics(chClient, inputTopic, outputTopics, startTime, endTime)
				if err != nil {
					logger.LogWarning("System", "KafkaBytesProcessor", fmt.Sprintf("Failed to fetch bytes metrics for %s: %v", src, err))
					continue
				}

				// Aggregate across sources (take maximum values)
				if sourceResult.MinInputBytesOutPerSec < result.MinInputBytesOutPerSec || result.MinInputBytesOutPerSec == 0 {
					result.MinInputBytesOutPerSec = sourceResult.MinInputBytesOutPerSec
				}
				if sourceResult.MaxInputBytesOutPerSec > result.MaxInputBytesOutPerSec {
					result.MaxInputBytesOutPerSec = sourceResult.MaxInputBytesOutPerSec
				}
				result.AvgInputBytesOutPerSec += sourceResult.AvgInputBytesOutPerSec

				if sourceResult.MinOutputBytesOutPerSec < result.MinOutputBytesOutPerSec || result.MinOutputBytesOutPerSec == 0 {
					result.MinOutputBytesOutPerSec = sourceResult.MinOutputBytesOutPerSec
				}
				if sourceResult.MaxOutputBytesOutPerSec > result.MaxOutputBytesOutPerSec {
					result.MaxOutputBytesOutPerSec = sourceResult.MaxOutputBytesOutPerSec
				}
				result.AvgOutputBytesOutPerSec += sourceResult.AvgOutputBytesOutPerSec

				if sourceResult.MinInputBytesInPerSec < result.MinInputBytesInPerSec || result.MinInputBytesInPerSec == 0 {
					result.MinInputBytesInPerSec = sourceResult.MinInputBytesInPerSec
				}
				if sourceResult.MaxInputBytesInPerSec > result.MaxInputBytesInPerSec {
					result.MaxInputBytesInPerSec = sourceResult.MaxInputBytesInPerSec
				}
				result.AvgInputBytesInPerSec += sourceResult.AvgInputBytesInPerSec

				if sourceResult.MinOutputBytesInPerSec < result.MinOutputBytesInPerSec || result.MinOutputBytesInPerSec == 0 {
					result.MinOutputBytesInPerSec = sourceResult.MinOutputBytesInPerSec
				}
				if sourceResult.MaxOutputBytesInPerSec > result.MaxOutputBytesInPerSec {
					result.MaxOutputBytesInPerSec = sourceResult.MaxOutputBytesInPerSec
				}
				result.AvgOutputBytesInPerSec += sourceResult.AvgOutputBytesInPerSec
			}
		}
	}

	// Average the averages across sources
	numSources := float64(len(o11ySources))
	if numSources > 0 {
		result.AvgInputBytesOutPerSec /= numSources
		result.AvgOutputBytesOutPerSec /= numSources
		result.AvgInputBytesInPerSec /= numSources
		result.AvgOutputBytesInPerSec /= numSources
	}

	logger.LogSuccess("System", "KafkaBytesProcessor", "Successfully processed Kafka bytes summary")
	return result, nil
}
