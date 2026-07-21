#!/usr/bin/env bash
set -euo pipefail

OUTPUT="${1:-quality-report.md}"
PROJECT_ROOT="${2:-.}"

cd "$PROJECT_ROOT"

report() {
  echo "# Quality Report"
  echo
  echo "Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo "Project: $(go list -m 2>/dev/null || echo 'unknown')"
  echo

  # ── Lint results ──
  echo "## Lint Results"
  echo
  if golangci-lint run --output.json.path=/tmp/golint-report.json 2>/dev/null; then
    :
  fi
  if [ -f /tmp/golint-report.json ] && [ -s /tmp/golint-report.json ]; then
    python3 -c "
import json, sys
try:
    with open('/tmp/golint-report.json') as f:
        data = json.load(f)
    issues = data.get('Issues', [])
    linters = [l['Name'] for l in data.get('Report', {}).get('Linters', []) if l.get('Enabled')]
    print(f'**Enabled linters:** {len(linters)}')
    print(f'**Issues found:** {len(issues)}')
    print()
    by_linter = {}
    for i in issues:
        l = i.get('FromLinter', 'unknown')
        by_linter[l] = by_linter.get(l, 0) + 1
    if by_linter:
        print('| Linter | Issues |')
        print('|--------|--------|')
        for l, c in sorted(by_linter.items(), key=lambda x: -x[1]):
            print(f'| {l} | {c} |')
        print()
        print('### Issue Details')
        print()
        print('| Location | Linter | Message |')
        print('|----------|--------|---------|')
        for i in issues[:50]:
            pos = i.get('Pos', {})
            loc = f'{pos.get(\"Filename\", \"\")}:{pos.get(\"Line\", \"\")}'
            print(f'| {loc} | {i.get(\"FromLinter\", \"\")} | {i.get(\"Text\", \"\")} |')
        if len(issues) > 50:
            print(f'... and {len(issues) - 50} more issues')
    else:
        print('No issues found. Clean!')
except Exception as e:
    print(f'Error parsing lint report: {e}')
" 2>&1 || echo 'Lint report parsing failed'
  else
    echo 'golangci-lint output not available'
  fi

  # ── Security ──
  echo
  echo "## Security Vulnerabilities"
  echo
  if command -v govulncheck &>/dev/null; then
    govulncheck -json ./... 2>/dev/null > /tmp/govulncheck.json 2>&1 || true
    if [ -f /tmp/govulncheck.json ] && [ -s /tmp/govulncheck.json ]; then
      python3 -c "
import json
try:
    vulns = []
    with open('/tmp/govulncheck.json') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                if 'Vulnerabilities' in obj:
                    vulns = obj.get('Vulnerabilities', [])
                    break
            except json.JSONDecodeError:
                continue
    if vulns:
        print(f'**Vulnerabilities found:** {len(vulns)}')
        print()
        print('| Module | Vulnerability | Fixed in |')
        print('|--------|--------------|----------|')
        for v in vulns:
            print(f'| {v.get(\"ModulePath\", \"\")} | {v.get(\"ID\", \"\")} | {v.get(\"FixedVersion\", \"\")} |')
    else:
        print('No known vulnerabilities found.')
except Exception as e:
    print(f'govulncheck parse error: {e}')
" 2>&1 || echo 'Vulnerability check failed'
    else
      echo 'No govulncheck output'
    fi
  else
    echo 'govulncheck not installed — run: go install golang.org/x/vuln/cmd/govulncheck@latest'
  fi

  # ── Test coverage ──
  echo
  echo "## Test Coverage"
  echo
  go test ./... -coverprofile=/tmp/cover.out -covermode=atomic -count=1 2>/dev/null | tail -1 || true
  if [ -f /tmp/cover.out ] && [ -s /tmp/cover.out ]; then
    echo
    echo '```'
    go tool cover -func=/tmp/cover.out 2>/dev/null || true
    echo '```'
  fi

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
  total_files=$(find . -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f 2>/dev/null | wc -l | tr -d ' ')
  total_lines=$(find . -not -path './.planning/*' -not -path './node_modules/*' -not -path './.git/*' -type f -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
  echo
  echo "**Total files:** $total_files  **Total lines:** ${total_lines:-0}"

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
}

report > "$OUTPUT"
echo "Report written to $OUTPUT"
