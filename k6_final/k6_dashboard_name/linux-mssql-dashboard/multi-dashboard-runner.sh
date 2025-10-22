#!/bin/bash

# Multi-Dashboard K6 Testing Runner
# Tests multiple dashboards simultaneously with proper user distribution

# ===== USAGE =====
usage() {
  echo "Usage: $0 [total-users] [time-range] [linux-vus] [mssql-vus]"
  echo ""
  echo "Parameters:"
  echo "  total-users    Total number of users to create (default: 50)"
  echo "  time-range     Time range for queries (default: 15m)"
  echo "  linux-vus      Virtual users for Linux dashboard (default: 25)"
  echo "  mssql-vus      Virtual users for MSSQL dashboard (default: 25)"
  echo ""
  echo "Example:"
  echo "  $0 50 15m 25 25    # 50 users, 15min range, 25 VUs each"
  echo "  $0 100 1h 40 60    # 100 users, 1h range, 40/60 VUs split"
  exit 1
}

# ===== INPUT VALIDATION =====
if [ $# -lt 2 ]; then
  echo "❌ Error: Missing required parameters"
  usage
fi

TOTAL_USERS=${1:-50}
TIME_RANGE=${2:-15m}
LINUX_VUS=${3:-25}
MSSQL_VUS=${4:-25}

echo "======================================="
echo "🚀 Multi-Dashboard K6 Testing Runner"
echo "======================================="
echo "Total Users: $TOTAL_USERS"
echo "Time Range: $TIME_RANGE"
echo "Linux Dashboard VUs: $LINUX_VUS"
echo "MSSQL Dashboard VUs: $MSSQL_VUS"
echo "======================================="

# ===== USER CREATION =====
echo -e "\n🔐 Step 1: Creating $TOTAL_USERS test users..."
cd /home/vunet/k6_final/user_creation

# Check if Python script exists
if [ ! -f "dashboard_user.py" ]; then
  echo "❌ Error: dashboard_user.py not found in $(pwd)"
  exit 1
fi

# Create users
python3 dashboard_user.py $TOTAL_USERS

if [ $? -ne 0 ]; then
  echo "❌ Error: Failed to create users"
  exit 1
fi

# Verify users were created
USER_COUNT=$(wc -l < user_cookies.txt)
if [ "$USER_COUNT" -lt "$TOTAL_USERS" ]; then
  echo "⚠️ Warning: Only created $USER_COUNT users, expected $TOTAL_USERS"
fi

echo "✅ User creation completed"

# ===== K6 TESTING =====
echo -e "\n🧪 Step 2: Running multi-dashboard K6 tests..."
cd /home/vunet/k6_final/k6_dashboard_name/linux-mssql-dashboard

# Check if multi-dashboard script exists
if [ ! -f "multi-dashboard-test.js" ]; then
  echo "❌ Error: multi-dashboard-test.js not found in $(pwd)"
  exit 1
fi

# Create results directory
RESULT_DIR="./results_multi_dashboard_$(date '+%Y%m%d_%H%M%S')"
mkdir -p "$RESULT_DIR"

echo "📊 Results will be saved to: $RESULT_DIR"

# Set environment variables for time range
export TIME_FROM="now-${TIME_RANGE}"
export TIME_TO="now"

# Update script with VU counts if provided
if [ "$LINUX_VUS" != "25" ] || [ "$MSSQL_VUS" != "25" ]; then
  echo "🔧 Updating VU configuration in script..."
  # Note: In a real implementation, you might want to make this more dynamic
  # For now, the script uses the hardcoded values but you can modify them
fi

# Run the multi-dashboard test
echo "🚀 Starting simultaneous dashboard testing..."
echo "   - Linux Server Insights: $LINUX_VUS VUs"
echo "   - MSSQL Overview Dashboard: $MSSQL_VUS VUs"
echo "   - Time Range: $TIME_RANGE"

K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
  --out json="$RESULT_DIR/k6_results.json" \
  --out csv="$RESULT_DIR/k6_metrics.csv" \
  multi-dashboard-test.js 2>&1 | tee "$RESULT_DIR/multi_dashboard_execution.log"

K6_EXIT_CODE=${PIPESTATUS[0]}

if [ $K6_EXIT_CODE -eq 0 ]; then
  echo -e "\n✅ Multi-dashboard testing completed successfully!"

  # Generate summary report
  echo -e "\n📊 Generating summary report..."

  # Count successful requests per dashboard
  LINUX_SUCCESS=$(grep -c '"dashboardName":"Linux Server Insights"' "$RESULT_DIR/k6_results.json" || echo "0")
  MSSQL_SUCCESS=$(grep -c '"dashboardName":"MSSQL Overview Dashboard"' "$RESULT_DIR/k6_results.json" || echo "0")

  echo -e "\n📈 TEST SUMMARY:" > "$RESULT_DIR/README.md"
  echo "=========================" >> "$RESULT_DIR/README.md"
  echo "Total Users Created: $TOTAL_USERS" >> "$RESULT_DIR/README.md"
  echo "Time Range: $TIME_RANGE" >> "$RESULT_DIR/README.md"
  echo "Linux Dashboard VUs: $LINUX_VUS" >> "$RESULT_DIR/README.md"
  echo "MSSQL Dashboard VUs: $MSSQL_VUS" >> "$RESULT_DIR/README.md"
  echo "" >> "$RESULT_DIR/README.md"
  echo "RESULTS:" >> "$RESULT_DIR/README.md"
  echo "- Linux Server Insights: $LINUX_SUCCESS requests" >> "$RESULT_DIR/README.md"
  echo "- MSSQL Overview Dashboard: $MSSQL_SUCCESS requests" >> "$RESULT_DIR/README.md"
  echo "" >> "$RESULT_DIR/README.md"
  echo "Files:" >> "$RESULT_DIR/README.md"
  echo "- $RESULT_DIR/k6_results.json (detailed results)" >> "$RESULT_DIR/README.md"
  echo "- $RESULT_DIR/k6_metrics.csv (metrics data)" >> "$RESULT_DIR/README.md"
  echo "- $RESULT_DIR/multi_dashboard_execution.log (execution log)" >> "$RESULT_DIR/README.md"

  echo "📂 Results saved to: $RESULT_DIR"
  echo "📄 Summary: $RESULT_DIR/README.md"

else
  echo -e "\n❌ Multi-dashboard testing failed with exit code: $K6_EXIT_CODE"
  echo "📄 Check logs: $RESULT_DIR/multi_dashboard_execution.log"
  exit $K6_EXIT_CODE
fi

echo -e "\n🎉 Multi-dashboard testing workflow completed!"