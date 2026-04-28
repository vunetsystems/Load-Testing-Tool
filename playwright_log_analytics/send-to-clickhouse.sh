#!/bin/bash

# ============================================================================
# Send Playwright Log Analytics Test Results to ClickHouse
# ============================================================================

set -e

# Configuration
CLICKHOUSE_POD="chi-clickhouse-vusmart-0-0-0"
CLICKHOUSE_NAMESPACE="vsmaps"
CLICKHOUSE_USER="vusmartmanager"
CLICKHOUSE_PASSWORD="Vunet#1234"
CLICKHOUSE_DB="vusmart"
CLICKHOUSE_TABLE="monitoring.playwright_log_analytics"

# Find the most recent CSV file
RESULTS_DIR="./results"
CSV_FILE=$(ls -t "$RESULTS_DIR"/log_analytics_results_*.csv 2>/dev/null | head -1)

if [ -z "$CSV_FILE" ]; then
    echo "❌ No CSV results file found in $RESULTS_DIR"
    echo "   Please run the test first: npm test"
    exit 1
fi

echo "📁 Found results file: $CSV_FILE"
echo "🔍 Checking file contents..."

# Count lines (excluding header)
LINE_COUNT=$(tail -n +2 "$CSV_FILE" | wc -l)
echo "📊 Found $LINE_COUNT test result(s) to insert"

if [ "$LINE_COUNT" -eq 0 ]; then
    echo "⚠️  No data to insert (file is empty or only contains headers)"
    exit 0
fi

echo ""
echo "🧹 Cleaning data (handling NaN, N/A, empty values)..."

# Create a temporary cleaned file
CLEANED_FILE=$(mktemp)

# Clean the CSV data:
# - Replace NaN with 0
# - Replace N/A with 0
# - Replace empty fields with 0
sed -E \
    -e 's/,NaN,/,0,/g' \
    -e 's/,nan,/,0,/g' \
    -e 's/,N\/A,/,0,/g' \
    -e 's/,,/,0,/g' \
    -e 's/,$/,0/' \
    "$CSV_FILE" > "$CLEANED_FILE"

echo "✅ Data cleaned successfully"
echo ""
echo "🚀 Inserting data into ClickHouse..."
echo "   Pod: $CLICKHOUSE_POD"
echo "   Namespace: $CLICKHOUSE_NAMESPACE"
echo "   Table: $CLICKHOUSE_TABLE"
echo ""

# Insert data into ClickHouse
cat "$CLEANED_FILE" | \
kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
clickhouse-client \
    -d "$CLICKHOUSE_DB" \
    --user "$CLICKHOUSE_USER" \
    --password "$CLICKHOUSE_PASSWORD" \
    -q "INSERT INTO $CLICKHOUSE_TABLE FORMAT CSVWithNames"

if [ $? -eq 0 ]; then
    echo "✅ Successfully inserted $LINE_COUNT row(s) into ClickHouse"
    echo ""
    echo "📊 Verifying insertion..."
    
    # Verify the insertion
    kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
    clickhouse-client \
        -d "$CLICKHOUSE_DB" \
        --user "$CLICKHOUSE_USER" \
        --password "$CLICKHOUSE_PASSWORD" \
        -q "SELECT COUNT(*) as total_rows FROM $CLICKHOUSE_TABLE"
    
    echo ""
    echo "🔍 Latest 5 entries:"
    kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
    clickhouse-client \
        -d "$CLICKHOUSE_DB" \
        --user "$CLICKHOUSE_USER" \
        --password "$CLICKHOUSE_PASSWORD" \
        -q "SELECT timestamp, username, filter, success, total_time_ms FROM $CLICKHOUSE_TABLE ORDER BY timestamp DESC LIMIT 5" \
        --format PrettyCompact
else
    echo "❌ Failed to insert data into ClickHouse"
    rm -f "$CLEANED_FILE"
    exit 1
fi

# Cleanup
rm -f "$CLEANED_FILE"

echo ""
echo "✅ ClickHouse insertion complete!"
