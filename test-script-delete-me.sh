cat > /tmp/send_combined_risk_test.sh <<'SH'
#!/usr/bin/env bash
set -euo pipefail

for i in 0 1 2 3 4; do
  case $i in
    0)
      pressure=65
      moisture=26.5
      drive_load=55
      ;;
    1)
      pressure=67
      moisture=26.0
      drive_load=57
      ;;
    2)
      pressure=69
      moisture=25.5
      drive_load=60
      ;;
    3)
      pressure=71
      moisture=25.0
      drive_load=62
      ;;
    4)
      pressure=73
      moisture=24.5
      drive_load=64
      ;;
  esac

  measured_at="2026-05-02T11:30:0${i}Z"

  echo "send batch $i: pressure=$pressure moisture=$moisture drive_load=$drive_load"

  curl -sS -X POST http://localhost:8080/api/telemetry \
    -H "Content-Type: application/json" \
    -d "{
      \"parameterType\": \"pressure\",
      \"value\": $pressure,
      \"unit\": \"bar\",
      \"sourceId\": \"manual-combined-risk-test\",
      \"measuredAt\": \"$measured_at\"
    }" > /dev/null

  curl -sS -X POST http://localhost:8080/api/telemetry \
    -H "Content-Type: application/json" \
    -d "{
      \"parameterType\": \"moisture\",
      \"value\": $moisture,
      \"unit\": \"percent\",
      \"sourceId\": \"manual-combined-risk-test\",
      \"measuredAt\": \"$measured_at\"
    }" > /dev/null

  curl -sS -X POST http://localhost:8080/api/telemetry \
    -H "Content-Type: application/json" \
    -d "{
      \"parameterType\": \"drive_load\",
      \"value\": $drive_load,
      \"unit\": \"percent\",
      \"sourceId\": \"manual-combined-risk-test\",
      \"measuredAt\": \"$measured_at\"
    }" > /dev/null
done
SH

bash /tmp/send_combined_risk_test.sh