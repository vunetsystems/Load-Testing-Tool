package kafka_ch_reset

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
	"vuDataSim/src/logger"
	"crypto/tls"
	"github.com/segmentio/kafka-go"
	"gopkg.in/yaml.v3"
	ch "vuDataSim/src/clickhouse"
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
	Pipeline         []string    `yaml:"pipeline"`
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
	admin      *kafka.Client
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
	"Traces" :  "Traces",	
	"K8s": "Kubernetes",
}

// translateSourceName translates source names between conf.yml and topics_tables.yaml naming conventions
func (km *KafkaManager) translateSourceName(sourceName string) string {
	if translatedName, exists := sourceNameTranslation[sourceName]; exists {
		return translatedName
	}
	// Return original name if no translation found
	return sourceName
}

// getKafkaNodeName retrieves the node name where the Kafka pod is running
func getKafkaNodeName() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pod", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "-o", "jsonpath={.spec.nodeName}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Kafka node name: %v", err)
	}
	nodeName := strings.TrimSpace(string(output))
	if nodeName == "" {
		return "", fmt.Errorf("empty node name returned")
	}
	return nodeName, nil
}

// NewKafkaManager creates a new KafkaManager instance
func NewKafkaManager(configPath string) (*KafkaManager, error) {
	// Get the node name dynamically
	nodeName, err := getKafkaNodeName()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kafka node name: %v", err)
	}

	// Configure TLS to resolve the "broker appears to be expecting TLS" error on port 9094
    tlsConfig := &tls.Config{
        // WARNING: InsecureSkipVerify is ONLY for E2E/testing where broker certs are self-signed.
        // For production, you would configure RootCAs.
        InsecureSkipVerify: true,
    }
	admin := &kafka.Client{
		Addr: kafka.TCP(nodeName + ":9094"),
		Transport: &kafka.Transport{
            TLS: tlsConfig,
        },
	}
	return &KafkaManager{
		configPath: configPath,
		admin:      admin,
	}, nil
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

	logger.Info().Str("topic", topicName).Int("partitions", metadata.PartitionCount).Int("replicationFactor", metadata.ReplicationFactor).Msg("Topic described successfully")
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
	topicMetadatas, err := km.parseBulkTopicDescription(string(output), topics)
	if err != nil {
		return nil, err
	}

	logger.Info().Int("topics", len(topicMetadatas)).Msg("Bulk describe command executed successfully")
	return topicMetadatas, nil
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

// RecreateTopicsForEnabledSourcesUsingClient recreates topics for all enabled o11y sources using kafka-go client
func (km *KafkaManager) RecreateTopicsForEnabledSourcesUsingClient(confPath string) error {
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

	// Batch describe all topics
	topicMetadatas, err := km.DescribeTopicsBulkUsingClient(allTopics)
	if err != nil {
		return fmt.Errorf("failed to describe topics: %v", err)
	}

	// Batch delete all topics
	if err := km.DeleteTopicsBulkUsingClient(allTopics); err != nil {
		return fmt.Errorf("failed to delete topics: %v", err)
	}

	// Wait for topics to be fully deleted before recreating
	if err := km.waitForTopicsDeletion(allTopics); err != nil {
		return fmt.Errorf("failed waiting for topics deletion: %v", err)
	}

	// Batch create all topics
	if err := km.CreateTopicsBulkUsingClient(topicMetadatas); err != nil {
		return fmt.Errorf("failed to create topics: %v", err)
	}

	logger.Info().Int("sources", len(enabledSources)).Int("topics", len(allTopics)).Msg("Successfully recreated topics for enabled o11y sources using client")
	return nil
}

// DescribeTopicsBulkUsingClient describes multiple topics using kafka-go client
func (km *KafkaManager) DescribeTopicsBulkUsingClient(topics []string) (map[string]*TopicMetadata, error) {
	if len(topics) == 0 {
		return make(map[string]*TopicMetadata), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := kafka.MetadataRequest{
		Topics: topics,
	}

	res, err := km.admin.Metadata(ctx, &req)
	if err != nil {
		logger.Error().Err(err).Strs("topics", topics).Msg("Failed to describe topics using client")
		return nil, fmt.Errorf("failed to describe topics: %v", err)
	}

	topicMetadatas := make(map[string]*TopicMetadata)
	for _, topic := range res.Topics {
		if topic.Error != nil {
			logger.Error().Err(topic.Error).Str("topic", topic.Name).Msg("Failed to describe topic")
			return nil, fmt.Errorf("failed to describe topic %s: %v", topic.Name, topic.Error)
		}

		// Get partition count and replication factor from partitions
		if len(topic.Partitions) == 0 {
			return nil, fmt.Errorf("no partitions found for topic %s", topic.Name)
		}

		partitionCount := len(topic.Partitions)
		replicationFactor := len(topic.Partitions[0].Replicas)

		topicMetadatas[topic.Name] = &TopicMetadata{
			TopicName:         topic.Name,
			PartitionCount:    partitionCount,
			ReplicationFactor: replicationFactor,
		}
	}

	// Check if we got metadata for all expected topics
	for _, topic := range topics {
		if _, exists := topicMetadatas[topic]; !exists {
			return nil, fmt.Errorf("failed to get metadata for topic %s", topic)
		}
	}

	logger.Info().Int("topics", len(topicMetadatas)).Msg("Bulk describe command executed successfully using client")
	return topicMetadatas, nil
}

// DeleteTopicsBulkUsingClient deletes multiple topics using kafka-go client
func (km *KafkaManager) DeleteTopicsBulkUsingClient(topics []string) error {
	if len(topics) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := kafka.DeleteTopicsRequest{
		Topics: topics,
	}

	_, err := km.admin.DeleteTopics(ctx, &req)
	if err != nil {
		logger.Warn().Err(err).Strs("topics", topics).Msg("Delete command had some failures")
		// Don't return error for delete failures as topics might not exist
	}

	logger.Info().Int("topics", len(topics)).Msg("Bulk delete command executed using client")
	return nil
}

// CreateTopicsBulkUsingClient creates multiple topics using kafka-go client
func (km *KafkaManager) CreateTopicsBulkUsingClient(topicMetadatas map[string]*TopicMetadata) error {
	if len(topicMetadatas) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var topics []kafka.TopicConfig
	for topicName, meta := range topicMetadatas {
		topics = append(topics, kafka.TopicConfig{
			Topic:             topicName,
			NumPartitions:     meta.PartitionCount,
			ReplicationFactor: meta.ReplicationFactor,
		})
	}

	req := kafka.CreateTopicsRequest{
		Topics: topics,
	}

	res, err := km.admin.CreateTopics(ctx, &req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to execute bulk create command using client")
		return fmt.Errorf("failed to create topics: %v", err)
	}

	// Check for errors in individual topic creation
	for topicName, topicErr := range res.Errors {
		if topicErr != nil {
			logger.Error().Err(topicErr).Str("topic", topicName).Msg("Failed to create topic")
			return fmt.Errorf("failed to create topic %s: %v", topicName, topicErr)
		}
	}

	logger.Info().Int("topics", len(topicMetadatas)).Msg("Bulk create command executed successfully using client")
	return nil
}

// waitForTopicsDeletion waits for topics to be fully deleted before proceeding
func (km *KafkaManager) waitForTopicsDeletion(topics []string) error {
	maxRetries := 30 // Wait up to 30 seconds (30 * 1 second)
	retryInterval := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		allDeleted := true

		for _, topic := range topics {
			// Check if topic still exists
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			req := kafka.MetadataRequest{
				Topics: []string{topic},
			}

			res, err := km.admin.Metadata(ctx, &req)
			cancel()

			if err != nil {
				logger.Warn().Err(err).Str("topic", topic).Msg("Error checking topic existence during deletion wait")
				allDeleted = false
				break
			}

			// If topic exists in metadata, it's not fully deleted
			for _, topicMeta := range res.Topics {
				if topicMeta.Name == topic && topicMeta.Error == nil {
					allDeleted = false
					break
				}
			}

			if !allDeleted {
				break
			}
		}

		if allDeleted {
			logger.Info().Strs("topics", topics).Msg("All topics successfully deleted")
			return nil
		}

		logger.Debug().Int("attempt", i+1).Int("maxRetries", maxRetries).Msg("Waiting for topics to be fully deleted")
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("timeout waiting for topics to be fully deleted after %d seconds", maxRetries)
}

// TruncateTable truncates a single ClickHouse table on cluster vusmart
func (km *KafkaManager) TruncateTable(tableName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Use admin client for truncate operation
	adminClient := ch.GetClickHouseAdminClient()
	if adminClient == nil {
		logger.Warn().Msg("ClickHouse admin client not available, falling back to regular client")
		adminClient = ch.GetClickHouseClient()
		if adminClient == nil {
			return fmt.Errorf("ClickHouse client not initialized")
		}
	}

	// Execute the truncate query using admin client
	query := fmt.Sprintf("TRUNCATE TABLE %s ON CLUSTER vusmart", tableName)
	err := adminClient.Client.Exec(ctx, query)
	if err != nil {
		logger.Error().Err(err).Str("table", tableName).Msg("Failed to truncate ClickHouse table")
		return fmt.Errorf("failed to truncate table %s: %v", tableName, err)
	}

	logger.Info().Str("table", tableName).Msg("Successfully truncated ClickHouse table")
	return nil
}


// GetTablesForEnabledSources returns ClickHouse tables for all enabled o11y sources
func (km *KafkaManager) GetTablesForEnabledSources(confPath string) ([]string, error) {
	// Load o11y config
	o11yConfig, err := km.LoadO11yConfig(confPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load o11y config: %v", err)
	}

	// Get enabled sources
	var enabledSources []string
	for source, config := range o11yConfig.IncludeModuleDirs {
		if config.Enabled {
			enabledSources = append(enabledSources, source)
		}
	}

	if len(enabledSources) == 0 {
		return nil, fmt.Errorf("no enabled o11y sources found")
	}

	// Collect all tables for enabled sources
	var allTables []string
	for _, source := range enabledSources {
		translatedName := km.translateSourceName(source)
		for _, topicGroup := range km.topics {
			if topicGroup.Name == translatedName {
				allTables = append(allTables, topicGroup.ClickhouseTables...)
				break
			}
		}
	}

	if len(allTables) == 0 {
		return nil, fmt.Errorf("no tables found for enabled sources")
	}

	return allTables, nil
}

// TruncateTablesForEnabledSources truncates all ClickHouse tables for enabled o11y sources on cluster vusmart
func (km *KafkaManager) TruncateTablesForEnabledSources(confPath string) error {
	// Get tables for enabled sources
	tables, err := km.GetTablesForEnabledSources(confPath)
	if err != nil {
		return err
	}

	logger.Info().Strs("tables", tables).Msg("Starting truncation of ClickHouse tables")

	// Truncate each table and stop immediately on failure
	for _, table := range tables {
		if err := km.TruncateTable(table); err != nil {
			logger.Error().Err(err).Str("table", table).Msg("Failed to truncate table, stopping entire operation")
			return fmt.Errorf("failed to truncate table %s: %v", table, err)
		}
	}

	logger.Info().Int("tables", len(tables)).Msg("Successfully truncated all ClickHouse tables for enabled o11y sources")
	return nil
}

// SchedulePodDeletion schedules the deletion of pods for given o11y sources after a specified timeout using a bash script and 'at'
func (km *KafkaManager) SchedulePodDeletion(timeoutSeconds int, o11ySources []string) (time.Time, error) {
	// Step 1: Load topics_tables.yaml to get pipeline info (unchanged)
	if err := km.LoadConfig(); err != nil {
		logger.Error().Err(err).Msg("Failed to load topics config")
		return time.Time{}, fmt.Errorf("failed to load topics config: %v", err)
	}

	// Step 2: Translate o11y sources and collect unique pipeline names (unchanged)
	pipelineNames := make(map[string]bool)
	for _, source := range o11ySources {
		translatedName := km.translateSourceName(source)
		for _, topicGroup := range km.topics {
			if topicGroup.Name == translatedName {
				for _, pipeline := range topicGroup.Pipeline {
					pipelineNames[pipeline] = true
				}
				break
			}
		}
	}
	if len(pipelineNames) == 0 {
		return time.Time{}, fmt.Errorf("no pipelines found for provided o11y sources")
	}

	// Calculate the scheduled time
	scheduledTime := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	// Step 3: Generate a bash script for pod deletion
	scriptContent := km.generatePodDeletionScript(pipelineNames)
	scriptPath, err := km.writeTempScript(scriptContent)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create script: %v", err)
	}

	// Step 4: Schedule the script using 'at' command
	if err := km.scheduleWithAt(scriptPath, timeoutSeconds); err != nil {
		os.Remove(scriptPath)  // Clean up on failure
		return time.Time{}, fmt.Errorf("failed to schedule script: %v", err)
	}

	logger.Info().Int("timeout_seconds", timeoutSeconds).Strs("o11y_sources", o11ySources).Str("script_path", scriptPath).Time("scheduled_time", scheduledTime).Msg("Pod deletion script scheduled with 'at'")
	return scheduledTime, nil
}

// generatePodDeletionScript creates the bash script content for retrieving and deleting pods
func (km *KafkaManager) generatePodDeletionScript(pipelineNames map[string]bool) string {
	// Build grep pattern for pod names (e.g., "pipeline1-\|pipeline2-\")
	var patterns []string
	for pipeline := range pipelineNames {
		patterns = append(patterns, fmt.Sprintf("%s-", pipeline))
	}
	grepPattern := strings.Join(patterns, "\\|")

	// Script: Get pods, filter by pattern, delete if any found
	script := fmt.Sprintf(`#!/bin/bash
NAMESPACE="vsmaps"
POD_PATTERN="%s"

# Get pod names matching the pattern
PODS=$(kubectl get pods -n $NAMESPACE --no-headers -o custom-columns=NAME:.metadata.name | grep -E "$POD_PATTERN")

if [ -z "$PODS" ]; then
	   echo "No pods found for deletion"
	   exit 0
fi

# Delete all matching pods in one command
kubectl delete pod $PODS -n $NAMESPACE
echo "Deleted pods: $PODS"
`, grepPattern)

	return script
}

// writeTempScript writes the script to a temporary file and makes it executable
func (km *KafkaManager) writeTempScript(content string) (string, error) {
	tmpDir := "/tmp"  // Or use os.TempDir()
	file, err := os.CreateTemp(tmpDir, "pod_deletion_*.sh")
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return "", err
	}

	scriptPath := file.Name()
	if err := os.Chmod(scriptPath, 0755); err != nil {  // Make executable
		return "", err
	}

	return scriptPath, nil
}

// scheduleWithAt schedules the script using the 'at' command
func (km *KafkaManager) scheduleWithAt(scriptPath string, timeoutSeconds int) error {
	// Calculate future time: current time + timeout
	timeoutSeconds=timeoutSeconds+60 // add buffer to avoid early execution
	futureTime := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	timeStr := futureTime.Format("15:04 2006-01-02")  // 'at' format: HH:MM YYYY-MM-DD

	// Schedule with 'at': echo "/path/to/script.sh" | at <time>
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | at '%s'", scriptPath, timeStr))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to schedule with 'at': %v", err)
	}

	return nil
}

