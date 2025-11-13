// Package kafka_processors implements Kafka topic specifications processing
// This processor captures partition and replication factor details for topics
// used in test runs and stores them in the database as JSON.
package kafka_processors

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vuDataSim/src/kafka_ch_reset"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// TopicSpec represents the specification for a single topic
type TopicSpec struct {
	Name             string `json:"name"`
	Partitions       int    `json:"partitions"`
	ReplicationFactor int   `json:"replication_factor"`
}

// KafkaSpecs represents the complete Kafka specifications for a test run
type KafkaSpecs struct {
	InputTopics  []TopicSpec `json:"input_topics"`
	OutputTopics []TopicSpec `json:"output_topics"`
}

// SourcesConfig represents the wrapper structure for sources from topics_tables.yaml
type SourcesConfig struct {
	Sources []kafka_ch_reset.TopicConfig `yaml:"sources"`
}

// ProcessKafkaSpecs processes Kafka topic specifications for a test run
// It collects topic names from o11y_sources, describes them using Kafka client,
// and stores the partition/replication specs in the kafka_specs column
func ProcessKafkaSpecs(db *sql.DB, testID string, o11ySources []string) error {
	logger.LogWithNode("System", "KafkaSpecsProcessor", fmt.Sprintf("Processing Kafka specs for test %s", testID), "info")

	// Load topics_tables.yaml configuration
	yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
	if err != nil {
		return fmt.Errorf("failed to read topics_tables.yaml: %w", err)
	}

	var cfg SourcesConfig
	err = yaml.Unmarshal(yamlData, &cfg)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Collect all input and output topics for the specified o11y sources
	var inputTopics []string
	var outputTopics []string

	for _, o11ySource := range o11ySources {
		// Normalize o11y source name: lowercase and remove spaces
		normalizedO11ySource := strings.ToLower(strings.ReplaceAll(o11ySource, " ", ""))

		for _, source := range cfg.Sources {
			// Normalize source name: lowercase and remove spaces
			normalizedSourceName := strings.ToLower(strings.ReplaceAll(source.Name, " ", ""))

			if normalizedSourceName == normalizedO11ySource {
				// Collect input topics
				for _, inputTopic := range source.InputTopic {
					inputTopics = append(inputTopics, inputTopic.Name)
				}
				// Collect output topics
				for _, outputTopic := range source.OutputTopic {
					outputTopics = append(outputTopics, outputTopic.Name)
				}
				break
			}
		}
	}

	if len(inputTopics) == 0 && len(outputTopics) == 0 {
		logger.LogWarning("System", "KafkaSpecsProcessor", fmt.Sprintf("No topics found for o11y sources: %v", o11ySources))
		// Store empty specs instead of failing
		emptySpecs := KafkaSpecs{
			InputTopics:  []TopicSpec{},
			OutputTopics: []TopicSpec{},
		}
		return storeKafkaSpecs(db, testID, emptySpecs)
	}

	// Initialize Kafka manager to get topic metadata
	kafkaManager, err := kafka_ch_reset.NewKafkaManager("src/configs/topics_tables.yaml")
	if err != nil {
		return fmt.Errorf("failed to create Kafka manager: %w", err)
	}

	// Load configuration
	if err := kafkaManager.LoadConfig(); err != nil {
		logger.LogWarning("System", "KafkaSpecsProcessor", fmt.Sprintf("Failed to load Kafka config: %v", err))
		// Continue with empty specs
		emptySpecs := KafkaSpecs{
			InputTopics:  []TopicSpec{},
			OutputTopics: []TopicSpec{},
		}
		return storeKafkaSpecs(db, testID, emptySpecs)
	}

	// Describe all topics using the Kafka client
	allTopics := append(inputTopics, outputTopics...)
	topicMetadatas, err := kafkaManager.DescribeTopicsBulkUsingClient(allTopics)
	if err != nil {
		logger.LogWarning("System", "KafkaSpecsProcessor", fmt.Sprintf("Failed to describe topics: %v", err))
		// Continue with empty specs
		emptySpecs := KafkaSpecs{
			InputTopics:  []TopicSpec{},
			OutputTopics: []TopicSpec{},
		}
		return storeKafkaSpecs(db, testID, emptySpecs)
	}

	// Build the specs structure
	var inputSpecs []TopicSpec
	var outputSpecs []TopicSpec

	// Process input topics
	for _, topicName := range inputTopics {
		if meta, exists := topicMetadatas[topicName]; exists {
			inputSpecs = append(inputSpecs, TopicSpec{
				Name:              topicName,
				Partitions:        meta.PartitionCount,
				ReplicationFactor: meta.ReplicationFactor,
			})
		} else {
			logger.LogWarning("System", "KafkaSpecsProcessor", fmt.Sprintf("Metadata not found for input topic: %s", topicName))
		}
	}

	// Process output topics
	for _, topicName := range outputTopics {
		if meta, exists := topicMetadatas[topicName]; exists {
			outputSpecs = append(outputSpecs, TopicSpec{
				Name:              topicName,
				Partitions:        meta.PartitionCount,
				ReplicationFactor: meta.ReplicationFactor,
			})
		} else {
			logger.LogWarning("System", "KafkaSpecsProcessor", fmt.Sprintf("Metadata not found for output topic: %s", topicName))
		}
	}

	specs := KafkaSpecs{
		InputTopics:  inputSpecs,
		OutputTopics: outputSpecs,
	}

	// Store in database
	return storeKafkaSpecs(db, testID, specs)
}

// storeKafkaSpecs stores the Kafka specifications in the database
func storeKafkaSpecs(db *sql.DB, testID string, specs KafkaSpecs) error {
	specsJSON, err := json.Marshal(specs)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka specs to JSON: %w", err)
	}

	_, err = db.Exec(`
		UPDATE test_runs
		SET kafka_specs = ?
		WHERE test_id = ?
	`, string(specsJSON), testID)

	if err != nil {
		return fmt.Errorf("failed to update test run with Kafka specs: %w", err)
	}

	logger.LogSuccess("System", "KafkaSpecsProcessor", fmt.Sprintf("Successfully stored Kafka specs for test %s", testID))
	return nil
}