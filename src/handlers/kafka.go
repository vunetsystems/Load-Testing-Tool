package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"vuDataSim/src/kafka_ch_reset"
	"vuDataSim/src/logger"
	"github.com/gorilla/mux"
)


// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// KafkaHandler handles Kafka-related API endpoints
type KafkaHandler struct {
	kafkaManager *kafka_ch_reset.KafkaManager
}

// NewKafkaHandler creates a new KafkaHandler instance
func NewKafkaHandler() (*KafkaHandler, error) {
	configPath := filepath.Join("src", "configs", "topics_tables.yaml")
	kafkaManager, err := kafka_ch_reset.NewKafkaManager(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka manager: %v", err)
	}

	// Load configuration - this is required for topic operations
	if err := kafkaManager.LoadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load Kafka topic configuration: %v", err)
	}

	return &KafkaHandler{
		kafkaManager: kafkaManager,
	}, nil
}

// GetTopics handles GET /api/kafka/topics - returns all configured topics
func (kh *KafkaHandler) GetTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	topics := kh.kafkaManager.GetAllTopics()

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d topic groups", len(topics)),
		Data:    topics,
	})
}


// GetTopicStatus handles GET /api/kafka/status - returns status of all topics
func (kh *KafkaHandler) GetTopicStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	status, err := kh.kafkaManager.GetTopicStatus()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get topic status")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get topic status: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Topic status retrieved successfully",
		Data:    status,
	})
}

// DescribeTopic handles GET /api/kafka/describe/{topic} - describes a single topic
func (kh *KafkaHandler) DescribeTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Extract topic name from URL path
	vars := mux.Vars(r)
	topicName := vars["topic"]

	if topicName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Topic name is required",
		})
		return
	}

	metadata, err := kh.kafkaManager.DescribeTopic(topicName)
	if err != nil {
		logger.Error().Err(err).Str("topic", topicName).Msg("Failed to describe topic")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to describe topic %s: %v", topicName, err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Topic %s described successfully", topicName),
		Data:    metadata,
	})
}

// DeleteTopic handles DELETE /api/kafka/delete/{topic} - deletes a single topic
func (kh *KafkaHandler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use DELETE.",
		})
		return
	}

	// Extract topic name from URL path
	vars := mux.Vars(r)
	topicName := vars["topic"]

	if topicName == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Topic name is required",
		})
		return
	}

	err := kh.kafkaManager.DeleteTopic(topicName)
	if err != nil {
		logger.Error().Err(err).Str("topic", topicName).Msg("Failed to delete topic")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete topic %s: %v", topicName, err),
		})
		return
	}

	logger.Info().Str("topic", topicName).Msg("Topic deleted successfully")
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Topic %s deleted successfully", topicName),
	})
}

// CreateTopic handles POST /api/kafka/create - creates a new topic
func (kh *KafkaHandler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	var requestData struct {
		Name             string `json:"name"`
		PartitionCount   int    `json:"partitionCount"`
		ReplicationFactor int   `json:"replicationFactor"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	if requestData.Name == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Topic name is required",
		})
		return
	}

	if requestData.PartitionCount <= 0 {
		requestData.PartitionCount = 1 // Default to 1 partition
	}

	if requestData.ReplicationFactor <= 0 {
		requestData.ReplicationFactor = 1 // Default to 1 replication factor
	}

	err := kh.kafkaManager.CreateTopic(requestData.Name, requestData.PartitionCount, requestData.ReplicationFactor)
	if err != nil {
		logger.Error().Err(err).Str("topic", requestData.Name).Msg("Failed to create topic")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create topic %s: %v", requestData.Name, err),
		})
		return
	}

	logger.Info().Str("topic", requestData.Name).
		Int("partitions", requestData.PartitionCount).
		Int("replicationFactor", requestData.ReplicationFactor).
		Msg("Topic created successfully")

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Topic %s created successfully with %d partitions and replication factor %d",
			requestData.Name, requestData.PartitionCount, requestData.ReplicationFactor),
		Data: map[string]interface{}{
			"name":              requestData.Name,
			"partitionCount":    requestData.PartitionCount,
			"replicationFactor": requestData.ReplicationFactor,
		},
	})
}




// RecreateEnabledTopics handles POST /api/kafka/recreate-enabled - recreates topics for enabled o11y sources
func (kh *KafkaHandler) RecreateEnabledTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	confPath := "src/migrate/conf.d/conf.yml"
	err := kh.kafkaManager.RecreateTopicsForEnabledSources(confPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to recreate topics for enabled sources")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to recreate topics: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Topics recreated successfully for all enabled o11y sources",
	})
}

// RecreateEnabledTopicsUsingClient handles POST /api/kafka/recreate-enabled-client - recreates topics for enabled o11y sources using kafka client
func (kh *KafkaHandler) RecreateEnabledTopicsUsingClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	confPath := "src/migrate/conf.d/conf.yml"
	err := kh.kafkaManager.RecreateTopicsForEnabledSourcesUsingClient(confPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to recreate topics for enabled sources using client")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to recreate topics: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Topics recreated successfully for all enabled o11y sources using client",
	})
}

// TruncateEnabledTables handles POST /api/kafka/truncate-enabled-tables - truncates ClickHouse tables for enabled o11y sources
func (kh *KafkaHandler) TruncateEnabledTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed. Use POST.",
		})
		return
	}

	confPath := "src/migrate/conf.d/conf.yml"
	err := kh.kafkaManager.TruncateTablesForEnabledSources(confPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to truncate tables for enabled sources")
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to truncate tables: %v", err),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "ClickHouse tables truncated successfully for all enabled o11y sources",
	})
}

// getAllTableNames extracts all table names from the configuration
func getAllTableNames(km *kafka_ch_reset.KafkaManager) []string {
	tableNames := make([]string, 0)

	for _, topicGroup := range km.GetAllTopics() {
		tableNames = append(tableNames, topicGroup.ClickhouseTables...)
	}

	return tableNames
}