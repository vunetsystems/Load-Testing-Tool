package kafka_ch_reset

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"vuDataSim/src/logger"
	"gopkg.in/yaml.v3"
)

// TopicName represents a topic name structure
type TopicName struct {
	Name string `yaml:"name"`
}

// TopicConfig represents the configuration for a topic group
type TopicConfig struct {
	Name             string      `yaml:"name"`
	InputTopic       []TopicName `yaml:"inputTopic"`
	OutputTopic      []TopicName `yaml:"outputTopic"`
	ClickhouseTables []string    `yaml:"clickhouseTables"`
}



// TopicMetadata stores partition and replication factor for a topic
type TopicMetadata struct {
	TopicName        string
	PartitionCount   int
	ReplicationFactor int
}

// KafkaManager handles Kafka topic operations
type KafkaManager struct {
	configPath string
	topics     []TopicConfig
}

// O11ySourceConfig represents the configuration for o11y sources from conf.yml
type O11ySourceConfig struct {
	DataGenerationTime struct {
		Type string `yaml:"type"`
	} `yaml:"data_generation_time"`
	IncludeModuleDirs map[string]struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"include_module_dirs"`
}

// Source name translation dictionary to map between conf.yml and topics_tables.yaml naming conventions
var sourceNameTranslation = map[string]string{
	"LinuxMonitor":      "Linux Monitor",
	"MongoDB":           "MongoDB",
	"Mssql":             "MSSQL",
	"Apache":            "Apache",
	"Azure_Firewall":    "Azure Firewall",
	"Azure_Redis_Cache": "Azure Redis Cache",
}

// translateSourceName translates source names between conf.yml and topics_tables.yaml naming conventions
func (km *KafkaManager) translateSourceName(sourceName string) string {
	if translatedName, exists := sourceNameTranslation[sourceName]; exists {
		return translatedName
	}
	// Return original name if no translation found
	return sourceName
}

// NewKafkaManager creates a new KafkaManager instance
func NewKafkaManager(configPath string) *KafkaManager {
	return &KafkaManager{
		configPath: configPath,
	}
}

// SourcesConfig represents the wrapper structure for sources
type SourcesConfig struct {
	Sources []TopicConfig `yaml:"sources"`
}

// LoadConfig loads the topic configuration from YAML file
func (km *KafkaManager) LoadConfig() error {
	logger.Info().Str("path", km.configPath).Msg("Loading topic configuration")
	data, err := exec.Command("cat", km.configPath).Output()
	if err != nil {
		logger.Error().Err(err).Str("path", km.configPath).Msg("Failed to read config file")
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config SourcesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		logger.Error().Err(err).Str("path", km.configPath).Str("data", string(data)).Msg("Failed to parse YAML config")
		return fmt.Errorf("failed to parse YAML config: %v", err)
	}

	logger.Info().Int("sources", len(config.Sources)).Msg("Successfully loaded topic configurations")
	for i, source := range config.Sources {
		logger.Debug().Int("index", i).Str("source", source.Name).Msg("Loaded source configuration")
	}

	km.topics = config.Sources
	return nil
}

// GetAllTopics returns all configured topics
func (km *KafkaManager) GetAllTopics() []TopicConfig {
	return km.topics
}


// DescribeTopic describes a single topic and returns its metadata
func (km *KafkaManager) DescribeTopic(topicName string) (*TopicMetadata, error) {
	describeCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --describe --topic %s", topicName)
	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Str("topic", topicName).Msg("Failed to execute describe command")
		return nil, fmt.Errorf("failed to describe topic %s: %v", topicName, err)
	}

	metadata, err := km.parseTopicDescription(string(output))
	if err != nil {
		logger.Error().Err(err).Str("topic", topicName).Str("output", string(output)).Msg("Failed to parse topic description")
		return nil, fmt.Errorf("failed to parse topic description for %s: %v", topicName, err)
	}

	return metadata, nil
}

// DeleteTopic deletes a single topic
func (km *KafkaManager) DeleteTopic(topicName string) error {
	deleteCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --delete --topic %s", topicName)
	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", deleteCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Warn().Err(err).Str("topic", topicName).Str("output", string(output)).Msg("Delete command failed (might be expected if topic doesn't exist)")
		// Note: Delete might fail if topic doesn't exist, which is okay for some use cases
		return fmt.Errorf("failed to delete topic %s: %v", topicName, err)
	}

	logger.Info().Str("topic", topicName).Msg("Topic delete command executed successfully")
	return nil
}

// CreateTopic creates a single topic with specified metadata
func (km *KafkaManager) CreateTopic(topicName string, partitionCount, replicationFactor int) error {
	createCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --create --topic %s --partitions %d --replication-factor %d",
		topicName, partitionCount, replicationFactor)

	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", createCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Str("topic", topicName).Int("partitions", partitionCount).Int("replicationFactor", replicationFactor).Str("output", string(output)).Msg("Failed to create topic")
		return fmt.Errorf("failed to create topic %s: %v", topicName, err)
	}

	logger.Info().Str("topic", topicName).Int("partitions", partitionCount).Int("replicationFactor", replicationFactor).Msg("Topic created successfully")
	return nil
}

// LoadO11yConfig loads the o11y source configuration from conf.yml file
func (km *KafkaManager) LoadO11yConfig(confPath string) (*O11ySourceConfig, error) {
	data, err := exec.Command("cat", confPath).Output()
	if err != nil {
		logger.Error().Err(err).Str("path", confPath).Msg("Failed to read o11y config file")
		return nil, fmt.Errorf("failed to read o11y config file: %v", err)
	}

	var config O11ySourceConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		logger.Error().Err(err).Str("path", confPath).Str("data", string(data)).Msg("Failed to parse YAML o11y config")
		return nil, fmt.Errorf("failed to parse YAML o11y config: %v", err)
	}

	logger.Info().Str("path", confPath).Msg("Successfully loaded o11y config")
	return &config, nil
}


// RecreateTopicsForO11ySources recreates topics for enabled o11y sources from conf.yml using parallel processing
func (km *KafkaManager) RecreateTopicsForO11ySources() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": true,
		"results": make(map[string]string),
		"errors":  make([]string, 0),
		"processed_sources": make([]string, 0),
	}

	// Use a mutex to protect shared result map
	var resultMu sync.Mutex

	// Step 1: Load o11y configuration from conf.yml
	confPath := "src/migrate/conf.d/conf.yml"
	o11yConfig, err := km.LoadO11yConfig(confPath)
	if err != nil {
		resultMu.Lock()
		result["success"] = false
		result["errors"] = append(result["errors"].([]string), fmt.Sprintf("Failed to load o11y config: %v", err))
		resultMu.Unlock()
		return result, err
	}

	// Step 2: Find enabled o11y sources
	enabledSources := make([]string, 0)
	for sourceName, sourceConfig := range o11yConfig.IncludeModuleDirs {
		if sourceConfig.Enabled {
			enabledSources = append(enabledSources, sourceName)
			logger.Info().Str("source", sourceName).Msg("Found enabled o11y source")
		}
	}

	if len(enabledSources) == 0 {
		resultMu.Lock()
		result["success"] = false
		result["errors"] = append(result["errors"].([]string), "No enabled o11y sources found in conf.yml")
		resultMu.Unlock()
		return result, fmt.Errorf("no enabled o11y sources found")
	}

	resultMu.Lock()
	result["processed_sources"] = enabledSources
	resultMu.Unlock()

	// Step 3: Collect all topics that need to be recreated
	var allTopics []string
	sourceMap := make(map[string]*TopicConfig)

	for _, sourceName := range enabledSources {
		translatedName := km.translateSourceName(sourceName)
		logger.Info().Str("source", sourceName).Str("translated", translatedName).Msg("Processing enabled source")

		// Find the topic configuration for this source
		var sourceTopicConfig *TopicConfig
		for _, topicConfig := range km.topics {
			if topicConfig.Name == translatedName {
				sourceTopicConfig = &topicConfig
				break
			}
		}

		if sourceTopicConfig == nil {
			errMsg := fmt.Sprintf("No topic configuration found for source: %s (translated: %s)", sourceName, translatedName)
			resultMu.Lock()
			result["success"] = false
			result["errors"] = append(result["errors"].([]string), errMsg)
			resultMu.Unlock()
			logger.Error().Str("source", sourceName).Str("translated", translatedName).Msg("No topic configuration found")
			continue
		}

		sourceMap[sourceName] = sourceTopicConfig

		// Collect all input and output topics
		for _, inputTopic := range sourceTopicConfig.InputTopic {
			allTopics = append(allTopics, inputTopic.Name)
		}
		for _, outputTopic := range sourceTopicConfig.OutputTopic {
			allTopics = append(allTopics, outputTopic.Name)
		}
	}

	// Step 4: Describe all topics in parallel first
	metadataMap := make(map[string]*TopicMetadata)
	var metadataMu sync.Mutex
	var describeError error
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, topicName := range allTopics {
		wg.Add(1)
		go func(topic string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				metadata, err := km.DescribeTopic(topic)
				if err != nil {
					metadataMu.Lock()
					if describeError == nil { // Only set error once
						describeError = err
					}
					metadataMu.Unlock()
					resultMu.Lock()
					result["success"] = false
					errorMsg := fmt.Sprintf("Describe operation failed for topic %s: %v", topic, err)
					result["errors"] = append(result["errors"].([]string), errorMsg)
					resultMu.Unlock()
					cancel() // Cancel all other goroutines
					return
				}
				metadataMu.Lock()
				metadataMap[topic] = metadata
				metadataMu.Unlock()
			}
		}(topicName)
	}

	wg.Wait()

	if describeError != nil {
		logger.Error().Err(describeError).Msg("Stopping topic recreation due to describe error")
		return result, describeError
	}

	logger.Info().Int("total_topics", len(allTopics)).Msg("All topics described successfully")

	// Step 5: Delete all topics in parallel
	for _, topicName := range allTopics {
		wg.Add(1)
		go func(topic string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				err := km.DeleteTopic(topic)
				if err != nil {
					resultMu.Lock()
					result["success"] = false
					errorMsg := fmt.Sprintf("Delete operation failed for topic %s: %v", topic, err)
					result["errors"] = append(result["errors"].([]string), errorMsg)
					resultMu.Unlock()
					cancel() // Cancel all other goroutines
				}
			}
		}(topicName)
	}

	wg.Wait()

	resultMu.Lock()
	success := result["success"].(bool)
	resultMu.Unlock()

	if !success {
		logger.Error().Msg("Stopping topic recreation due to delete errors")
		return result, fmt.Errorf("delete errors occurred")
	}

	logger.Info().Int("total_topics", len(allTopics)).Msg("All topics deleted successfully")

	// Add wait delay between delete and create operations
	logger.Info().Msg("Waiting 5 seconds before creating topics...")
	time.Sleep(5 * time.Second)

	// Step 6: Create all topics in parallel
	for _, topicName := range allTopics {
		wg.Add(1)
		go func(topic string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				metadataMu.Lock()
				metadata := metadataMap[topic]
				metadataMu.Unlock()
				err := km.CreateTopic(topic, metadata.PartitionCount, metadata.ReplicationFactor)
				if err != nil {
					resultMu.Lock()
					result["success"] = false
					errorMsg := fmt.Sprintf("Create operation failed for topic %s: %v", topic, err)
					result["errors"] = append(result["errors"].([]string), errorMsg)
					resultMu.Unlock()
					cancel() // Cancel all other goroutines
				} else {
					resultMu.Lock()
					result["results"].(map[string]string)[topic] = "recreated"
					resultMu.Unlock()
				}
			}
		}(topicName)
	}

	wg.Wait()

	resultMu.Lock()
	finalSuccess := result["success"].(bool)
	resultMu.Unlock()

	if !finalSuccess {
		logger.Error().Msg("Stopping topic recreation due to create errors")
		return result, fmt.Errorf("create errors occurred")
	}

	logger.Info().Int("total_topics", len(allTopics)).Msg("Completed parallel topic recreation")

	return result, nil
}

// parseTopicDescription parses the output of kafka-topics --describe command
func (km *KafkaManager) parseTopicDescription(output string) (*TopicMetadata, error) {
	lines := strings.Split(output, "\n")
	metadata := &TopicMetadata{}

	// Regex patterns to extract information
	partitionPattern := regexp.MustCompile(`PartitionCount:\s*(\d+)`)
	replicationPattern := regexp.MustCompile(`ReplicationFactor:\s*(\d+)`)

	for _, line := range lines {
		// Skip the Jolokia warning line
		if strings.Contains(line, "Could not start Jolokia agent") {
			continue
		}

		// Extract partition count
		if match := partitionPattern.FindStringSubmatch(line); match != nil {
			if count, err := strconv.Atoi(match[1]); err == nil {
				metadata.PartitionCount = count
			}
		}

		// Extract replication factor
		if match := replicationPattern.FindStringSubmatch(line); match != nil {
			if factor, err := strconv.Atoi(match[1]); err == nil {
				metadata.ReplicationFactor = factor
			}
		}

		// Extract topic name from the Topic: line
		if strings.HasPrefix(line, "Topic:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				metadata.TopicName = parts[1]
			}
		}
	}

	// Validate that we got the required information
	if metadata.PartitionCount == 0 || metadata.ReplicationFactor == 0 {
		return nil, fmt.Errorf("could not extract partition count or replication factor from output")
	}

	return metadata, nil
}

// GetTableNamesForO11ySources returns table names for enabled o11y sources from conf.yml
func (km *KafkaManager) GetTableNamesForO11ySources() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": true,
		"results": make(map[string][]string),
		"errors":  make([]string, 0),
		"processed_sources": make([]string, 0),
	}

	// Step 1: Load o11y configuration from conf.yml
	confPath := "src/migrate/conf.d/conf.yml"
	o11yConfig, err := km.LoadO11yConfig(confPath)
	if err != nil {
		result["success"] = false
		result["errors"] = append(result["errors"].([]string), fmt.Sprintf("Failed to load o11y config: %v", err))
		return result, err
	}

	// Step 2: Find enabled o11y sources
	enabledSources := make([]string, 0)
	for sourceName, sourceConfig := range o11yConfig.IncludeModuleDirs {
		if sourceConfig.Enabled {
			enabledSources = append(enabledSources, sourceName)
			logger.Info().Str("source", sourceName).Msg("Found enabled o11y source")
		}
	}

	if len(enabledSources) == 0 {
		result["success"] = false
		result["errors"] = append(result["errors"].([]string), "No enabled o11y sources found in conf.yml")
		return result, fmt.Errorf("no enabled o11y sources found")
	}

	result["processed_sources"] = enabledSources

	// Step 3: Collect all table names for enabled sources
	var allTables []string
	sourceTableMap := make(map[string][]string)

	for _, sourceName := range enabledSources {
		translatedName := km.translateSourceName(sourceName)
		logger.Info().Str("source", sourceName).Str("translated", translatedName).Msg("Processing enabled source for table names")

		// Find the topic configuration for this source
		var sourceTopicConfig *TopicConfig
		for _, topicConfig := range km.topics {
			if topicConfig.Name == translatedName {
				sourceTopicConfig = &topicConfig
				break
			}
		}

		if sourceTopicConfig == nil {
			errMsg := fmt.Sprintf("No topic configuration found for source: %s (translated: %s)", sourceName, translatedName)
			result["success"] = false
			result["errors"] = append(result["errors"].([]string), errMsg)
			logger.Error().Str("source", sourceName).Str("translated", translatedName).Msg("No topic configuration found")
			continue
		}

		// Collect all ClickHouse tables for this source
		sourceTables := sourceTopicConfig.ClickhouseTables
		sourceTableMap[sourceName] = sourceTables
		allTables = append(allTables, sourceTables...)

		logger.Info().Str("source", sourceName).Int("table_count", len(sourceTables)).Msg("Found ClickHouse tables")
	}

	result["results"] = sourceTableMap
	result["all_tables"] = allTables
	result["total_tables"] = len(allTables)

	logger.Info().Int("total_sources", len(enabledSources)).Int("total_tables", len(allTables)).Msg("Completed table name collection for enabled o11y sources")

	return result, nil
}

// TruncateClickHouseTablesForO11ySources truncates ClickHouse tables for enabled o11y sources
func (km *KafkaManager) TruncateClickHouseTablesForO11ySources() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": true,
		"results": make(map[string]string),
		"errors":  make([]string, 0),
		"processed_sources": make([]string, 0),
		"truncated_tables": make([]string, 0),
	}

	// Step 1: Get table names for enabled o11y sources
	tableResult, err := km.GetTableNamesForO11ySources()
	if err != nil {
		result["success"] = false
		result["errors"] = append(result["errors"].([]string), fmt.Sprintf("Failed to get table names: %v", err))
		return result, err
	}

	// Check if table collection was successful
	if !tableResult["success"].(bool) {
		result["success"] = false
		result["errors"] = tableResult["errors"].([]string)
		return result, fmt.Errorf("failed to collect table names")
	}

	sourceTableMap := tableResult["results"].(map[string][]string)
	processedSources := tableResult["processed_sources"].([]string)
	result["processed_sources"] = processedSources

	// Step 2: Truncate each table
	for sourceName, tables := range sourceTableMap {
		for _, tableName := range tables {
			logger.Info().Str("source", sourceName).Str("table", tableName).Msg("Truncating ClickHouse table")

			// Execute truncate command
			truncateCmd := fmt.Sprintf("clickhouse-client --query \"TRUNCATE TABLE vusmart.%s ON CLUSTER vusmart\"", tableName)
			cmd := exec.Command("kubectl", "exec", "chi-clickhouse-vusmart-0-0-0", "-n", "vsmaps", "--", "bash", "-c", truncateCmd)

			output, err := cmd.Output()
			if err != nil {
				errMsg := fmt.Sprintf("Failed to truncate table %s: %v (output: %s)", tableName, err, string(output))
				result["success"] = false
				result["errors"] = append(result["errors"].([]string), errMsg)
				result["results"].(map[string]string)[tableName] = fmt.Sprintf("failed: %v", err)
				logger.Error().Err(err).Str("table", tableName).Msg("Failed to truncate table")
			} else {
				result["results"].(map[string]string)[tableName] = "truncated"
				result["truncated_tables"] = append(result["truncated_tables"].([]string), tableName)
				logger.Info().Str("table", tableName).Msg("Table truncated successfully")
			}
		}
	}

	totalTruncated := len(result["truncated_tables"].([]string))
	totalErrors := len(result["errors"].([]string))

	logger.Info().Int("truncated", totalTruncated).Int("errors", totalErrors).Msg("Completed ClickHouse table truncation")

	return result, nil
}

// GetTopicStatus returns the status of all topics
func (km *KafkaManager) GetTopicStatus() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	topics := make([]map[string]interface{}, 0)

	for _, topicGroup := range km.topics {
		// Check input topics
		for _, inputTopic := range topicGroup.InputTopic {
			status := km.getSingleTopicStatus(inputTopic.Name)
			topics = append(topics, map[string]interface{}{
				"name":   inputTopic.Name,
				"type":   "input",
				"status": status,
			})
		}

		// Check output topics
		for _, outputTopic := range topicGroup.OutputTopic {
			status := km.getSingleTopicStatus(outputTopic.Name)
			topics = append(topics, map[string]interface{}{
				"name":   outputTopic.Name,
				"type":   "output",
				"status": status,
			})
		}
	}

	result["topics"] = topics
	result["total_count"] = len(topics)

	return result, nil
}

// getSingleTopicStatus checks if a single topic exists and its status
func (km *KafkaManager) getSingleTopicStatus(topicName string) string {
	describeCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --describe --topic %s", topicName)
	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

	output, err := cmd.Output()
	if err != nil {
		return "not_found"
	}

	if strings.Contains(string(output), "Topic:") {
		return "exists"
	}

	return "unknown"
}
