# Usage Guide: YonoAuditTransLogs

This guide explains how to run the YonoAuditTransLogs Kafka message generator, required files, and how to configure key parameters like Kafka settings, throughput (EPS), and duration.

## 1. Required Files

To run the application, you need the following files:

1.  **Binary**: `bin/kafka-message-generator` (or the source code if running with `go run`).
2.  **Configuration Files**:
    - `config/app_config.yaml`: Contains execution settings (Duration, EPS, Kafka connection).
    - `config/data_config.yaml`: Contains message templates and data generation rules.

Ensure the configuration files are accessible relative to where you run the binary, or provide absolute paths.

## 2. How to Run

```bash
# Run the binary
./bin/kafka-message-generator --app-config config/app_config.yaml --data-config config/data_config.yaml
```

## 3. Configuration

All execution-related configurations are found in **`config/app_config.yaml`**.

### How to Change Kafka Specs

To modify the Kafka broker addresses or topic names, edit the `kafka` section:

```yaml
kafka:
  # List of Kafka brokers
  brokers:
    - "e2e-83-134:9094"
    - "another-broker:9092"

  # Topic names
  topic: "yono-adt-trans" # Main audit topic
  access_log_topic: "ms-logs" # Access logs topic
  eis_topic: "yono-adt-service" # EIS logs topic
```

### How to Change Events Per Second (EPS)

To increase or decrease the load, change the `eps` value in the `execution` section:

```yaml
execution:
  # Target number of events (transactions) per second
  eps: 5000
```

_Note: Depending on configuration, 1 "event" might generate multiple Kafka messages (Audit + Access + EIS)._

### How to Change Run Duration

To control how long the generator runs, modify the `mode` and `duration` fields in the `execution` section:

```yaml
execution:
  # Mode can be "duration" (time-based) or "count" (fixed number of messages)
  mode: "duration"

  # How long to run: "10s", "5m", "1h", etc.
  duration: "30m"
```

## 4. Troubleshooting config issues

If you see errors about missing configuration files, ensure you are providing the correct paths via the `--app-config` and `--data-config` flags. The default values assume you are running the command from the project root.
