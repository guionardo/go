# Quality Report

<p>
<img src='https://img.shields.io/badge/Lint-191%20issues-red' alt='Lint'>
<img src='https://img.shields.io/badge/Security-0%20known-brightgreen' alt='Security'>
<img src='https://img.shields.io/badge/Coverage-75%25-yellow' alt='Coverage'>
<img src='https://img.shields.io/badge/Build-passing-brightgreen' alt='Build'>
</p>
Generated: 2026-08-08T10:55:49Z
Project: github.com/guionardo/go

## Lint Results

**Enabled linters:** 48  **Issues found:** 191

| Linter | Issues |
|--------|--------|
| wsl_v5 | 98 |
| modernize | 24 |
| paralleltest | 19 |
| errcheck | 16 |
| copyloopvar | 9 |
| funlen | 8 |
| staticcheck | 6 |
| cyclop | 3 |
| gocognit | 3 |
| errorlint | 2 |
| decorder | 1 |
| lll | 1 |
| maintidx | 1 |

<details>
<summary>Issue Details (191 total)</summary>

| Location | Linter | Message |
|----------|--------|---------|
| cache/bench_test.go:142 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:163 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:363 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:376 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:406 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:453 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:466 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:496 | copyloopvar | The copy of the 'for' variable "n" can be deleted (Go 1.22+) |
| cache/bench_test.go:675 | copyloopvar | The copy of the 'for' variable "p" can be deleted (Go 1.22+) |
| cache/bench_test.go:156 | cyclop | calculated cyclomatic complexity for function BenchmarkSingleflightGetOrSet is 17, max is 10 |
| cache/bench_test.go:358 | cyclop | calculated cyclomatic complexity for function runBatchBenchmarks is 20, max is 10 |
| cache/bench_test.go:446 | cyclop | calculated cyclomatic complexity for function BenchmarkBatch is 36, max is 10 |
| cache/cache.go:34 | decorder | multiple "type" declarations are not allowed; use parentheses instead |
| cache/bench_test.go:173 | errcheck | Error return value of `c.Close` is not checked |
| cache/bench_test.go:203 | errcheck | Error return value of `provider.Close` is not checked |
| cache/bench_test.go:237 | errcheck | Error return value of `provider.Close` is not checked |
| cache/bench_test.go:268 | errcheck | Error return value of `provider.Close` is not checked |
| cache/bench_test.go:311 | errcheck | Error return value of `pgProvider.Close` is not checked |
| cache/bench_test.go:448 | errcheck | Error return value of `c.Close` is not checked |
| cache/bench_test.go:678 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:99 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:112 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:120 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:136 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:153 | errcheck | Error return value of `c.Close` is not checked |
| cache/memcache/memcache_test.go:161 | errcheck | Error return value of `c.Close` is not checked |
| cache/postgres/postgres.go:160 | errcheck | Error return value of `br.Close` is not checked |
| cache/postgres/postgres.go:202 | errcheck | Error return value of `br.Close` is not checked |
| cache/postgres/postgres.go:231 | errcheck | Error return value of `br.Close` is not checked |
| cache/memcache/memcache.go:235 | errorlint | comparing with == will fail on wrapped errors. Use errors.Is to check for a specific error |
| cache/redis/redis.go:108 | errorlint | comparing with == will fail on wrapped errors. Use errors.Is to check for a specific error |
| cache/bench_test.go:156 | funlen | Function 'BenchmarkSingleflightGetOrSet' is too long (149 > 60) |
| cache/bench_test.go:358 | funlen | Function 'runBatchBenchmarks' is too long (82 > 60) |
| cache/bench_test.go:446 | funlen | Function 'BenchmarkBatch' is too long (232 > 60) |
| cache/concrete_cache_test.go:228 | funlen | Function 'TestConcreteCache_Batch' is too long (81 > 60) |
| cache/mem/mem_test.go:166 | funlen | Function 'TestMemCache_Batch' is too long (74 > 60) |
| cache/memcache/memcache_test.go:94 | funlen | Function 'TestMemcacheCache_Batch' is too long (71 > 60) |
| cache/redis/redis_test.go:92 | funlen | Function 'TestRedisCache_Batch' is too long (78 > 60) |
| cache/valkey/valkey_test.go:92 | funlen | Function 'TestValkeyCache_Batch' is too long (78 > 60) |
| cache/bench_test.go:156 | gocognit | cognitive complexity 32 of func `BenchmarkSingleflightGetOrSet` is high (> 30) |
| cache/bench_test.go:358 | gocognit | cognitive complexity 86 of func `runBatchBenchmarks` is high (> 30) |
| cache/bench_test.go:446 | gocognit | cognitive complexity 118 of func `BenchmarkBatch` is high (> 30) |
| cache/postgres/postgres.go:186 | lll | The line is 136 characters long, which exceeds the maximum of 120 characters. |
| cache/bench_test.go:446 | maintidx | Function name: BenchmarkBatch, Cyclomatic Complexity: 36, Halstead Volume: 7574.08, Maintainability Index: 16 |
| cache/bench_test.go:84 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:86 | modernize | waitgroupgo: Goroutine creation can be simplified using WaitGroup.Go |
| cache/bench_test.go:115 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:117 | modernize | waitgroupgo: Goroutine creation can be simplified using WaitGroup.Go |
| cache/bench_test.go:142 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:163 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:300 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:363 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:376 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:380 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:391 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:406 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:410 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:425 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:453 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:466 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:470 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:481 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:496 | modernize | forvar: copying variable is unneeded |
| cache/bench_test.go:500 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:515 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:658 | modernize | rangeint: for loop can be modernized using range over int |
| cache/bench_test.go:675 | modernize | forvar: copying variable is unneeded |
| cache/concrete_cache_test.go:96 | modernize | mapsloop: Replace m[k]=v loop with maps.Copy |
| cache/memcache/memcache_internal_test.go:23 | paralleltest | Function TestMemcacheBatch_NoServer missing the call to method parallel in the test run
 |
| cache/memcache/memcache_internal_test.go:28 | paralleltest | Function TestMemcacheBatch_NoServer missing the call to method parallel in the test run
 |
| cache/memcache/memcache_internal_test.go:34 | paralleltest | Function TestMemcacheBatch_NoServer missing the call to method parallel in the test run
 |
| cache/memcache/memcache_internal_test.go:40 | paralleltest | Function TestMemcacheBatch_NoServer missing the call to method parallel in the test run
 |
| cache/memcache/memcache_internal_test.go:45 | paralleltest | Function TestMemcacheBatch_NoServer missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:94 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel
 |
| cache/memcache/memcache_test.go:97 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:110 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:118 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:134 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:151 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/memcache/memcache_test.go:159 | paralleltest | Function TestMemcacheCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:130 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel
 |
| cache/postgres/postgres_test.go:134 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:145 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:150 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:163 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:168 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/postgres/postgres_test.go:182 | paralleltest | Function TestPostgresCache_Batch missing the call to method parallel in the test run
 |
| cache/bench_test.go:184 | staticcheck | SA1019: tcredis.RunContainer is deprecated: use Run instead RunContainer creates an instance of the Redis container type  |
| cache/bench_test.go:215 | staticcheck | SA1019: wait.ForAll(
	wait.ForLog("* Ready to accept connections"),
	wait.ForListeningPort("6379/tcp"),
).WithStartupTimeout is deprecated: use WithDeadline  |
| cache/bench_test.go:276 | staticcheck | SA1019: tcpostgres.RunContainer is deprecated: use Run instead RunContainer creates an instance of the Postgres container type  |
| cache/bench_test.go:545 | staticcheck | SA1019: tcredis.RunContainer is deprecated: use Run instead RunContainer creates an instance of the Redis container type  |
| cache/bench_test.go:575 | staticcheck | SA1019: wait.ForAll(
	wait.ForLog("* Ready to accept connections"),
	wait.ForListeningPort("6379/tcp"),
).WithStartupTimeout is deprecated: use WithDeadline  |
| cache/bench_test.go:634 | staticcheck | SA1019: tcpostgres.RunContainer is deprecated: use Run instead RunContainer creates an instance of the Postgres container type  |
| cache/bench_test.go:33 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:37 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:51 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:72 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:78 | wsl_v5 | missing whitespace above this line (invalid statement above for) |
| cache/bench_test.go:88 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:92 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:96 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:98 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:109 | wsl_v5 | missing whitespace above this line (invalid statement above for) |
| cache/bench_test.go:126 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:130 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:132 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:174 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:196 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:202 | wsl_v5 | missing whitespace above this line (too many statements above defer) |
| cache/bench_test.go:204 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:230 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:236 | wsl_v5 | missing whitespace above this line (too many statements above defer) |
| cache/bench_test.go:238 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:261 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:267 | wsl_v5 | missing whitespace above this line (too many statements above defer) |
| cache/bench_test.go:269 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:290 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:299 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/bench_test.go:305 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:307 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:312 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:327 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:329 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:346 | wsl_v5 | missing whitespace above this line (no shared variables above range) |
| cache/bench_test.go:351 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:383 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:394 | wsl_v5 | missing whitespace above this line (invalid statement above range) |
| cache/bench_test.go:415 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:417 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:430 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:432 | wsl_v5 | missing whitespace above this line (invalid statement above range) |
| cache/bench_test.go:449 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:473 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:484 | wsl_v5 | missing whitespace above this line (invalid statement above range) |
| cache/bench_test.go:505 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:507 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:520 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:522 | wsl_v5 | missing whitespace above this line (invalid statement above range) |
| cache/bench_test.go:557 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:563 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:590 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:596 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:620 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:626 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/bench_test.go:648 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/bench_test.go:657 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/bench_test.go:663 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/bench_test.go:665 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/bench_test.go:679 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/concrete_cache_test.go:84 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/concrete_cache_test.go:98 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/concrete_cache_test.go:112 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/mem/mem.go:96 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/mem/mem.go:108 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/mem/mem.go:119 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/memcache/memcache.go:171 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/memcache/memcache.go:175 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/memcache/memcache.go:177 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/memcache/memcache.go:198 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/memcache/memcache.go:207 | wsl_v5 | missing whitespace above this line (invalid statement above send) |
| cache/memcache/memcache.go:212 | wsl_v5 | missing whitespace above this line (no shared variables above range) |
| cache/memcache/memcache.go:239 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/memcache/memcache.go:243 | wsl_v5 | missing whitespace above this line (invalid statement above send) |
| cache/memcache/memcache.go:248 | wsl_v5 | missing whitespace above this line (no shared variables above range) |
| cache/postgres/postgres.go:165 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/postgres/postgres.go:169 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/postgres/postgres.go:172 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/postgres/postgres.go:176 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/postgres/postgres.go:178 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/postgres/postgres.go:198 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/postgres/postgres.go:206 | wsl_v5 | missing whitespace above this line (no shared variables above range) |
| cache/postgres/postgres.go:235 | wsl_v5 | missing whitespace above this line (no shared variables above range) |
| cache/redis/redis.go:99 | wsl_v5 | missing whitespace above this line (too many statements above range) |
| cache/redis/redis.go:103 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/redis/redis.go:111 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/redis/redis.go:114 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/redis/redis.go:118 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/redis/redis.go:120 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/redis/redis.go:134 | wsl_v5 | missing whitespace above this line (invalid statement above expr) |
| cache/redis/redis.go:140 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/redis/redis.go:149 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/redis/redis.go:152 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/valkey/valkey.go:129 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/valkey/valkey.go:137 | wsl_v5 | missing whitespace above this line (invalid statement above if) |
| cache/valkey/valkey.go:140 | wsl_v5 | missing whitespace above this line (never cuddle decl) |
| cache/valkey/valkey.go:144 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/valkey/valkey.go:146 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/valkey/valkey.go:164 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/valkey/valkey.go:168 | wsl_v5 | missing whitespace above this line (invalid statement above assign) |
| cache/valkey/valkey.go:177 | wsl_v5 | missing whitespace above this line (too many lines above return) |
| cache/valkey/valkey.go:198 | wsl_v5 | missing whitespace above this line (too many lines above return) |

</details>

## Security Vulnerabilities

No known vulnerabilities found.

## Test Coverage

**Total coverage:** 75%

<details>
<summary>Per-function coverage</summary>

```
github.com/guionardo/go/br_docs/brdocs.go:16:			IsCPF				100.0%
github.com/guionardo/go/br_docs/brdocs.go:27:			IsCNPJ				100.0%
github.com/guionardo/go/br_docs/brdocs.go:38:			isCadastro			100.0%
github.com/guionardo/go/br_docs/brdocs.go:65:			calcCadastroDigit		100.0%
github.com/guionardo/go/br_docs/brdocs.go:85:			RemoveNonDigitAndLetters	100.0%
github.com/guionardo/go/br_docs/brdocs.go:98:			allEq				100.0%
github.com/guionardo/go/cache/concrete_cache.go:26:		NewConcreteCache		100.0%
github.com/guionardo/go/cache/concrete_cache.go:33:		Get				100.0%
github.com/guionardo/go/cache/concrete_cache.go:39:		Set				100.0%
github.com/guionardo/go/cache/concrete_cache.go:44:		Delete				100.0%
github.com/guionardo/go/cache/concrete_cache.go:55:		GetOrSet			100.0%
github.com/guionardo/go/cache/concrete_cache.go:65:		Close				100.0%
github.com/guionardo/go/cache/concrete_cache.go:71:		MGet				100.0%
github.com/guionardo/go/cache/concrete_cache.go:76:		MSet				100.0%
github.com/guionardo/go/cache/concrete_cache.go:81:		MDel				100.0%
github.com/guionardo/go/cache/errors.go:23:			Error				100.0%
github.com/guionardo/go/cache/errors.go:29:			Unwrap				75.0%
github.com/guionardo/go/cache/errors.go:50:			newPanic			100.0%
github.com/guionardo/go/cache/mem/config.go:14:			WithDefaultTTL			100.0%
github.com/guionardo/go/cache/mem/config.go:20:			WithMaxEntries			100.0%
github.com/guionardo/go/cache/mem/config.go:25:			WithSweepInterval		100.0%
github.com/guionardo/go/cache/mem/mem.go:20:			New				100.0%
github.com/guionardo/go/cache/mem/mem.go:43:			GetFunc				100.0%
github.com/guionardo/go/cache/mem/mem.go:55:			SetFunc				100.0%
github.com/guionardo/go/cache/mem/mem.go:63:			DeleteFunc			100.0%
github.com/guionardo/go/cache/mem/mem.go:71:			CloseFunc			100.0%
github.com/guionardo/go/cache/mem/mem.go:86:			MGetFunc			100.0%
github.com/guionardo/go/cache/mem/mem.go:100:			MSetFunc			100.0%
github.com/guionardo/go/cache/mem/mem.go:112:			MDelFunc			100.0%
github.com/guionardo/go/cache/mem/mem.go:122:			resolveTTL			88.9%
github.com/guionardo/go/cache/mem/store.go:21:			isExpired			100.0%
github.com/guionardo/go/cache/mem/store.go:25:			newMemStore			100.0%
github.com/guionardo/go/cache/mem/store.go:32:			get				100.0%
github.com/guionardo/go/cache/mem/store.go:46:			set				100.0%
github.com/guionardo/go/cache/mem/store.go:52:			validateMaxEntries		83.3%
github.com/guionardo/go/cache/mem/store.go:66:			delete				100.0%
github.com/guionardo/go/cache/mem/store.go:70:			removeExpired			100.0%
github.com/guionardo/go/cache/mem/sweeper.go:8:			sweepLoop			85.7%
github.com/guionardo/go/cache/mem/sweeper.go:24:		sweep				100.0%
github.com/guionardo/go/cache/memcache/memcache.go:33:		New				100.0%
github.com/guionardo/go/cache/memcache/memcache.go:50:		GetFunc				0.0%
github.com/guionardo/go/cache/memcache/memcache.go:84:		SetFunc				0.0%
github.com/guionardo/go/cache/memcache/memcache.go:114:		DeleteFunc			0.0%
github.com/guionardo/go/cache/memcache/memcache.go:135:		CloseFunc			100.0%
github.com/guionardo/go/cache/memcache/memcache.go:142:		MGetFunc			73.7%
github.com/guionardo/go/cache/memcache/memcache.go:183:		MSetFunc			81.0%
github.com/guionardo/go/cache/memcache/memcache.go:229:		MDelFunc			77.8%
github.com/guionardo/go/cache/memcache/memcache.go:264:		resolveTTL			90.9%
github.com/guionardo/go/cache/memcache/options.go:16:		defaultConfig			100.0%
github.com/guionardo/go/cache/memcache/options.go:25:		WithServers			100.0%
github.com/guionardo/go/cache/memcache/options.go:32:		WithTimeout			100.0%
github.com/guionardo/go/cache/memcache/options.go:39:		WithDefaultTTL			100.0%
github.com/guionardo/go/cache/memcache/options.go:46:		WithMaxIdleConns		100.0%
github.com/guionardo/go/cache/options.go:21:			Apply				100.0%
github.com/guionardo/go/cache/options.go:26:			WithDefaultTTL			100.0%
github.com/guionardo/go/cache/postgres/options.go:17:		defaultConfig			100.0%
github.com/guionardo/go/cache/postgres/options.go:27:		WithConnString			100.0%
github.com/guionardo/go/cache/postgres/options.go:35:		WithTableName			100.0%
github.com/guionardo/go/cache/postgres/options.go:42:		WithPoolSize			100.0%
github.com/guionardo/go/cache/postgres/options.go:49:		WithSweepInterval		100.0%
github.com/guionardo/go/cache/postgres/options.go:56:		WithDefaultTTL			100.0%
github.com/guionardo/go/cache/postgres/postgres.go:37:		New				53.8%
github.com/guionardo/go/cache/postgres/postgres.go:70:		GetFunc				0.0%
github.com/guionardo/go/cache/postgres/postgres.go:97:		SetFunc				0.0%
github.com/guionardo/go/cache/postgres/postgres.go:118:		DeleteFunc			0.0%
github.com/guionardo/go/cache/postgres/postgres.go:133:		CloseFunc			0.0%
github.com/guionardo/go/cache/postgres/postgres.go:148:		MGetFunc			0.0%
github.com/guionardo/go/cache/postgres/postgres.go:184:		MSetFunc			0.0%
github.com/guionardo/go/cache/postgres/postgres.go:219:		MDelFunc			0.0%
github.com/guionardo/go/cache/postgres/postgres.go:246:		resolveTTL			100.0%
github.com/guionardo/go/cache/postgres/sweeper.go:13:		sweepLoop			0.0%
github.com/guionardo/go/cache/postgres/sweeper.go:29:		sweep				0.0%
github.com/guionardo/go/cache/redis/options.go:17:		defaultConfig			100.0%
github.com/guionardo/go/cache/redis/options.go:25:		WithAddr			100.0%
github.com/guionardo/go/cache/redis/options.go:32:		WithPassword			100.0%
github.com/guionardo/go/cache/redis/options.go:39:		WithDB				100.0%
github.com/guionardo/go/cache/redis/options.go:46:		WithPoolSize			100.0%
github.com/guionardo/go/cache/redis/options.go:53:		WithDefaultTTL			100.0%
github.com/guionardo/go/cache/redis/redis.go:25:		New				100.0%
github.com/guionardo/go/cache/redis/redis.go:45:		GetFunc				0.0%
github.com/guionardo/go/cache/redis/redis.go:66:		SetFunc				71.4%
github.com/guionardo/go/cache/redis/redis.go:81:		DeleteFunc			0.0%
github.com/guionardo/go/cache/redis/redis.go:90:		CloseFunc			100.0%
github.com/guionardo/go/cache/redis/redis.go:97:		MGetFunc			0.0%
github.com/guionardo/go/cache/redis/redis.go:125:		MSetFunc			0.0%
github.com/guionardo/go/cache/redis/redis.go:144:		MDelFunc			0.0%
github.com/guionardo/go/cache/redis/redis.go:157:		resolveTTL			100.0%
github.com/guionardo/go/cache/singleflight.go:35:		Do				93.3%
github.com/guionardo/go/cache/singleflight.go:77:		callSetter			100.0%
github.com/guionardo/go/cache/valkey/options.go:17:		defaultConfig			100.0%
github.com/guionardo/go/cache/valkey/options.go:25:		WithAddr			100.0%
github.com/guionardo/go/cache/valkey/options.go:32:		WithPassword			100.0%
github.com/guionardo/go/cache/valkey/options.go:39:		WithDB				100.0%
github.com/guionardo/go/cache/valkey/options.go:46:		WithPoolSize			100.0%
github.com/guionardo/go/cache/valkey/options.go:53:		WithDefaultTTL			100.0%
github.com/guionardo/go/cache/valkey/valkey.go:28:		New				100.0%
github.com/guionardo/go/cache/valkey/valkey.go:48:		GetFunc				0.0%
github.com/guionardo/go/cache/valkey/valkey.go:73:		SetFunc				16.7%
github.com/guionardo/go/cache/valkey/valkey.go:96:		DeleteFunc			0.0%
github.com/guionardo/go/cache/valkey/valkey.go:110:		CloseFunc			66.7%
github.com/guionardo/go/cache/valkey/valkey.go:120:		MGetFunc			0.0%
github.com/guionardo/go/cache/valkey/valkey.go:151:		MSetFunc			0.0%
github.com/guionardo/go/cache/valkey/valkey.go:182:		MDelFunc			0.0%
github.com/guionardo/go/cache/valkey/valkey.go:203:		resolveTTL			100.0%
github.com/guionardo/go/cmd/example-updater/main.go:20:		main				0.0%
github.com/guionardo/go/config/environment/environment.go:14:	GetEnv				100.0%
github.com/guionardo/go/config/environment/environment.go:33:	ParseEnvironment		91.7%
github.com/guionardo/go/config/environment/environment.go:109:	getFieldEnvValue		100.0%
github.com/guionardo/go/config/environment/environment.go:119:	setField			82.6%
github.com/guionardo/go/config/logging.go:15:			getConfigurationLog		100.0%
github.com/guionardo/go/config/logging.go:28:			getMapFromStruct		100.0%
github.com/guionardo/go/config/logging.go:57:			fieldPath			100.0%
github.com/guionardo/go/config/merger/maps.go:8:		MergeMaps			100.0%
github.com/guionardo/go/config/merger/maps.go:18:		updateMapValues			100.0%
github.com/guionardo/go/config/options.go:12:			postInit			100.0%
github.com/guionardo/go/config/options.go:36:			WithProfilesPath		100.0%
github.com/guionardo/go/config/options.go:49:			WithLogger			100.0%
github.com/guionardo/go/config/options.go:57:			WithDebugLogger			100.0%
github.com/guionardo/go/config/options.go:69:			WithScope			100.0%
github.com/guionardo/go/config/options.go:77:			WithDefaultScope		100.0%
github.com/guionardo/go/config/profile/profile.go:16:		GetScopedProfileContent		100.0%
github.com/guionardo/go/config/profile/profile.go:25:		getProfileMap			100.0%
github.com/guionardo/go/config/profile/profile.go:46:		readProfileMap			100.0%
github.com/guionardo/go/config/profile/profile.go:63:		getProfileFiles			100.0%
github.com/guionardo/go/config/profile/profile.go:90:		findYAMLFile			100.0%
github.com/guionardo/go/config/provider.go:43:			NewProvider			100.0%
github.com/guionardo/go/config/provider.go:64:			GetConfiguration		100.0%
github.com/guionardo/go/config/provider.go:85:			UpdateConfiguration		100.0%
github.com/guionardo/go/config/provider.go:92:			updateConfiguration		100.0%
github.com/guionardo/go/config/provider.go:114:			loadStaticConfiguration		64.7%
github.com/guionardo/go/config/provider_base.go:19:		getProfilesPath			100.0%
github.com/guionardo/go/config/provider_base.go:30:		validateConfiguration		100.0%
github.com/guionardo/go/config/validation/validator.go:21:	getValidator			100.0%
github.com/guionardo/go/config/validation/validator.go:32:	Validate			100.0%
github.com/guionardo/go/flow/default.go:4:			Default				100.0%
github.com/guionardo/go/flow/if.go:4:				If				100.0%
github.com/guionardo/go/flow/ordered_map.go:13:			NewOrderedMap			100.0%
github.com/guionardo/go/flow/ordered_map.go:19:			Set				100.0%
github.com/guionardo/go/flow/ordered_map.go:24:			Get				100.0%
github.com/guionardo/go/flow/ordered_map.go:29:			GetDefault			100.0%
github.com/guionardo/go/flow/ordered_map.go:37:			Delete				100.0%
github.com/guionardo/go/flow/ordered_map.go:41:			Keys				80.0%
github.com/guionardo/go/flow/ordered_map.go:55:			Range				75.0%
github.com/guionardo/go/flow/ordered_map.go:65:			Len				0.0%
github.com/guionardo/go/flow/ordered_map.go:69:			validateRange			100.0%
github.com/guionardo/go/flow/slices.go:3:			SliceFirstOrDefault		100.0%
github.com/guionardo/go/flow/slices.go:11:			SliceLastOrDefault		100.0%
github.com/guionardo/go/flow/slices.go:19:			RemoveItem			100.0%
github.com/guionardo/go/fraction/fraction.go:55:		New				100.0%
github.com/guionardo/go/fraction/fraction.go:86:		FromFloat64			93.8%
github.com/guionardo/go/fraction/fraction.go:149:		Add				100.0%
github.com/guionardo/go/fraction/fraction.go:161:		Divide				100.0%
github.com/guionardo/go/fraction/fraction.go:171:		Equal				100.0%
github.com/guionardo/go/fraction/fraction.go:176:		Multiply			100.0%
github.com/guionardo/go/fraction/fraction.go:182:		Subtract			100.0%
github.com/guionardo/go/fraction/fraction.go:188:		Float64				100.0%
github.com/guionardo/go/fraction/fraction.go:193:		Denominator			100.0%
github.com/guionardo/go/fraction/fraction.go:198:		Numerator			100.0%
github.com/guionardo/go/fraction/fraction.go:203:		abs				100.0%
github.com/guionardo/go/fraction/fraction.go:212:		gcd				100.0%
github.com/guionardo/go/fraction/fraction.go:221:		lcm				100.0%
github.com/guionardo/go/httptest_mock/builder.go:10:		NewMock				100.0%
github.com/guionardo/go/httptest_mock/builder.go:27:		WithQueryParam			100.0%
github.com/guionardo/go/httptest_mock/builder.go:33:		WithPathParam			100.0%
github.com/guionardo/go/httptest_mock/builder.go:39:		WithHeader			100.0%
github.com/guionardo/go/httptest_mock/builder.go:45:		WithBody			100.0%
github.com/guionardo/go/httptest_mock/builder.go:51:		WithResponseStatus		100.0%
github.com/guionardo/go/httptest_mock/builder.go:57:		WithResponseBody		100.0%
github.com/guionardo/go/httptest_mock/builder.go:63:		WithResponseHeader		100.0%
github.com/guionardo/go/httptest_mock/builder.go:69:		WithAssertion			100.0%
github.com/guionardo/go/httptest_mock/builder.go:77:		WithCustomHandler		100.0%
github.com/guionardo/go/httptest_mock/builder.go:95:		FastServe			100.0%
github.com/guionardo/go/httptest_mock/handler.go:56:		ServeHTTP			93.5%
github.com/guionardo/go/httptest_mock/handler.go:115:		Validate			77.8%
github.com/guionardo/go/httptest_mock/handler.go:135:		DoPreResponseHook		100.0%
github.com/guionardo/go/httptest_mock/handler.go:149:		Assert				100.0%
github.com/guionardo/go/httptest_mock/handler.go:156:		AddMocks			100.0%
github.com/guionardo/go/httptest_mock/handler.go:170:		log				100.0%
github.com/guionardo/go/httptest_mock/helpers.go:9:		GetMockHandlerFromServer	100.0%
github.com/guionardo/go/httptest_mock/helpers.go:26:		GetMocksFrom			100.0%
github.com/guionardo/go/httptest_mock/mock.go:65:		String				100.0%
github.com/guionardo/go/httptest_mock/mock.go:79:		Validate			100.0%
github.com/guionardo/go/httptest_mock/mock.go:87:		RegisterHit			100.0%
github.com/guionardo/go/httptest_mock/mock.go:103:		Assert				87.5%
github.com/guionardo/go/httptest_mock/mock.go:119:		Matches				100.0%
github.com/guionardo/go/httptest_mock/mock.go:125:		WriteResponse			100.0%
github.com/guionardo/go/httptest_mock/mock.go:136:		AcceptsPartialMatch		100.0%
github.com/guionardo/go/httptest_mock/mock.go:140:		AppendLog			0.0%
github.com/guionardo/go/httptest_mock/mock.go:146:		Logs				100.0%
github.com/guionardo/go/httptest_mock/mock.go:153:		Name				100.0%
github.com/guionardo/go/httptest_mock/mock.go:157:		GetPathValue			100.0%
github.com/guionardo/go/httptest_mock/mock.go:162:		GetQueryValue			0.0%
github.com/guionardo/go/httptest_mock/mock.go:167:		GetHeaderValue			0.0%
github.com/guionardo/go/httptest_mock/request.go:56:		String				100.0%
github.com/guionardo/go/httptest_mock/request.go:69:		match				100.0%
github.com/guionardo/go/httptest_mock/request.go:97:		setMatchLog			100.0%
github.com/guionardo/go/httptest_mock/request.go:112:		matchPath			100.0%
github.com/guionardo/go/httptest_mock/request.go:143:		matchPathParams			100.0%
github.com/guionardo/go/httptest_mock/request.go:161:		matchQueryParams		100.0%
github.com/guionardo/go/httptest_mock/request.go:180:		matchHeaders			100.0%
github.com/guionardo/go/httptest_mock/request.go:205:		matchBody			100.0%
github.com/guionardo/go/httptest_mock/request.go:229:		compareBody			100.0%
github.com/guionardo/go/httptest_mock/request.go:244:		marshalSorted			100.0%
github.com/guionardo/go/httptest_mock/response.go:29:		String				100.0%
github.com/guionardo/go/httptest_mock/response.go:39:		writeResponse			100.0%
github.com/guionardo/go/httptest_mock/response.go:49:		writeHeaderAndBody		100.0%
github.com/guionardo/go/httptest_mock/response.go:86:		setContentTypeIfNotSet		83.3%
github.com/guionardo/go/httptest_mock/setup.go:41:		SetupServer			84.6%
github.com/guionardo/go/httptest_mock/setup.go:84:		WithRequests			100.0%
github.com/guionardo/go/httptest_mock/setup.go:109:		WithRequestsFrom		100.0%
github.com/guionardo/go/httptest_mock/setup.go:130:		WithPostRequestHook		100.0%
github.com/guionardo/go/httptest_mock/setup.go:146:		WithAddMockInfoToResponse	100.0%
github.com/guionardo/go/httptest_mock/setup.go:163:		WithoutLog			100.0%
github.com/guionardo/go/httptest_mock/setup.go:175:		WithExtraLogger			75.0%
github.com/guionardo/go/httptest_mock/setup.go:188:		WithAcceptingPartialMatch	100.0%
github.com/guionardo/go/httptest_mock/setup.go:194:		readMocksFromPath		88.2%
github.com/guionardo/go/httptest_mock/setup.go:230:		readMocks			84.6%
github.com/guionardo/go/httptest_mock/setup.go:257:		readMock			100.0%
github.com/guionardo/go/httptest_mock/setup.go:277:		unmarshalMock			90.9%
github.com/guionardo/go/httptest_mock/string_parts.go:26:	String				100.0%
github.com/guionardo/go/httptest_mock/string_parts.go:61:	Set				100.0%
github.com/guionardo/go/httptest_mock/test_utils.go:15:		CreateTestRequest		100.0%
github.com/guionardo/go/httptest_mock/test_utils.go:25:		getBodyReader			88.9%
github.com/guionardo/go/mid/machineid_darwin.go:10:		MachineID			84.6%
github.com/guionardo/go/path_tools/find_file_path.go:14:	FindFileInPath			70.6%
github.com/guionardo/go/path_tools/path_tool.go:8:		DirExists			100.0%
github.com/guionardo/go/path_tools/path_tool.go:14:		CreatePath			100.0%
github.com/guionardo/go/path_tools/path_tool.go:23:		FileExists			100.0%
github.com/guionardo/go/path_tools/path_tool_darwin.go:7:	createPath			100.0%
github.com/guionardo/go/path_tools/root_directory.go:12:	init				50.0%
github.com/guionardo/go/path_tools/root_directory.go:21:	IsRootDirectory			100.0%
github.com/guionardo/go/path_tools/root_directory.go:26:	windowsPathBaseFunc		100.0%
github.com/guionardo/go/path_tools/root_folder.go:12:		GetRootFolder			92.9%
github.com/guionardo/go/reflect_tools/reflect_tools.go:12:	IsZeroValue			100.0%
github.com/guionardo/go/release/embed_swapper.go:19:		runtimeExeSuffix		66.7%
github.com/guionardo/go/release/embed_swapper.go:27:		swapperFilename			100.0%
github.com/guionardo/go/release/embed_swapper.go:31:		ExtractSwapper			75.0%
github.com/guionardo/go/release/release.go:55:			getCurrentModule		71.4%
github.com/guionardo/go/release/release.go:69:			GetThisLatestRelease		72.7%
github.com/guionardo/go/release/release.go:108:			GetLatestRelease		83.3%
github.com/guionardo/go/release/release.go:128:			Download			75.0%
github.com/guionardo/go/release/self_update.go:47:		String				100.0%
github.com/guionardo/go/release/self_update.go:63:		updateLockPath			75.0%
github.com/guionardo/go/release/self_update.go:72:		computeFileSHA256		100.0%
github.com/guionardo/go/release/self_update.go:84:		acquireUpdateLock		77.8%
github.com/guionardo/go/release/self_update.go:103:		downloadAndSwap			27.3%
github.com/guionardo/go/release/self_update.go:145:		resolveCurrentVersion		100.0%
github.com/guionardo/go/release/self_update.go:154:		PerformSelfUpdate		89.7%
github.com/guionardo/go/release/swapper/main.go:23:		main				0.0%
github.com/guionardo/go/release/swapper/main.go:85:		restoreBackup			80.0%
github.com/guionardo/go/release/swapper/main.go:97:		verifyChecksum			100.0%
github.com/guionardo/go/release/swapper/swap_unix.go:11:	atomicReplace			88.9%
github.com/guionardo/go/release/swapper/swap_unix.go:35:	relaunch			0.0%
github.com/guionardo/go/release/update.go:27:			apply				100.0%
github.com/guionardo/go/release/update.go:33:			WithOwner			100.0%
github.com/guionardo/go/release/update.go:39:			WithRepo			100.0%
github.com/guionardo/go/release/update.go:45:			WithGitHubToken			100.0%
github.com/guionardo/go/release/update.go:51:			CheckForUpdate			60.5%
github.com/guionardo/go/release/update.go:123:			DownloadUpdate			73.3%
github.com/guionardo/go/release/update.go:149:			findAsset			100.0%
github.com/guionardo/go/release/version.go:11:			ParseVersion			100.0%
github.com/guionardo/go/release/version.go:20:			GetCurrentVersion		71.4%
github.com/guionardo/go/set/marshal.go:7:			MarshalJSON			100.0%
github.com/guionardo/go/set/marshal.go:12:			UnmarshalJSON			100.0%
github.com/guionardo/go/set/scanner_valuer.go:9:		Scan				100.0%
github.com/guionardo/go/set/scanner_valuer.go:21:		Value				100.0%
github.com/guionardo/go/set/set.go:13:				New				100.0%
github.com/guionardo/go/set/set.go:21:				Add				100.0%
github.com/guionardo/go/set/set.go:27:				AddMultiple			100.0%
github.com/guionardo/go/set/set.go:36:				Remove				100.0%
github.com/guionardo/go/set/set.go:41:				Union				100.0%
github.com/guionardo/go/set/set.go:47:				Diff				100.0%
github.com/guionardo/go/set/set.go:66:				Intersection			100.0%
github.com/guionardo/go/set/set.go:80:				Iter				100.0%
github.com/guionardo/go/set/set.go:91:				UpdateFrom			100.0%
github.com/guionardo/go/set/set.go:100:				ToArray				100.0%
github.com/guionardo/go/set/set.go:110:				Has				100.0%
github.com/guionardo/go/set/set.go:116:				HasAll				100.0%
github.com/guionardo/go/set/set.go:127:				Filter				100.0%
github.com/guionardo/go/set/set.go:140:				Equals				100.0%
github.com/guionardo/go/set/set.go:155:				Clear				100.0%
github.com/guionardo/go/shell_tools/environment.go:8:		GetEnv				100.0%
github.com/guionardo/go/shell_tools/shell_args.go:12:		NewQuotedShellArgs		95.2%
github.com/guionardo/go/shell_tools/shell_args.go:59:		extractQuotedPrefix		84.6%
github.com/guionardo/go/shell_tools/shell_args.go:86:		String				85.7%
github.com/guionardo/go/time_tools/parser.go:20:		init				100.0%
github.com/guionardo/go/time_tools/parser.go:49:		Parse				100.0%
github.com/guionardo/go/time_tools/parser.go:75:		SetLayouts			100.0%
total:								(statements)			75.8%
```

</details>

## Lines of Code

| Extension | Files | Lines |
|-----------|-------|-------|
| .go | 165 | 14458 |
| .mod | 1 | 119 |
| .yml | 6 | 409 |
| .yaml | 10 | 198 |
| .json | 48 | 2455 |
| .md | 47 | 7812 |
| .sh | 2 | 356 |

**Total files:** 3866  **Total lines:** 1335925

## Direct Dependencies

| Module |
|--------|
github.com/guionardo/go

<details>
<summary>All Dependencies (212 modules)</summary>

| Module | Version |
|--------|---------|
| github.com/guionardo/go | |
| cel.dev/expr v0.25.1 | |
| cloud.google.com/go v0.123.0 | |
| cloud.google.com/go/auth v0.21.0 | |
| cloud.google.com/go/auth/oauth2adapt v0.2.8 | |
| cloud.google.com/go/compute/metadata v0.9.0 | |
| cloud.google.com/go/iam v1.5.2 | |
| cloud.google.com/go/longrunning v0.5.6 | |
| cloud.google.com/go/monitoring v1.24.2 | |
| cloud.google.com/go/storage v1.56.0 | |
| cloud.google.com/go/translate v1.10.3 | |
| dario.cat/mergo v1.0.2 | |
| github.com/AdaLogics/go-fuzz-headers v0.0.0-20240806141605-e8a1dd7889d6 | |
| github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0 | |
| github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0 | |
| github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 | |
| github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c | |
| github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 | |
| github.com/BurntSushi/toml v1.6.0 | |
| github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 | |
| github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.53.0 | |
| github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 | |
| github.com/Masterminds/semver/v3 v3.5.0 | |
| github.com/Microsoft/go-winio v0.6.2 | |
| github.com/anthropics/anthropic-sdk-go v1.57.0 | |
| github.com/aws/aws-sdk-go-v2 v1.30.3 | |
| github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 | |
| github.com/aws/aws-sdk-go-v2/config v1.27.27 | |
| github.com/aws/aws-sdk-go-v2/credentials v1.17.27 | |
| github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 | |
| github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 | |
| github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 | |
| github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 | |
| github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 | |
| github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 | |
| github.com/aws/aws-sdk-go-v2/service/sso v1.22.4 | |
| github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 | |
| github.com/aws/aws-sdk-go-v2/service/sts v1.30.3 | |
| github.com/aws/smithy-go v1.20.3 | |
| github.com/bahlo/generic-list-go v0.2.0 | |
| github.com/bradfitz/gomemcache v0.0.0-20260422231931-4d751bb6e37c | |
| github.com/bsm/ginkgo/v2 v2.12.0 | |
| github.com/bsm/gomega v1.27.10 | |
| github.com/buger/jsonparser v1.2.0 | |
| github.com/ccojocar/zxcvbn-go v1.0.4 | |
| github.com/cenkalti/backoff/v4 v4.3.0 | |
| github.com/cespare/xxhash/v2 v2.3.0 | |
| github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 | |
| github.com/containerd/errdefs v1.0.0 | |
| github.com/containerd/errdefs/pkg v0.3.0 | |
| github.com/containerd/log v0.1.0 | |
| github.com/containerd/platforms v0.2.1 | |
| github.com/containerd/typeurl/v2 v2.2.0 | |
| github.com/cpuguy83/dockercfg v0.3.2 | |
| github.com/creack/pty v1.1.24 | |
| github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc | |
| github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f | |
| github.com/distribution/reference v0.6.0 | |
| github.com/dnaeon/go-vcr v1.2.0 | |
| github.com/docker/go-connections v0.6.0 | |
| github.com/docker/go-units v0.5.0 | |
| github.com/ebitengine/purego v0.10.0 | |
| github.com/eliben/go-sentencepiece v0.7.0 | |
| github.com/envoyproxy/go-control-plane v0.14.0 | |
| github.com/envoyproxy/go-control-plane/envoy v1.37.0 | |
| github.com/envoyproxy/go-control-plane/ratelimit v0.1.0 | |
| github.com/envoyproxy/protoc-gen-validate v1.3.3 | |
| github.com/felixge/httpsnoop v1.1.0 | |
| github.com/gabriel-vasile/mimetype v1.4.13 | |
| github.com/go-jose/go-jose/v4 v4.1.4 | |
| github.com/go-logr/logr v1.4.3 | |
| github.com/go-logr/stdr v1.2.2 | |
| github.com/go-ole/go-ole v1.2.6 | |
| github.com/go-playground/assert/v2 v2.2.0 | |
| github.com/go-playground/locales v0.14.1 | |
| github.com/go-playground/universal-translator v0.18.1 | |
| github.com/go-playground/validator/v10 v10.30.3 | |
| github.com/go-task/slim-sprig/v3 v3.0.0 | |
| github.com/gogo/protobuf v1.3.2 | |
| github.com/golang-jwt/jwt/v5 v5.2.1 | |
| github.com/golang/glog v1.2.5 | |
| github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da | |
| github.com/golang/protobuf v1.5.4 | |
| github.com/golang/snappy v0.0.4 | |
| github.com/google/go-cmdtest v0.4.1-0.20220921163831-55ab3332a786 | |
| github.com/google/go-cmp v0.7.0 | |
| github.com/google/go-pkcs11 v0.3.0 | |
| github.com/google/jsonschema-go v0.4.2 | |
| github.com/google/martian/v3 v3.3.3 | |
| github.com/google/pprof v0.0.0-20260709232956-b9395ee17fa0 | |
| github.com/google/renameio v0.1.0 | |
| github.com/google/s2a-go v0.1.9 | |
| github.com/google/uuid v1.6.0 | |
| github.com/googleapis/enterprise-certificate-proxy v0.3.18 | |
| github.com/googleapis/gax-go/v2 v2.23.0 | |
| github.com/gookit/assert v0.1.1 | |
| github.com/gookit/color v1.6.1 | |
| github.com/gorilla/websocket v1.5.3 | |
| github.com/hashicorp/go-version v1.9.0 | |
| github.com/invopop/jsonschema v0.14.0 | |
| github.com/jackc/pgpassfile v1.0.0 | |
| github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 | |
| github.com/jackc/pgx/v5 v5.10.0 | |
| github.com/jackc/puddle/v2 v2.2.2 | |
| github.com/klauspost/compress v1.18.5 | |
| github.com/klauspost/cpuid/v2 v2.2.10 | |
| github.com/kr/pretty v0.3.1 | |
| github.com/kr/text v0.2.0 | |
| github.com/kylelemons/godebug v1.1.0 | |
| github.com/leodido/go-urn v1.4.0 | |
| github.com/lib/pq v1.12.3 | |
| github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 | |
| github.com/magiconair/properties v1.8.10 | |
| github.com/mdelapenya/tlscert v0.2.0 | |
| github.com/moby/docker-image-spec v1.3.1 | |
| github.com/moby/go-archive v0.2.0 | |
| github.com/moby/moby/api v1.54.2 | |
| github.com/moby/moby/client v0.4.0 | |
| github.com/moby/patternmatcher v0.6.1 | |
| github.com/moby/sys/mount v0.3.4 | |
| github.com/moby/sys/mountinfo v0.7.2 | |
| github.com/moby/sys/reexec v0.1.0 | |
| github.com/moby/sys/sequential v0.6.0 | |
| github.com/moby/sys/user v0.4.0 | |
| github.com/moby/sys/userns v0.1.0 | |
| github.com/moby/term v0.5.2 | |
| github.com/modelcontextprotocol/go-sdk v1.3.1 | |
| github.com/mozilla/tls-observatory v0.0.0-20250923143331-eef96233227e | |
| github.com/onsi/ginkgo/v2 v2.32.0 | |
| github.com/onsi/gomega v1.42.1 | |
| github.com/openai/openai-go/v3 v3.42.0 | |
| github.com/opencontainers/go-digest v1.0.0 | |
| github.com/opencontainers/image-spec v1.1.1 | |
| github.com/pb33f/ordered-map/v2 v2.3.1 | |
| github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c | |
| github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 | |
| github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 | |
| github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 | |
| github.com/redis/go-redis/v9 v9.21.0 | |
| github.com/rogpeppe/go-internal v1.14.1 | |
| github.com/russross/blackfriday v1.6.0 | |
| github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 | |
| github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 | |
| github.com/securego/gosec/v2 v2.28.0 | |
| github.com/segmentio/asm v1.1.3 | |
| github.com/segmentio/encoding v0.5.4 | |
| github.com/shirou/gopsutil/v4 v4.26.5 | |
| github.com/sirupsen/logrus v1.9.4 | |
| github.com/spiffe/go-spiffe/v2 v2.6.0 | |
| github.com/standard-webhooks/standard-webhooks/libraries v0.0.1 | |
| github.com/stretchr/objx v0.5.3 | |
| github.com/stretchr/testify v1.11.1 | |
| github.com/testcontainers/testcontainers-go v0.43.0 | |
| github.com/testcontainers/testcontainers-go/modules/postgres v0.43.0 | |
| github.com/testcontainers/testcontainers-go/modules/redis v0.43.0 | |
| github.com/tidwall/gjson v1.19.0 | |
| github.com/tidwall/match v1.2.0 | |
| github.com/tidwall/pretty v1.2.1 | |
| github.com/tidwall/sjson v1.2.5 | |
| github.com/tklauser/go-sysconf v0.3.16 | |
| github.com/tklauser/numcpus v0.11.0 | |
| github.com/valkey-io/valkey-go v1.0.76 | |
| github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e | |
| github.com/yosida95/uritemplate/v3 v3.0.2 | |
| github.com/yuin/goldmark v1.4.13 | |
| github.com/yusufpapurcu/wmi v1.2.4 | |
| github.com/zeebo/errs v1.4.0 | |
| github.com/zeebo/xxh3 v1.1.0 | |
| go.opencensus.io v0.24.0 | |
| go.opentelemetry.io/auto/sdk v1.2.1 | |
| go.opentelemetry.io/contrib/detectors/gcp v1.43.0 | |
| go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.67.0 | |
| go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 | |
| go.opentelemetry.io/otel v1.44.0 | |
| go.opentelemetry.io/otel/metric v1.44.0 | |
| go.opentelemetry.io/otel/sdk v1.44.0 | |
| go.opentelemetry.io/otel/sdk/metric v1.44.0 | |
| go.opentelemetry.io/otel/trace v1.44.0 | |
| go.uber.org/atomic v1.11.0 | |
| go.yaml.in/yaml/v3 v3.0.4 | |
| go.yaml.in/yaml/v4 v4.0.0-rc.6 | |
| golang.org/x/crypto v0.54.0 | |
| golang.org/x/exp v0.0.0-20220909182711-5c715a9e8561 | |
| golang.org/x/mod v0.38.0 | |
| golang.org/x/net v0.57.0 | |
| golang.org/x/oauth2 v0.36.0 | |
| golang.org/x/sync v0.22.0 | |
| golang.org/x/sys v0.47.0 | |
| golang.org/x/telemetry v0.0.0-20260708182218-49f421fb7959 | |
| golang.org/x/term v0.45.0 | |
| golang.org/x/text v0.40.0 | |
| golang.org/x/time v0.15.0 | |
| golang.org/x/tools v0.48.0 | |
| golang.org/x/tools/go/expect v0.1.1-deprecated | |
| golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated | |
| golang.org/x/vuln v1.6.0 | |
| golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 | |
| gonum.org/v1/gonum v0.17.0 | |
| google.golang.org/api v0.288.0 | |
| google.golang.org/appengine v1.6.8 | |
| google.golang.org/genai v1.63.0 | |
| google.golang.org/genproto v0.0.0-20260319201613-d00831a3d3e7 | |
| google.golang.org/genproto/googleapis/api v0.0.0-20260630182238-925bb5da69e7 | |
| google.golang.org/genproto/googleapis/bytestream v0.0.0-20260630182238-925bb5da69e7 | |
| google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 | |
| google.golang.org/grpc v1.82.1 | |
| google.golang.org/protobuf v1.36.11 | |
| gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c | |
| gopkg.in/yaml.v2 v2.2.8 | |
| gopkg.in/yaml.v3 v3.0.1 | |
| gotest.tools/v3 v3.5.2 | |
| pgregory.net/rapid v1.2.0 | |

</details>
