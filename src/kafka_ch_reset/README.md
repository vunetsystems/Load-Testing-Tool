# Kafka and ClickHouse Reset Functionality

This module provides functionality to recreate Kafka topics and truncate ClickHouse tables for the vuDataSim load testing tool.

## Features

- **Kafka Topic Recreation**: Recreates Kafka topics with their original partition count and replication factor
- **ClickHouse Table Truncation**: Placeholder for truncating ClickHouse tables (to be implemented)
- **Configuration Management**: YAML-based configuration for topics and tables
- **REST API**: Full REST API for managing operations

## API Endpoints

### Get All Topics Configuration
```bash
curl -X GET http://localhost:8086/api/kafka/topics
```

### Recreate All Topics
```bash
curl -X POST http://localhost:8086/api/kafka/recreate
```

### Get Topic Status
```bash
curl -X GET http://localhost:8086/api/kafka/status
```

### Describe a Single Topic
```bash
curl -X GET http://localhost:8086/api/kafka/describe/linux-monitor-input
```

### Delete a Single Topic
```bash
curl -X DELETE http://localhost:8086/api/kafka/delete/linux-monitor-input
```

### Create a New Topic
```bash
curl -X POST http://localhost:8086/api/kafka/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-topic",
    "partitionCount": 3,
    "replicationFactor": 2
  }'
```

### Truncate ClickHouse Tables (Placeholder)
```bash
curl -X POST http://localhost:8086/api/clickhouse/truncate
```

## Configuration

The configuration is stored in `src/kafka_ch_reset/config.yaml`:

```yaml
kafka_topics:
  - name: "linuxmonitor"
    inputTopic:
      - name: "linux-monitor-input"
    outputTopic:
      - name: "linux-monitor-additional-metrics"
        tableName: "vmetrics_linux_monitor_additional_metrics"
        typeField: "type"
        sourceField: "target"
      - name: "linux-monitor-process-metrics"
        tableName: "vmetrics_linux_monitor_process_metrics"
        typeField: "type"
        sourceField: "target"
      - name: "linux-monitor-resource-metrics"
        tableName: "vmetrics_linux_monitor_resource_metrics"
        typeField: "type"
        sourceField: "target"
      - name: "linux-monitor-storage-metrics"
        tableName: "vmetrics_linux_monitor_storage_metrics"
        typeField: "type"
        sourceField: "target"
```

## How It Works

### Individual Topic Operations

The system now provides granular control over Kafka topic operations:

#### 1. Describe Topic
```bash
kubectl exec kafka-cluster-cp-kafka-0 -n vsmaps -- bash -c "kafka-topics --bootstrap-server localhost:9092 --describe --topic <topic-name>"
```
Extracts and returns topic metadata including partition count and replication factor.

#### 2. Delete Topic
```bash
kubectl exec kafka-cluster-cp-kafka-0 -n vsmaps -- bash -c "kafka-topics --bootstrap-server localhost:9092 --delete --topic <topic-name>"
```
Removes an existing topic from the Kafka cluster.

#### 3. Create Topic
```bash
kubectl exec kafka-cluster-cp-kafka-0 -n vsmaps -- bash -c "kafka-topics --bootstrap-server localhost:9092 --create --topic <topic-name> --partitions <count> --replication-factor <factor>"
```
Creates a new topic with specified configuration.

### Automated Topic Recreation Process

For bulk operations, the system follows this process:

1. **Describe Topics**: For each topic, extract metadata (partition count, replication factor)
2. **Extract Metadata**: Parse the Kafka response to get configuration details
3. **Delete Topics**: Remove existing topics in reverse dependency order
4. **Recreate Topics**: Create topics with original settings in proper order

### Topic Recreation Order

- **Input topics** are recreated first
- **Output topics** are recreated second
- This ensures proper data flow dependencies are maintained

## Testing the Functionality

### 1. Check Current Topics
```bash
curl -X GET http://localhost:8086/api/kafka/topics
```

Expected response:
```json
{
  "success": true,
  "message": "Retrieved 3 topic groups",
  "data": [
    {
      "name": "linuxmonitor",
      "inputTopic": [
        {
          "name": "linux-monitor-input"
        }
      ],
      "outputTopic": [
        {
          "name": "linux-monitor-additional-metrics",
          "tableName": "vmetrics_linux_monitor_additional_metrics",
          "typeField": "type",
          "sourceField": "target"
        }
      ]
    }
  ]
}
```

### 2. Check Topic Status
```bash
curl -X GET http://localhost:8086/api/kafka/status
```

### 3. Recreate All Topics
```bash
curl -X POST http://localhost:8086/api/kafka/recreate
```

Expected response:
```json
{
  "success": true,
  "message": "All topics recreated successfully",
  "data": {
    "success": true,
    "results": {
      "linux-monitor-input": "Successfully recreated topic linux-monitor-input with 1 partitions and replication factor 2",
      "linux-monitor-additional-metrics": "Successfully recreated topic linux-monitor-additional-metrics with 1 partitions and replication factor 2"
    },
    "errors": []
  }
}
```

### 4. Describe a Single Topic
```bash
curl -X GET http://localhost:8086/api/kafka/describe/linux-monitor-input
```

Expected response:
```json
{
  "success": true,
  "message": "Topic linux-monitor-input described successfully",
  "data": {
    "topicName": "linux-monitor-input",
    "partitionCount": 1,
    "replicationFactor": 2
  }
}
```

### 5. Delete a Single Topic
```bash
curl -X DELETE http://localhost:8086/api/kafka/delete/linux-monitor-input
```

Expected response:
```json
{
  "success": true,
  "message": "Topic linux-monitor-input deleted successfully"
}
```

### 6. Create a New Topic
```bash
curl -X POST http://localhost:8086/api/kafka/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-topic",
    "partitionCount": 3,
    "replicationFactor": 2
  }'
```

Expected response:
```json
{
  "success": true,
  "message": "Topic test-topic created successfully with 3 partitions and replication factor 2",
  "data": {
    "name": "test-topic",
    "partitionCount": 3,
    "replicationFactor": 2
  }
}
```

### 4. Test ClickHouse Truncation (Placeholder)
```bash
curl -X POST http://localhost:8086/api/clickhouse/truncate
```

### 5. Recreate Topics for Specific O11y Sources
```bash
curl -X POST http://localhost:8086/api/kafka/recreate/o11y \
  -H "Content-Type: application/json" \
  -d '{
    "o11ySources": ["MongoDB", "LinuxMonitor"]
  }'
```

Expected response:
```json
{
  "success": true,
  "message": "Topics recreated successfully for specified o11y sources",
  "data": {
    "success": true,
    "results": {
      "mongo-metrics-input": "Successfully recreated topic mongo-metrics-input with 3 partitions and replication factor 1",
      "mongo-metrics": "Successfully recreated topic mongo-metrics with 3 partitions and replication factor 1",
      "linux-monitor-input": "Successfully recreated topic linux-monitor-input with 3 partitions and replication factor 1",
      "linux-monitor-resource-metrics": "Successfully recreated topic linux-monitor-resource-metrics with 3 partitions and replication factor 1"
    },
    "errors": []
  }
}
```

#### Test with Single O11y Source
```bash
curl -X POST http://localhost:8086/api/kafka/recreate/o11y \
  -H "Content-Type: application/json" \
  -d '{
    "o11ySources": ["Apache"]
  }'
```

#### Test Error Cases
```bash
# Empty sources list
curl -X POST http://localhost:8086/api/kafka/recreate/o11y \
  -H "Content-Type: application/json" \
  -d '{
    "o11ySources": []
  }'

# Invalid JSON
curl -X POST http://localhost:8086/api/kafka/recreate/o11y \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

## Implementation Notes

- **Error Handling**: The system gracefully handles errors and continues processing other topics
- **Logging**: All operations are logged using the existing logger system
- **In-Memory Storage**: Partition and replication factors are stored in memory during the recreation process
- **Partial Success**: The system reports both successful operations and errors
- **ClickHouse Integration**: Currently a placeholder - needs actual ClickHouse client implementation

## O11y Source Name Translation

The system automatically translates o11y source names between different configuration files:

| conf.yml Name | topics_tables.yaml Name | Notes |
|---------------|------------------------|-------|
| `LinuxMonitor` | `Linux Monitor` | Space added |
| `MongoDB` | `MongoDB` | Same name |
| `Mssql` | `MSSQL` | Case change |
| `Apache` | `Apache` | Same name |

This translation ensures compatibility between the main configuration and topic definitions.

## Future Enhancements

- Implement actual ClickHouse table truncation logic
- Add support for custom partition/replication factor overrides
- Add topic validation before recreation
- Implement dry-run mode for testing
- Add backup/restore functionality for topic configurations