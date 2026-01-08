#!/bin/bash

# ======================================
# Usage
# ======================================
if [ $# -ne 2 ]; then
  echo "Usage: $0 <vus_per_batch> <batch_count>"
  exit 1
fi

VUS=$1
BATCHES=$2
TEST_NAME="login_batch_test_2"
RESULT_DIR="./login_batch_results"
CSV="$RESULT_DIR/k6_login.csv"

mkdir -p "$RESULT_DIR"

# CSV header
echo "timestamp,test_name,avg_response_time,status_code,success_rate,vus,vus_max,iterations,segment_number,duration,throughput_rps,error_4xx,error_5xx,error_connection,response_size_bytes,concurrent_users,error_rate,request_rate,p95_response_time,p99_response_time,test_id" > "$CSV"

# Loop over batches
for ((B=1; B<=BATCHES; B++)); do
  echo "🚀 Batch $B / $BATCHES"

  OUT="$RESULT_DIR/batch_$B.log"
  TEST_ID="batch-${B}-$(date +%s)"

  # Run k6 and redirect all output to a log file
  K6_INSECURE_SKIP_TLS_VERIFY=true \
  VUS=$VUS TEST_ID=$TEST_ID \
  k6 run --vus "$VUS" --iterations "$VUS" login.js &> "$OUT"

  # Parse metrics
  TOTAL=$(grep -c "\[K6-ROW\]" "$OUT")
  OK=$(grep -c "error_rate=0" "$OUT")
  ERR4=$(grep -c "error_4xx=1" "$OUT")
  ERR5=$(grep -c "error_5xx=1" "$OUT")
  CONN=$(grep -c "error_connection=1" "$OUT")

  # Avoid division by zero
  if [ "$TOTAL" -eq 0 ]; then
    SUCCESS_RATE=0
    ERROR_RATE=1
    RPS=0
    AVG_RT=0
    P95=0
    P99=0
  else
    SUCCESS_RATE=$(awk "BEGIN { printf \"%.2f\", ($OK/$TOTAL)*100 }")
    ERROR_RATE=$(awk "BEGIN { printf \"%.4f\", 1-($OK/$TOTAL) }")
    RPS=$(awk "BEGIN { printf \"%.2f\", $TOTAL/1 }")

    # Response times - join into one line to avoid line breaks
    RTs=$(grep -o "avg_response_time=[0-9.]\+" "$OUT" | sed 's/avg_response_time=//' | paste -sd " " -)

    # Average response time
    AVG_RT=$(echo $RTs | awk '{s=0; n=0; for(i=1;i<=NF;i++){s+=$i;n++} print (n>0?s/n:0)}')

    # P95 and P99
    P95=$(echo $RTs | awk -v n=$(echo $RTs | wc -w) '{split($0,a," "); idx=int(0.95*n+0.5); if(idx<1) idx=1; print a[idx]}')
    P99=$(echo $RTs | awk -v n=$(echo $RTs | wc -w) '{split($0,a," "); idx=int(0.99*n+0.5); if(idx<1) idx=1; print a[idx]}')
  fi

  TS=$(date "+%Y-%m-%d %H:%M:%S")

  # Write CSV rows safely using process substitution
  while read -r line; do
    USER_RT=$(echo "$line" | sed -E 's/.*avg_response_time=([0-9.]+).*/\1/')
    STATUS=$(echo "$line" | sed -E 's/.*status_code=([0-9]+).*/\1/')
    SIZE=$(echo "$line" | sed -E 's/.*response_size_bytes=([0-9]+).*/\1/')

    echo "$TS,$TEST_NAME,$USER_RT,$STATUS,$SUCCESS_RATE,$VUS,$VUS,1,$B,0s,$RPS,$ERR4,$ERR5,$CONN,$SIZE,$VUS,$ERROR_RATE,$RPS,$P95,$P99,$TEST_ID" >> "$CSV"
  done < <(grep -o "\[K6-ROW\].*" "$OUT")

done

echo "🚀 Inserting into ClickHouse..."

kubectl exec -i chi-clickhouse-vusmart-0-0-0 -n vsmaps -- \
clickhouse-client -d vusmart \
--user vusmartmanager --password 'Vunet#1234' \
-q "INSERT INTO monitoring.k6_login FORMAT CSVWithNames" \
< "$CSV"

echo "✅ Done."

