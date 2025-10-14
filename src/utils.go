package main

import (
	"fmt"
	"time"

	"vuDataSim/src/logger"
)

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Background goroutine to collect real-time metrics from nodes via HTTP
func collectRealMetrics() {
	logger.LogWithNode("System", "Metrics", "Starting HTTP-based real metrics collection", "info")
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for range ticker.C {
		logger.LogWithNode("System", "Metrics", "HTTP metrics collection tick", "info")
		appState.mutex.Lock()

		// Load current node configurations
		err := nodeManager.LoadNodesConfig()
		if err != nil {
			logger.LogWarning("System", "Metrics", fmt.Sprintf("Failed to load nodes config for metrics collection: %v", err))
			appState.mutex.Unlock()
			continue
		}

		// Collect real metrics for each active node via HTTP
		for nodeID, node := range appState.NodeData {
			if node.Status == "active" {
				// Get node configuration for HTTP connection
				nodeConfig, exists := nodeManager.GetNodes()[nodeID]
				if !exists {
					logger.LogWarning(nodeID, "Metrics", "Node not found in configuration")
					node.Status = "error"
					continue
				}

				// Poll metrics via HTTP
				logger.LogWithNode(nodeID, "Metrics", "Polling HTTP metrics", "info")
				metrics, err := pollNodeMetrics(nodeConfig)
				if err != nil {
					logger.LogError(nodeID, "Metrics", fmt.Sprintf("Failed to get HTTP metrics: %v", err))
					node.Status = "error"
				} else {
					// Log the received metrics data
					logger.LogMetric(nodeID, "Metrics", fmt.Sprintf("CPU: %.1f%% (%d cores), Memory: %.1f%% (%.1f/%.1f GB), Load: %.2f",
						metrics.System.CPU.UsedPercent, metrics.System.CPU.Cores,
						metrics.System.Memory.UsedPercent, metrics.System.Memory.UsedGB, metrics.System.Memory.TotalGB,
						metrics.System.CPU.Load1M))

					logger.LogSuccess(nodeID, "Metrics", "Metrics collection successful")

					// Update node metrics with HTTP data (maintaining compatibility with frontend)
					node.CPU = metrics.System.CPU.UsedPercent
					node.Memory = metrics.System.Memory.UsedPercent
					node.TotalCPU = float64(metrics.System.CPU.Cores)
					node.TotalMemory = metrics.System.Memory.TotalGB
					node.Status = "active"

					// Update load metrics (keep existing simulation data for now)
					// In a real implementation, you might want to get these from the binary itself
					// For now, we'll maintain the existing demo values
				}

				node.LastUpdate = time.Now()
			}
		}

		// Broadcast updates to WebSocket clients
		go appState.broadcastUpdate()
		appState.mutex.Unlock()
	}
}
