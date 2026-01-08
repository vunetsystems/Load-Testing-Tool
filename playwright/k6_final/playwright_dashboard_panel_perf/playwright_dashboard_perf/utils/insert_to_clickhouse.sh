#!/bin/bash

# Script to insert CSV data into ClickHouse dashboard_perf table
# Usage: ./insert_to_clickhouse.sh <csv_file>

CSV_FILE="$1"

if [ -z "$CSV_FILE" ]; then
    echo "Usage: $0 <csv_file>"
    echo "Example: $0 ../results/concurrent_dashboard_run_2025-12-29T18-28-15-705Z.csv"
    exit 1
fi

if [ ! -f "$CSV_FILE" ]; then
    echo "Error: CSV file '$CSV_FILE' not found"
    exit 1
fi

echo "Inserting data from $CSV_FILE into ClickHouse table dashboard_perf..."

kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d monitoring --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO dashboard_perf FORMAT CSVWithNames" < "$CSV_FILE"

if [ $? -eq 0 ]; then
    echo "Data insertion completed successfully."
else
    echo "Error: Data insertion failed."
    exit 1
fi