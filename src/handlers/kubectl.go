package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"vuDataSim/src/logger"
)

// HandleAPIMonitoringK8PodYAML handles GET /api/monitoring/k8/pod-yaml
func HandleAPIMonitoringK8PodYAML(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")

	if namespace == "" || podName == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Both namespace and podName query parameters are required",
		})
		return
	}

	logger.LogWithNode("System", "kubectl", fmt.Sprintf("Fetching YAML for pod %s in namespace %s", podName, namespace), "info")

	// Start kubectl proxy and fetch YAML
	yamlContent, err := fetchPodYAML(namespace, podName)
	if err != nil {
		logger.LogWithNode("System", "kubectl", fmt.Sprintf("Failed to fetch YAML for pod %s: %v", podName, err), "error")
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch pod YAML: %v", err),
		})
		return
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Pod YAML retrieved successfully",
		Data:    yamlContent,
	})
}

// fetchPodYAML starts kubectl proxy and fetches pod YAML
func fetchPodYAML(namespace, podName string) (string, error) {
	// Create context with 2-minute timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start kubectl proxy in background
	cmd := exec.CommandContext(ctx, "kubectl", "proxy", "--port=8001")

	// Start the proxy
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start kubectl proxy: %v", err)
	}

	// Ensure proxy is killed when function returns
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait a moment for proxy to start
	time.Sleep(2 * time.Second)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Construct API URL
	apiURL := fmt.Sprintf("http://127.0.0.1:8001/api/v1/namespaces/%s/pods/%s", namespace, podName)

	// Make request to get pod YAML
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch pod data from API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}