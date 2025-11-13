# 📊 Test Report - 25c5934d-5010-4266-b11c-2123300701d8

## 🧾 Test Summary

- **Test Id**: `25c5934d-5010-4266-b11c-2123300701d8`
- **Target Eps**: `25000`
- **Start Time**: `2025-11-12 04:07:08.229112349+00:00`
- **End Time**: `2025-11-12 05:07:08.508526133+00:00`
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
| Input  | `19856.26192010678` | `24060.13582720987` | `25278.6` |
| Output | `88.5459369302244` | `2685.2182086850585` | `4539.1973093545985` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `142367.0` | `37789181.205574915` | `75560042.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.3725891734920635` | `4.224661316666667` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.27731855746031747` | `2.8173356333333333` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.5270333723809524` | `5.888412266666667` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `10.837749867212205` | `99.98979568481445` |
| kafka_cluster_cp_kafka_1 | `0.0` | `13.232266846157255` | `99.99797344207764` |
| kafka_cluster_cp_kafka_2 | `0.0` | `14.06621081488473` | `99.9981164932251` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `1.6937921063492063` | `28.857065922222226` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.6300941116666667` | `9.300692044444443` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `10.290517379589776` | `80.19507913028492` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `3.6918022786201874` | `29.992193334242877` |

## 🔧 Pipeline Pod Metrics

### Pipeline Info

| Source | Pipeline Name | Threads | Instances |
|---------|----------------|----------|------------|
| `Kubernetes` | `kubernetes-metrics` | `4` | `1` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `11.0687523291632` | `100.56576700000002` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `0.0` | `12.576281278895053` | `110.77656250000001` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.91` | `2.71` | `3.61` |
| kafka_2_node | `1.4` | `6.11` | `9.93` |
| kafka_3_node | `1.65` | `2.09` | `2.7` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `64.37` | `67.61` | `73.88` |
| kafka_2_node | `53.27` | `58.77` | `60.6` |
| kafka_3_node | `46.07` | `54.4` | `59.68` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `5.94` | `7.95` | `9.67` |
| ch2_node | `9.44` | `11.96` | `15.11` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `58.07` | `58.57` | `59.32` |
| ch2_node | `57.03` | `58.63` | `60.89` |

