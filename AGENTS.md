# Project Instructions

## Before Every Commit

Run the coverage check and verify it passes:

```bash
make coverage-quick
```

This enforces the thresholds in `.testcoverage.yml`: packages ≥80%, files ≥70%, total ≥95%. Do not commit if it fails. Fix uncovered code or add tests first.
