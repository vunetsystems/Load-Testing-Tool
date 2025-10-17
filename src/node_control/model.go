package node_control

import "time"

type ClusterSettings struct {
	BackupRetentionDays int    `yaml:"backup_retention_days"`
	ConflictResolution  string `yaml:"conflict_resolution"`
	ConnectionTimeout   int    `yaml:"connection_timeout"`
	MaxRetries          int    `yaml:"max_retries"`
	SyncTimeout         int    `yaml:"sync_timeout"`
}

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

// HTTPMetricsResponse represents the response from node metrics API
type HTTPMetricsResponse struct {
	NodeID    string    `json:"nodeId"`
	Timestamp time.Time `json:"timestamp"`
	System    struct {
		CPU    HTTPNodeCPUInfo    `json:"cpu"`
		Memory HTTPNodeMemoryInfo `json:"memory"`
		Uptime int64              `json:"uptime_seconds"`
	} `json:"system"`
}

// HTTPNodeCPUInfo represents CPU metrics from HTTP API
type HTTPNodeCPUInfo struct {
	UsedPercent float64 `json:"used_percent"`
	Cores       int     `json:"cores"`
	Load1M      float64 `json:"load_1m"`
}

// HTTPNodeMemoryInfo represents memory metrics from HTTP API
type HTTPNodeMemoryInfo struct {
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	TotalGB     float64 `json:"total_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// NodeMetrics represents metrics for a single node
type NodeMetrics struct {
	NodeID      string    `json:"nodeId"`
	Status      string    `json:"status"`
	EPS         int       `json:"eps"`
	KafkaLoad   int       `json:"kafkaLoad"`
	CHLoad      int       `json:"chLoad"`
	CPU         float64   `json:"cpu"`         // CPU usage percentage (0-100)
	Memory      float64   `json:"memory"`      // Memory usage percentage (0-100)
	TotalCPU    float64   `json:"totalCpu"`    // Total CPU cores available
	TotalMemory float64   `json:"totalMemory"` // Total memory in GB available
	LastUpdate  time.Time `json:"lastUpdate"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
