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

/*

// pollNodeMetrics performs HTTP GET request to node's metrics endpoint
func pollNodeMetrics(nodeConfig NodeConfig) (*HTTPMetricsResponse, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Build metrics URL
	metricsURL := fmt.Sprintf("http://%s:%d/api/system/metrics", nodeConfig.Host, nodeConfig.MetricsPort)

	log.Printf("Making GET request to %s", metricsURL)

	// Make HTTP request
	resp, err := client.Get(metricsURL)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d", resp.StatusCode)

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Bad status: %d %s", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON response
	var metrics HTTPMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		log.Printf("JSON decode failed: %v", err)
		return nil, fmt.Errorf("JSON decode failed: %v", err)
	}

	log.Printf("Metrics response parsed successfully")
	return &metrics, nil
}

// Get local system total memory
func getLocalSystemMemory() (float64, error) {
	cmd := exec.Command("free", "-b")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute local free command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				bytes, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse memory bytes: %v", err)
				}
				// Convert bytes to GB
				return float64(bytes) / 1024 / 1024 / 1024, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find memory information in free output")
}

// Get local system total CPU cores
func getLocalSystemCPU() (float64, error) {
	cmd := exec.Command("nproc")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute local nproc command: %v", err)
	}

	cpuCores, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU cores: %v", err)
	}

	if cpuCores < 1 {
		return 4.0, nil // fallback to 4 cores if parsing fails
	}

	return cpuCores, nil
}

*/
