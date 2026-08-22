#!/usr/bin/env bash
set -euo pipefail

# Auto-increment semver patch version from git tags
# Usage: bash scripts/release/version.sh [manual_version]
# Output: v1.0.3 (or provided manual version)

MANUAL_VERSION="${1:-}"

# Handle --version flag prefix
if [[ "$MANUAL_VERSION" == "--version" ]]; then
    MANUAL_VERSION="${2:-}"
fi

if [[ -n "$MANUAL_VERSION" ]]; then
    # Validate format
    if [[ ! "$MANUAL_VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Error: Invalid version format: $MANUAL_VERSION" >&2
        echo "Expected format: v1.2.3 or 1.2.3" >&2
        exit 1
    fi
    # Ensure v prefix
    if [[ "$MANUAL_VERSION" != v* ]]; then
        MANUAL_VERSION="v$MANUAL_VERSION"
    fi
    echo "$MANUAL_VERSION"
    exit 0
fi

# Get the last git tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.2")

# Strip v prefix
VERSION="${LAST_TAG#v}"

# Parse major.minor.patch
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Validate numbers
if ! [[ "$MAJOR" =~ ^[0-9]+$ ]] || ! [[ "$MINOR" =~ ^[0-9]+$ ]] || ! [[ "$PATCH" =~ ^[0-9]+$ ]]; then
    echo "Error: Could not parse version: $LAST_TAG" >&2
    exit 1
fi

# Bump patch
NEW_PATCH=$((PATCH + 1))

# Output new version
echo "v${MAJOR}.${MINOR}.${NEW_PATCH}"
