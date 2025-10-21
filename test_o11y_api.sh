#!/bin/bash

# Test script for the new o11y sources topic recreation API endpoint
# This script demonstrates how to test the new API endpoint using curl

echo "Testing O11y Sources Topic Recreation API"
echo "========================================="

# API endpoint
API_URL="http://164.52.213.158:8086/api/kafka/recreate/o11y"

echo ""
echo "1. Testing with MongoDB and LinuxMonitor sources:"
echo "curl -X POST $API_URL -H 'Content-Type: application/json' -d '{\"o11ySources\": [\"MongoDB\", \"LinuxMonitor\"]}'"
echo ""

curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d '{"o11ySources": ["MongoDB", "LinuxMonitor"]}' \
  -w "\nStatus Code: %{http_code}\n" \
  -s

echo ""
echo ""
echo "2. Testing with only MSSQL source:"
echo "curl -X POST $API_URL -H 'Content-Type: application/json' -d '{\"o11ySources\": [\"Mssql\"]}'"
echo ""

curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d '{"o11ySources": ["Mssql"]}' \
  -w "\nStatus Code: %{http_code}\n" \
  -s

echo ""
echo ""
echo "3. Testing with Apache source:"
echo "curl -X POST $API_URL -H 'Content-Type: application/json' -d '{\"o11ySources\": [\"Apache\"]}'"
echo ""

curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d '{"o11ySources": ["Apache"]}' \
  -w "\nStatus Code: %{http_code}\n" \
  -s

echo ""
echo ""
echo "4. Testing error case - empty sources list:"
echo "curl -X POST $API_URL -H 'Content-Type: application/json' -d '{\"o11ySources\": []}'"
echo ""

curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d '{"o11ySources": []}' \
  -w "\nStatus Code: %{http_code}\n" \
  -s

echo ""
echo ""
echo "5. Testing error case - invalid JSON:"
echo "curl -X POST $API_URL -H 'Content-Type: application/json' -d 'invalid json'"
echo ""

curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d 'invalid json' \
  -w "\nStatus Code: %{http_code}\n" \
  -s

echo ""
echo "API Documentation:"
echo "=================="
echo "Endpoint: POST /api/kafka/recreate/o11y"
echo "Content-Type: application/json"
echo ""
echo "Request Body:"
echo '{'
echo '  "o11ySources": ["MongoDB", "LinuxMonitor", "Mssql", "Apache"]'
echo '}'
echo ""
echo "Response (Success):"
echo '{'
echo '  "success": true,'
echo '  "message": "Topics recreated successfully for specified o11y sources",'
echo '  "data": {'
echo '    "success": true,'
echo '    "results": {'
echo '      "topic1": "Successfully recreated topic topic1 with X partitions and replication factor Y",'
echo '      "topic2": "Successfully recreated topic topic2 with X partitions and replication factor Y"'
echo '    },'
echo '    "errors": []'
echo '  }'
echo '}'
echo ""
echo "Response (Partial Success):"
echo '{'
echo '  "success": false,'
echo '  "message": "Topic recreation for o11y sources completed with some errors",'
echo '  "data": {'
echo '    "success": false,'
echo '    "results": {'
echo '      "topic1": "Successfully recreated topic topic1 with X partitions and replication factor Y"'
echo '    },'
echo '    "errors": ["Failed to recreate topic topic2: error message"]'
echo '  }'
echo '}'
echo ""
echo "Source Name Translation:"
echo "- 'MongoDB' in conf.yml maps to 'MongoDB' in topics_tables.yaml"
echo "- 'LinuxMonitor' in conf.yml maps to 'Linux Monitor' in topics_tables.yaml"
echo "- 'Mssql' in conf.yml maps to 'MSSQL' in topics_tables.yaml"
echo "- 'Apache' in conf.yml maps to 'Apache' in topics_tables.yaml"