# Playwright Login Test

A unified Playwright script to test login with any number of concurrent users (1, 10, 100+).

## 🚀 Quick Start

```bash
# Edit users.txt with your credentials
node playwright-login-test.js
```

## 📁 Files

- **`playwright-login-test.js`** - Main test script (works for any number of users)
- **`users.txt`** - User credentials file (edit this to add/remove users)
- **`USAGE.md`** - Complete usage guide with examples
- **`package.json`** - Node.js dependencies
- **`results-*.json`** - Test results (auto-generated after each run)

## 📝 User File Format

Edit `users.txt`:
```
username,password
username2,password2
username3,password3
```

## 🚀 Quick Start (Using Makefile)

```bash
# Run full test with automatic reports
make test-with-reports

# Just generate reports from existing data
make report

# See all available commands
make help
```

## ⚙️ Advanced Usage

```bash
# Use custom credentials file
USER_FILE=my-users.txt node playwright-login-test.js

# See browser in action
HEADLESS=false node playwright-login-test.js

# Increase timeout
TIMEOUT=60000 node playwright-login-test.js

# Auto-send results to ClickHouse
SEND_TO_CLICKHOUSE=true node playwright-login-test.js
```

## 📊 ClickHouse Integration

Results can be automatically stored in ClickHouse for analysis and reporting.

**Auto-send after test:**
```bash
SEND_TO_CLICKHOUSE=true node playwright-login-test.js
# OR
make test-with-reports
```

**Manual send:**
```bash
bash send-to-clickhouse.sh
```

**Query results:**
```bash
make query-summary
```

See `USAGE.md` for complete documentation.
