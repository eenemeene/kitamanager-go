#!/usr/bin/env bash
#
# Run govulncheck and fail only on findings that are not in .govulncheck-allow.
#
# govulncheck has no ignore mechanism of its own (golang/go#60287), and without
# one a single unfixable upstream advisory turns every commit and every pull
# request red until the dependency ships a release. That is what GO-2026-6452
# in excelize did here from 2026-09-16: no fixed version exists, no change in
# this repository can clear it, and it blocked unrelated work.
#
# The allowlist is deliberately narrow. It matches on the OSV id alone, so a
# NEW vulnerability in the same module still fails, and it only ever applies to
# symbol-level findings — the ones govulncheck says your code actually reaches.
# Module- and package-level findings never fail the build in the first place,
# exactly as with plain govulncheck.
#
# Usage: scripts/govulncheck.sh [allowlist-file]
set -euo pipefail

ALLOWLIST="${1:-.govulncheck-allow}"
cd "$(dirname "$0")/.."

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck not found on PATH." >&2
  echo "Note that 'go install' puts it in \$(go env GOPATH)/bin, which is not" >&2
  echo "on PATH on every machine — check there before assuming it is missing." >&2
  exit 127
fi

allowed=""
if [ -f "$ALLOWLIST" ]; then
  # Ids start at column 0; justification lines are indented, comments start #.
  allowed=$(grep -E '^(GO|OSV)-[0-9]{4}-[0-9]+' "$ALLOWLIST" || true)
fi

report=$(mktemp)
trap 'rm -f "$report"' EXIT

set +e
govulncheck -format json $(go list ./... | grep -v internal/testutil) > "$report"
status=$?
set -e

# 0 = nothing found, 3 = vulnerabilities found. Anything else is govulncheck
# itself failing, and must not be swallowed by the allowlist.
if [ "$status" -ne 0 ] && [ "$status" -ne 3 ]; then
  echo "govulncheck failed to run (exit $status)" >&2
  cat "$report" >&2
  exit "$status"
fi

# A finding is symbol-level when its trace names a function; those are the ones
# govulncheck exits 3 for. jq reads the concatenated object stream natively.
found=$(jq -r 'select(.finding != null)
               | .finding
               | select((.trace // []) | length > 0)
               | select(.trace[0].function != null)
               | .osv' "$report" | sort -u)

# Advisory metadata, for a message that does not require opening the report.
summary_for() {
  jq -r --arg id "$1" 'select(.osv != null) | .osv | select(.id == $id) | .summary' "$report" | head -1
}

blocking=""
for id in $found; do
  if ! echo "$allowed" | grep -qx "$id"; then
    blocking="$blocking $id"
  fi
done

for id in $allowed; do
  if ! echo "$found" | grep -qx "$id"; then
    echo "NOTE: $ALLOWLIST still allows $id, which no longer appears."
    echo "      If upstream released a fix, delete that entry — a stale"
    echo "      suppression hides the vulnerability coming back."
  fi
done

for id in $found; do
  if echo "$allowed" | grep -qx "$id"; then
    echo "allowed: $id — $(summary_for "$id")"
  fi
done

if [ -n "$blocking" ]; then
  echo
  echo "govulncheck found vulnerabilities that are not allowed:" >&2
  for id in $blocking; do
    echo "  $id — $(summary_for "$id")" >&2
    echo "    https://pkg.go.dev/vuln/$id" >&2
  done
  echo >&2
  echo "Upgrade the dependency if a fixed release exists. If none does and the" >&2
  echo "exposure is genuinely bounded, add the id to $ALLOWLIST with a written" >&2
  echo "justification — see the existing entries for the standard expected." >&2
  exit 1
fi

echo "govulncheck: no unallowed vulnerabilities."
