# vuDataSim Load Testing Tool Documentation

## What is vuDataSim?

vuDataSim is a comprehensive load testing and data simulation platform designed for testing observability systems and infrastructure. It combines infrastructure-level data simulation with UI load testing to provide end-to-end performance validation of monitoring and observability platforms.

### Key Features

- **Data Simulation**: Generates realistic metrics data for various observability sources (Apache, MSSQL, Linux, Kubernetes, etc.)
- **Distributed Architecture**: Manages multiple nodes via SSH for scalable testing
- **Web UI**: Modern web interface for configuration and monitoring
- **K6 Integration**: Built-in load testing for UI performance validation
- **Real-time Monitoring**: Live metrics collection and visualization
- **Automated Reporting**: Combined infrastructure and UI performance reports
- **Test Run Tracking**: SQLite-based test management with completion checking

## How to Use

### Prerequisites

- Go 1.24+
- Python 3.x (for report generation)
- SSH access to target nodes
- Kafka and ClickHouse clusters (for data ingestion)

### Installation

1. Clone the repository
2. Build the Go binaries:
   ```bash
   go build -o vudatasim-manager src/main.go
   go build -o vudatasim-test src/main.go  # if needed
   ```

3. Configure environment variables in `.env`:
   ```
   APP_PORT=8086
   STATIC_DIR=./static
   SESSION_SECRET=your-secret
   OAUTH_CLIENT_ID=your-oauth-id
   OAUTH_CLIENT_SECRET=your-oauth-secret
   ```

### Starting the Application

```bash
./vudatasim-manager
```

Access the web UI at `http://localhost:8086`

### Basic Usage Workflow

1. **Node Management**: Add and configure target nodes via SSH
2. **Binary Control**: Deploy and manage vuDataSim binaries on nodes
3. **Configuration Sync**: Distribute observability source configurations
4. **Start Simulation**: Configure EPS targets and run data simulation
5. **K6 Testing**: Run UI load tests against dashboards
6. **Generate Reports**: Create combined performance reports

## File Structure

### Backend (src/)

The Go backend provides REST APIs and manages the distributed simulation:

```
src/
├── main.go                 # Main application entry point with HTTP server
├── auth.go                 # OAuth authentication handling
├── websocket.go            # Real-time WebSocket communication
├── middleware.go           # HTTP middleware (CORS, logging, auth)
├── handlers/               # API endpoint handlers
├── bin_control/            # Binary deployment and management
├── clickhouse/             # ClickHouse database operations
├── kafka_processors/       # Kafka message processing
├── node_control/           # SSH-based node management
├── pod_processors/         # Kubernetes pod monitoring
├── models/                 # Data models for test runs
├── logger/                 # Structured logging
└── migrate/                # Configuration migration utilities
```

### Frontend (static/)

Modern web UI built with vanilla JavaScript and Tailwind CSS:

```
static/
├── index.html              # Main dashboard page
├── main.js                 # Core application manager and modules
├── dashboard.js            # Dashboard-specific functionality
├── node-management.js      # Node configuration UI
├── binary-control.js       # Binary management interface
├── clickhouse-metrics.js   # Metrics visualization
├── o11y-sources.js         # Observability source selection
├── logs.js                 # Real-time log streaming
├── monitoring-js/          # Monitoring dashboard components
└── styles.css              # Custom styling
```

### Data Processing (data/)

Python scripts for report generation and data analysis:

```
data/
├── generate_combined_report.py    # Combined infra + UI reports
├── generate_html_reports.py       # Infrastructure performance reports
├── generate_reports.py            # Report generation utilities
├── k6.py                          # K6 test result processing
├── vudatasim.db                   # SQLite database for test tracking
├── combined_reports/              # Generated HTML reports
└── k6_html_reports/               # K6-specific reports
```

### K6 Load Testing (k6_final/)

K6 scripts for UI performance testing:

```
k6_final/
├── k6_dashboard_name/
│   ├── *.js                    # K6 test scripts
│   ├── k6_config.yaml          # K6 configuration
│   └── alert/, ibmb/, etc.     # Test-specific directories
```

## UI Components

### Main Dashboard

The main interface (`static/index.html`) provides:

- **Simulation Configuration**: Select observability sources, set EPS targets, configure timeouts
- **K6 Load Testing**: Configure and run UI performance tests
- **Configuration Sync Status**: Visual progress tracking of setup steps
- **Live Node Metrics**: Real-time CPU, memory, and process monitoring
- **Logs Preview**: Streaming logs from all nodes

### Key UI Modules

#### VuDataSimManager (main.js)
Central orchestrator managing all UI components:
- WebSocket connections for real-time updates
- API communication
- Module initialization and coordination

#### Node Management
- Add/remove SSH-configured nodes
- Bulk operations for cluster setup
- Node status monitoring

#### Binary Control
- Deploy binaries to nodes via SSH
- Start/stop processes remotely
- Bulk operations across all nodes

#### Observability Sources
- Multi-select interface for choosing data sources
- Category-based filtering (Azure, Web/DB, System/DB)
- Dynamic configuration distribution

#### Real-time Monitoring
- Live metrics collection every 3 seconds
- SSH-based system monitoring
- ClickHouse metrics aggregation

## Reporting and Expected Outcomes

### Report Types

1. **Infrastructure Reports**: Generated from SQLite database showing:
   - Throughput rates (input/output/process)
   - Resource utilization (CPU, memory per component)
   - Lag metrics for data processing
   - Ingestion rates per table

2. **K6 UI Reports**: Performance metrics for dashboard loading:
   - Login performance
   - Dashboard load times
   - Panel rendering breakdown
   - Overall UI responsiveness

3. **Combined Reports**: Toggle-able HTML reports combining both infra and UI metrics

### Expected Outcomes

After running a complete test:

- **Data Volume**: Configurable EPS (events per second) from 5K to 200K+
- **System Load**: Distributed across Kafka, ClickHouse, and application nodes
- **Performance Metrics**: End-to-end latency, throughput, and resource usage
- **UI Validation**: Dashboard performance under load
- **Comprehensive Reports**: HTML files with charts and detailed breakdowns

### Report Generation

Reports are automatically generated by running:

```bash
cd data
python generate_combined_report.py
```

This matches vuDataSim and K6 test runs by normalized names and creates combined HTML reports in `data/combined_reports/`.

## Architecture Overview

### Backend Architecture

- **HTTP Server**: Gorilla Mux-based REST API
- **Authentication**: OAuth integration with session management
- **Database**: SQLite for test run tracking
- **Real-time**: WebSocket for live updates
- **SSH Operations**: Remote node management and monitoring
- **Data Pipeline**: Kafka → ClickHouse for metrics storage

### Data Flow

1. **Configuration**: UI → API → SSH → Nodes
2. **Simulation**: Binaries generate metrics → Kafka → ClickHouse
3. **Monitoring**: SSH polling + ClickHouse queries → WebSocket → UI
4. **Reporting**: SQLite queries → Python processing → HTML reports

### Supported Observability Sources

- Apache HTTP Server
- AWS ALB, Azure Firewall, Azure API Management
- Cisco IOS Switch
- DNS Monitoring
- IBM MQ
- Kubernetes
- Linux System Monitoring
- MongoDB
- MSSQL Server
- Tomcat
- And more...

This platform enables comprehensive testing of observability infrastructure by simulating realistic data loads and measuring both backend performance and frontend user experience.