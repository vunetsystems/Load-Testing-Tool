#!/bin/bash

# Usage hint
usage() {
  echo "Usage: $0 [time-range (e.g. 15m)] [vus] [iterations] [interval-in-seconds]"
  exit 1
}

# Check input args
if [ $# -ne 4 ]; then
  usage
fi

# ====== USER INPUTS ======
TIME_RANGE=$1
VUS=$2
ITERATIONS=$3
INTERVAL=$4

# ====== CONFIGURATION ======
SCRIPT_DIR="/home/vunet/k6_final/k6_dashboard_name/linux-mssql-dashboard"
RESULT_DIR="./results_linux_mssql"
CSV_FILE="${RESULT_DIR}/dashboard_panel_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/dashboard_summary.txt"

mkdir -p "$RESULT_DIR"

# Initialize output files
echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations" > "$CSV_FILE"
echo -e "DASHBOARD PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

# Process each dashboard
for DASHBOARD_SCRIPT in "$SCRIPT_DIR"/*.js; do
  DASHBOARD_NAME=$(basename "$DASHBOARD_SCRIPT" .js | cut -d'_' -f1)
  TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
  
  echo -e "\n🔹 Processing Dashboard: $DASHBOARD_NAME"
  echo "   Script: $(basename $DASHBOARD_SCRIPT)"
  echo "   Time range: now-${TIME_RANGE} → now"
  echo "   VUs: $VUS | Iterations: $ITERATIONS"

  OUTPUT_FILE="${RESULT_DIR}/${DASHBOARD_NAME}_${TIME_RANGE}_result.txt"

  # Run k6 test
  K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
    -e TIME_FROM="now-${TIME_RANGE}" \
    -e TIME_TO="now" \
    --vus "$VUS" \
    --iterations "$ITERATIONS" \
    "$DASHBOARD_SCRIPT" | tee "$OUTPUT_FILE"

  # Extract dashboard metrics
  # Extract dashboard metrics
  DASHBOARD_AVG=$(grep "dashboard_response_time" "$OUTPUT_FILE" | grep -oP 'avg=\K[0-9.]+')
  DASHBOARD_STATUS=$(grep -Po 'Dashboard is status \K[0-9]+' "$OUTPUT_FILE" | head -n 1)
  DASHBOARD_SUCCESS_RATE=$(grep -Po 'dashboard_success_rate.*?: \K[0-9.]+%' "$OUTPUT_FILE" | head -n 1)


  echo -e "\n📊 Dashboard Average Response Time: ${DASHBOARD_AVG}ms" | tee -a "$SUMMARY_FILE"


  # Extract panel details
  declare -A PANEL_NAME_MAP
  declare -A PANEL_STATUS_MAP

  while IFS= read -r line; do 
    # Adjusted regex to match "Panel ID (Panel Name) is status <status>"
    if [[ "$line" =~ Panel[[:space:]]([0-9]+)[[:space:]]*\(([^\)]+)\)[[:space:]]is[[:space:]]status[[:space:]]([0-9]+) ]]; then
        panel_id="${BASH_REMATCH[1]}"
        panel_name="${BASH_REMATCH[2]}"
        panel_status="${BASH_REMATCH[3]}"
        PANEL_NAME_MAP["$panel_id"]="$panel_name"
        PANEL_STATUS_MAP["$panel_id"]="$panel_status"
    fi
  done < "$OUTPUT_FILE"

  # Process panel metrics
  echo -e "\n📋 Panel Response Times:" | tee -a "$SUMMARY_FILE"
  declare -A PANEL_DATA
  while IFS= read -r metric; do
    if [[ "$metric" =~ panel_response_time_([0-9]+).*avg=([0-9.]+)ms ]]; then
      panel_id="${BASH_REMATCH[1]}"
      PANEL_DATA["${panel_id}_response"]="${BASH_REMATCH[2]}"
    elif [[ "$metric" =~ panel_success_rate_([0-9]+).*:\ ([0-9.]+)% ]]; then
      panel_id="${BASH_REMATCH[1]}"
      PANEL_DATA["${panel_id}_success"]="${BASH_REMATCH[2]}"
      success_count=$(echo "$metric" | grep -oP '([0-9]+) out of ([0-9]+)' | cut -d' ' -f1)
      total_count=$(echo "$metric" | grep -oP '([0-9]+) out of ([0-9]+)' | cut -d' ' -f3)
      failure_count=$((total_count - success_count))
      PANEL_DATA["${panel_id}_success_count"]="$success_count"
      PANEL_DATA["${panel_id}_failure_count"]="$failure_count"
    fi
  done < <(grep -E "panel_response_time_[0-9]+|panel_success_rate_[0-9]+" "$OUTPUT_FILE")

  # Write to CSV and display
  for panel_id in $(echo "${!PANEL_DATA[@]}" | tr ' ' '\n' | cut -d'_' -f1 | sort -u); do
    panel_name="${PANEL_NAME_MAP[$panel_id]}"
    panel_status="${PANEL_STATUS_MAP[$panel_id]}"
    response_time="${PANEL_DATA[${panel_id}_response]}"
    success_rate="${PANEL_DATA[${panel_id}_success]}"
    success_count="${PANEL_DATA[${panel_id}_success_count]:-0}"
    failure_count="${PANEL_DATA[${panel_id}_failure_count]:-0}"
    
    if [[ -n "$response_time" && -n "$success_rate" ]]; then
      # Clean and escape panel name for CSV
      panel_name_clean=$(echo "$panel_name" | sed 's/"/""/g')
      
      # Write to CSV (with all required fields)
      echo "$TIMESTAMP,$DASHBOARD_NAME,$DASHBOARD_AVG,$panel_id,\"$panel_name_clean\",$DASHBOARD_STATUS,$DASHBOARD_SUCCESS_RATE,$panel_status,$success_rate,$response_time,$TIME_RANGE,$VUS,$VUS,$ITERATIONS" >> "$CSV_FILE"
      
      # Display in summary
      printf "  - Panel %-3s %-40s: %6.2fms (Success: %5.1f%%, %d✓ %d✗)\n" \
        "$panel_id" "$panel_name" "$response_time" "$success_rate" "$success_count" "$failure_count" | tee -a "$SUMMARY_FILE"
    fi
  done

  echo "✅ Completed Dashboard: $DASHBOARD_NAME"
  echo "⏳ Waiting $INTERVAL seconds before next test..."
  sleep "$INTERVAL"
done

# Final CSV cleanup to ensure proper formatting
sed -i 's/""""/""/g' "$CSV_FILE"  # Fix double-escaped quotes
sed -i 's/,"",/,,/g' "$CSV_FILE"   # Remove empty quoted fields

echo -e "\n📂 Results saved to:"
echo "  - Detailed CSV: $CSV_FILE"
echo "  - Summary: $SUMMARY_FILE"

# Insert the final CSV into ClickHouse
echo -e "\n🚀 Inserting data into ClickHouse..."

sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' "$CSV_FILE" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_results FORMAT CSVWithNames"

echo "✅ Data inserted into ClickHouse table: monitoring.k6_results"

