---
description: Automated release pipeline - sync versions, CI build, GitHub Release, Gumroad
---

# /release-pipeline — Automated Release for ScheduleGate

Ship a new ScheduleGate version to customers. This command:

1. Syncs **every** customer-facing version string to the new rev
2. Commits and pushes that sync to `main`
3. Triggers GitHub Actions (`.github/workflows/release.yml`) to build signed binaries, tag, and create the GitHub Release
4. After CI succeeds, updates the Gumroad product via Playwright browser automation

**Never hardcode credentials.** Gumroad login comes from `GUMROAD_EMAIL` / `GUMROAD_PASSWORD` env vars (or ask the user). Do not write passwords into files, command docs, git, or chat logs.

## Usage

```
/release-pipeline                    # Standard release (auto-detect, auto-increment patch)
/release-pipeline --force            # Force release even without product changes
/release-pipeline --version 1.1.0    # Manual version override
/release-pipeline --dry-run          # Trigger CI build+test only, no GitHub Release / Gumroad / website
```

## Split of responsibilities

| Where | What |
|-------|------|
| **Local (this command)** | Change detection, version bump, `versionsync` across files, commit, push, trigger workflow, wait, Gumroad Playwright |
| **GitHub Actions `release.yml`** | Tests, coverage gate, build 4 binaries with production `LICENSE_SECRET`, zip, GitHub Release + tag, Vercel website deploy |
| **GitHub Actions `build.yml`** | Regular CI on every push/PR (not a release). Triggered automatically by the version-sync push. |

Local machines **must not** build release binaries — the Makefile refuses to embed the production `LICENSE_SECRET` outside CI.

## Version-bearing files (must all match)

`go run ./cmd/versionsync` is the single source of truth. It rewrites / checks:

| File | Field |
|------|--------|
| `Makefile` | `VERSION ?= X.Y.Z` |
| `README.md` | `**Public release:** vX.Y.Z — Month Year` |
| `web/index.html` | download URL `schedulegate-vX.Y.Z.zip` |
| `internal/version/version.go` | `var Version = "X.Y.Z"` (ldflags still override at build) |
| `desktop/build/config.yml` | `info.version` only (not Wails `version: '3'`) |
| `docs/user-manual.html` | version badges (≥2) |
| `docs/assess-manual.html` | version badges (≥2) |

`RELEASE.md` version history is append-only documentation — do not rewrite historical examples.

## What to do when the user invokes `/release-pipeline`

Parse args: `--force`, `--version X.Y.Z`, `--dry-run`. Then execute **in order**. Abort (non-zero) on the first failure.

### Step 1 — Detect product-impacting changes

Skip this step if `--force`.

```bash
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -z "$LAST_TAG" ]]; then
  echo "No tags found — treating as first release"
else
  CHANGED_FILES=$(git diff --name-only "$LAST_TAG" HEAD)
  echo "Changed since $LAST_TAG:"
  echo "$CHANGED_FILES"
  if ! echo "$CHANGED_FILES" | grep -qE "^(cmd|internal|desktop)/|main\.go|Makefile|\.github/workflows/"; then
    echo "No product-impacting changes detected since $LAST_TAG"
    echo "Use --force to release anyway"
    exit 0
  fi
fi
```

Product-impacting paths: `cmd/`, `internal/`, `desktop/`, `main.go`, `Makefile`, `.github/workflows/`.

### Step 2 — Determine version

```bash
if [[ -n "$MANUAL_VERSION" ]]; then
  VERSION=$(bash scripts/release/version.sh --version "$MANUAL_VERSION")
else
  VERSION=$(bash scripts/release/version.sh)
fi
echo "Releasing $VERSION"
```

`VERSION` always has a `v` prefix (e.g. `v1.0.5`).

### Step 3 — Sync every version-bearing file

```bash
go run ./cmd/versionsync --apply "$VERSION"
go run ./cmd/versionsync --check "$VERSION"
go run ./cmd/manualcheck --expect-version "$VERSION"
```

Show `git diff --stat` to the user.

### Step 4 — Commit and push (skip file commit on `--dry-run`)

If `--dry-run`: do **not** commit version-sync changes (leave them in the working tree or revert). Go to Step 5 with `dry_run=true`.

Otherwise:

1. If there are unrelated uncommitted changes, list them and **ask the user** before committing.
2. Ask the user to confirm the release commit. When they agree:

```bash
git add Makefile README.md web/index.html internal/version/version.go \
  desktop/build/config.yml docs/user-manual.html docs/assess-manual.html
git commit -m "chore: bump version to ${VERSION}"
git push origin HEAD
```

Do not commit secrets, zip artifacts, or `release/`.

### Step 5 — Trigger GitHub Actions release workflow

Repo: `gjunqueira-sys/ScheduleGate`. Workflow file: `release.yml`.

```bash
ARGS=(-f "version=${VERSION}")
# if --force:  ARGS+=(-f force=true)
# if --dry-run: ARGS+=(-f dry_run=true)

gh workflow run release.yml --repo gjunqueira-sys/ScheduleGate "${ARGS[@]}"
sleep 5
RUN_ID=$(gh run list --repo gjunqueira-sys/ScheduleGate --workflow=release.yml --limit 1 --json databaseId --jq '.[0].databaseId')
echo "Workflow: https://github.com/gjunqueira-sys/ScheduleGate/actions/runs/${RUN_ID}"
gh run watch "$RUN_ID" --repo gjunqueira-sys/ScheduleGate --exit-status
```

`release.yml` will:

1. Re-check product changes (unless `force`)
2. Test on ubuntu / macos-14 / windows + 55% coverage gate
3. **Fail** if `versionsync --check` disagrees (skipped on dry-run)
4. Build macOS / Windows / Linux CLI + macOS GUI with production `LICENSE_SECRET` and `${VERSION}` ldflags
5. Zip via `scripts/release/zip-binaries.sh`
6. Create annotated tag + GitHub Release with `schedulegate-${VERSION}.zip`
7. Deploy `web/` to Vercel production

Regular `build.yml` also runs on the version-sync push — that is expected (smoke tests, not the release artifacts).

If `gh run watch` exits non-zero, **stop**. Do not touch Gumroad.

### Step 6 — Dry-run exit

If `--dry-run`, print the run URL and stop. No GitHub Release, no Gumroad, no website publish.

### Step 7 — Download the release zip

```bash
gh release download "$VERSION" --repo gjunqueira-sys/ScheduleGate \
  --pattern "schedulegate-${VERSION}.zip" --dir /tmp
ZIP="/tmp/schedulegate-${VERSION}.zip"
ls -lh "$ZIP"
```

Use this absolute path for the Gumroad upload. Do not upload a locally-built zip.

### Step 8 — Update Gumroad via Playwright

Credentials:

```bash
: "${GUMROAD_EMAIL:?set GUMROAD_EMAIL}"
: "${GUMROAD_PASSWORD:?set GUMROAD_PASSWORD}"
PRODUCT_URL="${GUMROAD_PRODUCT_URL:-https://junqueira5.gumroad.com/l/schedulegate}"
```

If env vars are missing, **ask the user** for them. Never echo the password. Never write it to a file.

Prepare description:

```bash
NOTES=$(bash scripts/release/release-notes.sh "$VERSION")
# Read scripts/release/gumroad/description-template.md
# Replace {VERSION} with $VERSION
# Replace {CHANGES} with the "What's New" section of $NOTES
```

Playwright steps (use browser tools; abort on any failure):

1. Navigate to `https://gumroad.com/login`
2. Fill email (`GUMROAD_EMAIL`) and password (`GUMROAD_PASSWORD`), submit, wait for dashboard
3. Navigate to `$PRODUCT_URL` and open **Edit product**
4. **Content tab — replace the file**
   - If a file is already attached: click the file's **Actions** button (`aria-label="Actions"`), click **Delete**, confirm
   - Click **Upload files** → **Computer files**
   - Upload `$ZIP` via the file chooser
   - Wait until the file name `schedulegate-vX.Y.Z.zip` appears
   - **CRITICAL: Click "Save changes" immediately after the upload completes and wait for "Changes saved!". Do NOT edit the description before this save — Gumroad drops the uploaded file if the editor changes first.**
5. **Description**
   - Read `scripts/release/gumroad/description-template.md`, substitute `{VERSION}` and `{CHANGES}`
   - Set the contenteditable description (JS `innerHTML` + `input` event)
6. Click **Save changes** again, wait for "Changes saved!"
7. Skip customer notification if prompted
8. Verify:
   - File name shows `schedulegate-${VERSION}` (not the previous rev)
   - File size is on the order of ~20 MB
   - Description shows `$VERSION`
9. Close the browser

Optional helper (prints a JSON step manifest, does not drive the browser):

```bash
GUMROAD_EMAIL=... GUMROAD_PASSWORD=... \
  go run ./scripts/release/gumroad --zip "$ZIP" --version "$VERSION" \
    --description scripts/release/gumroad/description-template.md
```

### Step 9 — Report

```
Released ${VERSION}
  GitHub:  https://github.com/gjunqueira-sys/ScheduleGate/releases/tag/${VERSION}
  Zip:     schedulegate-${VERSION}.zip
  Website: https://www.schedulegate.dev
  Gumroad: https://junqueira5.gumroad.com/l/schedulegate
```

## Flags

| Flag | Description |
|------|-------------|
| `--force` | Skip change detection, always release |
| `--version X.Y.Z` | Manual version override (skip auto-increment) |
| `--dry-run` | CI build + test only; skip GitHub Release, Gumroad, website |

## Failure policy

The pipeline fails completely on the first error. A GitHub Release without a Gumroad update is a **partial release** — fix Gumroad before telling the user it shipped. Do not create a second tag to retry Gumroad; re-run only Step 7–8.

## Notes

- First-tag fallback in `version.sh` is `v1.0.2` (only used when the repo has no tags).
- Website curl URL is versioned (`.../releases/latest/download/schedulegate-vX.Y.Z.zip`). That is why `web/index.html` must be synced **before** the Vercel deploy in `release.yml`.
- `cmd/manualcheck --update-version` is **not** used in CI. Version rewrites happen locally via `versionsync` and are committed, so git stays the source of truth.
