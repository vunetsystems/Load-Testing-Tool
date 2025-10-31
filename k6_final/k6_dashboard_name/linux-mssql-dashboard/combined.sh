#!/bin/bash

# ======================================
# Usage
# ======================================
usage() {
  echo "Usage: $0 [time-range (e.g. 15m)] [vus] [iterations] [interval-in-seconds]"
  exit 1
}

if [ $# -ne 4 ]; then
  usage
fi

TIME_RANGE=$1
VUS=$2
ITERATIONS=$3
INTERVAL=$4

# ======================================
# Adjust iteration count if invalid
# ======================================
if [ "$ITERATIONS" -lt "$VUS" ]; then
  echo "⚠️  Iterations ($ITERATIONS) < VUs ($VUS). Adjusting iterations = VUs"
  ITERATIONS=$VUS
fi

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
echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,iterations" > "$LOGIN_CSV"
echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,\
dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,\
panel_avg_response_time,time_range,vus,vus_max,iterations" > "$DASHBOARD_CSV"
echo -e "DASHBOARD PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

# ======================================
# STEP 1: LOGIN TEST
# ======================================
echo -e "\n🔐 STEP 1: Running LOGIN TEST..."
echo "   Script: login.js"
echo "   VUs: $VUS | Iterations: $ITERATIONS"

LOGIN_OUTPUT="${RESULT_DIR}/login_${TIME_RANGE}_result.txt"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  --vus "$VUS" \
  --iterations "$ITERATIONS" \
  "$BASE_DIR/login.js" 2>&1 | tee "$LOGIN_OUTPUT"

echo -e "\n📊 Parsing login results..."
TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")

# ======================================
# Parse both success and failure
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

  # Append parsed result to CSV
  echo "$TIMESTAMP,login,$RESPONSE_TIME,$STATUS,$SUCCESS_RATE,$VUS,$VUS,$ITERATIONS" >> "$LOGIN_CSV"
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
# STEP 2: DASHBOARD TEST
# ======================================
echo -e "\n⏳ Waiting $INTERVAL seconds before dashboard tests..."
sleep "$INTERVAL"

echo -e "\n📊 STEP 2: Running DASHBOARD TEST..."
echo "   Script: multi_dashboard_test.js"
echo "   Time range: now-${TIME_RANGE} → now"
echo "   VUs: $VUS | Iterations: $ITERATIONS"

DASHBOARD_OUTPUT="${RESULT_DIR}/multi_dashboard_test_${TIME_RANGE}_result.txt"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  -e TIME_FROM="now-${TIME_RANGE}" \
  -e TIME_TO="now" \
  --vus "$VUS" \
  --iterations "$ITERATIONS" \
  "$BASE_DIR/multi_dashboard_test.js" 2>&1 | tee "$DASHBOARD_OUTPUT"

# ======================================
# Parse Dashboard & Panel results
# ======================================
echo -e "\n📊 Parsing dashboard results from: $DASHBOARD_OUTPUT"

TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")

# DASHBOARD_DATA lines
grep 'DASHBOARD_DATA:' "$DASHBOARD_OUTPUT" | while IFS= read -r line; do
  dashboard_name=$(echo "$line" | sed -E 's/.*name=([^|]+).*/\1/' | xargs)
  status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
  response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')
  
  echo "📊 Dashboard=$dashboard_name | Status=$status | Time=${response_time}ms"
  
  # Append to CSV
  echo "$TIMESTAMP,$dashboard_name,$response_time,,,'$status',100,,,$response_time,$TIME_RANGE,$VUS,$VUS,$ITERATIONS" >> "$DASHBOARD_CSV"
  echo "🔹 Dashboard: $dashboard_name — ${response_time}ms (status=$status)" >> "$SUMMARY_FILE"
done

# PANEL_DATA lines
grep '\[PANEL_DATA\]' "$DASHBOARD_OUTPUT" | while IFS= read -r line; do
  dashboard_name=$(echo "$line" | sed -E 's/.*dashboard=([^|]+).*/\1/' | xargs)
  panel_id=$(echo "$line" | sed -E 's/.*panel_id=([0-9]+).*/\1/')
  panel_name=$(echo "$line" | sed -E 's/.*panel_name=([^|]+).*/\1/' | xargs)
  status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
  response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')
  
  echo "🧩 Panel=$panel_name | Dashboard=$dashboard_name | Time=${response_time}ms"
  
  # Append to CSV
  printf "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n" \
    "$TIMESTAMP" "$dashboard_name" "$response_time" "$panel_id" "$panel_name" \
    "$status" "100" "$status" "100" "$response_time" "$TIME_RANGE" "$VUS" "$VUS" "$ITERATIONS" >> "$DASHBOARD_CSV"

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
