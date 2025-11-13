# 📊 Test Report - dc83d3eb-ccd6-4f87-ae08-e6089daaecf9

## 🧾 Test Summary

- **Test Id**: `dc83d3eb-ccd6-4f87-ae08-e6089daaecf9`
- **Target Eps**: `10000`
- **Start Time**: `2025-11-11 18:00:08.120238133+00:00`
- **End Time**: `2025-11-11 19:00:16.910522743+00:00`
- **O11Y Sources**: `["LinuxMonitor"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `9973.88318783893` | `10010.66360759976` | `10591.4` |
| Output | `0.0` | `6829.176117248166` | `5147.718961844397` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `38000.0` | `5353975.291666667` | `11104796.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.3615357829761905` | `0.3615357829761905` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.22048556615079365` | `0.22048556615079365` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.19642745202380954` | `0.19642745202380954` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `6.140249967575073` | `6.140249967575073` | `6.140249967575073` |
| kafka_cluster_cp_kafka_1 | `10.320962043035598` | `10.320962043035598` | `10.320962043035598` |
| kafka_cluster_cp_kafka_2 | `9.597944191523961` | `9.597944191523961` | `9.597944191523961` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `1.1912808678306879` | `1.1912808678306879` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.590107922010582` | `0.590107922010582` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `7.357395930784424` | `7.357395930784424` | `7.357395930784424` |
| chi_clickhouse_vusmart_0_1_0 | `1.370037971090536` | `1.370037971090536` | `1.370037971090536` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `99.77371339999999` | `99.77371339999999` | `99.77371339999999` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `96.8546875` | `96.8546875` | `96.8546875` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.64` | `1.77` | `2.02` |
| kafka_2_node | `1.21` | `1.37` | `1.54` |
| kafka_3_node | `1.85` | `2.01` | `2.34` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `44.45` | `57.17` | `69.16` |
| kafka_2_node | `45.51` | `54.2` | `59.14` |
| kafka_3_node | `44.26` | `48.47` | `51.91` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `1.7` | `2.52` | `4.88` |
| ch2_node | `9.13` | `10.17` | `10.87` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `60.92` | `61.28` | `61.48` |
| ch2_node | `63.57` | `64.75` | `65.3` |

