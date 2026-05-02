#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"
USERNAME="maria.sokolova"

read -s -p "Password for ${USERNAME}: " PASSWORD
echo

echo "1) Check backend health"
curl -i "${BASE_URL}/health"
echo
echo

echo "2) Login"
LOGIN_RESPONSE=$(curl -sS -w "\n%{http_code}" -X POST "${BASE_URL}/api/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"username\": \"${USERNAME}\",
    \"password\": \"${PASSWORD}\"
  }")

HTTP_CODE=$(echo "$LOGIN_RESPONSE" | tail -n 1)
BODY=$(echo "$LOGIN_RESPONSE" | sed '$d')

echo "HTTP_CODE=${HTTP_CODE}"
echo "BODY=${BODY}"
echo

if [ "$HTTP_CODE" != "200" ]; then
  echo "Login failed"
  exit 1
fi

TOKEN=$(echo "$BODY" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')

echo "TOKEN loaded"
echo

echo "3) Check /api/me"
curl -sS "${BASE_URL}/api/me" \
  -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool

echo
echo "4) Check latest telemetry"
curl -sS "${BASE_URL}/api/telemetry/latest" \
  -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool

echo
echo "5) Check pressure history"
curl -sS "${BASE_URL}/api/telemetry/history?parameter=pressure&limit=10" \
  -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool
