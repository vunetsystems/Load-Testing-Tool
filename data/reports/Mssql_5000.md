# 📊 Test Report - fde8e630-d30b-4010-853f-e0f497478507

## 🧾 Test Summary

- **Test Id**: `fde8e630-d30b-4010-853f-e0f497478507`
- **Target Eps**: `5000`
- **Start Time**: `2025-11-11 15:29:07.967017281+00:00`
- **End Time**: `2025-11-11 16:29:16.910432546+00:00`
- **O11Y Sources**: `["Mssql"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Kafka Specs

| Spec | Value |
|------|--------|
| `input_topics` | `[{'name': 'mssql-telegraf', 'partitions': 1, 'replication_factor': 2}]` |
| `output_topics` | `[{'name': 'mssql-memory-clerks', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-database-io', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-net-response', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-hadr-replica', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-schedulers', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-requests', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-server-properties', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-performance', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-hadr-dbreplica', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-session', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-telegraf-health', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-volume-space', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-cpu', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-waitstats', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-cluster', 'partitions': 1, 'replication_factor': 2}, {'name': 'mssql-recentbackup', 'partitions': 1, 'replication_factor': 2}]` |

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `4819.654021925699` | `4884.7317699882815` | `5572.8` |
| Output | `0.0` | `4835.1132203327425` | `4883.372963338767` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `8.0` | `1590.0334928229665` | `3396.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.15446411265873014` | `1.2738853666666667` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.17470797777777777` | `1.3757480166666667` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.20046860087301588` | `1.5525250166666666` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `12.421902531669254` | `98.65572452545166` |
| kafka_cluster_cp_kafka_1 | `0.0` | `12.189204863139562` | `99.99644756317139` |
| kafka_cluster_cp_kafka_2 | `0.0` | `13.815035536175682` | `99.99713897705078` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `0.8398056988359789` | `11.287468377777781` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.38681836751322757` | `5.8093662` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `6.815300308355764` | `49.40321305218865` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `1.6922868199709082` | `13.962936401367188` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `6.062789487857143` | `46.8182473` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `9.144300595238096` | `66.11015625` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.67` | `1.85` | `2.06` |
| kafka_2_node | `1.1` | `1.24` | `1.48` |
| kafka_3_node | `1.26` | `1.5` | `1.88` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `62.17` | `63.84` | `64.64` |
| kafka_2_node | `60.75` | `67.08` | `73.39` |
| kafka_3_node | `59.29` | `62.23` | `63.92` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `4.62` | `6.18` | `6.84` |
| ch2_node | `8.65` | `9.34` | `10.09` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `61.23` | `61.36` | `61.48` |
| ch2_node | `63.92` | `64.16` | `64.37` |

