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
| Output | `0.0` | `4835.113220332747` | `3284.3938429990794` |

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
| kafka_cluster_cp_kafka_0 | `12.421902531669254` | `37.02587568570697` | `98.65572452545166` |
| kafka_cluster_cp_kafka_1 | `12.189204863139562` | `37.395217475436986` | `99.99644756317139` |
| kafka_cluster_cp_kafka_2 | `13.815035536175682` | `37.93739150440882` | `99.99713897705078` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `0.006491881904761905` | `12.6848292125` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.00952012880952381` | `6.51547475` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.28603690011160715` | `6.863012547215456` | `52.248334884643555` |
| chi_clickhouse_vusmart_0_1_0 | `0.3986549377441406` | `2.172860783874673` | `14.590513706207275` |

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

