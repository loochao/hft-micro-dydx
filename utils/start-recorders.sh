#!/bin/bash
# Start all market data recorders for hft-micro-dydx
# Run from: ~/workspace/helium/hft-micro-dydx/
# All recorders pull PUBLIC market data (no API keys needed)
set -euo pipefail

BIN="$(pwd)/bin"
DATA_ROOT="$HOME/MarketData"
LOG_DIR="$HOME/logs/recorders"
PROXY=""  # Set to e.g. "socks5://127.0.0.1:1080" if needed

mkdir -p "$LOG_DIR"

# Recorders and their data paths
declare -A RECORDERS=(
    ["recorder-bnuf"]="$DATA_ROOT/bnuf"
    ["recorder-bnus"]="$DATA_ROOT/bnus"
    ["recorder-bnuf-ohlcv"]="$DATA_ROOT/bnuf-ohlcv"
    ["recorder-bnus-ohlcv"]="$DATA_ROOT/bnus-ohlcv"
    ["recorder-bnus-bnuf-depth5-and-ticker"]="$DATA_ROOT/bnus-bnuf-depth5-and-ticker"
    ["recorder-kcuf"]="$DATA_ROOT/kcuf"
    ["recorder-kcus"]="$DATA_ROOT/kcus"
    ["recorder-kcuf-bnuf-depth5-and-ticker"]="$DATA_ROOT/kcuf-bnuf-depth5-and-ticker"
    ["recorder-kcus-bnuf-depth5-and-ticker"]="$DATA_ROOT/kcus-bnuf-depth5-and-ticker"
    ["recorder-okuf"]="$DATA_ROOT/okuf"
    ["recorder-okus"]="$DATA_ROOT/okus"
    ["recorder-hbuf-bnuf-depth-and-ticker"]="$DATA_ROOT/hbuf-bnuf-depth-and-ticker"
    ["recorder-cbus-bnuf-ticker"]="$DATA_ROOT/cbus-bnuf-ticker"
    ["recorder-bnspot-usd"]="$DATA_ROOT/bnspot-usd"
)

start_recorder() {
    local name=$1
    local datapath=$2
    local logfile="$LOG_DIR/${name}.log"

    mkdir -p "$datapath"

    # Check if already running
    if pgrep -f "$BIN/$name" > /dev/null 2>&1; then
        echo "  SKIP $name (already running, pid=$(pgrep -f "$BIN/$name"))"
        return
    fi

    local args="-path $datapath"
    if [ -n "$PROXY" ]; then
        args="$args -proxy $PROXY"
    fi

    nohup "$BIN/$name" $args > "$logfile" 2>&1 &
    local pid=$!
    echo "  START $name (pid=$pid, data=$datapath, log=$logfile)"
}

stop_all() {
    echo "Stopping all recorders..."
    for name in "${!RECORDERS[@]}"; do
        if pgrep -f "$BIN/$name" > /dev/null 2>&1; then
            pkill -f "$BIN/$name" && echo "  STOP $name" || true
        fi
    done
}

status_all() {
    echo "=== Recorder Status ==="
    for name in $(echo "${!RECORDERS[@]}" | tr ' ' '\n' | sort); do
        local datapath=${RECORDERS[$name]}
        if pgrep -f "$BIN/$name" > /dev/null 2>&1; then
            local pid=$(pgrep -f "$BIN/$name")
            local filecount=$(find "$datapath" -name '*.gz' -o -name '*.jl' -o -name '*.csv' 2>/dev/null | wc -l)
            echo "  RUNNING  $name (pid=$pid, files=$filecount)"
        else
            echo "  STOPPED  $name"
        fi
    done
}

case "${1:-start}" in
    start)
        echo "=== Starting Recorders ==="
        echo "Data root: $DATA_ROOT"
        echo "Log dir:   $LOG_DIR"
        echo ""
        for name in $(echo "${!RECORDERS[@]}" | tr ' ' '\n' | sort); do
            start_recorder "$name" "${RECORDERS[$name]}"
        done
        echo ""
        echo "Done. Check logs with: tail -f $LOG_DIR/recorder-*.log"
        ;;
    stop)
        stop_all
        ;;
    status)
        status_all
        ;;
    restart)
        stop_all
        sleep 2
        echo ""
        $0 start
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac
