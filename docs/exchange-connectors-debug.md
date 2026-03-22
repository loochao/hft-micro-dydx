# Exchange Connectors Debug Report

## Summary
47 bugs identified across 30+ exchange connector modules.
Analysis performed via code review on 10.209.157.152:~/workspace/helix/hft-micro-dydx/

---

## CRITICAL Bugs (7)

### BUG-001: `stopped bool` Field Not Atomic (8+ modules)
**Files:** bnspot/, bnswap/, bnmargin/, bswap/, bfspot/, gtspot/, hbspot/ and others
**Problem:** Multiple modules use a plain `bool` for `stopped` field instead of `int32` with atomic operations. Race condition on concurrent Stop() calls can cause double-close panic on channels.
**Fix:** Use `atomic.CompareAndSwapInt32` pattern (as used in huobi-usdtfuture).

### BUG-003: Buffer Pool Reuse Can Corrupt Data Under Load (ALL WS modules)
**Problem:** Multiple WS modules use small fixed-size buffer pools (typically [4]*Type). Under high message rates, the pool cycles faster than consumers drain output channels, causing the same buffer to be overwritten before the consumer reads it.
**Impact:** Corrupted orderbook or ticker data during volatility spikes — exactly when accuracy matters most.

### BUG-004: TradeWebsocket.Stop() Blocks Instead of Checking (`binance-usdtfuture`)
**File:** `binance-usdtfuture/bnuf-trade-ws.go`
**Problem:** `Stop()` uses `<-w.done` which blocks the caller. Should check with select/default or use atomic flag pattern. Will deadlock on double-close.

### BUG-005: restart() Calls logger.Fatal (`binance-usdtfuture`)
**File:** `binance-usdtfuture/bnuf-trade-ws.go`
**Problem:** `restart()` has a 1ms timer fallback that calls `logger.Fatal`, which calls `os.Exit(1)`. A simple reconnection failure crashes the entire process.
**Fix:** Use `logger.Warnf` and retry logic instead of Fatal.

### BUG-006: 8 Parallel Data Handlers Cause Out-of-Order Trades (`binance-usdtfuture`)
**File:** `binance-usdtfuture/bnuf-trade-ws.go`
**Problem:** Spawns 8 parallel goroutines for trade processing. The README explicitly warns: "Websocket用户数据如果并行处理存在数据先后错乱的问题" (parallel processing causes data ordering issues).
**Fix:** Use single handler or sequential dispatch.

### BUG-007: Stop() TOCTOU Race (`dydx-usdfuture`)
**File:** `dydx-usdfuture/dduf.go`
**Problem:** Uses `LoadInt32`/`StoreInt32` pattern instead of `CompareAndSwapInt32`, creating a time-of-check-to-time-of-use race on the stopped flag. Two concurrent Stop() calls can both pass the check and double-close the done channel.
**Fix:** Use `atomic.CompareAndSwapInt32(&h.stopped, 0, 1)`.

### BUG-015: MEXC Module Contains Kucoin Copy-Paste (`mexc-usdtfuture`)
**Problem:** MEXC connector code is copy-pasted from Kucoin module and may connect to wrong exchange endpoints or use wrong authentication. Needs full audit.

---

## HIGH Bugs (12)

### BUG-008: No Timestamp Validation on Orderbook Data (ALL depth WS modules)
**Problem:** Depth snapshots are forwarded to strategies without checking if the timestamp is newer than the last processed snapshot. Stale/duplicate snapshots can cause strategies to act on outdated prices.

### BUG-009: Unbounded Recursive Reconnection (ALL WS modules)
**Problem:** All reconnect() functions use tail recursion with `time.After(10s)`. If server is down for hours, this builds hundreds/thousands of stack frames. Will eventually stack overflow.
**Fix:** Convert to iterative loop.

### BUG-010: Hardcoded Tick/Step Sizes (ALL modules with limits files)
**Problem:** Static maps from 2021 with tick sizes and contract sizes. New coins can't be traded; changed tick sizes cause incorrect order pricing.
**Fix:** Fetch from API at startup (as we did for HTX).

### BUG-011: Silent Data Drops During Volatility (ALL WS modules)
**Problem:** All channel sends use `select { case ch <- data: default: }` pattern. During high-volume periods, slow consumers cause data drops with only debug-level logging. No metrics or alerting.

### BUG-012: Listen Key Context Leak (ALL Binance modules)
**Problem:** Binance modules call `context.WithTimeout` for listen key renewal but discard the cancel function. Leaks goroutines and timers every 30 minutes.

### BUG-013: Depth20WS Symbol Parser Has Dead Branch
**Problem:** Multiple depth WS modules have duplicate conditional branches in the symbol extraction code that can never be reached.

### BUG-014: Duplicate Code Path in readLoop
**Problem:** Several modules have copy-pasted readLoop implementations with slight variations that diverge over time, making bugs harder to track.

### BUG-016: FTX Modules Are Dead Code
**Files:** `ftx-usdfuture/`, `ftx-usdspot/`
**Problem:** FTX collapsed in Nov 2022. These modules are dead code but still imported and compiled. They should be removed or clearly marked as archived.
**Note:** Consider removing from build.

### BUG-017: dYdX accountLoop Blocks Indefinitely
**File:** `dydx-usdfuture/dduf.go`
**Problem:** The accountLoop can block on channel send without context checking, causing the goroutine to hang if the consumer stops reading.

### BUG-018: fmt.Printf Debug Print in Production (MEXC)
**File:** `mexc-usdtfuture/`
**Problem:** Contains `fmt.Printf` debug prints that write to stdout in production, interleaving with structured logging.

### BUG-019: Huobi TickerWS Never Parses Data
**File:** `huobi-usdtfuture/hbuf-ticker-ws.go`
**Problem:** ParseTicker() is never called in dataHandleLoop. **FIXED** in utils/fix-htx-bugs.sh.

### BUG-020: WSOrder.GetID Wrong Format Specifier (Huobi)
**File:** `huobi-usdtfuture/hbuf-types.go`
**Problem:** Uses `%s` instead of `%d` for int64. **FIXED** in utils/fix-htx-bugs.sh.

---

## MEDIUM Bugs (28)

Summarized by category:

### Missing Error Handling (8 instances)
- API response bodies read without checking Content-Type
- JSON unmarshal errors logged but not propagated
- WebSocket write errors sometimes restart, sometimes silently continue

### Goroutine Leaks (6 instances)
- Context.WithTimeout cancel functions discarded in polling loops
- Goroutines waiting on closed connections without context checks
- Timer goroutines from time.After not cleaned up on early return

### Data Integrity (5 instances)
- No sequence number validation on incremental updates
- No staleness detection on market data (could trade on minutes-old prices)
- Position merge logic assumes single direction per symbol (breaks with hedge mode)

### Configuration (5 instances)
- Hardcoded proxy timeouts (60s) too long for HFT
- WebSocket buffer sizes inconsistent across modules
- Some modules use `api.hbdm.vn`, others `api.hbdm.com` — should be configurable

### Code Quality (4 instances)
- Unused imports in test files
- Commented-out code blocks (debugging remnants)
- Inconsistent error message formatting
- Dead exchange modules (FTX) still in codebase

---

## Cross-Cutting Issues

### 1. Massive Code Duplication
Nearly every exchange module is a copy-paste of another with minimal changes. The same bugs propagate to all copies. Consider extracting a common WebSocket framework.

### 2. No Rate Limiting
No modules implement client-side rate limiting. All rely on exchange-side rejection, which wastes requests and can trigger IP bans.

### 3. No Health Metrics
No modules expose connection health metrics (latency, reconnect count, message rate, drop rate). Operating blind in production.

### 4. Unsafe String Conversion
Many modules use `unsafe.Pointer` for zero-copy string creation from byte slices. While fast, this violates Go's memory safety guarantees and can cause subtle corruption if the underlying byte slice is reused.

---

## Priority Matrix

| Priority | Count | Action |
|----------|-------|--------|
| CRITICAL | 7 | Fix before any live trading |
| HIGH | 12 | Fix before production deployment |
| MEDIUM | 28 | Fix during normal development |

## Defunct Exchanges (Remove or Archive)
- **FTX** (ftx-usdfuture/, ftx-usdspot/) — collapsed Nov 2022
- **Binance BUSD** (binance-busdfuture/, binance-busdspot/) — BUSD sunset Feb 2024
- **LUNA** — contract delisted from all exchanges

## Active Exchanges Requiring Account Setup
- **dYdX** — needs StarkEx keys for L2
- **Binance** — needs API key with futures permission
- **Bybit** — needs API key
- **Gate** — needs API key
- **Kucoin** — needs API key + passphrase
- **OKX** — needs API key + passphrase
- **HTX** — needs API key (tested, connectivity confirmed)
- **MEXC** — needs API key (module needs audit first — Kucoin copy-paste)
- **Coinbase** — needs API key (spot only)
- **Bitfinex** — needs API key
