# vuDataSim Cluster Manager

A comprehensive load testing and simulation management system with real-time dashboard, SSH-based node management, and distributed metrics collection.

## 🎯 Project Overview

The **vuDataSim Cluster Manager** is a sophisticated load testing and performance monitoring platform designed to manage distributed load testing nodes across multiple servers. It provides real-time visibility into cluster performance, automated metrics collection via SSH, and a modern web-based management interface.

### Core Mission
- **Distributed Load Testing**: Coordinate load testing across multiple nodes/servers
- **Real-time Monitoring**: Live metrics collection and visualization
- **SSH-based Management**: Automated deployment and configuration of load testing binaries
- **Performance Analytics**: Track CPU, memory, EPS, Kafka, and ClickHouse metrics
- **Cluster Orchestration**: Centralized control over distributed testing infrastructure

## 🚀 Key Features

### Real-time Dashboard & Monitoring
- **Live Metrics Collection**: CPU, memory, and system resource monitoring via SSH
- **Interactive Charts**: SVG-based animated charts with real-time updates
- **Node Status Indicators**: Visual status for all configured nodes with live connectivity
- **Performance Tracking**: EPS (Events Per Second), Kafka load, ClickHouse load monitoring
- **Responsive UI**: Modern dark/light theme with Material Design principles

### SSH-based Node Management
- **Automated Deployment**: Copy binaries and configurations to remote nodes via SCP
- **Remote Execution**: Execute commands and collect metrics via SSH
- **Node Lifecycle Management**: Add, remove, enable, disable nodes dynamically
- **Configuration Sync**: Synchronize configuration files across cluster nodes
- **Connection Management**: SSH key-based authentication with connection pooling concepts

### Load Testing Capabilities
- **Profile Management**: Low/Medium/High/Custom load testing profiles
- **Simulation Control**: Start/stop load testing simulations with real-time parameter adjustment
- **Distributed Testing**: Coordinate load generation across multiple nodes
- **Performance Validation**: Monitor system impact during load testing

### Real-time Communication
- **WebSocket Integration**: Bidirectional real-time updates
- **Live Log Streaming**: Filtered log viewing with real-time updates
- **Event-driven Architecture**: Real-time event propagation across components

## 🏗️ Architecture & Design

### System Architecture
```
┌─────────────────────────────────────────────────────────────────┐
│                    vuDataSim Cluster Manager                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Frontend   │  │   Backend   │  │   Metrics   │              │
│  │  (React.js) │◄►│   (Go)      │◄►│ Collection  │              │
│  │             │  │             │  │   (SSH)     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Node 1    │  │   Node 2    │  │   Node N    │              │
│  │ (SSH Agent) │  │ (SSH Agent) │  │ (SSH Agent) │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### Backend (Go)
- **Web Server**: Gorilla Mux-based HTTP server with WebSocket support
- **SSH Client**: Integrated SSH/SCP client for remote node management
- **Metrics Collector**: Real-time system resource monitoring via SSH
- **Node Manager**: CRUD operations for cluster nodes
- **WebSocket Hub**: Real-time communication with frontend clients

#### Frontend (Vanilla JavaScript)
- **Dashboard Interface**: Real-time monitoring dashboard
- **Node Management UI**: Forms and tables for node administration
- **Chart Visualizations**: SVG-based animated performance charts
- **Log Viewer**: Filtered real-time log streaming interface

#### Remote Agents
- **SSH Connectivity**: Each node accessible via SSH with key authentication
- **Metrics Providers**: System resource monitoring (CPU, memory, processes)
- **Binary Deployment**: Automated deployment of vuDataSim binaries and configs

## 📁 Project Structure

```
Load-Testing-Tool/
├── src/
│   ├── main.go                    # Main application server (1905 lines)
│   ├── finalvudatasim             # Load testing binary
│   ├── conf.d/                    # Configuration templates (50+ files)
│   │   ├── Apache/                # Apache monitoring configs
│   │   ├── AWS_ALB/               # AWS Load Balancer configs
│   │   ├── Azure_*/               # Azure service monitoring
│   │   ├── CiscoIoSSwitch/         # Network device configs
│   │   ├── DNS_Monitoring/         # DNS service configs
│   │   ├── IBMMQ/                 # IBM MQ monitoring
│   │   ├── K8s/                   # Kubernetes monitoring
│   │   ├── LinuxMonitor/          # Linux system monitoring
│   │   ├── MongoDB/               # MongoDB monitoring
│   │   ├── Mssql/                 # SQL Server monitoring
│   │   ├── Nginx/                 # Nginx monitoring
│   │   ├── Tomcat/                # Tomcat monitoring
│   │   └── WebLogic/              # WebLogic monitoring
│   ├── configs/
│   │   ├── nodes.yaml             # Node configurations
│   │   └── config.yaml            # Application settings
│   └── node_control/
│       ├── node_manager.go        # Node management logic
│       └── README.md              # Node control documentation
├── static/
│   ├── index.html                 # Main dashboard interface
│   ├── script.js                  # Frontend JavaScript (1000+ lines)
│   └── styles.css                 # Custom styling and animations
├── go.mod                         # Go dependencies
├── go.sum                         # Dependency checksums
└── README.md                      # This documentation
```

## 🛠️ Technologies & Dependencies

### Core Technologies
- **Backend**: Go 1.21 with concurrent processing
- **Frontend**: HTML5, CSS3, Vanilla JavaScript
- **Styling**: Tailwind CSS (CDN) with custom animations
- **Icons**: Google Material Symbols
- **Real-time**: WebSocket for bidirectional communication

### Go Dependencies
```go
github.com/gorilla/mux v1.8.0      # HTTP router and URL matcher
github.com/gorilla/websocket v1.5.0 # WebSocket implementation
github.com/rs/cors v1.10.1          # CORS middleware
gopkg.in/yaml.v3 v3.0.1            # YAML configuration parsing
```

### External Tools
- **SSH/SCP**: OpenSSH client for remote node management
- **System Monitoring**: Standard Linux tools (`free`, `vmstat`, `nproc`, `top`)

## 🚀 Quick Start Guide

### Prerequisites
- **Go 1.21+**: Modern Go runtime environment
- **SSH Access**: Key-based SSH access to target nodes
- **Web Browser**: Modern browser with WebSocket support
- **Network Access**: Nodes must be reachable via SSH (port 22)

### Installation & Setup

1. **Navigate to project directory:**
   ```bash
   cd Load-Testing-Tool
   ```

2. **Install Go dependencies:**
   ```bash
   go mod tidy
   ```

3. **Configure nodes (optional):**
   ```bash
   # Edit src/configs/nodes.yaml to add your nodes
   # Or use the web interface to add nodes dynamically
   ```

4. **Start the server:**
   ```bash
   go run src/main.go
   ```

5. **Access the dashboard:**
   Navigate to `http://localhost:3000`

### Node Configuration Example
```yaml
nodes:
  testing-node-1:
    host: 192.168.1.100
    user: vunet
    key_path: ~/.ssh/id_rsa
    conf_dir: /home/vunet/NEWTESTING
    binary_dir: /home/vunet/NEWTESTING
    description: Primary testing node
    enabled: true
```

## 📖 Detailed Usage Guide

### Dashboard Interface

#### Sidebar Controls (Left Panel)
- **Real-time Info Panel**: Shows current metrics collection status
- **SSH Connection Status**: Live connectivity indicators
- **Refresh Controls**: Manual data refresh and sync operations
- **System Status**: Overall cluster health indicators

#### Main Dashboard (Right Panel)
- **Node Status Bar**: Live connectivity status for all nodes
- **Cluster Overview Table**: CPU/Memory usage across all nodes
- **Live Charts**: Real-time system resource visualization
- **Node Management Modal**: Complete node administration interface
- **Log Viewer**: Filtered real-time log streaming

### Node Management Operations

#### Adding Nodes
1. **Via Web Interface**: Use the "Node Management" button
2. **Via CLI**: Use command-line node management commands
3. **Via Configuration**: Edit `nodes.yaml` directly

#### Node Operations
- **Add Node**: Deploy binaries and configurations via SSH/SCP
- **Remove Node**: Clean removal with file cleanup
- **Enable/Disable**: Activate/deactivate node monitoring
- **Update Configuration**: Modify node settings dynamically

### Metrics Collection Process

#### Current Implementation (SSH-based)
1. **Connection Establishment**: SSH connection to each node every 3 seconds
2. **CPU Metrics**: Execute `vmstat` or `top` commands remotely
3. **Memory Metrics**: Execute `free` command to get memory usage
4. **System Info**: Collect total CPU cores and memory capacity
5. **Data Processing**: Parse and clean SSH output for dashboard display

#### Collection Frequency
- **Metrics Interval**: Every 3 seconds (configurable)
- **SSH Connections**: 4 connections per cycle (2 nodes × 2 metrics each)
- **Update Rate**: Real-time WebSocket updates to dashboard

## 🔧 Configuration & Settings

### Application Configuration
```yaml
# src/configs/config.yaml
server:
  port: ":3000"
  static_dir: "./static"
  log_level: "info"

monitoring:
  interval_seconds: 3
  enable_real_metrics: true
  ssh_timeout: 10

websocket:
  enable_compression: true
  read_buffer_size: 1024
  write_buffer_size: 1024
```

### Node Configuration
```yaml
# src/configs/nodes.yaml
cluster_settings:
  backup_retention_days: 30
  conflict_resolution: manual
  connection_timeout: 10
  max_retries: 3
  sync_timeout: 60

nodes:
  node_name:
    host: "remote-server-ip"
    user: "ssh-username"
    key_path: "~/.ssh/id_rsa"
    conf_dir: "/remote/config/path"
    binary_dir: "/remote/binary/path"
    description: "Node description"
    enabled: true
```

## 🔌 API Reference

### Core Endpoints

#### Simulation Control
- `POST /api/simulation/start` - Start load testing simulation
- `POST /api/simulation/stop` - Stop current simulation
- `POST /api/config/sync` - Sync configuration settings

#### Data & Monitoring
- `GET /api/dashboard` - Get current dashboard data
- `GET /api/logs` - Get filtered log entries with pagination
- `GET /api/health` - Health check with uptime information

#### Node Management
- `GET /api/nodes` - List all configured nodes
- `POST /api/nodes/{name}` - Create new node
- `PUT /api/nodes/{name}` - Update node configuration
- `DELETE /api/nodes/{name}` - Remove node

#### Real-time Communication
- `WebSocket /ws` - Real-time bidirectional updates
- `PUT /api/nodes/{nodeId}/metrics` - Update node metrics

### CLI Node Management

#### Available Commands
```bash
# Node lifecycle management
go run src/main.go add <name> <host> <user> <key> <conf_dir> <bin_dir> [desc] [enabled]
go run src/main.go remove <name>
go run src/main.go enable <name>
go run src/main.go disable <name>

# Node inspection
go run src/main.go list              # List all nodes
go run src/main.go list-enabled      # List only enabled nodes

# Server mode
go run src/main.go web               # Start web server
```

## 🚨 Current Issues & Limitations

### SSH Metrics Collection Problems

#### Issue 1: Inefficient Connection Management
- **Problem**: Creates 4 SSH connections every 3 seconds (2 nodes × 2 metrics)
- **Impact**: High connection overhead, potential SSH rate limiting
- **Current Pattern**: 80+ SSH connections per minute for 2 nodes

#### Issue 2: Timing Inconsistencies
- **Problem**: SSH connection establishment adds 500ms-2s latency
- **Impact**: Inconsistent 3-second collection intervals become 5-7 seconds
- **Evidence**: Logs show 7-second gaps between collection cycles

#### Issue 3: No Connection Reuse
- **Problem**: Each metric collection creates fresh SSH connections
- **Impact**: Maximum inefficiency, no connection pooling benefits

### Recommended Improvements

#### Short-term Fixes
1. **Increase Collection Interval**: Change from 3s to 10-15s for better stability
2. **Batch SSH Commands**: Collect multiple metrics in single SSH session
3. **Connection Reuse**: Keep SSH connections open between collection cycles

#### Long-term Solutions
1. **HTTP-based Metrics**: Replace SSH with Prometheus node exporters
2. **SNMP Monitoring**: Use SNMP for standardized metrics collection
3. **Push-based Architecture**: Have nodes push metrics to central collector

## 🎯 Project Goals & Roadmap

### Current Goals (v1.0.0)
- [x] Basic SSH-based metrics collection
- [x] Real-time dashboard with WebSocket updates
- [x] Node management via web interface
- [x] Load testing simulation controls
- [ ] Optimize SSH metrics collection efficiency
- [ ] Add persistent SSH connection management

### Future Enhancements (v1.1+)
- [ ] Replace SSH with HTTP-based metrics collection
- [ ] Add Prometheus/Grafana integration
- [ ] Implement distributed load testing coordination
- [ ] Add alerting and notification system
- [ ] Support for containerized deployment
- [ ] Advanced analytics and reporting
- [ ] REST API rate limiting and authentication

### Technical Debt
- [ ] Refactor SSH metrics collection for efficiency
- [ ] Add comprehensive error handling and retry logic
- [ ] Implement connection pooling for SSH sessions
- [ ] Add metrics caching to reduce collection frequency
- [ ] Improve logging and debugging capabilities

## 🚦 Development Workflow

### Code Organization
- **main.go**: Central application logic with HTTP server, WebSocket hub, and metrics collection
- **Node Management**: Dedicated node manager with SSH/SCP functionality
- **Metrics Collection**: Separate functions for CPU, memory, and system metrics
- **WebSocket Handling**: Real-time communication with frontend clients

### Development Practices
- **Concurrent Processing**: Goroutines for metrics collection and WebSocket handling
- **Error Handling**: Comprehensive error handling with fallback mechanisms
- **Configuration Management**: YAML-based configuration with validation
- **Logging**: Structured logging with timestamps and context

## 🔒 Security Considerations

### Current Security Model
- **SSH Authentication**: Key-based SSH authentication for node access
- **CORS Configuration**: Development CORS policy (allows all origins)
- **Input Validation**: Basic validation for user inputs
- **No Authentication**: Currently no user authentication system

### Security Improvements Needed
- [ ] Implement user authentication and authorization
- [ ] Add API rate limiting and request validation
- [ ] Secure CORS configuration for production
- [ ] SSH key management and rotation
- [ ] Input sanitization and validation
- [ ] HTTPS/SSL configuration for production

## 🐛 Troubleshooting Guide

### Common Issues

#### SSH Connection Problems
```bash
# Test SSH connectivity manually
ssh -i ~/.ssh/id_rsa vunet@remote-host "free -b"

# Check SSH agent status
ssh-add -l

# Verify key permissions
chmod 600 ~/.ssh/id_rsa
```

#### Metrics Collection Issues
- **Empty Metrics**: Check SSH command outputs in server logs
- **Connection Timeouts**: Increase timeout values in configuration
- **Permission Denied**: Verify SSH key and user permissions on remote nodes

#### WebSocket Connection Issues
- **Connection Failed**: Check server logs for WebSocket errors
- **Real-time Updates Stop**: Verify network connectivity and server status

### Debug Mode
Enable verbose logging by checking the server console output. The application logs detailed information about:
- SSH command execution and output
- WebSocket connection events
- Metrics collection timing and results
- Node management operations

## 📈 Performance Characteristics

### Current Performance Profile
- **Concurrent Connections**: Supports multiple WebSocket clients
- **Collection Frequency**: Every 3 seconds (with timing variations)
- **SSH Overhead**: ~500ms-2s per connection establishment
- **Memory Usage**: Efficient state management with mutex locking
- **UI Responsiveness**: Smooth animations with CSS transitions

### Performance Optimizations
- **WebSocket Broadcasting**: Efficient client update mechanism
- **Metrics Caching**: Basic caching of static system information
- **Goroutine Management**: Proper concurrent processing patterns

## 🤝 Contributing & Development

### Development Setup
1. **Clone and Setup**: Standard Go project setup
2. **Edit Configurations**: Modify `nodes.yaml` for your environment
3. **Test SSH Connectivity**: Verify access to target nodes
4. **Run Development Server**: Use `go run src/main.go`

### Code Style Guidelines
- **Go Conventions**: Follow standard Go formatting and naming
- **Error Handling**: Comprehensive error handling with context
- **Concurrency**: Proper use of goroutines and channels
- **Documentation**: Comment complex functions and algorithms

## 📄 License & Support

This project is open source and available under the [MIT License](LICENSE).

### Support Channels
- **Documentation**: Comprehensive README and code comments
- **Issue Tracking**: GitHub issues for bug reports and features
- **Pull Requests**: Contributions welcome with clear descriptions

---

## 🎯 Summary

The **vuDataSim Cluster Manager** is a powerful load testing management platform that combines:
- **Real-time monitoring** via SSH-based metrics collection
- **Distributed node management** with automated deployment
- **Modern web interface** with live updates and visualizations
- **Scalable architecture** supporting multiple testing nodes

While currently using SSH for metrics collection (which has efficiency limitations), the platform provides a solid foundation for distributed load testing with plans for more efficient HTTP-based monitoring in future versions.

**Built with ❤️ for load testing and performance monitoring**