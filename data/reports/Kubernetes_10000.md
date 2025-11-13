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
| kafka_cluster_cp_kafka_0 | `4.996523383333334` | `4.996523383333334` | `4.996523383333334` |
| kafka_cluster_cp_kafka_1 | `2.7724014` | `2.7724014` | `2.7724014` |
| kafka_cluster_cp_kafka_2 | `1.5280989166666668` | `1.5280989166666668` | `1.5280989166666668` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `99.9903917312622` | `99.9903917312622` | `99.9903917312622` |
| kafka_cluster_cp_kafka_1 | `99.99792575836182` | `99.99792575836182` | `99.99792575836182` |
| kafka_cluster_cp_kafka_2 | `93.00861358642578` | `93.00861358642578` | `93.00861358642578` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.050409` | `14.61872736875` | `29.1870457375` |
| chi_clickhouse_vusmart_0_1_0 | `0.0912541` | `7.6375667187500005` | `15.1838793375` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.020263671875` | `33.64956974983215` | `65.2788758277893` |
| chi_clickhouse_vusmart_0_1_0 | `2.7894973754882812` | `13.88123631477356` | `24.972975254058838` |

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

