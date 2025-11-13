# 📊 Test Report - 98b53947-1ae7-4743-8452-d97d1afd9aa1

## 🧾 Test Summary

- **Test Id**: `98b53947-1ae7-4743-8452-d97d1afd9aa1`
- **Target Eps**: `5000`
- **Start Time**: `2025-11-11 13:05:07.961470855+00:00`
- **End Time**: `2025-11-11 14:05:08.96212791+00:00`
- **O11Y Sources**: `["Linux Monitor"]`
- **Timeout Seconds**: `3600`
- **Status**: `completed`

## 📈 Topic Metrics

### Input/Output Topic Metrics

| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |
|------|--------------|--------------|--------------|
| Input  | `4961.049541381013` | `5007.302350257911` | `5469.799999999999` |
| Output | `0.0` | `4948.844202409506` | `3019.4115464106812` |

### Lag Metrics

| Min Lag | Avg Lag | Max Lag |
|----------|----------|----------|
| `1.0` | `1004.7551020408164` | `16813.0` |

## 🖥️ Pod Metrics

### Kafka Cluster Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `0.0` | `0.3102413829761905` | `4.48104595` |
| kafka_cluster_cp_kafka_1 | `0.0` | `0.30456019980158733` | `3.59413265` |
| kafka_cluster_cp_kafka_2 | `0.0` | `0.27744596396825394` | `5.014965616666666` |

### Kafka Cluster Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_cluster_cp_kafka_0 | `9.995831989106678` | `36.66194741688077` | `99.99001026153564` |
| kafka_cluster_cp_kafka_1 | `10.640076171784173` | `36.87933397671533` | `99.99792575836182` |
| kafka_cluster_cp_kafka_2 | `10.27513969512213` | `36.49921890289065` | `99.2225170135498` |

### ClickHouse Pods CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.0` | `0.006600925178147269` | `16.2305860375` |
| chi_clickhouse_vusmart_0_1_0 | `0.0` | `0.010092263571428572` | `100.29312139999999` |

### ClickHouse Pods Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| chi_clickhouse_vusmart_0_0_0 | `0.2909037392949265` | `6.689594316746824` | `50.89014768600464` |
| chi_clickhouse_vusmart_0_1_0 | `0.39846329461960567` | `2.2197778546620928` | `15.193212032318115` |

### Pipeline Pod CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `71.538374` | `71.538374` | `71.538374` |

### Pipeline Pod Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| pipeline_pod | `55.54687499999999` | `55.54687499999999` | `55.54687499999999` |

## 💻 Node Metrics

### Kafka Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `1.72` | `2.23` | `3.04` |
| kafka_2_node | `8.01` | `8.66` | `9.33` |
| kafka_3_node | `1.45` | `2.23` | `3.91` |

### Kafka Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| kafka_1_node | `44.31` | `56.1` | `64.35` |
| kafka_2_node | `47.85` | `62.87` | `72.9` |
| kafka_3_node | `45.16` | `58.36` | `65.8` |

### ClickHouse Node CPU

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `2.08` | `3.02` | `4.25` |
| ch2_node | `9.59` | `10.43` | `11.88` |

### ClickHouse Node Memory

| Component | Min | Avg | Max |
|------------|-----|-----|-----|
| ch1_node | `60.6` | `61.24` | `61.53` |
| ch2_node | `63.86` | `64.95` | `66.24` |

