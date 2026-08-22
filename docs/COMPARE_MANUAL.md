# Schedule Comparison & Scoring Manual

This document details the mechanics, logic, and scoring formulas used by the `compare` command in the ScheduleGate.

## Overview

The comparison engine ("Delta Engine") takes two schedule files—a **Previous** version (Baseline) and a **Current** version—and benchmarks them to quantify stability and reliability.

## Command Reference

```bash
./schedulegate compare <previous_file> <current_file> [flags]
```

| Flag | Description |
|:---|:---|
| `--html <path>` | Generate an HTML comparison report (auto-opens in browser) |
| `--csv <path>` | Append comparison results to a CSV history database |
| `--detailed` | Include row-by-row task-level change detail in the HTML report |
| `--customer <name>` | Customer name for report metadata |
| `--project <id>` | Project ID for report metadata |
| `--pct-format <format>` | Percent complete scale: `"0-100"` (MS Project default) or `"fraction"` (0.0–1.0, Primavera exports) |
| `--date-locale <locale>` | Date parsing priority for ambiguous dates: `"US"` (MM/DD first, default) or `"EU"` (DD/MM first) |

## 1. Stability Score

The **Stability Score** (0-100) is a composite index derived from three weighted pillars.

### Pillar A: Schedule Stability (40% Weight)
Measures how well the schedule holds its dates.

*   **Metric**: `Finish Variance` (Difference in Finish Date between versions).
*   **Penalty Condition**: A task is penalized if its Finish Variance is **> 2.0 days**.
*   **Weighting**:
    *   Standard Task: **1.0 point**.
    *   Milestone: **2.0 points** (Milestone slippage is weighted double).
*   **Scoring Formula**:
    *   `Penalty = (Weighted Count of Slipping Tasks / Total Tasks) * 100`
    *   `Score = MAX(0, 40 - Penalty)`
    *   *Interpretation*: If 40% of your tasks slip by more than 2 days, you lose all 40 points in this pillar.

### Pillar B: Duration Reliability (30% Weight)
Measures "Task Bloat"—the tendency for tasks to expand in duration rather than finishing.

*   **Metric**: `Duration Delta` (Current Duration - Previous Duration).
*   **Penalty Condition**: A task is penalized if its duration grows by **> 10%**.
    *   *Example*: A 10-day task becoming 12 days (+20%) is penalized.
*   **Scoring Formula**:
    *   `Bloat % = (Count of Bloated Tasks / Total Tasks) * 100`
    *   `Penalty = Bloat % * 1.5`
    *   `Score = MAX(0, 30 - Penalty)`
    *   *Interpretation*: This is a harsh penalty. If **20%** of your tasks bloat, you lose all 30 points (20 * 1.5 = 30). This encourages realistic initial durations.

### Pillar C: Scope Churn (30% Weight)
Measures the volatility of the scope (tasks added or removed).

*   **Metric**: `Task Churn` (New Tasks + Deleted Tasks).
*   **Scoring Formula**:
    *   `Churn % = ((New Tasks + Deleted Tasks) / Total Tasks) * 100`
    *   `Penalty = Churn % * 2.0`
    *   `Score = MAX(0, 30 - Penalty)`
    *   *Interpretation*: High churn indicates poor planning or changing requirements. 15% churn results in a 0 score for this pillar.

---

## 2. Friction Index ("Ghost Tasks")

The **Friction Index** identifies "Ghost Tasks"—tasks that are haunting the schedule.

*   **Definition**: A **Ghost Task** is a task where:
    1.  **Planned Start Date** is in the *past* (before the status/data date).
    2.  **% Complete** is still **0%**.
*   **Meaning**: These tasks are "sliding" to the right. They should have started but haven't. They represent immediate execution bottlenecks.
*   **Botttleneck Identification**: The CLI aggregates these Ghost Tasks by their top-level **WBS** (Work Breakdown Structure) to show you exactly which phase of the project is stuck.

## 3. Change Metrics

The CLI reports raw counts for context:

*   **New Tasks**: Tasks present in Current but not Previous (matched by Task ID).
*   **Deleted Tasks**: Tasks present in Previous but not Current.
*   **Modified Tasks**: Tasks where Name, Duration, Percent Complete, or Dates have changed.
*   **Duration Bloat**: The specific count of tasks that triggered the Pillar B penalty (>10% growth).

## FAQ

**Q: Why is my Reliability Score 0?**
**A:** This usually means >20% of your tasks have increased their duration. Note that if you act on "Ghost Tasks" by simply extending their duration instead of starting them, you will improve stability (dates might hold if you extend others) but you will likely crash your Reliability score.

**Q: Does it handle renamed tasks?**
**A:** The comparison is based on **Task ID** — the value from the `Task ID` / `ID` column (the sequential row number in MS Project's left-hand column). If only a `Unique ID` column is present, that value is used instead. If you rename a task but keep the ID, it counts as "Modified". If you delete a task and add a new one with a new ID, it counts as "Deleted" + "New" (Churn).

## 4. CSV History Database (--csv)

Use `--csv` to append one summary row per comparison run to a flat CSV file (separate schema from the assess DCMA CSV). Pair with `--customer` and `--project` to track multiple programs across comparison cycles.

```bash
schedulegate compare prev.xlsx curr.xlsx \
  --customer Acme --project ST0410 --csv compare_history.csv \
  --date-locale US
```

*   **Append behavior**: If the file does not exist, a header row is written first; each subsequent run appends a new row.
*   **One row = one comparison cycle**: scores and change counts only (no task-level deltas).

| Column | Description |
| :--- | :--- |
| `Customer` | From `--customer` |
| `Project` | From `--project` |
| `Previous Schedule` | Previous file basename (no extension) |
| `Current Schedule` | Current file basename (no extension) |
| `Report Date` | Timestamp when the row was written |
| `Tool Version` | CLI version string |
| `Overall Score` | Composite stability score (0–100) |
| `Stability Score` | Pillar A (max 40) |
| `Reliability Score` | Pillar B (max 30) |
| `Scope Churn Score` | Pillar C (max 30) |
| `Total Tasks` | Unique non-summary tasks across both versions |
| `New Tasks` / `Deleted Tasks` / `Modified Tasks` / `Unchanged Tasks` | Change counts |
| `Ghost Tasks` | Ghost task count |
| `Duration Inflated Count` / `Duration Inflated Pct` | Pillar B bloat metrics |
| `Churn Pct` | `(New + Deleted) / Total Tasks * 100` |

## 5. Detailed Task Analysis (--detailed)

When using the `--detailed` flag with the `--html` option, the report includes a row-by-row breakdown of every significantly changed task.

### Symbology Legend

| Symbol | Meaning | Impact Type | Description |
| :---: | :--- | :--- | :--- |
| `⊕` | **New Task** | Scope Churn | Task was added in the new version. |
| `×` | **Deleted** | Scope Churn | Task was removed from the schedule. |
| `☒` | **Delayed** | Stability | Finish date slipped significantly (> 2 days). |
| `←` | **Pulled In** | Stability | Task is finishing earlier than planned. |
| `🐢` | **Bloated** | Reliability | Duration increased by > 10%. |
| `👻` | **Ghost Task** | Reliability | Planned start in past, 0% complete. |
| `📝` | **Modified** | Neutral | Minor changes (name, small date shifts). |
| `□` | **Unchanged** | Neutral | No significant changes. |

### How to Interpret the Table
The detailed table allows you to audit the score. If you see a lot of `☒` symbols, you know exactly which tasks are dragging down your **Stability** score. If you see many `🐢`, you know **Reliability** is suffering due to duration inflation.
