# 📊 Test Report - 81e47287-12b5-4b3b-b305-c93b6276a24f

## 🧾 Test Summary

- **Test Id**: `81e47287-12b5-4b3b-b305-c93b6276a24f`
- **Target Eps**: `10000`
- **Start Time**: `2025-11-11 19:12:08.096101847+00:00`
- **End Time**: `2025-11-11 20:12:16.909991952+00:00`
- **O11Y Sources**: `["Kubernetes"]`
- **Duration**: `1.00 hours`
- **Status**: `completed`

## 📈 Topic Metrics

### Kafka Specs

#### 📥 Input Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `kubernetes-metrics-input` | `1` | `2` |

#### 📤 Output Topics

| Topic Name | Partitions | Replication Factor |
|-------------|-------------|--------------------|
| `kubernetes-kubelet-metrics` | `1` | `2` |
| `kubernetes-kube-state-metrics` | `1` | `2` |
| `kubernetes-etcd-metrics` | `1` | `2` |
| `kubernetes-apiserver-metrics` | `1` | `2` |
| `kubernetes-controllermanager-metrics` | `1` | `2` |
| `kubernetes-scheduler-metrics` | `1` | `2` |
| `kubernetes-proxy-metrics` | `1` | `2` |


### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `0.0` | `9952.658547861813` | `10521.918594462366` |
| Output | `421.91096534254035` | `3668.5145267417947` | `4398.475915792779` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `612195.0` | `11127441.014534883` | `22419993.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.43707666599206346` | `4.996523383333334` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.26313119662698414` | `2.7724014` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.18263771484126984` | `1.5280989166666668` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `12.369298083441597` | `99.9903917312622` |
| kafka_cluster_cp_kafka_1 | `0.0` | `14.019653626850673` | `99.99792575836182` |
| kafka_cluster_cp_kafka_2 | `0.0` | `9.871969450087773` | `93.00861358642578` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `1.3897645280158728` | `25.954508644444445` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.5362979070370371` | `13.513199466666665` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `8.14073493166798` | `61.671391655417054` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `2.8193231523871756` | `23.7334083108341` |

## 🔧 Pipeline Pod Metrics

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `12.51699478095238` | `100.2496112` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `14.364938616071429` | `110.51171875000001` |

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

