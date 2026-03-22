#!/bin/bash
# Test HTX API connectivity and basic endpoints
# No API key needed for public endpoints
set -euo pipefail

BASE="https://api.hbdm.com"
BASE_VN="https://api.hbdm.vn"

echo "=== HTX API Connectivity Test ==="
echo ""

# Test 1: Heartbeat
echo "[1/7] Testing heartbeat endpoint..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/heartbeat/")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    SWAP_HB=$(echo "$BODY" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('data',{}).get('linear_swap_heartbeat', 'N/A'))" 2>/dev/null || echo "parse error")
    echo "  OK (HTTP 200) - linear_swap_heartbeat=$SWAP_HB"
else
    echo "  FAIL (HTTP $HTTP_CODE)"
fi

# Test 2: Contract info
echo "[2/7] Testing contract info (swap_contract_info)..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/linear-swap-api/v1/swap_contract_info")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | python3 -c "import sys, json; print(len(json.load(sys.stdin).get('data',[])))" 2>/dev/null || echo "?")
    echo "  OK (HTTP 200) - $COUNT contracts available"
else
    echo "  FAIL (HTTP $HTTP_CODE)"
fi

# Test 3: BTC-USDT depth
echo "[3/7] Testing market depth (BTC-USDT step6)..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/linear-swap-ex/market/depth?contract_code=BTC-USDT&type=step6")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    BID=$(echo "$BODY" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['tick']['bids'][0][0])" 2>/dev/null || echo "?")
    ASK=$(echo "$BODY" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['tick']['asks'][0][0])" 2>/dev/null || echo "?")
    echo "  OK (HTTP 200) - BTC-USDT bid=$BID ask=$ASK"
else
    echo "  FAIL (HTTP $HTTP_CODE)"
fi

# Test 4: BBO ticker
echo "[4/7] Testing BBO ticker (ETH-USDT)..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/linear-swap-ex/market/bbo?contract_code=ETH-USDT")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    BID=$(echo "$BODY" | python3 -c "import sys, json; d=json.load(sys.stdin); t=d['ticks'][0]; print(f\"bid={t['bid'][0]} ask={t['ask'][0]}\")" 2>/dev/null || echo "?")
    echo "  OK (HTTP 200) - ETH-USDT $BID"
else
    echo "  FAIL (HTTP $HTTP_CODE)"
fi

# Test 5: Funding rates
echo "[5/7] Testing batch funding rates..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/linear-swap-api/v1/swap_batch_funding_rate")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | python3 -c "import sys, json; print(len(json.load(sys.stdin).get('data',[])))" 2>/dev/null || echo "?")
    echo "  OK (HTTP 200) - $COUNT funding rates"
else
    echo "  FAIL (HTTP $HTTP_CODE)"
fi

# Test 6: VN domain
echo "[6/7] Testing api.hbdm.vn domain..."
RESP=$(curl -s -w "\n%{http_code}" --connect-timeout 5 "$BASE_VN/heartbeat/" 2>/dev/null)
HTTP_CODE=$(echo "$RESP" | tail -1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "  OK (HTTP 200) - api.hbdm.vn reachable"
else
    echo "  WARN (HTTP $HTTP_CODE) - api.hbdm.vn may not be reachable from this network"
fi

# Test 7: Batch merged tickers (new v2 endpoint)
echo "[7/7] Testing v2 batch merged tickers..."
RESP=$(curl -s -w "\n%{http_code}" "$BASE/v2/linear-swap-ex/market/detail/batch_merged")
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -1)
if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(echo "$BODY" | python3 -c "import sys, json; print(len(json.load(sys.stdin).get('ticks',[])))" 2>/dev/null || echo "?")
    echo "  OK (HTTP 200) - $COUNT tickers in batch"
else
    echo "  FAIL/UNAVAILABLE (HTTP $HTTP_CODE)"
fi

echo ""
echo "=== Test Complete ==="
echo ""
echo "Next steps:"
echo "  - If all public endpoints pass, the existing connector domain works"
echo "  - To test private endpoints, you need an HTX API key"
echo "  - Create one at: https://www.htx.com/en-us/apikey/"
echo "  - Enable 'USDT-M Futures' permission"
