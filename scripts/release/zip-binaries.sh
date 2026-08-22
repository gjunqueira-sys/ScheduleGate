#!/usr/bin/env bash
set -euo pipefail

# Package release binaries into a zip file with README and LICENSE
# Usage: bash scripts/release/zip-binaries.sh [version]

VERSION="${1:-}"
RELEASE_DIR="release"
ZIP_NAME="schedulegate-${VERSION}.zip"

if [[ -z "$VERSION" ]]; then
    echo "Error: Version not provided" >&2
    echo "Usage: bash scripts/release/zip-binaries.sh v1.0.3" >&2
    exit 1
fi

# Create release directory if it doesn't exist
mkdir -p "$RELEASE_DIR"

# Verify all binaries exist
BINARIES=(
    "$RELEASE_DIR/schedulegate"
    "$RELEASE_DIR/schedulegate.exe"
    "$RELEASE_DIR/schedulegate-linux"
    "$RELEASE_DIR/schedulegate-gui"
)

for bin in "${BINARIES[@]}"; do
    if [[ ! -f "$bin" ]]; then
        echo "Error: Missing binary: $bin" >&2
        echo "Build all binaries first." >&2
        exit 1
    fi
done

# Generate README.txt
cat > "$RELEASE_DIR/README.txt" <<'READMEEOF'
ScheduleGate - DCMA 14-Point Schedule Assessment CLI
====================================================

Binaries included:
  schedulegate       - macOS (arm64) CLI
  schedulegate.exe   - Windows (amd64) CLI
  schedulegate-linux - Linux (amd64) CLI
  schedulegate-gui   - macOS (arm64) Desktop GUI

Installation:
  1. Extract this zip file
  2. Move the binary to a directory in your PATH
     macOS/Linux: sudo mv schedulegate /usr/local/bin/
     Windows: Move schedulegate.exe to a folder in your PATH
  3. Verify: schedulegate --version

Quick Start:
  # Assess a schedule against DCMA 14-point metrics
  schedulegate assess schedule.xlsx

  # Compare two schedule versions
  schedulegate compare old.xlsx new.xlsx

  # Validate column structure
  schedulegate validate schedule.xlsx

  # Show help
  schedulegate --help

Documentation:
  GitHub: https://github.com/gjunqueira-sys/ScheduleGate
  User Manual: Open user-manual.html in any browser

License: MIT (see LICENSE file)
READMEEOF

# Verify LICENSE file exists at project root
if [[ ! -f "LICENSE" ]]; then
    echo "Warning: LICENSE file not found at project root" >&2
    echo "Creating placeholder LICENSE" >&2
    cat > "$RELEASE_DIR/LICENSE" <<'LICEOF'
MIT License

Copyright (c) 2026 Gil Junqueira

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
LICEOF
else
    cp LICENSE "$RELEASE_DIR/LICENSE"
fi

# Copy user manual to release directory
if [[ ! -f "docs/user-manual.html" ]]; then
    echo "Error: docs/user-manual.html not found" >&2
    exit 1
fi
cp docs/user-manual.html "$RELEASE_DIR/user-manual.html"

# Create zip file
cd "$RELEASE_DIR"
rm -f "../$ZIP_NAME"
zip -r "../$ZIP_NAME" \
    schedulegate \
    schedulegate.exe \
    schedulegate-linux \
    schedulegate-gui \
    README.txt \
    LICENSE \
    user-manual.html

cd ..

# Report
ZIP_SIZE=$(du -h "$ZIP_NAME" | cut -f1)
echo "Created: $ZIP_NAME ($ZIP_SIZE)"
echo "Contents:"
unzip -l "$ZIP_NAME" | tail -n +4 | head -n 8
