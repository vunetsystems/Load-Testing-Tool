# Kafka Message Generator - Quick Start Guide

## Prerequisites
- Go 1.21+
- Kafka cluster running (default: localhost:9092)

## Installation

```bash
cd /home/vunet/Load-Testing-Tool/SBI_Generator/YonoAuditTransLogs

# Install dependencies (already done)
make install-deps

# Build the application (already done)
make build
```

Binary location: `bin/kafka-message-generator`

## Quick Start

### 1. Start Kafka (if not running)
```bash
# Start Kafka and create topic
kafka-topics --create --topic yono-audit-messages --bootstrap-server localhost:9092
```

### 2. Run the Generator

**Default configuration** (1000 EPS for 5 minutes):
```bash
./bin/kafka-message-generator --config config/config.yaml
```

**Testing configuration** (1000 messages at 10 EPS):
```bash
./bin/kafka-message-generator --config config/testing.yaml
```

**High throughput** (10K EPS for 1 hour):
```bash
./bin/kafka-message-generator --config config/high-throughput.yaml
```

**Error-heavy** (90% messages with errors):
```bash
./bin/kafka-message-generator --config config/error-heavy.yaml
```

### 3. Monitor

**View logs** (real-time metrics every 10 seconds):
```
{"level":"info","msg":"Metrics","generated":50000,"sent":49998,"failed":2,"current_eps":1002.5,"uptime":"50s"}
```

**Prometheus metrics**:
```bash
curl http://localhost:9090/metrics
```

**Consume messages**:
```bash
kafka-console-consumer --topic yono-audit-messages --bootstrap-server localhost:9092 --from-beginning
```

### 4. Stop Gracefully
Press `Ctrl+C` - the application will drain buffers and show final statistics.

## Configuration Files

- `config/config.yaml` - Default (1000 EPS, 5 min)
- `config/testing.yaml` - Testing (1000 messages, 10 EPS)
- `config/high-throughput.yaml` - Load testing (10K EPS, 1 hour)
- `config/error-heavy.yaml` - Error testing (90% errors)

## Message Format

Messages are sent to Kafka in this format:
```json
{
  "yono_adt_error": "{...}",  // Optional
  "yono_adt_trans": "{...}"   // Optional
}
```

## Troubleshooting

**Kafka connection failed**:
- Verify Kafka is running: `telnet localhost 9092`
- Update broker address in config file

**Low throughput**:
- Increase `worker_pool_size` and `num_producers`
- Reduce `required_acks` to 0

**High memory**:
- Reduce `buffer_size` and `worker_pool_size`

## Documentation

- Full documentation: [README.md](README.md)
- PRD: [PRD.md](PRD.md)
- Implementation walkthrough: See artifacts

## Example Output

```
INFO  Starting Kafka Message Generator  mode=duration eps=1000 topic=yono-audit-messages
INFO  Prometheus metrics available  port=9090
INFO  Initializing Kafka producer pool  num_producers=10
INFO  Initializing worker pool  workers=100
INFO  Starting message generation
INFO  Metrics  generated=10000 sent=10000 failed=0 current_eps=1001.2 uptime=10s
INFO  Metrics  generated=20000 sent=20000 failed=0 current_eps=1000.8 uptime=20s
...
INFO  Message generation completed
INFO  Final Statistics  total_generated=300000 error_only=0 trans_only=210000 both=90000 sent=300000 failed=0 elapsed=5m0s avg_eps=1000.0
INFO  Shutdown complete
```
