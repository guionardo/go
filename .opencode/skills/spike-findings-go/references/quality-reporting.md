# Quality Reporting

## Requirements

- Report must be in markdown format
- Must include lint results from golangci-lint (v2.12.2 JSON output)
- Must include security vulnerability scan from govulncheck (line-delimited JSON)
- Must include test coverage per function from `go test -cover`
- Must include lines of code and file counts by extension
- Must include full dependency list from `go list -m all`
- Must be runnable via a single Makefile target

## How to Build It

### 1. Create the report script

Place `quality-report.sh` in `.planning/spikes/001-golangci-lint-report/`. The script:

1. Runs `golangci-lint run --output.json.path` and parses `Issues` array grouped by `FromLinter`
2. Runs `govulncheck -json` and parses the `Vulnerabilities` object from its line-delimited JSON output
3. Runs `go test ./... -coverprofile` and formats with `go tool cover -func`
4. Counts files and lines by extension using `find` and `wc -l`
5. Lists dependencies via `go list -m all`
6. Outputs everything as markdown sections

**Key code pattern:**

```bash
# golangci-lint JSON parsing
golangci-lint run --output.json.path=/tmp/report.json
python3 -c "
with open('/tmp/report.json') as f:
    data = json.load(f)
issues = data.get('Issues', [])
by_linter = {}
for i in issues:
    l = i.get('FromLinter', 'unknown')
    by_linter[l] = by_linter.get(l, 0) + 1
"

# govulncheck — line-delimited JSON, find the Vulnerabilities object
govulncheck -json ./... > /tmp/vuln.json
python3 -c "
with open('/tmp/vuln.json') as f:
    for line in f:
        obj = json.loads(line)
        if 'Vulnerabilities' in obj:
            process(obj.get('Vulnerabilities', []))
"
```

### 2. Add the Makefile target

```makefile
quality-report:  ## Generate and open quality report
	@bash .planning/spikes/001-golangci-lint-report/quality-report.sh quality-report.md .
	@echo "Report: quality-report.md"
	@echo "Open with: open quality-report.md"
```

### 3. Run it

```bash
make quality-report
```

## What to Avoid

- **Do not** try to parse govulncheck output as a single JSON object — it outputs line-delimited JSON with multiple root objects
- **Do not** use `--new-from-rev` or `--new-from-merge-base` in the lint run — the report should show ALL issues, not just new ones
- **Do not** try to run `golangci-lint` HTML output as a replacement — it only covers linting, not security/coverage/metrics

## Constraints

- golangci-lint v2+ required (JSON output format differs from v1)
- govulncheck requires Go ≥1.22 for some vulnerability checks
- govulncheck output format is line-delimited JSON (not a single JSON object)

## Origin

Synthesized from spikes: 001
Source files available in: sources/001-golangci-lint-report/
