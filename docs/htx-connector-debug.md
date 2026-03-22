# HTX (Huobi) USDT-Future Connector — Debug Report & Migration Guide

## Executive Summary

The existing `huobi-usdtfuture/` connector targets the old Huobi USDT-M linear swap API
on `api.hbdm.vn`. Good news: **the HTX API is backward-compatible** — domain `api.hbdm.com`
(and `.vn` variant) still works, all endpoint paths are unchanged, same auth method.

However, the connector has **critical bugs** that must be fixed, and should be updated to
support new HTX API features (unified account, v3 endpoints, dynamic contract info).

---

## Bug Report

### BUG-1: TickerWS.dataHandleLoop Never Parses Ticker (CRITICAL)

**File:** `hbuf-ticker-ws.go` lines ~210-240

```go
case msg := <-inputCh:
    index ++
    if index == 4 { index = 0 }
    ticker = pool[index]
    if err != nil && time.Now().Sub(logSilentTime) > 0 {
        // logs error but NEVER actually calls ParseTicker!
```

**Problem:** `ParseTicker(msg, ticker)` is never called. The `err` variable is only checked
but never assigned from parsing. The ticker sent to `outputCh` contains zeroed bid/ask data.

**Impact:** All BBO ticker data is garbage zeros. Any strategy relying on TickerWS gets no
price information. The `HuobiUsdtFutureWithMergedTicker` variant partially masks this because
it also uses `Depth20TickerWS` (which does work), but the TickerWS half is dead weight.

**Fix:** Add the missing call:
```go
case msg := <-inputCh:
    index++
    if index == 4 { index = 0 }
    ticker = pool[index]
    err = ParseTicker(msg, ticker)  // <-- THIS LINE IS MISSING
    if err != nil && time.Now().Sub(logSilentTime) > 0 {
```

### BUG-2: Context Leak in StreamFundingRate

**File:** `hbuf.go` ~StreamFundingRate

```go
subCtx, _ := context.WithTimeout(ctx, time.Minute)
```

**Problem:** The cancel function from `context.WithTimeout` is discarded (`_`). This leaks
a goroutine and timer every polling cycle until the parent context is cancelled.

**Impact:** Memory and goroutine leak proportional to uptime. With 30s pull interval,
thats ~2880 leaked contexts per day.

**Fix:** Capture and defer cancel:
```go
subCtx, cancel := context.WithTimeout(ctx, time.Minute)
// use subCtx
cancel()
```

Same bug exists in `systemStatusLoop`, `accountLoop`, `positionsLoop`.

### BUG-3: WSOrder.GetID Uses %s Format for int64

**File:** `hbuf-types.go`

```go
func (wsOrder *WSOrder) GetID() string {
    return fmt.Sprintf("%s", wsOrder.OrderID)  // OrderID is int64!
}
```

**Problem:** `%s` on an int64 produces garbage like `%!s(int64=832250359040368640)`.

**Fix:**
```go
return fmt.Sprintf("%d", wsOrder.OrderID)
```

### BUG-4: WSOrder.GetReduceOnly Always Returns False

**File:** `hbuf-types.go`

```go
func (wsOrder *WSOrder) GetReduceOnly() bool {
    return false
}
```

**Problem:** Should check `wsOrder.Offset == "close"` to determine reduce-only status.
Strategies that check `GetReduceOnly()` to track position-reducing fills will get wrong info.

### BUG-5: Static PriceTicks/ContractSizes Are Stale

**File:** `hbuf-limits.go`

**Problem:** Hardcoded tick sizes and contract sizes from mid-2021. HTX has since:
- Added 100+ new contracts (ARB, APT, SUI, OP, WLD, PEPE, etc.)
- Changed tick sizes for some existing contracts
- Delisted some contracts (LUNA, FTT, etc.)

**Impact:** Setup() will fail for any symbol not in the map. New coins cant be traded.

**Fix:** Fetch dynamically from `/linear-swap-api/v1/swap_contract_info` at startup:
```go
func (h *HuobiUsdtFuture) Setup(ctx context.Context, settings common.ExchangeSettings) error {
    // ... existing setup ...
    contracts, err := h.api.GetContracts(ctx)
    if err != nil {
        return fmt.Errorf("GetContracts: %w", err)
    }
    for _, c := range contracts {
        PriceTicks[c.Symbol] = c.PriceTick
        ContractSizes[c.Symbol] = c.ContractSize
    }
    // ... rest of setup ...
}
```

### BUG-6: Recursive Reconnect Can Stack Overflow

**Files:** `hbuf-ticker-ws.go`, `hbuf-user-ws.go`, `hbuf-depth20-ws.go`

```go
func (w *TickerWS) reconnect(..., counter int64) (*websocket.Conn, error) {
    // ...
    case <-time.After(time.Second * 10):
        return w.reconnect(ctx, wsUrl, proxy, counter+1)  // recursive!
}
```

**Problem:** If the server is down for hours, this builds up deep recursive call stacks.
With 10s retry, thats 360 stack frames per hour.

**Fix:** Use iterative loop instead of recursion.

### BUG-7: accountLoop Timer Reset Outside Select

**File:** `hbuf.go` ~accountLoop

```go
for {
    select {
    case <-ctx.Done():
        return
    case <-timer.C:
        // ... do work ...
    }
    timer.Reset(...)  // OUTSIDE the select - runs even on ctx.Done()!
}
```

The `timer.Reset` runs after every select case, including `ctx.Done()`.
The `return` prevents issues for `ctx.Done()`, but the pattern is fragile.

### BUG-8: ParseDepth20 Hardcoded Offset Assumptions

**File:** `hbuf-utils.go`

The parser uses hardcoded byte offsets (`bytes[12]`, `offset += 48`, etc.) that assume
a specific JSON field ordering and key lengths. If HTX ever adds a field or changes
ordering, the parser silently produces garbage.

**Risk:** Medium. JSON field ordering is technically unordered, though in practice HTX
has been stable. The manual parser is ~10x faster than json.Unmarshal, which matters for HFT.

---

## HTX API Changes Since 2021

### Still Working (No Changes Needed)
- REST base: `api.hbdm.vn` / `api.hbdm.com` — both active
- All `/linear-swap-api/v1/` endpoints — same paths, same params
- WebSocket URLs: `wss://api.hbdm.vn/linear-swap-ws` and `/linear-swap-notification`
- Auth: HMAC-SHA256, SignatureVersion=2 — unchanged
- Ping/pong format — unchanged
- Gzip compression on market WS — unchanged

### New Features Available
1. **Unified Account Mode** — merges isolated + cross margin
   - `/linear-swap-api/v3/swap_unified_account_type`
   - `/linear-swap-api/v3/swap_switch_account_type`
   - WS topic: `accounts_unify.USDT`

2. **V3 History Endpoints** (better pagination)
   - `/linear-swap-api/v3/swap_hisorders`
   - `/linear-swap-api/v3/swap_cross_hisorders`
   - `/linear-swap-api/v3/swap_matchresults`
   - `/linear-swap-api/v3/swap_financial_record`

3. **New Order Types**
   - `market` (cross-margin only, true market orders)
   - `fok_best_price`
   - Lightning close: `/linear-swap-api/v1/swap_cross_lightning_close_position`

4. **Trigger/TP-SL/Trailing Orders**
   - `/linear-swap-api/v1/swap_cross_trigger_order`
   - `/linear-swap-api/v1/swap_cross_tpsl_order`
   - `/linear-swap-api/v1/swap_cross_track_order`

5. **Batch Tickers** (single request for all symbols)
   - `/v2/linear-swap-ex/market/detail/batch_merged`

6. **Incremental Depth** (lower latency than full snapshots)
   - WS: `market.$contract_code.depth.size_20.high_freq`

7. **Ed25519 Signing** — alternative to HMAC-SHA256

8. **New Rate Limits**
   - Private: 144 req/3s per UID (72 trading + 72 query)
   - Public market: 800 req/s per IP
   - Trigger orders: 5 req/s per UID

### Account Requirements
- Need HTX account at htx.com (not huobi.com)
- API key creation at htx.com/en-us/apikey/
- Enable "USDT-M Futures" permission on the API key
- Whitelist IP if using IP restriction
- For unified account: must explicitly opt-in (irreversible)

---

## Recommended Migration Plan

### Phase 1: Bug Fixes (No API changes)
1. Fix TickerWS ParseTicker call (BUG-1)
2. Fix context leaks (BUG-2)
3. Fix WSOrder.GetID format (BUG-3)
4. Fix WSOrder.GetReduceOnly (BUG-4)
5. Fix recursive reconnect (BUG-6)

### Phase 2: Dynamic Contract Info
6. Fetch PriceTicks/ContractSizes from API at startup (BUG-5)
7. Add periodic refresh (contracts can be added/delisted)

### Phase 3: New Features
8. Add incremental depth WS option for lower latency
9. Add batch ticker REST endpoint
10. Add v3 history endpoints
11. Add trigger/TP-SL order support
12. Consider unified account mode

### Phase 4: Hardening
13. Add rate limiter
14. Iterative reconnect loop
15. Add connection health metrics
16. Add order receipt confirmation tracking

