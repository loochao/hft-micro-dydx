# HFT Micro DyDx - Debug Report (2026-03-21)

## CRITICAL BUGS

### BUG-001: Nil Pointer Dereference on strat.spread.EventTime
**Files:** 10+ modules (usd-ll-mt, usd-ll-mt-q, usd-tk-mt, usd-md-mt, usd-smt, usd-swap-mt, etc.)
**Location:** order.go updateXOrder() - inside if-block after spread==nil check
**Issue:** spread==nil short-circuits into if-block, but inside the block strat.spread.EventTime is accessed without nil guard. PANIC on startup before first spread calculation.
**Fix:** Add nil guard: if strat.spread != nil && time.Now().Sub(strat.spread.EventTime)...

### BUG-002: coin-deli-tt Config Overwrites Wrong Field (LogInterval)
**File:** coin-deli-tt/config.go:70-72
**Issue:** config.LoopInterval = time.Minute (should be config.LogInterval). Loop runs 60x slower, logs flood.

### BUG-003: coin-deli-tt Config Overwrites Wrong Field (ReportCount)
**File:** coin-deli-tt/config.go:103-105
**Issue:** config.RestartSilent = 1000 (should be config.ReportCount). RestartSilent becomes 0, ReportCount stays 0.

## HIGH-SEVERITY BUGS

### BUG-004: yAbsValue Reports xAbsValue (17 modules)
**Pattern:** fields["yAbsValue"] = st.xAbsValue (should be st.yAbsValue)
**Files:** usd-ll-mt:71, usd-ll-mt-q:72, usd-ll-mt-q-d:72, usd-ll-mt-q2:72, usd-ll-md-mt:56, usd-ll-tt:59, usd-long-short:55, usd-md-mt:60, usd-md-mt-q:72, usd-smt:72, usd-swap-mt:60, usd-tk-mt:64, usd-tk-mt-q:64, usd-tk-tt:65, usd-tt:61 (all influx.go)

### BUG-005: yDepthFilterRatio/yTickerFilterRatio Reports X Value (20+ modules)
**Pattern:** fields["yDepthFilterRatio"] = st.spreadReport.XTickerFilterRatio (should be Y)
**Files:** usd-ll-mt:123, usd-ll-mt-q:127, usd-ll-mt-q-d:127, usd-ll-mt-q2:126, usd-ll-md-mt:102, usd-ll-tt:107, usd-md-mt:105, usd-md-mt-q:127, usd-smt:126, usd-swap-mt:96, usd-tk-mt:110, usd-tk-mt-q:113, usd-tk-tt:113, usd-tk-tt-opt-q:122, usd-tk-tt-q:130, usd-tk-xt-basic:110, usd-tk-xt-q:109, usd-tk-yr-q:109, usd-tk-yt-q:109, usd-tt:107

### BUG-006: Missing Format Argument in Debugf
**File:** usd-tt/spread.go:104
**Issue:** logger.Debugf("%s strat.xDepth == strat.xNextDepth same pointer") - missing xSymbol arg

### BUG-007: Blocking Select on Order Channel (All modules)
**Pattern:** select { case ch <- order: } with no default/context case. Goroutine hangs if channel full.

### BUG-008: Asymmetric Cancel Error Handling (usd-long-short)
**File:** usd-long-short/loop.go - X cancel errors skip silent time, Y cancel errors set it.

## MEDIUM ISSUES

- ISSUE-001: Division by zero in turnover calc (xTradeVolume/xBalance when balance=0)
- ISSUE-002: No max position size limits
- ISSUE-003: Data race on XYStrategy state via saveCh pointer
- ISSUE-004: xOpenOrder not cleared on channel-full error paths
- ISSUE-005: Y-leg market orders have no slippage protection
- ISSUE-006: Timer.Reset without drain (Go doc violation)
- ISSUE-007: handleXPosition updates staleness timer for old positions
- ISSUE-008: usd-swap-mt missing funding rate guard before entry
- ISSUE-009: No atomic write for quantile file persistence
- ISSUE-010: Inconsistent logSilentTime check patterns
- ISSUE-011: usd-long-short external save not throttled
- ISSUE-012: Stale spread data used for order validation in isXOpenOrderOk

## LOW / CODE QUALITY

- L01: Deprecated io/ioutil in all 32 main.go files
- L02: Massive code duplication (36 copy-pasted dirs, root cause of BUG-004/005)
- L03: Hardcoded build timestamps in init.go
- L04: No graceful shutdown (5s sleep, no WaitGroup)
- L05: proxy/main.go is bare SOCKS5 with no auth
- L06: dydx-vc hardcodes Epoch "4"
- L07: Large commented-out code blocks
- L08: Test files contain experimental code, not actual tests

## ARCHITECTURE

Strategy families: usd-ll-mt* (limit maker, market taker), usd-tk-mt* (ticker maker-taker), usd-tk-tt* (ticker taker-taker), usd-tt (depth taker-taker), usd-smt (smart maker-taker), usd-swap-mt (dual limit), usd-xt-ma (cross-exchange MA), usd-long-short/close/rebalance (portfolio ops), coin-deli-tt (delivery futures).

Pattern: YAML config -> context+exchanges -> per-symbol goroutines with select loop -> depth walking -> spread calculation -> signal thresholds -> order placement -> Y-leg hedging.

Risk present: ReduceOnly, EnterTarget cap, funding rate gates, DryRun, unhedge detection.
Risk absent: No stop-loss, no max drawdown, no portfolio leverage cap, no circuit breaker.

## RECOMMENDATIONS

1. IMMEDIATE: Fix BUG-001 nil deref in 10+ order.go files
2. IMMEDIATE: Fix BUG-002/003 in coin-deli-tt/config.go
3. HIGH: Fix BUG-004/005 copy-paste bugs in 20+ influx.go files
4. HIGH: Add default/timeout to blocking select on order channels
5. MEDIUM: Refactor into common package to eliminate duplication
6. MEDIUM: Add max position and portfolio-level risk checks
7. LOW: Migrate ioutil, add graceful shutdown, clean dead code
