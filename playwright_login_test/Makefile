# Makefile for Playwright Login Test Suite
# ==========================================

.PHONY: help install build test test-with-reports report clean

# Default target
help:
	@echo "Playwright Login Test - Available Commands:"
	@echo "=============================================="
	@echo ""
	@echo "  make install           - Install Node.js dependencies"
	@echo "  make build             - Build Go report generator"
	@echo "  make test              - Run Playwright test (no ClickHouse)"
	@echo "  make test-with-reports - Run test + ClickHouse + HTML reports"
	@echo "  make report            - Generate HTML reports from existing data"
	@echo "  make clean             - Remove generated files"
	@echo ""
	@echo "Examples:"
	@echo "  make test-with-reports  # Full automated test with reports"
	@echo "  make report             # Just regenerate reports"
	@echo ""

# Install Node.js dependencies
install:
	@echo "📦 Installing Node.js dependencies..."
	npm install
	@echo "✅ Dependencies installed!"

# Build Go report generator
build:
	@echo "🔨 Building Go report generator..."
	go build -o generate-report generate_report.go
	@echo "✅ Report generator built!"

# Run Playwright test without ClickHouse
test:
	@echo "🚀 Running Playwright login test..."
	node playwright-login-test.js

# Run Playwright test with ClickHouse and automatic reports
test-with-reports:
	@echo "🚀 Running Playwright login test with ClickHouse and reports..."
	SEND_TO_CLICKHOUSE=true node playwright-login-test.js

# Generate HTML reports from ClickHouse data
report:
	@echo "📊 Generating HTML reports..."
	./generate-report

# Clean generated files
clean:
	@echo "🧹 Cleaning generated files..."
	rm -rf results/*.json
	rm -rf reports/*.html
	@echo "✅ Cleaned!"

# Advanced targets
.PHONY: test-headful test-10 test-30 truncate-db

# Run test with visible browser
test-headful:
	@echo "🚀 Running Playwright test (visible browser)..."
	HEADLESS=false SEND_TO_CLICKHOUSE=true node playwright-login-test.js

# Quick test with 10 users
test-10:
	@echo "📝 Creating users.txt with 10 users..."
	@head -10 users.txt > users_temp.txt && mv users_temp.txt users.txt
	@echo "🚀 Running test with 10 users..."
	SEND_TO_CLICKHOUSE=true node playwright-login-test.js

# Full test with 30 users
test-30:
	@echo "📝 Creating users.txt with 30 users..."
	@head -30 users.txt > users_temp.txt && mv users_temp.txt users.txt
	@echo "🚀 Running test with 30 users..."
	SEND_TO_CLICKHOUSE=true node playwright-login-test.js

# Truncate ClickHouse table
truncate-db:
	@echo "⚠️  Truncating ClickHouse table..."
	kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
		clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
		-q "TRUNCATE TABLE monitoring.playwright_login"
	@echo "✅ Table truncated!"

# Query ClickHouse data
.PHONY: query-db query-summary

query-db:
	@echo "📊 Querying ClickHouse data..."
	kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
		clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
		-q "SELECT * FROM monitoring.playwright_login ORDER BY timestamp DESC LIMIT 20 FORMAT PrettyCompact"

query-summary:
	@echo "📊 Test Summary from ClickHouse..."
	kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
		clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
		-q "SELECT test_id, count() as users, sum(success) as successful, round(avg(total_response_time_ms), 2) as avg_total_ms, round(avg(api_response_time_ms), 2) as avg_api_ms, round(avg(ui_render_time_ms), 2) as avg_ui_ms FROM monitoring.playwright_login GROUP BY test_id ORDER BY test_id DESC FORMAT PrettyCompact"
