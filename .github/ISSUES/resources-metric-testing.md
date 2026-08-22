# Issue: Resources Metric (DCMA #10) Not Tested Against Real Schedule Files

## Summary
The **Resources** metric (DCMA Metric #10) has unit tests but **no integration test** using a real schedule file with a Resources column. All existing test fixtures lack the `Resources` / `Resource Names` column, causing the metric to always be marked as **Not Applicable** during testing.

## Background
- **Original repo behavior**: The original repository did not include a Resources column in schedule exports
- **Current workaround**: The metric correctly returns `NotApplicable: true` when no resource data exists (see `internal/dcma/metrics.go:604-610`)
- **Unit tests exist**: `TestResources_NotApplicableWhenNoResourceColumn` and `TestResources_CountsAssignedTasks` in `internal/dcma/metrics_test.go:498-532`
- **Missing**: No end-to-end test with an actual `.xlsx`/`.csv` file containing resource assignments

## PAM 4.10 Specification Compliance

### Official Definition (DCMA PAM Section 4.10)
**Threshold:** ≥ 95% of incomplete work tasks must have resources assigned  
**Formula:** `Tasks with at least one resource / Total incomplete work tasks`

**Key PAM 4.10 Guidance:**
> *"Some contractors may not load their resources into the IMS. The IMS DID does not require the contractor to load resources directly into the schedule."*

### Current Implementation vs. PAM

| Requirement | PAM 4.10 | Current Implementation | Status |
|-------------|----------|----------------------|--------|
| **Universe** | Incomplete work tasks (non-summary, non-milestone) | `incompleteWorkTaskFunnel()` excludes summaries, milestones, 100% complete | ✅ Compliant |
| **Numerator** | Tasks with at least one resource assigned | Counts tasks where `Resources` field has content | ✅ Compliant |
| **Threshold** | ≥ 95% | `Threshold() = 0.95` | ✅ Compliant |
| **N/A handling** | Resource loading not required by IMS DID | Returns `NotApplicable: true` when no resource column exists | ✅ Compliant |
| **Exception detail** | List tasks missing resources | Exceptions include TaskID, Name, and condition note | ✅ Compliant |

**Implementation reference:** `internal/dcma/metrics.go:555-624`

### User Manual Documentation
Per `USER_MANUAL.md:231-240`:
- Metric correctly documented as PAM 4.10
- Threshold documented as ≥ 95%
- N/A behavior documented when resource column absent

## Current Test Coverage
```go
// internal/dcma/metrics_test.go
func TestResources_NotApplicableWhenNoResourceColumn(t *testing.T) {
    // Tests N/A behavior when Resources field is empty
}

func TestResources_CountsAssignedTasks(t *testing.T) {
    // Tests counting logic with synthetic data
}
```

**Problem**: These are unit tests with synthetic `model.Task` objects. No test file in `internal/reader/testdata/` or elsewhere contains a Resources column.

## Evidence
Test fixture headers (as of 2026-08-22):
```csv
# test_anonymous_schedule.csv
ID,Mechanical_Segment_Nbr,Control_Segment_Nbr,...,Active

# test_anonymous_schedule_v2.csv  
ID,Mechanical_Segment_Nbr,Control_Segment_Nbr,...,Active
```

**Missing column**: No `Resources`, `Resource Names`, `Resource`, or `Assigned Resources` column (see reader alias map at `internal/reader/reader.go:217`).

## Impact
1. **No regression protection**: If Resources metric logic breaks, only synthetic unit tests would catch it
2. **Reader column matching untested**: The fuzzy header matching for Resources column aliases hasn't been validated against real MS Project exports
3. **Integration gap**: No test verifies the full pipeline: `file → reader → model.Task.Resources → dcma.ResourcesMetric.Assess() → report`
4. **PAM 4.10 compliance unverified end-to-end**: While implementation matches spec, no integration test proves compliance with actual schedule data

## Required Work

### 1. Create Test Fixture with Resources Column
Generate a minimal test schedule file containing:
- ✅ Standard DCMA columns (ID, Name, Duration, Start, Finish, Predecessors, % Complete, etc.)
- ✅ **Resources column** with varied data:
  - Tasks with single resource assignments (e.g., "Alice")
  - Tasks with multiple resources (e.g., "Alice; Bob")
  - Tasks with no resources (empty string)
  - Mix of complete/incomplete tasks (to verify incomplete-only universe)
  - Mix of work tasks, milestones, summaries (to verify exclusions)
  - At least 20 incomplete work tasks to test 95% threshold boundary

**File location**: `internal/reader/testdata/test_resources_schedule.xlsx` (or `.csv`)

### 2. Add Integration Test
```go
// internal/dcma/integration_test.go (or metrics_test.go)
func TestResourcesMetric_Integration(t *testing.T) {
    // Load test_anonymous_schedule_resources.xlsx
    // Verify Resources metric:
    //   - Correctly counts tasks with resources (PAM 4.10 numerator)
    //   - Correctly identifies exceptions (tasks missing resources)
    //   - Produces expected Value and Passing status (≥95% threshold)
    //   - Exceptions report contains proper TaskID + Name
    //   - Universe excludes summaries, milestones, 100% complete (PAM 4.10 compliance)
}
```

### 3. Update CI Smoke Tests
Ensure `cmd/genfixture/` generates schedules with Resources column for:
- ✅ `assess` command smoke test
- ✅ `--exceptions-report` smoke test (verify Resources exceptions appear in Excel output)

### 4. Verify Reader Column Matching
Test that reader correctly maps these column header variations (per `internal/reader/reader.go:217`):
- `Resource Names` (canonical MS Project)
- `Resources`
- `Resource`
- `Assigned Resources`

Add test cases to `internal/reader/reader_test.go` if not present.

### 5. PAM 4.10 Compliance Verification
Document in test comments how the integration test validates:
- ✅ Incomplete work task universe (excludes summaries, milestones, completed)
- ✅ 95% threshold enforcement
- ✅ N/A behavior when resource column absent
- ✅ Exception detail for tasks missing resources

## Acceptance Criteria
- [ ] Test fixture file exists with Resources column
- [ ] Integration test passes with Resources metric showing realistic pass/fail
- [ ] Exceptions report includes Resources metric exceptions when applicable
- [ ] CI smoke tests include Resources in generated fixtures
- [ ] No change to existing N/A behavior when Resources column is absent
- [ ] Test explicitly validates PAM 4.10 compliance (universe, threshold, N/A handling)

## Related Files
- `internal/dcma/metrics.go:555-624` — ResourcesMetric implementation
- `internal/dcma/metrics_test.go:498-532` — Unit tests
- `internal/reader/reader.go:217` — Resources column alias map
- `internal/reader/reader.go:573` — Task.Resources field population
- `internal/model/schedule.go:23` — Task.Resources field definition
- `cmd/genfixture/` — CI fixture generator
- `USER_MANUAL.md:231-240` — PAM 4.10 documentation

## Priority
**Medium** — Metric logic is covered by unit tests, but integration gap leaves room for undetected regressions in:
- Reader column matching
- End-to-end data flow
- Exceptions report generation for Resources metric
- PAM 4.10 compliance verification with real schedule data

## Notes
- This issue was identified during a code review on 2026-08-22
- Original repo bypassed this metric as N/A due to missing Resources column
- Current implementation handles N/A case correctly, but lacks positive test coverage
- **PAM 4.10 compliance verified**: Implementation matches DCMA specification (universe, threshold, N/A handling)
