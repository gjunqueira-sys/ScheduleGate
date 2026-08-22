# ScheduleGate

```text
 _____ _____  _   _  ___________ _   _ _      _____ _____   ___ _____ _____
/  ___/  __ \| | | ||  ___|  _  \ | | | |    |  ___|  __ \ / _ \_   _|  ___|
\ `--. | /  \/| |_| || |__ | | | | | | | |    | |__ | |  \// /_\ \| | | |__
 `--. \ |    |  _  ||  __|| | | | | | | |    |  __|| | __ |  _  || | |  __|
/\__/ / \__/\| | | || |___| |/ /| |_| | |____| |___| |_\ \| | | || | | |___
\____/ \____/\_| |_/\____/|___/  \___/\_____/\____/ \____/\_| |_/\_/ \____/
                        Schedule Quality. Quantified.
```

A robust Go-based Command Line Interface (CLI) tool for performing **DCMA 14-Point Assessments** and **Schedule Comparison**. This tool helps project managers and schedulers evaluate the health, quality, and stability of their Integrated Master Schedules (IMS).

## Features

-   **DCMA 14-Point Assessment**: Automatically calculates all 14 standard metrics.
-   **Schedule Comparison**: Benchmark two schedule versions to track changes, stability, and churn.
-   **Visual Reports**: Modern, high-contrast terminal output with score cards.
-   **Export Capabilities**: Generate HTML reports or append results to a CSV database for PowerBI.
-   **Innovation Metrics**: Includes unique "Stability Score" and "Friction Index" to identify bottlenecks.

## Installation

### Prerequisites
-   Go 1.20 or higher

### Building from Source

To build the CLI for your current platform (macOS/Linux):

```bash
go build -o bin/schedulegate
```

### Building for Windows

You can use the provided `Makefile` to properly cross-compile the application for Windows from macOS or Linux.

1.  **Run the make command:**
    ```bash
    make build-windows
    ```
2.  The executable will be generated at `bin/schedulegate.exe`.

### Using Make

The project includes a `Makefile` for convenience:

*   `make build`: Builds for the current environment (default: macOS).
*   `make build-windows`: Cross-compiles for Windows (amd64).
*   `make build-mac`: Builds for macOS (arm64).
*   `make clean`: Removes the `bin` directory.

## Usage

The CLI has four primary commands: `assess`, `compare`, `validate`, and `check-patterns`.

### 1. Check Patterns (Compliance)

Validate schedule tasks against configurable pattern rules.

```bash
./schedulegate check-patterns <schedule-file> --rules <rules.yaml> [flags]
```

**Flags:**
-   `--rules <path>`: Path to YAML rules file (required).
-   `--html <path>`: Generate an HTML compliance report.
-   `--csv <path>`: Append results to a CSV file.
-   `--detailed`: Show matching task names for each rule.
-   `--pct-format <format>`: Percent complete scale — `"0-100"` (default) or `"fraction"`.
-   `--date-locale <locale>`: Date parsing — `"US"` (default) or `"EU"`.

**Rules File Format:**
```yaml
rules:
  - name: "Mechanical Order Entry Tasks"
    match:
      name: "*Order Entry*"              # glob pattern on task name
      discipline: "05 - Mechanical Engineering"  # filter by discipline
    min_count: 1

  - name: "Controls Drawings to Install"
    match:
      name: "*Drawings to Install*"
      discipline: "06 - Controls Engineering"
    min_count: 1

  - name: "Segment 5700 Tasks"
    match:
      mechanical_segment_nbr: "5700"     # filter by segment number
    min_count: 1

  - name: "Long duration items"
    match:
      name: "*"
    constraints:
      min_duration: 30                   # tasks >= 30 days
    min_count: 1
```

**Available Match Fields:**
- `name` / `task_name`: Task name (glob patterns supported)
- `discipline` / `task_discipline`: Task discipline category
- `mechanical_segment_nbr` / `mech_segment`: Mechanical segment number
- `control_segment_nbr` / `control_segment`: Control segment number
- `wbs`: WBS code
- `resources`: Assigned resources
- `constraint_type`: Constraint type

> **Note:** Pattern matching is **case-insensitive**. The pattern `*mechanical*` will match "Mechanical", "MECHANICAL", "mechanical", etc.

**Example:**
```bash
./schedulegate check-patterns schedule.xlsx --rules rules.yaml --html compliance.html
```

### 2. Validate Columns

Check if a schedule file contains the expected columns before running analysis.

```bash
./schedulegate validate <path-to-file> [flags]
```

**Flags:**
-   `--html <path>`: Generate an HTML validation report.
-   `--csv <path>`: Append validation results to a CSV file.

**Output:**
-   **Status**: `READY` (all required columns found) or `INCOMPLETE` (missing required columns).
-   **Required Columns**: 10 columns needed for core functionality.
-   **Optional Columns**: 7 additional columns that enhance analysis.
-   **Extra Columns**: Unmapped columns present in your file.

**Example:**
```bash
./schedulegate validate my_schedule.xlsx --html validation.html
```

### 3. Assess Schedule

Analyze a single schedule file for DCMA compliance.

```bash
./schedulegate assess <path-to-file> [flags]
```

**Flags:**
-   `-m, --metrics`: Comma-separated list of metric IDs to run (1-14). Defaults to all.
-   `--html <path>`: Generate a rich HTML report.
-   `--csv <path>`: Append results to a CSV database (optimized for PowerBI).
-   `--exceptions-report <path>`: Generate an Excel workbook with per-metric exception sheets.
-   `--customer <name>`: Add customer name to report metadata.
-   `--project <id>`: Add project ID to report metadata.
-   `--status-date <date>`: Override the schedule status date (default: today).
-   `--debug-logic`: Print per-task successor resolution trace for Metric 1.
-   `--pct-format <format>`: Percent complete scale — `"0-100"` (MS Project default) or `"fraction"` (0.0–1.0 from Primavera).
-   `--date-locale <locale>`: Date parsing priority — `"US"` (MM/DD first, default) or `"EU"` (DD/MM first).
-   `-v, --verbose`: Show raw numerator/denominator counts per metric.

**Example:**
```bash
./schedulegate assess schedule_v1.xlsx --html report.html --csv history.csv --status-date 2026-04-10
```

### 4. Compare Schedules (Benchmark)

Compare two versions of a schedule to understand what changed ("Delta Engine").

```bash
./schedulegate compare <previous_file> <current_file>
```

**Output Includes:**
-   **Stability Score (0-100)**: a weighted index based on three pillars:
    -   **Schedule Stability (40%)**: Penalizes finish date variances > 2 days.
    -   **Duration Reliability (30%)**: Penalizes duration growth ("bloat").
    -   **Scope Churn (30%)**: Penalizes added and deleted tasks.
-   **Change Metrics**: Counts of New, Deleted, and Modified tasks.
-   **Friction Index**: Identifies "Ghost Tasks" (tasks scheduled in the past with 0% progress) and ranks them by WBS to pinpoint bottlenecks.
-   **Detailed Task Analysis** (with `--detailed`): A comprehensive table showing exactly which tasks changed, with visual symbology for impact type (Stability, Reliability, Scope).

**Flags:**
-   `--html <path>`: Generate an HTML comparison report.
-   `--detailed`: Include row-by-row task-level changes in the HTML report.
-   `--csv <path>`: Append results to a comparison history CSV.
-   `--customer <name>`: Add customer name to report metadata.
-   `--project <id>`: Add project ID to report metadata.
-   `--pct-format <format>`: Percent complete scale — `"0-100"` (default) or `"fraction"`.
-   `--date-locale <locale>`: Date parsing — `"US"` (default) or `"EU"`.

**Example:**
```bash
./schedulegate compare baseline.xlsx current_progress.xlsx --html report.html --detailed
```
(The report will automatically open in your default browser)

## Supported Formats

The tool reads schedule data exported from Microsoft Project or other scheduling tools.

-   **Excel (.xlsx, .xls)**: Recommended.
-   **CSV (.csv)**: Comma-separated values.

### Required Fields
Ensure your export includes these columns (case-insensitive detection):

**Core Fields (Required — 7 columns):**
-   **ID**: `Task ID`, `ID` — the sequential row number visible in MS Project's left-hand column. If only a `Unique ID` column is present it is used as a fallback, but `Task ID` is always preferred so report values match the source schedule.
-   **Name**: `Task Name`, `Name`
-   **Duration**: `Duration`, `Original Duration`
-   **Start**: `Start`, `Start Date`
-   **Finish**: `Finish`, `Finish Date`
-   **Predecessors**: `Predecessors`, `Pred`
-   **% Complete**: `% Complete`, `Percent Complete`

**Optional Fields (Enhance Analysis):**
-   **Resources**: `Resource Names`, `Resources` — enables Metric 10
-   **WBS**: `WBS`, `WBS Code` — enables WBS grouping and Friction Index
-   **Constraint Type**: `Constraint Type` — enables Metric 5 (Hard Constraints)
-   **Constraint Date**: `Constraint Date`
-   **Total Slack**: `Total Slack`, `Total Float` — enables Metrics 6, 7, 12, 13
-   **Free Slack**: `Free Slack`
-   **Baseline Duration**: `Baseline_Duration`, `BL Duration` — enables Metric 8
-   **Baseline Start / Finish**: `Baseline_Start`, `BL Start` / `Baseline_Finish`, `BL Finish` — enables Metrics 11, 14
-   **Actual Start / Finish**: `Actual_Start`, `Actual Start` / `Actual_Finish`, `Actual Finish` — enables Metric 9
-   **Discipline**: `Task Discipline`, `Discipline` — grouping and reporting
-   **Mechanical Segment**: `Mechanical_Segment_Nbr`, `Mechanical Segment Nbr` — grouping and reporting
-   **Control Segment**: `Control_Segment_Nbr`, `Control Segment Nbr` — grouping and reporting
-   **Unique ID**: `Unique ID`, `UniqueID` — fallback task identifier
-   **Outline Level**: `Outline Level`, `Level`
-   **Summary / Rollup**: `Summary`, `Rollup` — summary task classification

## Metrics Explained

### DCMA 14-Point Assessment
1.  **Logic**: Tasks with missing predecessors or successors.
2.  **Leads**: Relationships with negative lag.
3.  **Lags**: Relationships with positive lag.
4.  **Relationship Types**: Finish-to-Start (FS) relationships.
5.  **Hard Constraints**: Constraints preventing logic flow (e.g., Must Finish On).
6.  **High Float**: Tasks with total slack > 60 working days.
7.  **Negative Float**: Tasks with negative total slack.
8.  **High Duration**: Tasks with duration > 60 working days.
9.  **Invalid Dates**: Tasks with invalid/missing dates.
10. **Resources**: Tasks with resources assigned.
11. **Missed Tasks**: Tasks passed status date but not finished.
12. **Critical Path Test**: Pass/Fail test for broken critical path.
13. **CPLI**: Critical Path Length Index.
14. **BEI**: Baseline Execution Index.

### Assessment Thresholds

| # | Metric | Threshold | Pass When |
|---|--------|-----------|-----------|
| 1 | Logic | 5% | < 5% tasks with missing logic |
| 2 | Leads | 0% | Zero leads (negative lag) allowed |
| 3 | Lags | 10% | < 10% relationships have lag |
| 4 | Relationship Types | ≥ 90% FS | ≥ 90% Finish-to-Start |
| 5 | Hard Constraints | < 5% | < 5% hard-constrained tasks |
| 6 | High Float | < 5% | < 5% tasks with slack > 44 working days |
| 7 | Negative Float | < 5% | < 5% tasks with negative float |
| 8 | High Duration | < 10% | < 10% tasks with duration > 60 working days |
| 9 | Invalid Dates | 0% | Zero invalid dates allowed |
| 10 | Resources | ≥ 95% | ≥ 95% tasks have resources |
| 11 | Missed Tasks | < 5% | < 5% of due tasks are incomplete |
| 12 | Critical Path Test | Pass/Fail | Critical path exists (static proxy) |
| 13 | CPLI | ≥ 1.0 | CPLI ≥ 1.0 |
| 14 | BEI | ≥ 0.95 | BEI ≥ 0.95 |

For full metric definitions see [USER_MANUAL.md](USER_MANUAL.md) or [docs/assess-manual.html](docs/assess-manual.html).

### Benchmarking Metrics
For a deep dive into how scores are calculated, see the [Comparison Manual](docs/COMPARE_MANUAL.md).

-   **Finish Variance (FV)**: Change in finish date between versions.
-   **Task Bloat**: Increase in duration between versions (>10% growth triggers penalty).
-   **Task Churn**: Ratio of added/deleted tasks to total tasks.
-   **Ghost Tasks**: Tasks that are "sliding" (Start date in past, 0% complete).
