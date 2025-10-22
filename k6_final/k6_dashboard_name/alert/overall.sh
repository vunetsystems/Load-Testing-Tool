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
SCRIPT_DIR="/home/vunet/k6_final/k6_dashboard_name/alert"  # <-- your alert scripts directory
RESULT_DIR="./results_alerts"
CSV_FILE="${RESULT_DIR}/alerts_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/alerts_summary.txt"

mkdir -p "$RESULT_DIR"

# Initialize output files
echo "timestamp,alert_name,execution_status,alert_execution_response_time_ms,vus,iterations" > "$CSV_FILE"
echo -e "ALERT EXECUTION SUMMARY\n" > "$SUMMARY_FILE"

# Process each alert script
for ALERT_SCRIPT in "$SCRIPT_DIR"/*.js; do
  ALERT_NAME=$(basename "$ALERT_SCRIPT" .js)
  TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
  
  echo -e "\n🔹 Executing Alert: $ALERT_NAME"
  echo "   Script: $(basename $ALERT_SCRIPT)"
  echo "   VUs: $VUS | Iterations: $ITERATIONS"

  OUTPUT_FILE="${RESULT_DIR}/${ALERT_NAME}_result.txt"

  # Measure start time in milliseconds
  START_TIME_MS=$(date +%s%3N)

  # Run k6 test
  K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
    --vus "$VUS" \
    --iterations "$ITERATIONS" \
    "$ALERT_SCRIPT" | tee "$OUTPUT_FILE"

  # Measure end time in milliseconds
  END_TIME_MS=$(date +%s%3N)

  # Calculate elapsed time
  ALERT_EXECUTION_RESPONSE_TIME=$((END_TIME_MS - START_TIME_MS))

  # Determine execution status
  if grep -q "100.00% ✓" "$OUTPUT_FILE"; then
    EXECUTION_STATUS="success"
  else
    EXECUTION_STATUS="failure"
  fi

  # Write to CSV
  echo "$TIMESTAMP,$ALERT_NAME,$EXECUTION_STATUS,$ALERT_EXECUTION_RESPONSE_TIME,$VUS,$ITERATIONS" >> "$CSV_FILE"

  # Update Summary
  echo "  - $ALERT_NAME : $EXECUTION_STATUS (${ALERT_EXECUTION_RESPONSE_TIME}ms)" | tee -a "$SUMMARY_FILE"

  echo "✅ Completed Alert: $ALERT_NAME"
  echo "⏳ Waiting $INTERVAL seconds before next alert..."
  sleep "$INTERVAL"
done

echo -e "\n📂 Results saved to:"
echo "  - Alerts CSV: $CSV_FILE"
echo "  - Summary: $SUMMARY_FILE"

# Insert the final CSV into ClickHouse
echo -e "\n🚀 Inserting data into ClickHouse..."

kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_alerts FORMAT CSVWithNames" < "$CSV_FILE"

echo "✅ Data inserted into ClickHouse table: monitoring.alerts_results"

