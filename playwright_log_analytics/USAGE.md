# Playwright Log Analytics Test - Usage Guide

## Table of Contents
1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Running Tests](#running-tests)
4. [Understanding Results](#understanding-results)
5. [Troubleshooting](#troubleshooting)

## Installation

### Prerequisites
- Node.js 16 or higher
- npm or yarn
- kubectl access to ClickHouse
- Go 1.21+ (for report generation)

### Setup Steps

```bash
# Navigate to the project directory
cd /home/vunet/Load-Testing-Tool/playwright_log_analytics

# Install Node.js dependencies
make install

# Create ClickHouse table (first time only)
make create-table
```

## Configuration

### 1. User Credentials (`users.txt`)

Add user credentials in CSV format:
```
username,password
vunetadmin,Qwerty@123
testuser1,Password123
```

**Format Rules:**
- One user per line
- Comma-separated username and password
- Lines starting with `#` are treated as comments
- Empty lines are ignored

### 2. Search Filters (`filters.txt`)

Add search filters to test, one per line:
```
# Simple keyword searches
500
200
error

# Field-specific searches
httpCode:404
target:10.10.10.206
apache2_access_method:GET
apache2_access_user_agent_name:curl
vublock_name:apache

# Complex queries
"GET+apache2_access_url:/api/v1/users"
```

**Filter Types:**
- **Simple keywords**: `500`, `error`, `warning`
- **Field:value pairs**: `httpCode:404`, `target:10.10.10.206`
- **Complex queries**: Use quotes for multi-word searches

### 3. Environment Variables

Customize test behavior with environment variables:

```bash
# Base URL
export BASE_URL="https://216.48.191.10"

# Log source to test
export LOG_SOURCE="Apachelogs"

# Time range for queries
export TIME_RANGE="Last 12 hours"
# Options: "Last 15 minutes", "Last 1 hour", "Last 12 hours", "Last 24 hours"

# Browser mode
export HEADLESS=true  # false to see browser

# Timeout (milliseconds)
export TIMEOUT=60000

# Auto-send to ClickHouse
export SEND_TO_CLICKHOUSE=true
```

## Running Tests

### Basic Test Execution

```bash
# Run test (headless mode)
make test

# Run test with visible browser
make test-visible

# Run test and send to ClickHouse
make test-send
```

### Advanced Usage

```bash
# Custom configuration
BASE_URL="https://custom.url" LOG_SOURCE="Systemlogs" make test

# Test specific time range
TIME_RANGE="Last 24 hours" make test

# Run with custom timeout
TIMEOUT=120000 make test

# Combine multiple options
HEADLESS=false TIME_RANGE="Last 1 hour" make test
```

### Manual Execution

```bash
# Run the test script directly
node playwright-log-analytics-test.js

# With environment variables
HEADLESS=false SEND_TO_CLICKHOUSE=true node playwright-log-analytics-test.js
```

## Understanding Results

### Console Output

During test execution, you'll see:
```
🚀 Playwright Log Analytics Load Test
======================================================================
📍 Base URL:      https://216.48.191.10
📁 User File:     /path/to/users.txt
📁 Filter File:   /path/to/filters.txt
🗂️  Log Source:    Apachelogs
⏰ Time Range:    Last 12 hours
🖥️  Headless:      true
======================================================================

[User 1/1] 🔹 Testing as: vunetadmin
[User 1/1] 📊 Filters to test: 8
[User 1/1] 🔑 Logging in...
[User 1/1] ✅ Login successful (3245ms)
[User 1/1] 🗂️  Navigating to Log Analytics...
[User 1/1] ✅ Navigation complete (4521ms)
[User 1/1] 🔍 Testing filter 1/8: "500"
[User 1/1] ✅ Filter "500" | Total: 5234ms | API: 3421ms | Render: 1813ms
...
```

### CSV Results

Results are saved in `results/log_analytics_results_<timestamp>.csv`:

```csv
timestamp,username,filter,success,total_time_ms,page_load_time_ms,search_api_time_ms,results_render_time_ms,test_id
2026-01-16T09:00:00Z,vunetadmin,"500",1,5234,4521,3421,1813,1737019200000_8filters
```

**Columns:**
- `timestamp` - When the test was executed
- `username` - User who performed the search
- `filter` - Search filter applied
- `success` - 1 for success, 0 for failure
- `total_time_ms` - Total search execution time
- `page_load_time_ms` - Initial page load time
- `search_api_time_ms` - Backend API response time
- `results_render_time_ms` - UI rendering time
- `test_id` - Unique test identifier

### HTML Reports

Generate reports with:
```bash
make report
```

Reports are saved in `reports/playwright_log_analytics_report_<test_id>.html`

**Report Sections:**
1. **Executive Summary** - Key metrics and success rate
2. **Timing Breakdown** - Visual comparison of API vs rendering time
3. **Filter Performance** - Per-filter statistics
4. **Detailed Results** - Complete test result table

### ClickHouse Data

Query results directly:
```bash
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "SELECT * FROM monitoring.playwright_log_analytics ORDER BY timestamp DESC LIMIT 10"
```

## Troubleshooting

### Common Issues

#### 1. "User file not found"
**Solution:** Ensure `users.txt` exists and contains valid credentials
```bash
cat users.txt
# Should show: username,password
```

#### 2. "Filter file not found"
**Solution:** Create `filters.txt` with at least one filter
```bash
echo "500" > filters.txt
```

#### 3. Login timeout
**Solution:** Increase timeout or check credentials
```bash
TIMEOUT=120000 make test
```

#### 4. ClickHouse connection failed
**Solution:** Verify kubectl access
```bash
kubectl get pods -n vsmaps | grep clickhouse
```

#### 5. Selector not found
**Solution:** UI may have changed. Run with visible browser to debug:
```bash
HEADLESS=false make test
```

### Debug Mode

Run with visible browser to see what's happening:
```bash
HEADLESS=false node playwright-log-analytics-test.js
```

### Performance Tips

1. **Reduce filters** - Test fewer filters for faster execution
2. **Increase timeout** - For slow networks: `TIMEOUT=120000`
3. **Sequential testing** - Tests run sequentially to avoid overwhelming the system
4. **Headless mode** - Always use headless mode for production tests

### Getting Help

Check logs in:
- Console output during test execution
- CSV files in `results/` directory
- ClickHouse table: `monitoring.playwright_log_analytics`

## Best Practices

1. **Start small** - Test with 1-2 filters first
2. **Verify credentials** - Ensure users have log analytics access
3. **Monitor resources** - Watch system resources during tests
4. **Regular cleanup** - Clean old results: `make clean`
5. **Backup data** - Export ClickHouse data periodically

## Example Workflows

### Daily Performance Testing
```bash
#!/bin/bash
cd /home/vunet/Load-Testing-Tool/playwright_log_analytics
make test-send
make report
# Email or upload report
```

### Compare Different Time Ranges
```bash
TIME_RANGE="Last 1 hour" make test-send
TIME_RANGE="Last 12 hours" make test-send
TIME_RANGE="Last 24 hours" make test-send
make report
```

### Test Multiple Log Sources
```bash
LOG_SOURCE="Apachelogs" make test-send
LOG_SOURCE="Systemlogs" make test-send
make report
```
