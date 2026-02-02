#!/bin/bash

# ======================================
# Usage
# ======================================
usage() {
  echo "Usage: $0 [time-range (e.g. 15m)] [vus] [duration (e.g. 1m)] [interval-in-seconds]"
  echo "Example: $0 15m 20 2m 30"
  exit 1
}

if [ $# -ne 4 ]; then
  usage
fi

TIME_RANGE=$1      # Time range for dashboard panels (e.g., 15m)
VUS=$2             # Number of concurrent virtual users
DURATION=$3        # Duration of test (e.g., 2m)
INTERVAL=$4        # Delay between login and dashboard test (in seconds)

# ======================================
# Paths
# ======================================
BASE_DIR="/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
RESULT_DIR="./results_combined"
mkdir -p "$RESULT_DIR"

LOGIN_CSV="${RESULT_DIR}/login_metrics.csv"
DASHBOARD_CSV="${RESULT_DIR}/dashboard_panel_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/dashboard_summary.txt"

# ======================================
# Init files
# ======================================
echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration" > "$LOGIN_CSV"
echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,\
dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,\
panel_avg_response_time,time_range,vus,vus_max,duration" > "$DASHBOARD_CSV"
echo -e "DASHBOARD PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

# ======================================
# STEP 1: LOGIN TEST (Concurrent logins)
# ======================================
echo -e "\n🔐 STEP 1: Running LOGIN TEST..."
echo "   Script: login.js"
echo "   VUs: $VUS | Duration: $DURATION"

LOGIN_OUTPUT="${RESULT_DIR}/login_${TIME_RANGE}_result.txt"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  --vus "$VUS" \
  --duration "$DURATION" \
  "$BASE_DIR/debug.js" 2>&1 | tee "$LOGIN_OUTPUT"

echo -e "\n📊 Parsing login results..."
TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")

# ======================================
# Parse login success/failure
# ======================================
grep -E "Login (successful|failed)" "$LOGIN_OUTPUT" | while IFS= read -r line; do
  if [[ "$line" =~ ✅[[:space:]]Login[[:space:]]successful[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
    USERNAME="${BASH_REMATCH[1]}"
    RESPONSE_TIME="${BASH_REMATCH[2]}"
    STATUS=200
    SUCCESS_RATE=100
    echo "✅ $USERNAME | ${RESPONSE_TIME}ms"
  elif [[ "$line" =~ ❌[[:space:]]Login[[:space:]]failed[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Status:[[:space:]]([0-9]+).*Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
    USERNAME="${BASH_REMATCH[1]}"
    STATUS="${BASH_REMATCH[2]}"
    RESPONSE_TIME="${BASH_REMATCH[3]}"
    SUCCESS_RATE=0
    echo "❌ $USERNAME | Status=$STATUS | ${RESPONSE_TIME}ms"
  else
    continue
  fi

  echo "$TIMESTAMP,login,$RESPONSE_TIME,$STATUS,$SUCCESS_RATE,$VUS,$VUS,$DURATION" >> "$LOGIN_CSV"
done

echo "✅ Completed Login Test."
echo "📂 CSV Output: $LOGIN_CSV"

# ======================================
# Insert LOGIN results into ClickHouse
# ======================================
echo -e "\n🚀 Inserting login data into ClickHouse..."
sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' "$LOGIN_CSV" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_login FORMAT CSVWithNames"

echo "✅ Login data inserted into ClickHouse."

# ======================================
# STEP 2: DASHBOARD TEST (Concurrent dashboard & panel hits)
# ======================================
echo -e "\n⏳ Waiting $INTERVAL seconds before dashboard tests..."
sleep "$INTERVAL"

echo -e "\n📊 STEP 2: Running DASHBOARD TEST..."
echo "   Script: multi_dashboard_test.js"
echo "   Time range: now-${TIME_RANGE} → now"
echo "   VUs: $VUS | Duration: $DURATION"

DASHBOARD_OUTPUT="${RESULT_DIR}/multi_dashboard_test_${TIME_RANGE}_result.txt"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  -e TIME_FROM="now-${TIME_RANGE}" \
  -e TIME_TO="now" \
  --vus "$VUS" \
  --duration "$DURATION" \
  "$BASE_DIR/debug.js" 2>&1 | tee "$DASHBOARD_OUTPUT"

# ======================================
# Parse Dashboard & Panel results
# ======================================
echo -e "\n📊 Parsing dashboard results from: $DASHBOARD_OUTPUT"

TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")

grep 'DASHBOARD_DATA:' "$DASHBOARD_OUTPUT" | while IFS= read -r line; do
  dashboard_name=$(echo "$line" | sed -E 's/.*name=([^|]+).*/\1/' | xargs)
  status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
  response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')
  
  echo "📊 Dashboard=$dashboard_name | Status=$status | Time=${response_time}ms"
  
  echo "$TIMESTAMP,$dashboard_name,$response_time,,,'$status',100,,,$response_time,$TIME_RANGE,$VUS,$VUS,$DURATION" >> "$DASHBOARD_CSV"
  echo "🔹 Dashboard: $dashboard_name — ${response_time}ms (status=$status)" >> "$SUMMARY_FILE"
done

grep '\[PANEL_DATA\]' "$DASHBOARD_OUTPUT" | while IFS= read -r line; do
  dashboard_name=$(echo "$line" | sed -E 's/.*dashboard=([^|]+).*/\1/' | xargs)
  panel_id=$(echo "$line" | sed -E 's/.*panel_id=([0-9]+).*/\1/')
  panel_name=$(echo "$line" | sed -E 's/.*panel_name=([^|]+).*/\1/' | xargs)
  status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
  response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')
  
  echo "🧩 Panel=$panel_name | Dashboard=$dashboard_name | Time=${response_time}ms"
  
  printf "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n" \
    "$TIMESTAMP" "$dashboard_name" "$response_time" "$panel_id" "$panel_name" \
    "$status" "100" "$status" "100" "$response_time" "$TIME_RANGE" "$VUS" "$VUS" "$DURATION" >> "$DASHBOARD_CSV"

  printf "  - Panel %-3s %-40s: %6.2fms (status: %s)\n" "$panel_id" "$panel_name" "$response_time" "$status" >> "$SUMMARY_FILE"
done

echo "✅ Completed Dashboard Test."
echo "📂 CSV Output: $DASHBOARD_CSV"

# ======================================
# Insert DASHBOARD results into ClickHouse
# ======================================
echo -e "\n🚀 Inserting dashboard data into ClickHouse..."
sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' "$DASHBOARD_CSV" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_results FORMAT CSVWithNames"

echo "✅ Dashboard data inserted into ClickHouse."
echo "📂 Results available in: $RESULT_DIR"
