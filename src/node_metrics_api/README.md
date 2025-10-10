# Node Metrics API

A lightweight HTTP server that collects and serves real-time system metrics from Linux nodes using the `/proc` filesystem.

## Overview

The Node Metrics API is designed to run on each load testing node to provide efficient, local metrics collection without the overhead of SSH polling. It serves as a replacement for the SSH-based metrics collection in the vuDataSim cluster manager.

## Features

- **Local System Metrics Collection**: Uses `/proc` filesystem for high-performance metrics gathering
- **Lightweight HTTP Server**: Minimal resource footprint
- **Real-time Updates**: Collects metrics every second in background
- **Standard JSON API**: Compatible with existing monitoring systems
- **Configurable Port**: Environment variable configuration
- **Health Check Endpoint**: Built-in health monitoring

## Endpoints

### GET /api/system/metrics

Returns comprehensive system metrics in JSON format:

```json
{
  "nodeId": "node1",
  "timestamp": "2024-10-10T11:51:44Z",
  "system": {
    "cpu": {
      "used_percent": 57.5,
      "cores": 4,
      "load_1m": 0.52
    },
    "memory": {
      "used_gb": 4.2,
      "available_gb": 3.8,
      "total_gb": 8.0,
      "used_percent": 52.5
    },
    "uptime_seconds": 900
  }
}
```

### GET /api/system/health

Returns health status information:

```json
{
  "status": "healthy",
  "nodeId": "node1",
  "timestamp": "2024-10-10T11:51:44Z",
  "uptime": "2m30s"
}
```

### GET /

Returns basic server information:

```json
{
  "status": "Node Metrics API is running",
  "nodeId": "node1",
  "version": "1.0.0"
}
```

## Configuration

Configure the server using environment variables:

- `METRICS_PORT`: Port to listen on (default: 8080)
- `NODE_ID`: Node identifier (default: hostname)

## Installation

1. **Build the binary**:
   ```bash
   make build
   ```

2. **Cross-compile for Linux** (for deployment):
   ```bash
   make build-linux
   ```

3. **Run the server**:
   ```bash
   make run
   ```

## Deployment

The Node Metrics API binary should be deployed alongside the main vuDataSim binary in the `bin/` directory on each node:

```
/home/vunet/vuDataSim/
├── bin/
│   ├── finalvudatasim      # Main load testing binary
│   └── node_metrics_api    # Metrics API binary
└── conf.d/                 # Configuration directory
```

## System Requirements

- Linux operating system
- `/proc` filesystem mounted (standard on Linux)
- Read access to system files in `/proc`

## Performance

- **Memory Usage**: < 10MB
- **CPU Usage**: < 1% average
- **Network**: Minimal outbound traffic
- **Collection Frequency**: 1 second intervals
- **Response Time**: < 10ms for API calls

## Integration

The Node Metrics API integrates with the central vuDataSim cluster manager, which polls each node's metrics endpoint every 5 seconds instead of using SSH.

## Development

### Build Requirements

- Go 1.21 or later
- Make (optional, for using Makefile)

### Development Commands

```bash
# Install dependencies
make deps

# Format code
make fmt

# Run tests
make test

# Build for current platform
make build

# Cross-compile for Linux
make build-linux

# Clean build artifacts
make clean
```

## Metrics Collection Details

### CPU Metrics

- **used_percent**: Calculated from `/proc/stat` (total - idle / total * 100)
- **cores**: Counted from `/proc/cpuinfo` processor entries
- **load_1m**: Parsed from `/proc/loadavg` 1-minute load average

### Memory Metrics

- **used_gb**: (MemTotal - MemAvailable) from `/proc/meminfo`
- **available_gb**: MemAvailable from `/proc/meminfo`
- **total_gb**: MemTotal from `/proc/meminfo`
- **used_percent**: (used_gb / total_gb) * 100

### System Uptime

- **uptime_seconds**: Parsed from `/proc/uptime`

## Error Handling

The API includes comprehensive error handling for:

- File system access errors (falls back to safe defaults)
- JSON encoding/decoding errors
- HTTP request method validation
- Environment variable parsing

## Security Considerations

- Runs as non-privileged user
- Read-only access to system files
- No authentication required (intended for internal network use)
- Consider firewall rules to restrict access to trusted networks

## Troubleshooting

### Common Issues

1. **Port already in use**: Change `METRICS_PORT` environment variable
2. **Permission denied**: Ensure read access to `/proc` filesystem
3. **High CPU usage**: Check collection frequency and system load
4. **Connection refused**: Verify server is running and port is accessible

### Debug Logging

Enable verbose logging by setting log level in your application or checking server logs for detailed error information.