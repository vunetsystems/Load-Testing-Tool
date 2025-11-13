# 📊 Test Report - aaa96676-c2b5-4e48-95e8-9b3f2c9c9b8c

## 🧾 Test Summary

- **Test Id**: `aaa96676-c2b5-4e48-95e8-9b3f2c9c9b8c`
- **Target Eps**: `5000`
- **Start Time**: `2025-11-11 14:17:07.904338581+00:00`
- **End Time**: `2025-11-11 15:17:15.515239781+00:00`
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
| Input  | `0.0` | `4979.809894965163` | `5136.91904416021` |
| Output | `0.0` | `4286.424801768304` | `4829.815751401648` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `357352.0` | `1374763.137026239` | `2645506.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| kafka_cluster_cp_kafka_0 | `6.0` | `0.0` | `0.2771869813095238` | `2.318989466666667` |
| kafka_cluster_cp_kafka_1 | `6.0` | `0.0` | `0.24369027166666668` | `1.9713053833333334` |
| kafka_cluster_cp_kafka_2 | `6.0` | `0.0` | `0.29345331154761906` | `2.3382661` |

### Kafka Cluster Pods Memory (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| kafka_cluster_cp_kafka_0 | `16.0` | `0.0` | `13.994209028425672` | `99.99029636383057` |
| kafka_cluster_cp_kafka_1 | `16.0` | `0.0` | `13.929254781632197` | `99.997878074646` |
| kafka_cluster_cp_kafka_2 | `16.0` | `0.0` | `14.101356949125018` | `99.99773502349854` |

### ClickHouse Pods CPU (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| chi_clickhouse_vusmart_0_0_0 | `9.0` | `0.0` | `1.1395028971693124` | `17.9473875` |
| chi_clickhouse_vusmart_0_1_0 | `9.0` | `0.0` | `0.7634914433333334` | `89.22955042222223` |

### ClickHouse Pods Memory (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| chi_clickhouse_vusmart_0_0_0 | `34.0` | `0.0` | `6.614981018194631` | `49.00320838479435` |
| chi_clickhouse_vusmart_0_1_0 | `34.0` | `0.0` | `1.6907489466733958` | `16.09632828656365` |

## 🔧 Pipeline Pod Metrics

### Pipeline Info

| Source | Pipeline Name | Threads | Instances |
|---------|----------------|----------|------------|
| `Kubernetes` | `kubernetes-metrics` | `4` | `1` |

### Pipeline Pod CPU (%)

| Pipeline Name | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| `kubernetes-metrics` | `1.0` | `0.0` | `13.761910772142857` | `100.49424919999998` |

### Pipeline Pod Memory (%)

| Pipeline Name | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| `kubernetes-metrics` | `0.48828125` | `0.0` | `14.350489211309526` | `110.12578125000002` |

## 💻 Node Metrics

### Kafka Node CPU (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| kafka_1_node | `` | `1.6` | `1.98` | `2.18` |
| kafka_2_node | `` | `1.24` | `1.42` | `1.54` |
| kafka_3_node | `` | `5.14` | `8.95` | `10.37` |

### Kafka Node Memory (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| kafka_1_node | `` | `63.51` | `64.09` | `64.63` |
| kafka_2_node | `` | `70.53` | `71.72` | `73.04` |
| kafka_3_node | `` | `63.26` | `64.4` | `65.09` |

### ClickHouse Node CPU (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| ch1_node | `` | `2.32` | `3.02` | `6.02` |
| ch2_node | `` | `8.9` | `10.26` | `11.53` |

### ClickHouse Node Memory (%)

| Component | Allocated | Min (%) | Avg (%) | Max (%) |
|--------------|-------------|----------|----------|----------|
| ch1_node | `` | `61.11` | `61.36` | `61.63` |
| ch2_node | `` | `63.22` | `64.01` | `64.41` |

