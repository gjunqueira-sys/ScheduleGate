#!/usr/bin/env bash
set -euo pipefail

# Sync or verify the product version across all customer-facing files.
# Usage:
#   bash scripts/release/sync-version.sh --apply v1.0.5
#   bash scripts/release/sync-version.sh --check v1.0.5
#   bash scripts/release/sync-version.sh --list

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"
exec go run ./cmd/versionsync "$@"
