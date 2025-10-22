package kafka_ch_reset

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	fmt.Printf("Loading config from: %s\n", km.configPath)
	data, err := exec.Command("cat", km.configPath).Output()
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config SourcesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse YAML config: %v", err)
	}

	fmt.Printf("Loaded %d topic configurations\n", len(config.Sources))
	for i, source := range config.Sources {
		fmt.Printf("Source %d: %s\n", i, source.Name)
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
		return nil, fmt.Errorf("failed to describe topic %s: %v", topicName, err)
	}

	metadata, err := km.parseTopicDescription(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse topic description for %s: %v", topicName, err)
	}

	return metadata, nil
}

// DeleteTopic deletes a single topic
func (km *KafkaManager) DeleteTopic(topicName string) error {
	deleteCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --delete --topic %s", topicName)
	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", deleteCmd)

	_, err := cmd.Output()
	if err != nil {
		// Note: Delete might fail if topic doesn't exist, which is okay for some use cases
		return fmt.Errorf("failed to delete topic %s: %v", topicName, err)
	}

	return nil
}

// CreateTopic creates a single topic with specified metadata
func (km *KafkaManager) CreateTopic(topicName string, partitionCount, replicationFactor int) error {
	createCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --create --topic %s --partitions %d --replication-factor %d",
		topicName, partitionCount, replicationFactor)

	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", createCmd)

	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %v", topicName, err)
	}

	return nil
}

// LoadO11yConfig loads the o11y source configuration from conf.yml file
func (km *KafkaManager) LoadO11yConfig(confPath string) (*O11ySourceConfig, error) {
	data, err := exec.Command("cat", confPath).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read o11y config file: %v", err)
	}

	var config O11ySourceConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML o11y config: %v", err)
	}

	return &config, nil
}

// recreateSingleTopic recreates a single topic by describing, deleting, and creating it
func (km *KafkaManager) recreateSingleTopic(topicName string) (string, error) {
	// Step 1: Describe the topic to get its metadata
	metadata, err := km.DescribeTopic(topicName)
	if err != nil {
		// If topic doesn't exist, we'll create it with default settings
		logger.Info().Str("topic", topicName).Msg("Topic does not exist, will create with default settings")
	} else {
		logger.Info().Str("topic", topicName).
			Int("partitions", metadata.PartitionCount).
			Int("replicationFactor", metadata.ReplicationFactor).
			Msg("Found existing topic metadata")
	}

	// Step 2: Delete the topic (ignore errors if topic doesn't exist)
	err = km.DeleteTopic(topicName)
	if err != nil {
		logger.Warn().Err(err).Str("topic", topicName).Msg("Failed to delete topic (may not exist)")
	} else {
		logger.Info().Str("topic", topicName).Msg("Topic deleted successfully")
	}

	// Step 3: Create the topic with the same or default settings
	partitionCount := 1
	replicationFactor := 1

	if metadata != nil {
		partitionCount = metadata.PartitionCount
		replicationFactor = metadata.ReplicationFactor
	}

	err = km.CreateTopic(topicName, partitionCount, replicationFactor)
	if err != nil {
		return "", fmt.Errorf("failed to create topic %s: %v", topicName, err)
	}

	logger.Info().Str("topic", topicName).
		Int("partitions", partitionCount).
		Int("replicationFactor", replicationFactor).
		Msg("Topic created successfully")

	return "recreated", nil
}

// RecreateTopicsForO11ySources recreates topics for enabled o11y sources from conf.yml using parallel processing
func (km *KafkaManager) RecreateTopicsForO11ySources() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": true,
		"results": make(map[string]string),
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
			result["success"] = false
			result["errors"] = append(result["errors"].([]string), errMsg)
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

	// Step 4: Process all topics in parallel using goroutines
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Channel to collect errors from goroutines
	errorChan := make(chan string, len(allTopics))

	for _, topicName := range allTopics {
		wg.Add(1)
		go func(topic string) {
			defer wg.Done()

			topicResult, err := km.recreateSingleTopic(topic)
			mu.Lock()
			if err != nil {
				result["success"] = false
				errorMsg := fmt.Sprintf("Failed to recreate topic %s: %v", topic, err)
				result["errors"] = append(result["errors"].([]string), errorMsg)
				errorChan <- errorMsg
			} else {
				result["results"].(map[string]string)[topic] = topicResult
			}
			mu.Unlock()
		}(topicName)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)

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
