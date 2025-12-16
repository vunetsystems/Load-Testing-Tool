package node_control

import (
	"fmt"
	"os"
	"vuDataSim/src/logger"

	"gopkg.in/yaml.v3"
)

type AddNodeRequest struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	User        string `json:"user"`
	KeyPath     string `json:"key_path"`
	ConfDir     string `json:"conf_dir"`
	BinaryDir   string `json:"binary_dir"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

const (
	ErrNodeNotFound    = "node %s not found"
	ErrSaveConfig      = "failed to save config: %v"
	ErrSaveNodesConfig = "failed to save nodes config: %v"
)

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
	CurrentNodeIP    string `yaml:"current_node_ip"`
}

type PathsConfig struct {
	LocalBackupsDir string `yaml:"local_backups_dir"`
	LocalLogsDir    string `yaml:"local_logs_dir"`
	RemoteBinaryDir string `yaml:"remote_binary_dir"`
	RemoteConfDir   string `yaml:"remote_conf_dir"`
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
	sshSemaphore    chan struct{} // Semaphore to limit concurrent SSH operations
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
		sshSemaphore: make(chan struct{}, 3), // Limit to 3 concurrent SSH operations
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
func (nm *NodeManager) LoadAppConfig() (AppConfig, error) {
	if _, err := os.Stat(nm.appConfigPath); os.IsNotExist(err) {
		// Create default app config if file doesn't exist
		if err := nm.SaveAppConfig(); err != nil {
			return AppConfig{}, err
		}
	}

	data, err := os.ReadFile(nm.appConfigPath)
	if err != nil {
		return AppConfig{}, fmt.Errorf("failed to read app config file: %v", err)
	}

	err = yaml.Unmarshal(data, &nm.appConfig)
	if err != nil {
		return AppConfig{}, fmt.Errorf("failed to parse app config file: %v", err)
	}

	return nm.appConfig, nil
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
func (nm *NodeManager) AddNode(req AddNodeRequest) error {
	if _, exists := nm.nodesConfig.Nodes[req.Name]; exists {
		return fmt.Errorf("node %s already exists", req.Name)
	}

	nodeConfig := NodeConfig{
		Host:        req.Host,
		User:        req.User,
		KeyPath:     req.KeyPath,
		ConfDir:     req.ConfDir,
		BinaryDir:   req.BinaryDir,
		Description: req.Description,
		Enabled:     req.Enabled,
	}

	nm.nodesConfig.Nodes[req.Name] = nodeConfig

	// Save configuration first
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("failed to save nodes config: %v", err)
	}

	// Copy files to remote node
	err = nm.copyFilesToNode(req.Name, nodeConfig)
	if err != nil {
		// Rollback configuration on copy failure
		delete(nm.nodesConfig.Nodes, req.Name)
		nm.SaveNodesConfig()
		return fmt.Errorf("failed to copy files to node: %v", err)
	}

	logger.LogSuccess(req.Name, "node_control", "Node added successfully")
	return nil
}

// RemoveNode removes a node from configuration and cleans up files
func (nm *NodeManager) RemoveNode(name string) error {
	_, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf("ErrNodeNotFound")
	}

	// Remove from configuration
	delete(nm.nodesConfig.Nodes, name)
	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf("ErrSaveConfig")
	}

	// Clean up snapshots and backups
	err = nm.cleanupNodeFiles(name)
	if err != nil {
		logger.LogWarning(name, "node_control", fmt.Sprintf("Failed to cleanup files: %v", err))
	}

	logger.LogSuccess(name, "node_control", "Node removed successfully")
	return nil
}

// EnableNode enables a node
func (nm *NodeManager) EnableNode(name string) error {
	logger.Info().Str("node", name).Str("module", "node_control").Msg("Enable node process started")
	logger.Info().Str("node", name).Str("module", "node_control").Msg("Attempting to enable node")

	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		logger.Error().Str("node", name).Str("module", "node_control").Msg("Node not found in configuration")
		return fmt.Errorf("ErrNodeNotFound")
	}

	logger.Info().Str("node", name).Str("host", nodeConfig.Host).Bool("enabled", nodeConfig.Enabled).Int("metrics_port", nodeConfig.MetricsPort).Msg("Found node configuration")

	nodeConfig.Enabled = true
	nm.nodesConfig.Nodes[name] = nodeConfig

	logger.Info().Str("node", name).Msg("Saving node configuration")
	err := nm.SaveNodesConfig()
	if err != nil {
		logger.Error().Str("node", name).Err(err).Msg("Failed to save node configuration")
		return fmt.Errorf("ErrSaveConfig")
	}

	logger.LogSuccess(name, "node_control", "Node enabled successfully in configuration")

	// Trigger fresh deployment to ensure both binaries are present
	logger.Info().Str("node", name).Msg("Starting fresh deployment")
	logger.Info().Str("node", name).Msg("Triggering fresh deployment to ensure both binaries are present")
	logger.Info().Str("node", name).Str("host", nodeConfig.Host).Str("binary_dir", nodeConfig.BinaryDir).Str("conf_dir", nodeConfig.ConfDir).Msg("Node deployment configuration")

	err = nm.copyFilesToNode(name, nodeConfig)
	if err != nil {
		logger.Error().Str("node", name).Err(err).Msg("Failed to deploy files to node")
		logger.Warn().Str("node", name).Msg("Node enabled but files may not be up to date")
	} else {
		logger.LogSuccess(name, "node_control", "Fresh deployment completed successfully")
	}

	// Verify metrics server is running
	logger.Info().Str("node", name).Str("host", nodeConfig.Host).Int("port", nodeConfig.MetricsPort).Msg("Verifying metrics server")
	err = nm.verifyMetricsServer(nodeConfig)
	if err != nil {
		logger.Error().Str("node", name).Err(err).Msg("Metrics server verification failed")
		logger.Warn().Str("node", name).Msg("Node enabled but metrics server may not be running properly")
	} else {
		logger.LogSuccess(name, "node_control", "Metrics server verified successfully")
	}

	logger.Info().Str("node", name).Msg("Enable node process completed")
	return nil
}

// DisableNode disables a node
func (nm *NodeManager) DisableNode(name string) error {
	nodeConfig, exists := nm.nodesConfig.Nodes[name]
	if !exists {
		return fmt.Errorf(ErrNodeNotFound, name)
	}

	nodeConfig.Enabled = false
	nm.nodesConfig.Nodes[name] = nodeConfig

	err := nm.SaveNodesConfig()
	if err != nil {
		return fmt.Errorf(ErrSaveConfig, err)
	}

	logger.LogSuccess(name, "node_control", "Node disabled successfully")
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

func InitNodeData(nm *NodeManager, appState interface{}) {
	logger.Info().Str("module", "node_control").Msg("Initializing node data from configuration")

	// Load nodes from configuration
	err := nm.LoadNodesConfig()
	if err != nil {
		logger.Error().Err(err).Str("module", "node_control").Msg("Failed to load nodes config")
		logger.Warn().Str("module", "node_control").Msg("Using default node configuration")
		return
	}

	logger.Info().Str("module", "node_control").Msg("Loaded nodes from config")
}
