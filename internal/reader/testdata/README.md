# Test Fixtures

This directory contains test fixtures for the ScheduleGate reader package.

## Anonymous Schedule Fixtures

### test_anonymous_schedule.csv
- **Size**: 6,146 tasks
- **Purpose**: Large-scale integration testing of DCMA 14-point assessment
- **Status Date**: 2026-04-10 (for consistent test results)
- **Known Results**: 46% overall score (6/13 metrics pass)

### test_anonymous_schedule_v2.csv
- **Size**: 6,146 tasks (same structure as v1)
- **Purpose**: Comparison testing (schedule benchmark)
- **Modifications**: Durations increased by ~10% from v1

### test_resources_schedule.csv
- **Size**: 29 tasks
- **Purpose**: Integration testing of DCMA Resources metric (PAM 4.10)
- **Features**:
  - Contains "Resource Names" column with varied assignments
  - Tasks with single resources (e.g., "Alice")
  - Tasks with multiple resources (e.g., "Alice; Bob")
  - Tasks with no resources (empty string)
  - Mix of complete/incomplete tasks
  - Mix of work tasks, milestones, summaries
  - Inactive tasks (Active=No)
- **Status Date**: 2026-04-10
- **Tests**: Used by `TestResources_Integration` and `TestResourcesColumnMatching`

## Anonymization

These fixtures are completely anonymous - no references to:
- Ross
- Dematic
- Customer names
- Project-specific identifiers

All task names, disciplines, and WBS codes have been sanitized while preserving structural data (dates, durations, predecessors, slack values) required for accurate DCMA metric calculations.

## Usage in CI

The fixtures are used in the CI/CD pipeline (`.github/workflows/build.yml`) smoke tests:

```bash
./schedulegate assess internal/reader/testdata/test_anonymous_schedule.csv \
  --status-date 2026-04-10 \
  --json-output report.json
```

## Regenerating Fixtures

To regenerate these fixtures from a real schedule export:

```bash
python3 scripts/sanitize_schedule.py input.csv internal/reader/testdata/test_anonymous_schedule.csv
```

Ensure all identifying information is removed before committing.
