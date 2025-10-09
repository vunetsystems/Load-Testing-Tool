# Node Manager

The Node Manager is a comprehensive Go-based tool for managing remote nodes in a load testing environment. It provides both CLI and web-based interfaces for adding, removing, enabling, and disabling nodes, with automatic file deployment via SSH.

## Features

- **Add Nodes**: Automatically copies `finalvudatasim` binary and `conf.d` directory to remote nodes via SSH
- **Remove Nodes**: Removes node configuration and cleans up associated files
- **Enable/Disable Nodes**: Toggle node availability
- **List Nodes**: View all configured nodes or just enabled ones
- **Web Interface**: Integrated web UI for easy node management
- **REST API**: Full API for programmatic access
- **YAML Configuration**: Multiple configuration files for different settings
- **Cluster Settings**: Centralized cluster-wide configuration

## Prerequisites

- SSH access to remote nodes
- SSH private key for authentication
- Local `finalvudatasim` binary file
- Local `conf.d` directory with configuration files
- Go 1.21+ for building

## Installation

1. Navigate to the node_control directory:
   ```bash
   cd src/node_control
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the node manager:
   ```bash
   go build -o node_manager
   ```

## Usage

### CLI Mode

#### Add a Node

```bash
./node_manager add <name> <host> <user> <key_path> <conf_dir> <binary_dir> [description] [enabled]
```

**Parameters:**
- `name`: Node identifier/name
- `host`: SSH host/IP address
- `user`: SSH username
- `key_path`: Path to SSH private key file
- `conf_dir`: Remote path for conf.d directory
- `binary_dir`: Remote path for binary directory
- `description`: Optional description (can contain spaces)
- `enabled`: "true" or "false" (default: true)

**Example:**
```bash
./node_manager add prod-node-1 192.168.1.100 admin /home/admin/.ssh/id_rsa /opt/vu-data-sim/conf.d /opt/vu-data-sim/bin "Production Node 1" true
```

#### Remove a Node

```bash
./node_manager remove <name>
```

#### Enable/Disable a Node

```bash
./node_manager enable <name>
./node_manager disable <name>
```

#### List Nodes

```bash
./node_manager list           # List all nodes
./node_manager list-enabled   # List only enabled nodes
```

### Web Interface Mode

Start the web server with integrated frontend:

```bash
./node_manager web
```

The web interface will be available at `http://localhost:8501` (or configured port).

## API Endpoints

When running in web mode, the following API endpoints are available:

- `GET /api/nodes` - List all nodes
- `POST /api/nodes/{name}` - Create a new node
- `PUT /api/nodes/{name}` - Update node (enable/disable)
- `DELETE /api/nodes/{name}` - Remove a node
- `GET /api/cluster-settings` - Get cluster settings
- `PUT /api/cluster-settings` - Update cluster settings
- `GET /api/config` - Get application configuration
- `PUT /api/config` - Update application configuration

## Configuration Files

### nodes.yaml

Stores node configurations and cluster settings:

```yaml
cluster_settings:
  backup_retention_days: 30
  conflict_resolution: manual
  connection_timeout: 10
  max_retries: 3
  sync_timeout: 60
nodes:
  e2e-108-10:
    binary_dir: /home/vunet/vuDataSim/vuDataSim/bin
    conf_dir: /home/vunet/vuDataSim/vuDataSim/conf.d
    description: Primary vuDataSim node
    enabled: true
    host: 216.48.191.10
    key_path: ~/.ssh/id_rsa
    user: vunet
  e2e-83-184:
    binary_dir: /home/vunet/traefik-vudatasim/vudatasim/bin
    conf_dir: /home/vunet/traefik-vudatasim/vudatasim/conf.d
    description: ''
    enabled: true
    host: 164.52.214.184
    key_path: ~/.ssh/id_rsa
    user: vunet
```

### config.yaml

Application-wide configuration:

```yaml
backup:
  retention_days: 7
binaries:
  primary_binary: finalvudatasim
  supported_binaries:
  - vuDataSim
  - RakvuDataSim
  - finalvudatasim
  - gvudatsim
eps:
  default_unique_key: 1
  max_unique_key: 1000000000
logging:
  log_backup_count: 5
  log_file: node-manager.log
  log_max_size: 10485760
network:
  remote_host: 127.0.0.1
  remote_user: vunet
  streamlit_address: 0.0.0.0
  streamlit_port: 8501
paths:
  local_backups_dir: backups
  local_logs_dir: logs
  remote_binary_dir: /home/vunet/vuDataSim/vuDataSim/bin/
  remote_ssh_key: ~/.ssh/id_rsa
process:
  default_timeout: 300
  graceful_shutdown_timeout: 10
  remote_timeout: 300
```

## What Gets Copied

When adding a node, the following files are automatically copied:

1. **Binary**: `finalvudatasim` executable is copied to `<binary_dir>/finalvudatasim`
2. **Configuration Directory**: The entire `conf.d` directory is recursively copied to `<conf_dir>/`

## Security Considerations

- SSH keys should have appropriate permissions (600 for private keys)
- The tool uses `StrictHostKeyChecking=no` for automation (consider the security implications)
- Ensure remote directories exist or can be created by the SSH user
- Web interface should be protected in production environments

## Error Handling

The tool includes comprehensive error handling for:
- Missing local files (binary, conf.d directory)
- SSH connection failures
- Remote directory creation failures
- Configuration file read/write errors
- Node already exists/doesn't exist scenarios
- API validation errors

## Troubleshooting

1. **SSH Connection Issues**: Verify SSH key permissions and network connectivity
2. **File Copy Failures**: Ensure sufficient disk space on remote node and proper permissions
3. **Configuration Errors**: Check YAML syntax in configuration files
4. **Web Interface Issues**: Ensure no port conflicts and proper CORS configuration

## Directory Structure

```
src/node_control/
├── node_manager.go    # Main Go source file with web server
├── node_manager       # Compiled binary (CLI + Web UI)
├── nodes.yaml         # Node configurations and cluster settings
├── config.yaml        # Application configuration
├── static/           # Frontend files
│   ├── index.html    # Main dashboard
│   ├── script.js     # Frontend JavaScript
│   └── styles.css    # Styles (if any)
├── logs/             # Application logs (auto-created)
├── node_snapshots/   # Node snapshots (auto-created)
├── node_backups/     # Node backups (auto-created)
└── README.md         # This file
```

## Development

The application is built as a single binary that includes:
- Node management logic
- REST API server
- Static file serving for the web UI
- Configuration management

To modify the frontend, edit files in the `static/` directory. The web server will serve them automatically.