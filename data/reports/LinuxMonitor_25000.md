# 📊 Test Report - e9298fff-ef74-45af-902d-1e852635f0e1

## 🧾 Test Summary

- **Test Id**: `e9298fff-ef74-45af-902d-1e852635f0e1`
- **Target Eps**: `25000`
- **Start Time**: `2025-11-12 02:55:08.204105864+00:00`
- **End Time**: `2025-11-12 03:55:08.507972696+00:00`
- **O11Y Sources**: `["LinuxMonitor"]`
- **Duration**: `1.00 hours`
- **Status**: `completed`

## 📈 Topic Metrics

### Kafka Specs

#### 📥 Input Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `linux-monitor-input` | `1` | `2` |

#### 📤 Output Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `linux-monitor-additional-metrics` | `1` | `2` |
| `linux-monitor-process-metrics` | `1` | `2` |
| `linux-monitor-resource-metrics` | `1` | `2` |
| `linux-monitor-storage-metrics` | `1` | `2` |


### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `0.0` | `23268.191925184256` | `28086.711019918646` |
| Output | `3856.6297033803503` | `9646.17312330937` | `13237.296092732688` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `495018.0` | `24277221.165266108` | `48511791.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.34145666055555557` | `3.0037947833333334` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.10983434063492063` | `1.0102167833333333` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.5670086909920635` | `5.841236533333333` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `12.184617746443974` | `99.99032020568848` |
| kafka_cluster_cp_kafka_1 | `0.0` | `6.074797596250262` | `50.84228515625` |
| kafka_cluster_cp_kafka_2 | `0.0` | `12.798268511181787` | `99.99794960021973` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `1.8508471865079363` | `39.01725418888889` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.5130950504761904` | `7.489649999999999` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `8.812460431865617` | `76.23016133027917` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `4.107015780708036` | `36.126967037425324` |

## 🔧 Pipeline Pod Metrics

### Pipeline Info

| Source | Pipeline Name | Threads | Instances |
|---------|----------------|----------|------------|
| `LinuxMonitor` | `linuxmonitor` | `4` | `1` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `13.142238399285713` | `100.19030610000002` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `12.845982142857146` | `106.93046875` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `2.11` | `2.76` | `3.28` |
| kafka_2_node | `1.01` | `1.15` | `1.33` |
| kafka_3_node | `6.84` | `8.98` | `10.13` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `47.23` | `66.28` | `74.31` |
| kafka_2_node | `45.75` | `48.27` | `50.09` |
| kafka_3_node | `46.65` | `57.55` | `60.63` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `2.47` | `2.88` | `3.46` |
| ch2_node | `9.99` | `12.55` | `18.85` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `56.28` | `57.55` | `59.57` |
| ch2_node | `55.23` | `57.63` | `61.02` |

