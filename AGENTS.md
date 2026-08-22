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

Automated release pipeline invoked via `/release-pipeline` slash command. Builds 4 binaries (CLI x3 + Desktop GUI), packages into zip, creates GitHub Release, and updates Gumroad product via browser automation.

```bash
/release-pipeline                    # Auto-detect changes, auto-increment patch
/release-pipeline --force            # Force release even without product changes
/release-pipeline --version 1.1.0    # Manual version override
/release-pipeline --dry-run          # Build + test only, no publish
```

**Release scripts** (in `scripts/release/`):
- `version.sh` — Auto-increment semver patch from git tags
- `release-notes.sh` — Generate changelog from commits since last tag
- `zip-binaries.sh` — Package binaries + README + LICENSE into zip
- `gumroad/main.go` — Gumroad browser automation helper

**Gumroad credentials**: Set `GUMROAD_EMAIL`, `GUMROAD_PASSWORD`, `GUMROAD_PRODUCT_URL` as environment variables or GitHub Secrets. Script prompts if not set.

**Zip contents**: `schedulegate` (macOS), `schedulegate.exe` (Windows), `schedulegate-linux` (Linux), `schedulegate-gui` (Desktop GUI), `README.txt`, `LICENSE`, `user-manual.html`

**Manual freshness gate**: `cmd/manualcheck` verifies `docs/user-manual.html` matches the CLI surface (commands, flags, DCMA metric thresholds). Runs in release pipeline before packaging; `go test ./cmd/manualcheck/...` enforces it in regular CI.

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

## Gotchas

- **`.gitignore` blocks all `*.csv`, `*.xlsx`, `*.xls`, `*.html`** — test fixture files and sample rules must use `!` overrides or go in non-ignored paths. Currently only `internal/reader/testdata/*.csv` and `internal/reader/testdata/*.xlsx` have overrides.
- **Reader column matching is fuzzy/case-insensitive** — `internal/reader/reader.go` uses alias maps and `normalizeString()`. Required and optional columns are defined as exported vars `RequiredColumns`/`OptionalColumns`.
- **Dependencies**: cobra, lipgloss, excelize, viper, pflag, yaml.v3.
- **Status date logic** in `cmd/assess.go:parseStatusDate()` accepts specific date formats only (MM/DD/YY is ambiguous — treated as US month-first).
- **Python scripts** in root (`_audit_random.py`, `_check_cols.py`, etc.) are audit/verification tools, not part of the Go build. They depend on `.venv/`.
- **Desktop is a separate module** — `desktop/go.mod` uses `replace github.com/gjunqueira-sys/ScheduleGate => ../`. Run `go test` from `desktop/` to test it; root `go test ./...` does not cover it.
- **License feature gating** — `compare`, `check-patterns`, `--json`, `--json-output`, `--csv`, `--exceptions-report` require a valid license key (Pro tier or above). CI smoke tests mint ephemeral keys via `license generate --key-only`.
