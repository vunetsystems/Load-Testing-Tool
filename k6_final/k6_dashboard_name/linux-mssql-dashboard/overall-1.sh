#!/bin/bash

# ===============================
# Usage and Arguments
# ===============================
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

# ===============================
# Configuration
# ===============================
SCRIPT_DIR="/home/vunet/Load-Testing-Tool/k6_final/k6_dashboard_name/linux-mssql-dashboard"
RESULT_DIR="./results_linux_mssql"
CSV_FILE="${RESULT_DIR}/dashboard_panel_metrics.csv"
SUMMARY_FILE="${RESULT_DIR}/dashboard_summary.txt"

mkdir -p "$RESULT_DIR"

# Initialize files
echo "timestamp,dashboard_name,dashboard_avg_response_time,panel_id,panel_name,dashboard_status,dashboard_success_rate,panel_status,panel_success_rate,panel_avg_response_time,time_range,vus,vus_max,iterations" > "$CSV_FILE"
echo -e "DASHBOARD PERFORMANCE SUMMARY\n" > "$SUMMARY_FILE"

# ===============================
# Iterate over dashboard test scripts
# ===============================
for DASHBOARD_SCRIPT in "$SCRIPT_DIR"/*.js; do
  TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
  DASHBOARD_FILE=$(basename "$DASHBOARD_SCRIPT")
  echo -e "\n🔹 Running script: $DASHBOARD_FILE"
  echo "   Time range: now-${TIME_RANGE} → now"
  echo "   VUs: $VUS | Iterations: $ITERATIONS"

  OUTPUT_FILE="${RESULT_DIR}/${DASHBOARD_FILE}_${TIME_RANGE}_result.txt"

  # ===============================
  # Run the k6 test
  # ===============================
  K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
    -e TIME_FROM="now-${TIME_RANGE}" \
    -e TIME_TO="now" \
    --vus "$VUS" \
    --iterations "$ITERATIONS" \
    "$DASHBOARD_SCRIPT" 2>&1 | tee "$OUTPUT_FILE"

  # ===============================
  # Parse k6 output
  # ===============================
  echo -e "\n📊 Parsing results from: $OUTPUT_FILE"

  # We must use associative arrays (maps) to handle interleaved output
  declare -A dashboard_statuses
  declare -A dashboard_success_rates
  declare -A found_dashboards

  # --- FIRST PASS: Get Dashboard-level metrics ---
  # This pass reads the file and maps dashboard names to their status/success rate
  current_dashboard_name=""
  while IFS= read -r line; do
    if [[ "$line" =~ Dashboard:[[:space:]](.+)source=console ]]; then
      current_dashboard_name=$(echo "${BASH_REMATCH[1]}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      
      # Log to summary file only once
      if [[ -z "${found_dashboards[$current_dashboard_name]}" ]]; then
        echo -e "\n🔹 Found Dashboard: $current_dashboard_name" | tee -a "$SUMMARY_FILE"
        found_dashboards["$current_dashboard_name"]=1
      fi

    elif [[ "$line" =~ Dashboard[[:space:]]is[[:space:]]status[[:space:]]([0-9]+) ]]; then
      if [[ -n "$current_dashboard_name" ]]; then
        dashboard_statuses["$current_dashboard_name"]="${BASH_REMATCH[1]}"
      fi
    elif [[ "$line" =~ dashboard_success_rate:[[:space:]]([0-9.]+)% ]]; then
      if [[ -n "$current_dashboard_name" ]]; then
        dashboard_success_rates["$current_dashboard_name"]="${BASH_REMATCH[1]}"
        current_dashboard_name="" # Reset after success rate
      fi
    fi
  done < "$OUTPUT_FILE"


  # --- SECOND PASS: Parse Panel data and write CSV ---
  # This pass reads the file again, finds our new atomic PANEL_DATA lines,
  # and combines them with the dashboard metrics we found in pass 1.
  while IFS= read -r line; do
    
    # Match our new, robust, single-line format
    if [[ "$line" =~ PANEL_DATA:[[:space:]]dashboard_name=([^|]+)[[:space:]]*\|[[:space:]]*panel_id=([^|]+)[[:space:]]*\|[[:space:]]*panel_name=([^|]+)[[:space:]]*\|[[:space:]]*panel_status=([^|]+)[[:space:]]*\|[[:space:]]*panel_avg=([^|]+)[[:space:]]*\|[[:space:]]*panel_success_rate=([^|]+) ]]; then
      
      db_name=$(echo "${BASH_REMATCH[1]}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      panel_id="${BASH_REMATCH[2]}"
      panel_name=$(echo "${BASH_REMATCH[3]}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
      panel_status="${BASH_REMATCH[4]}"
      panel_avg="${BASH_REMATCH[5]}"
      panel_success_rate="${BASH_REMATCH[6]}"

      # Look up the dashboard metrics we stored in pass 1
      db_status=${dashboard_statuses["$db_name"]}
      db_success_rate=${dashboard_success_rates["$db_name"]}

      # Write to CSV
      panel_name_clean=$(echo "$panel_name" | sed 's/"/""/g')
      echo "$TIMESTAMP,\"$db_name\",${dashboard_avg:-},$panel_id,\"$panel_name_clean\",${db_status:-},${db_success_rate:-},${panel_status:-},${panel_success_rate:-},${panel_avg:-},$TIME_RANGE,$VUS,$VUS,$ITERATIONS" >> "$CSV_FILE"
      
      # Write to Summary
      printf "  - Panel %-3s %-40s: %6.2fms (Success: %5.1f%%)\n" "$panel_id" "$panel_name" "$panel_avg" "$panel_success_rate" >> "$SUMMARY_FILE"
    
    fi
  done < "$OUTPUT_FILE"

  echo "✅ Completed processing for: $DASHBOARD_FILE"
  echo "⏳ Waiting $INTERVAL seconds before next test..."
  sleep "$INTERVAL"
done

# ===============================
# Final Cleanup
# ===============================
sed -i 's/""""/""/g' "$CSV_FILE"
sed -i 's/,"",/,,/g' "$CSV_FILE"

echo -e "\n📂 Results saved to:"
echo "  - Detailed CSV: $CSV_FILE"
echo "  - Summary: $SUMMARY_FILE"

# ===============================
# ClickHouse Insert
# ===============================
echo -e "\n🚀 Inserting data into ClickHouse..."

sed 's/\([0-9]\+\.[0-9]\+\)%/\1/g' "$CSV_FILE" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_results FORMAT CSVWithNames"

echo "✅ Data inserted into ClickHouse table: monitoring.k6_results"
