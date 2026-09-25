# Phase 8: Benchmark suite - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-08
**Phase:** 8-Benchmark suite
**Areas discussed:** Benchmark file organization, Thundering-herd scenarios, Batch benchmark design, Provider coverage & Docker, Make benchmark target

---

## Benchmark File Organization

| Option | Description | Selected |
|--------|-------------|----------|
| Single file — cache/bench_test.go | All benchmarks in one file. Simplest, follows Go convention. | ✓ |
| Two domain files | Separate singleflight_bench_test.go + batch_bench_test.go | |
| Per-provider bench files | Benchmarks in each provider directory | |

**User's choice:** Single file — cache/bench_test.go
**Notes:** Keep it simple, one file for both benchmark domains.

---

## Thundering-Herd Scenarios

| Option | Description | Selected |
|--------|-------------|----------|
| Single concurrency level | Benchmark N=50 concurrent callers | |
| Multiple concurrency levels | Benchmark N=10, 50, 100, 500 concurrent callers | ✓ |
| Multiple providers | Benchmark across all 5 providers | |

**User's choice:** Multiple concurrency levels
**Notes:** Show how dedup win scales with contention using mem provider (zero-dependency).

---

## Batch Benchmark Design

| Option | Description | Selected |
|--------|-------------|----------|
| Single batch size, all ops | N=100 keys for MGet/MSet/MDel | |
| Multiple batch sizes, all ops | N=1, 10, 100, 1000 keys for MGet/MSet/MDel | |
| Incl. per-key fallback comparison | Compare batch ops against concreteCache per-key fallback | ✓ |

**User's choice:** Multiple batch sizes with per-key fallback comparison
**Notes:** Show exact win factor of batching vs naive per-key loop.

---

## Provider Coverage & Docker

| Option | Description | Selected |
|--------|-------------|----------|
| All 5 providers, Docker gated | All providers with skipIfNoDocker for non-mem | ✓ |
| mem only | Always runnable, zero deps | |
| mem + built-in Makefile variants | mem as default, separate benchmark-e2e target | |

**User's choice:** All 5 providers, Docker gated
**Notes:** Use same skipIfNoDocker pattern as existing E2E tests.

---

## Make Benchmark Target

| Option | Description | Selected |
|--------|-------------|----------|
| go test -bench=. -benchmem | Simple, standard Go pattern | |
| Named sub-targets | make benchmark, make benchmark-e2e, etc. | |
| With -benchtime and -count | -benchtime=1s -count=5 for statistical reliability | ✓ |

**User's choice:** With -benchtime and -count
**Notes:** Plus a make benchmark-quick for faster development runs.

---

## the agent's Discretion

- Benchmark naming conventions (subtest names, function naming)
- Specific benchmark implementation patterns

## Deferred Ideas

None.
