# 📊 Test Report - aaa96676-c2b5-4e48-95e8-9b3f2c9c9b8c

## 🧾 Test Summary

- **Test Id**: `aaa96676-c2b5-4e48-95e8-9b3f2c9c9b8c`
- **Target Eps**: `5000`
- **Start Time**: `2025-11-11 14:17:07.904338581+00:00`
- **End Time**: `2025-11-11 15:17:15.515239781+00:00`
- **O11Y Sources**: `["Kubernetes"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `0.0` | `4979.809894965163` | `5136.91904416021` |
| Output | `0.0` | `4286.424801768308` | `2939.199385660771` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `357352.0` | `1374763.137026239` | `2645506.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `2.318989466666667` | `2.318989466666667` | `2.318989466666667` |
| kafka_cluster_cp_kafka_1 | `1.9713053833333334` | `1.9713053833333334` | `1.9713053833333334` |
| kafka_cluster_cp_kafka_2 | `2.3382661` | `2.3382661` | `2.3382661` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `99.99029636383057` | `99.99029636383057` | `99.99029636383057` |
| kafka_cluster_cp_kafka_1 | `99.997878074646` | `99.997878074646` | `99.997878074646` |
| kafka_cluster_cp_kafka_2 | `99.99773502349854` | `99.99773502349854` | `99.99773502349854` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0644354` | `10.120350343750001` | `20.1762652875` |
| chi_clickhouse_vusmart_0_1_0 | `0.0914026` | `50.22883239375` | `100.3662621875` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `2.0204544067382812` | `26.922321319580078` | `51.824188232421875` |
| chi_clickhouse_vusmart_0_1_0 | `2.8079986572265625` | `9.832298755645752` | `16.85659885406494` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `100.42961729999999` | `100.42961729999999` | `100.42961729999999` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `99.98125` | `99.98125` | `99.98125` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.6` | `1.98` | `2.18` |
| kafka_2_node | `1.24` | `1.42` | `1.54` |
| kafka_3_node | `5.14` | `8.95` | `10.37` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `63.51` | `64.09` | `64.63` |
| kafka_2_node | `70.53` | `71.72` | `73.04` |
| kafka_3_node | `63.26` | `64.4` | `65.09` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `2.32` | `3.02` | `6.02` |
| ch2_node | `8.9` | `10.26` | `11.53` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `61.11` | `61.36` | `61.63` |
| ch2_node | `63.22` | `64.01` | `64.41` |

