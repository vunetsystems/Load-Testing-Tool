// Package pod_processors implements pod resource metrics summarization
package pod_processors

import (
	"context"
	"os"

	"fmt"
	"strings"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/database"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

// TopicConfig represents a single source configuration from topics_tables.yaml
type TopicConfig struct {
	Name     string   `yaml:"name"`
	Pipeline []string `yaml:"pipeline"`
}

// TopicsConfig represents the structure of topics_tables.yaml
type TopicsConfig struct {
	Sources []TopicConfig `yaml:"sources"`
}

// PodStat represents pod resource usage data from ClickHouse
// PodStat represents aggregated pod resource usage data
type PodStat struct {
	PodName       string
	ContainerName string // For pipeline pods
	MaxUsedCPU    float64
	MinUsedCPU    float64
	AvgUsedCPU    float64
	CPULimit      float64
	MaxUsedMemory float64
	MinUsedMemory float64
	AvgUsedMemory float64
	MemoryLimit   float64
	UsedCPUCores  float64 // For pipeline pods
	UsedMemoryGB  float64 // For pipeline pods
	CPULimitCores float64 // For pipeline pods
	MemoryLimitGB float64 // For pipeline pods
	Restarts      int64
}

// PodMetrics represents individual pod resource metrics for database columns
type PodMetrics struct {
	KafkaClusterCpKafka0CpuMin          float64
	KafkaClusterCpKafka0CpuAvg          float64
	KafkaClusterCpKafka0CpuMax          float64
	KafkaClusterCpKafka0MemMin          float64
	KafkaClusterCpKafka0MemAvg          float64
	KafkaClusterCpKafka0MemMax          float64
	KafkaClusterCpKafka1CpuMin          float64
	KafkaClusterCpKafka1CpuAvg          float64
	KafkaClusterCpKafka1CpuMax          float64
	KafkaClusterCpKafka1MemMin          float64
	KafkaClusterCpKafka1MemAvg          float64
	KafkaClusterCpKafka1MemMax          float64
	KafkaClusterCpKafka2CpuMin          float64
	KafkaClusterCpKafka2CpuAvg          float64
	KafkaClusterCpKafka2CpuMax          float64
	KafkaClusterCpKafka2MemMin          float64
	KafkaClusterCpKafka2MemAvg          float64
	KafkaClusterCpKafka2MemMax          float64
	ChiClickhouseVusmart000CpuMin       float64
	ChiClickhouseVusmart000CpuAvg       float64
	ChiClickhouseVusmart000CpuMax       float64
	ChiClickhouseVusmart000MemMin       float64
	ChiClickhouseVusmart000MemAvg       float64
	ChiClickhouseVusmart000MemMax       float64
	ChiClickhouseVusmart010CpuMin       float64
	ChiClickhouseVusmart010CpuAvg       float64
	ChiClickhouseVusmart010CpuMax       float64
	ChiClickhouseVusmart010MemMin       float64
	ChiClickhouseVusmart010MemAvg       float64
	ChiClickhouseVusmart010MemMax       float64
	PipelinePodCpuMin                   float64
	PipelinePodCpuAvg                   float64
	PipelinePodCpuMax                   float64
	PipelinePodMemMin                   float64
	PipelinePodMemAvg                   float64
	PipelinePodMemMax                   float64
	KafkaClusterCpKafka0CpuAllocated    float64
	KafkaClusterCpKafka0MemAllocated    float64
	KafkaClusterCpKafka1CpuAllocated    float64
	KafkaClusterCpKafka1MemAllocated    float64
	KafkaClusterCpKafka2CpuAllocated    float64
	KafkaClusterCpKafka2MemAllocated    float64
	ChiClickhouseVusmart000CpuAllocated float64
	ChiClickhouseVusmart000MemAllocated float64
	ChiClickhouseVusmart010CpuAllocated float64
	ChiClickhouseVusmart010MemAllocated float64
	PipelinePodCpuAllocated             float64
	PipelinePodMemAllocated             float64
	TraefikCpuAllocated                 float64
	TraefikMemAllocated                 float64
}

// PodStatWithNode represents pod resource usage data including node_name
type PodStatWithNode struct {
	PodName       string
	ContainerName string // For pipeline pods
	MaxUsedCPU    float64
	MinUsedCPU    float64
	AvgUsedCPU    float64
	CPULimit      float64
	MaxUsedMemory float64
	MinUsedMemory float64
	AvgUsedMemory float64
	MemoryLimit   float64
	UsedCPUCores  float64 // For pipeline pods
	UsedMemoryGB  float64 // For pipeline pods
	CPULimitCores float64 // For pipeline pods
	MemoryLimitGB float64 // For pipeline pods
	NodeName      string  // New field for node name
	Restarts      int64   // New field for restart count
}

func FetchPodMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStat, error) {
	query := fmt.Sprintf(`
WITH per_container AS (
    SELECT
        pod_name,
        container_name,

        -- Max values (include all)
        MAX(cpu_usage_nanocores) / 1000000000. AS max_cpu,
        MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
        MAX(memory_usage_bytes) / 1073741824 AS max_mem,
        MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit,

        -- Min and Avg values (ignore 0 or negative)
        MINIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS min_cpu,
        AVGIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS avg_cpu,
        MINIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS min_mem,
        AVGIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS avg_mem

    FROM monitoring.kubernetes_pod_container_data
    WHERE pod_name IN (
        'chi-clickhouse-vusmart-0-0-0',
        'chi-clickhouse-vusmart-0-1-0',
        'kafka-cluster-cp-kafka-0',
        'kafka-cluster-cp-kafka-1',
        'kafka-cluster-cp-kafka-2'
    )
      AND timestamp >= toDateTime('%s')
      AND timestamp <= toDateTime('%s')
    GROUP BY pod_name, container_name
)
SELECT
    pod_name,
    sum(max_cpu) AS max_used_cpu,
    sum(min_cpu) AS min_used_cpu,
    sum(avg_cpu) AS avg_used_cpu,
    sum(cpu_limit) AS cpu_limit,
    sum(max_mem) AS max_used_memory,
    sum(min_mem) AS min_used_memory,
    sum(avg_mem) AS avg_used_memory,
    sum(mem_limit) AS memory_limit
FROM per_container
GROUP BY pod_name
ORDER BY pod_name ASC
`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStat
	for rows.Next() {
		var stat PodStat
		if err := rows.Scan(
			&stat.PodName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pod metrics: %w", err)
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

func ComputePodStats(stats []PodStat) (map[string]interface{}, bool) {
	if len(stats) == 0 {
		return nil, false
	}

	perPodData := make(map[string]interface{})
	for _, s := range stats {
		calcPercent := func(value, limit float64) float64 {
			if limit == 0 {
				return 0
			}
			return (value / limit) * 100
		}

		perPodData[s.PodName] = map[string]interface{}{
			"max_cpu_percent":    calcPercent(s.MaxUsedCPU, s.CPULimit),
			"min_cpu_percent":    calcPercent(s.MinUsedCPU, s.CPULimit),
			"avg_cpu_percent":    calcPercent(s.AvgUsedCPU, s.CPULimit),
			"max_memory_percent": calcPercent(s.MaxUsedMemory, s.MemoryLimit),
			"min_memory_percent": calcPercent(s.MinUsedMemory, s.MemoryLimit),
			"avg_memory_percent": calcPercent(s.AvgUsedMemory, s.MemoryLimit),
			"cpu_allocated":      s.CPULimit,
			"mem_allocated":      s.MemoryLimit,
		}
	}
	return perPodData, true
}

// ProcessPodResourceSummary processes pod resource metrics for a test run
func ProcessPodResourceSummary(chClient *clickhouse.ClickHouseClient, testID string, start, end time.Time) (PodMetrics, bool, error) {
	stats, err := FetchPodMetrics(chClient, start, end)
	if err != nil {
		return PodMetrics{}, false, err
	}

	summary, found := ComputePodStats(stats)

	// Get test run to fetch o11y_sources
	testRun, err := database.GetTestRun(testID)
	if err != nil {
		logger.LogWarning("System", "PodResourceProcessor", fmt.Sprintf("Failed to get test run %s: %v", testID, err))
		// Continue without pipeline pods
	} else {
		// Load topics_tables.yaml
		yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
		if err != nil {
			logger.LogWarning("System", "PodResourceProcessor", fmt.Sprintf("Failed to read topics_tables.yaml: %v", err))
		} else {
			var cfg TopicsConfig
			err = yaml.Unmarshal(yamlData, &cfg)
			if err != nil {
				logger.LogWarning("System", "PodResourceProcessor", fmt.Sprintf("Failed to parse YAML: %v", err))
			} else {
				// Map o11y_sources to pipelines
				pipelines := make(map[string]bool)
				for _, src := range testRun.O11ySources {
					for _, s := range cfg.Sources {
						if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
							for _, p := range s.Pipeline {
								pipelines[p] = true
							}
							break
						}
					}
				}

				if len(pipelines) > 0 {
					// Fetch pipeline pod metrics
					pipelineStats, err := FetchPipelinePodMetrics(chClient, pipelines, start, end)
					if err != nil {
						logger.LogWarning("System", "PodResourceProcessor", fmt.Sprintf("Failed to fetch pipeline pod metrics: %v", err))
					} else {
						// Compute aggregated metrics for pipeline pods
						pipelineSummary := ComputePipelinePodStats(pipelineStats)
						// Add to summary
						for k, v := range pipelineSummary {
							summary[k] = v
						}
						// Update found if pipeline has data
						if len(pipelineSummary) > 0 {
							found = true
						}
					}
				}
			}
		}
	}

	// Extract metrics for specific pods
	metrics := PodMetrics{}

	// Fetch traefik pod metrics
	traefikStats, err := FetchTraefikPodMetrics(chClient, start, end)
	if err != nil {
		logger.LogWarning("System", "PodResourceProcessor", fmt.Sprintf("Failed to fetch traefik pod metrics: %v", err))
	} else {
		traefikCpuAllocated, traefikMemAllocated := ComputeTraefikPodStats(traefikStats)
		metrics.TraefikCpuAllocated = traefikCpuAllocated
		metrics.TraefikMemAllocated = traefikMemAllocated
		// Update found flag if Traefik has allocated resources
		if traefikCpuAllocated > 0 || traefikMemAllocated > 0 {
			found = true
		}
	}

	// If no metrics found at all, return false
	if !found {
		return PodMetrics{}, false, nil
	}

	if podData, exists := summary["kafka-cluster-cp-kafka-0"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.KafkaClusterCpKafka0CpuMin = data["min_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0CpuMax = data["max_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemMin = data["min_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemAvg = data["avg_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemMax = data["max_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka0CpuAllocated = data["cpu_allocated"].(float64)
			metrics.KafkaClusterCpKafka0MemAllocated = data["mem_allocated"].(float64)
		}
	}

	if podData, exists := summary["kafka-cluster-cp-kafka-1"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.KafkaClusterCpKafka1CpuMin = data["min_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka1CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka1CpuMax = data["max_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka1MemMin = data["min_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka1MemAvg = data["avg_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka1MemMax = data["max_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka1CpuAllocated = data["cpu_allocated"].(float64)
			metrics.KafkaClusterCpKafka1MemAllocated = data["mem_allocated"].(float64)
		}
	}

	if podData, exists := summary["kafka-cluster-cp-kafka-2"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.KafkaClusterCpKafka2CpuMin = data["min_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka2CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka2CpuMax = data["max_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka2MemMin = data["min_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka2MemAvg = data["avg_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka2MemMax = data["max_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka2CpuAllocated = data["cpu_allocated"].(float64)
			metrics.KafkaClusterCpKafka2MemAllocated = data["mem_allocated"].(float64)
		}
	}

	if podData, exists := summary["chi-clickhouse-vusmart-0-0-0"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.ChiClickhouseVusmart000CpuMin = data["min_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart000CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart000CpuMax = data["max_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart000MemMin = data["min_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart000MemAvg = data["avg_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart000MemMax = data["max_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart000CpuAllocated = data["cpu_allocated"].(float64)
			metrics.ChiClickhouseVusmart000MemAllocated = data["mem_allocated"].(float64)
		}
	}

	if podData, exists := summary["chi-clickhouse-vusmart-0-1-0"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.ChiClickhouseVusmart010CpuMin = data["min_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart010CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart010CpuMax = data["max_cpu_percent"].(float64)
			metrics.ChiClickhouseVusmart010MemMin = data["min_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart010MemAvg = data["avg_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart010MemMax = data["max_memory_percent"].(float64)
			metrics.ChiClickhouseVusmart010CpuAllocated = data["cpu_allocated"].(float64)
			metrics.ChiClickhouseVusmart010MemAllocated = data["mem_allocated"].(float64)
		}
	}

	// Extract pipeline pod metrics
	if pipelineData, exists := summary["pipeline_pod_cpu_min"]; exists {
		metrics.PipelinePodCpuMin = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_cpu_avg"]; exists {
		metrics.PipelinePodCpuAvg = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_cpu_max"]; exists {
		metrics.PipelinePodCpuMax = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_mem_min"]; exists {
		metrics.PipelinePodMemMin = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_mem_avg"]; exists {
		metrics.PipelinePodMemAvg = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_mem_max"]; exists {
		metrics.PipelinePodMemMax = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_cpu_allocated"]; exists {
		metrics.PipelinePodCpuAllocated = pipelineData.(float64)
	}
	if pipelineData, exists := summary["pipeline_pod_mem_allocated"]; exists {
		metrics.PipelinePodMemAllocated = pipelineData.(float64)
	}

	return metrics, true, nil
}

// // Helper functions for slices
// func maxSlice(vals []float64) float64 {
// 	if len(vals) == 0 {
// 		return 0
// 	}
// 	max := vals[0]
// 	for _, v := range vals[1:] {
// 		if v > max {
// 			max = v
// 		}
// 	}
// 	return max
// }

// func minSlice(vals []float64) float64 {
// 	if len(vals) == 0 {
// 		return 0
// 	}
// 	min := vals[0]
// 	for _, v := range vals[1:] {
// 		if v < min {
// 			min = v
// 		}
// 	}
// 	return min
// }

// func avgSlice(vals []float64) float64 {
// 	if len(vals) == 0 {
// 		return 0
// 	}
// 	sum := 0.0
// 	for _, v := range vals {
// 		sum += v
// 	}
// 	return sum / float64(len(vals))
// }

func FetchPipelinePodMetricsWithNode(chClient *clickhouse.ClickHouseClient, pipelines map[string]bool, start, end time.Time) ([]PodStatWithNode, error) {
	if len(pipelines) == 0 {
		return nil, nil
	}

	// Build the LIKE conditions for pipelines
	conditions := make([]string, 0, len(pipelines))
	for p := range pipelines {
		conditions = append(conditions, fmt.Sprintf("pod_name LIKE '%s%%'", p))
	}
	pipelineCondition := strings.Join(conditions, " OR ")

	query := fmt.Sprintf(`
WITH per_container AS (
    SELECT
        pod_name,
        container_name,
        node_name,

        -- Max values (include all)
        MAX(cpu_usage_nanocores) / 1000000000. AS max_cpu,
        MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
        MAX(memory_working_set_bytes) / 1073741824 AS max_mem,
        MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit,
        -- Min and Avg values (ignore 0 or negative)
        MINIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS min_cpu,
        AVGIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS avg_cpu,
        MINIf(memory_working_set_bytes / 1073741824, memory_working_set_bytes > 0) AS min_mem,
        AVGIf(memory_working_set_bytes / 1073741824, memory_working_set_bytes > 0) AS avg_mem

    FROM monitoring.kubernetes_pod_container_data
    WHERE (%s)
      AND pod_name NOT LIKE '%%debug%%'
      AND timestamp >= toDateTime('%s')
      AND timestamp <= toDateTime('%s')
    GROUP BY pod_name, container_name, node_name
)
SELECT
    pod_name,
    node_name,
    toFloat64(sum(max_cpu)) AS max_used_cpu,
    toFloat64(sum(min_cpu)) AS min_used_cpu,
    toFloat64(sum(avg_cpu)) AS avg_used_cpu,
    toFloat64(sum(cpu_limit)) AS cpu_limit,
    toFloat64(sum(max_mem)) AS max_used_memory,
    toFloat64(sum(min_mem)) AS min_used_memory,
    toFloat64(sum(avg_mem)) AS avg_used_memory,
    toFloat64(sum(mem_limit)) AS memory_limit
FROM per_container
GROUP BY pod_name, node_name
ORDER BY pod_name ASC
`, pipelineCondition, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	logger.LogWithNode("System", "PipelinePodFetcherWithNode", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStatWithNode
	for rows.Next() {
		var stat PodStatWithNode
		if err := rows.Scan(
			&stat.PodName,
			&stat.NodeName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline pod metrics with node: %w", err)
		}
		logger.LogWithNode("System", "PipelinePodFetcherWithNode", fmt.Sprintf("Scanned pod: %s, CPULimit: %f, MinCPU: %f, AvgCPU: %f, MaxCPU: %f, MemLimit: %f, MinMem: %f, AvgMem: %f, MaxMem: %f",
			stat.PodName, stat.CPULimit, stat.MinUsedCPU, stat.AvgUsedCPU, stat.MaxUsedCPU,
			stat.MemoryLimit, stat.MinUsedMemory, stat.AvgUsedMemory, stat.MaxUsedMemory), "debug")
		stats = append(stats, stat)
	}

	restartMap, err := FetchPodRestarts(chClient, pipelineCondition, start, end)
	if err != nil {
		logger.LogWarning("System", "FetchPipelinePodMetricsWithNode", fmt.Sprintf("Failed to fetch pod restarts: %v", err))
	}
	for i := range stats {
		if totalRestarts, ok := restartMap[stats[i].PodName]; ok {
			stats[i].Restarts = totalRestarts
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "PipelinePodFetcherWithNode", "No pipeline pod metrics found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "PipelinePodFetcherWithNode", fmt.Sprintf("Fetched %d pipeline pod metric rows", len(stats)))
	}

	return stats, nil
}

func FetchPipelinePodMetrics(chClient *clickhouse.ClickHouseClient, pipelines map[string]bool, start, end time.Time) ([]PodStat, error) {
	if len(pipelines) == 0 {
		return nil, nil
	}

	// Build the LIKE conditions for pipelines
	conditions := make([]string, 0, len(pipelines))
	for p := range pipelines {
		conditions = append(conditions, fmt.Sprintf("pod_name LIKE '%s%%'", p))
	}
	pipelineCondition := strings.Join(conditions, " OR ")

	query := fmt.Sprintf(`
WITH per_container AS (
    SELECT
        pod_name,
        container_name,

        -- Max values (include all)
        MAX(cpu_usage_nanocores) / 1000000000. AS max_cpu,
        MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
        MAX(memory_usage_bytes) / 1073741824 AS max_mem,
        MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit,

        -- Min and Avg values (ignore 0 or negative)
        MINIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS min_cpu,
        AVGIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS avg_cpu,
        MINIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS min_mem,
        AVGIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS avg_mem

    FROM monitoring.kubernetes_pod_container_data
    WHERE (%s)
      AND pod_name NOT LIKE '%%debug%%'
      AND timestamp >= toDateTime('%s')
      AND timestamp <= toDateTime('%s')
    GROUP BY pod_name, container_name
)
SELECT
    pod_name,
    sum(max_cpu) AS max_used_cpu,
    sum(min_cpu) AS min_used_cpu,
    sum(avg_cpu) AS avg_used_cpu,
    sum(cpu_limit) AS cpu_limit,
    sum(max_mem) AS max_used_memory,
    sum(min_mem) AS min_used_memory,
    sum(avg_mem) AS avg_used_memory,
    sum(mem_limit) AS memory_limit
FROM per_container
GROUP BY pod_name
ORDER BY pod_name ASC
`, pipelineCondition, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	logger.LogWithNode("System", "PipelinePodFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStat
	for rows.Next() {
		var stat PodStat
		if err := rows.Scan(
			&stat.PodName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline pod metrics: %w", err)
		}
		stats = append(stats, stat)
	}

	restartMap, err := FetchPodRestarts(chClient, "pod_name LIKE 'traefik-%%'", start, end)
	if err != nil {
		logger.LogWarning("System", "FetchPipelinePodMetrics", fmt.Sprintf("Failed to fetch pod restarts: %v", err))
	}
	for i := range stats {
		if totalRestarts, ok := restartMap[stats[i].PodName]; ok {
			stats[i].Restarts = totalRestarts
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "PipelinePodFetcher", "No pipeline pod metrics found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "PipelinePodFetcher", fmt.Sprintf("Fetched %d pipeline pod metric rows", len(stats)))
	}

	return stats, nil
}

func FetchTraefikPodMetricsWithNode(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStatWithNode, error) {
	query := fmt.Sprintf(`
WITH per_container AS (
	   SELECT
	       pod_name,
	       container_name,
	       node_name,

	       -- Max values (include all)
	       MAX(cpu_usage_nanocores) / 1000000000. AS max_cpu,
	       MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
	       MAX(memory_usage_bytes) / 1073741824 AS max_mem,
	       MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit,
	       MAX(restarts_total) AS last_restart,
	       -- Min and Avg values (ignore 0 or negative)
	       MINIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS min_cpu,
	       AVGIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS avg_cpu,
	       MINIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS min_mem,
	       AVGIf(memory_usage_bytes / 1073741824, memory_usage_bytes > 0) AS avg_mem

	   FROM monitoring.kubernetes_pod_container_data
	   WHERE pod_name LIKE 'traefik-%%'
	     AND timestamp >= toDateTime('%s')
	     AND timestamp <= toDateTime('%s')
	   GROUP BY pod_name, container_name, node_name
)
SELECT
	   pod_name,
	   node_name,
	   toFloat64(sum(max_cpu)) AS max_used_cpu,
	   toFloat64(sum(min_cpu)) AS min_used_cpu,
	   toFloat64(sum(avg_cpu)) AS avg_used_cpu,
	   toFloat64(sum(cpu_limit)) AS cpu_limit,
	   toFloat64(sum(max_mem)) AS max_used_memory,
	   toFloat64(sum(min_mem)) AS min_used_memory,
	   toFloat64(sum(avg_mem)) AS avg_used_memory,
	   toFloat64(sum(mem_limit)) AS memory_limit,
	   toInt64(MAX(last_restart) - MIN(last_restart)) AS restarts
FROM per_container
GROUP BY pod_name, node_name
ORDER BY pod_name ASC
`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStatWithNode
	for rows.Next() {
		var stat PodStatWithNode
		if err := rows.Scan(
			&stat.PodName,
			&stat.NodeName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
			&stat.Restarts,
		); err != nil {
			return nil, fmt.Errorf("failed to scan traefik pod metrics with node: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "TraefikPodFetcherWithNode", "No traefik pod metrics found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "TraefikPodFetcherWithNode", fmt.Sprintf("Fetched %d traefik pod metric rows", len(stats)))
	}

	return stats, nil
}

func FetchPodRestarts(chClient *clickhouse.ClickHouseClient, podConditions string, start, end time.Time) (map[string]int64, error) {
	query := fmt.Sprintf(`
WITH
    raw AS (
        SELECT
            pod_name,
            toStartOfMinute(timestamp) AS minute_bucket,
            restarts_total
        FROM monitoring.kubernetes_pod_container_data
        WHERE (%s)
          AND timestamp >= toDateTime('%s')
          AND timestamp <= toDateTime('%s')
    ),

    per_minute AS (
        SELECT
            pod_name,
            minute_bucket,
            max(restarts_total) AS restarts_upper_bound
        FROM raw
        GROUP BY pod_name, minute_bucket
    ),

    with_prev AS (
        SELECT
            pod_name,
            minute_bucket,
            restarts_upper_bound,
            lagInFrame(restarts_upper_bound, 1) OVER (PARTITION BY pod_name ORDER BY minute_bucket ASC) AS prev_restarts,
            row_number() OVER (PARTITION BY pod_name ORDER BY minute_bucket ASC) AS rn
        FROM per_minute
    ),

    with_increment AS (
        SELECT
            pod_name,
            if(rn = 1, restarts_upper_bound, prev_restarts) AS prev_restarts,
            restarts_upper_bound - if(rn = 1, restarts_upper_bound, prev_restarts) AS increment_this_minute
        FROM with_prev
    )

SELECT
    pod_name,
    sum(increment_this_minute) AS total_restarts
FROM with_increment
GROUP BY pod_name
ORDER BY pod_name ASC
`, podConditions, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	restartMap := make(map[string]int64)
	for rows.Next() {
		var podName string
		var totalRestarts int64
		if err := rows.Scan(&podName, &totalRestarts); err != nil {
			return nil, fmt.Errorf("failed to scan restart data: %w", err)
		}
		restartMap[podName] = totalRestarts
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return restartMap, nil
}

// ComputePodJSONMetrics computes pods_cpu and pods_memory as JSON-serializable maps
func FetchTraefikPodMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStat, error) {
	query := fmt.Sprintf(`
WITH per_container AS (
	   SELECT
	       pod_name,
	       container_name,
	       MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
	       MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit
	   FROM monitoring.kubernetes_pod_container_data
	   WHERE pod_name LIKE 'traefik-%%'
	     AND timestamp >= toDateTime('%s')
	     AND timestamp <= toDateTime('%s')
	   GROUP BY pod_name, container_name
)
SELECT
	   pod_name,
	   toFloat64(0) AS max_used_cpu,
	   toFloat64(0) AS min_used_cpu,
	   toFloat64(0) AS avg_used_cpu,
	   sum(cpu_limit) AS cpu_limit,
	   toFloat64(0) AS max_used_memory,
	   toFloat64(0) AS min_used_memory,
	   toFloat64(0) AS avg_used_memory,
	   sum(mem_limit) AS memory_limit
FROM per_container
GROUP BY pod_name
ORDER BY pod_name ASC
`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStat
	for rows.Next() {
		var stat PodStat
		if err := rows.Scan(
			&stat.PodName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan traefik pod metrics: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "TraefikPodFetcher", "No traefik pod metrics found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "TraefikPodFetcher", fmt.Sprintf("Fetched %d traefik pod metric rows", len(stats)))
	}

	return stats, nil
}

func ComputePipelinePodStats(stats []PodStat) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"pipeline_pod_cpu_min": 0.0,
			"pipeline_pod_cpu_avg": 0.0,
			"pipeline_pod_cpu_max": 0.0,
			"pipeline_pod_mem_min": 0.0,
			"pipeline_pod_mem_avg": 0.0,
			"pipeline_pod_mem_max": 0.0,
		}
	}

	var sumMaxCPU, sumMinCPU, sumAvgCPU, sumCPULimit float64
	var sumMaxMem, sumMinMem, sumAvgMem, sumMemLimit float64

	for _, stat := range stats {
		sumMaxCPU += stat.MaxUsedCPU
		sumMinCPU += stat.MinUsedCPU
		sumAvgCPU += stat.AvgUsedCPU
		sumCPULimit += stat.CPULimit

		sumMaxMem += stat.MaxUsedMemory
		sumMinMem += stat.MinUsedMemory
		sumAvgMem += stat.AvgUsedMemory
		sumMemLimit += stat.MemoryLimit
	}

	calcPercent := func(value, limit float64) float64 {
		if limit == 0 {
			return 0
		}
		return (value / limit) * 100
	}

	return map[string]interface{}{
		"pipeline_pod_cpu_min":       calcPercent(sumMinCPU, sumCPULimit),
		"pipeline_pod_cpu_avg":       calcPercent(sumAvgCPU, sumCPULimit),
		"pipeline_pod_cpu_max":       calcPercent(sumMaxCPU, sumCPULimit),
		"pipeline_pod_mem_min":       calcPercent(sumMinMem, sumMemLimit),
		"pipeline_pod_mem_avg":       calcPercent(sumAvgMem, sumMemLimit),
		"pipeline_pod_mem_max":       calcPercent(sumMaxMem, sumMemLimit),
		"pipeline_pod_cpu_allocated": sumCPULimit,
		"pipeline_pod_mem_allocated": sumMemLimit,
	}
}

func ComputeTraefikPodStats(stats []PodStat) (float64, float64) {
	var totalCpu, totalMem float64
	for _, stat := range stats {
		totalCpu += stat.CPULimit
		totalMem += stat.MemoryLimit
	}
	return totalCpu, totalMem
}

func FetchPodMetricsWithNode(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStatWithNode, error) {
	query := fmt.Sprintf(`
WITH per_container AS (
    SELECT
        pod_name,
        container_name,
        node_name,

        -- Max values (include all)
        MAX(cpu_usage_nanocores) / 1000000000. AS max_cpu,
        MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit,
        MAX(memory_working_set_bytes) / 1073741824 AS max_mem,
        MAX(resource_limits_memory_bytes) / 1073741824 AS mem_limit,
        -- Min and Avg values (ignore 0 or negative)
        MINIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS min_cpu,
        AVGIf(cpu_usage_nanocores / 1000000000., cpu_usage_nanocores > 0) AS avg_cpu,
        MINIf(memory_working_set_bytes / 1073741824, memory_working_set_bytes > 0) AS min_mem,
        AVGIf(memory_working_set_bytes / 1073741824, memory_working_set_bytes > 0) AS avg_mem

    FROM monitoring.kubernetes_pod_container_data
    WHERE pod_name IN (
        'chi-clickhouse-vusmart-0-0-0',
        'chi-clickhouse-vusmart-0-1-0',
        'kafka-cluster-cp-kafka-0',
        'kafka-cluster-cp-kafka-1',
        'kafka-cluster-cp-kafka-2'
    )
      AND timestamp >= toDateTime('%s')
      AND timestamp <= toDateTime('%s')
    GROUP BY pod_name, container_name, node_name
)
SELECT
    pod_name,
    node_name,
    toFloat64(sum(max_cpu)) AS max_used_cpu,
    toFloat64(sum(min_cpu)) AS min_used_cpu,
    toFloat64(sum(avg_cpu)) AS avg_used_cpu,
    toFloat64(sum(cpu_limit)) AS cpu_limit,
    toFloat64(sum(max_mem)) AS max_used_memory,
    toFloat64(sum(min_mem)) AS min_used_memory,
    toFloat64(sum(avg_mem)) AS avg_used_memory,
    toFloat64(sum(mem_limit)) AS memory_limit
FROM per_container
GROUP BY pod_name, node_name
ORDER BY pod_name ASC
`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStatWithNode
	for rows.Next() {
		var stat PodStatWithNode
		if err := rows.Scan(
			&stat.PodName,
			&stat.NodeName,
			&stat.MaxUsedCPU,
			&stat.MinUsedCPU,
			&stat.AvgUsedCPU,
			&stat.CPULimit,
			&stat.MaxUsedMemory,
			&stat.MinUsedMemory,
			&stat.AvgUsedMemory,
			&stat.MemoryLimit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pod metrics with node: %w", err)
		}
		stats = append(stats, stat)
	}

	restartMap, err := FetchPodRestarts(chClient, "pod_name IN ('chi-clickhouse-vusmart-0-0-0', 'chi-clickhouse-vusmart-0-1-0', 'kafka-cluster-cp-kafka-0', 'kafka-cluster-cp-kafka-1', 'kafka-cluster-cp-kafka-2')", start, end)
	if err != nil {
		logger.LogWarning("System", "FetchPodMetricsWithNode", fmt.Sprintf("Failed to fetch pod restarts: %v", err))
	}
	for i := range stats {
		if totalRestarts, ok := restartMap[stats[i].PodName]; ok {
			stats[i].Restarts = totalRestarts
		}
	}

	return stats, nil
}

// ComputePodJSONMetrics computes pods_cpu and pods_memory as JSON-serializable maps
func ComputePodJSONMetrics(stats []PodStatWithNode) (map[string]interface{}, map[string]interface{}, bool) {
	if len(stats) == 0 {
		return nil, nil, false
	}

	podsCPU := make(map[string]interface{})
	podsMemory := make(map[string]interface{})

	calcPercent := func(value, limit float64) float64 {
		if limit == 0 {
			return 0
		}
		return (value / limit) * 100
	}

	for _, s := range stats {
		// CPU metrics with node_name
		podsCPU[s.PodName] = map[string]interface{}{
			"allocated": s.CPULimit,
			"min":       calcPercent(s.MinUsedCPU, s.CPULimit),
			"avg":       calcPercent(s.AvgUsedCPU, s.CPULimit),
			"max":       calcPercent(s.MaxUsedCPU, s.CPULimit),
			"node_name": s.NodeName,
		}

		// Memory metrics with node_name
		podsMemory[s.PodName] = map[string]interface{}{
			"allocated": s.MemoryLimit,
			"min":       calcPercent(s.MinUsedMemory, s.MemoryLimit),
			"avg":       calcPercent(s.AvgUsedMemory, s.MemoryLimit),
			"max":       calcPercent(s.MaxUsedMemory, s.MemoryLimit),
			"node_name": s.NodeName,
		}
	}

	return podsCPU, podsMemory, true
}

// ComputePodRestartsJSON computes pod_restarts as JSON-serializable map
func ComputePodRestartsJSON(stats []PodStatWithNode) (map[string]interface{}, bool) {
	if len(stats) == 0 {
		return nil, false
	}

	podRestarts := make(map[string]interface{})

	for _, s := range stats {
		podRestarts[s.PodName] = s.Restarts
	}

	return podRestarts, true
}

// ProcessPodResourceSummaryWithNode processes pod resource metrics with node_name and returns JSON columns
func ProcessPodResourceSummaryWithNode(chClient *clickhouse.ClickHouseClient, testID string, start, end time.Time) (map[string]interface{}, map[string]interface{}, map[string]interface{}, bool, error) {
	coreStats, err := FetchPodMetricsWithNode(chClient, start, end)
	if err != nil {
		return nil, nil, nil, false, err
	}
	logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Fetched %d core pod stats for test %s", len(coreStats), testID), "info")

	// Get o11y_sources for pipeline pods
	o11ySources, err := database.GetTestRunO11ySources(testID)
	if err != nil {
		logger.LogWarning("System", "PodResourceProcessorWithNode", fmt.Sprintf("Failed to get o11y_sources for test run %s: %v", testID, err))
		// Continue without pipeline pods
	} else {
		logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Retrieved o11y_sources: %v for test %s", o11ySources, testID), "info")
		// Load topics_tables.yaml
		yamlData, err := os.ReadFile("src/configs/topics_tables.yaml")
		if err != nil {
			logger.LogWarning("System", "PodResourceProcessorWithNode", fmt.Sprintf("Failed to read topics_tables.yaml: %v", err))
		} else {
			var cfg TopicsConfig
			err = yaml.Unmarshal(yamlData, &cfg)
			if err != nil {
				logger.LogWarning("System", "PodResourceProcessorWithNode", fmt.Sprintf("Failed to parse YAML: %v", err))
			} else {
				// Map o11y_sources to pipelines
				pipelines := make(map[string]bool)
				for _, src := range o11ySources {
					for _, s := range cfg.Sources {
						if strings.ToLower(strings.ReplaceAll(s.Name, " ", "")) == strings.ToLower(strings.ReplaceAll(src, " ", "")) {
							for _, p := range s.Pipeline {
								pipelines[p] = true
							}
							break
						}
					}
				}
				logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Mapped pipelines: %v for test %s", pipelines, testID), "info")

				if len(pipelines) > 0 {
					// Fetch pipeline pod metrics with node
					pipelineStats, err := FetchPipelinePodMetricsWithNode(chClient, pipelines, start, end)
					if err != nil {
						logger.LogWarning("System", "PodResourceProcessorWithNode", fmt.Sprintf("Failed to fetch pipeline pod metrics with node: %v", err))
					} else {
						logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Fetched %d pipeline pod stats for test %s", len(pipelineStats), testID), "info")
						// Merge pipeline stats with core stats for JSON computation
						coreStats = append(coreStats, pipelineStats...)
					}
				}
			}
		}
	}

	// Fetch Traefik pod metrics with node
	traefikStats, err := FetchTraefikPodMetricsWithNode(chClient, start, end)
	if err != nil {
		logger.LogWarning("System", "PodResourceProcessorWithNode", fmt.Sprintf("Failed to fetch traefik pod metrics with node: %v", err))
	} else {
		logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Fetched %d traefik pod stats for test %s", len(traefikStats), testID), "info")
		// Merge traefik stats with core stats for JSON computation
		coreStats = append(coreStats, traefikStats...)
	}

	logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("Total coreStats after merging: %d for test %s", len(coreStats), testID), "info")

	podsCPU, podsMemory, found := ComputePodJSONMetrics(coreStats)
	podRestarts, _ := ComputePodRestartsJSON(coreStats)
	logger.LogWithNode("System", "PodResourceProcessorWithNode", fmt.Sprintf("ComputePodJSONMetrics found: %t, podsCPU keys: %d, podsMemory keys: %d for test %s", found, len(podsCPU), len(podsMemory), testID), "info")
	if !found {
		return nil, nil, nil, false, nil
	}

	return podsCPU, podsMemory, podRestarts, true, nil
}
