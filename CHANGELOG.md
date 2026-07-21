# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.4] — 2026-07-21

### Added
- Generic `Cache[K, V]` interface with 5 providers: in-memory, Redis, Valkey, Memcache, Postgres
- In-memory cache provider with background TTL sweep and concurrent-safe access
- Redis cache provider via go-redis/v9 with JSON serialization and functional options
- Valkey cache provider via valkey-go with sub-second TTL support (PX command)
- Memcache cache provider via gomemcache with goroutine-per-call context wrapping
- Postgres cache provider via pgx/v5 with UNLOGGED table, pg_prewarm, and background TTL sweep
- 50 E2E integration tests across all 5 providers using testcontainers-go
- Build-tag separation (`e2e`) for Docker-dependent tests

### Changed
- Makefile: `test-e2e` target passes `-tags=e2e` for build-tag separation

## [v1.3]

### Added
- Initial release with utility packages: config, flow, fraction, httptest_mock, mid, path_tools, reflect_tools, set, shell_tools, time_tools, br_docs

### Changed
- **config**: Fixed deadlock in `loadStaticConfiguration`; improved test coverage from 41% to 97%
- **set**: Added `example_test.go` with usage examples; minor improvements
- **mid**: Fixed `TestCollect` calling `MachineID()` instead of each collector; improved test coverage from 68% to 100%
- **time_tools**: Added example tests; parser improvements
- **shell_tools**: Added example tests; minor refactoring
- **reflect_tools**: Minor improvements
- **path_tools**: Cross-platform improvements; added example tests
- **httptest_mock**: Added example tests; handler and mock improvements
- **flow**: Added example tests; minor improvement
- **fraction**: Added example tests
- **CI**: Added `contents: read` permission to GitHub Actions workflow
