#!/usr/bin/env bash
set -euo pipefail

# Package CLI release binaries into a zip with README, licenses, and user manual.
# Usage: bash scripts/release/zip-binaries.sh [version]

VERSION="${1:-}"
RELEASE_DIR="release"
ZIP_NAME="schedulegate-${VERSION}.zip"

if [[ -z "$VERSION" ]]; then
    echo "Error: Version not provided" >&2
    echo "Usage: bash scripts/release/zip-binaries.sh v1.0.3" >&2
    exit 1
fi

mkdir -p "$RELEASE_DIR"

BINARIES=(
    "$RELEASE_DIR/schedulegate"
    "$RELEASE_DIR/schedulegate.exe"
    "$RELEASE_DIR/schedulegate-linux"
)

for bin in "${BINARIES[@]}"; do
    if [[ ! -f "$bin" ]]; then
        echo "Error: Missing binary: $bin" >&2
        echo "Build all CLI binaries first." >&2
        exit 1
    fi
done

cat > "$RELEASE_DIR/README.txt" <<'READMEEOF'
ScheduleGate - DCMA 14-Point Schedule Assessment CLI
====================================================

Binaries included:
  schedulegate       - macOS (arm64) CLI
  schedulegate.exe   - Windows (amd64) CLI
  schedulegate-linux - Linux (amd64) CLI

Installation:
  1. Extract this zip file
  2. Move the binary for your OS into a directory on your PATH
     macOS:   sudo mv schedulegate /usr/local/bin/
     Linux:   sudo mv schedulegate-linux /usr/local/bin/schedulegate
     Windows: Move schedulegate.exe to a folder in your PATH
  3. Verify: schedulegate --version

Tiers:
  Community (free, AGPLv3)
    - No license key required
    - 1 assessment per calendar month, terminal output only
    - validate is always free

  Pro (commercial)
    - Unlimited assessments
    - HTML, CSV, Excel, and JSON reports
    - compare and check-patterns
    - Activate: schedulegate license set SG-...
    - Buy: https://schedulegate.dev

Quick Start:
  # Community: terminal assessment
  schedulegate assess schedule.xlsx

  # After activating Pro
  schedulegate assess schedule.xlsx --html report.html
  schedulegate compare old.xlsx new.xlsx

Documentation:
  Open user-manual.html in any browser
  Website: https://schedulegate.dev
  GitHub:  https://github.com/gjunqueira-sys/ScheduleGate

License:
  Community use is licensed under AGPLv3 (see LICENSE).
  Purchased Pro keys are licensed under LICENSE-COMMERCIAL.
READMEEOF

if [[ ! -f "LICENSE" ]]; then
    echo "Error: LICENSE file not found at project root" >&2
    exit 1
fi
cp LICENSE "$RELEASE_DIR/LICENSE"

if [[ ! -f "LICENSE-COMMERCIAL" ]]; then
    echo "Error: LICENSE-COMMERCIAL file not found at project root" >&2
    exit 1
fi
cp LICENSE-COMMERCIAL "$RELEASE_DIR/LICENSE-COMMERCIAL"

if [[ ! -f "docs/user-manual.html" ]]; then
    echo "Error: docs/user-manual.html not found" >&2
    exit 1
fi
cp docs/user-manual.html "$RELEASE_DIR/user-manual.html"

cd "$RELEASE_DIR"
rm -f "../$ZIP_NAME"
zip -r "../$ZIP_NAME" \
    schedulegate \
    schedulegate.exe \
    schedulegate-linux \
    README.txt \
    LICENSE \
    LICENSE-COMMERCIAL \
    user-manual.html

cd ..

ZIP_SIZE=$(du -h "$ZIP_NAME" | cut -f1)
echo "Created: $ZIP_NAME ($ZIP_SIZE)"
echo "Contents:"
unzip -l "$ZIP_NAME"
