.PHONY: help install test test-send create-table report clean

help:
	@echo "Playwright Log Analytics Test - Available Commands"
	@echo "=================================================="
	@echo "make install       - Install Node.js dependencies"
	@echo "make test          - Run test with login (slower, full journey)"
	@echo "make test-cookies  - Run test with cookies (faster, like K6)"
	@echo "make test-send     - Run test and send results to ClickHouse"
	@echo "make create-table  - Create ClickHouse table"
	@echo "make report        - Generate HTML report from ClickHouse data"
	@echo "make build-report  - Build Go report generator"
	@echo "make clean         - Clean results and reports"
	@echo ""

install:
	@echo "📦 Installing dependencies..."
	npm install
	@echo "✅ Dependencies installed"

test:
	@echo "🚀 Running Playwright log analytics test (with login)..."
	node playwright-log-analytics-test.js

test-cookies:
	@echo "🚀 Running Playwright log analytics test (with cookies - faster!)..."
	node playwright-log-analytics-test-cookies.js

test-cookies-send:
	@echo "🚀 Running cookie-based test with ClickHouse integration..."
	SEND_TO_CLICKHOUSE=true node playwright-log-analytics-test-cookies.js

test-send:
	@echo "🚀 Running test with ClickHouse integration..."
	SEND_TO_CLICKHOUSE=true node playwright-log-analytics-test.js

create-table:
	@echo "📊 Creating ClickHouse table..."
	kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
		clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
		< create_table.sql
	@echo "✅ Table created successfully"

build-report:
	@echo "🔨 Building Go report generator..."
	go build -o generate-report generate_report.go
	chmod +x generate-report
	@echo "✅ Report generator built"

report: build-report
	@echo "📊 Generating HTML report..."
	./generate-report

clean:
	@echo "🧹 Cleaning results and reports..."
	rm -rf results/*.csv
	rm -rf reports/*.html
	@echo "✅ Cleaned"

# Quick test commands
test-headless:
	HEADLESS=true node playwright-log-analytics-test.js

test-visible:
	HEADLESS=false node playwright-log-analytics-test.js

# Custom filter test
test-filter:
	@echo "Testing with custom filter..."
	@read -p "Enter filter: " filter; \
	echo "$$filter" > filters.txt; \
	node playwright-log-analytics-test.js
