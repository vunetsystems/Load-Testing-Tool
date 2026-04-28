# Playwright Log Analytics Load Testing

Automated performance testing framework for Log Analytics using Playwright. This tool simulates real user interactions with the log analytics interface, applies search filters, and measures performance metrics.

## Features

- 🎭 **Real Browser Automation** - Uses Playwright to simulate actual user behavior
- 🔍 **Filter Testing** - Tests multiple search filters automatically
- ⏱️ **Performance Metrics** - Tracks search API time, results rendering time, and total time
- 📊 **ClickHouse Integration** - Stores results in ClickHouse for analysis
- 📈 **HTML Reports** - Generates beautiful performance reports with charts
- 🔄 **Batch Testing** - Test multiple filters and users in sequence

## Quick Start

```bash
# 1. Install dependencies
make install

# 2. Create ClickHouse table (first time only)
make create-table

# 3. Run the test
make test

# 4. Run test and send to ClickHouse
make test-send

# 5. Generate HTML report
make report
```

## Project Structure

```
playwright_log_analytics/
├── playwright-log-analytics-test.js  # Main test script
├── users.txt                          # User credentials
├── filters.txt                        # Search filters to test
├── package.json                       # Node.js dependencies
├── create_table.sql                   # ClickHouse table schema
├── send-to-clickhouse.sh             # Data insertion script
├── generate_report.go                 # Report generator
├── report_template.gohtml             # HTML report template
├── Makefile                           # Automation commands
├── results/                           # CSV test results
└── reports/                           # Generated HTML reports
```

## Configuration

### Users (`users.txt`)
```
username,password
user1,password1
user2,password2
```

### Filters (`filters.txt`)
```
500
httpCode:404
target:10.10.10.206
apache2_access_method:GET
```

### Environment Variables
- `BASE_URL` - Base URL (default: https://216.48.191.10)
- `LOG_SOURCE` - Log source name (default: Apachelogs)
- `TIME_RANGE` - Time range for queries (default: Last 12 hours)
- `HEADLESS` - Run browser in headless mode (default: true)
- `TIMEOUT` - Request timeout in ms (default: 60000)
- `SEND_TO_CLICKHOUSE` - Auto-send to ClickHouse (default: false)

## Usage Examples

```bash
# Run test with visible browser
make test-visible

# Run test in headless mode
make test-headless

# Custom time range
TIME_RANGE="Last 24 hours" make test

# Custom log source
LOG_SOURCE="Systemlogs" make test
```

## Metrics Collected

- **Total Time** - Complete search execution time
- **Search API Time** - Backend API response time
- **Results Render Time** - UI rendering time
- **Success Rate** - Percentage of successful searches
- **Per-Filter Stats** - Performance breakdown by filter

## Report Features

- Executive summary with key metrics
- Performance charts (Search API vs Rendering)
- Filter-by-filter breakdown
- Detailed test results table
- Bottleneck identification

## Requirements

- Node.js 16+
- Playwright
- Go 1.21+ (for report generation)
- kubectl access to ClickHouse pod
- ClickHouse database

## Author

VuNet Systems - Load Testing Team
