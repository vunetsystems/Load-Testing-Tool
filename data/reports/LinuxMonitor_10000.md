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
| kafka_cluster_cp_kafka_0 | `2.9364283833333333` | `2.9364283833333333` | `2.9364283833333333` |
| kafka_cluster_cp_kafka_1 | `1.7814835833333333` | `1.7814835833333333` | `1.7814835833333333` |
| kafka_cluster_cp_kafka_2 | `1.577254516666667` | `1.577254516666667` | `1.577254516666667` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `54.77902889251709` | `54.77902889251709` | `54.77902889251709` |
| kafka_cluster_cp_kafka_1 | `99.99833106994629` | `99.99833106994629` | `99.99833106994629` |
| kafka_cluster_cp_kafka_2 | `98.78261089324951` | `98.78261089324951` | `98.78261089324951` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0542877` | `7.095099756249999` | `14.135911812499998` |
| chi_clickhouse_vusmart_0_1_0 | `0.0965686` | `50.340742143750006` | `100.58491568750001` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.0261764526367188` | `30.31069040298462` | `58.59520435333252` |
| chi_clickhouse_vusmart_0_1_0 | `2.794647216796875` | `8.580631017684937` | `14.366614818572998` |

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

