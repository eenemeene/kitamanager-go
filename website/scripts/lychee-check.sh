#!/usr/bin/env bash
# Build the website and run lychee link check against the resulting public/.
# Mirrors the lychee step in .github/workflows/website.yml so PRs cannot land
# with broken internal links. Used by the website-lychee pre-commit hook.
#
# Resolution order: native binary > cargo install > podman > docker > error.
set -euo pipefail

cd "$(dirname "$0")/.."
hugo --quiet

if command -v lychee >/dev/null 2>&1; then
    exec lychee --config lychee.toml --offline --root-dir public public
fi

if command -v cargo >/dev/null 2>&1; then
    cargo install --locked lychee
    exec lychee --config lychee.toml --offline --root-dir public public
fi

# Fall back to a container if available — matches what CI does internally
# (lycheeverse/lychee-action publishes the same image).
for runtime in podman docker; do
    if command -v "$runtime" >/dev/null 2>&1; then
        exec "$runtime" run --rm -v "$(pwd):/work:Z" -w /work \
            lycheeverse/lychee:latest \
            --config lychee.toml --offline --root-dir /work/public public
    fi
done

cat >&2 <<'EOF'
lychee is not installed and no install path is available.
Install it via one of:
  - cargo install --locked lychee
  - https://lychee.cli.rs/installation/
  - or make sure podman/docker is on PATH so the hook can use the container image
EOF
exit 1
