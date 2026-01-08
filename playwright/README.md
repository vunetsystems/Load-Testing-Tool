# K6 Dashboard Performance Testing Project

## Overview

This project provides a comprehensive solution for performance testing Grafana dashboards using Playwright browser automation. It simulates concurrent user access to dashboards, measures loading times, panel rendering performance, and API response times. The system is designed for load testing monitoring dashboards in a VuNet-based infrastructure monitoring platform.

## Project Structure

```
k6_final/
├── playwright_dashboard_panel_perf/
│   ├── playwright_dashboard_perf/
│   │   ├── package.json
│   │   ├── playwright.config.js
│   │   ├── run_dashboard_tests.sh
│   │   ├── config/
│   │   │   ├── test_config.json
│   │   │   ├── users.json
│   │   │   └── cookies.json
│   │   ├── tests/
│   │   │   └── dashboard_perf.spec.js
│   │   ├── utils/
│   │   │   ├── convert_json_to_csv.js
│   │   │   ├── cookie_loader.js
│   │   │   ├── global_setup.js
│   │   │   ├── global_teardown.js
│   │   │   ├── insert_to_clickhouse.sh
│   │   │   ├── metrics_writer.js
│   │   │   └── panel_mapper.js
│   │   ├── results/
│   │   │   ├── *.json (performance results)
│   │   │   ├── *.csv (performance results)
│   │   │   └── temp_partials/ (intermediate results)
│   │   └── test-results/
│   └── playwright_dashboard_panel_perf.zip
└── user_creation_k6/
    ├── rakshith_creation.py
    ├── user_cookies.py
    ├── user_cookies.txt
    └── timeout.txt
```

## Components

### 1. Playwright Dashboard Performance Testing

**Purpose**: Automated performance testing of Grafana dashboards using browser automation.

**Key Features**:
- Concurrent user simulation (configurable number of attempts)
- Automatic login and authentication
- Dashboard navigation and time range setting
- Panel loading detection and performance measurement
- API request monitoring and timing
- Results aggregation and reporting

### 2. User Creation and Authentication

**Purpose**: Automated creation of test users in Keycloak for load testing scenarios.

**Key Features**:
- Keycloak user provisioning
- Group assignment
- Cookie extraction for authenticated sessions
- Batch user creation

## Prerequisites

### System Requirements
- Node.js (v14+)
- npm
- Python 3.x
- jq (optional, for JSON manipulation)
- Playwright browsers
- Access to VuNet/Grafana instance
- Keycloak instance (for user creation)

### Dependencies
```json
{
  "@playwright/test": "^1.57.0",
  "playwright": "^1.57.0",
  "rimraf": "^6.1.2"
}
```

## Configuration

### Test Configuration (`config/test_config.json`)

```json
{
  "base_urls": {
    "dashboard_api": "https://your-instance/vui/api/dashboards/uid/",
    "dashboard_ui": "https://your-instance/vui/d/"
  },
  "timeRange": "Last 24 hours",
  "dashboards": [
    {
      "id": "dashboard-uuid",
      "name": "Dashboard Name",
      "slug": "dashboard-slug",
      "enabled": true
    }
  ],
  "concurrent_attempts": 10
}
```

### User Configuration (`config/users.json`)

Contains an array of user objects with authentication details:
```json
[
  {
    "username": "load_user_XXXXX",
    "password": "Password123!",
    "cookies": [
      {
        "name": "vunet_session",
        "value": "session_value",
        "domain": "your-instance",
        "path": "/",
        "httpOnly": true,
        "secure": true
      }
    ]
  }
]
```

## Usage

### Running Dashboard Performance Tests

#### Manual Test Execution
```bash
cd k6_final/playwright_dashboard_panel_perf/playwright_dashboard_perf
npx playwright test tests/dashboard_perf.spec.js
```

#### Automated Continuous Testing
```bash
cd k6_final/playwright_dashboard_panel_perf/playwright_dashboard_perf
./run_dashboard_tests.sh
```

The automation script:
- Runs tests in a continuous loop
- Alternates between configured dashboards
- Includes wait intervals between test runs
- Logs all activities and statistics
- Can be stopped with Ctrl+C

### Creating Test Users

```bash
cd k6_final/user_creation_k6
python3 rakshith_creation.py <number_of_users>
```

This script:
- Creates users in Keycloak
- Assigns them to the load test group
- Logs in to extract authentication cookies
- Saves cookies for use in performance tests

## Test Execution Flow

1. **Initialization**: Load configuration and user data
2. **Browser Setup**: Launch headless Chromium instances
3. **Authentication**: Use pre-configured cookies or perform login
4. **Dashboard Navigation**: Navigate to target dashboard URL
5. **Time Range Setting**: Configure dashboard time range
6. **Panel Loading**: Scroll through dashboard to trigger all panels
7. **Metrics Collection**: Monitor network requests and response times
8. **Results Processing**: Aggregate and format performance data
9. **Data Export**: Save JSON and CSV results
10. **Database Insertion**: Insert results into ClickHouse

## Performance Metrics Collected

### Dashboard Level Metrics
- Total dashboard load time
- Dashboard status (SUCCESS/FAILED/ERROR)
- Number of concurrent attempts
- Test execution timestamp

### Panel Level Metrics
- Panel ID and title
- Request count per panel
- Individual request timings
- Response status codes
- Start/end times for each request

### Request Level Metrics
- Request ID
- Start time
- End time
- Duration (milliseconds)
- HTTP status code

## Output Formats

### JSON Results
Structured performance data with full details:
```json
{
  "attempt": 1,
  "user": "load_user_12345",
  "totalDashboardLoadTimeMs": 45231,
  "dashboardName": "Azure Storage Blob Overview",
  "dashboardId": "dashboard-uuid",
  "dashboardStatus": "SUCCESS",
  "panels": [
    {
      "panelId": "26",
      "title": "Groups",
      "requestCount": 2,
      "requests": [
        {
          "requestId": "1-1",
          "startTime": 1767803685139,
          "endTime": 1767803685794,
          "durationMs": 655,
          "status": 200
        }
      ]
    }
  ],
  "testName": "Grafana Performance - Dashboard Name - Last 24 hours - 10 - timestamp"
}
```

### CSV Results
Tabular format for analysis:
```csv
panel_id,panel_title,requestCount,durationMs,status,startTime,endTime
26,"Groups",2,655,200,1767803685139,1767803685794
```

## Data Storage

Results are automatically inserted into ClickHouse database using the `insert_to_clickhouse.sh` script. The data can then be visualized and analyzed for:

- Performance trends over time
- Dashboard comparison metrics
- Load testing insights
- Panel-specific performance analysis

## Automation Features

### Continuous Testing Loop
- Configurable wait intervals between runs
- Automatic dashboard switching
- Statistics tracking (success/failure rates)
- Graceful shutdown handling

### Error Handling
- Partial result saving on failures
- Cookie refresh mechanisms
- Network timeout handling
- Browser crash recovery

### Logging
- Comprehensive activity logging
- Performance statistics
- Error reporting
- Progress indicators

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify cookie validity
   - Check Keycloak configuration
   - Ensure user accounts are active

2. **Dashboard Loading Issues**
   - Confirm dashboard URLs are correct
   - Check network connectivity
   - Verify Grafana permissions

3. **Panel Detection Problems**
   - Adjust scroll timing in test script
   - Verify panel IDs in configuration
   - Check dashboard structure changes

4. **Database Connection Issues**
   - Verify ClickHouse credentials
   - Check network access to database
   - Validate table schema

### Debug Mode
Run tests with verbose logging:
```bash
DEBUG=pw:api npx playwright test tests/dashboard_perf.spec.js
```

## Contributing

1. Update configuration files for new dashboards
2. Modify user creation parameters as needed
3. Adjust timing parameters for different environments
4. Add new metrics collection points
5. Enhance error handling and recovery

## End Results

The project produces comprehensive performance testing data that enables:

- **Performance Monitoring**: Track dashboard loading times under load
- **Scalability Testing**: Measure system performance with concurrent users
- **Regression Detection**: Identify performance degradation over time
- **Optimization Insights**: Pinpoint slow-loading panels and components
- **Capacity Planning**: Understand system limits and bottlenecks

The automated nature allows for continuous performance monitoring, ensuring dashboard performance remains optimal as the system evolves and user load increases.