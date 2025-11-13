# Performance Test Summary

## General Information
| Metric | Value |
| :--- | :--- |
| **Test ID** | `959b0c79-262d-46b4-a275-59bc41b8c1e5` |
| **Status** | `completed` |
| **Target EPS** | `25000` |
| **Duration** | 1h 0m 1s |
| **Start Time (IST)** | `2025-11-12 07:30:08.241190756+00:00` |
| **End Time (IST)** | `2025-11-12 08:30:35.787835163+00:00` |
| **O11y Sources** | `Mssql` |

---

## Throughput and Rates
| Metric | Value (msgs/sec) |
| :--- | :--- |
| **Avg Input Rate** | `24498.71` |
| **Max Input Rate** | `26989.72` |
| **Avg Output Rate** | `11280.68` |
| **Min Input Rate** | `0.00` |
| **Min Output Rate** | `0.00` |
| **Avg Process Rate** | `11367.49` |
| **Max Process Rate** | `12156.56` |

### Ingestion Summary (Average EPS)
| Table | Avg EPS |
| :--- | :--- |
| `vmetrics_mssql_monitor_server_performance_data` | `7617.34` |
| `vmetrics_mssql_monitor_memory_clerks_data` | `1351.26` |
| `vmetrics_mssql_monitor_cluster_data` | `593.76` |
| `vmetrics_mssql_monitor_sql_session_data` | `589.53` |
| `vmetrics_mssql_monitor_server_cpu_data` | `194.90` |
| `vmetrics_mssql_monitor_IO_statistics_data` | `193.70` |
| `vmetrics_mssql_monitor_server_waitstats_data` | `193.62` |
| `vmetrics_mssql_monitor_server_requests_data` | `193.61` |
| `vmetrics_mssql_monitor_server_recent_backup_data` | `193.59` |
| `vmetrics_mssql_monitor_server_volume_space_data` | `193.41` |
| **Total Estimated Avg EPS** | **`11314.71`** |

---

## Lag Metrics
| Metric | Value |
| :--- | :--- |
| **Min Lag** | `68602.0` |
| **Avg Lag** | `23748229.41160221` |
| **Max Lag** | `47095525.0` |

---

## Resource Utilization (in %)
| Component | CPU Avg | CPU Min | CPU Max | Mem Avg | Mem Min | Mem Max |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Kafka Node 1** | `2.28` | `2.09` | `3.11` | `53.29` | `46.07` | `59.15` |
| **Kafka Node 2** | `10.92` | `9.84` | `13.32` | `45.08` | `44.49` | `45.66` |
| **Kafka Node 3** | `2.09` | `1.85` | `2.56` | `57.40` | `46.96` | `65.98` |
| **ClickHouse Node 0-0-0** | `7.45` | `0.11` | `14.79` | `27.62` | `2.01` | `53.23` |
| **ClickHouse Node 0-1-0** | `52.62` | `0.29` | `104.94` | `7.02` | `3.01` | `11.03` |
| **Kafka Pod cp-kafka-0** | `2.81` | `2.81` | `2.81` | `93.62` | `93.62` | `93.62` |
| **Kafka Pod cp-kafka-1** | `1.20` | `1.20` | `1.20` | `31.53` | `31.53` | `31.53` |
| **Kafka Pod cp-kafka-2** | `2.35` | `2.35` | `2.35` | `99.99` | `99.99` | `99.99` |
| **Pipeline Pod** | `99.98` | `99.98` | `99.98` | `58.32` | `58.32` | `58.32` |

---

✅ **Generated automatically from SQLite: `vudatasim.db`**
