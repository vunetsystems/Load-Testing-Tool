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
TIME_RANGES_CSV="$1"       # "15m,30m,5m"
VUS="$2"                   # integer
TOTAL_DURATION_RAW="$3"    # e.g. 60m

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
echo "TIME RANGES      : ${TIME_RANGES[*]}"
echo "VUS              : $VUS"
echo "TOTAL DURATION   : ${TOTAL_MINUTES}m ($TOTAL_SECONDS s)"
echo "SEGMENT DURATION : $(( SEGMENT_SECONDS / 60 ))m ($SEGMENT_SECONDS s)"
echo "---------------------------------------------"
echo "Run Durations:"
echo "  - Seg 1 (Alt) : ${DUR_SEG1}s (Seg/4)"
echo "  - Seg 2 (Par) : ${DUR_SEG2}s (Seg)"
echo "  - Seg 3 (Seq) : ${DUR_SEG3}s (Seg/2)"
echo "============================================="

# ----------------------
# Create CSV headers (if missing)
# ----------------------
# UPDATED: dashboard_panel_metrics.csv now matches all 30 columns
[ -f "$LOGIN_CSV" ] || echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,duration,segment_number,iterations" > "$LOGIN_CSV"
[ -f "$DASHBOARD_CSV" ] || echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations,segment_number,duration,dashboard_throughput,dashboard_error_4xx,dashboard_error_5xx,dashboard_connection_error,panel_throughput,panel_error_4xx,panel_error_5xx,panel_connection_error,panel_contribution_percent,concurrent_users,error_rate,request_rate,p95_response_time,p99_response_time" > "$DASHBOARD_CSV"
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
  local script_name="$1"     # "login.js" or "multi_dashboard_test.js"
  local time_range="$2"      # e.g. 15m
  local vus="$3"
  local duration_sec="$4"    # seconds
  local label="$5"           # descriptive label
  local segnum="$6"          # segment number
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

  # Get total iterations and req/sec from JSON
  iterations=0
  reqs_total=0
  if [ -f "$K6_JSON" ]; then
    iterations=$(jq '.metrics.iterations.count // 0' "$K6_JSON")
    reqs_total=$(jq '.metrics.http_reqs.count // 0' "$K6_JSON") # k6 total HTTP requests
  fi

  # Temp file for THIS run only
  local temp_csv="${RESULT_DIR}/temp_login_$(date +%s%N).csv"
  # This header matches the k6_login table + new columns
  echo "timestamp,test_name,avg_response_time,status_code,success_rate,throughput_rps,error_4xx,error_5xx,error_connection,response_size_bytes,vus,vus_max,duration,segment_number,iterations" > "$temp_csv"

  # Read log line by line
  grep -E "Login (successful|failed)" "$login_output" | while IFS= read -r line; do
    local timestamp RESPONSE_TIME STATUS SUCCESS_RATE RESPONSE_SIZE THROUGHPUT
    local error_4xx=0
    local error_5xx=0
    local error_connection=0

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

      if [ "$STATUS" -ge 400 ] && [ "$STATUS" -lt 500 ]; then
        error_4xx=1
      elif [ "$STATUS" -ge 500 ]; then
        error_5xx=1
      fi
    else
      # could not parse, skip
      continue
    fi

    # Throughput per iteration (req/sec)
    if (( $(echo "$RESPONSE_TIME > 0" | bc -l) )); then
      THROUGHPUT=$(echo "scale=2; 1000 / $RESPONSE_TIME" | bc)
    else
      THROUGHPUT=0
    fi

    RESPONSE_SIZE=-1 # Not captured

    row="\"$timestamp\",login,$RESPONSE_TIME,$STATUS,$SUCCESS_RATE,$THROUGHPUT,$error_4xx,$error_5xx,$error_connection,$RESPONSE_SIZE,$vus,$vus,\"$duration_str\",$segnum,$iterations"
    echo "$row" >> "$temp_csv"
    echo "$row" >> "$LOGIN_CSV"
  done

  # Insert ONLY the temp file
  if [ -s "$temp_csv" ] && [ $(wc -l < "$temp_csv") -gt 1 ]; then
    echo "Inserting login data to ClickHouse..."
    tail -n +2 "$temp_csv" | \
      kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
      clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" \
      -q "INSERT INTO monitoring.k6_login (timestamp,test_name,avg_response_time,status_code,success_rate,throughput_rps,error_4xx,error_5xx,error_connection,response_size_bytes,vus,vus_max,duration,segment_number,iterations) FORMAT CSV"
  fi
  rm -f "$temp_csv"
}


# =========================================================================
# === EDITED parse_and_insert_dashboard FUNCTION ===
# =========================================================================
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
    iterations=$(jq '.metrics.iterations.count // 0' "$K6_JSON")
  fi

  # Temp file for THIS run only
  local temp_csv="${RESULT_DIR}/temp_dash_$(date +%s%N).csv"
  
  # This header MUST match the 30 columns in the printf statements below
  echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations,segment_number,duration,dashboard_throughput,dashboard_error_4xx,dashboard_error_5xx,dashboard_connection_error,panel_throughput,panel_error_4xx,panel_error_5xx,panel_connection_error,panel_contribution_percent,concurrent_users,error_rate,request_rate,p95_response_time,p99_response_time" > "$temp_csv"

  # These variables will hold the data from the [DASHBOARD_DATA] line
  # to be used by subsequent [PANEL_DATA] lines
  local current_dash_name=""
  local current_dash_response_time=0
  local current_dash_status=0
  local current_dash_success_rate=0
  local current_dash_throughput=0
  local current_dash_err4xx=0
  local current_dash_err5xx=0
  local current_dash_conn_err=0
  
  # Read the log file line by line
  while IFS= read -r line; do

    # Skip lines that are not our custom logs
    if [[ "$line" != *"[DASHBOARD_DATA]"* ]] && [[ "$line" != *"[PANEL_DATA]"* ]]; then
      continue
    fi

    # Extract the msg content from K6 log format
    msg_content=$(echo "$line" | sed 's/.*msg="//; s/".*//')

    # --- Dashboard ---
    # Format: [DASHBOARD_DATA] | Timestamp | Dashboard Name | Status | Resp Time | Throughput | 4xx | 5xx | Conn Err
    if [[ "$msg_content" == *"[DASHBOARD_DATA]"* ]]; then
      # Use | as the delimiter
      IFS='|' read -r _ timestamp dash_name status resp_time throughput err4xx err5xx conn_err <<< "$msg_content"
      
      # Clean whitespace
      current_dash_name=$(echo "$dash_name" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//') # Clean whitespace
      current_dash_status=$(echo "$status" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      current_dash_response_time=$(echo "$resp_time" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      current_dash_throughput=$(echo "$throughput" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      current_dash_err4xx=$(echo "$err4xx" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      current_dash_err5xx=$(echo "$err5xx" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      current_dash_conn_err=$(echo "$conn_err" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      timestamp=$(echo "$timestamp" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

      if [ "$current_dash_status" -eq 200 ]; then
        current_dash_success_rate=100
      else
        current_dash_success_rate=0
      fi
      
      # Write the DASHBOARD-ONLY row
      # All panel-specific fields are 0 or ''
      # All aggregate fields (26-30) are 0
      local row
      row=$(printf "\"%s\",\"%s\",%s,%s,\"%s\",%s,%s,%s,%s,%s,\"%s\",%s,%s,%s,%s,\"%s\",%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s" \
        "$timestamp" "$current_dash_name" "$current_dash_response_time" \
        0 "" "$current_dash_status" "$current_dash_success_rate" \
        0 0 0 \
        "$time_range" "$vus" "$vus" "$iterations" "$segnum" "$duration_str" \
        "$current_dash_throughput" "$current_dash_err4xx" "$current_dash_err5xx" "$current_dash_conn_err" \
        0 0 0 0 0 \
        0 0 0 0 0
      )
      
      echo "$row" >> "$temp_csv"
      echo "$row" >> "$DASHBOARD_CSV"
      echo "🔹 Dashboard: $current_dash_name — ${current_dash_response_time}ms (status=$current_dash_status)" >> "$SUMMARY_FILE"

    # --- Panel ---
    # Format: [PANEL_DATA] | Timestamp | Dashboard Name | Panel ID | Panel Name | Status | Resp Time | Throughput | 4xx | 5xx | Conn Err | Contribution %
    elif [[ "$msg_content" == *"[PANEL_DATA]"* ]]; then
      IFS='|' read -r _ timestamp dash_name panel_id panel_name status resp_time throughput err4xx err5xx conn_err contribution <<< "$msg_content"

      # Clean variables
      local panel_id=$(echo "$panel_id" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_name=$(echo "$panel_name" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//') # Clean whitespace
      local panel_status=$(echo "$status" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_resp_time=$(echo "$resp_time" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_throughput=$(echo "$throughput" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_err4xx=$(echo "$err4xx" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_err5xx=$(echo "$err5xx" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_conn_err=$(echo "$conn_err" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local panel_contribution=$(echo "$contribution" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      local timestamp=$(echo "$timestamp" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      
      local panel_success_rate=0
      if [ "$panel_status" -eq 200 ]; then
        panel_success_rate=100
      fi
      
      # Write the PANEL row
      # We use the 'current_dash_*' variables saved from the last dashboard line
      # All aggregate fields (26-30) are 0
      local row
      row=$(printf "\"%s\",\"%s\",%s,%s,\"%s\",%s,%s,%s,%s,%s,\"%s\",%s,%s,%s,%s,\"%s\",%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s" \
        "$timestamp" "$current_dash_name" "$current_dash_response_time" \
        "$panel_id" "$panel_name" "$current_dash_status" "$current_dash_success_rate" \
        "$panel_status" "$panel_success_rate" "$panel_resp_time" \
        "$time_range" "$vus" "$vus" "$iterations" "$segnum" "$duration_str" \
        "$current_dash_throughput" "$current_dash_err4xx" "$current_dash_err5xx" "$current_dash_conn_err" \
        "$panel_throughput" "$panel_err4xx" "$panel_err5xx" "$panel_conn_err" "$panel_contribution" \
        0 0 0 0 0
      )
      
      echo "$row" >> "$temp_csv"
      echo "$row" >> "$DASHBOARD_CSV"
      printf "   - Panel %-3s %-40s: %6.2fms (status: %s)\n" "$panel_id" "$panel_name" "$panel_resp_time" "$panel_status" >> "$SUMMARY_FILE"
    
    fi
  done < "$dash_output" # Read from the log file

  # Insert ONLY the temp file (if it has data)
  if [ -s "$temp_csv" ] && [ $(wc -l < "$temp_csv") -gt 1 ]; then
    echo "Inserting dashboard data to ClickHouse..."
    # Read from temp_csv, skipping header
    tail -n +2 "$temp_csv" | \
      kubectl exec -i "$CLICKHOUSE_POD" -n "$CLICKHOUSE_NS" -- \
      clickhouse-client -d "$CLICKHOUSE_DB" --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASS" \
      -q "INSERT INTO monitoring.k6_results (timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations,segment_number,duration,dashboard_throughput,dashboard_error_4xx,dashboard_error_5xx,dashboard_connection_error,panel_throughput,panel_error_4xx,panel_error_5xx,panel_connection_error,panel_contribution_percent,concurrent_users,error_rate,request_rate,p95_response_time,p99_response_time) FORMAT CSV"
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