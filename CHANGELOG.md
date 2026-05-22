# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
