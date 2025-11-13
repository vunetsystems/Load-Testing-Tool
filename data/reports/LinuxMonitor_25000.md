# 📊 Test Report - e9298fff-ef74-45af-902d-1e852635f0e1

## 🧾 Test Summary

- **Test Id**: `e9298fff-ef74-45af-902d-1e852635f0e1`
- **Target Eps**: `25000`
- **Start Time**: `2025-11-12 02:55:08.204105864+00:00`
- **End Time**: `2025-11-12 03:55:08.507972696+00:00`
- **O11Y Sources**: `["LinuxMonitor"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `0.0` | `23268.191925184256` | `28086.711019918646` |
| Output | `218.4` | `9646.173123309374` | `7945.644570999732` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `495018.0` | `24277221.165266108` | `48511791.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `3.0037947833333334` | `3.0037947833333334` | `3.0037947833333334` |
| kafka_cluster_cp_kafka_1 | `1.0102167833333333` | `1.0102167833333333` | `1.0102167833333333` |
| kafka_cluster_cp_kafka_2 | `5.841236533333333` | `5.841236533333333` | `5.841236533333333` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `99.99032020568848` | `99.99032020568848` | `99.99032020568848` |
| kafka_cluster_cp_kafka_1 | `50.84228515625` | `50.84228515625` | `50.84228515625` |
| kafka_cluster_cp_kafka_2 | `99.99794960021973` | `99.99794960021973` | `99.99794960021973` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0536758` | `21.96751339375` | `43.8813509875` |
| chi_clickhouse_vusmart_0_1_0 | `0.1005014` | `4.25344686875` | `8.4063923375` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.002716064453125` | `41.378384828567505` | `80.75405359268188` |
| chi_clickhouse_vusmart_0_1_0 | `2.7820587158203125` | `20.462745428085327` | `38.14343214035034` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `100.13067950000001` | `100.13067950000001` | `100.13067950000001` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `97.71484375` | `97.71484375` | `97.71484375` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `2.11` | `2.76` | `3.28` |
| kafka_2_node | `1.01` | `1.15` | `1.33` |
| kafka_3_node | `6.84` | `8.98` | `10.13` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `47.23` | `66.28` | `74.31` |
| kafka_2_node | `45.75` | `48.27` | `50.09` |
| kafka_3_node | `46.65` | `57.55` | `60.63` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `2.47` | `2.88` | `3.46` |
| ch2_node | `9.99` | `12.55` | `18.85` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `56.28` | `57.55` | `59.57` |
| ch2_node | `55.23` | `57.63` | `61.02` |

