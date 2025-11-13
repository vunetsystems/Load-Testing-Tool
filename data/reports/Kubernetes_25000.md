# 📊 Test Report - 25c5934d-5010-4266-b11c-2123300701d8

## 🧾 Test Summary

- **Test Id**: `25c5934d-5010-4266-b11c-2123300701d8`
- **Target Eps**: `25000`
- **Start Time**: `2025-11-12 04:07:08.229112349+00:00`
- **End Time**: `2025-11-12 05:07:08.508526133+00:00`
- **O11Y Sources**: `["Kubernetes"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `19856.26192010678` | `24060.13582720987` | `25278.6` |
| Output | `3.7437381940456804` | `2685.218208685057` | `2718.607847884849` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `142367.0` | `37789181.205574915` | `75560042.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `4.224661316666667` | `4.224661316666667` | `4.224661316666667` |
| kafka_cluster_cp_kafka_1 | `2.8173356333333333` | `2.8173356333333333` | `2.8173356333333333` |
| kafka_cluster_cp_kafka_2 | `5.888412266666667` | `5.888412266666667` | `5.888412266666667` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `99.98979568481445` | `99.98979568481445` | `99.98979568481445` |
| kafka_cluster_cp_kafka_1 | `99.99797344207764` | `99.99797344207764` | `99.99797344207764` |
| kafka_cluster_cp_kafka_2 | `99.9981164932251` | `99.9981164932251` | `99.9981164932251` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0576563` | `16.2542387` | `32.4508211` |
| chi_clickhouse_vusmart_0_1_0 | `0.2339923` | `5.330622825000001` | `10.42725335` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.0086288452148438` | `43.48739981651306` | `84.96617078781128` |
| chi_clickhouse_vusmart_0_1_0 | `2.7936935424804688` | `17.209088802337646` | `31.624484062194824` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `100.49909690000001` | `100.49909690000001` | `100.49909690000001` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `99.99296875` | `99.99296875` | `99.99296875` |

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

