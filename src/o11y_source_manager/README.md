# O11y Source Manager

The O11y Source Manager is a comprehensive system for managing observability (o11y) source configurations and distributing Events Per Second (EPS) across different data sources in the vuDataSim load testing tool.

## Overview

This module provides functionality to:
- **Load and manage** o11y source configurations from YAML files
- **Calculate EPS** based on configuration parameters (NumUniqKey × submodule keys)
- **Distribute EPS proportionally** across selected sources based on maximum EPS limits
- **Enable/disable** o11y sources dynamically
- **Update configurations** in real-time without restarting the simulator

## Architecture

### Core Components

1. **O11ySourceManager**: Main manager class that orchestrates all operations
2. **Configuration Readers**: YAML parsers for different configuration files
3. **EPS Calculator**: Logic for computing total EPS from configuration parameters
4. **Distribution Algorithm**: Proportional EPS distribution based on max EPS values
5. **Configuration Writers**: Functions to update YAML files with new values

### Configuration Files

The system reads from several configuration sources:

- **`src/configs/max_eps.yaml`**: Maximum EPS limits for each o11y source
- **`src/conf.d/conf.yml`**: Main configuration with global settings and source enable/disable flags
- **`src/conf.d/{SourceName}/conf.yml`**: Individual source configurations with NumUniqKey settings
- **`src/conf.d/{SourceName}/*.yml`**: Submodule configuration files

## API Endpoints

### Get Available Sources
```bash
GET /api/o11y/sources
```
Returns a list of all available o11y sources.

**Example Response:**
```json
{
  "success": true,
  "data": [
    "LinuxMonitor",
    "Apache",
    "Nginx",
    "Mssql",
    "K8s"
  ]
}
```

### Get Enabled Sources
```bash
GET /api/o11y/sources/enabled
```
Returns a list of currently enabled o11y sources.

### Get Source Details
```bash
GET /api/o11y/sources/{sourceName}
```
Returns detailed EPS information for a specific source.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "sourceName": "LinuxMonitor",
    "assignedEps": 50000,
    "mainUniqueKeys": 2000,
    "totalSubKeys": 25,
    "subModuleBreakdown": {
      "cpu": 2000,
      "core": 2000,
      "memory": 2000,
      "diskio": 2000,
      "filesystem": 2000,
      "load": 2000,
      "network": 2000,
      "process": 2000,
      "socket": 2000,
      "socket_summary": 2000
    }
  }
}
```

### Get Current EPS Distribution
```bash
GET /api/o11y/eps/current
```
Returns the current total EPS and breakdown across all sources.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "totalEPS": 120000,
    "breakdown": {
      "LinuxMonitor": {
        "sourceName": "LinuxMonitor",
        "assignedEps": 50000,
        "mainUniqueKeys": 2000,
        "totalSubKeys": 25
      },
      "Apache": {
        "sourceName": "Apache",
        "assignedEps": 42000,
        "mainUniqueKeys": 1680,
        "totalSubKeys": 25
      },
      "Mssql": {
        "sourceName": "Mssql",
        "assignedEps": 28000,
        "mainUniqueKeys": 1120,
        "totalSubKeys": 25
      }
    }
  }
}
```

### Distribute EPS
```bash
POST /api/o11y/eps/distribute
Content-Type: application/json

{
  "selectedSources": ["LinuxMonitor", "Apache", "Mssql"],
  "totalEps": 100000
}
```

Distributes the specified total EPS across the selected sources proportionally based on their maximum EPS limits. **Note:** This will ENABLE the selected sources and DISABLE all other sources in the main configuration.

**Example Response:**
```json
{
  "success": true,
  "message": "Successfully distributed 100000 EPS across 3 sources",
  "data": {
    "totalEps": 100000,
    "selectedSources": ["LinuxMonitor", "Apache", "Mssql"],
    "sourceBreakdown": {
      "LinuxMonitor": {
        "sourceName": "LinuxMonitor",
        "assignedEps": 35000,
        "mainUniqueKeys": 1400,
        "totalSubKeys": 25
      },
      "Apache": {
        "sourceName": "Apache",
        "assignedEps": 30000,
        "mainUniqueKeys": 1200,
        "totalSubKeys": 25
      },
      "Mssql": {
        "sourceName": "Mssql",
        "assignedEps": 35000,
        "mainUniqueKeys": 1400,
        "totalSubKeys": 25
      }
    },
    "newTotalEps": 100000
  }
}
```

### Enable Source
```bash
POST /api/o11y/sources/{sourceName}/enable
```
Enables a specific o11y source.

**Example Response:**
```json
{
  "success": true,
  "message": "Source LinuxMonitor enabled successfully"
}
```

### Disable Source
```bash
POST /api/o11y/sources/{sourceName}/disable
```
Disables a specific o11y source.

### Get Max EPS Configuration
```bash
GET /api/o11y/max-eps
```
Returns the maximum EPS configuration for all sources.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "Apache": 42000,
    "AWS_ALB": 53000,
    "AzureStorageBlob": 51000,
    "LinuxMonitor": 50000,
    "Mssql": 89000,
    "Nginx": 42000,
    "K8s": 48000
  }
}
```

## EPS Calculation Logic

### Formula
```
Total EPS = MainUniqueKeys × TotalSubModuleKeys
```

Where:
- **MainUniqueKeys**: Value from `NumUniqKey` in the source's main `conf.yml`
- **TotalSubModuleKeys**: Sum of `NumUniqKey` values from all submodule files (or 1 if not specified)

### Distribution Algorithm

1. **Proportional Distribution**: EPS is distributed based on each source's maximum EPS limit
2. **Ratio Calculation**: `SourceRatio = SourceMaxEPS / TotalMaxEPS`
3. **EPS Assignment**: `AssignedEPS = TotalEPS × SourceRatio`
4. **Main Keys Calculation**: `MainUniqueKeys = AssignedEPS / TotalSubModuleKeys`

## Manual Testing with curl Commands

Here are detailed examples of how to test each API endpoint manually using curl commands. Make sure your vuDataSim server is running on `http://localhost:3000`.

### 1. Get All Available Sources
**Endpoint:** `GET /api/o11y/sources`

```bash
curl -X GET http://localhost:3000/api/o11y/sources
```

**Expected Response:**
```json
{
  "success": true,
  "data": [
    "Apache",
    "AWS_ALB",
    "AzureStorageBlob",
    "BRUM",
    "CiscoIoSSwitch",
    "LinuxMonitor",
    "Mssql",
    "Nginx",
    "K8s",
    "Tomcat"
  ]
}
```

### 2. Get Currently Enabled Sources
**Endpoint:** `GET /api/o11y/sources/enabled`

```bash
curl -X GET http://localhost:3000/api/o11y/sources/enabled
```

**Expected Response:**
```json
{
  "success": true,
  "data": [
    "LinuxMonitor",
    "Mssql"
  ]
}
```

### 3. Get Source Details
**Endpoint:** `GET /api/o11y/sources/{sourceName}`

```bash
# Get LinuxMonitor details
curl -X GET http://localhost:3000/api/o11y/sources/LinuxMonitor

# Get Apache details
curl -X GET http://localhost:3000/api/o11y/sources/Apache
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "sourceName": "LinuxMonitor",
    "assignedEps": 50000,
    "mainUniqueKeys": 2000,
    "totalSubKeys": 25,
    "subModuleBreakdown": {
      "cpu": 2000,
      "core": 2000,
      "diskio": 2000,
      "filesystem": 2000,
      "load": 2000,
      "memory": 2000,
      "network": 2000,
      "process": 2000,
      "socket": 2000,
      "socket_summary": 2000
    }
  }
}
```

### 4. Get Current EPS Distribution
**Endpoint:** `GET /api/o11y/eps/current`

```bash
curl -X GET http://localhost:3000/api/o11y/eps/current
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "totalEPS": 120000,
    "breakdown": {
      "LinuxMonitor": {
        "sourceName": "LinuxMonitor",
        "assignedEps": 50000,
        "mainUniqueKeys": 2000,
        "totalSubKeys": 25
      },
      "Mssql": {
        "sourceName": "Mssql",
        "assignedEps": 70000,
        "mainUniqueKeys": 2800,
        "totalSubKeys": 25
      }
    }
  }
}
```

### 5. Get Maximum EPS Configuration
**Endpoint:** `GET /api/o11y/max-eps`

```bash
curl -X GET http://localhost:3000/api/o11y/max-eps
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "Apache": 42000,
    "AWS_ALB": 53000,
    "AzureStorageBlob": 51000,
    "BRUM": 40000,
    "CiscoIoSSwitch": 48000,
    "LinuxMonitor": 50000,
    "Mssql": 89000,
    "Nginx": 42000,
    "K8s": 48000,
    "Tomcat": 65000
  }
}
```

### 6. Distribute EPS Across Sources
**Endpoint:** `POST /api/o11y/eps/distribute`

```bash
# Distribute 100k EPS across LinuxMonitor and Apache
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache"],
    "totalEps": 100000
  }'

# Distribute 150k EPS across 3 sources
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache", "Mssql"],
    "totalEps": 150000
  }'

# Distribute 200k EPS across 4 sources
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache", "Mssql", "K8s"],
    "totalEps": 200000
  }'
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Successfully distributed 100000 EPS across 2 sources",
  "data": {
    "totalEps": 100000,
    "selectedSources": ["LinuxMonitor", "Apache"],
    "sourceBreakdown": {
      "LinuxMonitor": {
        "sourceName": "LinuxMonitor",
        "assignedEps": 54348,
        "mainUniqueKeys": 2174,
        "totalSubKeys": 25
      },
      "Apache": {
        "sourceName": "Apache",
        "assignedEps": 45652,
        "mainUniqueKeys": 1826,
        "totalSubKeys": 25
      }
    },
    "newTotalEps": 100000,
    "updatedConfigs": {
      "LinuxMonitor": 2174,
      "Apache": 1826
    }
  }
}
```

### 7. Enable O11y Sources
**Endpoint:** `POST /api/o11y/sources/{sourceName}/enable`

```bash
# Enable Apache
curl -X POST http://localhost:3000/api/o11y/sources/Apache/enable

# Enable Nginx
curl -X POST http://localhost:3000/api/o11y/sources/Nginx/enable

# Enable K8s
curl -X POST http://localhost:3000/api/o11y/sources/K8s/enable

# Enable Tomcat
curl -X POST http://localhost:3000/api/o11y/sources/Tomcat/enable
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Source Apache enabled successfully"
}
```

### 8. Disable O11y Sources
**Endpoint:** `POST /api/o11y/sources/{sourceName}/disable`

```bash
# Disable LinuxMonitor
curl -X POST http://localhost:3000/api/o11y/sources/LinuxMonitor/disable

# Disable Mssql
curl -X POST http://localhost:3000/api/o11y/sources/Mssql/disable

# Disable Apache
curl -X POST http://localhost:3000/api/o11y/sources/Apache/disable
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Source LinuxMonitor disabled successfully"
}
```

## Practical Testing Scenarios

### Scenario 1: Basic Setup
```bash
# 1. Check what sources are available
curl -X GET http://localhost:3000/api/o11y/sources

# 2. Check current EPS distribution
curl -X GET http://localhost:3000/api/o11y/eps/current

# 3. Distribute 100k EPS across selected sources (this will ENABLE only these sources and DISABLE all others)
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache"],
    "totalEps": 100000
  }'

# 4. Verify the distribution and enabled sources
curl -X GET http://localhost:3000/api/o11y/eps/current
curl -X GET http://localhost:3000/api/o11y/sources/enabled
```

### Scenario 2: Scale Testing
```bash
# 1. Distribute 500k EPS across multiple sources (this enables only these sources)
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache", "Mssql", "K8s"],
    "totalEps": 500000
  }'

# 2. Check detailed breakdown and enabled sources
curl -X GET http://localhost:3000/api/o11y/sources/enabled
curl -X GET http://localhost:3000/api/o11y/sources/LinuxMonitor
curl -X GET http://localhost:3000/api/o11y/sources/Apache
curl -X GET http://localhost:3000/api/o11y/sources/Mssql
curl -X GET http://localhost:3000/api/o11y/sources/K8s
```

### Scenario 3: Dynamic Adjustment
```bash
# 1. Start with 100k EPS across 2 sources
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache"],
    "totalEps": 100000
  }'

# 2. Scale up to 200k EPS (same sources, different EPS)
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache"],
    "totalEps": 200000
  }'

# 3. Change sources and redistribute (this will disable LinuxMonitor and Apache, enable Mssql)
curl -X POST http://localhost:3000/api/o11y/eps/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "selectedSources": ["LinuxMonitor", "Apache", "Mssql"],
    "totalEps": 250000
  }'

# 4. Check which sources are now enabled
curl -X GET http://localhost:3000/api/o11y/sources/enabled
```

## Command Reference

| Action | Method | Endpoint | Description |
|--------|--------|----------|-------------|
| List Sources | GET | `/api/o11y/sources` | Get all available o11y sources |
| List Enabled | GET | `/api/o11y/sources/enabled` | Get currently enabled sources |
| Source Details | GET | `/api/o11y/sources/{name}` | Get detailed EPS info for a source |
| Current EPS | GET | `/api/o11y/eps/current` | Get current EPS distribution |
| Max EPS Config | GET | `/api/o11y/max-eps` | Get maximum EPS configuration |
| Distribute EPS | POST | `/api/o11y/eps/distribute` | Distribute EPS across sources |
| Enable Source | POST | `/api/o11y/sources/{name}/enable` | Enable a specific source |
| Disable Source | POST | `/api/o11y/sources/{name}/disable` | Disable a specific source |

## Tips for Testing

1. **Always check available sources first** to see what you can work with
2. **Start small** with 2-3 sources and smaller EPS values (10k-50k)
3. **Verify after each change** by checking the current EPS distribution
4. **Use detailed view** to see how EPS is distributed across submodules
5. **Scale gradually** to avoid overwhelming your system
6. **Check logs** if you encounter errors - the system provides detailed error messages
7. **Note on source enabling**: The `/api/o11y/eps/distribute` endpoint will ENABLE only the selected sources and DISABLE all others automatically

## Error Handling Examples

If you get an error response, it will look like this:

```json
{
  "success": false,
  "message": "Source not found: InvalidSource"
}
```

Common error scenarios:
- **Source not found**: Check the source name spelling (case-sensitive)
- **Invalid EPS value**: Ensure totalEps is a positive integer
- **No sources selected**: Provide at least one source in selectedSources array
- **Configuration errors**: Check that configuration files exist and are valid YAML

## Configuration File Structure

### Main Configuration (src/conf.d/conf.yml)
```yaml
include_module_dirs:
  LinuxMonitor:
    enabled: true
  Apache:
    enabled: false
  Mssql:
    enabled: true
  # ... other sources
```

### Source Configuration (src/conf.d/LinuxMonitor/conf.yml)
```yaml
enabled: true
uniquekey:
  name: "host"
  DataType: IPv4
  ValueType: "RandomFixed"
  Value: "10.10.10.1"
  NumUniqKey: 2000  # This value is modified by the EPS manager
period: 1s
Include_sub_modules:
  - cpu
  - core
  - memory
  # ... other submodules
```

### Max EPS Configuration (src/configs/max_eps.yaml)
```yaml
max_eps_config:
  LinuxMonitor: 50000
  Apache: 42000
  Mssql: 89000
  Nginx: 42000
  # ... other sources
```

## Error Handling

The system includes comprehensive error handling for:

- **File I/O errors**: When reading/writing configuration files
- **YAML parsing errors**: When configuration files have invalid syntax
- **Validation errors**: When EPS values or source names are invalid
- **Configuration inconsistencies**: When source configs don't match max EPS config

## Integration with Main Application

The O11y Source Manager integrates seamlessly with the existing vuDataSim application:

1. **Initialization**: Automatically loads all configurations on startup
2. **Real-time Updates**: Changes are applied immediately without restart
3. **WebSocket Integration**: Updates are broadcast to connected clients
4. **Node Management**: Works alongside existing node and binary management features

## Best Practices

1. **Backup Configurations**: Always backup configuration files before making bulk changes
2. **Validate EPS Values**: Ensure EPS values are within reasonable limits for your system
3. **Monitor Distribution**: Check EPS distribution after making changes to ensure expected results
4. **Use Proportional Distribution**: Leverage the max EPS configuration for optimal load balancing

## Troubleshooting

### Common Issues

1. **"Source not found" errors**: Ensure the source name matches exactly (case-sensitive)
2. **"Failed to load config" errors**: Check file permissions and YAML syntax
3. **Unexpected EPS values**: Verify submodule configurations and NumUniqKey calculations
4. **Distribution failures**: Check that selected sources exist in max_eps.yaml

### Debug Mode

Enable debug logging by modifying the logging configuration in `src/conf.d/conf.yml`:

```yaml
logging:
  level: debug
  console: true
  file: true
```

This will provide detailed information about configuration loading, EPS calculations, and distribution operations.