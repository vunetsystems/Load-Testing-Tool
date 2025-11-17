// Package models defines data structures for the test run tracking process
package models

import "time"

// TestRun represents a test run record
type TestRun struct {
	TestID                  string    `json:"test_id"`
	TestName                string    `json:"test_name,omitempty"`
	TargetEPS               int       `json:"target_eps"`
	StartTime               time.Time `json:"start_time"`
	EndTime                 *time.Time `json:"end_time,omitempty"`
	O11ySources             []string  `json:"o11y_sources"`
	TimeoutSeconds          int       `json:"timeout_seconds"`
	Status                  string    `json:"status"` // 'running', 'completed', 'stopped', 'failed'
	Kafka1NodeCpuMin        float64   `json:"kafka_1_node_cpu_min,omitempty"`
	Kafka1NodeCpuAvg        float64   `json:"kafka_1_node_cpu_avg,omitempty"`
	Kafka1NodeCpuMax        float64   `json:"kafka_1_node_cpu_max,omitempty"`
	Kafka1NodeMemMin        float64   `json:"kafka_1_node_mem_min,omitempty"`
	Kafka1NodeMemAvg        float64   `json:"kafka_1_node_mem_avg,omitempty"`
	Kafka1NodeMemMax        float64   `json:"kafka_1_node_mem_max,omitempty"`
	Kafka2NodeCpuMin        float64   `json:"kafka_2_node_cpu_min,omitempty"`
	Kafka2NodeCpuAvg        float64   `json:"kafka_2_node_cpu_avg,omitempty"`
	Kafka2NodeCpuMax        float64   `json:"kafka_2_node_cpu_max,omitempty"`
	Kafka2NodeMemMin        float64   `json:"kafka_2_node_mem_min,omitempty"`
	Kafka2NodeMemAvg        float64   `json:"kafka_2_node_mem_avg,omitempty"`
	Kafka2NodeMemMax        float64   `json:"kafka_2_node_mem_max,omitempty"`
	Kafka3NodeCpuMin        float64   `json:"kafka_3_node_cpu_min,omitempty"`
	Kafka3NodeCpuAvg        float64   `json:"kafka_3_node_cpu_avg,omitempty"`
	Kafka3NodeCpuMax        float64   `json:"kafka_3_node_cpu_max,omitempty"`
	Kafka3NodeMemMin        float64   `json:"kafka_3_node_mem_min,omitempty"`
	Kafka3NodeMemAvg        float64   `json:"kafka_3_node_mem_avg,omitempty"`
	Kafka3NodeMemMax        float64   `json:"kafka_3_node_mem_max,omitempty"`
	Ch1NodeCpuMin           float64   `json:"ch1_node_cpu_min,omitempty"`
	Ch1NodeCpuAvg           float64   `json:"ch1_node_cpu_avg,omitempty"`
	Ch1NodeCpuMax           float64   `json:"ch1_node_cpu_max,omitempty"`
	Ch1NodeMemMin           float64   `json:"ch1_node_mem_min,omitempty"`
	Ch1NodeMemAvg           float64   `json:"ch1_node_mem_avg,omitempty"`
	Ch1NodeMemMax           float64   `json:"ch1_node_mem_max,omitempty"`
	Ch2NodeCpuMin           float64   `json:"ch2_node_cpu_min,omitempty"`
	Ch2NodeCpuAvg           float64   `json:"ch2_node_cpu_avg,omitempty"`
	Ch2NodeCpuMax           float64   `json:"ch2_node_cpu_max,omitempty"`
	Ch2NodeMemMin           float64   `json:"ch2_node_mem_min,omitempty"`
	Ch2NodeMemAvg           float64   `json:"ch2_node_mem_avg,omitempty"`
	Ch2NodeMemMax           float64   `json:"ch2_node_mem_max,omitempty"`
	Kafka1NodeCpuTotal      float64   `json:"kafka_1_node_cpu_total,omitempty"`
	Kafka1NodeMemTotal      float64   `json:"kafka_1_node_mem_total,omitempty"`
	Kafka2NodeCpuTotal      float64   `json:"kafka_2_node_cpu_total,omitempty"`
	Kafka2NodeMemTotal      float64   `json:"kafka_2_node_mem_total,omitempty"`
	Kafka3NodeCpuTotal      float64   `json:"kafka_3_node_cpu_total,omitempty"`
	Kafka3NodeMemTotal      float64   `json:"kafka_3_node_mem_total,omitempty"`
	Ch1NodeCpuTotal         float64   `json:"ch1_node_cpu_total,omitempty"`
	Ch1NodeMemTotal         float64   `json:"ch1_node_mem_total,omitempty"`
	Ch2NodeCpuTotal         float64   `json:"ch2_node_cpu_total,omitempty"`
	Ch2NodeMemTotal         float64   `json:"ch2_node_mem_total,omitempty"`
	KafkaClusterCpKafka0CpuMin float64 `json:"kafka_cluster_cp_kafka_0_cpu_min,omitempty"`
	KafkaClusterCpKafka0CpuAvg float64 `json:"kafka_cluster_cp_kafka_0_cpu_avg,omitempty"`
	KafkaClusterCpKafka0CpuMax float64 `json:"kafka_cluster_cp_kafka_0_cpu_max,omitempty"`
	KafkaClusterCpKafka0MemMin float64 `json:"kafka_cluster_cp_kafka_0_mem_min,omitempty"`
	KafkaClusterCpKafka0MemAvg float64 `json:"kafka_cluster_cp_kafka_0_mem_avg,omitempty"`
	KafkaClusterCpKafka0MemMax float64 `json:"kafka_cluster_cp_kafka_0_mem_max,omitempty"`
	KafkaClusterCpKafka1CpuMin float64 `json:"kafka_cluster_cp_kafka_1_cpu_min,omitempty"`
	KafkaClusterCpKafka1CpuAvg float64 `json:"kafka_cluster_cp_kafka_1_cpu_avg,omitempty"`
	KafkaClusterCpKafka1CpuMax float64 `json:"kafka_cluster_cp_kafka_1_cpu_max,omitempty"`
	KafkaClusterCpKafka1MemMin float64 `json:"kafka_cluster_cp_kafka_1_mem_min,omitempty"`
	KafkaClusterCpKafka1MemAvg float64 `json:"kafka_cluster_cp_kafka_1_mem_avg,omitempty"`
	KafkaClusterCpKafka1MemMax float64 `json:"kafka_cluster_cp_kafka_1_mem_max,omitempty"`
	KafkaClusterCpKafka2CpuMin float64 `json:"kafka_cluster_cp_kafka_2_cpu_min,omitempty"`
	KafkaClusterCpKafka2CpuAvg float64 `json:"kafka_cluster_cp_kafka_2_cpu_avg,omitempty"`
	KafkaClusterCpKafka2CpuMax float64 `json:"kafka_cluster_cp_kafka_2_cpu_max,omitempty"`
	KafkaClusterCpKafka2MemMin float64 `json:"kafka_cluster_cp_kafka_2_mem_min,omitempty"`
	KafkaClusterCpKafka2MemAvg float64 `json:"kafka_cluster_cp_kafka_2_mem_avg,omitempty"`
	KafkaClusterCpKafka2MemMax float64 `json:"kafka_cluster_cp_kafka_2_mem_max,omitempty"`
	ChiClickhouseVusmart000CpuMin float64 `json:"chi_clickhouse_vusmart_0_0_0_cpu_min,omitempty"`
	ChiClickhouseVusmart000CpuAvg float64 `json:"chi_clickhouse_vusmart_0_0_0_cpu_avg,omitempty"`
	ChiClickhouseVusmart000CpuMax float64 `json:"chi_clickhouse_vusmart_0_0_0_cpu_max,omitempty"`
	ChiClickhouseVusmart000MemMin float64 `json:"chi_clickhouse_vusmart_0_0_0_mem_min,omitempty"`
	ChiClickhouseVusmart000MemAvg float64 `json:"chi_clickhouse_vusmart_0_0_0_mem_avg,omitempty"`
	ChiClickhouseVusmart000MemMax float64 `json:"chi_clickhouse_vusmart_0_0_0_mem_max,omitempty"`
	ChiClickhouseVusmart010CpuMin float64 `json:"chi_clickhouse_vusmart_0_1_0_cpu_min,omitempty"`
	ChiClickhouseVusmart010CpuAvg float64 `json:"chi_clickhouse_vusmart_0_1_0_cpu_avg,omitempty"`
	ChiClickhouseVusmart010CpuMax float64 `json:"chi_clickhouse_vusmart_0_1_0_cpu_max,omitempty"`
	ChiClickhouseVusmart010MemMin float64 `json:"chi_clickhouse_vusmart_0_1_0_mem_min,omitempty"`
	ChiClickhouseVusmart010MemAvg float64 `json:"chi_clickhouse_vusmart_0_1_0_mem_avg,omitempty"`
	ChiClickhouseVusmart010MemMax float64 `json:"chi_clickhouse_vusmart_0_1_0_mem_max,omitempty"`
	PipelinePodCpuMin        float64   `json:"pipeline_pod_cpu_min,omitempty"`
	PipelinePodCpuAvg        float64   `json:"pipeline_pod_cpu_avg,omitempty"`
	PipelinePodCpuMax        float64   `json:"pipeline_pod_cpu_max,omitempty"`
	PipelinePodMemMin        float64   `json:"pipeline_pod_mem_min,omitempty"`
	PipelinePodMemAvg        float64   `json:"pipeline_pod_mem_avg,omitempty"`
	PipelinePodMemMax        float64   `json:"pipeline_pod_mem_max,omitempty"`
	MinInputMsgsPerSec       float64   `json:"min_input_msgs_per_sec,omitempty"`
	AvgInputMsgsPerSec       float64   `json:"avg_input_msgs_per_sec,omitempty"`
	MaxInputMsgsPerSec       float64   `json:"max_input_msgs_per_sec,omitempty"`
	MinOutputMsgsPerSec      float64   `json:"min_output_msgs_per_sec,omitempty"`
	AvgOutputMsgsPerSec      float64   `json:"avg_output_msgs_per_sec,omitempty"`
	MaxOutputMsgsPerSec      float64   `json:"max_output_msgs_per_sec,omitempty"`
	MinLag                   float64   `json:"min_lag,omitempty"`
	AvgLag                   float64   `json:"avg_lag,omitempty"`
	MaxLag                   float64   `json:"max_lag,omitempty"`
	DataLossPct              float64   `json:"data_loss_pct,omitempty"`
	KafkaSummaryGenerated    bool      `json:"kafka_summary_generated,omitempty"`
	PodResourceCheck         bool      `json:"pod_resource_check,omitempty"`
	ProcessRateSummary       string    `json:"process_rate_summary,omitempty"` // JSON summary of process rates per o11y source
	IngestionSummary         string    `json:"ingestion_summary,omitempty"` // JSON summary of hyperscale ingestion table-wise EPS data
	PipelineInfo             string    `json:"pipeline_info,omitempty"` // JSON mapping of o11y sources to pipeline details
	TraefikCpuAllocated      float64   `json:"traefik_cpu_allocated,omitempty"`
	TraefikMemAllocated      float64   `json:"traefik_mem_allocated,omitempty"`
}

// TestRunStartRequest represents the request payload for starting a test run
type TestRunStartRequest struct {
	TestName       string   `json:"test_name,omitempty"`
	TargetEPS      int      `json:"target_eps"`
	O11ySources    []string `json:"o11y_sources"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

// TestRunStartResponse represents the response for starting a test run
type TestRunStartResponse struct {
	Success bool     `json:"success"`
	Data    TestRun  `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// TestRunStopResponse represents the response for stopping a test run
type TestRunStopResponse struct {
	Success bool      `json:"success"`
	Data    TestRun   `json:"data"`
	Error   string    `json:"error,omitempty"`
}

// TestRunListResponse represents the response for listing test runs
type TestRunListResponse struct {
	Success bool       `json:"success"`
	Data    []*TestRun `json:"data"`
	Error   string     `json:"error,omitempty"`
}

// TestRunDetailResponse represents the response for getting a specific test run
type TestRunDetailResponse struct {
	Success bool     `json:"success"`
	Data    TestRun  `json:"data"`
	Error   string   `json:"error,omitempty"`
}

// NextTestIDResponse represents the response for getting the next test ID
type NextTestIDResponse struct {
	Success     bool `json:"success"`
	Data        NextTestIDData `json:"data"`
	Error       string `json:"error,omitempty"`
}

// NextTestIDData contains the next test ID information
type NextTestIDData struct {
	NextTestID string `json:"next_test_id"`
}