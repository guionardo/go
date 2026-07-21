#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-quality-report.md}"
PROJECT_ROOT="${2:-.}"
cd "$PROJECT_ROOT"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# ── Collect data ──

# Lint
golangci-lint run --output.json.path="$TMPDIR/lint.json" 2>/dev/null || true
LINT_ISSUES=0
LINT_ISSUES=$(python3 -c "
import json
with open('$TMPDIR/lint.json') as f:
    d = json.load(f)
print(len(d.get('Issues', [])))
" 2>/dev/null || echo 0)

# Security
govulncheck -json ./... > "$TMPDIR/vuln.json" 2>/dev/null || true
VULN_COUNT=$(python3 -c "
import json, sys
with open('$TMPDIR/vuln.json') as f:
    for line in f:
        try:
            obj = json.loads(line)
            if 'Vulnerabilities' in obj:
                sys.stdout.write(str(len(obj.get('Vulnerabilities', []))))
                break
        except json.JSONDecodeError:
            continue
" 2>/dev/null)
VULN_COUNT=${VULN_COUNT:-0}

# Coverage
go test ./... -coverprofile="$TMPDIR/cover.out" -covermode=atomic -count=1 2>/dev/null | tail -1 || true
COV_PCT=0
COV_PCT=$(go tool cover -func="$TMPDIR/cover.out" 2>/dev/null | grep "^total:" | awk '{print $3}' | tr -d '%' | cut -d. -f1 || echo 0)

# Build
BUILD_OK=0
go build ./... 2>/dev/null && BUILD_OK=1

# Collect LOC
TOTAL_FILES=$(find . -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f 2>/dev/null | wc -l | tr -d ' ')
TOTAL_LINES=$(find . -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')

# Linters enabled
LINTERS=$(python3 -c "
import json
with open('$TMPDIR/lint.json') as f:
    d = json.load(f)
linters = [l['Name'] for l in d.get('Report', {}).get('Linters', []) if l.get('Enabled')]
print(len(linters))
" 2>/dev/null || echo 0)

# Issues by linter
ISSUES_BY_LINTER=$(python3 -c "
import json
with open('$TMPDIR/lint.json') as f:
    d = json.load(f)
by = {}
for i in d.get('Issues', []):
    l = i.get('FromLinter', 'unknown')
    by[l] = by.get(l, 0) + 1
for l, c in sorted(by.items(), key=lambda x: -x[1]):
    print(f'{l}|{c}')
" 2>/dev/null || true)

# ── Health ──

HEALTH=""
[ "$LINT_ISSUES" -le 5 ] && HEALTH+="✅ Lint  " || HEALTH+="⚠️ Lint($LINT_ISSUES)  "
[ "${VULN_COUNT:-0}" -eq 0 ] && HEALTH+="✅ Security  " || HEALTH+="❌ Security(${VULN_COUNT})  "
[ "${COV_PCT:-0}" -ge 80 ] && HEALTH+="✅ Coverage ${COV_PCT}%  " || HEALTH+="⚠️ Coverage ${COV_PCT}%  "
[ "$BUILD_OK" -eq 1 ] && HEALTH+="✅ Build  " || HEALTH+="❌ Build  "

# ── Generate report ──

{
  echo "# Quality Report"
  echo
  echo "**Code Health:** $HEALTH"
  echo
  SHIELDS_BASE="https://img.shields.io/badge"
  LINT_COLOR=brightgreen; [ "$LINT_ISSUES" -gt 5 ] && LINT_COLOR=yellow; [ "$LINT_ISSUES" -gt 50 ] && LINT_COLOR=red
  SEC_COLOR=brightgreen; [ "${VULN_COUNT:-0}" -gt 0 ] && SEC_COLOR=red
  COV_COLOR=brightgreen; [ "${COV_PCT:-0}" -lt 80 ] && COV_COLOR=yellow; [ "${COV_PCT:-0}" -lt 60 ] && COV_COLOR=red
  BLD_COLOR=brightgreen; [ "$BUILD_OK" -eq 0 ] && BLD_COLOR=red
  echo "<p>"
  echo "<img src='${SHIELDS_BASE}/Lint-${LINT_ISSUES}%20issues-${LINT_COLOR}' alt='Lint'>"
  echo "<img src='${SHIELDS_BASE}/Security-${VULN_COUNT}%20known-${SEC_COLOR}' alt='Security'>"
  echo "<img src='${SHIELDS_BASE}/Coverage-${COV_PCT}%25-${COV_COLOR}' alt='Coverage'>"
  echo "<img src='${SHIELDS_BASE}/Build-$([ $BUILD_OK -eq 1 ] && echo passing || echo failing)-${BLD_COLOR}' alt='Build'>"
  echo "</p>"
  echo "Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo "Project: $(go list -m 2>/dev/null || echo 'unknown')"
  echo

  # ── Lint ──
  echo "## Lint Results"
  echo
  echo "**Enabled linters:** $LINTERS  **Issues found:** $LINT_ISSUES"
  echo
  if [ "$LINT_ISSUES" -gt 0 ]; then
    echo "| Linter | Issues |"
    echo "|--------|--------|"
    while IFS='|' read -r linter count; do
      echo "| $linter | $count |"
    done <<< "$ISSUES_BY_LINTER"
    echo
    echo "### Issue Details"
    echo
    echo '| Location | Linter | Message |'
    echo '|----------|--------|---------|'
    python3 -c "
import json
with open('$TMPDIR/lint.json') as f:
    d = json.load(f)
for i in d.get('Issues', [])[:50]:
    pos = i.get('Pos', {})
    loc = f'{pos.get(\"Filename\", \"\")}:{pos.get(\"Line\", \"\")}'
    print(f'| {loc} | {i.get(\"FromLinter\", \"\")} | {i.get(\"Text\", \"\")} |')
" 2>/dev/null
    if [ "$LINT_ISSUES" -gt 50 ]; then
      echo "... and $((LINT_ISSUES - 50)) more issues"
    fi
  else
    echo 'No issues found. Clean!'
  fi

  # ── Security ──
  echo
  echo "## Security Vulnerabilities"
  echo
  if [ "$VULN_COUNT" -gt 0 ]; then
    echo "**Vulnerabilities found:** $VULN_COUNT"
    echo
    echo '| Module | Vulnerability | Fixed in |'
    echo '|--------|--------------|----------|'
    python3 -c "
import json
with open('$TMPDIR/vuln.json') as f:
    for line in f:
        try:
            obj = json.loads(line)
            for v in obj.get('Vulnerabilities', []):
                print(f'| {v.get(\"ModulePath\", \"\")} | {v.get(\"ID\", \"\")} | {v.get(\"FixedVersion\", \"\")} |')
        except: pass
" 2>/dev/null
  else
    echo 'No known vulnerabilities found.'
  fi

  # ── Coverage ──
  echo
  echo "## Test Coverage"
  echo
  echo "**Total coverage:** ${COV_PCT}%"
  echo
  echo '```'
  go tool cover -func="$TMPDIR/cover.out" 2>/dev/null || true
  echo '```'

  # ── Lines of Code ──
  echo
  echo "## Lines of Code"
  echo
  echo '| Extension | Files | Lines |'
  echo '|-----------|-------|-------|'
  for ext in go mod yml yaml json md sh tf; do
    files=$(find . -name "*.$ext" -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f 2>/dev/null | wc -l | tr -d ' ')
    if [ "$files" -gt 0 ]; then
      lines=$(find . -name "*.$ext" -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
      echo "| .$ext | $files | ${lines:-0} |"
    fi
  done
  echo
  echo "**Total files:** $TOTAL_FILES  **Total lines:** ${TOTAL_LINES:-0}"

  # ── Dependencies ──
  echo
  echo "## Direct Dependencies"
  echo
  echo '| Module |'
  echo '|--------|'
  go list -m 2>/dev/null || true
  echo
  echo "### All Dependencies"
  echo
  echo '| Module | Version |'
  echo '|--------|---------|'
  go list -m all 2>/dev/null | while IFS= read -r dep; do
    echo "| $dep | |"
  done || true
} > "$OUTPUT"

echo "Report written to $OUTPUT"
