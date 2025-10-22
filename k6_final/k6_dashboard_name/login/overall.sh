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
INTERVAL=$3  # No longer used, but kept for backward compatibility or optional future use

# ====== CONFIGURATION ======
SCRIPT_DIR="/home/vunet/k6_final/k6_dashboard_name/login"   # Adjust if your .js is in a different path
RESULT_DIR="./results_login_test"
CSV_FILE="${RESULT_DIR}/login_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/login_summary.txt"
LOGIN_SCRIPT="${SCRIPT_DIR}/login.js"
TEST_NAME="login_test"

mkdir -p "$RESULT_DIR"

# Initialize CSV file if not exists
if [ ! -f "$CSV_FILE" ]; then
  echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,iterations" > "$CSV_FILE"
fi

# Summary file for current run
echo -e "LOGIN PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
OUTPUT_FILE="${RESULT_DIR}/${TEST_NAME}_result_$(date +%s).txt"

echo -e "\n🔐 Running Login Test..."
echo "   Script: $(basename $LOGIN_SCRIPT)"
echo "   VUs: $VUS | Iterations: $ITERATIONS"

# Run k6 test
K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  --vus "$VUS" \
  --iterations "$ITERATIONS" \
  "$LOGIN_SCRIPT" | tee "$OUTPUT_FILE"

# Extract login metrics
AVG_TIME=$(grep "login_response_time" "$OUTPUT_FILE" | grep -oP 'avg=\K[0-9.]+')
# Extract status code from the K6 log
STATUS_CODE=$(grep -Po '\[K6-METRIC\] status_code=\K[0-9]+' "$OUTPUT_FILE" | tail -n 1)

SUCCESS_RATE=$(grep -Po 'login_success_rate.*?: \K[0-9.]+%' "$OUTPUT_FILE" | head -n 1)

echo -e "\n📊 Login Average Response Time: ${AVG_TIME}ms" | tee -a "$SUMMARY_FILE"
echo "   Status Code: $STATUS_CODE | Success Rate: $SUCCESS_RATE" | tee -a "$SUMMARY_FILE"

# Write to CSV
echo "$TIMESTAMP,$TEST_NAME,$AVG_TIME,$STATUS_CODE,$SUCCESS_RATE,$VUS,$VUS,$ITERATIONS" >> "$CSV_FILE"

echo "✅ Completed Login Test."
echo "📂 CSV Output: $CSV_FILE"

# Insert the final CSV into ClickHouse
echo -e "\n🚀 Inserting data into ClickHouse..."

tail -n +2 "$CSV_FILE" | \
sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_login FORMAT CSV"

echo "✅ Data inserted into ClickHouse table: monitoring.k6_login"

