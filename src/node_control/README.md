# Node Control Package

The `node_control` package provides comprehensive node management functionality for the vuDataSim cluster system. It handles node configuration, deployment, monitoring, and SSH operations.

## Overview

This package manages the lifecycle of nodes in a distributed vuDataSim cluster, including:

- Node configuration management
- SSH-based file deployment
- Node enable/disable operations
- Metrics server verification
- Cluster settings management

## Files

### `node_manager.go`
Core node management functionality including:
- Node configuration loading/saving
- Node addition, removal, enable/disable operations
- File deployment via SSH
- Cluster settings management

### `ssh_operations.go`
SSH and SCP operations for remote node management:
- SSH command execution
- File and directory copying
- Remote directory creation
- SSH connection management

### `metrics.go`
Metrics server verification and monitoring:
- HTTP health checks for node metrics servers
- Metrics endpoint polling
- System resource monitoring

### `model.go`
Data structures and types:
- Node configuration structures
- Cluster settings
- API response models
- Metrics data structures

### `api_handlers.go`
HTTP API handlers for node management (currently commented out):
- REST API endpoints for node operations
- JSON response handling
- HTTP request processing

## Key Components

### NodeManager
The main struct that orchestrates all node operations:

```go
type NodeManager struct {
    nodesConfigPath string
    appConfigPath   string
    snapshotsDir    string
    backupsDir      string
    logsDir         string
    nodesConfig     NodesConfig
    appConfig       AppConfig
}
```

### Node Configuration
Nodes are configured via YAML with the following structure:

```yaml
cluster_settings:
  backup_retention_days: 30
  conflict_resolution: manual
  connection_timeout: 10
  max_retries: 3
  sync_timeout: 60

nodes:
  node1:
    host: 192.168.1.100
    user: admin
    key_path: /path/to/ssh/key
    conf_dir: /remote/conf/dir
    binary_dir: /remote/bin/dir
    metrics_port: 8086
    description: Production node
    enabled: true
```

## Usage

### Basic Operations

```go
// Create node manager
nm := NewNodeManager()

// Load configurations
err := nm.LoadNodesConfig()
if err != nil {
    log.Fatal(err)
}

// Add a new node
req := AddNodeRequest{
    Name:        "node1",
    Host:        "192.168.1.100",
    User:        "admin",
    KeyPath:     "/path/to/key",
    ConfDir:     "/remote/conf",
    BinaryDir:   "/remote/bin",
    Description: "Production server",
    Enabled:     true,
}

err = nm.AddNode(req)
if err != nil {
    log.Fatal(err)
}

// Enable a node
err = nm.EnableNode("node1")
if err != nil {
    log.Fatal(err)
}
```

### SSH Operations

The package provides secure SSH operations for remote management:

```go
// Execute command on remote node
output, err := nm.SSHExecWithOutput(nodeConfig, "ls -la")

// Copy files to remote node
err = nm.copyFilesToNode("node1", nodeConfig)
```

### Metrics Verification

Nodes include metrics server verification:

```go
// Verify metrics server is running
err = nm.verifyMetricsServer(nodeConfig)
if err != nil {
    log.Printf("Metrics server check failed: %v", err)
}
```

## Configuration Files

### Nodes Configuration (`src/configs/nodes.yaml`)
Contains cluster settings and individual node configurations.

### Application Configuration (`src/configs/config.yaml`)
Contains application-wide settings for backup, logging, network, etc.

## Deployment Process

When enabling a node, the system performs:

1. Configuration update
2. File deployment (binaries and configs)
3. Metrics server verification
4. SSH connectivity validation

## Error Handling

The package provides comprehensive error handling with specific error messages for:
- Node not found errors
- SSH connection failures
- File deployment issues
- Configuration validation errors

## Dependencies

- `gopkg.in/yaml.v3` for YAML configuration handling
- Standard Go libraries for HTTP, SSH, and file operations

## Security

- SSH key-based authentication
- Strict host key checking disabled for automation
- Connection timeouts for reliability
- Secure file permissions on configuration files