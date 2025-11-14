#!/bin/bash
set -euo pipefail

# ======================================
# Usage
# ======================================
usage() {
  echo "Usage: $0 [grafana_time_ranges_csv] [vus] [total_ingestion_duration]"
  echo "Example: $0 \"15m,30m,5m\" 30 60m"
  exit 1
}

if [ $# -ne 3 ]; then
  usage
fi

# ----------------------
# Inputs
# ----------------------
TIME_RANGES_CSV="$1"        # "15m,30m,5m"
VUS="$2"                    # integer
TOTAL_DURATION_RAW="$3"     # e.g. 60m

# ----------------------
# Paths & ClickHouse
# ----------------------
BASE_DIR="/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
RESULT_DIR="./results_phase1"
mkdir -p "$RESULT_DIR"

LOGIN_CSV="${RESULT_DIR}/login_metrics.csv"
DASHBOARD_CSV="${RESULT_DIR}/dashboard_panel_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/dashboard_summary.txt"

CLICKHOUSE_POD="chi-clickhouse-vusmart-0-0-0"
CLICKHOUSE_NS="vsmaps"
CLICKHOUSE_DB="vusmart"
CLICKHOUSE_USER="vusmartmanager"
CLICKHOUSE_PASS="Vunet#1234"

# ----------------------
# Convert time ranges CSV to array
# ----------------------
IFS=',' read -r -a TIME_RANGES <<< "$TIME_RANGES_CSV"

# ----------------------
# Duration Math (Calculated in Seconds for precision)
# ----------------------
if [[ "$TOTAL_DURATION_RAW" =~ ^([0-9]+)m$ ]]; then
  TOTAL_MINUTES="${BASH_REMATCH[1]}"
  TOTAL_SECONDS=$(( TOTAL_MINUTES * 60 ))
else
  echo "TOTAL_DURATION must be in minutes format like '60m'. Got: $TOTAL_DURATION_RAW"
  exit 1
fi

# 1. Total Segment Duration (Total / 3)
SEGMENT_SECONDS=$(( TOTAL_SECONDS / 3 ))

if [ "$SEGMENT_SECONDS" -lt 10 ]; then
  echo "Error: Segment duration is too short ($SEGMENT_SECONDS sec). Increase Total Duration."
  exit 1
fi

# 2. Calculate Run Durations based on specific Phase Logic

# Segment 1 Duration: Segment / 4 (e.g., 2m / 4 = 30s)
DUR_SEG1=$(( SEGMENT_SECONDS / 4 ))
[ "$DUR_SEG1" -lt 1 ] && DUR_SEG1=1

# Segment 2 Duration: Full Segment (Parallel run takes the whole slot)
# e.g. 2m
DUR_SEG2=$(( SEGMENT_SECONDS ))

# Segment 3 Duration: Segment / 2 (e.g., 2m / 2 = 1m)
DUR_SEG3=$(( SEGMENT_SECONDS / 2 ))
[ "$DUR_SEG3" -lt 1 ] && DUR_SEG3=1

echo "============================================="
echo "Phase-1 Plan Configuration"
echo "---------------------------------------------"
echo "TIME RANGES     : ${TIME_RANGES[*]}"
echo "VUS             : $VUS"
echo "TOTAL DURATION  : ${TOTAL_MINUTES}m ($TOTAL_SECONDS s)"
echo "SEGMENT DURATION: $(( SEGMENT_SECONDS / 60 ))m ($SEGMENT_SECONDS s)"
echo "---------------------------------------------"
echo "Run Durations:"
echo "  - Seg 1 (Alt) : ${DUR_SEG1}s (Seg/4)"
echo "  - Seg 2 (Par) : ${DUR_SEG2}s (Seg)"
echo "  - Seg 3 (Seq) : ${DUR_SEG3}s (Seg/2)"
echo "============================================="

# ----------------------
# Create CSV headers (if missing)
# ----------------------
[ -f "$LOGIN_CSV" ] || echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration,segment_number,iterations" > "$LOGIN_CSV"
[ -f "$DASHBOARD_CSV" ] || echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,duration,segment_number,iterations" > "$DASHBOARD_CSV"
[ -f "$SUMMARY_FILE" ] || echo -e "DASHBOARD PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

# ----------------------
# Ensure ClickHouse tables have the segment_number and iterations columns
# ----------------------
echo "Ensuring ClickHouse tables have 'segment_number' and 'iterations' columns..."
kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
  clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" -q \
  "ALTER TABLE monitoring.k6_login ADD COLUMN IF NOT EXISTS segment_number UInt8 DEFAULT 0;"

kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
  clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" -q \
  "ALTER TABLE monitoring.k6_results ADD COLUMN IF NOT EXISTS segment_number UInt8 DEFAULT 0;"

kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
  clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" -q \
  "ALTER TABLE monitoring.k6_login ADD COLUMN IF NOT EXISTS iterations UInt64 DEFAULT 0;"

kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
  clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" -q \
  "ALTER TABLE monitoring.k6_results ADD COLUMN IF NOT EXISTS iterations UInt64 DEFAULT 0;"

# ----------------------
# Helper: run k6
# ----------------------
run_k6() {
  local script_name="$1"    # "login.js" or "multi_dashboard_test.js"
  local time_range="$2"     # e.g. 15m
  local vus="$3"
  local duration_sec="$4"   # seconds
  local label="$5"          # descriptive label
  local segnum="$6"         # segment number
  local out="${RESULT_DIR}/${label}_${time_range}.log"
  local K6_JSON="${RESULT_DIR}/${label}_${time_range}.json"

  echo -e "\n--- K6 START [$label] (range=${time_range}, vus=${vus}, duration=${duration_sec}s, seg=${segnum}) ---"
  
  if [[ "$script_name" == "login.js" ]]; then
    # This wrapper catches a non-zero exit from k6 (e.g. threshold failure)
    # and prevents `set -e` from stopping the entire script.
    ( K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
      --vus "$vus" \
      --duration "${duration_sec}s" \
      --summary-export "$K6_JSON" \
      "$BASE_DIR/login.js" ) 2>&1 | tee "$out" || echo "⚠️ k6 login run finished with non-zero status, continuing..."
  else
    ( K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
      -e TIME_FROM="now-${time_range}" \
      -e TIME_TO="now" \
      --vus "$vus" \
      --duration "${duration_sec}s" \
      --summary-export "$K6_JSON" \
      "$BASE_DIR/multi_dashboard_test.js" ) 2>&1 | tee "$out" || echo "⚠️ k6 dashboard run finished with non-zero status, continuing..."
  fi

  echo "K6 finished; log: $out"

  # Parse and insert immediately
  if [[ "$script_name" == "login.js" ]]; then
    parse_and_insert_login "$out" "$vus" "${duration_sec}s" "$segnum" "$K6_JSON"
  else
    parse_and_insert_dashboard "$out" "$time_range" "$vus" "${duration_sec}s" "$segnum" "$K6_JSON"
  fi
}

# ----------------------
# Parsing Functions (Fixed to prevent duplicates and log errors)
# ----------------------
parse_and_insert_login() {
  local login_output="$1"
  local vus="$2"
  local duration_str="$3"
  local segnum="$4"
  local K6_JSON="$5"

  # Get total iterations from JSON
  iterations=0
  if [ -f "$K6_JSON" ]; then
    iterations=$(jq '.metrics.iterations.count' "$K6_JSON")
  fi
  
  # Temp file for THIS run only
  local temp_csv="${RESULT_DIR}/temp_login_$(date +%s%N).csv"
  echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration,segment_number,iterations" > "$temp_csv"

  # This logic already correctly handles success and failure
  grep -E "Login (successful|failed)" "$login_output" | while IFS= read -r line; do
    if [[ "$line" =~ \[([^\]]+)\][[:space:]]✅[[:space:]]Login[[:space:]]successful[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
      timestamp="${BASH_REMATCH[1]}"
      timestamp=$(date -d "${timestamp}" "+%Y-%m-%d %H:%M:%S")
      USERNAME="${BASH_REMATCH[2]}"
      RESPONSE_TIME="${BASH_REMATCH[3]}"
      STATUS=200
      SUCCESS_RATE=100
    elif [[ "$line" =~ \[([^\]]+)\][[:space:]]❌[[:space:]]Login[[:space:]]failed[[:space:]]\|[[:space:]]User:[[:space:]]([^|]+)[[:space:]]\|[[:space:]]Status:[[:space:]]([0-9]+).*Response[[:space:]]Time:[[:space:]]([^|]+)[[:space:]]ms ]]; then
      timestamp="${BASH_REMATCH[1]}"
      timestamp=$(date -d "${timestamp}" "+%Y-%m-%d %H:%M:%S")
      USERNAME="${BASH_REMATCH[2]}"
      STATUS="${BASH_REMATCH[3]}"
      RESPONSE_TIME="${BASH_REMATCH[4]}"
      SUCCESS_RATE=0
    else
      continue
    fi
    
    row="\"$timestamp\",login,$RESPONSE_TIME,$STATUS,$SUCCESS_RATE,$vus,$vus,$duration_str,$segnum,$iterations"
    echo "$row" >> "$temp_csv"
    echo "$row" >> "$LOGIN_CSV" # Append to main history file
  done

  # Insert ONLY the temp file
  if [ -s "$temp_csv" ]; then
    echo "Inserting login data to ClickHouse..."
    # Read from temp_csv, skipping header
    tail -n +2 "$temp_csv" | \
      kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
      clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" \
      -q "INSERT INTO monitoring.k6_login (timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration,segment_number,iterations) FORMAT CSV"
  fi
  rm -f "$temp_csv"
}

parse_and_insert_dashboard() {
  local dash_output="$1"
  local time_range="$2"
  local vus="$3"
  local duration_str="$4"
  local segnum="$5"
  local K6_JSON="$6"

  # Get total iterations from JSON
  iterations=0
  if [ -f "$K6_JSON" ]; then
    iterations=$(jq '.metrics.iterations.count' "$K6_JSON")
  fi

  # Temp file for THIS run only
  local temp_csv="${RESULT_DIR}/temp_dash_$(date +%s%N).csv"
  echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,duration,segment_number,iterations" > "$temp_csv"

  local row=""
  local dashboard_response_time=""

  # Read the log file line by line to capture dashboard and panel data
  while IFS= read -r line; do

    # --- Dashboard ---
    if [[ "$line" == *"[DASHBOARD_DATA]"* ]]; then
      timestamp=$(echo "$line" | sed -E 's/.*time="([^"]+)".*/\1/')
      timestamp=$(date -d "${timestamp}" "+%Y-%m-%d %H:%M:%S")
      dashboard_name=$(echo "$line" | sed -E 's/.*name=([^|]+).*/\1/' | xargs)
      status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
      response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')
      local success_rate=0
      if [ "$status" -eq 200 ]; then
        success_rate=100
      fi
      dashboard_response_time="$response_time"

      row="\"$timestamp\",$dashboard_name,$response_time,,,'$status',$success_rate,,,,"$time_range",$vus,$vus,$duration_str,$segnum,$iterations"
      echo "$row" >> "$temp_csv"
      echo "$row" >> "$DASHBOARD_CSV"
      echo "🔹 Dashboard: $dashboard_name — ${response_time}ms (status=$status)" >> "$SUMMARY_FILE"

    # --- Panel ---
    elif [[ "$line" == *"[PANEL_DATA]"* ]]; then
      timestamp=$(echo "$line" | sed -E 's/.*time="([^"]+)".*/\1/')
      timestamp=$(date -d "${timestamp}" "+%Y-%m-%d %H:%M:%S")
      dashboard_name=$(echo "$line" | sed -E 's/.*dashboard=([^|]+).*/\1/' | xargs)
      panel_id=$(echo "$line" | sed -E 's/.*panel_id=([0-9]+).*/\1/')
      panel_name=$(echo "$line" | sed -E 's/.*panel_name=([^|]+).*/\1/' | xargs)
      status=$(echo "$line" | sed -E 's/.*status=([0-9]+).*/\1/')
      response_time=$(echo "$line" | sed -E 's/.*response_time=([0-9.]+)ms.*/\1/')

      local panel_success_rate=0
      if [ "$status" -eq 200 ]; then
        panel_success_rate=100
      fi

      row=$(printf "\"%s\",%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s" \
        "$timestamp" "$dashboard_name" "$dashboard_response_time" "$panel_id" "$panel_name" \
        "" "" "$status" "$panel_success_rate" "$response_time" "$time_range" "$vus" "$vus" "$duration_str" "$segnum" "$iterations")

      echo "$row" >> "$temp_csv"
      echo "$row" >> "$DASHBOARD_CSV"
      printf "  - Panel %-3s %-40s: %6.2fms (status: %s)\n" "$panel_id" "$panel_name" "$response_time" "$status" >> "$SUMMARY_FILE"

    fi
  done < "$dash_output" # Read from the log file

  # Insert ONLY the temp file
  if [ -s "$temp_csv" ]; then
    echo "Inserting dashboard data to ClickHouse..."
    # Read from temp_csv, skipping header
    tail -n +2 "$temp_csv" | \
      kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
      clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" \
      -q "INSERT INTO monitoring.k6_results (timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,duration,segment_number,iterations) FORMAT CSV"
  fi
  rm -f "$temp_csv"
}

# ======================================
# EXECUTION START
# ======================================

# --- SEGMENT 1: Alternate (Login -> Dashboard) ---
# Logic: Each run is SEGMENT/4.
# If Seg=2m, Run=30s. (Login 30s -> Dash 30s -> Login 30s -> Dash 30s)
echo -e "\n=== SEGMENT 1 (Alternate) ==="
echo "Duration: ${SEGMENT_SECONDS}s | Sub-run: ${DUR_SEG1}s"
seg=1
seg_start=$(date +%s)
elapsed=0
iter=0

while [ "$elapsed" -lt "$SEGMENT_SECONDS" ]; do
  # Safety check: if remaining time is less than sub-run duration, break
  remaining=$(( SEGMENT_SECONDS - elapsed ))
  if [ "$remaining" -lt "$DUR_SEG1" ]; then
    echo "Segment 1 time ending, moving to next segment."
    break
  fi

  tr="${TIME_RANGES[$(( iter % ${#TIME_RANGES[@]} ))]}"
  
  run_k6 "login.js" "$tr" "$VUS" "$DUR_SEG1" "seg${seg}_iter${iter}_login" "$seg"
  run_k6 "multi_dashboard_test.js" "$tr" "$VUS" "$DUR_SEG1" "seg${seg}_iter${iter}_dashboard" "$seg"

  iter=$((iter + 1))
  elapsed=$(( $(date +%s) - seg_start ))
done

# --- SEGMENT 2: Parallel (Login + Dashboard) ---
# Logic: Run parallely for the FULL segment duration.
# If Seg=2m, we run 2m Parallel.
echo -e "\n=== SEGMENT 2 (Parallel) ==="
echo "Duration: ${SEGMENT_SECONDS}s | Sub-run: ${DUR_SEG2}s"
seg=2
seg_start=$(date +%s)
elapsed=0
iter=0
PAR_VUS=$(( VUS / 2 ))
[ "$PAR_VUS" -lt 1 ] && PAR_VUS=1

while [ "$elapsed" -lt "$SEGMENT_SECONDS" ]; do
  remaining=$(( SEGMENT_SECONDS - elapsed ))
  # Since this runs for the full segment, we check a minimal threshold (e.g., 10s)
  if [ "$remaining" -lt 10 ]; then
    echo "Segment 2 time ending, moving to next segment."
    break
  fi

  tr="${TIME_RANGES[$(( iter % ${#TIME_RANGES[@]} ))]}"
  
  echo "Launching Parallel Tests..."
  run_k6 "login.js" "$tr" "$PAR_VUS" "$DUR_SEG2" "seg${seg}_iter${iter}_login_par" "$seg" &
  pid1=$!
  run_k6 "multi_dashboard_test.js" "$tr" "$PAR_VUS" "$DUR_SEG2" "seg${seg}_iter${iter}_dash_par" "$seg" &
  pid2=$!
  
  wait "$pid1" "$pid2"

  iter=$((iter + 1))
  elapsed=$(( $(date +%s) - seg_start ))
done

# --- SEGMENT 3: Sequential (Login -> Dashboard) ---
# Logic: Each run is SEGMENT/2.
# If Seg=2m, Run=1m. (Login 1m -> Dash 1m)
echo -e "\n=== SEGMENT 3 (Sequential) ==="
echo "Duration: ${SEGMENT_SECONDS}s | Sub-run: ${DUR_SEG3}s"
seg=3
seg_start=$(date +%s)
elapsed=0
iter=0

while [ "$elapsed" -lt "$SEGMENT_SECONDS" ]; do
  remaining=$(( SEGMENT_SECONDS - elapsed ))
  if [ "$remaining" -lt "$DUR_SEG3" ]; then
    echo "Segment 3 time ending."
    break
  fi

  tr="${TIME_RANGES[$(( iter % ${#TIME_RANGES[@]} ))]}"

  run_k6 "login.js" "$tr" "$VUS" "$DUR_SEG3" "seg${seg}_iter${iter}_login" "$seg"
  run_k6 "multi_dashboard_test.js" "$tr" "$VUS" "$DUR_SEG3" "seg${seg}_iter${iter}_dashboard" "$seg"

  iter=$((iter + 1))
  elapsed=$(( $(date +%s) - seg_start ))
done

echo -e "\n✅ PHASE 1 COMPLETE."
echo "Logs: $RESULT_DIR"
echo "Metrics: $LOGIN_CSV, $DASHBOARD_CSV"