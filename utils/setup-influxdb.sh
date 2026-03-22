#!/bin/bash
# InfluxDB Setup Script for hft-micro-dydx
# Target: 192.155.88.24:8086
# Chronograf: 192.155.88.24:8888
set -euo pipefail

INFLUX_HOST="192.155.88.24"
INFLUX_PORT="8086"
INFLUX_URL="http://${INFLUX_HOST}:${INFLUX_PORT}"
CHRONOGRAF_URL="http://${INFLUX_HOST}:8888"

echo "=== InfluxDB Setup for hft-micro-dydx ==="
echo "InfluxDB:   $INFLUX_URL"
echo "Chronograf: $CHRONOGRAF_URL"
echo ""

# Test connectivity
echo "[1/6] Testing InfluxDB connectivity..."
RESP=$(curl -s -w "\n%{http_code}" "${INFLUX_URL}/ping")
HTTP_CODE=$(echo "$RESP" | tail -1)
if [ "$HTTP_CODE" = "204" ]; then
    VERSION=$(curl -s -I "${INFLUX_URL}/ping" | grep -i 'X-Influxdb-Version' | tr -d '\r' | awk '{print $2}')
    echo "  OK - InfluxDB version: ${VERSION:-unknown}"
else
    echo "  FAIL (HTTP $HTTP_CODE) - Cannot reach InfluxDB"
    exit 1
fi

echo "[2/6] Testing Chronograf connectivity..."
RESP=$(curl -s -o /dev/null -w "%{http_code}" "${CHRONOGRAF_URL}" 2>/dev/null)
if [ "$RESP" = "200" ] || [ "$RESP" = "301" ] || [ "$RESP" = "302" ]; then
    echo "  OK - Chronograf reachable"
else
    echo "  WARN (HTTP $RESP) - Chronograf may not be reachable"
fi

# Show existing databases
echo "[3/6] Listing existing databases..."
curl -s "${INFLUX_URL}/query?q=SHOW+DATABASES" | python3 -c "
import sys, json
data = json.load(sys.stdin)
dbs = [v[0] for v in data['results'][0]['series'][0]['values']]
for db in sorted(dbs):
    print(f'  - {db}')
"

# Create databases for hft-micro
echo "[4/6] Creating databases for hft-micro strategies..."

# Main trading database - internal metrics (high frequency, short retention)
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode "q=CREATE DATABASE hft_internal" > /dev/null
echo "  Created: hft_internal (internal strategy metrics)"

# External/reporting database (lower frequency, longer retention)
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode "q=CREATE DATABASE hft_external" > /dev/null
echo "  Created: hft_external (external reporting metrics)"

# Research/backtest database
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode "q=CREATE DATABASE hft_research" > /dev/null
echo "  Created: hft_research (research and backtest data)"

# Market data recording database
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode "q=CREATE DATABASE hft_marketdata" > /dev/null
echo "  Created: hft_marketdata (recorded market data)"

# Set up retention policies
echo "[5/6] Setting up retention policies..."

# Internal: 7 days (high frequency data, short-lived)
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode \
    "q=CREATE RETENTION POLICY rp_7d ON hft_internal DURATION 7d REPLICATION 1 DEFAULT" > /dev/null
echo "  hft_internal: 7d retention (default)"

# Also create 30d policy for important internal metrics
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode \
    "q=CREATE RETENTION POLICY rp_30d ON hft_internal DURATION 30d REPLICATION 1" > /dev/null
echo "  hft_internal: 30d retention (optional)"

# External: 90 days
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode \
    "q=CREATE RETENTION POLICY rp_90d ON hft_external DURATION 90d REPLICATION 1 DEFAULT" > /dev/null
echo "  hft_external: 90d retention (default)"

# Research: infinite (keep all backtest data)
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode \
    "q=CREATE RETENTION POLICY rp_inf ON hft_research DURATION 0s REPLICATION 1 DEFAULT" > /dev/null
echo "  hft_research: infinite retention (default)"

# Market data: 30 days
curl -s -XPOST "${INFLUX_URL}/query" --data-urlencode \
    "q=CREATE RETENTION POLICY rp_30d ON hft_marketdata DURATION 30d REPLICATION 1 DEFAULT" > /dev/null
echo "  hft_marketdata: 30d retention (default)"

# Write a test point
echo "[6/6] Writing test data..."
TIMESTAMP=$(date +%s)000000000
curl -s -XPOST "${INFLUX_URL}/write?db=hft_internal&precision=ns" \
    --data-binary "system_test,host=hft-micro-dydx,test=setup value=1 ${TIMESTAMP}" > /dev/null

# Verify the test point
RESULT=$(curl -s "${INFLUX_URL}/query?db=hft_internal&q=SELECT+*+FROM+system_test+ORDER+BY+time+DESC+LIMIT+1" | python3 -c "
import sys, json
data = json.load(sys.stdin)
try:
    vals = data['results'][0]['series'][0]['values'][0]
    print(f'  OK - Test point written and read back: time={vals[0]}, value={vals[2]}')
except:
    print('  WARN - Could not verify test point')
")
echo "$RESULT"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Databases created:"
echo "  hft_internal   - Strategy internal metrics (7d default, 30d optional)"
echo "  hft_external   - External reporting (90d)"
echo "  hft_research   - Backtest/research (infinite)"
echo "  hft_marketdata - Recorded market data (30d)"
echo ""
echo "Chronograf dashboard: $CHRONOGRAF_URL"
echo ""
echo "Connection string for YAML configs:"
echo "  internalInflux:"
echo "    address: \"$INFLUX_URL\""
echo "    database: \"hft_internal\""
echo "    batchSize: 5000"
echo "    saveInterval: 60s"
echo ""
echo "  externalInflux:"
echo "    address: \"$INFLUX_URL\""
echo "    database: \"hft_external\""
echo "    batchSize: 1000"
echo "    saveInterval: 60s"
