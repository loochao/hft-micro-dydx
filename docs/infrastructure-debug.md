# Infrastructure Module Debug Report

**Date:** 2026-03-21  
**Scope:** All infrastructure modules in hft-micro-dydx  
**Files Reviewed:** 120+ Go source files across 13 modules

---

## Executive Summary

The infrastructure layer is a custom-built HFT crypto trading framework in Go, covering order book management, exchange interfaces, statistical streaming computations, StarkEx cryptographic signing (dydx), data recording, and various numeric utilities. The code is generally functional but contains several categories of bugs ranging from **critical** (potential money-losing issues in order signing, broken EMA/correlation) to **moderate** (memory leaks, numerical drift) to **minor** (code duplication, missing tests).

**Critical issues found: 8**  
**Moderate issues found: 14**  
**Minor issues found: 12**

---

## Module-by-Module Analysis

---

### 1. common/ (Core Types and Utilities)

#### CRITICAL BUGS

**1.1 SortedFloatSlice.Delete() - Incorrect handling of duplicates**
- **File:** common/types.go, SortedFloatSlice.Delete()
- When there are duplicate values, SearchFloat64s returns the index of ANY matching element, not necessarily the one intended to be deleted. This could corrupt the sorted slice used for median/min/max calculations in TimedMedian.
- **Impact:** Incorrect median calculations in spread monitoring, leading to wrong trading signals.

**1.2 ParseFloat() - Integer overflow for large mantissas**
- **File:** common/utils.go, ParseFloat()
- The custom float parser accumulates digits in a uint64 mantissa. For numbers with more than 19 significant digits, mantissa overflows silently. The fallback to strconv.ParseFloat only triggers on bad bytes, not overflow.
- **Impact:** Silent precision loss on large numbers in price parsing.

**1.3 ParseFloat() range check is inverted**
- **File:** common/utils.go
- `if -exp < 0 || -exp > 22` - when exp is positive (no decimal), -exp is negative, triggering the error path for valid integers. The fast path fails for integers, falling through to slow strconv.ParseFloat.
- **Impact:** Significant performance bug in a hot path for an HFT system.

**1.4 UnsafeBytesToString uses deprecated reflect.StringHeader**
- **File:** common/utils.go
- Uses reflect.StringHeader and unsafe.Pointer which is officially deprecated. The GC may collect the underlying byte slice since there is no proper reference keeping it alive.
- **Impact:** Potential use-after-free / memory corruption in hot JSON parsing paths.

#### MODERATE BUGS

**1.5 TimedSum/TimedMean/TimedMedian - Unbounded memory growth**
- **Files:** common/math.go
- All timed data structures use append() to grow slices and slice[cutIndex:] to shrink. Sub-slicing does NOT release the underlying array memory. Over hours/days, the backing arrays grow monotonically.
- **Impact:** Gradual memory leak. In a 24/7 trading system, will eventually cause OOM.

**1.6 RollingSum panics if window=0**
- **File:** common/math.go
- index starts at -1, wraps via modulo. If window is 0, modulo by zero causes panic.

**1.7 WalkDepthBBMAA - Panic on empty order book**
- **File:** common/depth.go
- output.BestBidPrice = bids[0][0] accessed before checking if bids/asks are empty.
- **Impact:** Process crash on empty depth updates during exchange maintenance.

**1.8 WalkDepthBMA / WalkCoinDepthWithMultiplier - Same empty book panic**
- **File:** common/depth.go

**1.9 KLinesMap.Load() / Save() - Resource leak on error**
- **File:** common/types.go
- If gzip.NewReader or decoder.Decode fails, the opened file f is never closed.

**1.10 GetFloatPrecision infinite loop on zero**
- **File:** common/float.go
- `for f < 1.0` - if f is 0.0 or negative, loops forever.
- **Impact:** Goroutine hang if called with zero tick/step size.

**1.11 Rank function - incomplete tie handling**
- **File:** common/rank.go
- Last group of tied elements is never averaged because the same flag only resolves on next non-equal element.

#### MINOR ISSUES

**1.12** Error messages use string types not error types (TickSizeNotFoundError etc.)

**1.13** ArchiveDailyJlGzFiles has no return in ctx.Done case (goroutine leak).

**1.14** RawWSMessageSaveLoop calls time.Sleep(5s) inside timer handler, blocking message processing.

**1.15** Typo: MircoPrice should be MicroPrice (WalkedDepthBBMAA and elsewhere).

---

### 2. logger/

#### MODERATE BUGS

**2.1 Errorf() uses warn logger instead of err logger**
- Errorf writes through warn.Output with "W" prefix instead of "E" prefix.
- **Impact:** Error messages misclassified in log output.

**2.2 Log level checking is fragile**
- String equality chains without case normalization. LOG_LEVEL=debug (lowercase) silently suppresses all logs.

**2.3 No log rotation or size limits.**

---

### 3. influx/ (InfluxDB Client)

Vendored copy of official InfluxDB v1 client library.

#### MODERATE BUGS

**3.1 InfluxWriter.watchPoints() - Stuck points never get flushed**
- **File:** common/influx-writer.go
- Points only flushed when len exceeds batchSize. No periodic flush timer.
- **Impact:** Data loss - last batch before shutdown may never reach InfluxDB.

**3.2 saveSilent suppresses writes for 1 minute**
- After a save, saveSilent=true for 1 minute. Points exceeding batchSize are not saved during that window.

**3.3 InfluxWriter.Stop() - Race between close(done) and watchPoints**
- If context cancelled simultaneously, final save may not execute.

---

### 4. starkex/ (StarkEx Crypto for dydx)

#### CRITICAL BUGS

**4.1 StarkwareOrder.CalculateHash() MUTATES input BigInts (MONEY BUG)**
- **File:** starkex/skex-types.go, CalculateHash()
- part1 := quantumsAmountSell then part1.Lsh(part1, ...) modifies quantumsAmountSell in place because big.Int methods modify the receiver. Since part1 is assigned by reference (not copied), the original so.QuantumsAmountCollateral or so.QuantumsAmountSynthetic is destroyed.
- **Impact:** If the order object is reused or inspected after signing, the amounts are corrupted. If CalculateHash is called twice, it produces a DIFFERENT hash. Direct money-losing bug.

**4.2 GoSign() seed mutation**
- **File:** starkex/skex-signature.go
- seed.And(seed, big.NewInt(1)) modifies seed in place. Callers seed gets corrupted.

**4.3 EcMult() - Recursive with up to 256-level depth**
- Significant allocation pressure in HFT signing path.

**4.4 Pedersen params parsing ignores errors**
- **File:** starkex/skex-types.go
- pd.FieldPrime, _ = new(big.Int).SetString(...) - ok bool silently discarded. Corrupt JSON causes nil pointer dereference.

#### MODERATE BUGS

**4.5 ToQuantumsExact uses math.Round but comment says exact**
- Floating point multiplication can produce non-integer results. Rounding may send slightly wrong quantities.

---

### 5. hdrhistogram/

Well-tested vendored HDR Histogram. No critical bugs. Window histogram creates GC pressure on rotation.

---

### 6. tdigest/

Vendored t-digest implementation.

**6.1 Uses global math/rand - not thread-safe pre-Go 1.20**

**6.2 Serialization precision loss** - Means serialized as float32 deltas. Crypto prices lose precision.

---

### 7. stream-stats/

#### CRITICAL BUGS

**7.1 TimedCorrelation computes COVARIANCE, not correlation**
- **File:** stream-stats/timed-correlation.go
- Computes E[XY] - E[X]E[Y] (covariance) but never divides by product of standard deviations. Correlation() returns covariance directly.
- **Impact:** Trading signals using "correlation" actually use covariance. Wrong scale, wrong interpretation. Could cause wildly incorrect position sizing.

**7.2 TimedEMA.Insert() never updates LastTime**
- **File:** stream-stats/timed-ema.go
- tm.LastTime is never set after initialization. diff always computed against zero time. Period converges to ~1.0, k ~= 1.0. EMA just tracks latest value with no smoothing.
- **Impact:** EMA is broken - behaves as pass-through. Any signal relying on EMA smoothing gets raw values.

#### MODERATE BUGS

**7.3 TimedVariance - Numerical instability**
- Uses naive formula E[X^2] - (E[X])^2 which suffers from catastrophic cancellation.
- **Impact:** Variance can go negative, producing NaN when taking sqrt.

**7.4 All timed structures share the same memory leak pattern** (sub-slicing never releases backing arrays).

**7.5 TimedMin - delete() can corrupt on floating-point mismatch**
- sort.SearchFloat64s returns insertion point, not exact match. Wrong element may get deleted.

**7.6 XY stats modules duplicate large amounts of logic** across 4 files.

---

### 8. rfc6979/

Standard RFC 6979 deterministic signature. Clean, well-tested. No bugs found.

---

### 9. talib/

Pure Go port of TA-Lib.

**9.1** Functions return zero-initialized arrays for warmup period without signaling this.

**9.2** No input validation. Empty slices or zero periods cause panics.

---

### 10. gogen/

Code generator for symbol-to-index lookup.

**10.1** Panics on empty BN_SYMBOLS environment variable.

---

### 11. ml/

Trivial Gorgonia test file. No production use.

---

### 12. round-trip-times/

**12.1** Counter increments twice per iteration in bnuf and kcuf. Only ~50 measurements instead of 100.

**12.2** bnus has no warmup skip unlike bnuf.

---

## Cross-Cutting Issues

### Memory Management
Almost every timed data structure uses slice[cutIndex:] which retains the original backing array. Fix: copy into fresh slices periodically.

### Concurrency Safety
None of the stream-stats or common math structures are thread-safe. Not documented or enforced.

### Error Handling
Errors logged at Debug level and swallowed. InfluxDB failures silently ignored.

### Duplicate Implementations
common/math.go and stream-stats/ both implement TimedMean, TimedSum, TimedWeightedMean with different APIs.

---

## Priority Fix List

| Priority | Issue | Module | Impact |
|----------|-------|--------|--------|
| P0 | 4.1 CalculateHash mutates BigInts | starkex | Corrupted order signatures |
| P0 | 7.2 TimedEMA never updates LastTime | stream-stats | EMA is non-functional |
| P0 | 7.1 Correlation returns covariance | stream-stats | Wrong statistical signals |
| P1 | 1.7/1.8 Empty orderbook panics | common | Process crashes |
| P1 | 1.10 GetFloatPrecision infinite loop | common | Goroutine hang |
| P1 | 2.1 Errorf uses warn logger | logger | Misclassified errors |
| P1 | 3.1 InfluxWriter unflushed points | common | Data loss |
| P1 | 1.2 ParseFloat overflow | common | Price parsing errors |
| P2 | 7.3 Variance numerical instability | stream-stats | NaN in variance |
| P2 | 1.5 Unbounded memory in timed structs | common/stream-stats | OOM over time |
| P2 | 4.4 Pedersen ignores parse errors | starkex | Nil pointer crash |
| P2 | 1.11 Rank tie-handling incomplete | common | Wrong rankings |
| P2 | 1.13 ArchiveDailyJlGzFiles goroutine leak | common | Resource leak |
| P3 | 1.4 UnsafeBytesToString deprecated | common | Memory corruption risk |
| P3 | 6.1 Global RNG contention | tdigest | Performance |
| P3 | Code duplication common/ vs stream-stats/ | multiple | Maintenance burden |

---

## Architecture Notes

### Positive Patterns
- Clean interface-based design for exchanges (CoinExchange, UsdExchange)
- Atomic types properly implemented using sync/atomic
- Ring buffers well-implemented with growth support
- StarkEx crypto implementation comprehensive (Pedersen hash, ECDSA)
- Good use of channels for decoupled data flow

### Patterns to Improve
- No context propagation in many goroutines
- Hardcoded constants should be configurable
- No graceful degradation when channel buffers full (drop and log)
- fmt.Errorf with user data (format string injection risk)

---

*Report generated by infrastructure code review, 2026-03-21*
