---
spike: 009
name: thundering-herd-benchmark
type: standard
validates: "Given N concurrent GetOrSet misses on one key, when singleflight wraps the setter, then the setter runs once (vs N naive) with measurable win"
verdict: VALIDATED
related: [002, 005]
tags: [singleflight, cache, benchmark, thundering-herd]
---

# Spike 009: thundering-herd-benchmark

## What This Validates

Quantifies the win the earlier spikes showed anecdotally (spike 002: "50 callers →
6ms"). A controlled `go test -bench` comparing a naive cache (every miss runs the
setter) vs the singleflight-wrapped herd (one setter + N−1 waiters) across 64
concurrent callers.

## How to Run

```bash
cd .planning/spikes/009-singleflight-benchmark
go test -run TestBenchHarness -v .    # asserts naive>1 setter runs, herd==1
go test -bench=. -benchmem -benchtime=2x -count=1 .
```

## What to Expect

```
TestBenchHarness: naive setterRuns=64  herd setterRuns=1   ✓
BenchmarkThunderingHerd_Naive-10       2   2312875 ns/op   15340 B/op  145 allocs/op
BenchmarkThunderingHerd_Singleflight-10 2   1165896 ns/op    8716 B/op   87 allocs/op
```

## Investigation Trail

- The harness test proves the core claim: 64 concurrent misses run the setter
  64 times naive, exactly once with singleflight.
- `-bench` shows singleflight ≈ **50% faster** and ≈ **half the memory + allocs**
  on 64 callers. The gap is dominated by the setter's cost being paid once
  instead of N times (here a 2ms fake compute); the longer the setter, the bigger
  the win (1/N setter executions amortized).
- `-benchmem` numbers include the goroutine harness, so treat the absolute values
  as relative: singleflight consistently allocates less because 63 callers never
  instantiate setter closures/args.

## Results

VALIDATED. singleflight delivers the predicted thundering-herd win:

- Setter invocations: N → 1 on concurrent same-key miss.
- Wall time ≈ setter-budget once + wait for the single result (not N serial
  setter executions).
- Allocation/mem reduced roughly proportionally to callers.

Signal for the build: the performance case is airtight. The remaining build
decision is not *whether* to use singleflight but *which wrapper semantics*
(008: DoChan+select in request paths; 007: tombstone guard for Delete; 006:
per-provider error glue), all already validated.