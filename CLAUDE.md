# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

A CLI tool for DCMA (Defense Contract Management Agency) 14-Point Schedule Assessments and schedule benchmarking on Microsoft Project exports. It reads Excel/CSV schedule files and produces HTML/CSV compliance reports.

## Build Commands

```bash
make build          # Build for macOS arm64 (default)
make build-windows  # Cross-compile for Windows amd64
make all            # Clean + build macOS + build Windows
make clean          # Remove bin/
go build -o bin/schedulegate  # Manual build
```

No test suite or linter is configured.

## CLI Commands

```bash
# Assess a single schedule (DCMA 14-point)
./bin/schedulegate assess schedule.xlsx

# Compare two schedule versions
./bin/schedulegate compare old.xlsx new.xlsx

# Validate column schema
./bin/schedulegate validate schedule.xlsx

# Check YAML rule compliance
./bin/schedulegate check-patterns schedule.xlsx --rules rules.yaml
```

## Architecture

### Data Flow

```
CLI input (cobra cmd/)
  → internal/reader: parse Excel/CSV → model.Schedule
  → Assessment engine (dcma/, compare/, rules/)
  → internal/report: generate HTML/CSV
  → Open in browser or save to disk
```

### Package Responsibilities

- **`cmd/`** — Cobra command definitions; each command is self-contained with its own flags
- **`internal/model/`** — `Task` and `Schedule` structs; the canonical data shapes everything flows through
- **`internal/reader/`** — File parsing for Excel and CSV with case-insensitive fuzzy column header matching; also handles `ColumnValidationResult`
- **`internal/dcma/`** — Implements the `Metric` interface for each of the 14 DCMA points; `DCMAAssessment` orchestrates them
- **`internal/compare/`** — Delta engine that matches tasks by TaskID across two schedule versions, scores via three pillars (Stability 40%, Reliability 30%, Churn 30%), and classifies impacts via `symbology.go`
- **`internal/rules/`** — YAML-driven rule engine with glob pattern matching on task fields and count/duration constraints
- **`internal/report/`** — HTML templates (dark theme) and CSV writers optimized for Power BI ingestion
- **`internal/ui/`** — Lipgloss-based terminal styling, ASCII logo, and help text

### Key Interfaces and Types

The **`Metric` interface** is the extension point for DCMA metrics:
```go
type Metric interface {
    ID() int
    Name() string
    Description() string
    Threshold() float64
    Assess(schedule *model.Schedule) MetricResult
}
```
Each metric in `internal/dcma/metrics.go` implements this independently.

**`BenchmarkResult`** (in `internal/compare/model.go`) is the output of a comparison — it holds the overall score, three pillar scores, task deltas, and friction index aggregated by WBS.

**`TaskDelta`** represents a single task's change across schedule versions, enriched by `AssignSymbology()` with visual indicators (`⊕ × ☒ ← 🐢 👻 📝 □`).

### YAML Rule Format

Rules files (`sample_rules.yaml`, `schedule_template_rules.yaml`) use this structure:
```yaml
rules:
  - name: "Rule Name"
    match:
      name: "*pattern*"          # glob, case-insensitive
      discipline: "05 - Mechanical"
    constraints:
      min_duration: 30
      max_duration: 100
    min_count: 1
    max_count: 50
```

### Scoring Logic

Documented in `docs/COMPARE_MANUAL.md`. Three-pillar weighted composite (0–100):
- **Pillar A (Stability)**: Finish variance, milestones weighted 2×
- **Pillar B (Reliability)**: Duration growth >10%, 1.5× multiplier
- **Pillar C (Churn)**: New + deleted tasks, 2× multiplier

### Column Mapping

`internal/reader/reader.go` maps Excel/CSV headers to canonical field names using case-insensitive matching and aliases. Required columns (10) and optional columns (7) are defined as exported vars `RequiredColumns` and `OptionalColumns`.
