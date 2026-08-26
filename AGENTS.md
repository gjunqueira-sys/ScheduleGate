# AGENTS.md

## Project Summary

CLI tool for DCMA 14-Point Schedule Assessments on Microsoft Project exports. Reads Excel/CSV schedule files, produces HTML/CSV compliance reports. Single Go binary with cobra commands.

## Build & Test

```bash
make build              # macOS arm64 (CGO_ENABLED=0) → bin/schedulegate
make build-windows      # Windows amd64 cross-compile → bin/schedulegate.exe
make clean              # Remove bin/

go test ./...           # Run all tests (std lib only; no test framework or linter configured)
```

Build injects version metadata via ldflags from `internal/version/version.go`. Bump `VERSION ?=` in the Makefile when releasing. A `LICENSE_SECRET` build-time var can also be injected — leave empty for Community tier dev builds.

### CI/CD (`.github/workflows/build.yml`)

- **test**: `go vet ./...` + `go test ./...` on ubuntu/macos-14/windows, plus a coverage gate (ubuntu only, min 55% — raise/lower the threshold in the `coverage` step).
- **build-and-smoke**: builds the real binary per OS and smoke-tests it. Pro-gated commands (compare, check-patterns, `--json`, `--json-output`, `--csv`, `--exceptions-report`) work because the smoke build embeds a **test-only** license secret (`CI_LICENSE_SECRET`) and mints a throwaway Pro key with `license generate --key-only`. That secret is never the production signing secret. Test fixtures are generated at CI time via `go run ./cmd/genfixture`.
- **desktop**: frontend `npm ci && npm run build` (tsc + vite), `go vet`/`go test` on the desktop module, and a CGO darwin `go build` of the Wails GUI.
- **cross-compile**: compile-only for the remaining OS/arch combos.

Local tests all run in CI; the binary smoke suite additionally exercises the built artifact per OS.

### Release Pipeline (`.github/workflows/release.yml`)

Invoked via `/release-pipeline`. Local command syncs every version-bearing file, commits, and triggers CI. CI builds signed binaries, tags, and publishes the GitHub Release + website. Gumroad is updated **locally** via Playwright after CI succeeds (the workflow does not log into Gumroad).

```bash
/release-pipeline                    # Auto-detect changes, auto-increment patch
/release-pipeline --force            # Force release even without product changes
/release-pipeline --version 1.1.0    # Manual version override
/release-pipeline --dry-run          # CI build + test only, no publish
```

**Release scripts** (in `scripts/release/`):
- `version.sh` — Auto-increment semver patch from git tags
- `sync-version.sh` / `cmd/versionsync` — Rewrite/check version strings across Makefile, README, website, manuals, desktop config, `version.go`
- `release-notes.sh` — Generate changelog from commits since last tag
- `zip-binaries.sh` — Package binaries + README + LICENSE into zip
- `gumroad/main.go` — Prints a Playwright step manifest (credentials from env vars only)

**Gumroad credentials**: `GUMROAD_EMAIL`, `GUMROAD_PASSWORD`, optional `GUMROAD_PRODUCT_URL`. Never hardcode. Never commit.

**Zip contents**: `schedulegate` (macOS), `schedulegate.exe` (Windows), `schedulegate-linux` (Linux), `schedulegate-gui` (Desktop GUI), `README.txt`, `LICENSE`, `user-manual.html`

**Version sync gate**: `cmd/versionsync --check` fails the release job unless every customer-facing file shows the release version. `cmd/manualcheck` verifies `docs/user-manual.html` matches the CLI surface (commands, flags, DCMA metric thresholds). Both run in `release.yml`; `go test ./cmd/manualcheck/... ./cmd/versionsync/...` covers them in regular CI.

## Desktop GUI (Wails v3)

The `desktop/` directory is a **separate Go module** (`desktop/go.mod` with a `replace` directive to `../`). It imports the root module's packages.

### Prerequisites
- Node.js (for frontend build)
- Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### Build & Run

```bash
cd desktop/

# 1. Install frontend dependencies
cd frontend && npm ci && cd ..

# 2. Generate Wails bindings (only needed when Go service structs change)
wails3 generate module

# 3. Build frontend
cd frontend && npm run build && cd ..

# 4. Build Go binary (embeds frontend/dist)
GOOS=darwin CGO_ENABLED=1 go build -o bin/schedule-benchmark-gui

# Or use Taskfile:
wails3 task darwin:build
wails3 task darwin:run
```

Generated artifacts `frontend/bindings/` and `frontend/dist/` are tracked in git so a clone can build with just `npm ci && npm run build && go build`.

## CLI Commands

```bash
./bin/schedulegate assess file.xlsx                 # DCMA 14-point
./bin/schedulegate compare old.xlsx new.xlsx
./bin/schedulegate validate file.xlsx               # Column schema validation
./bin/schedulegate check-patterns file.xlsx --rules rules.yaml
./bin/schedulegate --version                        # Full build identity
```

Key assess flags: `--metrics 1,5,12`, `--html report.html`, `--csv db.csv`, `--exceptions-report exceptions.xlsx`, `--json` (stdout JSON), `--json-output report.json`, `--status-date 2026-04-10`, `--debug-logic`, `-v` (verbose counts). Files are positional args (no `--file` flag).

## Architecture

```
CLI (cobra cmd/) → internal/reader (parse Excel/CSV) → model.Schedule
  → engine (dcma/ | compare/ | rules/) → internal/report (HTML/CSV/Excel)
```

**Single entrypoint**: `main.go` calls `cmd.Execute()`.  
**Root command**: `cmd/root.go` — has two `init()` blocks (Go merges them), sets up viper config at `~/.schedulegate.yaml`.

### Package map

| Package | Role |
|---|---|
| `cmd/` | Cobra commands (self-contained, each adds itself via `init()`) |
| `cmd/genfixture/` | Test fixture generator used by CI smoke tests |
| `cmd/license-server/` | Separate binary — license minting server (store webhook backend) |
| `cmd/manualcheck/` | User-manual freshness gate (CLI surface vs `docs/user-manual.html`) |
| `cmd/versionsync/` | Rewrite/check product version across Makefile, README, website, manuals, desktop config |
| `internal/model/` | `Task` struct, `Schedule` struct — canonical types |
| `internal/reader/` | Excel/CSV parsing, case-insensitive fuzzy column header matching |
| `internal/dcma/` | `Metric` interface + 14 implementations; `DCMAAssessment` orchestrator |
| `internal/compare/` | Two-version delta engine; three-pillar scoring (Stability 40%/Reliability 30%/Churn 30%) |
| `internal/rules/` | YAML rule engine with glob pattern match on task fields |
| `internal/report/` | HTML (dark theme), CSV (Power BI), Excel exceptions writers |
| `internal/ui/` | Lipgloss terminal styling, ASCII logo |
| `internal/version/` | Build-time version vars (overridden via `-ldflags`) |
| `internal/license/` | HMAC-signed license tiers (Community/Pro/Team/Enterprise/Lifetime); feature gating |
| `desktop/` | Separate Go module — Wails v3 GUI with dark-themed xterm.js terminal |

### Extension point: DCMA Metric interface

```go
type Metric interface {
    ID() int
    Name() string
    Description() string
    Threshold() float64
    Assess(schedule *model.Schedule) MetricResult
}
```

## License Server Deployment

The license minting server (`cmd/license-server`) is deployed on **Railway** and uses **Resend HTTP API** for email delivery (not SMTP — Railway blocks outbound SMTP).

### Railway Service

| Detail | Value |
|---|---|
| **Project** | `schedulegate-license-server` |
| **Service** | `schedulegate-license-server` |
| **Public URL** | `https://schedulegate-license-server-production.up.railway.app` |
| **Private domain** | `schedulegate-license-server.railway.internal` |
| **Status** | Online (production environment) |

### Deploy command

```bash
make build-license-server     # → bin/license-server
railway service schedulegate-license-server
railway up                    # deploys from repo root via Dockerfile
```

### Environment Variables (set on Railway)

| Variable | Purpose |
|---|---|
| `SG_SECRET` | HMAC signing secret — MUST match CLI `LICENSE_SECRET` ldflag |
| `SG_ADMIN_TOKEN` | Bearer token for `POST /api/v1/mint` |
| `SG_WEBHOOK_TOKEN` | Token for Gumroad webhook `?token=` query param |
| `SG_SMTP_PASS` | Resend API key (uses Resend HTTP API, not SMTP) |
| `SG_SMTP_FROM` | From address: `noreply@schedulegate.dev` |

Note: `SG_SMTP_HOST`, `SG_SMTP_PORT`, `SG_SMTP_USER` are **no longer needed** — email is sent via Resend's HTTP API (`POST https://api.resend.com/emails`) to bypass Railway's SMTP block.

### Gumroad Ping (Webhook) Integration

Gumroad calls webhooks "**Ping**". Configuration path: *Settings → Advanced → Ping endpoint*.

**Ping URL**: `https://schedulegate-license-server-production.up.railway.app/api/v1/webhooks/gumroad?token=<SG_WEBHOOK_TOKEN>`

Key facts:
- Gumroad cannot send custom headers → auth is via `?token=` query param (constant-time compared)
- Gumroad sends form-urlencoded POST with `email` field for buyer's email
- Webhook response body is **not surfaced to the buyer** → email via Resend is the delivery mechanism
- The `POST /api/v1/mint` admin endpoint does **not** email — only webhook endpoint triggers email

### Testing Webhook End-to-End

```bash
# Test webhook (simulates Gumroad sale) — mints key AND emails it
curl -X POST "https://schedulegate-license-server-production.up.railway.app/api/v1/webhooks/gumroad?token=<SG_WEBHOOK_TOKEN>" \
  -d "email=test@example.com&sale_id=12345"

# Test admin mint (no email sent)
curl -X POST "https://schedulegate-license-server-production.up.railway.app/api/v1/mint" \
  -H "Authorization: Bearer <SG_ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"email":"buyer@example.com","tier":"pro"}'

# Check server health
curl https://schedulegate-license-server-production.up.railway.app/health

# Check recent logs
railway logs -n 20
```

### Email Provider

Resend is the configured email provider. The SMTP credentials work for both SMTP and the HTTP API:
- **SMTP** (blocked by Railway): `smtp.resend.com:587`
- **HTTP API** (used): `POST https://api.resend.com/emails` with `Authorization: Bearer <SG_SMTP_PASS>`
- Domain: `schedulegate.dev` (verified on Resend)

### Purchaser Email

The owner's test purchase email is set as a known buyer. Minted keys for this email include:
- Pro tier (1-year): via webhook path
- Lifetime tier (via admin mint endpoint, no email)

## Gotchas

- **`.gitignore` blocks all `*.csv`, `*.xlsx`, `*.xls`, `*.html`** — test fixture files and sample rules must use `!` overrides or go in non-ignored paths. Currently only `internal/reader/testdata/*.csv` and `internal/reader/testdata/*.xlsx` have overrides.
- **Reader column matching is fuzzy/case-insensitive** — `internal/reader/reader.go` uses alias maps and `normalizeString()`. Required and optional columns are defined as exported vars `RequiredColumns`/`OptionalColumns`.
- **Dependencies**: cobra, lipgloss, excelize, viper, pflag, yaml.v3.
- **Status date logic** in `cmd/assess.go:parseStatusDate()` accepts specific date formats only (MM/DD/YY is ambiguous — treated as US month-first).
- **Python scripts** in root (`_audit_random.py`, `_check_cols.py`, etc.) are audit/verification tools, not part of the Go build. They depend on `.venv/`.
- **Desktop is a separate module** — `desktop/go.mod` uses `replace github.com/gjunqueira-sys/ScheduleGate => ../`. Run `go test` from `desktop/` to test it; root `go test ./...` does not cover it.
- **License feature gating** — `compare`, `check-patterns`, `--json`, `--json-output`, `--csv`, `--exceptions-report` require a valid license key (Pro tier or above). CI smoke tests mint ephemeral keys via `license generate --key-only`.
- **⚠️ SECURITY: Never build locally with the production `LICENSE_SECRET`** — a binary with the prod secret can generate valid Pro licenses. Use `make build` (empty secret) for dev. The Makefile will warn if you try. Rotate script: `scripts/release/rotate-secret.sh`.

## Commit & Push Workflow

**When asked to commit and push to the public directory:**

1. **Scan for sensitive material** before committing:
   - API keys, secrets, tokens, passwords
   - License keys or signing secrets (especially `LICENSE_SECRET`, `GUMROAD_*`)
   - Personal information (emails, phone numbers, addresses)
   - Credentials in config files (`.env`, `*.yaml`, `*.json`)
   - Hardcoded URLs that may be internal/private

2. **Flag items for user review** if found:
   - List each sensitive file/line with explanation
   - Ask user to confirm before proceeding
   - Suggest `.gitignore` additions if appropriate

3. **Check `.gitignore` coverage**:
   - Verify sensitive patterns are covered
   - Ensure test fixtures using `!` overrides don't leak secrets

4. **Wait for explicit confirmation** before committing flagged items
