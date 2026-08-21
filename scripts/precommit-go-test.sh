#!/usr/bin/env bash
#
# Run the race-enabled tests for the packages a commit actually touches.
#
# "Touches" includes embedded assets, not just Go files: a commit that only
# edits internal/i18n/locales/de.json or a file under
# internal/database/migrations changes what its package does, and the tests that
# judge it live in that package. See the `files` pattern on the go-test hook in
# .pre-commit-config.yaml for which paths route here, and
# resolve_embedding_package below for how they map back to a package.
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

# Resolve an embedded asset to the package that compiles it in.
#
# //go:embed can only reach into subdirectories of the embedding package, so the
# nearest ancestor that is a Go package is the one whose behaviour the file
# changes. Walking up rather than consulting a table means a new //go:embed
# needs a line in .pre-commit-config.yaml and nothing here.
resolve_embedding_package() {
  local dir="$1"
  while [ "$dir" != "." ] && [ "$dir" != "/" ]; do
    if go list "./$dir" >/dev/null 2>&1; then
      printf './%s\n' "$dir"
      return 0
    fi
    dir=$(dirname "$dir")
  done
  return 1
}

# Map each staged file to its package directory, then deduplicate. Files in the
# repository root map to ".", which `go test` understands.
#
# A .go file maps to its own directory. Anything else is here because the hook's
# `files` pattern routed it in as an embedded asset -- editing
# internal/i18n/locales/de.json changes what internal/i18n does without touching
# a Go file, and the tests that would catch a dropped translation live in that
# package. Those resolve by walking up; a .go file does not, because when
# `go list` fails on its own directory the cause is a build tag, and substituting
# the parent package would run the wrong tests rather than report the skip.
packages=$(for file in "$@"; do
  dir=$(dirname "$file")
  dir=${dir#./}
  case "$file" in
  *.go)
    printf './%s\n' "$dir"
    ;;
  *)
    resolve_embedding_package "$dir" ||
      echo "skipping $file: no enclosing Go package" >&2
    ;;
  esac
done | sort -u)

# Drop packages whose files are all behind a build tag this run does not set --
# internal/contract (-tags=contract) and internal/integration (-tags=integration).
# `go test` treats "build constraints exclude all Go files" as a failure, so
# editing one of them would fail the hook for a package it never intended to
# build. Both have their own dedicated hooks that pass the right tag, so nothing
# goes unrun. Detected via `go list` rather than by name, so a future tag-gated
# package needs no change here.
buildable=""
for pkg in $packages; do
  if go list "$pkg" >/dev/null 2>&1; then
    buildable="$buildable $pkg"
  else
    echo "skipping $pkg: no files build without a tag (it has its own hook)"
  fi
done
packages="$buildable"

if [ -z "${packages// /}" ]; then
  exit 0
fi

# A package directory that holds no test files makes `go test` print "no test
# files" rather than fail, which is fine and keeps the output honest about what
# ran.
echo "go test -race -p 1 for the packages this commit touches:"
printf '  %s\n' $packages

# shellcheck disable=SC2086 -- word splitting is intended: one argument per package
# The default 10-minute per-package timeout is not enough for internal/service
# under -race: that package alone runs ~4 minutes without the race detector, and
# the detector's overhead pushes it past the limit — the failure is a panic
# reporting "test timed out", which reads like a hang rather than a budget.
# Serialized testcontainer startup is part of that cost and is deliberate.
exec go test -race -p 1 -timeout 30m $packages
