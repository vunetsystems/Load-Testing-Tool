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

# Example parser for LOGIN metrics
grep "✅ Login successful" "$LOGIN_OUTPUT" | while IFS= read -r line; do
  if [[ "$line" =~ ✅[[:space:]]Login[[:space:]]successful[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
    USERNAME="${BASH_REMATCH[1]}"
    RESPONSE_TIME="${BASH_REMATCH[2]}"
    echo "$TIMESTAMP,login,$RESPONSE_TIME,200,100,$VUS,$VUS,$ITERATIONS" >> "$LOGIN_CSV"
  fi
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
# Parse Dashboard results
# ======================================
echo -e "\n📊 Parsing dashboard results from: $DASHBOARD_OUTPUT"

declare -A dashboard_statuses
declare -A dashboard_success_rates
declare -A found_dashboards
current_dashboard_name=""

while IFS= read -r line; do
  if [[ "$line" =~ Dashboard:[[:space:]](.+)source=console ]]; then
    current_dashboard_name=$(echo "${BASH_REMATCH[1]}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    if [[ -z "${found_dashboards[$current_dashboard_name]}" ]]; then
      echo -e "\n🔹 Found Dashboard: $current_dashboard_name" | tee -a "$SUMMARY_FILE"
      found_dashboards["$current_dashboard_name"]=1
    fi
  elif [[ "$line" =~ Dashboard[[:space:]]is[[:space:]]status[[:space:]]([0-9]+) ]]; then
    dashboard_statuses["$current_dashboard_name"]="${BASH_REMATCH[1]}"
  elif [[ "$line" =~ dashboard_success_rate:[[:space:]]([0-9.]+)% ]]; then
    dashboard_success_rates["$current_dashboard_name"]="${BASH_REMATCH[1]}"
    current_dashboard_name=""
  fi
done < "$DASHBOARD_OUTPUT"

while IFS= read -r line; do
  if [[ "$line" =~ PANEL_DATA:[[:space:]]dashboard_name=([^|]+)[[:space:]]*\|[[:space:]]*panel_id=([^|]+)[[:space:]]*\|[[:space:]]*panel_name=([^|]+)[[:space:]]*\|[[:space:]]*panel_status=([^|]+)[[:space:]]*\|[[:space:]]*panel_avg=([^|]+)[[:space:]]*\|[[:space:]]*panel_success_rate=([^|]+) ]]; then
    echo "DEBUG: Matched line: $line" >&2
    db_name=$(echo "${BASH_REMATCH[1]}" | xargs)
    panel_id="${BASH_REMATCH[2]}"
    panel_name=$(echo "${BASH_REMATCH[3]}" | xargs)
    panel_status="${BASH_REMATCH[4]}"
    panel_avg="${BASH_REMATCH[5]}"
    panel_success_rate="${BASH_REMATCH[6]}"

    # Clean the captured values
    panel_avg=$(echo "$panel_avg" | sed 's/[[:space:]]*$//; s/ source=console.*//')
    panel_success_rate=$(echo "$panel_success_rate" | sed 's/[[:space:]]*$//; s/ source=console.*//')

    db_status=${dashboard_statuses["$db_name"]}
    db_success_rate=${dashboard_success_rates["$db_name"]}
    clean_db_name=$(echo "$db_name" | sed 's/"//g' | sed 's/,/ /g')
    clean_panel_name=$(echo "$panel_name" | sed 's/"//g' | sed 's/,/ /g')
    clean_dashboard_avg=$(echo "${dashboard_avg:-0}" | sed 's/[^0-9.\-]//g')
    clean_panel_avg=$(echo "${panel_avg:-0}" | sed 's/[^0-9.\-]//g')
    clean_panel_success_rate=$(echo "${panel_success_rate:-0}" | sed 's/[^0-9.\-]//g')
    clean_db_success_rate=$(echo "${db_success_rate:-0}" | sed 's/[^0-9.\-]//g')

    printf "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n" \
      "$TIMESTAMP" "$clean_db_name" "$clean_dashboard_avg" "$panel_id" "$clean_panel_name" \
      "${db_status:-0}" "$clean_db_success_rate" "$panel_status" "$clean_panel_success_rate" \
      "$clean_panel_avg" "$TIME_RANGE" "$VUS" "$VUS" "$ITERATIONS" >> "$DASHBOARD_CSV"
    printf "  - Panel %-3s %-40s: %6.2fms (Success: %5.1f%%)\n" "$panel_id" "$panel_name" "$clean_panel_avg" "$clean_panel_success_rate" >> "$SUMMARY_FILE"
  fi
done < "$DASHBOARD_OUTPUT"

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
