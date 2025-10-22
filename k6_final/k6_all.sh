#!/bin/bash

# ===== Configuration =====
INTERVAL=300  # 5 minutes = 300 seconds
LOG_DIR="/home/vunet/k6_final/logs"

# ===== Setup =====
mkdir -p "$LOG_DIR"
timestamp() { date '+%Y-%m-%d %H:%M:%S'; }

echo "======================================="
echo "Starting K6 dashboard scripts runner at $(timestamp)"
echo "Log directory: $LOG_DIR"
echo "Interval between scripts: ${INTERVAL}s"
echo "======================================="

# ===== Script list =====
declare -a scripts=(
 # "/home/vunet/k6_final/k6_dashboard_name/traces/overall-1.sh 15m 1 1 10"
  "/home/vunet/k6_final/k6_dashboard_name/linux-mssql-dashboard/overall-1.sh 6h 5 5 10"
  "/home/vunet/k6_final/k6_dashboard_name/login/overall.sh 5 5 10"
  "/home/vunet/k6_final/k6_dashboard_name/reports/overall.sh 1 1 10"
#  "/home/vunet/k6_final/k6_dashboard_name/log_analytics/overall-1.sh 1 1 15m"
)

# ===== Execution loop =====
for entry in "${scripts[@]}"; do
  script_path=$(echo "$entry" | awk '{print $1}')
  args=$(echo "$entry" | cut -d' ' -f2-)
  script_name=$(basename "$script_path")
  log_file="$LOG_DIR/${script_name%.sh}_$(date '+%Y%m%d_%H%M%S').log"

  if [ -x "$script_path" ]; then
    echo "[$(timestamp)] ▶ Running $script_path $args"
    bash "$script_path" $args > "$log_file" 2>&1
    echo "[$(timestamp)] ✅ Completed $script_name (logs → $log_file)"
  else
    echo "[$(timestamp)] ⚠️ Skipping $script_path — not executable."
  fi

  echo "[$(timestamp)] ⏳ Waiting ${INTERVAL}s before next script..."
  sleep "$INTERVAL"
done

echo "======================================="
echo "All scripts completed at $(timestamp)"
echo "======================================="

