package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"
)

// Get real CPU usage from node via SSH
func getNodeCPUUsage(nodeConfig node_control.NodeConfig) (float64, error) {
	output, err := executeCPUCommand(nodeConfig)
	if err != nil {
		return 0, err
	}

	cpuUsage, err := parseCPUUsage(output)
	if err != nil {
		return 0, err
	}

	return validateAndClampCPUUsage(cpuUsage), nil
}

// executeCPUCommand executes CPU monitoring command with fallback strategy
func executeCPUCommand(nodeConfig node_control.NodeConfig) (string, error) {
	// Use 'vmstat' for more reliable CPU metrics
	vmstatCmd := "vmstat 1 2 | tail -1 | awk '{print $13}'"
	output, err := sshExec(nodeConfig, vmstatCmd)
	if err == nil {
		return output, nil
	}

	// Fallback to top command if vmstat fails
	topCmd := "top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'"
	output, err = sshExec(nodeConfig, topCmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute CPU command: %v", err)
	}

	return output, nil
}

// parseCPUUsage parses CPU usage from command output
func parseCPUUsage(output string) (float64, error) {
	cleanOutput := strings.TrimSpace(output)

	if strings.Contains(cleanOutput, "%") {
		return parseCPUFromTopOutput(cleanOutput)
	}
	return parseCPUFromVmstatOutput(cleanOutput)
}

// parseCPUFromTopOutput parses CPU usage from top command output (contains % symbol)
func parseCPUFromTopOutput(output string) (float64, error) {
	cpuUsage, err := strconv.ParseFloat(strings.TrimSuffix(output, "%"), 64)
	if err == nil {
		return cpuUsage, nil
	}

	// Try regex extraction as fallback
	return extractNumericValue(output, "CPU usage")
}

// parseCPUFromVmstatOutput parses idle CPU from vmstat output (no % symbol)
func parseCPUFromVmstatOutput(output string) (float64, error) {
	idle, err := strconv.ParseFloat(output, 64)
	if err == nil {
		return 100 - idle, nil // Convert idle % to usage %
	}

	// Try regex extraction as fallback
	idle, err = extractNumericValue(output, "idle CPU")
	if err != nil {
		return 0, err
	}

	return 100 - idle, nil
}

// extractNumericValue extracts numeric value using regex fallback
func extractNumericValue(output, valueType string) (float64, error) {
	re := regexp.MustCompile(`\d+\.?\d*`)
	matches := re.FindAllString(output, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("failed to parse %s from output: %q", valueType, output)
	}

	value, err := strconv.ParseFloat(matches[len(matches)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s from output: %q", valueType, output)
	}

	return value, nil
}

// validateAndClampCPUUsage ensures CPU usage is within valid range [0, 100]
func validateAndClampCPUUsage(cpuUsage float64) float64 {
	if cpuUsage < 0 {
		return 0
	}
	if cpuUsage > 100 {
		return 100
	}
	return cpuUsage
}

// Get real memory usage from node via SSH
func getNodeMemoryUsage(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'free' command with better parsing
	cmd := "free -b | grep Mem | awk '{printf \"%.2f\", ($3 / $2) * 100}'"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to execute memory command: %v", err)
	}

	// Parse memory usage from output
	memUsage, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory usage: %v", err)
	}

	// Ensure memory usage is within valid range
	if memUsage < 0 {
		memUsage = 0
	} else if memUsage > 100 {
		memUsage = 100
	}

	return memUsage, nil
}

// Get total memory from node via SSH
func getNodeTotalMemory(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'free' command to get total memory in GB
	cmd := "free -b | grep Mem | awk '{printf \"%.2f\", $2 / 1024 / 1024 / 1024}'"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to execute total memory command: %v", err)
	}

	// Parse total memory from output
	totalMemory, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse total memory: %v", err)
	}

	// Ensure we have a reasonable minimum
	if totalMemory < 1 {
		return 8.0, nil // fallback to 8GB if parsing fails
	}

	return totalMemory, nil
}

// pollNodeMetrics performs HTTP GET request to node's metrics endpoint
func pollNodeMetrics(nodeConfig node_control.NodeConfig) (*node_control.HTTPMetricsResponse, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Build metrics URL
	metricsURL := fmt.Sprintf("http://%s:%d/api/system/metrics", nodeConfig.Host, nodeConfig.MetricsPort)

	logger.LogWithNode(nodeConfig.Host, "HTTP", fmt.Sprintf("Making GET request to %s", metricsURL), "info")

	// Make HTTP request
	resp, err := client.Get(metricsURL)
	if err != nil {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("Request failed: %v", err))
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	logger.LogWithNode(nodeConfig.Host, "HTTP", fmt.Sprintf("Response status: %d", resp.StatusCode), "info")

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("Bad status: %d %s", resp.StatusCode, resp.Status))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON response
	var metrics node_control.HTTPMetricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		logger.LogError(nodeConfig.Host, "HTTP", fmt.Sprintf("JSON decode failed: %v", err))
		return nil, fmt.Errorf("JSON decode failed: %v", err)
	}

	logger.LogSuccess(nodeConfig.Host, "HTTP", "Metrics response parsed successfully")
	return &metrics, nil
}

// Get total CPU cores from node via SSH (legacy function - kept for compatibility)
func getNodeTotalCPU(nodeConfig node_control.NodeConfig) (float64, error) {
	// Use 'nproc' command to get CPU count
	cmd := "nproc"
	output, err := sshExec(nodeConfig, cmd)
	if err != nil {
		// Fallback to parsing /proc/cpuinfo
		cmd = "grep -c 'processor' /proc/cpuinfo"
		output, err = sshExec(nodeConfig, cmd)
		if err != nil {
			return 0, fmt.Errorf("failed to execute CPU command: %v", err)
		}
	}

	// Parse CPU cores from output
	cpuCores, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU cores: %v", err)
	}

	// Ensure we have a reasonable minimum
	if cpuCores < 1 {
		return 4.0, nil // fallback to 4 cores if parsing fails
	}

	return cpuCores, nil
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

// Execute SSH command and return output
func sshExec(nodeConfig node_control.NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR", // Reduce SSH warnings
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)

	// Get stdout and stderr separately
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipes: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start SSH command: %v", err)
	}

	// Read stdout
	stdoutBytes, _ := io.ReadAll(stdout)

	// Read stderr (to capture warnings)
	stderrBytes, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("SSH command failed: %v, stderr: %s", err, string(stderrBytes))
	}

	// Clean the output by removing SSH warnings and connection messages
	output := string(stdoutBytes)
	log.Printf("Raw stdout: %q", output) // Debug log
	output = cleanSSHOutput(output)
	log.Printf("Cleaned stdout: %q", output) // Debug log

	// If output is still empty or contains warnings, try stderr
	if strings.TrimSpace(output) == "" || strings.TrimSpace(output) == "0" {
		output = string(stderrBytes)
		log.Printf("Raw stderr: %q", output) // Debug log
		output = cleanSSHOutput(output)
		log.Printf("Cleaned stderr: %q", output) // Debug log
	}

	return output, nil
}

// Clean SSH output by extracting numeric values using regex
func cleanSSHOutput(output string) string {
	// Use regex to find all numeric values (including decimals) in the output
	re := regexp.MustCompile(`\d+\.?\d*`)
	matches := re.FindAllString(output, -1)

	// Return the last numeric match (should be the command output)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}

	// If no numeric value found, return default
	return "0"
}

// Check if a string is numeric
func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
