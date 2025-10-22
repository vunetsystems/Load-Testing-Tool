#!/bin/bash

# Usage hint
usage() {
  echo "Usage: $0 [vus] [iterations] [interval-in-seconds]"
  exit 1
}

# Check input args
if [ $# -ne 3 ]; then
  usage
fi

# ====== USER INPUTS ======
VUS=$1
ITERATIONS=$2
INTERVAL=$3

# ====== CONFIGURATION ======
SCRIPT_DIR="/home/vunet/k6_final/k6_dashboard_name/reports"  # <-- your report scripts directory
RESULT_DIR="./results_reports"
CSV_FILE="${RESULT_DIR}/reports_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/reports_summary.txt"

mkdir -p "$RESULT_DIR"

# Initialize output files
echo "timestamp,report_name,execution_status,report_execution_response_time_ms,vus,iterations" > "$CSV_FILE"
echo -e "REPORT EXECUTION SUMMARY\n" > "$SUMMARY_FILE"

# Process each report script
for REPORT_SCRIPT in "$SCRIPT_DIR"/*.js; do
  REPORT_NAME=$(basename "$REPORT_SCRIPT" .js)
  TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
  
  echo -e "\n🔹 Executing Report: $REPORT_NAME"
  echo "   Script: $(basename $REPORT_SCRIPT)"
  echo "   VUs: $VUS | Iterations: $ITERATIONS"

  OUTPUT_FILE="${RESULT_DIR}/${REPORT_NAME}_result.txt"

  # Measure start time in milliseconds
  START_TIME_MS=$(date +%s%3N)

  # Run k6 test
  K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
    --vus "$VUS" \
    --iterations "$ITERATIONS" \
    "$REPORT_SCRIPT" | tee "$OUTPUT_FILE"

  # Measure end time in milliseconds
  END_TIME_MS=$(date +%s%3N)

  # Calculate elapsed time
  REPORT_EXECUTION_RESPONSE_TIME=$((END_TIME_MS - START_TIME_MS))

  # Determine execution status
  if grep -q "100.00% ✓" "$OUTPUT_FILE"; then
    EXECUTION_STATUS="success"
  else
    EXECUTION_STATUS="failure"
  fi

  # Write to CSV
  echo "$TIMESTAMP,$REPORT_NAME,$EXECUTION_STATUS,$REPORT_EXECUTION_RESPONSE_TIME,$VUS,$ITERATIONS" >> "$CSV_FILE"

  # Update Summary
  echo "  - $REPORT_NAME : $EXECUTION_STATUS (${REPORT_EXECUTION_RESPONSE_TIME}ms)" | tee -a "$SUMMARY_FILE"

  echo "✅ Completed Report: $REPORT_NAME"
  echo "⏳ Waiting $INTERVAL seconds before next report..."
  sleep "$INTERVAL"
done

echo -e "\n📂 Results saved to:"
echo "  - Reports CSV: $CSV_FILE"
echo "  - Summary: $SUMMARY_FILE"

# Insert the final CSV into ClickHouse
echo -e "\n🚀 Inserting data into ClickHouse..."

kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_reports FORMAT CSVWithNames" < "$CSV_FILE"

echo "✅ Data inserted into ClickHouse table: monitoring.reports_results"

