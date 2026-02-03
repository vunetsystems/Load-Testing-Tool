# Product Requirements Document (PRD): Kafka Message Generator

## 1. Overview

### 1.1 Purpose
A Go-based Kafka message generator that produces configurable YONO audit messages (`yono_adt_error` and `yono_adt_trans`) to Kafka topics. The generator must support high concurrency, precise throughput control, and comprehensive configuration via YAML.

### 1.2 Scope
- Generate two types of JSON messages: error audit and transaction audit
- Support configurable message distribution and field values
- Produce messages to Kafka with controlled throughput (Events Per Second - EPS)
- Enable duration-based or count-based execution
- Implement efficient concurrency using goroutines with proper resource management
- Provide comprehensive Kafka producer configuration

---

## 2. Functional Requirements

### 2.1 Message Structure

#### 2.1.1 Top-Level Message Format
```json
{
  "yono_adt_error": "<JSON_STRING>",  // Optional
  "yono_adt_trans": "<JSON_STRING>"   // Optional, at least one must be present
}
```

#### 2.1.2 yono_adt_error Schema (Stringified JSON)
```json
{
  "msgId": null,
  "trnsId": "string",
  "sessnTknId": "string",
  "errCd": "string",
  "usrId": 1212301889885902848,
  "errType": "string",
  "errDscrptn": "string",  // Stringified JSON, e.g., "{\"TEACS206\":{\"message\":\"Input provided is invalid\"}}"
  "errDtls": "string",
  "errTime": 1768397920085,
  "crtdBy": "string",
  "crtdOn": 1768397920085
}
```

**Field Requirements:**
- `msgId`: Always `null`
- `trnsId`: Auto-generated unique transaction ID
- `sessnTknId`: Session token, should rotate periodically (configurable interval)
- `errCd`: Configurable error code from predefined list
- `usrId`: Configurable user ID (can be random range or fixed list)
- `errType`: Configurable (e.g., "BUSINESS_EXCEPTION")
- `errDscrptn`: Stringified JSON with configurable error messages
- `errDtls`: Configurable string
- `errTime`: Current epoch milliseconds at generation time
- `crtdBy`: Configurable string (e.g., "System")
- `crtdOn`: Current epoch milliseconds at generation time

#### 2.1.3 yono_adt_trans Schema (Stringified JSON)
```json
{
  "msgId": null,
  "trnsId": "string",
  "sessnTknId": "string",
  "clntIpAddress": null,
  "reqNo": "string",
  "usrRltnshpNo": "string",
  "trnsStts": "string",
  "strtTime": 1768397919979,
  "endTime": 1768397920085,
  "bizReqInput": "string",   // Base64 encoded
  "bizRegInput": "string",   // Base64 encoded (optional)
  "bizRespOutput": "string", // Base64 encoded
  "usrId": 1212301889885902848,
  "usrTyp": "string",
  "cmndId": "string",
  "crtdBy": "string",
  "crtdOn": 1768397920085,
  "chnlId": 22,
  "traceId": "string"
}
```

**Field Requirements:**
- `msgId`: Always `null`
- `clntIpAddress`: Always `null`
- `trnsId`: Auto-generated unique transaction ID (same as error if both present)
- `sessnTknId`: Session token (same session management as error)
- `reqNo`: Auto-generated unique request number
- `usrRltnshpNo`: Configurable relationship number
- `trnsStts`: Configurable ("success" or "error")
- `strtTime`: Current epoch milliseconds (slightly before endTime)
- `endTime`: Current epoch milliseconds at generation time
- `bizReqInput`: Base64 encoded JSON (configurable templates)
- `bizRegInput`: Base64 encoded JSON (optional, configurable templates)
- `bizRespOutput`: Base64 encoded JSON (configurable templates)
- `usrId`: Configurable user ID (should match error if both present)
- `usrTyp`: Configurable (e.g., "ETB_CUSTOMER")
- `cmndId`: Configurable command ID from predefined list
- `crtdBy`: Configurable string
- `crtdOn`: Current epoch milliseconds
- `chnlId`: Configurable integer
- `traceId`: Auto-generated UUID or hex string

---

## 3. Configuration Requirements

### 3.1 YAML Configuration Structure

```yaml
# Execution Control
execution:
  mode: "duration"              # "duration" or "count"
  duration: "5m"                # Duration string (e.g., "30s", "5m", "1h")
  count: 10000                  # Total messages to generate (if mode=count)
  eps: 1000                     # Events per second target
  
# Message Distribution
distribution:
  error_only_percent: 0         # % of messages with only yono_adt_error
  trans_only_percent: 70        # % of messages with only yono_adt_trans
  both_percent: 30              # % of messages with both fields
  
# Session Management
session:
  rotation_interval: "30s"      # How often to generate new session ID
  session_id_prefix: "DSASLX6BkBafb"
  
# Message Templates
templates:
  error:
    error_codes:
      - "BSee0e01"
      - "BSeeeee1"
    error_types:
      - "BUSINESS_EXCEPTION"
    error_descriptions:
      - '{"TEACS206":{"message":"Input provided is invalid"}}'
      - '{"TEACS207":{"message":"Account not found"}}'
    error_details:
      - "BUSINESS_EXCEPTION"
    created_by: "System"
    
  transaction:
    statuses:
      - weight: 90
        value: "success"
      - weight: 10
        value: "error"
    user_types:
      - "ETB_CUSTOMER"
    command_ids:
      - "statement_view"
      - "home_dashboard_cards_getAllCBSAccountSummary"
      - "home_dashboard_notification_getAllNotifications"
    channel_ids:
      - 22
    user_relationship_numbers:
      - "02111764664914472"
      - "82111764664914472"
    biz_req_inputs:
      - '{"viewAccountStatementRequest":{"fromDate":null,"accountNo":"30096060714","toDate":null}}'
      - '{}'
    biz_resp_outputs:
      - '{}'
      - '{"response":"{\"ITERATE\":\"Y\",\"STATUS\":\"Y\"}"}'
    created_by: "System"
    
# User ID Configuration
user_ids:
  mode: "fixed"                 # "fixed" or "range"
  fixed_list:
    - 1212301889885902848
    - 1090951129735557800
  range_min: 1000000000000000000
  range_max: 9999999999999999999
  
# ID Generation
id_generation:
  transaction_id_pattern: "cRhKTOYInnV"  # Part of generated trnsId
  request_no_length: 27
  trace_id_format: "hex"        # "hex" or "uuid"
  trace_id_length: 32
  
# Kafka Producer Configuration
kafka:
  brokers:
    - "localhost:9092"
  topic: "yono-audit-messages"
  
  # Producer settings
  producer:
    num_producers: 10           # Number of concurrent Kafka producers
    required_acks: 1            # 0=NoResponse, 1=WaitForLocal, -1=WaitForAll
    compression: "snappy"       # "none", "gzip", "snappy", "lz4", "zstd"
    max_message_bytes: 1000000
    flush_frequency: "100ms"
    flush_messages: 100
    idempotent: false
    retry_max: 3
    retry_backoff: "100ms"
    
  # Connection settings
  connection:
    timeout: "10s"
    keep_alive: "30s"
    
# Concurrency Control
concurrency:
  worker_pool_size: 100         # Number of goroutines generating messages
  buffer_size: 1000             # Channel buffer size between generators and producers
  rate_limiter_burst: 100       # Token bucket burst size
  
# Logging
logging:
  level: "info"                 # "debug", "info", "warn", "error"
  format: "json"                # "json" or "text"
  metrics_interval: "10s"       # How often to log metrics
  
# Monitoring
monitoring:
  enable_metrics: true
  metrics_port: 9090            # Prometheus metrics port
```

---

## 4. Technical Requirements

### 4.1 Concurrency Architecture

#### 4.1.1 Worker Pool Pattern
- Implement a fixed-size worker pool (configurable via `concurrency.worker_pool_size`)
- Workers should consume from a rate-limited job channel
- Avoid spawning unbounded goroutines

#### 4.1.2 Rate Limiting
- Use `golang.org/x/time/rate` limiter for precise EPS control
- Configure burst size to allow smooth message flow
- Rate limiter should be shared across all workers

#### 4.1.3 Producer Pool
- Create a pool of Kafka producers (configurable via `kafka.producer.num_producers`)
- Round-robin or least-loaded distribution of messages to producers
- Each producer should have its own goroutine for async sending

#### 4.1.4 Graceful Shutdown
- Handle SIGINT and SIGTERM signals
- Drain message buffers before exit
- Ensure all in-flight messages are sent or logged
- Timeout for shutdown process (e.g., 30 seconds)

### 4.2 Message Generation Logic

#### 4.2.1 Distribution Logic
```
For each message:
1. Determine message type based on distribution percentages
2. Generate appropriate fields based on type
3. Ensure shared fields (trnsId, sessnTknId, usrId) are consistent when both types present
```

#### 4.2.2 Dynamic Field Generation
- **Transaction ID (`trnsId`)**: 
  - Format: `<epoch_millis><random_pattern><suffix>`
  - Example: `1768397919967cRhKTOYInnV058352`
  - Should be unique per message
  
- **Session Token ID (`sessnTknId`)**:
  - Format: `<epoch_millis><prefix><random_suffix>`
  - Should rotate every `session.rotation_interval`
  - All messages within rotation window share same session ID
  
- **Request Number (`reqNo`)**:
  - Random numeric string of length `id_generation.request_no_length`
  - Should be unique per message
  
- **Trace ID (`traceId`)**:
  - UUID v4 or hex string based on `id_generation.trace_id_format`
  - Should be unique per message
  
- **Timestamps**:
  - `errTime`, `crtdOn`, `endTime`: Current epoch milliseconds
  - `strtTime`: `endTime - random(10, 200)` milliseconds

#### 4.2.3 Template Selection
- Randomly select from configured lists with equal probability
- For weighted fields (e.g., transaction status), use weighted random selection
- Base64 encode `bizReqInput`, `bizRegInput`, `bizRespOutput` from templates

### 4.3 Kafka Integration

#### 4.3.1 Library
Use `github.com/IBM/sarama` for Kafka producer implementation

#### 4.3.2 Producer Configuration Mapping
```go
config.Producer.RequiredAcks = sarama.RequiredAcks(kafka.producer.required_acks)
config.Producer.Compression = sarama.CompressionCodec(kafka.producer.compression)
config.Producer.MaxMessageBytes = kafka.producer.max_message_bytes
config.Producer.Flush.Frequency = kafka.producer.flush_frequency
config.Producer.Flush.Messages = kafka.producer.flush_messages
config.Producer.Idempotent = kafka.producer.idempotent
config.Producer.Retry.Max = kafka.producer.retry_max
config.Producer.Retry.Backoff = kafka.producer.retry_backoff
```

#### 4.3.3 Message Format
- **Key**: Use `trnsId` as message key for ordering guarantees
- **Value**: JSON string containing `yono_adt_error` and/or `yono_adt_trans`
- **Headers**: Optional - add generation timestamp, message type flags

### 4.4 Error Handling

#### 4.4.1 Kafka Producer Errors
- Log all producer errors with context (message ID, timestamp, error details)
- Implement retry logic based on `kafka.producer.retry_max`
- Track failed message count in metrics

#### 4.4.2 Configuration Validation
- Validate YAML structure on startup
- Ensure distribution percentages sum to 100
- Validate duration strings, EPS values, connection strings
- Fail fast on invalid configuration

### 4.5 Monitoring & Metrics

#### 4.5.1 Metrics to Track
- Messages generated (total, per type)
- Messages sent successfully
- Messages failed
- Current EPS (actual vs target)
- Producer queue depth
- Worker pool utilization
- Session rotation count
- Latency percentiles (p50, p95, p99)

#### 4.5.2 Prometheus Metrics
Expose metrics in Prometheus format on configured port:
- `kafka_messages_generated_total{type="error|trans|both"}`
- `kafka_messages_sent_total`
- `kafka_messages_failed_total`
- `kafka_producer_queue_depth`
- `kafka_message_generation_duration_seconds`
- `kafka_current_eps`

#### 4.5.3 Logging
- Startup: Log configuration summary
- Runtime: Log metrics at `logging.metrics_interval`
- Shutdown: Log final statistics
- Debug level: Log individual message details (truncated)

---

## 5. Project Structure

```
kafka-message-generator/
├── cmd/
│   └── generator/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # YAML config struct
│   │   └── validator.go         # Config validation
│   ├── generator/
│   │   ├── message.go           # Message generation logic
│   │   ├── session.go           # Session ID management
│   │   ├── id_generator.go      # ID generation utilities
│   │   └── template.go          # Template selection
│   ├── producer/
│   │   ├── kafka.go             # Kafka producer wrapper
│   │   └── pool.go              # Producer pool management
│   ├── worker/
│   │   ├── pool.go              # Worker pool implementation
│   │   └── rate_limiter.go      # Rate limiting logic
│   └── metrics/
│       ├── metrics.go           # Metrics collection
│       └── prometheus.go        # Prometheus exporter
├── pkg/
│   └── models/
│       ├── error.go             # Error message struct
│       └── transaction.go       # Transaction message struct
├── config/
│   └── config.yaml              # Default configuration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. Implementation Steps

### Phase 1: Configuration & Setup
1. Define YAML configuration structure
2. Implement configuration loading and validation
3. Set up logging framework (e.g., zap or logrus)

### Phase 2: Message Generation
1. Define Go structs for error and transaction messages
2. Implement ID generators (trnsId, sessnTknId, reqNo, traceId)
3. Implement session rotation logic
4. Create message builder with template support
5. Implement distribution logic

### Phase 3: Kafka Integration
1. Set up Sarama producer with configuration
2. Implement producer pool
3. Add error handling and retry logic
4. Test connectivity and basic message sending

### Phase 4: Concurrency & Rate Limiting
1. Implement worker pool pattern
2. Add rate limiter with configurable EPS
3. Create message generation pipeline
4. Implement graceful shutdown

### Phase 5: Monitoring & Metrics
1. Add Prometheus metrics
2. Implement periodic metrics logging
3. Add runtime statistics tracking

### Phase 6: Testing & Documentation
1. Unit tests for message generation
2. Integration tests with Kafka
3. Load testing for target EPS
4. Update README with usage instructions

---

## 7. Example Usage

### 7.1 Basic Execution
```bash
# Run with default config
./kafka-message-generator --config config.yaml

# Run with custom config
./kafka-message-generator --config custom-config.yaml

# Override specific values
./kafka-message-generator --config config.yaml --eps 5000 --duration 10m
```

### 7.2 Configuration Examples

**High Throughput Scenario:**
```yaml
execution:
  eps: 10000
  duration: "1h"
concurrency:
  worker_pool_size: 200
  buffer_size: 5000
kafka:
  producer:
    num_producers: 20
    required_acks: 1
    compression: "snappy"
```

**Error-Heavy Scenario:**
```yaml
distribution:
  error_only_percent: 0
  trans_only_percent: 10
  both_percent: 90  # Most messages have errors
```

**Testing Scenario:**
```yaml
execution:
  mode: "count"
  count: 1000
  eps: 10
logging:
  level: "debug"
```

---

## 8. Non-Functional Requirements

### 8.1 Performance
- Support sustained EPS up to 10,000 events/second
- CPU usage should be efficient (< 50% on 4-core machine at 5K EPS)
- Memory usage should be bounded (< 500MB)

### 8.2 Reliability
- No message loss in normal operation
- Graceful degradation under Kafka unavailability
- Accurate EPS regardless of Kafka performance

### 8.3 Maintainability
- Clean, idiomatic Go code
- Comprehensive inline documentation
- Clear error messages
- Easily extensible for new message types

### 8.4 Observability
- Prometheus metrics for monitoring
- Structured logging for debugging
- Real-time EPS tracking

---

## 9. Testing Requirements

### 9.1 Unit Tests
- Message generation correctness
- Distribution logic accuracy
- ID generation uniqueness
- Configuration validation

### 9.2 Integration Tests
- Kafka connectivity
- Message format validation in Kafka
- Producer error handling

### 9.3 Performance Tests
- EPS accuracy (±5% of target)
- Sustained load testing (30+ minutes)
- Resource usage profiling

---

## 10. Dependencies

### 10.1 Core Dependencies
```
github.com/IBM/sarama                 # Kafka client
gopkg.in/yaml.v3                      # YAML parsing
golang.org/x/time/rate                # Rate limiting
github.com/prometheus/client_golang   # Metrics
go.uber.org/zap                       # Logging (or logrus)
github.com/google/uuid                # UUID generation
```

### 10.2 Development Dependencies
```
github.com/stretchr/testify          # Testing utilities
github.com/golang/mock               # Mocking
```

---

## 11. Delivery Checklist

- [ ] Complete Go implementation
- [ ] Default `config.yaml` with sensible defaults
- [ ] `README.md` with setup and usage instructions
- [ ] Unit tests (>70% coverage)
- [ ] Integration test suite
- [ ] Makefile with build, test, run targets
- [ ] Docker support (optional)
- [ ] Example configurations for common scenarios
- [ ] Performance benchmarking results

---

## 12. Edge Cases & Considerations

1. **Zero EPS**: Should still generate messages but with very low rate
2. **Session Rotation**: Ensure thread-safe session ID updates
3. **Kafka Downtime**: Buffer messages or fail gracefully based on config
4. **Large Batch Sizes**: Avoid OOM by controlling buffer sizes
5. **Time Skew**: Use monotonic timestamps for strtTime/endTime consistency
6. **UTF-8 Encoding**: Ensure all JSON strings are properly escaped
7. **Base64 Padding**: Verify base64 encoding of bizReq/bizResp fields

---

## 13. Success Criteria

1. ✅ Generates valid YONO audit messages matching exact schema
2. ✅ Achieves target EPS within ±5% accuracy
3. ✅ Runs stable for extended durations (hours)
4. ✅ Proper distribution of message types
5. ✅ Session IDs rotate as configured
6. ✅ All timestamps are current and realistic
7. ✅ Graceful shutdown with zero message loss
8. ✅ Comprehensive monitoring and metrics
9. ✅ Clean, maintainable codebase

---

**End of PRD**