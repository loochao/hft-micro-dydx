#!/bin/bash
# Check balances on all connected exchanges
# Usage: bash utils/check-balances.sh
# Run from: ~/workspace/helium/hft-micro-dydx/
set -euo pipefail

export PATH=$PATH:$HOME/go/bin
PROXY="socks5://127.0.0.1:1083"

# Source credentials from env file if exists
if [ -f "$HOME/.crypto-keys" ]; then
    source "$HOME/.crypto-keys"
fi

# Build if needed
if [ ! -f bin/check-balances ]; then
    echo "Building check-balances..."
    go build -o bin/check-balances ./tools/check-balances/
fi

echo "Proxy: $PROXY"
echo "Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo ""

bin/check-balances \
    -proxy "$PROXY" \
    -bn-key "${BN_API_KEY:-}" \
    -bn-secret "${BN_API_SECRET:-}" \
    -dydx-addr "${DYDX_ADDRESS:-}"
