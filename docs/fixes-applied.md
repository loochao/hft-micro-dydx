# HFT Micro DyDx - Bug Fixes Applied (Session 2)

Date: 2026-03-22

## Critical Fixes

### Fix 1: StarkEx CalculateHash() Mutates Inputs
**File:** `starkex/skex-types.go`
**Issue:** CalculateHash() operated directly on the StarkwareOrder's big.Int fields via pointer aliasing. Lsh/Add operations mutated the original QuantumsAmountSell, QuantumsAmountBuy, QuantumsAmountFee, Nonce, PositionId, and ExpirationEpochHours fields, corrupting the order for any subsequent use (e.g., retry, logging).
**Fix:** Created defensive copies with `new(big.Int).Set(original)` before any arithmetic. All 8 big.Int fields that feed into part1/part2 computation are now copied.

### Fix 2: TimedEMA Never Updates LastTime
**File:** `stream-stats/timed-ema.go`
**Issue:** The Insert() method computed EMA using `timestamp.Sub(tm.LastTime)` but never updated `tm.LastTime`. After first initialization (zero-value), every subsequent call computed diff from epoch, making the adaptive period calculation meaningless.
**Fix:** Added `tm.LastTime = timestamp` after the EMA value update inside the `if diff > 0` block.

### Fix 3: TimedCorrelation Returns Covariance Not Correlation
**File:** `stream-stats/timed-correlation.go`
**Issue:** Insert() computed E[XY] - E[X]E[Y] (covariance) but the method and struct are named "Correlation". The Correlation() method returned raw covariance, which is not bounded to [-1, 1] and has different scale properties.
**Fix:** Added normalization by computing stdX and stdY from the stored samples, then dividing covariance by (stdX * stdY). Added `math` import. Returns covariance unchanged when either std is zero (degenerate case).

### Fix 4: Nil Pointer Dereference in order.go
**Files:** 8 application modules fixed:
- `applications/usd-ll-md-mt/order.go`
- `applications/usd-ll-mt-q-d/order.go`
- `applications/usd-ll-mt-q/order.go`
- `applications/usd-ll-mt/order.go`
- `applications/usd-md-mt-q/order.go`
- `applications/usd-md-mt/order.go`
- `applications/usd-swap-mt/order.go`
- `applications/usd-tk-mt/order.go`

**Issue:** The outer if-condition includes `strat.spread == nil` in an OR chain. When spread is nil, the if-block is entered. Inside the block, code like `if time.Now().Sub(strat.spread.EventTime) > strat.config.SpreadTimeToCancel` dereferences nil spread, causing a panic.
**Fix:** Added `strat.spread != nil &&` guard before each inner `strat.spread.EventTime` access inside the if-block.

**Note:** 11 other files were analyzed and confirmed safe - they either have no inner spread.EventTime dereference, or access EventTime only after the nil-guard block (where spread is guaranteed non-nil due to Go's || short-circuit evaluation).

### Fix 5: coin-deli-tt Config Wrong Field Assignments
**File:** `applications/coin-deli-tt/config.go`
**Issue 1:** Line in SetDefaultIfNotSet(): `if config.LogInterval == 0 { config.LoopInterval = time.Minute }` -- assigns to LoopInterval instead of LogInterval, overwriting an already-set LoopInterval.
**Fix 1:** Changed to `config.LogInterval = time.Minute`.

**Issue 2:** `if config.ReportCount == 0 { config.RestartSilent = 1000 }` -- assigns integer 1000 to RestartSilent (a time.Duration) instead of ReportCount, corrupting RestartSilent (1000ns = 1us) and leaving ReportCount at 0.
**Fix 2:** Changed to `config.ReportCount = 1000`.

## High-Severity Fixes

### Fix 6: yAbsValue = xAbsValue Copy-Paste Bug
**Files:** 15 influx.go files fixed:
usd-ll-md-mt, usd-ll-mt-q-d, usd-ll-mt-q, usd-ll-mt-q2, usd-ll-mt, usd-ll-tt, usd-long-short, usd-md-mt-q, usd-md-mt, usd-smt, usd-swap-mt, usd-tk-mt-q, usd-tk-mt, usd-tk-tt, usd-tt

**Issue:** `fields["yAbsValue"] = st.xAbsValue` reports the X leg's absolute value as the Y leg's, making the Y position size metric in InfluxDB always mirror X.
**Fix:** Changed to `fields["yAbsValue"] = st.yAbsValue`.

### Fix 7: yTickerFilterRatio = XTickerFilterRatio Copy-Paste Bug
**Files:** 10 influx.go files fixed:
merge-accounts, usd-tk-mt-q, usd-tk-mt, usd-tk-tt-opt-q, usd-tk-tt-q, usd-tk-tt, usd-tk-xt-basic, usd-tk-xt-q, usd-tk-yr-q, usd-tk-yt-q

**Issue:** `fields["yTickerFilterRatio"] = st.spreadReport.XTickerFilterRatio` reports X ticker filter ratio as Y's.
**Fix:** Changed to `fields["yTickerFilterRatio"] = st.spreadReport.YTickerFilterRatio`.

### Fix 8: Stop() TOCTOU Race
**File:** `dydx-usdfuture/dduf.go`
**Issue:** Load-then-Store pattern on dd.stopped allows two goroutines to both see 0, both store 1, and both call close(dd.done), causing a panic on double-close.
**Fix:** Replaced with `atomic.CompareAndSwapInt32(&dd.stopped, 0, 1)` which is atomic and guarantees exactly one goroutine succeeds.

### Fix 9: Recursive Reconnect in WS Modules
**Files:**
- `dydx-usdfuture/dduf-depth-ticker-ws.go` (TickerWS)
- `dydx-usdfuture/dduf-depth-ws.go` (DepthWS)
- `dydx-usdfuture/dduf-user-ws.go` (UserWebsocket)

**Issue:** reconnect() calls itself recursively on dial failure. Extended network outages cause unbounded stack growth (each retry adds a stack frame with dialer, headers, etc.), eventually leading to stack overflow.
**Fix:** Wrapped function body in `for { ... }` loop and replaced recursive `return w.reconnect(ctx, wsUrl, proxy, counter+1)` with `counter++; continue`.

### Fix 10: Context Leaks (Discarded Cancel Functions)
**Files:**
- `dydx-usdfuture/dduf.go` (4 occurrences)
- `kucoin-coinfuture/kccf-utils.go`
- `kucoin-coinfuture/kccf.go`
- `kucoin-usdtfuture/kcuf-raw-funding-rate.go`
- `kucoin-usdtfuture/kcuf.go`
- `bybit-usdtfuture/bbuf-raw-funding-rate.go`
- `bybit-usdtfuture/bbuf.go`

**Issue:** `subCtx, _ := context.WithTimeout(ctx, time.Minute)` discards the cancel function. The context's timer goroutine and resources leak until the timeout expires (1 minute each). In high-frequency loops, this accumulates thousands of leaked goroutines.
**Fix:** Captured cancel function as `subCancel` and called `subCancel()` immediately after the API call completes, ensuring prompt resource cleanup.
