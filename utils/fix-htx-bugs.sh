#!/bin/bash
# HTX Connector Bug Fix Script
# Applies all critical bug fixes to huobi-usdtfuture/
# Run from: ~/workspace/helix/hft-micro-dydx/
set -euo pipefail

DIR="huobi-usdtfuture"
BACKUP_DIR="huobi-usdtfuture.bak.$(date +%Y%m%d%H%M%S)"

echo "=== HTX Connector Bug Fix Script ==="
echo "Backing up $DIR to $BACKUP_DIR..."
cp -r "$DIR" "$BACKUP_DIR"

# ============================================================
# BUG-1: TickerWS.dataHandleLoop never calls ParseTicker
# File: hbuf-ticker-ws.go
# The line "ticker = pool[index]" is followed by checking err
# but ParseTicker(msg, ticker) is never called
# ============================================================
echo "[BUG-1] Fixing TickerWS.dataHandleLoop - adding ParseTicker call..."

python3 << 'PYEOF'
with open("huobi-usdtfuture/hbuf-ticker-ws.go", "r") as f:
    content = f.read()

old = """			ticker = pool[index]
			if err != nil && time.Now().Sub(logSilentTime) > 0 {"""

new = """			ticker = pool[index]
			err = ParseTicker(msg, ticker)
			if err != nil && time.Now().Sub(logSilentTime) > 0 {"""

content = content.replace(old, new)

with open("huobi-usdtfuture/hbuf-ticker-ws.go", "w") as f:
    f.write(content)
PYEOF

echo "[BUG-1] Fixed."

# ============================================================
# BUG-2: Context leaks - missing cancel() calls
# Files: hbuf.go (multiple polling loops)
# ============================================================
echo "[BUG-2] Fixing context leaks in polling loops..."

python3 << 'PYEOF'
with open("huobi-usdtfuture/hbuf.go", "r") as f:
    content = f.read()

# Fix all "subCtx, _" patterns
content = content.replace(
    "subCtx, _ := context.WithTimeout(ctx, time.Minute)",
    "subCtx, subCancel := context.WithTimeout(ctx, time.Minute)"
)
content = content.replace(
    "subCtx, _ := context.WithTimeout(ctx, time.Second*3)",
    "subCtx, subCancel := context.WithTimeout(ctx, time.Second*3)"
)

with open("huobi-usdtfuture/hbuf.go", "w") as f:
    f.write(content)
PYEOF

echo "[BUG-2] Fixed (variable rename). Note: still need to add subCancel() calls after each use."

# ============================================================
# BUG-3: WSOrder.GetID uses %s for int64
# File: hbuf-types.go
# ============================================================
echo "[BUG-3] Fixing WSOrder.GetID format specifier..."

python3 << 'PYEOF'
with open("huobi-usdtfuture/hbuf-types.go", "r") as f:
    content = f.read()

content = content.replace(
    'return fmt.Sprintf("%s", wsOrder.OrderID)',
    'return fmt.Sprintf("%d", wsOrder.OrderID)'
)

with open("huobi-usdtfuture/hbuf-types.go", "w") as f:
    f.write(content)
PYEOF

echo "[BUG-3] Fixed."

# ============================================================
# BUG-4: WSOrder.GetReduceOnly always returns false
# File: hbuf-types.go
# ============================================================
echo "[BUG-4] Fixing WSOrder.GetReduceOnly..."

python3 << 'PYEOF'
with open("huobi-usdtfuture/hbuf-types.go", "r") as f:
    content = f.read()

old = """func (wsOrder *WSOrder) GetReduceOnly() bool {
	return false
}"""

new = """func (wsOrder *WSOrder) GetReduceOnly() bool {
	return wsOrder.Offset == OrderOffsetClose
}"""

content = content.replace(old, new)

with open("huobi-usdtfuture/hbuf-types.go", "w") as f:
    f.write(content)
PYEOF

echo "[BUG-4] Fixed."

# ============================================================
# Summary
# ============================================================
echo ""
echo "=== Summary ==="
echo "Fixed bugs:"
echo "  1. ParseTicker call added to TickerWS.dataHandleLoop"
echo "  3. WSOrder.GetID format: %s -> %d"
echo "  4. WSOrder.GetReduceOnly: checks Offset == close"
echo ""
echo "Partially fixed:"
echo "  2. Context leak variables renamed, need manual subCancel() placement"
echo ""
echo "Remaining manual work:"
echo "  5. Replace static PriceTicks/ContractSizes with API fetch"
echo "  6. Convert recursive reconnect() to iterative loop"
echo "  7. Move timer.Reset inside select cases"
echo "  8. Consider JSON parser fallback"
echo ""
echo "Backup at: $BACKUP_DIR"
echo "Review changes with: diff -r $BACKUP_DIR $DIR"
