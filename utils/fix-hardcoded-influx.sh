#!/bin/bash
# Fix hardcoded localhost:8086 references in quantiles/research apps
# These apps have InfluxDB address hardcoded instead of via config
# Run from: ~/workspace/helix/hft-micro-dydx/
set -euo pipefail

INFLUX_URL="http://192.155.88.24:8086"
OLD_URL="http://localhost:8086"

echo "=== Fix Hardcoded InfluxDB Addresses ==="
echo "Replacing: $OLD_URL"
echo "With:      $INFLUX_URL"
echo ""

# Find all files with hardcoded localhost:8086
FILES=$(grep -rn "$OLD_URL" applications/ --include='*.go' -l 2>/dev/null || true)
COUNT=$(echo "$FILES" | grep -c '.' 2>/dev/null || echo "0")

if [ "$COUNT" = "0" ]; then
    echo "No files found with hardcoded $OLD_URL"
    exit 0
fi

echo "Found $COUNT files with hardcoded InfluxDB address:"
echo "$FILES" | while read f; do
    echo "  $f"
done
echo ""

# Create backup
BACKUP="applications.influx-fix.bak.$(date +%Y%m%d%H%M%S)"
echo "Creating backup list at: $BACKUP.txt"
echo "$FILES" > "$BACKUP.txt"

# Apply fix using python3 for safety
echo "Applying fix..."
echo "$FILES" | while read f; do
    python3 -c "
with open('$f', 'r') as fh:
    content = fh.read()
content = content.replace('$OLD_URL', '$INFLUX_URL')
with open('$f', 'w') as fh:
    fh.write(content)
print(f'  Fixed: $f')
"
done

echo ""
echo "=== Done ==="
echo "Fixed $COUNT files."
echo "File list saved to: $BACKUP.txt"
echo ""
echo "To verify: grep -rn 'localhost:8086' applications/ --include='*.go'"
echo "To revert: use the file list and sed to restore localhost:8086"
