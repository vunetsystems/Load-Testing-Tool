#!/bin/bash

# ======================================
# Usage
# ======================================
usage() {
  echo "Usage: $0 [time-range (e.g. 15m)] [vus] [duration (e.g. 1m)]"
  echo "Example: $0 15m 20 2m"
  exit 1
}

if [ $# -ne 3 ]; then
  usage
fi

TIME_RANGE=$1
VUS=$2
DURATION=$3

# ======================================
# Paths
# ======================================
BASE_DIR="/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
RESULT_DIR="./results_login_only"
mkdir -p "$RESULT_DIR"

LOGIN_CSV="${RESULT_DIR}/login_metrics.csv"

# ======================================
# Init CSV
# ======================================
echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration" > "$LOGIN_CSV"

# ======================================
# STEP 1: LOGIN TEST ONLY
# ======================================
echo -e "\n🔐 Running LOGIN TEST ONLY..."
echo "   Script: login.js"
echo "   VUs: $VUS | Duration: $DURATION"

LOGIN_OUTPUT="${RESULT_DIR}/login_${TIME_RANGE}_result.txt"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  --vus "$VUS" \
  --duration "$DURATION" \
  "$BASE_DIR/login.js" 2>&1 | tee "$LOGIN_OUTPUT"

# ======================================
# Parse login results
# ======================================
echo -e "\n📊 Parsing login results..."
TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")

grep -E "Login (successful|failed)" "$LOGIN_OUTPUT" | while IFS= read -r line; do
  if [[ "$line" =~ ✅[[:space:]]Login[[:space:]]successful[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
    RESPONSE_TIME="${BASH_REMATCH[2]}"
    STATUS=200
    SUCCESS_RATE=100
  elif [[ "$line" =~ ❌[[:space:]]Login[[:space:]]failed[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Status:[[:space:]]([0-9]+).*Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
    STATUS="${BASH_REMATCH[2]}"
    RESPONSE_TIME="${BASH_REMATCH[3]}"
    SUCCESS_RATE=0
  else
    continue
  fi

  echo "$TIMESTAMP,login,$RESPONSE_TIME,$STATUS,$SUCCESS_RATE,$VUS,$VUS,$DURATION" >> "$LOGIN_CSV"
done

echo "✅ Login Test Completed"
echo "📂 CSV Output: $LOGIN_CSV"

# ======================================
# Insert LOGIN results into ClickHouse
# ======================================
echo -e "\n🚀 Inserting login data into ClickHouse..."
sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' "$LOGIN_CSV" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart \
--user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_login FORMAT CSVWithNames"

echo "✅ Login data inserted into ClickHouse."

# ======================================
# DASHBOARD TEST — DISABLED
# ======================================
# All dashboard execution, parsing, sleep, and inserts
# have been intentionally commented out
