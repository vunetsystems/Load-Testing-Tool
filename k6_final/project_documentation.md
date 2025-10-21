# K6 Dashboard Performance Testing Project Documentation

## Overview

This project is a comprehensive performance testing suite for Grafana dashboards in a vuSmartMaps observability platform. It uses the K6 load testing framework to simulate multiple users accessing various dashboards and their panels, measuring response times, success rates, and overall dashboard performance.

## Project Purpose

The system performs automated performance testing and monitoring of Grafana dashboards within the vuSmartMaps platform. It tests multiple dashboard types including:

- **Linux Server Insights**: System monitoring dashboards
- **MSSQL Overview Dashboard**: Database performance monitoring
- **Traces Dashboards**: APM (Application Performance Monitoring) trace analysis
- **Alert Dashboards**: Alert rule execution and monitoring
- **Log Analytics**: Log data analysis and visualization
- **Reports**: Report generation and performance
- **Login Testing**: Authentication performance

## Architecture

### Core Components

1. **Main Orchestrator** (`k6_all.sh`)
   - Master script that runs all dashboard tests sequentially
   - Configurable intervals between test executions
   - Centralized logging

2. **Dashboard Test Scripts** (`.js` files)
   - K6-based performance tests for individual dashboards
   - Panel-level testing with time range parameters
   - Custom metrics collection

3. **Wrapper Scripts** (`overall-1.sh` files)
   - Bash wrappers that execute K6 tests
   - Result processing and CSV generation
   - ClickHouse data insertion

4. **User Management** (`user_creation/`)
   - Automated user creation in Keycloak
   - Cookie harvesting for authenticated testing
   - Multi-user simulation

## Installation and Dependencies

### System Requirements

- **K6**: Load testing framework (v0.45+)
- **Node.js**: For JavaScript execution
- **Python 3.8+**: For user creation scripts
- **kubectl**: For Kubernetes/ClickHouse access
- **Playwright**: For browser automation in user creation

### Python Dependencies

```bash
pip install requests playwright
playwright install
```

### K6 Installation

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install k6

# Or download from https://k6.io/docs/get-started/installation/
```

## Configuration

### Environment Variables

The system uses environment variables for configuration:

- `TIME_FROM`: Start time for dashboard queries (default: 'now-15m')
- `TIME_TO`: End time for dashboard queries (default: 'now')
- `K6_INSECURE_SKIP_TLS_VERIFY=true`: Skip SSL verification for HTTPS requests

### Script Parameters

Most test scripts accept these parameters:
- `time-range`: Time range for queries (e.g., '15m', '1h', '6h')
- `vus`: Number of virtual users
- `iterations`: Number of test iterations
- `interval`: Sleep time between tests (seconds)

### User Configuration

Users are defined in `user_creation/user_cookies.txt` with format:
```
username,password,vunet_session,X-VuNet-HTTP-Info,grafana_session_expiry
```

## Execution Flow

### 1. User Creation Process

```bash
cd user_creation
python dashboard_user.py <num_users>
```

This creates users in Keycloak and harvests authentication cookies for testing.

### 2. Main Execution

```bash
./k6_all.sh
```

The main script runs dashboard tests in sequence with configurable intervals.

### 3. Individual Dashboard Testing

Each dashboard can be tested individually:

```bash
# Linux/MSSQL Dashboard
./k6_dashboard_name/linux-mssql-dashboard/overall-1.sh 6h 5 5 10

# Traces Dashboard
./k6_dashboard_name/traces/overall-1.sh 15m 1 1 10
```

### 4. Result Processing

Results are automatically:
- Saved to CSV files with detailed metrics
- Inserted into ClickHouse database (`monitoring.k6_results` table)
- Logged with timestamps and execution details

## File Structure and Order of Execution

### Execution Hierarchy

1. **k6_all.sh** (Main orchestrator)
   - Calls individual dashboard wrapper scripts
   - Manages execution intervals
   - Central logging

2. **overall-1.sh** (Dashboard wrappers)
   - Set up result directories and files
   - Execute K6 tests for each dashboard
   - Process results and generate summaries
   - Insert data into ClickHouse

3. **Dashboard JS files** (.js)
   - K6 test scripts for individual dashboards
   - Handle authentication and panel testing
   - Collect custom metrics

### Key Directories

- `k6_dashboard_name/`: Main test scripts organized by dashboard type
- `results/`: Test result outputs
- `logs/`: Execution logs
- `user_creation/`: User management scripts

## What Each Component Does

### Dashboard Test Scripts

Each `.js` file performs:

1. **Authentication**: Uses harvested cookies for login
2. **Dashboard Metadata**: Fetches dashboard JSON structure
3. **Panel Discovery**: Extracts panel information from dashboard
4. **Panel Testing**: Tests each panel with time range parameters
5. **Metrics Collection**: Records response times, success rates, custom metrics

### Metrics Collected

- **Dashboard Level**:
  - Average response time
  - Success rate
  - Status codes

- **Panel Level**:
  - Individual panel response times
  - Success/failure counts
  - Custom panel-specific metrics

### Result Formats

#### CSV Output Format
```csv
timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations
```

#### Summary Files
Human-readable summaries with panel performance details.

#### ClickHouse Storage
Data inserted into `monitoring.k6_results` table for long-term storage and analysis.

## Configuration Options

### Time Range Configuration

- Supports Grafana time formats: `now-15m`, `now-1h`, `now-6h`, etc.
- Configurable via `TIME_FROM` and `TIME_TO` environment variables
- Default: Last 15 minutes

### Load Configuration

- **VUs**: Number of concurrent virtual users
- **Iterations**: Total test iterations per dashboard
- **Intervals**: Sleep time between tests

### Dashboard-Specific Settings

Each dashboard has unique panel counts and IDs:
- Linux Server Insights: Up to 400 panels
- MSSQL Overview: Up to 300 panels
- Traces: Up to 150 panels (varies by trace type)

## Getting Started

### Quick Start

1. **Install Dependencies**
   ```bash
   # Install K6
   sudo apt install k6

   # Install Python dependencies
   pip install requests playwright
   playwright install
   ```

2. **Create Test Users**
   ```bash
   cd user_creation
   python dashboard_user.py 10  # Create 10 test users
   ```

3. **Run Tests**
   ```bash
   ./k6_all.sh
   ```

### Advanced Usage

- **Single Dashboard Testing**: Run individual `overall-1.sh` scripts
- **Custom Time Ranges**: Set `TIME_FROM` and `TIME_TO` variables
- **Load Variation**: Adjust VUs and iterations in script calls
- **Monitoring**: Check logs in `logs/` directory and results in `results/`

## Troubleshooting

### Common Issues

1. **No Users Found**: Ensure `user_cookies.txt` exists and has valid entries
2. **SSL Errors**: Set `K6_INSECURE_SKIP_TLS_VERIFY=true`
3. **ClickHouse Connection**: Verify kubectl access to ClickHouse pod
4. **Dashboard Access**: Check user permissions and cookie validity

### Logs and Debugging

- Execution logs: `logs/` directory
- Test outputs: Individual `.txt` files in result directories
- ClickHouse insertion logs: Check kubectl output

## Data Flow

1. **User Creation** → Keycloak users created, cookies harvested
2. **Test Execution** → K6 scripts run with user authentication
3. **Data Collection** → Metrics gathered from dashboard/panel responses
4. **Result Processing** → CSV generation and summaries
5. **Storage** → Data inserted into ClickHouse for analysis

This system provides comprehensive performance monitoring of Grafana dashboards, ensuring reliable access and response times for observability platforms.