# 📊 Test Report - fde8e630-d30b-4010-853f-e0f497478507

## 🧾 Test Summary

- **Test Id**: `fde8e630-d30b-4010-853f-e0f497478507`
- **Target Eps**: `5000`
- **Start Time**: `2025-11-11 15:29:07.967017281+00:00`
- **End Time**: `2025-11-11 16:29:16.910432546+00:00`
- **O11Y Sources**: `["Mssql"]`
- **Duration**: `1.00 hours`
- **Status**: `completed`

## 📈 Topic Metrics

### Kafka Specs

#### 📥 Input Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `mssql-telegraf` | `1` | `2` |

#### 📤 Output Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `mssql-memory-clerks` | `1` | `2` |
| `mssql-database-io` | `1` | `2` |
| `mssql-net-response` | `1` | `2` |
| `mssql-hadr-replica` | `1` | `2` |
| `mssql-schedulers` | `1` | `2` |
| `mssql-requests` | `1` | `2` |
| `mssql-server-properties` | `1` | `2` |
| `mssql-performance` | `1` | `2` |
| `mssql-hadr-dbreplica` | `1` | `2` |
| `mssql-session` | `1` | `2` |
| `mssql-telegraf-health` | `1` | `2` |
| `mssql-volume-space` | `1` | `2` |
| `mssql-cpu` | `1` | `2` |
| `mssql-waitstats` | `1` | `2` |
| `mssql-cluster` | `1` | `2` |
| `mssql-recentbackup` | `1` | `2` |


### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `4819.654021925699` | `4884.7317699882815` | `5572.8` |
| Output | `0.0` | `4835.113220332743` | `4883.372963338767` |

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

## 🔧 Pipeline Pod Metrics

### Pipeline Info

| Source | Pipeline Name | Threads | Instances |
|---------|----------------|----------|------------|
| `Mssql` | `mssql-telegraf-pipeline` | `1` | `1` |

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

