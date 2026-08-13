#!/usr/bin/env bash
#
# Run the race-enabled tests for the packages a commit actually touches.
#
# The hook used to run `go test -race -p 1 ./...` on every commit containing a Go
# file: the whole suite, serialized, with the race detector. That is roughly
# fourteen minutes before each commit lands, and CI runs the same suite again on
# the pull request a few minutes later. The local run is there for fast feedback,
# not to be the authority — so it now tests what changed and leaves exhaustive
# proof to CI.
#
# Serialization stays. Each package brings up its own Postgres testcontainer, so
# the default per-CPU parallelism starts several at once and flakes on resource
# exhaustion (`connection refused` mid-run), which -race's memory overhead makes
# worse.
#
# What this deliberately does not do is chase dependents. Editing internal/models
# can break internal/service without touching it, and this will not catch that;
# the pull request will, minutes later. Trading that for a hook that runs in
# seconds is the point — a fourteen-minute pre-commit gate is one people learn to
# skip with --no-verify, which protects nothing at all.

set -euo pipefail

if [ "$#" -eq 0 ]; then
  exit 0
fi

# Map each staged Go file to its package directory, then deduplicate. Files in
# the repository root map to ".", which `go test` understands.
packages=$(for file in "$@"; do
  dir=$(dirname "$file")
  printf './%s\n' "${dir#./}"
done | sort -u)

# A package directory that holds no test files makes `go test` print "no test
# files" rather than fail, which is fine and keeps the output honest about what
# ran.
echo "go test -race -p 1 for the packages this commit touches:"
printf '  %s\n' $packages

# shellcheck disable=SC2086 -- word splitting is intended: one argument per package
exec go test -race -p 1 $packages
