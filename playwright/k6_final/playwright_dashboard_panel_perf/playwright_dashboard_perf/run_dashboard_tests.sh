#!/bin/bash

################################################################################
# Dashboard Performance Test Automation Script
#
# This script automatically runs Playwright dashboard performance tests in a
# continuous loop, testing two dashboards sequentially with 72-minute delays.
#
# Usage:
#   ./run_dashboard_tests.sh
#
# To run in background:
#   nohup ./run_dashboard_tests.sh > automation.log 2>&1 &
################################################################################

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CONFIG_FILE="$SCRIPT_DIR/config/test_config.json"
BACKUP_FILE="$SCRIPT_DIR/config/test_config.json.backup"
LOG_FILE="$SCRIPT_DIR/dashboard_test_automation.log"
WAIT_MINUTES=2

# Dashboard configurations
DASHBOARD_1_ID="cf5c449b-42e0-433d-9183-d5f717ae3a83"
DASHBOARD_1_NAME="Linux Server Insights"
DASHBOARD_2_ID="a83852d5-e2af-4347-ac5d-98785ffa5b95"
DASHBOARD_2_NAME="Azure Storage Blob Overview"

# Statistics
TOTAL_RUNS=0
SUCCESSFUL_RUNS=0
FAILED_RUNS=0
LOOP_COUNT=0

################################################################################
# Helper Functions
################################################################################

# Log message with timestamp
log_message() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $message" | tee -a "$LOG_FILE"
}

# Log message without timestamp (for formatting)
log_plain() {
    local message="$1"
    echo "$message" | tee -a "$LOG_FILE"
}

# Backup configuration file
backup_config() {
    if [ -f "$BACKUP_FILE" ]; then
        log_message "INFO: Backup file already exists at $BACKUP_FILE"
    else
        log_message "INFO: Creating backup of test_config.json..."
        cp "$CONFIG_FILE" "$BACKUP_FILE"
        if [ $? -eq 0 ]; then
            log_message "SUCCESS: Configuration backup created"
        else
            log_message "ERROR: Failed to create backup"
            exit 1
        fi
    fi
}

# Restore configuration from backup
restore_config() {
    if [ -f "$BACKUP_FILE" ]; then
        log_message "INFO: Restoring configuration from backup..."
        cp "$BACKUP_FILE" "$CONFIG_FILE"
        if [ $? -eq 0 ]; then
            log_message "SUCCESS: Configuration restored"
        else
            log_message "ERROR: Failed to restore configuration"
        fi
    fi
}

# Enable specific dashboard by ID
enable_dashboard() {
    local dashboard_id="$1"
    local dashboard_name="$2"

    log_message "INFO: Enabling dashboard: $dashboard_name (ID: $dashboard_id)"

    # Check if jq is available
    if command -v jq &> /dev/null; then
        # Use jq for JSON manipulation
        jq --arg id "$dashboard_id" \
           '.dashboards |= map(if .id == $id then .enabled = true else .enabled = false end)' \
           "$CONFIG_FILE" > "$CONFIG_FILE.tmp"

        if [ $? -eq 0 ]; then
            mv "$CONFIG_FILE.tmp" "$CONFIG_FILE"
            log_message "SUCCESS: Dashboard enabled using jq"
            return 0
        else
            log_message "ERROR: Failed to enable dashboard using jq"
            rm -f "$CONFIG_FILE.tmp"
            return 1
        fi
    else
        # Fallback to Python
        log_message "WARNING: jq not found, using Python fallback"
        python3 -c "
import json
import sys

try:
    with open('$CONFIG_FILE', 'r') as f:
        config = json.load(f)

    for dashboard in config['dashboards']:
        dashboard['enabled'] = (dashboard['id'] == '$dashboard_id')

    with open('$CONFIG_FILE', 'w') as f:
        json.dump(config, f, indent=2)

    print('SUCCESS: Dashboard enabled using Python')
    sys.exit(0)
except Exception as e:
    print(f'ERROR: {str(e)}', file=sys.stderr)
    sys.exit(1)
"

        if [ $? -eq 0 ]; then
            log_message "SUCCESS: Dashboard enabled using Python"
            return 0
        else
            log_message "ERROR: Failed to enable dashboard using Python"
            return 1
        fi
    fi
}

# Run Playwright test
run_test() {
    local dashboard_name="$1"

    log_plain ""
    log_plain "=========================================="
    log_message "STARTING TEST: $dashboard_name"
    log_plain "=========================================="

    TOTAL_RUNS=$((TOTAL_RUNS + 1))

    # Run the test
    cd "$SCRIPT_DIR"
    npx playwright test tests/dashboard_perf.spec.js

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        SUCCESSFUL_RUNS=$((SUCCESSFUL_RUNS + 1))
        log_plain "=========================================="
        log_message "TEST COMPLETED SUCCESSFULLY: $dashboard_name"
        log_plain "=========================================="
    else
        FAILED_RUNS=$((FAILED_RUNS + 1))
        log_plain "=========================================="
        log_message "TEST FAILED: $dashboard_name (Exit code: $exit_code)"
        log_plain "=========================================="
    fi

    log_plain ""

    return $exit_code
}

# Wait for specified minutes with progress updates
wait_interval() {
    local minutes="$1"
    local seconds=$((minutes * 60))

    log_message "INFO: Waiting for $minutes minutes before next test..."

    local end_time=$(date -d "+${minutes} minutes" '+%Y-%m-%d %H:%M:%S' 2>/dev/null)
    if [ $? -eq 0 ]; then
        log_message "INFO: Next test will start at approximately: $end_time"
    fi

    # Show countdown every 5 minutes
    local remaining=$seconds
    while [ $remaining -gt 0 ]; do
        if [ $remaining -eq $seconds ] || [ $((remaining % 300)) -eq 0 ]; then
            local mins_left=$((remaining / 60))
            log_message "INFO: $mins_left minutes remaining..."
        fi
        sleep 60
        remaining=$((remaining - 60))
    done

    log_message "INFO: Wait period completed"
}

# Print statistics
print_statistics() {
    log_plain ""
    log_plain "=========================================="
    log_plain "STATISTICS"
    log_plain "=========================================="
    log_message "Total test runs: $TOTAL_RUNS"
    log_message "Successful runs: $SUCCESSFUL_RUNS"
    log_message "Failed runs: $FAILED_RUNS"
    log_message "Loop iterations: $LOOP_COUNT"
    log_plain "=========================================="
    log_plain ""
}

# Cleanup on exit
cleanup() {
    log_plain ""
    log_message "INFO: Script interrupted (SIGINT/SIGTERM received)"
    print_statistics

    # Ask user if they want to restore config
    echo ""
    echo "Do you want to restore the original configuration? (y/n)"
    read -t 10 -n 1 response
    echo ""

    if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
        restore_config
    else
        log_message "INFO: Configuration not restored. Manual restore available from: $BACKUP_FILE"
    fi

    log_message "INFO: Script terminated"
    exit 0
}

# Check dependencies
check_dependencies() {
    log_message "INFO: Checking dependencies..."

    # Check for node/npm
    if ! command -v npx &> /dev/null; then
        log_message "ERROR: npx not found. Please install Node.js and npm"
        exit 1
    fi

    # Check for jq (optional)
    if ! command -v jq &> /dev/null; then
        log_message "WARNING: jq not found. Will use Python fallback for JSON manipulation"

        # Check for Python
        if ! command -v python3 &> /dev/null; then
            log_message "ERROR: Neither jq nor python3 found. Please install one of them"
            exit 1
        fi
    fi

    # Check if config file exists
    if [ ! -f "$CONFIG_FILE" ]; then
        log_message "ERROR: Configuration file not found: $CONFIG_FILE"
        exit 1
    fi

    log_message "SUCCESS: All dependencies are available"
}

################################################################################
# Main Execution
################################################################################

# Trap signals for graceful shutdown
trap cleanup SIGINT SIGTERM

# Clear screen and show header
clear
log_plain "################################################################################"
log_plain "#                                                                              #"
log_plain "#           Dashboard Performance Test Automation Script                      #"
log_plain "#                                                                              #"
log_plain "################################################################################"
log_plain ""

# Start logging
log_message "INFO: Script started"
log_message "INFO: Working directory: $SCRIPT_DIR"
log_message "INFO: Log file: $LOG_FILE"
log_plain ""

# Check dependencies
check_dependencies

# Create backup
backup_config

log_plain ""
log_plain "=========================================="
log_plain "CONFIGURATION"
log_plain "=========================================="
log_message "Dashboard 1: $DASHBOARD_1_NAME"
log_message "Dashboard 2: $DASHBOARD_2_NAME"
log_message "Wait interval: $WAIT_MINUTES minutes"
log_message "Mode: Continuous loop (Ctrl+C to stop)"
log_plain "=========================================="
log_plain ""

# Main loop
log_message "INFO: Starting continuous test loop..."
log_plain ""

while true; do
    LOOP_COUNT=$((LOOP_COUNT + 1))

    log_plain "=========================================="
    log_message "LOOP ITERATION: $LOOP_COUNT"
    log_plain "=========================================="
    log_plain ""

    # Test Run 1: Linux Server Insights
    enable_dashboard "$DASHBOARD_1_ID" "$DASHBOARD_1_NAME"
    if [ $? -eq 0 ]; then
        run_test "$DASHBOARD_1_NAME"
        wait_interval $WAIT_MINUTES
    else
        log_message "ERROR: Failed to enable $DASHBOARD_1_NAME, skipping test"
    fi

    # Test Run 2: Azure Storage Blob Overview
    enable_dashboard "$DASHBOARD_2_ID" "$DASHBOARD_2_NAME"
    if [ $? -eq 0 ]; then
        run_test "$DASHBOARD_2_NAME"
        wait_interval $WAIT_MINUTES
    else
        log_message "ERROR: Failed to enable $DASHBOARD_2_NAME, skipping test"
    fi

    # Print statistics after each loop
    print_statistics
done
