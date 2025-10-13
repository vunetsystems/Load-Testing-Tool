package node_control

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ClusterSettings represents cluster-wide configuration
type ClusterSettings struct {
	BackupRetentionDays int    `yaml:"backup_retention_days"`
	ConflictResolution  string `yaml:"conflict_resolution"`
	ConnectionTimeout   int    `yaml:"connection_timeout"`
	MaxRetries          int    `yaml:"max_retries"`
	SyncTimeout         int    `yaml:"sync_timeout"`
}

// NodeConfig represents a single node configuration
type NodeConfig struct {
	Host        string `yaml:"host"`
	User        string `yaml:"user"`
	KeyPath     string `yaml:"key_path"`
	ConfDir     string `yaml:"conf_dir"`
	BinaryDir   string `yaml:"binary_dir"`
	MetricsPort int    `yaml:"metrics_port"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
}

// NodesConfig represents the entire nodes configuration
type NodesConfig struct {
	ClusterSettings ClusterSettings       `yaml:"cluster_settings"`
	Nodes           map[string]NodeConfig `yaml:"nodes"`
}

// AppConfig represents application configuration
type AppConfig struct {
	Backup   BackupConfig   `yaml:"backup"`
	Binaries BinariesConfig `yaml:"binaries"`
	EPS      EPSConfig      `yaml:"eps"`
	Logging  LoggingConfig  `yaml:"logging"`
	Network  NetworkConfig  `yaml:"network"`
	Paths    PathsConfig    `yaml:"paths"`
	Process  ProcessConfig  `yaml:"process"`
}

type BackupConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

type BinariesConfig struct {
	PrimaryBinary     string   `yaml:"primary_binary"`
	SupportedBinaries []string `yaml:"supported_binaries"`
}

type EPSConfig struct {
	DefaultUniqueKey int `yaml:"default_unique_key"`
	MaxUniqueKey     int `yaml:"max_unique_key"`
}

type LoggingConfig struct {
	LogBackupCount int    `yaml:"log_backup_count"`
	LogFile        string `yaml:"log_file"`
	LogMaxSize     int    `yaml:"log_max_size"`
}

type NetworkConfig struct {
	RemoteHost       string `yaml:"remote_host"`
	RemoteUser       string `yaml:"remote_user"`
	StreamlitAddress string `yaml:"streamlit_address"`
	StreamlitPort    int    `yaml:"streamlit_port"`
}

type PathsConfig struct {
	LocalBackupsDir string `yaml:"local_backups_dir"`
	LocalLogsDir    string `yaml:"local_logs_dir"`
	RemoteBinaryDir string `yaml:"remote_binary_dir"`
	RemoteSSHKey    string `yaml:"remote_ssh_key"`
}

type ProcessConfig struct {
	DefaultTimeout          int `yaml:"default_timeout"`
	GracefulShutdownTimeout int `yaml:"graceful_shutdown_timeout"`
	RemoteTimeout           int `yaml:"remote_timeout"`
}

// NodeManager handles node operations
type NodeManager struct {
	nodesConfigPath string
	appConfigPath   string
	snapshotsDir    string
	backupsDir      string
	logsDir         string
	nodesConfig     NodesConfig
	appConfig       AppConfig
}

// NewNodeManager creates a new node manager instance
func NewNodeManager() *NodeManager {
	return &NodeManager{
		nodesConfigPath: "src/configs/nodes.yaml",
		appConfigPath:   "src/configs/config.yaml",
		snapshotsDir:    "src/node_control/node_snapshots",
		backupsDir:      "src/node_control/node_backups",
		logsDir:         "src/node_control/logs",
		nodesConfig: NodesConfig{
			ClusterSettings: ClusterSettings{
				BackupRetentionDays: 30,
				ConflictResolution:  "manual",
				ConnectionTimeout:   10,
				MaxRetries:          3,
				SyncTimeout:         60,
			},
			Nodes: make(map[string]NodeConfig),
		},
	}
}

// LoadNodesConfig loads the nodes configuration from YAML file
func (nm *NodeManager) LoadNodesConfig() error {
	if _, err := os.Stat(nm.nodesConfigPath); os.IsNotExist(err) {
		// Create default config if file doesn't exist
		return nm.SaveNodesConfig()
	}

	data, err := os.ReadFile(nm.nodesConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read nodes config file: %v", err)
	}

	err = yaml.Unmarshal(data, &nm.nodesConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nodes config file: %v", err)
	}

	return nil
}

// SaveNodesConfig saves the nodes configuration to YAML file
func (nm *NodeManager) SaveNodesConfig() error {
	data, err := yaml.Marshal(nm.nodesConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes config: %v", err)
	}

	err = os.WriteFile(nm.nodesConfigPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write nodes config file: %v", err)
	}

	return nil
}

// LoadAppConfig loads the application configuration from YAML file
func (nm *NodeManager) LoadAppConfig() error {
	if _, err := os.Stat(nm.appConfigPath); os.IsNotExist(err) {
		// Create default app config if file doesn't exist
		return nm.SaveAppConfig()
	}

	data, err := os.ReadFile(nm.appConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read app config file: %v", err)
	}

	err = yaml.Unmarshal(data, &nm.appConfig)
	if err != nil {
		return fmt.Errorf("failed to parse app config file: %v", err)
	}

	return nil
}

// SaveAppConfig saves the application configuration to YAML file
func (nm *NodeManager) SaveAppConfig() error {
	data, err := yaml.Marshal(nm.appConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal app config: %v", err)
	}

	err = os.WriteFile(nm.appConfigPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write app config file: %v", err)
	}

	return nil
}

// AddNode adds a new node to the configuration and copies files via SSH
func (nm *NodeManager) AddNode(name, host, user, keyPath, confDir, binaryDir, description string, enabled bool) error {
	if _, exists := nm.nodesConfig.Nodes[name]; exists {
		return fmt.Errorf("node %s already exists", name)
	}

	nodeConfig := NodeConfig{
		Host:        host,
		User:        user,
		KeyPath:     keyPath,
		ConfDir:     confDir,
		BinaryDir:   binaryDir,
		Description: description,
		Enabled:     enabled,
	}

	nm.nodesConfig.Nodes[name] = nodeConfig

	// Save configuration first
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save nodes config: %v", err)
	}

	// Copy files to remote node
	err = nm.copyFilesToNode(name, nodeConfig)
	if err != nil {
		// Rollback configuration on copy failure
		delete(nm.nodesConfig.Nodes, name)
		nm.SaveNodesConfig()
		return fmt.Errorf("failed to copy files to node: %v", err)
	}

	log.Printf("Successfully added node %s", name)
	return nil
}

// RemoveNode removes a node from configuration and cleans up files
func (nm *NodeManager) RemoveNode(name string) error {
	_, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	// Remove from configuration
	delete(nm.nodesConfig.Nodes, name)
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	// Clean up snapshots and backups
	err = nm.cleanupNodeFiles(name)
	if err != nil {
		log.Printf("Warning: failed to cleanup files for node %s: %v", name, err)
	}

	log.Printf("Successfully removed node %s", name)
	return nil
}

// EnableNode enables a node
func (nm *NodeManager) EnableNode(name string) error {
	log.Printf("=== ENABLE NODE PROCESS STARTED ===")
	log.Printf("Attempting to enable node: %s", name)

	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		log.Printf("ERROR: Node %s not found in configuration", name)
		return fmt.Errorf("node %s not found", name)
	}

	log.Printf("Found node %s with config: Host=%s, Enabled=%v, MetricsPort=%d",
		name, nodeConfig.Host, nodeConfig.Enabled, nodeConfig.MetricsPort)

	nodeConfig.Enabled = true
	nm.nodesConfig.Nodes[name] = nodeConfig

	log.Printf("Saving configuration for node %s...", name)
	err := nm.SaveNodesConfig()
	if err != nil {
		log.Printf("ERROR: Failed to save config for node %s: %v", name, err)
		return fmt.Errorf("failed to save config: %v", err)
	}

	log.Printf("✓ Successfully enabled node %s in configuration", name)

	// Trigger fresh deployment to ensure both binaries are present
	log.Printf("=== STARTING FRESH DEPLOYMENT ===")
	log.Printf("Triggering fresh deployment for node %s to ensure both binaries are present", name)
	log.Printf("Node config: Host=%s, BinaryDir=%s, ConfDir=%s",
		nodeConfig.Host, nodeConfig.BinaryDir, nodeConfig.ConfDir)

	err = nm.copyFilesToNode(name, nodeConfig)
	if err != nil {
		log.Printf("ERROR: Failed to deploy files to node %s: %v", name, err)
		log.Printf("Node enabled but files may not be up to date")
	} else {
		log.Printf("✓ Fresh deployment completed successfully for node %s", name)
	}

	// Verify metrics server is running
	log.Printf("=== VERIFYING METRICS SERVER ===")
	log.Printf("Verifying metrics server on %s:%d", nodeConfig.Host, nodeConfig.MetricsPort)
	err = nm.verifyMetricsServer(nodeConfig)
	if err != nil {
		log.Printf("ERROR: Metrics server verification failed for node %s: %v", name, err)
		log.Printf("Node enabled but metrics server may not be running properly")
	} else {
		log.Printf("✓ Metrics server verified successfully for node %s", name)
	}

	log.Printf("=== ENABLE NODE PROCESS COMPLETED ===")
	return nil
}

// DisableNode disables a node
func (nm *NodeManager) DisableNode(name string) error {
	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	nodeConfig.Enabled = false
	nm.nodesConfig.Nodes[name] = nodeConfig

	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	log.Printf("Successfully disabled node %s", name)
	return nil
}

// GetNodes returns all nodes
func (nm *NodeManager) GetNodes() map[string]NodeConfig {
	return nm.nodesConfig.Nodes
}

// GetEnabledNodes returns only enabled nodes
func (nm *NodeManager) GetEnabledNodes() map[string]NodeConfig {
	enabledNodes := make(map[string]NodeConfig)
	for name, config := range nm.nodesConfig.Nodes {
		if config.Enabled {
			enabledNodes[name] = config
		}
	}
	return enabledNodes
}

// GetClusterSettings returns the cluster settings
func (nm *NodeManager) GetClusterSettings() ClusterSettings {
	return nm.nodesConfig.ClusterSettings
}

// UpdateClusterSettings updates the cluster settings
func (nm *NodeManager) UpdateClusterSettings(settings ClusterSettings) error {
	nm.nodesConfig.ClusterSettings = settings
	return nm.SaveNodesConfig()
}

// SSHExecWithOutput executes a command on the remote node via SSH and returns the output
func (nm *NodeManager) SSHExecWithOutput(nodeConfig NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// copyFilesToNode copies the binaries and conf.d directory to the remote node
func (nm *NodeManager) copyFilesToNode(nodeName string, nodeConfig NodeConfig) error {
	localMainBinary := "src/finalvudatasim"
	localMetricsBinary := "src/node_metrics_api/build/node_metrics_api"
	localConfDir := "src/conf.d"

	log.Printf("DEBUG: Deployment paths for node %s:", nodeName)
	log.Printf("  Main binary path: %s", localMainBinary)
	log.Printf("  Metrics binary path: %s", localMetricsBinary)
	log.Printf("  Conf dir path: %s", localConfDir)

	// Check if local files exist
	if _, err := os.Stat(localMainBinary); os.IsNotExist(err) {
		return fmt.Errorf("local main binary file %s not found", localMainBinary)
	}

	if _, err := os.Stat(localMetricsBinary); os.IsNotExist(err) {
		return fmt.Errorf("local metrics binary file %s not found", localMetricsBinary)
	}

	if _, err := os.Stat(localConfDir); os.IsNotExist(err) {
		return fmt.Errorf("local conf.d directory %s not found", localConfDir)
	}

	// Create remote directories
	err := nm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s %s", nodeConfig.BinaryDir, nodeConfig.ConfDir))
	if err != nil {
		return fmt.Errorf("failed to create remote directories: %v", err)
	}

	// Copy main binary file
	log.Printf("Copying main binary from %s to %s", localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	err = nm.scpCopy(nodeConfig, localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	if err != nil {
		log.Printf("ERROR: Failed to copy main binary: %v", err)
		return fmt.Errorf("failed to copy main binary: %v", err)
	}
	log.Printf("✓ Main binary copied successfully")

	// Copy metrics API binary
	log.Printf("Copying metrics binary from %s to %s", localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api"))
	err = nm.scpCopy(nodeConfig, localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api"))
	if err != nil {
		log.Printf("ERROR: Failed to copy metrics binary: %v", err)
		return fmt.Errorf("failed to copy metrics binary: %v", err)
	}
	log.Printf("✓ Metrics binary copied successfully")

	// Copy conf.d directory recursively
	log.Printf("Copying conf.d directory from %s to %s", localConfDir, nodeConfig.ConfDir)
	err = nm.scpCopyDir(nodeConfig, localConfDir, nodeConfig.ConfDir)
	if err != nil {
		log.Printf("ERROR: Failed to copy conf.d directory: %v", err)
		return fmt.Errorf("failed to copy conf.d directory: %v", err)
	}
	log.Printf("✓ Conf.d directory copied successfully")

	log.Printf("Successfully copied files to node %s", nodeName)
	return nil
}

// cleanupNodeFiles removes snapshots and backups for a node
func (nm *NodeManager) cleanupNodeFiles(nodeName string) error {
	nodeSnapshotDir := filepath.Join(nm.snapshotsDir, nodeName)
	nodeBackupDir := filepath.Join(nm.backupsDir, nodeName)

	if _, err := os.Stat(nodeSnapshotDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeSnapshotDir)
		if err != nil {
			return fmt.Errorf("failed to remove snapshot directory: %v", err)
		}
	}

	if _, err := os.Stat(nodeBackupDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeBackupDir)
		if err != nil {
			return fmt.Errorf("failed to remove backup directory: %v", err)
		}
	}

	return nil
}

// verifyMetricsServer checks if the metrics server is running on the node
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

	log.Printf("Metrics server health check successful for node %s", nodeConfig.Host)
	return nil
}

// sshExec executes a command on the remote node via SSH
func (nm *NodeManager) sshExec(nodeConfig NodeConfig, command string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH command failed: %v", err)
	}

	return nil
}

// scpCopy copies a single file to the remote node
/*
func (nm *NodeManager) scpCopy(nodeConfig NodeConfig, localPath, remotePath string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-r", // recursive for directories
		localPath,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP copy failed: %v", err)
	}

	return nil
}*/

func (nm *NodeManager) scpCopy(nodeConfig NodeConfig, localPath, remotePath string) error {
	log.Printf("DEBUG: SCP copying %s to %s@%s:%s", localPath, nodeConfig.User, nodeConfig.Host, remotePath)

	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
	}

	// Add -r only if localPath is a directory
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local path %s: %v", localPath, err)
	}
	if info.IsDir() {
		args = append(args, "-r")
		log.Printf("DEBUG: Copying directory with -r flag")
	}

	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath))

	log.Printf("DEBUG: Executing SCP command: scp %v", args)

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("ERROR: SCP command failed for %s: %v", localPath, err)
		return fmt.Errorf("SCP copy failed: %v", err)
	}

	log.Printf("DEBUG: SCP copy successful for %s", localPath)
	return nil
}

// scpCopyDir copies a directory recursively to the remote node

func (nm *NodeManager) scpCopyDir(nodeConfig NodeConfig, localDir, remoteDir string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-r",
		localDir,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remoteDir),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP directory copy failed: %v", err)
	}

	return nil
}

// ListNodes prints all nodes with their status
func (nm *NodeManager) ListNodes() {
	fmt.Println("Configured Nodes:")
	fmt.Println("================")

	if len(nm.nodesConfig.Nodes) == 0 {
		fmt.Println("No nodes configured")
		return
	}

	for name, config := range nm.nodesConfig.Nodes {
		status := "Disabled"
		if config.Enabled {
			status = "Enabled"
		}

		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
	}
}

func main() {
	nm := NewNodeManager()

	// Load configurations
	err := nm.LoadNodesConfig()
	if err != nil {
		log.Fatalf("Failed to load nodes config: %v", err)
	}

	err = nm.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load app config: %v", err)
	}

	// Setup logging
	if err := os.MkdirAll(nm.logsDir, 0755); err != nil {
		log.Printf("Warning: failed to create logs directory: %v", err)
	}

	// Check if web UI mode is requested
	if len(os.Args) > 1 && os.Args[1] == "web" {
		startWebServer(nm)
		return
	}

	// CLI mode
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "add":
		err = handleAddNode(nm, args)
	case "remove":
		err = handleRemoveNode(nm, args)
	case "enable":
		err = handleEnableNode(nm, args)
	case "disable":
		err = handleDisableNode(nm, args)
	case "list":
		nm.ListNodes()
		return
	case "list-enabled":
		handleListEnabledNodes(nm)
		return
	default:
		printUsage()
		return
	}

	if err != nil {
		log.Fatal(err)
	}
}

// startWebServer starts the web server with the integrated frontend
func startWebServer(nm *NodeManager) {
	// Setup API routes
	http.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
		handleAPINodes(w, r, nm)
	})

	http.HandleFunc("/api/nodes/", func(w http.ResponseWriter, r *http.Request) {
		handleAPINodeActions(w, r, nm)
	})

	http.HandleFunc("/api/cluster-settings", func(w http.ResponseWriter, r *http.Request) {
		handleAPIClusterSettings(w, r, nm)
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		handleAPIAppConfig(w, r, nm)
	})

	// Serve static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Route to appropriate handler
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// API routes are handled above
			http.NotFound(w, r)
			return
		}

		// Serve static files from the static directory relative to current working directory
		staticPath := filepath.Join("static", r.URL.Path)

		// If path is just "/", serve index.html
		if r.URL.Path == "/" {
			staticPath = filepath.Join("static", "index.html")
		} else if !strings.Contains(r.URL.Path, ".") {
			// If no extension, assume it's a directory and serve index.html
			staticPath = filepath.Join("static", r.URL.Path, "index.html")
		}

		http.ServeFile(w, r, staticPath)
	})

	port := nm.appConfig.Network.StreamlitPort
	address := fmt.Sprintf("%s:%d", nm.appConfig.Network.StreamlitAddress, port)

	log.Printf("Starting web server on %s", address)
	log.Printf("Web UI available at http://%s", address)
	log.Printf("API endpoints available at http://%s/api/", address)

	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}

func handleAddNode(nm *NodeManager, args []string) error {
	if len(args) < 6 {
		return fmt.Errorf("usage: node_manager add <name> <host> <user> <key_path> <conf_dir> <binary_dir> [description] [enabled]")
	}

	name := args[0]
	host := args[1]
	user := args[2]
	keyPath := args[3]
	confDir := args[4]
	binaryDir := args[5]

	description := ""
	if len(args) > 6 {
		description = strings.Join(args[6:len(args)-1], " ")
	}

	enabled := true
	if len(args) > 7 {
		enabled = args[len(args)-1] == "true"
	}

	return nm.AddNode(name, host, user, keyPath, confDir, binaryDir, description, enabled)
}

func handleRemoveNode(nm *NodeManager, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: node_manager remove <name>")
	}

	name := args[0]
	return nm.RemoveNode(name)
}

func handleEnableNode(nm *NodeManager, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: node_manager enable <name>")
	}

	name := args[0]
	return nm.EnableNode(name)
}

func handleDisableNode(nm *NodeManager, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: node_manager disable <name>")
	}

	name := args[0]
	return nm.DisableNode(name)
}

func handleListEnabledNodes(nm *NodeManager) {
	enabledNodes := nm.GetEnabledNodes()
	fmt.Println("Enabled Nodes:")
	fmt.Println("==============")

	if len(enabledNodes) == 0 {
		fmt.Println("No enabled nodes")
		return
	}

	for name, config := range enabledNodes {
		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
	}
}

// API Handlers

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// handleAPINodes handles GET /api/nodes (list all nodes)
func handleAPINodes(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	nodes := nm.GetNodes()
	nodeList := make([]map[string]interface{}, 0)

	for name, config := range nodes {
		status := "Disabled"
		if config.Enabled {
			status = "Enabled"
		}

		nodeList = append(nodeList, map[string]interface{}{
			"name":        name,
			"host":        config.Host,
			"user":        config.User,
			"status":      status,
			"description": config.Description,
			"binary_dir":  config.BinaryDir,
			"conf_dir":    config.ConfDir,
			"enabled":     config.Enabled,
		})
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    nodeList,
	})
}

// handleAPINodeActions handles POST/PUT/DELETE /api/nodes/{name}
func handleAPINodeActions(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	// Extract node name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Node name is required",
		})
		return
	}

	nodeName := parts[0]

	switch r.Method {
	case http.MethodPost:
		handleCreateNode(w, r, nm, nodeName)
	case http.MethodPut:
		handleUpdateNode(w, r, nm, nodeName)
	case http.MethodDelete:
		handleDeleteNode(w, r, nm, nodeName)
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// handleCreateNode handles POST /api/nodes/{name}
func handleCreateNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
	var nodeData struct {
		Host        string `json:"host"`
		User        string `json:"user"`
		KeyPath     string `json:"key_path"`
		ConfDir     string `json:"conf_dir"`
		BinaryDir   string `json:"binary_dir"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	err := nm.AddNode(nodeName, nodeData.Host, nodeData.User, nodeData.KeyPath,
		nodeData.ConfDir, nodeData.BinaryDir, nodeData.Description, nodeData.Enabled)

	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s created successfully", nodeName),
	})
}

// handleUpdateNode handles PUT /api/nodes/{name}
func handleUpdateNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
	var nodeData struct {
		Enabled *bool `json:"enabled,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&nodeData); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON data",
		})
		return
	}

	if nodeData.Enabled != nil {
		if *nodeData.Enabled {
			err := nm.EnableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		} else {
			err := nm.DisableNode(nodeName)
			if err != nil {
				sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
					Success: false,
					Message: err.Error(),
				})
				return
			}
		}
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s updated successfully", nodeName),
	})
}

// handleDeleteNode handles DELETE /api/nodes/{name}
func handleDeleteNode(w http.ResponseWriter, r *http.Request, nm *NodeManager, nodeName string) {
	err := nm.RemoveNode(nodeName)
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node %s deleted successfully", nodeName),
	})
}

// handleAPIClusterSettings handles GET/PUT /api/cluster-settings
func handleAPIClusterSettings(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nm.nodesConfig.ClusterSettings,
		})
	case http.MethodPut:
		var settings ClusterSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		nm.nodesConfig.ClusterSettings = settings
		err := nm.SaveNodesConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Cluster settings updated successfully",
		})
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

// handleAPIAppConfig handles GET/PUT /api/config
func handleAPIAppConfig(w http.ResponseWriter, r *http.Request, nm *NodeManager) {
	switch r.Method {
	case http.MethodGet:
		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    nm.appConfig,
		})
	case http.MethodPut:
		var config AppConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			sendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "Invalid JSON data",
			})
			return
		}

		nm.appConfig = config
		err := nm.SaveAppConfig()
		if err != nil {
			sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		sendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Application configuration updated successfully",
		})
	default:
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  node_manager add <name> <host> <user> <key_path> <conf_dir> <binary_dir> [description] [enabled]")
	fmt.Println("  node_manager remove <name>")
	fmt.Println("  node_manager enable <name>")
	fmt.Println("  node_manager disable <name>")
	fmt.Println("  node_manager list")
	fmt.Println("  node_manager list-enabled")
	fmt.Println("  node_manager web")
	fmt.Println()
	fmt.Println("Web UI:")
	fmt.Println("  node_manager web    # Start web interface")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  node_manager add node1 192.168.1.100 admin /path/to/key /remote/conf /remote/bin \"Production server\" true")
	fmt.Println("  node_manager remove node1")
	fmt.Println("  node_manager enable node1")
	fmt.Println("  node_manager disable node1")
	fmt.Println("  node_manager list")
	fmt.Println("  node_manager list-enabled")
	fmt.Println("  node_manager web")
}
