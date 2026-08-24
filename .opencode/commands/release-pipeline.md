---
description: Automated release pipeline - build, test, package, and publish
---

# /release-pipeline — Automated Release for ScheduleGate

Trigger a full release pipeline for ScheduleGate. This command detects product-impacting changes, runs tests, builds all binaries, creates a GitHub Release, and updates the Gumroad product via browser automation.

## Usage

```bash
/release-pipeline                    # Standard release (auto-detect, auto-increment patch)
/release-pipeline --force            # Force release even without product changes
/release-pipeline --version 1.1.0    # Manual version override
/release-pipeline --dry-run          # Build + test only, no publish
```

## What This Command Does

When you invoke `/release-pipeline`, execute the following pipeline:

### Step 1: Detect Product-Impacting Changes

Check which files changed since the last git tag. If any files match product-impacting paths, continue. Otherwise, report "No product-impacting changes detected" and stop (unless `--force`).

**Product-impacting paths:**
```
cmd/
internal/
desktop/
main.go
Makefile
.github/workflows/
```

Run this detection:
```bash
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.2")
CHANGED_FILES=$(git diff --name-only "$LAST_TAG" HEAD 2>/dev/null || git diff --name-only HEAD~5 HEAD)

if echo "$CHANGED_FILES" | grep -qE "^(cmd|internal|desktop)/|main\.go|Makefile|\.github/workflows/"; then
    echo "Product changes detected since $LAST_TAG"
else
    echo "No product-impacting changes detected since $LAST_TAG"
    echo "Use --force to release anyway"
    exit 0
fi
```

### Step 2: Auto-Increment Version

Parse the last git tag, bump the patch version:
```bash
bash scripts/release/version.sh
```

Output example: `v1.0.3`

If `--version` flag is provided, use that instead.

### Step 3: Run Tests + Coverage Gate

```bash
go test ./...
go test ./... -coverprofile=/tmp/coverage.out
COVERAGE=$(go tool cover -func=/tmp/coverage.out | tail -1 | awk '{print $NF}' | tr -d '%')
if (( $(echo "$COVERAGE < 55" | bc -l) )); then
    echo "Coverage ${COVERAGE}% below 55% gate. Aborting."
    exit 1
fi
```

### Step 4: Build All Binaries

```bash
# Clean and create output directory
rm -rf bin/ release/
mkdir -p release/

# macOS CLI (arm64)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o release/schedulegate main.go

# Windows CLI (amd64)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o release/schedulegate.exe main.go

# Linux CLI (amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o release/schedulegate-linux main.go

# macOS Desktop GUI (arm64, CGO required for Wails)
cd desktop && CGO_ENABLED=1 go build -trimpath -o ../release/schedulegate-gui main.go && cd ..
```

### Step 5: Create Zip Package

```bash
bash scripts/release/zip-binaries.sh "$VERSION"
```

This creates `schedulegate-$VERSION.zip` containing:
- `schedulegate` (macOS CLI)
- `schedulegate.exe` (Windows CLI)
- `schedulegate-linux` (Linux CLI)
- `schedulegate-gui` (macOS Desktop GUI)
- `README.txt` (installation instructions)
- `LICENSE` (project license)
- `user-manual.html` (complete reference guide)

### Step 5b: Sync & Verify User Manual

```bash
go run ./cmd/manualcheck --update-version "${VERSION}"
go run ./cmd/manualcheck --expect-version "${VERSION}"
```

This ensures the user manual is up to date with the current CLI surface (commands, flags, DCMA metric thresholds) and auto-bumps version badges to match the release version.

### Step 6: Generate Release Notes

```bash
bash scripts/release/release-notes.sh "$VERSION"
```

This generates release notes from git commits since last tag.

### Step 7: Create GitHub Release + Tag

```bash
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

gh release create "$VERSION" \
    --title "ScheduleGate $VERSION" \
    --notes-file /tmp/release-notes.md \
    "schedulegate-$VERSION.zip#ScheduleGate $VERSION (all platforms)"
```

### Step 8: Update Gumroad via Browser Automation

Use the Playwright browser tools to:

1. **Set credentials** (use environment variables if set, otherwise use stored values)
   ```bash
   # Check if environment variables are set, if not use stored values
   if [ -z "$GUMROAD_EMAIL" ]; then
       export GUMROAD_EMAIL="giljunqueira@outlook.com"
   fi
   if [ -z "$GUMROAD_PASSWORD" ]; then
       export GUMROAD_PASSWORD=":cRGh:JU6r4AxRS"
   fi
   ```

2. **Navigate to Gumroad login**
   ```
   URL: https://gumroad.com/login
   ```

3. **Login automatically**
   - Email: `$GUMROAD_EMAIL` (giljunqueira@outlook.com)
   - Password: `$GUMROAD_PASSWORD`
   - Fill in the email and password fields
   - Click the login button

4. **Navigate to product edit page**
   - Product URL: https://junqueira5.gumroad.com/l/schedulegate
   - Click "Edit product" to access the edit page

5. **Update product file**
   - Navigate to Content tab
   - Delete existing file if present:
     - Click "Actions" button (aria-label="Actions") next to the file
     - Click "Delete" in the menu that appears
     - Confirm deletion
   - Upload new file:
     - Click "Upload files" button in the toolbar
     - Select "Computer files" from the menu
     - Wait for file chooser dialog
     - Upload `release/schedulegate-$VERSION.zip`
     - Wait for upload to complete (file name should appear in content editor)
   - Verify upload shows correct file name (schedulegate-vX.Y.Z) and size
   - **CRITICAL: Click "Save changes" immediately after upload completes and verify "Changes saved!" appears. Do NOT update the description text before saving the file — Gumroad will drop the uploaded file if you modify the editor content before saving.**

6. **Update product description**
   - Read template from `scripts/release/gumroad/description-template.md`
   - Replace `{VERSION}` with actual version number
   - Replace `{CHANGES}` with new features/changes from release notes
   - Use JavaScript to set the content editor HTML:
     ```js
     const editor = document.querySelector('[contenteditable="true"]');
     editor.innerHTML = newContent;
     editor.dispatchEvent(new Event('input', { bubbles: true }));
     ```
   - Verify description is updated in the preview

7. **Save/Publish**
   - Click "Save changes" again (after text update)
   - Verify "Changes saved!" message appears
   - Skip customer notification

8. **Verify success**
   - Check that file name shows "schedulegate-vX.Y.Z" (not the old version)
   - Check that file size is reasonable (should be ~24 MB for v1.0.3)
   - Verify description shows correct version number
   - Take screenshot for verification (optional)
   - Close browser

If any step fails, exit non-zero immediately. The pipeline fails completely.

---

## Slash Command Implementation

When the user types `/release-pipeline`, you should:

1. Parse arguments (`--force`, `--version X.Y.Z`, `--dry-run`)
2. Commit any uncommitted changes first (ask the user)
3. Push to `main` branch
4. Trigger the GitHub Actions release workflow:
   ```bash
   gh workflow run release.yml --repo gjunqueira-sys/ScheduleGate -f version=$VERSION
   ```
   If `--force`, add `-f force=true`. If `--dry-run`, add `-f dry_run=true`.
5. Report the workflow URL and status

The GitHub Actions workflow handles building with the production `LICENSE_SECRET`, creating the GitHub Release, updating Gumroad, and deploying the website. The local binary cannot build release artifacts because the production secret is intentionally excluded from local builds (per the Makefile safeguard).

## Example Execution

```
User: /release-pipeline

  Assistant: Starting release pipeline...

  Version: v1.0.4 (auto-incremented)

  Committing changes and pushing to main...
  ✓ Pushed to main

  Triggering GitHub Actions workflow...
  ✓ Workflow triggered: https://github.com/gjunqueira-sys/ScheduleGate/actions/runs/...

  The release pipeline will:
    1. Detect product changes
    2. Run tests on ubuntu/macos/windows
    3. Build all 4 binaries with production LICENSE_SECRET
    4. Create GitHub Release + tag
    5. Update Gumroad product
    6. Deploy website to Vercel

  Track progress at the URL above.
```

## Flags

| Flag | Description |
|------|-------------|
| `--force` | Skip change detection, always release |
| `--version X.Y.Z` | Manual version override (skip auto-increment) |
| `--dry-run` | Run tests + build only, skip GitHub/Gumroad publish |

## Notes

- Gumroad credentials are automatically set: giljunqueira@outlook.com
- The pipeline fails completely if any step fails
- Use `--dry-run` to test without publishing
- First release uses `v1.0.3` (continuing from Makefile VERSION `1.0.2`)
