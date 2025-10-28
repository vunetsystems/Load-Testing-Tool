#!/bin/bash

# Usage check
if [ "$#" -ne 3 ]; then
  echo "Usage: bash test.sh <VUS> <ITERATIONS> <TIME_RANGE>"
  echo "Example: bash test.sh 5 5 15m"
  exit 1
fi

VUS=$1
ITERATIONS=$2
TIME_RANGE=$3

FILTER_FILE="/home/vunet/k6_dashboard_name/log_analytics/filters.txt"
K6_SCRIPT="/home/vunet/k6_dashboard_name/log_analytics/log_analytics.js"
CSV_FILE="./results/log_analytics_results.csv"
SUMMARY_FILE="./results/log_analytics_summary.txt"

mkdir -p ./results

# Prepare CSV headers if not exists
if [ ! -f "$CSV_FILE" ]; then
  echo "timestamp,filter,avg_response_time_ms,vus,iterations,status_code,time_range" > "$CSV_FILE"
fi

# Time calculations
END_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Convert <TIME_RANGE> like 15m, 1h
TIME_NUMBER=$(echo "$TIME_RANGE" | grep -oE '^[0-9]+')
TIME_UNIT=$(echo "$TIME_RANGE" | grep -oE '[a-zA-Z]+$')

case "$TIME_UNIT" in
  m) DATE_ARG="$TIME_NUMBER minutes ago" ;;
  h) DATE_ARG="$TIME_NUMBER hours ago" ;;
  d) DATE_ARG="$TIME_NUMBER days ago" ;;
  *) echo "❌ Unsupported time unit: $TIME_UNIT"; exit 1 ;;
esac

START_TIME=$(date -u -d "$DATE_ARG" +"%Y-%m-%dT%H:%M:%SZ")

# Loop through each filter in the file
while IFS= read -r FILTER_LINE || [[ -n "$FILTER_LINE" ]]; do
  RAW_FILTER=$(echo "$FILTER_LINE" | xargs)
  [ -z "$RAW_FILTER" ] && continue

  FILTER_LIST=$(echo "$RAW_FILTER" | awk -F '+' '{
    print "["
    for (i = 1; i <= NF; i++) {
      gsub(/"/, "", $i)
      split($i, kv, ":")
      printf "  {\"key\": \"%s\", \"value\": \"%s\"}%s\n", kv[1], kv[2], (i<NF ? "," : "")
    }
    print "]"
  }')

  FILTER_JSON=$(echo "$FILTER_LIST" | jq -c .)
  echo -e "\n▶ Running test for filter: $RAW_FILTER"
  TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")

  OUTPUT_FILE=$(mktemp)
  K6_INSECURE_SKIP_TLS_VERIFY=true k6 run \
    --env FILTER="$FILTER_JSON" \
    --env START_TIME="$START_TIME" \
    --env END_TIME="$END_TIME" \
    --env VUS="$VUS" \
    --env ITERATIONS="$ITERATIONS" \
    "$K6_SCRIPT" > "$OUTPUT_FILE" 2>&1

  cat "$OUTPUT_FILE"

  # Extract avg response time in ms (strip trailing ms if exists)
  avg_response_time=$(grep 'http_req_duration' "$OUTPUT_FILE" | grep -Eo 'avg=[0-9.]+ms' | head -1 | sed 's/avg=//' | sed 's/ms//')
  [ -z "$avg_response_time" ] && avg_response_time="N/A"

  # Extract last HTTP status code seen in user output
  status_code=$(grep -o 'Status: [0-9]\+' "$OUTPUT_FILE" | tail -1 | awk '{print $2}')
  [ -z "$status_code" ] && status_code="N/A"

  # Write to CSV and summary
  echo "$TIMESTAMP,\"$RAW_FILTER\",$avg_response_time,$VUS,$ITERATIONS,$status_code,$TIME_RANGE" >> "$CSV_FILE"
  echo "✓ Status: $status_code | Filter: $RAW_FILTER | Avg Time: ${avg_response_time}ms" >> "$SUMMARY_FILE"

  rm "$OUTPUT_FILE"
done < "$FILTER_FILE"

echo -e "\n✅ Test completed. Results saved to $CSV_FILE"

echo -e "\n🚀 Inserting data into ClickHouse..."

sed -e 's/\([,]\)NaN\([,]\)/\10.0\2/g' \
    -e 's/\([,]\)nan\([,]\)/\10.0\2/g' \
    -e 's/,\([,]\)/,0.0\1/g' "$CSV_FILE" | \
kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart --user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_log_analytics FORMAT CSVWithNames"

echo "✅ Data inserted into ClickHouse table: monitoring.k6_log_analytics"

