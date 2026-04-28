#!/bin/bash

# ============================================================================
# Send Playwright Login Test Results to ClickHouse
# ============================================================================

# ClickHouse Configuration
CLICKHOUSE_POD="chi-clickhouse-vusmart-0-0-0"
CLICKHOUSE_NAMESPACE="vsmaps"
CLICKHOUSE_DB="vusmart"
CLICKHOUSE_USER="vusmartmanager"
CLICKHOUSE_PASSWORD="Vunet#1234"

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR"

# ============================================================================
# Functions
# ============================================================================

create_table() {
    echo "📊 Creating ClickHouse table if not exists..."
    
    kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
    clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" \
    -q "$(cat "$SCRIPT_DIR/create_table.sql")"
    
    if [ $? -eq 0 ]; then
        echo "✅ Table monitoring.playwright_login ready"
    else
        echo "❌ Failed to create table"
        exit 1
    fi
}

insert_data() {
    local json_file="$1"
    
    if [ ! -f "$json_file" ]; then
        echo "❌ Results file not found: $json_file"
        exit 1
    fi
    
    echo "📤 Processing results from: $(basename "$json_file")"
    
    # Extract test_id from filename (timestamp)
    local test_id=$(basename "$json_file" | grep -oP '\d+(?=\.json)')
    
    # Convert JSON to CSV format
    local csv_file="${json_file%.json}.csv"
    
    # Use jq to convert JSON to CSV
    jq -r --arg test_id "$test_id" '
        .details[] | 
        [
            .timestamp,
            $test_id,
            .username,
            (if .success then 1 else 0 end),
            (.totalResponseTime // .responseTime // 0),
            (.apiResponseTime // 0),
            (.uiRenderTime // 0),
            (.pageLoadTime // 0),
            .finalUrl,
            (.error // "")
        ] | @csv
    ' "$json_file" > "$csv_file"
    
    if [ ! -s "$csv_file" ]; then
        echo "❌ No data to insert (CSV file is empty)"
        rm -f "$csv_file"
        exit 1
    fi
    
    echo "🚀 Inserting data into ClickHouse..."
    
    # Insert CSV data into ClickHouse
    cat "$csv_file" | \
    kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
    clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" \
    -q "INSERT INTO monitoring.playwright_login FORMAT CSV"
    
    if [ $? -eq 0 ]; then
        local row_count=$(wc -l < "$csv_file")
        echo "✅ Successfully inserted $row_count rows into monitoring.playwright_login"
        rm -f "$csv_file"
    else
        echo "❌ Failed to insert data into ClickHouse"
        echo "📄 CSV file saved at: $csv_file (for debugging)"
        exit 1
    fi
}

verify_data() {
    echo ""
    echo "🔍 Verifying data in ClickHouse..."
    
    kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NAMESPACE" -- \
    clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" \
    -q "SELECT 
            test_id,
            count() as total_logins,
            sum(success) as successful_logins,
            round(avg(total_response_time_ms), 2) as avg_total_ms,
            round(avg(api_response_time_ms), 2) as avg_api_ms,
            round(avg(ui_render_time_ms), 2) as avg_ui_ms
        FROM monitoring.playwright_login 
        WHERE test_id = (SELECT max(test_id) FROM monitoring.playwright_login)
        GROUP BY test_id
        FORMAT PrettyCompact"
}

# ============================================================================
# Main
# ============================================================================

main() {
    echo "🎯 Playwright Login Test → ClickHouse Integration"
    echo "=" | awk '{for(i=0;i<60;i++)printf "="}; END{print ""}'
    
    # Create table if needed
    create_table
    
    # Find the latest results file
    local latest_file=$(ls -t "$RESULTS_DIR"/results/results-*users-*.json 2>/dev/null | head -1)
    
    if [ -z "$latest_file" ]; then
        echo "❌ No results files found in $RESULTS_DIR/results/"
        echo "   Run 'node playwright-login-test.js' first to generate results"
        exit 1
    fi
    
    # Insert data
    insert_data "$latest_file"
    
    # Verify
    verify_data
    
    echo ""
    echo "=" | awk '{for(i=0;i<60;i++)printf "="}; END{print ""}'
    echo "✅ ClickHouse integration complete!"
}

# Run main function
main "$@"
