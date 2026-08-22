package dcma

import (
	"fmt"
	"testing"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func date(year, month, day int) *time.Time {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &t
}

func newSchedule(dataDate time.Time, tasks ...*model.Task) *model.Schedule {
	return &model.Schedule{
		Name:     "test",
		DataDate: dataDate,
		Tasks:    tasks,
	}
}

func workTask(id string, pct float64, bf *time.Time) *model.Task {
	return &model.Task{
		TaskID:         id,
		Name:           "Task " + id,
		Duration:       5,
		IsMilestone:    false,
		IsSummary:      false,
		PercentComplete: pct,
		BaselineFinish: bf,
	}
}

func summaryTask(id string) *model.Task {
	return &model.Task{TaskID: id, Name: "Summary " + id, IsSummary: true, Duration: 10}
}

func milestoneTask(id string, pct float64) *model.Task {
	return &model.Task{TaskID: id, Name: "Milestone " + id, IsMilestone: true, Duration: 0, PercentComplete: pct}
}

func milestoneTaskBF(id string, pct float64, bf *time.Time) *model.Task {
	return &model.Task{TaskID: id, Name: "Milestone " + id, IsMilestone: true, Duration: 0, PercentComplete: pct, BaselineFinish: bf}
}

// ---------------------------------------------------------------------------
// BEI — Baseline Execution Index
// ---------------------------------------------------------------------------
// Formula (DAU APMT-009):
//   BEI = all completed work tasks / work tasks with BaselineFinish ≤ status date
//
// Key invariants tested:
//  1. Numerator = ALL completed tasks, work + milestones (not just those with BF due)
//  2. Denominator = only tasks with BF ≤ status date (no "missing baseline" penalty)
//  3. Tasks with nil BaselineFinish are excluded from both numerator and denominator
//  4. Summaries are excluded; milestones are part of the universe (contractual gates)

func TestBEI_StandardCase(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	tasks := []*model.Task{
		workTask("1", 100, date(2026, 3, 1)),          // complete, BF before dd → in denom
		workTask("2", 100, date(2026, 4, 5)),          // complete, BF before dd → in denom
		workTask("3", 0, date(2026, 4, 5)),            // incomplete, BF before dd → in denom
		workTask("4", 100, date(2026, 5, 1)),          // complete, BF AFTER dd → not in denom
		workTask("5", 0, date(2026, 5, 1)),            // incomplete, BF after dd → not in denom
		workTask("6", 0, nil),                         // no BF → excluded from both
		summaryTask("s1"),                             // summary → excluded
		milestoneTaskBF("m1", 100, date(2026, 3, 15)), // milestone, BF before dd → in both (contractual gate)
	}

	s := newSchedule(dd, tasks...)
	result := (&BEIMetric{}).Assess(s)

	// completed = 4 (tasks 1, 2, 4, m1 — ALL completed, regardless of when BF falls)
	// denominator = 4 (tasks 1, 2, 3, m1 — BF ≤ dd)
	// BEI = 4/4 = 1.0
	if result.Details["completed"] != 4 {
		t.Errorf("BEI completed = %v, want 4", result.Details["completed"])
	}
	if result.Details["baseline_count"] != 4 {
		t.Errorf("BEI baseline_count = %v, want 4", result.Details["baseline_count"])
	}
	wantVal := 1.0
	if result.Value != wantVal {
		t.Errorf("BEI value = %.4f, want %.4f", result.Value, wantVal)
	}
	if !result.Passing {
		t.Error("BEI should PASS at 100%")
	}
}

func TestBEI_NoMissingBaselinePenalty(t *testing.T) {
	// Before the fix: denominator = BF_due + missing_BF (penalty).
	// After the fix:  denominator = BF_due only.
	// With 175 unbaselined tasks the old formula drove BEI to ~75%; the fixed
	// formula correctly ignores them.
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	tasks := []*model.Task{
		workTask("1", 100, date(2026, 3, 1)), // complete, BF due
		workTask("2", 100, date(2026, 4, 5)), // complete, BF due
		workTask("3", 0, date(2026, 4, 8)),   // incomplete, BF due
	}
	// Add 10 tasks with no baseline — must NOT affect the ratio
	for i := 4; i < 14; i++ {
		tasks = append(tasks, workTask(string(rune('0'+i)), 0, nil))
	}

	s := newSchedule(dd, tasks...)
	result := (&BEIMetric{}).Assess(s)

	if result.Details["missing_baseline"] != 10 {
		t.Errorf("missing_baseline = %v, want 10", result.Details["missing_baseline"])
	}
	// BEI = 2/3, not 2/13
	wantVal := 2.0 / 3.0
	if result.Value < wantVal-0.001 || result.Value > wantVal+0.001 {
		t.Errorf("BEI value = %.4f, want %.4f — missing-baseline penalty must not affect the ratio", result.Value, wantVal)
	}
}

func TestBEI_CompletedCountsAllCompleted(t *testing.T) {
	// Numerator must include a completed task even when its BF is AFTER the
	// status date (it finished early). Before the inner fix, such a task would
	// not be counted because the completed++ was inside the BF≤dd branch.
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 100, date(2026, 4, 8)),  // complete, BF before dd → in denom
		workTask("2", 100, date(2026, 6, 1)),  // complete, BF AFTER dd → not in denom, but IS complete
		workTask("3", 0, date(2026, 4, 9)),    // incomplete, BF before dd
	}
	s := newSchedule(dd, tasks...)
	result := (&BEIMetric{}).Assess(s)

	// completed = 2 (tasks 1 and 2)
	// baseline_count (denom) = 2 (tasks 1 and 3, BF ≤ dd)
	// BEI = 2/2 = 1.0
	if result.Details["completed"] != 2 {
		t.Errorf("BEI completed = %v, want 2 (task finished early must count)", result.Details["completed"])
	}
	if result.Details["baseline_count"] != 2 {
		t.Errorf("BEI baseline_count = %v, want 2", result.Details["baseline_count"])
	}
}

func TestBEI_NoBaselinedTasksDue(t *testing.T) {
	// No tasks with BF ≤ status date → denominator = 0 → return 1.0 (pass)
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 0, date(2026, 5, 1)),
		workTask("2", 0, date(2026, 6, 1)),
	}
	s := newSchedule(dd, tasks...)
	result := (&BEIMetric{}).Assess(s)
	if result.Value != 1.0 || !result.Passing {
		t.Errorf("BEI with no due tasks should return 1.0 PASS, got %.2f passing=%v", result.Value, result.Passing)
	}
}

func TestBEI_ExcludesSummariesButIncludesMilestones(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 100, date(2026, 3, 1)),
		summaryTask("s1"),
		milestoneTaskBF("m1", 100, date(2026, 3, 20)),
	}
	// task 1 and milestone m1 both count — 2 completed / 2 due = 100%
	s := newSchedule(dd, tasks...)
	result := (&BEIMetric{}).Assess(s)
	if result.Details["completed"] != 2 {
		t.Errorf("BEI should exclude summaries but include milestones in completed count; got %v", result.Details["completed"])
	}
	if result.Details["baseline_count"] != 2 {
		t.Errorf("BEI should include due milestones in the denominator; got %v", result.Details["baseline_count"])
	}
}

// ---------------------------------------------------------------------------
// Missed Tasks (Late to Baseline Finish)
// ---------------------------------------------------------------------------

func TestMissedTasks_Basic(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 0, date(2026, 4, 5)),   // BF before dd, incomplete → LATE
		workTask("2", 100, date(2026, 4, 5)), // BF before dd, complete → not late
		workTask("3", 0, date(2026, 5, 1)),   // BF after dd → not in denominator
		workTask("4", 0, nil),                // no BF → not in denominator
		summaryTask("s1"),
		milestoneTask("m1", 0),
	}
	s := newSchedule(dd, tasks...)
	result := (&MissedTasksMetric{}).Assess(s)

	// denominator = 2 (tasks 1 and 2, BF ≤ dd)
	// count (late) = 1 (task 1)
	if result.Details["count"] != 1 {
		t.Errorf("MissedTasks count = %v, want 1", result.Details["count"])
	}
	if result.Details["total"] != 2 {
		t.Errorf("MissedTasks total = %v, want 2", result.Details["total"])
	}
	wantVal := 0.5
	if result.Value < wantVal-0.001 || result.Value > wantVal+0.001 {
		t.Errorf("MissedTasks value = %.4f, want %.4f", result.Value, wantVal)
	}
}

func TestMissedTasks_AllOnTime(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 100, date(2026, 3, 1)),
		workTask("2", 100, date(2026, 4, 5)),
	}
	s := newSchedule(dd, tasks...)
	result := (&MissedTasksMetric{}).Assess(s)
	if result.Value != 0 || !result.Passing {
		t.Errorf("MissedTasks with all complete should be 0%% PASS, got %.2f passing=%v", result.Value, result.Passing)
	}
}

func TestMissedTasks_NoDueTasks(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		workTask("1", 0, date(2026, 5, 1)),
		workTask("2", 0, nil),
	}
	s := newSchedule(dd, tasks...)
	result := (&MissedTasksMetric{}).Assess(s)
	if result.Value != 0 || !result.Passing {
		t.Errorf("MissedTasks with no due tasks should return 0 PASS")
	}
}

// ---------------------------------------------------------------------------
// Logic (Missing Links)
// ---------------------------------------------------------------------------

func TestLogic_MissingBothEnds(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	// Task 1: has predecessor 2, and is referenced by task 2 as pred → has successor
	// Task 2: no predecessor, referenced as pred by task 1's predecessor chain
	// Task 3: isolated — no predecessor, no successor
	tasks := []*model.Task{
		{TaskID: "1", Name: "T1", Duration: 5, Predecessors: "2", PercentComplete: 0, IsSummary: false, IsMilestone: false},
		{TaskID: "2", Name: "T2", Duration: 5, Predecessors: "", PercentComplete: 0, IsSummary: false, IsMilestone: false},
		{TaskID: "3", Name: "T3", Duration: 5, Predecessors: "", PercentComplete: 0, IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)

	// Task 1: has pred ✓, has successor (nobody lists 1 as pred) ✗ → missing
	// Task 2: no pred ✗ → missing (also has successor via task 1)
	// Task 3: no pred ✗, no successor ✗ → missing
	// missing = 3 out of 3 incomplete work tasks
	if result.Details["total"] != 3 {
		t.Errorf("Logic total = %v, want 3", result.Details["total"])
	}
	if result.Details["missing"] == 0 {
		t.Error("Logic missing should be > 0 for isolated tasks")
	}
}

func TestLogic_CompleteTasksExcluded(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Name: "T1", Duration: 5, Predecessors: "", PercentComplete: 100, IsSummary: false, IsMilestone: false},
		{TaskID: "2", Name: "T2", Duration: 5, Predecessors: "1", PercentComplete: 0, IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)
	// Only task 2 is incomplete
	if result.Details["total"] != 1 {
		t.Errorf("Logic total = %v, want 1 (complete tasks must be excluded)", result.Details["total"])
	}
}

// ---------------------------------------------------------------------------
// Hard Constraints
// ---------------------------------------------------------------------------

func TestHardConstraints_HardTypes(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	hardTypes := []string{"Must Finish On", "Must Start On", "Start No Later Than", "Finish No Later Than"}
	softTypes := []string{"As Soon As Possible", "Start No Earlier Than", "Finish No Earlier Than"}

	for _, ct := range hardTypes {
		t.Run(ct, func(t *testing.T) {
			tasks := []*model.Task{
				{TaskID: "1", Name: "T", Duration: 5, PercentComplete: 0, ConstraintType: ct, IsSummary: false, IsMilestone: false},
			}
			s := newSchedule(dd, tasks...)
			result := (&HardConstraintsMetric{}).Assess(s)
			if result.Details["count"] != 1 {
				t.Errorf("HardConstraints: constraint type %q should be counted as hard", ct)
			}
		})
	}

	for _, ct := range softTypes {
		t.Run(ct+"_soft", func(t *testing.T) {
			tasks := []*model.Task{
				{TaskID: "1", Name: "T", Duration: 5, PercentComplete: 0, ConstraintType: ct, IsSummary: false, IsMilestone: false},
			}
			s := newSchedule(dd, tasks...)
			result := (&HardConstraintsMetric{}).Assess(s)
			if result.Details["count"] != 0 {
				t.Errorf("HardConstraints: constraint type %q should NOT be counted as hard", ct)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// High Float
// ---------------------------------------------------------------------------

func TestHighFloat_Threshold(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 45, IsSummary: false, IsMilestone: false},  // > 44 → high
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 44, IsSummary: false, IsMilestone: false},  // = 44 → not high
		{TaskID: "3", Duration: 5, PercentComplete: 0, TotalSlack: 10, IsSummary: false, IsMilestone: false},  // < 44 → not high
		{TaskID: "4", Duration: 5, PercentComplete: 0, TotalSlack: 100, IsSummary: false, IsMilestone: false}, // > 44 → high
		{TaskID: "5", Duration: 5, PercentComplete: 100, TotalSlack: 100, IsSummary: false, IsMilestone: false}, // complete → excluded
	}
	s := newSchedule(dd, tasks...)
	result := (&HighFloatMetric{}).Assess(s)

	if result.Details["count"] != 2 {
		t.Errorf("HighFloat count = %v, want 2 (tasks with slack > 44)", result.Details["count"])
	}
	if result.Details["total"] != 4 {
		t.Errorf("HighFloat total = %v, want 4 (incomplete work tasks)", result.Details["total"])
	}
}

// ---------------------------------------------------------------------------
// Negative Float
// ---------------------------------------------------------------------------

func TestNegativeFloat(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: -1, IsSummary: false, IsMilestone: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 0, IsSummary: false, IsMilestone: false},
		{TaskID: "3", Duration: 5, PercentComplete: 0, TotalSlack: 5, IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&NegativeFloatMetric{}).Assess(s)

	if result.Details["count"] != 1 {
		t.Errorf("NegativeFloat count = %v, want 1", result.Details["count"])
	}
	if result.Passing {
		t.Error("NegativeFloat should FAIL when any task has negative float")
	}
}

func TestNegativeFloat_ZeroIsNotNegative(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 0, IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&NegativeFloatMetric{}).Assess(s)
	if result.Details["count"] != 0 {
		t.Error("Zero slack must not be counted as negative float")
	}
}

// ---------------------------------------------------------------------------
// Leads (Negative Lag)
// ---------------------------------------------------------------------------

func TestLeads_Detection(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "1FS-3d", IsSummary: false, IsMilestone: false}, // lead
		{TaskID: "3", Duration: 5, PercentComplete: 0, Predecessors: "1FS+5d", IsSummary: false, IsMilestone: false}, // lag, not lead
		{TaskID: "4", Duration: 5, PercentComplete: 0, Predecessors: "1FS", IsSummary: false, IsMilestone: false},    // no lag
		{TaskID: "5", Duration: 5, PercentComplete: 0, Predecessors: "1", IsSummary: false, IsMilestone: false},      // no lag (implicit FS)
	}
	s := newSchedule(dd, tasks...)
	result := (&LeadsMetric{}).Assess(s)

	if result.Details["leads"] != 1 {
		t.Errorf("Leads count = %v, want 1", result.Details["leads"])
	}
	if result.Details["total"] != 4 {
		t.Errorf("Leads total relationships = %v, want 4", result.Details["total"])
	}
}

func TestLeads_ZeroTolerance(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, Predecessors: "2FS-1d", IsSummary: false, IsMilestone: false}, // 1 lead
	}
	s := newSchedule(dd, tasks...)
	result := (&LeadsMetric{}).Assess(s)

	if result.Passing {
		t.Error("Leads should FAIL with zero tolerance when any lead exists")
	}
	if result.Value != 1.0 {
		t.Errorf("Leads value = %v, want 1.0 (100%% of relationships have leads)", result.Value)
	}
}

func TestLeads_Passing(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, Predecessors: "1FS", IsSummary: false, IsMilestone: false},    // no lag
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "2FS+5d", IsSummary: false, IsMilestone: false}, // lag only
	}
	s := newSchedule(dd, tasks...)
	result := (&LeadsMetric{}).Assess(s)

	if !result.Passing {
		t.Error("Leads should PASS when no leads exist (zero tolerance)")
	}
	if result.Details["leads"] != 0 {
		t.Errorf("Leads count = %v, want 0", result.Details["leads"])
	}
}

// ---------------------------------------------------------------------------
// Lags (Positive Lag)
// ---------------------------------------------------------------------------

func TestLags_Detection(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "1FS+5d", IsSummary: false, IsMilestone: false},  // lag
		{TaskID: "3", Duration: 5, PercentComplete: 0, Predecessors: "1FS+10d", IsSummary: false, IsMilestone: false}, // lag
		{TaskID: "4", Duration: 5, PercentComplete: 0, Predecessors: "1FS-3d", IsSummary: false, IsMilestone: false},  // lead, not lag
		{TaskID: "5", Duration: 5, PercentComplete: 0, Predecessors: "1FS", IsSummary: false, IsMilestone: false},     // no lag
	}
	s := newSchedule(dd, tasks...)
	result := (&LagsMetric{}).Assess(s)

	if result.Details["lags"] != 2 {
		t.Errorf("Lags count = %v, want 2", result.Details["lags"])
	}
}

// ---------------------------------------------------------------------------
// Relationship Types
// ---------------------------------------------------------------------------

func TestRelationshipTypes_FSRatio(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "1FS", IsSummary: false, IsMilestone: false},  // FS
		{TaskID: "3", Duration: 5, PercentComplete: 0, Predecessors: "1", IsSummary: false, IsMilestone: false},    // implicit FS
		{TaskID: "4", Duration: 5, PercentComplete: 0, Predecessors: "1SS", IsSummary: false, IsMilestone: false},  // SS
		{TaskID: "5", Duration: 5, PercentComplete: 0, Predecessors: "1FF", IsSummary: false, IsMilestone: false},  // FF
	}
	s := newSchedule(dd, tasks...)
	result := (&RelationshipTypesMetric{}).Assess(s)

	// 2 FS out of 4 total = 50%
	wantVal := 0.5
	if result.Value < wantVal-0.001 || result.Value > wantVal+0.001 {
		t.Errorf("RelationshipTypes FS ratio = %.4f, want %.4f", result.Value, wantVal)
	}
	if result.Passing {
		t.Error("RelationshipTypes should FAIL when FS ratio < 90%")
	}
}

func TestRelationshipTypes_AllFS(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "1FS", IsSummary: false, IsMilestone: false},
		{TaskID: "3", Duration: 5, PercentComplete: 0, Predecessors: "1", IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&RelationshipTypesMetric{}).Assess(s)
	if result.Value != 1.0 || !result.Passing {
		t.Errorf("RelationshipTypes all-FS should be 100%% PASS, got %.2f passing=%v", result.Value, result.Passing)
	}
}

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

func TestResources_NotApplicableWhenNoResourceColumn(t *testing.T) {
	// When no task has resources assigned, the metric should be N/A
	// (resource column was not exported), not a FAIL.
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, Resources: "", IsSummary: false, IsMilestone: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, Resources: "", IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&ResourcesMetric{}).Assess(s)
	if !result.NotApplicable {
		t.Error("Resources should be N/A when no resource data exists in the export")
	}
}

func TestResources_CountsAssignedTasks(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, Resources: "Alice", IsSummary: false, IsMilestone: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, Resources: "", IsSummary: false, IsMilestone: false},
		{TaskID: "3", Duration: 5, PercentComplete: 0, Resources: "Bob", IsSummary: false, IsMilestone: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&ResourcesMetric{}).Assess(s)

	if result.NotApplicable {
		t.Fatal("Resources should not be N/A when some tasks have resources")
	}
	if result.Details["count"] != 2 {
		t.Errorf("Resources count = %v, want 2", result.Details["count"])
	}
	if result.Details["total"] != 3 {
		t.Errorf("Resources total = %v, want 3", result.Details["total"])
	}
}

func TestResources_Integration(t *testing.T) {
	// PAM 4.10 compliance integration test:
	// - Universe: incomplete non-summary, non-milestone work tasks
	// - Numerator: tasks with at least one resource assigned
	// - Threshold: ≥95%
	// - Exceptions: tasks missing resources with TaskID + Name
	// - N/A: when no Resource Names column exists in export
	r := reader.NewScheduleReader("../reader/testdata/test_resources_schedule.csv")
	s, err := r.Read()
	if err != nil {
		t.Fatalf("Failed to load test fixture: %v", err)
	}

	result := (&ResourcesMetric{}).Assess(s)

	// Should NOT be N/A since Resource Names column exists
	if result.NotApplicable {
		t.Fatal("Resources should not be N/A when Resource Names column exists")
	}

	// Verify universe (incomplete non-summary, non-milestone work tasks)
	// From fixture: Tasks 2, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 21, 22, 23, 25, 26, 29
	// Excluded: 1 (summary, 100%), 3 (100%), 6 (milestone, 100%), 16-19 (milestones/summary/100%), 20 (inactive), 24 (100%), 27 (milestone), 28 (summary)
	total := result.Details["total"].(int)
	if total == 0 {
		t.Error("Resources total should not be zero - fixture has incomplete work tasks")
	}

	// Verify numerator (tasks with resources assigned)
	count := result.Details["count"].(int)
	if count == 0 {
		t.Error("Resources count should not be zero - fixture has tasks with resources")
	}

	// Verify PAM 4.10 threshold (≥95%)
	if result.Threshold != 0.95 {
		t.Errorf("Resources threshold = %.4f, want 0.95", result.Threshold)
	}

	// Verify exceptions contain TaskID and Name for tasks missing resources
	if len(result.Exceptions) == 0 {
		t.Error("Resources should have exceptions for tasks without resources")
	}
	for _, exc := range result.Exceptions {
		if exc.TaskID == "" {
			t.Error("Resource exception missing TaskID")
		}
		if exc.Name == "" {
			t.Error("Resource exception missing Name")
		}
	}

	// Verify population scope label mentions PAM 4.10 universe
	if result.Population == nil {
		t.Fatal("Resources Population should not be nil")
	}
	if result.Population.ScopeLabel == "" {
		t.Error("Resources Population ScopeLabel should describe the universe")
	}
}

// ---------------------------------------------------------------------------
// Critical Path Test
// ---------------------------------------------------------------------------

func TestCriticalPathTest_PassWhenZeroFloat(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 0, IsSummary: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 10, IsSummary: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&CriticalPathTestMetric{}).Assess(s)
	if !result.Passing {
		t.Error("CriticalPathTest should PASS when at least one task has zero float")
	}
}

func TestCriticalPathTest_FailWhenNoZeroFloat(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 5, IsSummary: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 10, IsSummary: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&CriticalPathTestMetric{}).Assess(s)
	if result.Passing {
		t.Error("CriticalPathTest should FAIL when no task has zero or negative float")
	}
}

// ---------------------------------------------------------------------------
// High Baseline Duration
// ---------------------------------------------------------------------------

func TestHighDuration_Threshold(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "1", Duration: 5, BaselineDuration: 61, PercentComplete: 0, IsSummary: false, IsMilestone: false}, // > 60 → high
		{TaskID: "2", Duration: 5, BaselineDuration: 60, PercentComplete: 0, IsSummary: false, IsMilestone: false}, // = 60 → not high
		{TaskID: "3", Duration: 5, BaselineDuration: 10, PercentComplete: 0, IsSummary: false, IsMilestone: false}, // < 60 → not high
	}
	s := newSchedule(dd, tasks...)
	result := (&HighDurationMetric{}).Assess(s)

	if result.Details["count"] != 1 {
		t.Errorf("HighDuration count = %v, want 1 (only baseline duration > 60)", result.Details["count"])
	}
}

// ---------------------------------------------------------------------------
// Summary-task exclusion — Metric 1 (Logic)
// ---------------------------------------------------------------------------
// Summary tasks must not contribute predecessor links to the hasSuccessor map.
// If they did, their listed predecessors would falsely mark work tasks as
// "having a successor", hiding real logic gaps.

// TestLogic_SummarySuccessorNotCounted verifies that a work task which is only
// referenced as a predecessor *from a summary task* is still flagged as missing
// a real successor.
func TestLogic_SummarySuccessorNotCounted(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	// Task "1" has a predecessor (so it won't be flagged for missing predecessor).
	// Only the summary row lists "1" as its predecessor — no real work task does.
	// Task "1" should therefore still be flagged for missing a successor.
	tasks := []*model.Task{
		{
			TaskID:          "1",
			Name:            "Work A",
			Duration:        5,
			PercentComplete: 0,
			IsSummary:       false,
			IsMilestone:     false,
			Predecessors:    "", // no predecessor → will also be flagged for that
		},
		{
			TaskID:          "s1",
			Name:            "Summary S1",
			Duration:        10,
			PercentComplete: 0,
			IsSummary:       true,
			Predecessors:    "1", // summary references task 1 — must NOT count as a successor link
		},
	}
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)

	// Task "1" has neither a real predecessor nor a real (non-summary) successor.
	// missing = 1, total work tasks = 1 → value = 1.0 (100%) → FAIL
	if result.Value != 1.0 {
		t.Errorf("Logic value = %.4f, want 1.0; summary predecessor link should not count as a successor", result.Value)
	}
	if result.Passing {
		t.Error("Logic should FAIL when the only successor reference comes from a summary task")
	}
	if len(result.Exceptions) != 1 {
		t.Errorf("Logic exceptions = %d, want 1", len(result.Exceptions))
	}
}

// TestLogic_SummaryTaskNotCountedAsWorkTask confirms that summary tasks themselves
// are never included in the denominator (work task count) for the Logic metric.
func TestLogic_SummaryTaskNotCountedAsWorkTask(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	// Three work tasks forming a chain: 1 → 2 → 3.
	// One summary task with a predecessor — its link must not influence the map.
	// All work tasks have both a predecessor and a successor: 0 missing → PASS.
	tasks := []*model.Task{
		{
			TaskID:          "1",
			Name:            "Work A",
			Duration:        5,
			PercentComplete: 0,
			IsSummary:       false,
			IsMilestone:     false,
			Predecessors:    "0", // has a predecessor
		},
		{
			TaskID:          "2",
			Name:            "Work B",
			Duration:        5,
			PercentComplete: 0,
			IsSummary:       false,
			IsMilestone:     false,
			Predecessors:    "1", // task 1 gets a successor; task 2 has a predecessor
		},
		{
			TaskID:          "3",
			Name:            "Work C",
			Duration:        5,
			PercentComplete: 0,
			IsSummary:       false,
			IsMilestone:     false,
			Predecessors:    "2", // task 2 gets a successor; task 3 has a predecessor
		},
		{
			TaskID:       "s1",
			Name:         "Summary",
			Duration:     10,
			IsSummary:    true,
			Predecessors: "0", // summary links must not pollute the successor map
		},
	}
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)

	// Work tasks 1, 2, 3: all have at least one predecessor or successor;
	// task 3 has a predecessor but no successor — it is the end of the chain,
	// so it will be flagged.  The point of this test is that the denominator is 3
	// (not 4), confirming the summary task is not counted as a work task.
	if result.Details["total"] != 3 {
		t.Errorf("Logic total = %v, want 3; summary task must not count as work task", result.Details["total"])
	}
}

// ---------------------------------------------------------------------------
// Summary-task exclusion — Metric 12 (Critical Path Test)
// ---------------------------------------------------------------------------
// A summary task's TotalSlack is a roll-up value, not a scheduler-computed
// critical-path indicator. It must not satisfy the critical-path existence check.

// TestCriticalPathTest_SummaryZeroSlackDoesNotPass verifies that a schedule
// where only a summary task has TotalSlack ≤ 0 still FAILs the metric.
func TestCriticalPathTest_SummaryZeroSlackDoesNotPass(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		// Summary row: TotalSlack = 0 — must NOT count toward critical path.
		{TaskID: "s1", Name: "Summary", Duration: 10, IsSummary: true, TotalSlack: 0},
		// All real work tasks have positive float.
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 5, IsSummary: false},
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 10, IsSummary: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&CriticalPathTestMetric{}).Assess(s)

	if result.Passing {
		t.Error("CriticalPathTest must FAIL when only a summary task has zero slack; summary TotalSlack must not satisfy the check")
	}
	if result.Details["critical_tasks"] != 0 {
		t.Errorf("critical_tasks = %v, want 0 (summary excluded)", result.Details["critical_tasks"])
	}
}

// TestCriticalPathTest_WorkTaskZeroSlackPasses confirms that a real work task
// with TotalSlack ≤ 0 still satisfies the metric even when summary tasks are present.
func TestCriticalPathTest_WorkTaskZeroSlackPasses(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{TaskID: "s1", Name: "Summary", Duration: 10, IsSummary: true, TotalSlack: 0},
		{TaskID: "1", Duration: 5, PercentComplete: 0, TotalSlack: 0, IsSummary: false}, // critical
		{TaskID: "2", Duration: 5, PercentComplete: 0, TotalSlack: 10, IsSummary: false},
	}
	s := newSchedule(dd, tasks...)
	result := (&CriticalPathTestMetric{}).Assess(s)

	if !result.Passing {
		t.Error("CriticalPathTest should PASS when a real work task has zero slack")
	}
	if result.Details["critical_tasks"] != 1 {
		t.Errorf("critical_tasks = %v, want 1 (only the work task)", result.Details["critical_tasks"])
	}
}

// ---------------------------------------------------------------------------
// Invalid Dates — Zero Tolerance
// ---------------------------------------------------------------------------

func TestInvalidDates_ZeroTolerance(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{
			TaskID: "1", Duration: 5, PercentComplete: 0,
			Finish: date(2026, 4, 5), // Finish before data date = invalid
			IsSummary: false, IsMilestone: false,
			Active: true,
		},
	}
	s := newSchedule(dd, tasks...)
	result := (&InvalidDatesMetric{}).Assess(s)

	if result.Passing {
		t.Errorf("InvalidDates should FAIL with zero tolerance when any invalid date exists. Value=%v, Details=%v", result.Value, result.Details)
	}
	if result.Value != 1.0 {
		t.Errorf("InvalidDates value = %v, want 1.0 (100%% of tasks have invalid dates)", result.Value)
	}
}

func TestInvalidDates_Passing(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	tasks := []*model.Task{
		{
			TaskID: "1", Duration: 5, PercentComplete: 0,
			Start:  date(2026, 4, 11),
			Finish: date(2026, 4, 20), // valid dates
			IsSummary: false, IsMilestone: false,
		},
		{
			TaskID: "2", Duration: 5, PercentComplete: 0,
			Start:  date(2026, 4, 12),
			Finish: date(2026, 4, 25), // valid dates
			IsSummary: false, IsMilestone: false,
		},
	}
	s := newSchedule(dd, tasks...)
	result := (&InvalidDatesMetric{}).Assess(s)

	if !result.Passing {
		t.Error("InvalidDates should PASS when all dates are valid (zero tolerance)")
	}
	if result.Details["count"] != 0 {
		t.Errorf("InvalidDates count = %v, want 0", result.Details["count"])
	}
}

// ---------------------------------------------------------------------------
// Logic — 5% Threshold
// ---------------------------------------------------------------------------

func TestLogic_Threshold_5Percent(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	// Create 20 work tasks + 1 milestone
	// Task 1: no predecessor → missing (1 out of 20 = 5%)
	// Tasks 2-20: chain with full logic
	// Milestone M1: references task 20, so task 20 has a successor (milestone excluded from denominator)
	tasks := []*model.Task{
		// Task 1: no predecessor
		{TaskID: "1", Duration: 5, PercentComplete: 0, Predecessors: "", IsSummary: false, IsMilestone: false},
	}
	// Tasks 2-20: chain
	for i := 2; i <= 20; i++ {
		tasks = append(tasks, &model.Task{
			TaskID:       fmt.Sprintf("%d", i),
			Duration:     5,
			PercentComplete: 0,
			Predecessors: fmt.Sprintf("%d", i-1),
			IsSummary:    false,
			IsMilestone:  false,
		})
	}
	// Milestone that gives task 20 a successor
	tasks = append(tasks, &model.Task{
		TaskID: "M1", Duration: 0, PercentComplete: 0, Predecessors: "20",
		IsSummary: false, IsMilestone: true,
	})
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)

	// 1 out of 20 = 5%, which is exactly at threshold → should PASS
	if result.Details["total"] != 20 {
		t.Errorf("Logic total = %v, want 20 (milestone excluded from denominator)", result.Details["total"])
	}
	if result.Details["missing"] != 1 {
		t.Errorf("Logic missing = %v, want 1", result.Details["missing"])
	}
	if result.Value < 0.0499 || result.Value > 0.0501 {
		t.Errorf("Logic value = %.4f, want 0.05", result.Value)
	}
	// At exactly 5% threshold → should PASS (val <= threshold)
	if !result.Passing {
		t.Errorf("Logic should PASS when exactly 5%% of tasks have missing logic (at threshold). Value=%.4f, Threshold=%.4f", result.Value, result.Threshold)
	}
}

func TestLogic_FailAboveThreshold(t *testing.T) {
	dd := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	// Create 18 work tasks + 1 milestone
	// Tasks 1-2: no predecessor → missing (2 out of 18 = 11.1% > 5%)
	// Tasks 3-18: chain with full logic
	// Milestone M1: references task 18, so task 18 has a successor
	tasks := []*model.Task{
		// Task 1: no predecessor
		{TaskID: "1", Duration: 5, PercentComplete: 0, Predecessors: "", IsSummary: false, IsMilestone: false},
		// Task 2: no predecessor
		{TaskID: "2", Duration: 5, PercentComplete: 0, Predecessors: "", IsSummary: false, IsMilestone: false},
	}
	// Tasks 3-18: chain
	for i := 3; i <= 18; i++ {
		tasks = append(tasks, &model.Task{
			TaskID:       fmt.Sprintf("%d", i),
			Duration:     5,
			PercentComplete: 0,
			Predecessors: fmt.Sprintf("%d", i-1),
			IsSummary:    false,
			IsMilestone:  false,
		})
	}
	// Milestone that gives task 18 a successor
	tasks = append(tasks, &model.Task{
		TaskID: "M1", Duration: 0, PercentComplete: 0, Predecessors: "18",
		IsSummary: false, IsMilestone: true,
	})
	s := newSchedule(dd, tasks...)
	result := (&LogicMetric{}).Assess(s)

	// 2 out of 18 = 11.1%, which exceeds 5% threshold → should FAIL
	if result.Details["total"] != 18 {
		t.Errorf("Logic total = %v, want 18", result.Details["total"])
	}
	if result.Details["missing"] != 2 {
		t.Errorf("Logic missing = %v, want 2", result.Details["missing"])
	}
	if result.Value < 0.11 || result.Value > 0.112 {
		t.Errorf("Logic value = %.4f, want ~0.111", result.Value)
	}
	if result.Passing {
		t.Error("Logic should FAIL when 11.1% of tasks have missing logic (above 5% threshold)")
	}
}
