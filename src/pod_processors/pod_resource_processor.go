// Package pod_processors implements pod resource metrics summarization
package pod_processors

import (
	"context"
	"encoding/json"
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

// fetchPodMetrics fetches pod resource metrics from ClickHouse for the given time range
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
		WHERE (pod_name LIKE '%%kafka%%') OR (pod_name LIKE '%%click%%')
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
func ProcessPodResourceSummary(chClient *clickhouse.ClickHouseClient, start, end time.Time) (string, bool, error) {
	stats, err := FetchPodMetrics(chClient, start, end)
	if err != nil {
		return "", false, err
	}

	summary, found := ComputePodStats(stats)
	if !found {
		return "", false, nil
	}

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal pod summary: %w", err)
	}

	return string(summaryJSON), true, nil
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