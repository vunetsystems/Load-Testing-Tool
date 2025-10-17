package kafka_ch_reset

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TopicConfig represents the configuration for a topic group
type TopicConfig struct {
	Name        string      `yaml:"name"`
	InputTopic  []TopicInfo `yaml:"inputTopic"`
	OutputTopic []TopicInfo `yaml:"outputTopic"`
}

// TopicInfo represents individual topic information
type TopicInfo struct {
	Name        string `yaml:"name"`
	TableName   string `yaml:"tableName"`
	TypeField   string `yaml:"typeField"`
	SourceField string `yaml:"sourceField"`
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

// NewKafkaManager creates a new KafkaManager instance
func NewKafkaManager(configPath string) *KafkaManager {
	return &KafkaManager{
		configPath: configPath,
	}
}

// LoadConfig loads the topic configuration from YAML file
func (km *KafkaManager) LoadConfig() error {
	data, err := exec.Command("cat", km.configPath).Output()
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, &km.topics)
	if err != nil {
		return fmt.Errorf("failed to parse YAML config: %v", err)
	}

	return nil
}

// GetAllTopics returns all configured topics
func (km *KafkaManager) GetAllTopics() []TopicConfig {
	return km.topics
}

// RecreateTopics recreates all input and output topics
func (km *KafkaManager) RecreateTopics() (map[string]interface{}, error) {
	result := map[string]interface{}{
		"success": true,
		"results": make(map[string]string),
		"errors":  make([]string, 0),
	}

	// Step 1: Recreate all input topics first
	for _, topicGroup := range km.topics {
		for _, inputTopic := range topicGroup.InputTopic {
			topicResult, err := km.recreateTopic(inputTopic.Name)
			if err != nil {
				result["success"] = false
				result["errors"] = append(result["errors"].([]string), fmt.Sprintf("Failed to recreate input topic %s: %v", inputTopic.Name, err))
			} else {
				result["results"].(map[string]string)[inputTopic.Name] = topicResult
			}
		}
	}

	// Step 2: Recreate all output topics
	for _, topicGroup := range km.topics {
		for _, outputTopic := range topicGroup.OutputTopic {
			topicResult, err := km.recreateTopic(outputTopic.Name)
			if err != nil {
				result["success"] = false
				result["errors"] = append(result["errors"].([]string), fmt.Sprintf("Failed to recreate output topic %s: %v", outputTopic.Name, err))
			} else {
				result["results"].(map[string]string)[outputTopic.Name] = topicResult
			}
		}
	}

	return result, nil
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

// recreateTopic handles the recreation of a single topic
func (km *KafkaManager) recreateTopic(topicName string) (string, error) {
	// For now, let's simulate the process with individual commands
	// In a real implementation, you'd need to handle the interactive session properly

	// Step 2: Describe the topic to get metadata
	describeCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --describe --topic %s", topicName)

	// Execute kubectl exec with the describe command
	cmd := exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", describeCmd)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to describe topic %s: %v", topicName, err)
	}

	// Parse the output to extract partition count and replication factor
	metadata, err := km.parseTopicDescription(string(output))
	if err != nil {
		return "", fmt.Errorf("failed to parse topic description for %s: %v", topicName, err)
	}

	// Step 3: Delete the topic
	deleteCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --delete --topic %s", topicName)
	cmd = exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", deleteCmd)

	_, err = cmd.Output()
	if err != nil {
		// Note: Delete might fail if topic doesn't exist, which is okay
		fmt.Printf("Warning: Failed to delete topic %s (might not exist): %v\n", topicName, err)
	}

	// Wait a moment for deletion to complete
	time.Sleep(2 * time.Second)

	// Step 4: Recreate the topic with stored metadata
	createCmd := fmt.Sprintf("kafka-topics --bootstrap-server localhost:9092 --create --topic %s --partitions %d --replication-factor %d",
		topicName, metadata.PartitionCount, metadata.ReplicationFactor)

	cmd = exec.Command("kubectl", "exec", "kafka-cluster-cp-kafka-0", "-n", "vsmaps", "--", "bash", "-c", createCmd)

	_, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create topic %s: %v", topicName, err)
	}

	return fmt.Sprintf("Successfully recreated topic %s with %d partitions and replication factor %d",
		topicName, metadata.PartitionCount, metadata.ReplicationFactor), nil
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