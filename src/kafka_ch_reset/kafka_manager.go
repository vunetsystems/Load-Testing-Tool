package kafka_ch_reset

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	describeCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --describe --topic %s", topicName)
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --delete --topic %s", topicName)
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", deleteCmd)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --create --topic %s --partitions %d --replication-factor %d",
		topicName, partitionCount, replicationFactor)

	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", createCmd)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	describeCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --describe --topic %s", topicName)
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

	output, err := cmd.Output()
	if err != nil {
		return "not_found"
	}

	if strings.Contains(string(output), "Topic:") {
		return "exists"
	}

	return "unknown"
}

// DescribeTopicsBulk describes multiple topics in a single kubectl exec call
func (km *KafkaManager) DescribeTopicsBulk(topics []string) (map[string]*TopicMetadata, error) {
	if len(topics) == 0 {
		return make(map[string]*TopicMetadata), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build a single command that describes all topics
	topicList := strings.Join(topics, ",")
	describeCmd := fmt.Sprintf("for topic in %s; do echo '===TOPIC_START==='; kafka-topics --bootstrap-server localhost:9092 --describe --topic $topic; echo '===TOPIC_END==='; done", topicList)

	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Str("topics", topicList).Msg("Failed to execute bulk describe command")
		return nil, fmt.Errorf("failed to describe topics: %v", err)
	}

	// Parse the output to extract metadata for each topic
	return km.parseBulkTopicDescription(string(output), topics)
}

// DeleteTopicsBulk deletes multiple topics in a single kubectl exec call
func (km *KafkaManager) DeleteTopicsBulk(topics []string) error {
	if len(topics) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build a single command that deletes all topics
	topicList := strings.Join(topics, ",")
	deleteCmd := fmt.Sprintf("for topic in %s; do kafka-topics --bootstrap-server localhost:9092 --delete --topic $topic; done", topicList)

	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", deleteCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Warn().Err(err).Str("topics", topicList).Str("output", string(output)).Msg("Bulk delete command had some failures")
		// Don't return error for delete failures as topics might not exist
	}

	logger.Info().Int("topics", len(topics)).Msg("Bulk delete command executed")
	return nil
}

// CreateTopicsBulk creates multiple topics in a single kubectl exec call
func (km *KafkaManager) CreateTopicsBulk(topicMetadatas map[string]*TopicMetadata) error {
	if len(topicMetadatas) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build a single command that creates all topics
	var createCommands []string
	for topicName, meta := range topicMetadatas {
		createCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --create --topic %s --partitions %d --replication-factor %d",
			topicName, meta.PartitionCount, meta.ReplicationFactor)
		createCommands = append(createCommands, createCmd)
	}

	fullCmd := strings.Join(createCommands, " && ")
	cmd := exec.CommandContext(ctx, "kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", fullCmd)

	output, err := cmd.Output()
	if err != nil {
		logger.Error().Err(err).Str("output", string(output)).Msg("Failed to execute bulk create command")
		return fmt.Errorf("failed to create topics: %v", err)
	}

	logger.Info().Int("topics", len(topicMetadatas)).Msg("Bulk create command executed successfully")
	return nil
}

// parseBulkTopicDescription parses the output of bulk describe command
func (km *KafkaManager) parseBulkTopicDescription(output string, expectedTopics []string) (map[string]*TopicMetadata, error) {
	topicMetadatas := make(map[string]*TopicMetadata)

	// Split output by topic separators
	sections := strings.Split(output, "===TOPIC_START===")
	for _, section := range sections[1:] { // Skip first empty section
		endIndex := strings.Index(section, "===TOPIC_END===")
		if endIndex == -1 {
			continue
		}
		topicOutput := section[:endIndex]

		// Parse individual topic description
		meta, err := km.parseTopicDescription(topicOutput)
		if err != nil {
			logger.Warn().Err(err).Str("output", topicOutput).Msg("Failed to parse topic description in bulk output")
			continue
		}

		if meta.TopicName != "" {
			topicMetadatas[meta.TopicName] = meta
		}
	}

	// Check if we got metadata for all expected topics
	for _, topic := range expectedTopics {
		if _, exists := topicMetadatas[topic]; !exists {
			return nil, fmt.Errorf("failed to get metadata for topic %s", topic)
		}
	}

	return topicMetadatas, nil
}

// RecreateTopicsForEnabledSources recreates topics for all enabled o11y sources from conf.yml
func (km *KafkaManager) RecreateTopicsForEnabledSources(confPath string) error {
	// Load o11y config
	o11yConfig, err := km.LoadO11yConfig(confPath)
	if err != nil {
		return fmt.Errorf("failed to load o11y config: %v", err)
	}

	// Get enabled sources
	var enabledSources []string
	for source, config := range o11yConfig.IncludeModuleDirs {
		if config.Enabled {
			enabledSources = append(enabledSources, source)
		}
	}

	if len(enabledSources) == 0 {
		return fmt.Errorf("no enabled o11y sources found")
	}

	// Collect all topics for enabled sources
	var allTopics []string
	sourceTopics := make(map[string][]string)
	for _, source := range enabledSources {
		translatedName := km.translateSourceName(source)
		for _, topicGroup := range km.topics {
			if topicGroup.Name == translatedName {
				for _, inputTopic := range topicGroup.InputTopic {
					allTopics = append(allTopics, inputTopic.Name)
					sourceTopics[source] = append(sourceTopics[source], inputTopic.Name)
				}
				for _, outputTopic := range topicGroup.OutputTopic {
					allTopics = append(allTopics, outputTopic.Name)
					sourceTopics[source] = append(sourceTopics[source], outputTopic.Name)
				}
				break
			}
		}
	}

	if len(allTopics) == 0 {
		return fmt.Errorf("no topics found for enabled sources")
	}

	// Describe all topics in bulk
	topicMetadatas, err := km.DescribeTopicsBulk(allTopics)
	if err != nil {
		return fmt.Errorf("failed to describe topics: %v", err)
	}

	// Delete all topics in bulk
	if err := km.DeleteTopicsBulk(allTopics); err != nil {
		return fmt.Errorf("failed to delete topics: %v", err)
	}

	// Create all topics in bulk
	if err := km.CreateTopicsBulk(topicMetadatas); err != nil {
		return fmt.Errorf("failed to create topics: %v", err)
	}

	logger.Info().Int("sources", len(enabledSources)).Int("topics", len(allTopics)).Msg("Successfully recreated topics for enabled o11y sources")
	return nil
}
