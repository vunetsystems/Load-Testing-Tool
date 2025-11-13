# 📊 Test Report - 81e47287-12b5-4b3b-b305-c93b6276a24f

## 🧾 Test Summary

- **Test Id**: `81e47287-12b5-4b3b-b305-c93b6276a24f`
- **Target Eps**: `10000`
- **Start Time**: `2025-11-11 19:12:08.096101847+00:00`
- **End Time**: `2025-11-11 20:12:16.909991952+00:00`
- **O11Y Sources**: `["Kubernetes"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `0.0` | `9952.658547861813` | `10521.918594462366` |
| Output | `18.81009661929979` | `3668.514526741794` | `2655.2162274286216` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `612195.0` | `11127441.014534883` | `22419993.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.43707666599206346` | `0.43707666599206346` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.26313119662698414` | `0.26313119662698414` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.18263771484126984` | `0.18263771484126984` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `12.369298083441597` | `12.369298083441597` | `12.369298083441597` |
| kafka_cluster_cp_kafka_1 | `14.019653626850673` | `14.019653626850673` | `14.019653626850673` |
| kafka_cluster_cp_kafka_2 | `9.871969450087773` | `9.871969450087773` | `9.871969450087773` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `1.3897645280158728` | `1.3897645280158728` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.5362979070370371` | `0.5362979070370371` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `8.14073493166798` | `8.14073493166798` | `8.14073493166798` |
| chi_clickhouse_vusmart_0_1_0 | `2.8193231523871756` | `2.8193231523871756` | `2.8193231523871756` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `100.1808901` | `100.1808901` | `100.1808901` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `100.0` | `100.0` | `100.0` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.58` | `1.83` | `2.04` |
| kafka_2_node | `1.17` | `1.54` | `1.92` |
| kafka_3_node | `1.61` | `4.62` | `10.15` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `47.89` | `58.23` | `67.51` |
| kafka_2_node | `59.18` | `62.01` | `64.4` |
| kafka_3_node | `54.55` | `59.54` | `61.96` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `2.24` | `2.79` | `3.81` |
| ch2_node | `8.27` | `10.48` | `15.68` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `59.67` | `60.28` | `61.25` |
| ch2_node | `63.69` | `64.38` | `65.72` |

