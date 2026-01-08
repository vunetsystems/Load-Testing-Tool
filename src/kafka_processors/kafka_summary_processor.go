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
	"os/exec"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
	"vuDataSim/src/pod_processors"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// KafkaStat represents a single Kafka metric data point from ClickHouse
// This is an update for Kafka summarization functionality
type KafkaStat struct {
	Timestamp     time.Time
	Topic         string
	OneMinuteRate float64
	Count         float64 // changed from int64 -> float64 to match ClickHouse
}

// ProcessRateStat represents process rate metrics from ClickHouse
// Added for process rate summary functionality
type ProcessRateStat struct {
	MaxProcessRate float64 `json:"max_process_rate"`
	MinProcessRate float64 `json:"min_process_rate"`
	AvgProcessRate float64 `json:"avg_process_rate"`
}

// type TopicEntry struct {
// 	Name string `yaml:"name"`
// }
// type TopicConfig struct {
// 	Name             string       `yaml:"name"`
// 	InputTopic       []TopicEntry `yaml:"inputTopic"`
// 	OutputTopic      []TopicEntry `yaml:"outputTopic"`
// 	ClickHouseTables []string     `yaml:"clickhouseTables"`
// 	Pipeline         []string     `yaml:"pipeline"`
// }
// type TopicsConfig struct {
// 	Sources []TopicConfig `yaml:"sources"`
// }

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
	var startTimeStr string
	var endTimeNull sql.NullString
	var targetEPS int
	err := row.Scan(&testID, &startTimeStr, &endTimeNull, &o11ySourcesStr, &targetEPS)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.LogWithNode("System", "KafkaSummaryProcessor", "No unsummarized completed tests found", "info")
			return nil
		}
		return fmt.Errorf("failed to query pending test: %w", err)
	}

	// Parse start_time
	startTime, err := time.Parse(time.RFC3339Nano, startTimeStr)
	if err != nil {
		// Try alternative format if RFC3339Nano fails
		startTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", startTimeStr)
		if err != nil {
			return fmt.Errorf("failed to parse start_time '%s': %w", startTimeStr, err)
		}
	}

	// Parse end_time if not null
	var endTime time.Time
	if endTimeNull.Valid {
		endTime, err = time.Parse(time.RFC3339Nano, endTimeNull.String)
		if err != nil {
			// Try alternative format if RFC3339Nano fails
			endTime, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", endTimeNull.String)
			if err != nil {
				return fmt.Errorf("failed to parse end_time '%s': %w", endTimeNull.String, err)
			}
		}
	} else {
		// If end_time is null, use start_time + timeout or something, but for now, set to start_time
		endTime = startTime
	}
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

	// Fetch pipeline information from PostgreSQL
	pipelineInfoJSON, err := ProcessPipelineInfo(o11ySources, endTime)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to fetch pipeline info: %v", err))
		pipelineInfoJSON = ""
	}

	// Process Kafka specs for this test run
	err = ProcessKafkaSpecs(db, testID, o11ySources)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to process Kafka specs: %v", err))
		// Continue with summarization even if specs fail
	}

	// 3. Collect input topics for consumer lag processing
	var inputTopics []string
	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				if len(s.InputTopic) > 0 {
					for _, inputTopic := range s.InputTopic {
						inputTopics = append(inputTopics, inputTopic.Name)
					}
				}
				break
			}
		}
	}

	// 4. Collect app-ids for process rate fetching
	// Added for process rate summary - map o11y sources to app-ids via pipeline field
	appIDMap := make(map[string]string) // source -> app-id
	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				if len(s.Pipeline) > 0 {
					appIDMap[src] = s.Pipeline[0]
				}
				break
			}
		}
	}

	// Fetch process rates for all app-ids
	// Added for process rate summary - query ClickHouse for process-rate metrics
	var appIDs []string
	for _, appID := range appIDMap {
		appIDs = append(appIDs, appID)
	}
	processRates, err := fetchProcessRates(chClient, appIDs, startTime, endTime)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to fetch process rates: %v", err))
		processRates = make(map[string]ProcessRateStat)
	}

	// 5. Collect ClickHouse tables for ingestion EPS calculation
	// Added for ingestion summary - collect all clickhouse tables from o11y sources
	var allClickHouseTables []string
	tableToSource := make(map[string]string)
	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				if len(s.ClickHouseTables) > 0 {
					for _, table := range s.ClickHouseTables {
						tableToSource[table] = src
						allClickHouseTables = append(allClickHouseTables, table)
					}
				}
				break
			}
		}
	}

	// Fetch ingestion EPS data
	// Added for ingestion summary - calculate EPS for all relevant ClickHouse tables
	var ingestionEntries []IngestionEPSEntry
	if len(allClickHouseTables) > 0 {
		ingestionEntries, err = fetchIngestionEPS(chClient, allClickHouseTables, startTime, endTime)
		if err != nil {
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to fetch ingestion EPS: %v", err))
			ingestionEntries = []IngestionEPSEntry{}
		}
	}

	// Populate O11ySource for each ingestion entry
	for i := range ingestionEntries {
		if source, exists := tableToSource[ingestionEntries[i].Table]; exists {
			ingestionEntries[i].O11ySource = source
		}
	}

	// 4. Build summary for each o11y source
	summary := make(map[string]interface{})
	processRateSummary := make(map[string]ProcessRateStat) // Separate summary for process rates

	for _, src := range o11ySources {
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				// guard for empty inputTopic
				if len(s.InputTopic) == 0 {
					logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("No inputTopic for source %s", s.Name))
					continue
				}
				inputTopic := s.InputTopic[0].Name
				var outputTopics []string
				for _, ot := range s.OutputTopic {
					outputTopics = append(outputTopics, ot.Name)
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

				// Add process rate data if available
				// Added for process rate summary - include in both full summary and separate summary
				sourceSummary := map[string]interface{}{
					"input":  inputSummary,
					"output": outputSummary,
				}

				if appID, exists := appIDMap[src]; exists {
					if stat, found := processRates[appID]; found {
						sourceSummary["process_rate"] = stat
						processRateSummary[src] = stat
					}
				}

				summary[src] = sourceSummary
			}
		}
	}

	// 4. Fetch and compute pod JSON metrics with node
	podsCpuMap, podsMemoryMap, podRestartsMap, found, err := pod_processors.ProcessPodResourceSummaryWithNode(chClient, testID, startTime, endTime)
	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("ProcessPodResourceSummaryWithNode returned: found=%t, err=%v", found, err), "info")
	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("podsCpuMap is nil: %t, len: %d", podsCpuMap == nil, len(podsCpuMap)), "info")
	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("podsMemoryMap is nil: %t, len: %d", podsMemoryMap == nil, len(podsMemoryMap)), "info")

	var podsCpuJSON, podsMemoryJSON, podRestartsJSON string
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to process pod JSON metrics: %v", err))
		podsCpuJSON = ""
		podsMemoryJSON = ""
		podRestartsJSON = ""
	} else {
		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("About to marshal maps - CPU map: %v", podsCpuMap), "debug")
		podsCpuBytes, marshalErr := json.Marshal(podsCpuMap)
		if marshalErr != nil {
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to marshal podsCpuMap: %v", marshalErr))
		}
		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Marshaled podsCpuBytes length: %d, content: %s", len(podsCpuBytes), string(podsCpuBytes)), "info")

		podsMemoryBytes, marshalErr := json.Marshal(podsMemoryMap)
		if marshalErr != nil {
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to marshal podsMemoryMap: %v", marshalErr))
		}
		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Marshaled podsMemoryBytes length: %d, content: %s", len(podsMemoryBytes), string(podsMemoryBytes)), "info")

		podRestartsBytes, marshalErr := json.Marshal(podRestartsMap)
		if marshalErr != nil {
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to marshal podRestartsMap: %v", marshalErr))
		}

		podsCpuJSON = string(podsCpuBytes)
		podsMemoryJSON = string(podsMemoryBytes)
		podRestartsJSON = string(podRestartsBytes)

		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Final JSON strings - CPU len: %d, Memory len: %d, Restarts len: %d", len(podsCpuJSON), len(podsMemoryJSON), len(podRestartsJSON)), "info")
	}

	// 5. Fetch and compute consumer lag metrics
	minLag, avgLag, maxLag, err := ProcessConsumerLagSummary(chClient, inputTopics, startTime, endTime)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to process consumer lag metrics: %v", err))
		minLag, avgLag, maxLag = 0.0, 0.0, 0.0
	}

	// 5.1. Fetch and compute consumer lag metrics per source (NEW)
	logger.LogWithNode("System", "KafkaSummaryProcessor", "About to call ProcessPerSourceLagSummary", "info")
	err = ProcessPerSourceLagSummary(db, chClient, testID, o11ySources, startTime, endTime)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to process per-source lag: %v", err))
		// Continue with summarization even if per-source lag fails
	} else {
		logger.LogWithNode("System", "KafkaSummaryProcessor", "Per-source lag processing completed successfully", "info")
	}

	// 7. Fetch and compute node JSON metrics
	nodesCpuMap, nodesMemoryMap, _, err := ProcessNodeResourceSummaryJSON(chClient, startTime, endTime)
	var nodesCpuJSON, nodesMemoryJSON string
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to process node JSON metrics: %v", err))
		nodesCpuJSON = ""
		nodesMemoryJSON = ""
	} else {
		nodesCpuBytes, _ := json.Marshal(nodesCpuMap)
		nodesMemoryBytes, _ := json.Marshal(nodesMemoryMap)
		nodesCpuJSON = string(nodesCpuBytes)
		nodesMemoryJSON = string(nodesMemoryBytes)
	}

	// 8. Compute detailed topic metrics
	detailedTopicMetrics, err := computeDetailedTopicMetrics(o11ySources, cfg, chClient, startTime, endTime)
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to compute detailed topic metrics: %v", err))
		detailedTopicMetrics = make(map[string]map[string]map[string]interface{})
	}
	topicMetricsJSON, _ := json.Marshal(detailedTopicMetrics)

	// 9. Store the summary back to the database
	processRateSummaryJSON, _ := json.Marshal(processRateSummary) // Added for process rate summary - store in separate column
	ingestionSummaryJSON, _ := json.Marshal(ingestionEntries)     // Added for ingestion summary - store EPS data for ClickHouse tables

	// Calculate averages across all sources
	var avgInputRate, avgOutputRate, peakInputRate, peakOutputRate, minInputRate, minOutputRate float64
	minInputRate = math.MaxFloat64
	minOutputRate = math.MaxFloat64

	for _, srcSummary := range summary {
		if srcData, ok := srcSummary.(map[string]interface{}); ok {
			if inputData, ok := srcData["input"].(map[string]interface{}); ok {
				avgInputRate += inputData["avg_msgs_per_sec"].(float64)
				if inputData["max_msgs_per_sec"].(float64) > peakInputRate {
					peakInputRate = inputData["max_msgs_per_sec"].(float64)
				}
				if inputData["min_msgs_per_sec"].(float64) < minInputRate {
					minInputRate = inputData["min_msgs_per_sec"].(float64)
				}
			}
			if outputData, ok := srcData["output"].(map[string]interface{}); ok {
				avgOutputRate += outputData["avg_msgs_per_sec"].(float64)
				if outputData["max_msgs_per_sec"].(float64) > peakOutputRate {
					peakOutputRate = outputData["max_msgs_per_sec"].(float64)
				}
				if outputData["min_msgs_per_sec"].(float64) < minOutputRate {
					minOutputRate = outputData["min_msgs_per_sec"].(float64)
				}
			}
		}
	}

	// If min rates stayed at MaxFloat64 (no data), set to 0
	if minInputRate == math.MaxFloat64 {
		minInputRate = 0.0
	}
	if minOutputRate == math.MaxFloat64 {
		minOutputRate = 0.0
	}

	_, err = db.Exec(`
		UPDATE test_runs
		SET avg_input_msgs_per_sec = ?, avg_output_msgs_per_sec = ?,
		    max_input_msgs_per_sec = ?, max_output_msgs_per_sec = ?,
		    min_input_msgs_per_sec = ?, min_output_msgs_per_sec = ?,
		    kafka_summary_generated = 1,
		    process_rate_summary = ?, ingestion_summary = ?, pipeline_info = ?,
		    pods_cpu = ?, pods_memory = ?, pod_restarts = ?,
		    nodes_cpu = ?, nodes_memory = ?,
		    min_lag = ?, avg_lag = ?, max_lag = ?, topic_metrics_json = ?
		WHERE test_id = ?;
	`, avgInputRate, avgOutputRate,
		peakInputRate, peakOutputRate, minInputRate, minOutputRate,
		string(processRateSummaryJSON), string(ingestionSummaryJSON), pipelineInfoJSON,
		podsCpuJSON, podsMemoryJSON, podRestartsJSON,
		nodesCpuJSON, nodesMemoryJSON,
		minLag, avgLag, maxLag, string(topicMetricsJSON),
		testID)
	if err != nil {
		return fmt.Errorf("failed to update test run with summary: %w", err)
	}

	logger.LogSuccess("System", "KafkaSummaryProcessor", fmt.Sprintf("Successfully processed Kafka summary for test %s", testID))

	// Run the combined report generator script for this specific test
	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Attempting to execute generate_combined_report.py for test %s", testID), "info")
	cmd := exec.Command("python3", "generate_combined_report.py", "--test-id", testID, "--table", "test_runs")
	cmd.Dir = "data" // Set working directory to 'data' since the script and its dependencies are located there

	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Script working directory: %s", cmd.Dir), "debug")
	logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Script command: python3 generate_combined_report.py --test-id %s --table test_runs", testID), "debug")

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to execute generate_combined_report.py: %v", err))
		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Script output: %s", string(output)), "debug")
	} else {
		logger.LogSuccess("System", "KafkaSummaryProcessor", "Successfully executed generate_combined_report.py")
		logger.LogWithNode("System", "KafkaSummaryProcessor", fmt.Sprintf("Script output: %s", string(output)), "info")
	}

	return nil
}

// fetchKafkaMetrics fetches Kafka metrics from ClickHouse for given topics and time range
// This fixed version correctly expands the IN clause (ClickHouse drivers don’t auto-expand slices).
func fetchKafkaMetrics(chClient *clickhouse.ClickHouseClient, topics []string, start, end time.Time) ([]KafkaStat, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}

	// Build the IN clause dynamically for the topics slice
	topicList := make([]string, len(topics))
	for i, t := range topics {
		topicList[i] = fmt.Sprintf("'%s'", t)
	}
	topicStr := strings.Join(topicList, ",")

	query := fmt.Sprintf(`
		WITH per_second AS (
			SELECT
				toStartOfInterval(timestamp, toIntervalSecond(1)) AS ts,
				topic,
				sum(OneMinuteRate) AS eps_per_sec
			FROM monitoring.kafka_Broker_Topic_Metrics
			WHERE topic IN (%s)
			  AND name = 'MessagesInPerSec'
			  AND timestamp >= toDateTime('%s')
			  AND timestamp <= toDateTime('%s')
			GROUP BY ts, topic
		)
		SELECT ts, topic, eps_per_sec FROM per_second ORDER BY ts ASC
	`, topicStr, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	// debug log to confirm the query and topic list
	logger.LogWithNode("System", "KafkaFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	// NOTE: use the field exposed by your clickhouse wrapper. In this project the wrapper exposes Client.
	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []KafkaStat

	for rows.Next() {
		var stat KafkaStat
		// Important: eps_per_sec is Float64 in ClickHouse!
		var oneMinuteRate float64

		if err := rows.Scan(&stat.Timestamp, &stat.Topic, &oneMinuteRate); err != nil {
			// log and continue so a single bad row doesn't stop processing
			logger.LogWarning("System", "KafkaSummaryProcessor", fmt.Sprintf("Failed to scan metric row: %v", err))
			continue
		}

		stat.OneMinuteRate = oneMinuteRate
		stat.Count = 0 // Not available in per-second aggregation
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "KafkaFetcher", "No Kafka metrics found for given topics/time range", "warn")
	} else {
		logger.LogSuccess("System", "KafkaFetcher", fmt.Sprintf("Fetched %d Kafka metric rows", len(stats)))
	}

	return stats, nil
}

// IngestionEPSEntry represents a single ingestion EPS entry for a table
// Added for ingestion summary functionality
type IngestionEPSEntry struct {
	Table      string  `json:"table"`
	O11ySource string  `json:"o11y_source"`
	AvgEPS     float64 `json:"avg_eps"`
	MinEPS     float64 `json:"min_eps"`
	MaxEPS     float64 `json:"max_eps"`
}

// fetchIngestionEPS fetches ingestion EPS metrics from ClickHouse system.part_log for given tables and time range
// Added for ingestion summary functionality - calculates average EPS for tables based on part log events
func fetchIngestionEPS(chClient *clickhouse.ClickHouseClient, tables []string, start, end time.Time) ([]IngestionEPSEntry, error) {
	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables provided")
	}

	// Build the IN clause for tables
	tableList := make([]string, len(tables))
	for i, t := range tables {
		tableList[i] = fmt.Sprintf("'%s'", t)
	}
	tableStr := strings.Join(tableList, ",")

	query := fmt.Sprintf(`
		SELECT
			`+"`table`"+`,
			avg(eps) AS avg_eps,
			min(eps) AS min_eps,
			max(eps) AS max_eps
		FROM
		(
			SELECT
				`+"`table`"+`,
				toStartOfMinute(event_time) AS minute,
				sum(rows) / 60 AS eps
			FROM system.part_log
			WHERE
				event_type = 'NewPart'
				AND event_time >= toDateTime('%s')
				AND event_time <= toDateTime('%s')
				AND `+"`table`"+` IN (%s)
			GROUP BY
				minute, `+"`table`"+`
		)
		GROUP BY
			`+"`table`"+`
		ORDER BY
			avg_eps DESC
	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"), tableStr)

	logger.LogWithNode("System", "IngestionEPSFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var entries []IngestionEPSEntry

	for rows.Next() {
		var entry IngestionEPSEntry

		if err := rows.Scan(&entry.Table, &entry.AvgEPS, &entry.MinEPS, &entry.MaxEPS); err != nil {
			logger.LogWarning("System", "IngestionEPSFetcher", fmt.Sprintf("Failed to scan ingestion EPS row: %v", err))
			continue
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(entries) == 0 {
		logger.LogWithNode("System", "IngestionEPSFetcher", "No ingestion EPS data found for given tables/time range", "warn")
	} else {
		logger.LogSuccess("System", "IngestionEPSFetcher", fmt.Sprintf("Fetched ingestion EPS for %d tables", len(entries)))
	}

	return entries, nil
}

// fetchProcessRates fetches process rate metrics from ClickHouse for given app-ids and time range
// Added for process rate summary functionality - queries monitoring.kafka_streams_ThreadMetrics
func fetchProcessRates(chClient *clickhouse.ClickHouseClient, appIDs []string, start, end time.Time) (map[string]ProcessRateStat, error) {
	if len(appIDs) == 0 {
		return nil, fmt.Errorf("no app-ids provided")
	}

	// Build the IN clause for app-ids
	appIDList := make([]string, len(appIDs))
	for i, id := range appIDs {
		appIDList[i] = fmt.Sprintf("'%s'", id)
	}
	appIDStr := strings.Join(appIDList, ",")

	query := fmt.Sprintf(`SELECT
	   `+"`app-id`"+` AS app_id,
	   MAX(interval_sum) AS max_process_rate,
	   MIN(interval_sum) AS min_process_rate,
	   AVG(interval_sum) AS avg_process_rate
FROM (
	   SELECT
	       `+"`app-id`"+`,
	       toStartOfInterval(timestamp, toIntervalSecond(1)) AS time,
	       sum(`+"`process-rate`"+`) AS interval_sum
	   FROM monitoring.kafka_streams_ThreadMetrics
	   WHERE `+"`app-id`"+` IN (%s)
	     AND timestamp >= toDateTime64('%s', 9)
	     AND timestamp <= toDateTime64('%s', 9)
	   GROUP BY `+"`app-id`"+`, time
	   HAVING interval_sum > 0
) AS per_second
GROUP BY `+"`app-id`"+`
ORDER BY `+"`app-id`"+``, appIDStr, start.Format("2006-01-02 15:04:05.000000000"), end.Format("2006-01-02 15:04:05.000000000"))

	logger.LogWithNode("System", "ProcessRateFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	processRates := make(map[string]ProcessRateStat)

	for rows.Next() {
		var appID string
		var stat ProcessRateStat

		if err := rows.Scan(&appID, &stat.MaxProcessRate, &stat.MinProcessRate, &stat.AvgProcessRate); err != nil {
			logger.LogWarning("System", "ProcessRateFetcher", fmt.Sprintf("Failed to scan process rate row: %v", err))
			continue
		}

		processRates[appID] = stat
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(processRates) == 0 {
		logger.LogWithNode("System", "ProcessRateFetcher", "No process rate metrics found for given app-ids/time range", "warn")
	} else {
		logger.LogSuccess("System", "ProcessRateFetcher", fmt.Sprintf("Fetched process rates for %d app-ids", len(processRates)))
	}

	return processRates, nil
}

// computeKafkaStats computes statistics for a single topic
// This is an update for Kafka summarization functionality
func computeKafkaStats(stats []KafkaStat, topicName string) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"topic":              topicName,
			"avg_msgs_per_sec":   0.0,
			"max_msgs_per_sec":   0.0,
			"min_msgs_per_sec":   0.0,
			"stdev_msgs_per_sec": 0.0,
			"anomaly_spikes":     0,
			"total_messages":     0.0,
		}
	}

	var sum, min, max float64
	min = math.MaxFloat64
	var maxCount float64
	values := make([]float64, len(stats))

	for i, stat := range stats {
		sum += stat.OneMinuteRate
		if stat.OneMinuteRate < min {
			min = stat.OneMinuteRate
		}
		if stat.OneMinuteRate > max {
			max = stat.OneMinuteRate
		}
		if stat.Count > maxCount {
			maxCount = stat.Count
		}
		values[i] = stat.OneMinuteRate
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

	// Total messages is the maximum Count value (cumulative)
	totalMessages := maxCount

	return map[string]interface{}{
		"topic":              topicName,
		"avg_msgs_per_sec":   mean,
		"max_msgs_per_sec":   max,
		"min_msgs_per_sec":   min,
		"stdev_msgs_per_sec": stdev,
		"anomaly_spikes":     anomalies,
		"total_messages":     totalMessages,
	}
}

// computeKafkaStatsMulti computes statistics for multiple output topics combined
// This is an update for Kafka summarization functionality
func computeKafkaStatsMulti(stats []KafkaStat, topicNames []string) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"total_topics":          len(topicNames),
			"avg_msgs_per_sec":      0.0,
			"max_msgs_per_sec":      0.0,
			"min_msgs_per_sec":      0.0,
			"stdev_msgs_per_sec":    0.0,
			"anomaly_spikes":        0,
			"total_messages":        0.0,
			"output_efficiency_pct": 0.0,
		}
	}

	// Group stats by topic for total_messages calculation
	grouped := make(map[string][]KafkaStat)
	for _, stat := range stats {
		grouped[stat.Topic] = append(grouped[stat.Topic], stat)
	}

	// Aggregate rates by timestamp across all topics
	timestampSums := make(map[time.Time]float64)
	for _, stat := range stats {
		timestampSums[stat.Timestamp] += stat.OneMinuteRate
	}

	// Calculate max, min, and avg from timestamp sums
	var maxRate float64
	var minRate float64 = math.MaxFloat64
	var totalSum float64
	var count int
	var sums []float64

	for _, sum := range timestampSums {
		if sum > maxRate {
			maxRate = sum
		}
		if sum < minRate {
			minRate = sum
		}
		totalSum += sum
		sums = append(sums, sum)
		count++
	}

	avgRate := totalSum / float64(count)

	// Calculate stdev from timestamp sums
	var stdev float64
	if len(sums) > 0 {
		mean := totalSum / float64(len(sums))
		var variance float64
		for _, v := range sums {
			variance += (v - mean) * (v - mean)
		}
		stdev = math.Sqrt(variance / float64(len(sums)))
	}

	// Count anomalies (beyond ±2σ) from timestamp sums
	anomalies := 0
	if len(sums) > 0 {
		mean := totalSum / float64(len(sums))
		for _, v := range sums {
			if math.Abs(v-mean) > 2*stdev {
				anomalies++
			}
		}
	}

	// Calculate total messages (sum of max count per topic)
	var totalMessages float64
	for _, tstats := range grouped {
		var topicMaxCount float64
		for _, stat := range tstats {
			if float64(stat.Count) > topicMaxCount {
				topicMaxCount = float64(stat.Count)
			}
		}
		totalMessages += topicMaxCount
	}

	return map[string]interface{}{
		"total_topics":          len(grouped),
		"avg_msgs_per_sec":      avgRate,
		"max_msgs_per_sec":      maxRate,
		"min_msgs_per_sec":      minRate,
		"stdev_msgs_per_sec":    stdev,
		"anomaly_spikes":        anomalies,
		"total_messages":        totalMessages,
		"output_efficiency_pct": 0.0, // will be computed by caller
	}
}

// ifaceToFloat64 safely converts a variety of numeric interface{} values to float64
func ifaceToFloat64(v interface{}) float64 {
	switch t := v.(type) {
	case nil:
		return 0
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case uint:
		return float64(t)
	case uint64:
		return float64(t)
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f
		}
		return 0
	default:
		return 0
	}
}

// computeDetailedTopicMetrics computes per-topic avg/min/max message rates for all topics across all o11y sources
// Returns a nested map structure: source -> topic -> {avg, min, max, topic_type}
func computeDetailedTopicMetrics(o11ySources []string, cfg TopicsConfig, chClient *clickhouse.ClickHouseClient, start, end time.Time) (map[string]map[string]map[string]interface{}, error) {
	detailedMetrics := make(map[string]map[string]map[string]interface{})

	for _, src := range o11ySources {
		sourceMetrics := make(map[string]map[string]interface{})

		// Find the source config
		var sourceConfig TopicConfig
		found := false
		for _, s := range cfg.Sources {
			if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
				sourceConfig = s
				found = true
				break
			}
		}
		if !found {
			continue
		}

		// Process input topics
		for _, inputTopic := range sourceConfig.InputTopic {
			stats, err := fetchKafkaMetrics(chClient, []string{inputTopic.Name}, start, end)
			if err != nil {
				logger.LogWarning("System", "DetailedTopicMetrics", fmt.Sprintf("Failed to fetch input metrics for %s topic %s: %v", src, inputTopic.Name, err))
				continue
			}
			summary := computeKafkaStats(stats, inputTopic.Name)
			sourceMetrics[inputTopic.Name] = map[string]interface{}{
				"avg":        summary["avg_msgs_per_sec"].(float64),
				"min":        summary["min_msgs_per_sec"].(float64),
				"max":        summary["max_msgs_per_sec"].(float64),
				"topic_type": "input",
			}
		}

		// Process output topics
		for _, outputTopic := range sourceConfig.OutputTopic {
			stats, err := fetchKafkaMetrics(chClient, []string{outputTopic.Name}, start, end)
			if err != nil {
				logger.LogWarning("System", "DetailedTopicMetrics", fmt.Sprintf("Failed to fetch output metrics for %s topic %s: %v", src, outputTopic.Name, err))
				continue
			}
			summary := computeKafkaStats(stats, outputTopic.Name)
			sourceMetrics[outputTopic.Name] = map[string]interface{}{
				"avg":        summary["avg_msgs_per_sec"].(float64),
				"min":        summary["min_msgs_per_sec"].(float64),
				"max":        summary["max_msgs_per_sec"].(float64),
				"topic_type": "output",
			}
		}

		if len(sourceMetrics) > 0 {
			detailedMetrics[src] = sourceMetrics
		}
	}

	return detailedMetrics, nil
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

// ProcessPerSourceLagSummary processes lag metrics per source for a test run
func ProcessPerSourceLagSummary(db *sql.DB, chClient *clickhouse.ClickHouseClient, testID string, o11ySources []string, startTime, endTime time.Time) error {
	logger.LogWithNode("System", "PerSourceLagProcessor", fmt.Sprintf("Starting per-source lag processing for test %s", testID), "info")
	logger.LogWithNode("System", "PerSourceLagProcessor", fmt.Sprintf("Processing time range: %s to %s", startTime, endTime), "info")
	logger.LogWithNode("System", "PerSourceLagProcessor", fmt.Sprintf("Processing sources: %v", o11ySources), "info")

	// Load config
	cfg, err := LoadTopicsConfigForLag()
	if err != nil {
		logger.LogError("System", "PerSourceLagProcessor", fmt.Sprintf("Failed to load topics config: %v", err))
		return fmt.Errorf("failed to load topics config: %w", err)
	}

	// Compute per-source lag
	lagPerSource, err := ProcessConsumerLagSummaryPerSource(chClient, o11ySources, cfg, startTime, endTime)
	if err != nil {
		logger.LogError("System", "PerSourceLagProcessor", fmt.Sprintf("Failed to process per-source lag: %v", err))
		return fmt.Errorf("failed to process per-source lag: %w", err)
	}

	logger.LogWithNode("System", "PerSourceLagProcessor", fmt.Sprintf("Computed lag data: %+v", lagPerSource), "info")

	// Convert to JSON and update database
	lagPerSourceJSON, _ := json.Marshal(lagPerSource)
	logger.LogWithNode("System", "PerSourceLagProcessor", fmt.Sprintf("Storing JSON: %s", string(lagPerSourceJSON)), "info")

	_, err = db.Exec(`
		UPDATE test_runs
		SET lag_per_source = ?
		WHERE test_id = ?`,
		string(lagPerSourceJSON), testID)

	if err != nil {
		logger.LogError("System", "PerSourceLagProcessor", fmt.Sprintf("Failed to update database: %v", err))
		return fmt.Errorf("failed to update lag_per_source: %w", err)
	}

	logger.LogSuccess("System", "PerSourceLagProcessor", fmt.Sprintf("Successfully processed per-source lag for test %s", testID))
	return nil
}
