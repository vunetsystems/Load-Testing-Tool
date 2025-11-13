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

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `4819.654021925699` | `4884.7317699882815` | `5572.8` |
| Output | `0.0` | `4835.113220332746` | `3284.3938429990794` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `8.0` | `1590.0334928229665` | `3396.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `1.2738853666666667` | `1.2738853666666667` | `1.2738853666666667` |
| kafka_cluster_cp_kafka_1 | `1.3757480166666667` | `1.3757480166666667` | `1.3757480166666667` |
| kafka_cluster_cp_kafka_2 | `1.5525250166666666` | `1.5525250166666666` | `1.5525250166666666` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `98.65572452545166` | `98.65572452545166` | `98.65572452545166` |
| kafka_cluster_cp_kafka_1 | `99.99644756317139` | `99.99644756317139` | `99.99644756317139` |
| kafka_cluster_cp_kafka_2 | `99.99713897705078` | `99.99713897705078` | `99.99713897705078` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0562194` | `6.37052430625` | `12.6848292125` |
| chi_clickhouse_vusmart_0_1_0 | `0.0888702` | `3.302172475` | `6.51547475` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.0257949829101562` | `27.137064933776855` | `52.248334884643555` |
| chi_clickhouse_vusmart_0_1_0 | `2.8034210205078125` | `8.696967363357544` | `14.590513706207275` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `46.7632794` | `46.7632794` | `46.7632794` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `57.475781250000004` | `57.475781250000004` | `57.475781250000004` |

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

