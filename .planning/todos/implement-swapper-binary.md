---
title: Implement swapper binary
date: 2026-07-21
priority: high
---

# Implement swapper binary

- [ ] Create `release/swapper` package with `//go:embed` compatible design
- [ ] Implement binary swap with backup/restore
- [ ] Forward original program args to new binary
- [ ] Re-verify checksum of new binary before exec
- [ ] Clean up temp files and backup after successful swap
- [ ] Write tests for all error paths (swap fails, checksum mismatch, etc.)
