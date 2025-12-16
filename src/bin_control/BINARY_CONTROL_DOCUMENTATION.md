# Binary Control Package Documentation

## Overview

The `bin_control` package provides a comprehensive Go-based solution for managing and controlling binaries on remote nodes in a distributed system. It enables starting, stopping, monitoring, and debugging binaries across multiple nodes via SSH connections, with support for both individual node operations and bulk operations across all enabled nodes.

The package is designed for load testing and monitoring systems where binaries need to be deployed and managed across a cluster of machines. It handles configuration management, concurrent operations, error handling, and provides detailed logging and status reporting.

## Architecture

### Core Components

1. **BinaryControl**: Main controller struct that manages node configurations and operations
2. **Node Configuration**: YAML-based configuration system for defining cluster nodes
3. **SSH Operations**: Secure remote command execution with timeout and concurrency controls
4. **Status Monitoring**: Real-time status checking and detailed metrics collection
5. **Bulk Operations**: Concurrent operations across multiple nodes

### Key Features

- **Concurrent SSH Operations**: Limited to 3 concurrent connections to prevent resource exhaustion
- **Configuration Management**: Dynamic reloading of node configurations
- **Error Handling**: Comprehensive error handling with graceful degradation
- **Logging Integration**: Structured logging with node-specific context
- **Timeout Management**: Configurable timeouts for different operation types
- **Health Checks**: Built-in health verification for started services

## Data Structures

### NodeConfig
Represents a single node in the cluster:
```go
type NodeConfig struct {
    Host        string `yaml:"host"`        // IP address or hostname
    User        string `yaml:"user"`        // SSH username
    KeyPath     string `yaml:"key_path"`    // Path to SSH private key
    ConfDir     string `yaml:"conf_dir"`    // Configuration directory path
    BinaryDir   string `yaml:"binary_dir"` // Binary installation directory
    MetricsPort int    `yaml:"metrics_port"` // Port for metrics collection
    Description string `yaml:"description"` // Human-readable description
    Enabled     bool   `yaml:"enabled"`     // Whether node is active
}
```

### BinaryControl
Main controller struct:
```go
type BinaryControl struct {
    nodesConfigPath string           // Path to YAML config file
    nodesConfig     NodesConfig      // Loaded configuration
    configMutex     sync.RWMutex     // Protects config access
    sshSemaphore    chan struct{}    // Limits concurrent SSH ops
}
```

## Main Functions

### NewBinaryControl()

**Concept**: Factory function that creates and initializes a new BinaryControl instance.

**Logic**:
- Initializes the controller with default configuration path
- Creates empty node configuration map
- Sets up read-write mutex for thread-safe config access
- Initializes SSH semaphore with capacity of 3 for concurrency control

**Usage**:
```go
bc := bin_control.NewBinaryControl()
```

**Reason**: Provides a clean initialization pattern ensuring all internal state is properly set up before use. The semaphore prevents overwhelming remote nodes with too many concurrent SSH connections.

---

### LoadNodesConfig()

**Concept**: Loads and parses the YAML configuration file containing node definitions.

**Logic**:
- Acquires write lock to protect configuration updates
- Checks if config file exists
- Reads entire file content
- Unmarshals YAML into NodesConfig struct
- Releases lock

**Usage**:
```go
err := bc.LoadNodesConfig()
if err != nil {
    log.Fatal("Failed to load config:", err)
}
```

**Reason**: Configuration needs to be loaded before any node operations. The locking ensures thread safety when multiple goroutines might trigger config reloads. File existence check prevents cryptic YAML parsing errors.

---

### GetEnabledNodes()

**Concept**: Returns a filtered map containing only enabled nodes from the configuration.

**Logic**:
- Acquires read lock for thread-safe access
- Iterates through all configured nodes
- Filters nodes where `Enabled` field is true
- Returns new map with only enabled nodes

**Usage**:
```go
enabledNodes := bc.GetEnabledNodes()
for nodeName, config := range enabledNodes {
    fmt.Printf("Node %s: %s\n", nodeName, config.Host)
}
```

**Reason**: Most operations should only target enabled nodes. This prevents accidental operations on disabled/test nodes and improves performance by reducing unnecessary iterations.

---

### StartBinary(nodeName string, timeout int)

**Concept**: Starts the main application binary (`finalvudatasim`) on a specified remote node.

**Logic**:
1. **Validation Phase**:
   - Checks for nil controller
   - Validates timeout parameter (> 0)
   - Reloads configuration
   - Verifies node exists and is enabled

2. **Pre-Check Phase**:
   - Checks if binary is already running
   - Verifies binary file exists on remote node

3. **Execution Phase**:
   - Constructs appropriate start command with timeout handling
   - Executes SSH command in background
   - Handles SSH timeouts gracefully

4. **Verification Phase**:
   - Waits briefly for startup
   - Checks binary status
   - Returns appropriate response

**Usage**:
```go
response, err := bc.StartBinary("node-01", 300) // 5 minute timeout
if err != nil {
    log.Printf("Start failed: %v", err)
} else if response.Success {
    log.Printf("Binary started: %s", response.Message)
}
```

**Reason**: The timeout parameter allows for test scenarios where binaries should auto-terminate. Background execution prevents SSH session blocking. Multiple verification steps ensure reliability despite network latencies.

---

### StopBinary(nodeName string, timeout int)

**Concept**: Gracefully stops the running binary on a specified node.

**Logic**:
1. **Validation Phase**:
   - Nil check and config reload
   - Node existence and enablement verification

2. **Status Check Phase**:
   - Verifies binary is actually running
   - Gets detailed status including PID

3. **Termination Phase**:
   - Attempts graceful kill (SIGTERM)
   - Falls back to force kill (SIGKILL) if graceful fails
   - Waits for process termination

4. **Verification Phase**:
   - Confirms binary is stopped
   - Returns detailed status

**Usage**:
```go
response, err := bc.StopBinary("node-01", 30)
if response.Success {
    log.Printf("Binary stopped successfully")
}
```

**Reason**: Graceful shutdown allows processes to clean up resources. Force kill fallback ensures termination even for unresponsive processes. The timeout parameter is included for API consistency but primarily used for logging.

---

### StartMetricsBinary(nodeName string, timeout int)

**Concept**: Starts the metrics collection API binary (`node_metrics_api`) on a node.

**Logic**:
1. **Validation Phase**: Standard node validation
2. **Pre-Check Phase**:
   - Checks if metrics API is already running
   - Verifies binary exists and is executable

3. **Execution Phase**:
   - Starts binary with logging redirection
   - Waits for startup completion

4. **Health Check Phase**:
   - Performs HTTP health check on port 8086
   - Verifies API responsiveness

**Usage**:
```go
response, err := bc.StartMetricsBinary("node-01", 60)
if response.Success {
    log.Printf("Metrics API running on port 8086")
}
```

**Reason**: Metrics API requires health verification beyond just process existence. The HTTP health check ensures the service is actually functional, not just running.

---

### StopMetricsBinary(nodeName string, timeout int)

**Concept**: Stops all instances of the metrics API binary on a node.

**Logic**:
1. **Validation Phase**: Standard validation
2. **Discovery Phase**:
   - Finds all running metrics API processes
   - Parses PIDs from pgrep output

3. **Termination Phase**:
   - Attempts graceful kill on each PID
   - Falls back to force kill if needed
   - Tracks successful stops

4. **Verification Phase**:
   - Confirms all processes are stopped

**Usage**:
```go
response, err := bc.StopMetricsBinary("node-01", 30)
log.Printf("Stopped %d processes", response.Data.(map[string]interface{})["stoppedCount"])
```

**Reason**: Metrics API might have multiple instances. Process enumeration ensures all instances are stopped. Individual PID handling prevents partial failures.

---

### DebugMetricsBinary(nodeName string)

**Concept**: Performs comprehensive debugging of the metrics binary on a node.

**Logic**:
1. **Binary File Check**: Verifies file existence and permissions
2. **Process Check**: Lists running processes
3. **Port Check**: Verifies port 8086 availability
4. **Log Analysis**: Retrieves startup logs
5. **Manual Test**: Attempts manual binary execution
6. **System Resources**: Checks disk and memory

**Usage**:
```go
debugInfo, err := bc.DebugMetricsBinary("node-01")
if err == nil {
    fmt.Printf("Debug info: %+v", debugInfo.Data)
}
```

**Reason**: Debugging requires comprehensive system state information. Multiple checks help identify whether issues are with the binary, system resources, or configuration.

---

### GetBinaryStatus(nodeName string)

**Concept**: Performs lightweight status check of the main binary on a node.

**Logic**:
1. **Validation Phase**: Config reload and node checks
2. **Status Check**: Uses `pgrep` to find running processes
3. **Response**: Returns basic running/stopped status

**Usage**:
```go
status, err := bc.GetBinaryStatus("node-01")
if status.Status == "running" {
    log.Printf("Binary is running")
}
```

**Reason**: Lightweight check suitable for frequent monitoring. Avoids expensive operations like detailed process metrics collection.

---

### getDetailedBinaryStatus(nodeName string)

**Concept**: Collects comprehensive status and metrics for a running binary.

**Logic**:
1. **PID Discovery**: Finds process ID
2. **Metrics Collection**:
   - Start time from `ps` command
   - CPU/memory usage
   - Process command line
3. **Validation**: Ensures PID is valid (> 0)

**Usage**: Internal function called by status collection operations.

**Reason**: Detailed metrics are expensive to collect, so separated from basic status checks. Provides comprehensive monitoring data when needed.

---

### GetAllBinaryStatuses()

**Concept**: Concurrently retrieves status for all enabled nodes.

**Logic**:
1. **Preparation**: Gets list of enabled nodes
2. **Concurrent Execution**: Launches goroutines for each node
3. **Collection**: Gathers results with timeout protection
4. **Aggregation**: Combines results into structured response

**Usage**:
```go
allStatuses, err := bc.GetAllBinaryStatuses()
statuses := allStatuses.Data.([]BinaryStatus)
for _, status := range statuses {
    fmt.Printf("%s: %s\n", status.NodeName, status.Status)
}
```

**Reason**: Bulk operations improve efficiency for monitoring dashboards. Concurrency reduces total time for status checks across multiple nodes.

---

### StartAllBinaries(timeout int)

**Concept**: Concurrently starts binaries on all enabled nodes.

**Logic**:
1. **Preparation**: Config reload and node enumeration
2. **Concurrent Launch**: Goroutines for each node start operation
3. **Result Collection**: Gathers results with timeout
4. **Aggregation**: Summarizes success/failure counts

**Usage**:
```go
bulkResult, err := bc.StartAllBinaries(300)
log.Printf("Started %d/%d nodes", bulkResult.Data.Successful, bulkResult.Data.TotalNodes)
```

**Reason**: Bulk operations enable cluster-wide deployments. Concurrency maximizes speed while respecting SSH connection limits.

---

### StopAllBinaries(timeout int)

**Concept**: Concurrently stops binaries on all enabled nodes.

**Logic**: Similar to StartAllBinaries but for stop operations, with extended collection timeout (60 seconds) due to potential graceful shutdown delays.

**Usage**:
```go
bulkResult, err := bc.StopAllBinaries(30)
log.Printf("Stopped %d/%d nodes", bulkResult.Data.Successful, bulkResult.Data.TotalNodes)
```

**Reason**: Cluster-wide shutdown requires coordination. Extended timeout accommodates graceful shutdown periods.

---

### sshExec(node NodeConfig, command string)

**Concept**: Executes a command on a remote node via SSH without capturing output.

**Logic**:
1. **Concurrency Control**: Acquires semaphore slot
2. **Command Construction**: Builds SSH arguments with security options
3. **Execution**: Runs command with 30-second timeout
4. **Error Handling**: Graceful handling of timeouts and connection issues

**Usage**: Internal function for fire-and-forget commands.

**Reason**: Semaphore prevents SSH connection exhaustion. Timeout prevents hanging on unresponsive commands. Security options ensure safe connections.

---

### sshExecWithOutput(node NodeConfig, command string)

**Concept**: Executes a command on a remote node and captures output.

**Logic**: Similar to sshExec but captures stdout/stderr with 10-second timeout for faster feedback on info commands.

**Usage**: Internal function for commands requiring output analysis.

**Reason**: Shorter timeout for interactive commands. Output capture enables status checking and debugging.

## Error Handling

The package implements comprehensive error handling:

- **Nil Checks**: Prevents panics from nil receivers
- **Validation**: Input parameter validation
- **Graceful Degradation**: Continues operation when possible
- **Timeout Handling**: Prevents indefinite hangs
- **SSH Error Classification**: Distinguishes between different failure types

## Concurrency Management

- **SSH Semaphore**: Limits concurrent connections to 3
- **Mutex Protection**: Thread-safe configuration access
- **Goroutine Coordination**: Proper synchronization in bulk operations
- **Timeout Channels**: Prevents goroutine leaks

## Configuration Management

The package uses YAML configuration for node definitions:

```yaml
cluster_settings:
  backup_retention_days: 30
  conflict_resolution: "latest"
  connection_timeout: 10
  max_retries: 3
  sync_timeout: 300

nodes:
  node-01:
    host: "192.168.1.10"
    user: "admin"
    key_path: "/home/admin/.ssh/id_rsa"
    conf_dir: "/opt/vudatasim/config"
    binary_dir: "/opt/vudatasim/bin"
    metrics_port: 8086
    description: "Primary load generator"
    enabled: true
```

## Logging Integration

All operations integrate with the custom logger package:

- **Node-specific logging**: `logger.LogWithNode(nodeName, operation, message, level)`
- **Structured logging**: Consistent log levels and context
- **Success/Error tracking**: Comprehensive operation tracking

## Usage Patterns

### Individual Node Management
```go
bc := bin_control.NewBinaryControl()
bc.LoadNodesConfig()

// Start binary with 5-minute timeout
response, _ := bc.StartBinary("node-01", 300)

// Check status
status, _ := bc.GetBinaryStatus("node-01")

// Stop binary
bc.StopBinary("node-01", 30)
```

### Bulk Operations
```go
// Start all enabled nodes
bulkStart, _ := bc.StartAllBinaries(300)

// Check all statuses
allStatuses, _ := bc.GetAllBinaryStatuses()

// Stop all nodes
bulkStop, _ := bc.StopAllBinaries(30)
```

### Metrics API Management
```go
// Start metrics collection
bc.StartMetricsBinary("node-01", 60)

// Debug if issues occur
debugInfo, _ := bc.DebugMetricsBinary("node-01")

// Stop metrics
bc.StopMetricsBinary("node-01", 30)
```

## Best Practices

1. **Always load config** before operations
2. **Check node status** before starting/stopping
3. **Use appropriate timeouts** for different scenarios
4. **Handle bulk operation results** for monitoring
5. **Implement retry logic** for transient failures
6. **Monitor SSH semaphore usage** in high-load scenarios

## Performance Considerations

- SSH operations are the primary bottleneck
- Semaphore limits prevent connection exhaustion
- Concurrent bulk operations maximize throughput
- Configuration caching reduces file I/O
- Timeout management prevents resource leaks

## Security Features

- SSH key-based authentication only
- Strict host key checking disabled for automation
- Command injection prevention through proper quoting
- Timeout limits prevent DoS scenarios
- Limited concurrent connections prevent resource abuse