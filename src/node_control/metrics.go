package node_control

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"vuDataSim/src/logger"

	"time"
)

func (nm *NodeManager) verifyMetricsServer(nodeConfig NodeConfig) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Build health check URL
	healthURL := fmt.Sprintf("http://%s:%d/api/system/health", nodeConfig.Host, nodeConfig.MetricsPort)

	// Make HTTP request
	resp, err := client.Get(healthURL)
	if err != nil {
		return fmt.Errorf("HTTP request to metrics server failed: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metrics server returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON response to verify it's our metrics server
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(body, &healthResponse); err != nil {
		return fmt.Errorf("failed to parse health response JSON: %v", err)
	}

	// Verify expected fields
	if status, ok := healthResponse["status"].(string); !ok || status != "healthy" {
		return fmt.Errorf("unexpected health status: %v", status)
	}

	if nodeID, ok := healthResponse["nodeId"].(string); !ok || nodeID == "" {
		return fmt.Errorf("missing or invalid nodeId in health response")
	}

	logger.LogSuccess(nodeConfig.Host, "node_control", "Metrics server health check successful")
	return nil
}
