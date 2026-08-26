#!/usr/bin/env bash
set -euo pipefail

# Generate release notes from git commits since last tag
# Usage: bash scripts/release/release-notes.sh [version]
# Output: Markdown release notes to stdout

VERSION="${1:-}"
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.2")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%d)

# Get commits since last tag
COMMITS=$(git log --oneline "$LAST_TAG"..HEAD 2>/dev/null || git log --oneline -10)

# Categorize commits
FIXES=$(echo "$COMMITS" | grep -i "^.*fix" || true)
FEATS=$(echo "$COMMITS" | grep -i "^.*feat" || true)
DOCS=$(echo "$COMMITS" | grep -i "^.*docs" || true)
OTHER=$(echo "$COMMITS" | grep -viE "^.*(fix|feat|docs|build|chore|style|refactor|test|ci)" || true)

cat <<EOF
# ScheduleGate ${VERSION}

**Release Date:** ${DATE}
**Commit:** ${COMMIT}

## What's New

EOF

if [[ -n "$FEATS" ]]; then
    echo "### Features"
    echo "$FEATS" | sed 's/^[a-f0-9]* /- /'
    echo ""
fi

if [[ -n "$FIXES" ]]; then
    echo "### Bug Fixes"
    echo "$FIXES" | sed 's/^[a-f0-9]* /- /'
    echo ""
fi

if [[ -n "$DOCS" ]]; then
    echo "### Documentation"
    echo "$DOCS" | sed 's/^[a-f0-9]* /- /'
    echo ""
fi

if [[ -n "$OTHER" ]]; then
    echo "### Other Changes"
    echo "$OTHER" | sed 's/^[a-f0-9]* /- /'
    echo ""
fi

cat <<EOF
## Downloads

- \`schedulegate\` — macOS (arm64) CLI
- \`schedulegate.exe\` — Windows (amd64) CLI
- \`schedulegate-linux\` — Linux (amd64) CLI

CLI binaries are bundled in a single zip file for convenience.

## Installation

\`\`\`bash
# 1. Download and extract
unzip schedulegate-${VERSION}.zip
cd schedulegate-${VERSION}

# 2. macOS/Linux: add to PATH
sudo mv schedulegate /usr/local/bin/

# 3. Verify
schedulegate --version
\`\`\`

## Documentation

- [GitHub Repository](https://github.com/gjunqueira-sys/ScheduleGate)
- [User Guide](https://github.com/gjunqueira-sys/ScheduleGate/blob/main/RELEASE.md)
- [CLI Reference](https://github.com/gjunqueira-sys/ScheduleGate/blob/main/AGENTS.md)

---
Community use is licensed under AGPLv3. Purchased Pro keys are licensed under LICENSE-COMMERCIAL.
EOF
