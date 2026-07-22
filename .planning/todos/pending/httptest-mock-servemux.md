---
title: "Use http.ServeMux to route mock requests"
status: pending
priority: medium
created: 2026-07-21
---

**GitHub issue:** [#35](https://github.com/guionardo/go/issues/35)

**Description:** Replace the manual for-loop mock matching with `http.ServeMux` patterns. The expected behavior is to register mocks as handlers on a ServeMux rather than iterating through a list of mocks on every request.
