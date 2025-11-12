// Package pod_processors implements pod resource metrics summarization
package pod_processors

import (
	"context"
	
	"fmt"
	"time"

	"vuDataSim/src/clickhouse"
	"vuDataSim/src/logger"
)

// PodStat represents pod resource usage data from ClickHouse
type PodStat struct {
	PodName             string
	ContainerName       string
	UsedCPUCores        float64
	UsedMemoryGB        float64
	CPULimitCores       float64
	MemoryLimitGB       float64
}

// PodMetrics represents individual pod resource metrics for database columns
type PodMetrics struct {
	KafkaClusterCpKafka0CpuMin float64
	KafkaClusterCpKafka0CpuAvg float64
	KafkaClusterCpKafka0CpuMax float64
	KafkaClusterCpKafka0MemMin float64
	KafkaClusterCpKafka0MemAvg float64
	KafkaClusterCpKafka0MemMax float64
	KafkaClusterCpKafka1CpuMin float64
	KafkaClusterCpKafka1CpuAvg float64
	KafkaClusterCpKafka1CpuMax float64
	KafkaClusterCpKafka1MemMin float64
	KafkaClusterCpKafka1MemAvg float64
	KafkaClusterCpKafka1MemMax float64
	KafkaClusterCpKafka2CpuMin float64
	KafkaClusterCpKafka2CpuAvg float64
	KafkaClusterCpKafka2CpuMax float64
	KafkaClusterCpKafka2MemMin float64
	KafkaClusterCpKafka2MemAvg float64
	KafkaClusterCpKafka2MemMax float64
	ChiClickhouseVusmart000CpuMin float64
	ChiClickhouseVusmart000CpuAvg float64
	ChiClickhouseVusmart000CpuMax float64
	ChiClickhouseVusmart000MemMin float64
	ChiClickhouseVusmart000MemAvg float64
	ChiClickhouseVusmart000MemMax float64
	ChiClickhouseVusmart010CpuMin float64
	ChiClickhouseVusmart010CpuAvg float64
	ChiClickhouseVusmart010CpuMax float64
	ChiClickhouseVusmart010MemMin float64
	ChiClickhouseVusmart010MemAvg float64
	ChiClickhouseVusmart010MemMax float64
}

// fetchPodMetrics fetches pod resource metrics from ClickHouse for the given time range
// func FetchPodMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStat, error) {
// 	query := fmt.Sprintf(`
// 		SELECT
// 			pod_name,
// 			container_name,
// 			MAX(cpu_usage_nanocores) / 1000000000 AS used_cpu_cores,
// 			MAX(memory_usage_bytes) / 1073741824 AS used_memory_gb,
// 			MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit_cores,
// 			MAX(resource_limits_memory_bytes) / 1073741824 AS memory_limit_gb
// 		FROM monitoring.kubernetes_pod_container_data
// 		WHERE (pod_name LIKE '%%kafka%%') OR (pod_name LIKE '%%click%%')
// 		  AND timestamp >= toDateTime('%s')
// 		  AND timestamp <= toDateTime('%s')
// 		GROUP BY pod_name, container_name
// 		ORDER BY pod_name ASC
// 	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

// 	logger.LogWithNode("System", "PodFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

// 	rows, err := chClient.Client.Query(context.Background(), query)
// 	if err != nil {
// 		return nil, fmt.Errorf("query execution failed: %w", err)
// 	}
// 	defer rows.Close()

// 	var stats []PodStat
// 	for rows.Next() {
// 		var stat PodStat
// 		err := rows.Scan(&stat.PodName, &stat.ContainerName, &stat.UsedCPUCores, &stat.UsedMemoryGB, &stat.CPULimitCores, &stat.MemoryLimitGB)
// 		if err != nil {
// 			logger.LogWarning("System", "PodSummaryProcessor", fmt.Sprintf("Failed to scan pod metric row: %v", err))
// 			continue
// 		}
// 		stats = append(stats, stat)
// 	}

// 	if err := rows.Err(); err != nil {
// 		return nil, fmt.Errorf("row iteration error: %w", err)
// 	}

// 	if len(stats) == 0 {
// 		logger.LogWithNode("System", "PodFetcher", "No pod metrics found for given time range", "warn")
// 	} else {
// 		logger.LogSuccess("System", "PodFetcher", fmt.Sprintf("Fetched %d pod metric rows", len(stats)))
// 	}

// 	return stats, nil
// }

func FetchPodMetrics(chClient *clickhouse.ClickHouseClient, start, end time.Time) ([]PodStat, error) {
	query := fmt.Sprintf(`
		SELECT
			pod_name,
			container_name,
			MAX(cpu_usage_nanocores) / 1000000000 AS used_cpu_cores,
			MAX(memory_usage_bytes) / 1073741824 AS used_memory_gb,
			MAX(resource_limits_millicpu_units) / 1000 AS cpu_limit_cores,
			MAX(resource_limits_memory_bytes) / 1073741824 AS memory_limit_gb
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
		ORDER BY pod_name ASC
	`, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	logger.LogWithNode("System", "PodFetcher", fmt.Sprintf("Running ClickHouse query:\n%s", query), "debug")

	rows, err := chClient.Client.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var stats []PodStat
	for rows.Next() {
		var stat PodStat
		err := rows.Scan(&stat.PodName, &stat.ContainerName, &stat.UsedCPUCores, &stat.UsedMemoryGB, &stat.CPULimitCores, &stat.MemoryLimitGB)
		if err != nil {
			logger.LogWarning("System", "PodSummaryProcessor", fmt.Sprintf("Failed to scan pod metric row: %v", err))
			continue
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(stats) == 0 {
		logger.LogWithNode("System", "PodFetcher", "No pod metrics found for given time range", "warn")
	} else {
		logger.LogSuccess("System", "PodFetcher", fmt.Sprintf("Fetched %d pod metric rows", len(stats)))
	}

	return stats, nil
}



// ComputePodStats computes aggregated pod resource statistics
func ComputePodStats(stats []PodStat) (map[string]interface{}, bool) {
	if len(stats) == 0 {
		return nil, false
	}

	// Group by pod_name
	podGroups := make(map[string][]PodStat)
	for _, stat := range stats {
		podGroups[stat.PodName] = append(podGroups[stat.PodName], stat)
	}

	perPodData := make(map[string]interface{})

	for podName, podStats := range podGroups {
		var podTotalCPUCores, podTotalMemoryGB float64
		var cpuPercents, memoryPercents []float64

		for _, stat := range podStats {
			podTotalCPUCores += stat.CPULimitCores
			podTotalMemoryGB += stat.MemoryLimitGB

			if stat.CPULimitCores > 0 {
				cpuPercent := (stat.UsedCPUCores / stat.CPULimitCores) * 100
				cpuPercents = append(cpuPercents, cpuPercent)
			}
			if stat.MemoryLimitGB > 0 {
				memoryPercent := (stat.UsedMemoryGB / stat.MemoryLimitGB) * 100
				memoryPercents = append(memoryPercents, memoryPercent)
			}
		}

		// Per-pod data
		perPodData[podName] = map[string]interface{}{
			"total_allocated_cpu_cores":  podTotalCPUCores,
			"total_allocated_memory_gb":  podTotalMemoryGB,
			"max_cpu_percent":            maxSlice(cpuPercents),
			"min_cpu_percent":            minSlice(cpuPercents),
			"avg_cpu_percent":            avgSlice(cpuPercents),
			"max_memory_percent":         maxSlice(memoryPercents),
			"min_memory_percent":         minSlice(memoryPercents),
			"avg_memory_percent":         avgSlice(memoryPercents),
		}
	}

	return perPodData, true
}


// ProcessPodResourceSummary processes pod resource metrics for a test run
func ProcessPodResourceSummary(chClient *clickhouse.ClickHouseClient, start, end time.Time) (PodMetrics, bool, error) {
	stats, err := FetchPodMetrics(chClient, start, end)
	if err != nil {
		return PodMetrics{}, false, err
	}

	summary, found := ComputePodStats(stats)
	if !found {
		return PodMetrics{}, false, nil
	}

	// Extract metrics for specific pods
	metrics := PodMetrics{}

	if podData, exists := summary["kafka-cluster-cp-kafka-0"]; exists {
		if data, ok := podData.(map[string]interface{}); ok {
			metrics.KafkaClusterCpKafka0CpuMin = data["min_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0CpuAvg = data["avg_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0CpuMax = data["max_cpu_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemMin = data["min_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemAvg = data["avg_memory_percent"].(float64)
			metrics.KafkaClusterCpKafka0MemMax = data["max_memory_percent"].(float64)
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
		}
	}

	return metrics, true, nil
}

// Helper functions for slices
func maxSlice(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func minSlice(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func avgSlice(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}