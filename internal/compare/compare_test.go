package compare

import (
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

func workTask(id, name string, duration, pct float64, finish *time.Time) *model.Task {
	return &model.Task{
		TaskID:          id,
		Name:            name,
		Duration:        duration,
		PercentComplete: pct,
		Start:           date(2026, 1, 1),
		Finish:          finish,
		IsSummary:       false,
		IsMilestone:     false,
	}
}

func summaryTask(id string, finish *time.Time) *model.Task {
	return &model.Task{
		TaskID:    id,
		Name:      "Summary " + id,
		Duration:  20,
		IsSummary: true,
		Start:     date(2026, 1, 1),
		Finish:    finish,
	}
}

func newSchedule(tasks ...*model.Task) *model.Schedule {
	return &model.Schedule{
		Name:     "test",
		DataDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Tasks:    tasks,
	}
}

// ---------------------------------------------------------------------------
// filterNonSummaryTasks
// ---------------------------------------------------------------------------

// TestFilterNonSummaryTasks_RemovesSummaries verifies the helper removes all
// tasks where IsSummary is true and retains all others.
func TestFilterNonSummaryTasks_RemovesSummaries(t *testing.T) {
	tasks := []*model.Task{
		{TaskID: "1", IsSummary: false},
		{TaskID: "s1", IsSummary: true},
		{TaskID: "2", IsSummary: false},
		{TaskID: "s2", IsSummary: true},
	}
	got := filterNonSummaryTasks(tasks)
	if len(got) != 2 {
		t.Fatalf("filterNonSummaryTasks returned %d tasks, want 2", len(got))
	}
	for _, task := range got {
		if task.IsSummary {
			t.Errorf("filterNonSummaryTasks returned a summary task (id=%s)", task.TaskID)
		}
	}
}

// TestFilterNonSummaryTasks_AllWork confirms empty summary input passes through cleanly.
func TestFilterNonSummaryTasks_AllWork(t *testing.T) {
	tasks := []*model.Task{
		{TaskID: "1", IsSummary: false},
		{TaskID: "2", IsSummary: false},
	}
	got := filterNonSummaryTasks(tasks)
	if len(got) != 2 {
		t.Fatalf("filterNonSummaryTasks returned %d tasks, want 2", len(got))
	}
}

// ---------------------------------------------------------------------------
// CompareSchedules — summary task exclusion
// ---------------------------------------------------------------------------

// TestCompareSchedules_SummaryTasksExcludedFromTotalCount verifies that summary
// tasks in both prev and curr are not counted toward TotalTasks or any pillar
// score calculation.
func TestCompareSchedules_SummaryTasksExcludedFromTotalCount(t *testing.T) {
	finish := date(2026, 6, 1)

	// 2 real work tasks + 1 summary task in each snapshot.
	prev := newSchedule(
		workTask("1", "Task A", 10, 0, finish),
		workTask("2", "Task B", 10, 0, finish),
		summaryTask("s1", finish),
	)
	curr := newSchedule(
		workTask("1", "Task A", 10, 0, finish),
		workTask("2", "Task B", 10, 0, finish),
		summaryTask("s1", finish),
	)

	result := CompareSchedules(prev, curr)

	// Only 2 work tasks should be visible to the engine.
	if result.TotalTasks != 2 {
		t.Errorf("TotalTasks = %d, want 2 (summary tasks must be excluded)", result.TotalTasks)
	}
}

// TestCompareSchedules_SummaryChurnIgnored verifies that a summary task that
// appears in curr but not in prev is not counted as new churn.
func TestCompareSchedules_SummaryChurnIgnored(t *testing.T) {
	finish := date(2026, 6, 1)

	prev := newSchedule(
		workTask("1", "Task A", 10, 0, finish),
	)
	curr := newSchedule(
		workTask("1", "Task A", 10, 0, finish),
		summaryTask("s_new", finish), // new summary — must not inflate churn
	)

	result := CompareSchedules(prev, curr)

	if result.NewTasks != 0 {
		t.Errorf("NewTasks = %d, want 0; summary-only additions must not count as churn", result.NewTasks)
	}
	if result.TotalTasks != 1 {
		t.Errorf("TotalTasks = %d, want 1", result.TotalTasks)
	}
}

// TestCompareSchedules_SummaryFinishVarianceIgnored verifies that a summary
// task whose finish date shifted does not contribute to Pillar A (finish variance).
func TestCompareSchedules_SummaryFinishVarianceIgnored(t *testing.T) {
	stableFinish := date(2026, 6, 1)

	prev := newSchedule(
		workTask("1", "Stable Task", 10, 0, stableFinish),
		summaryTask("s1", date(2026, 5, 1)), // summary finish shifts significantly
	)
	curr := newSchedule(
		workTask("1", "Stable Task", 10, 0, stableFinish), // unchanged
		summaryTask("s1", date(2026, 7, 1)),                // +60 day shift — must not penalise Pillar A
	)

	result := CompareSchedules(prev, curr)

	// No work task has a finish variance > 2 days; Pillar A should be perfect (40).
	if result.PillarAScore != 40.0 {
		t.Errorf("PillarAScore = %.2f, want 40.0; summary finish variance must not penalise Pillar A", result.PillarAScore)
	}
}

// ---------------------------------------------------------------------------
// Anonymous Schedule Integration Tests
// ---------------------------------------------------------------------------

func TestCompareSchedules_AnonymousSchedules(t *testing.T) {
	// Test comparison of the large anonymous test schedules (v1 vs v2)
	r1 := reader.NewScheduleReader("../../internal/reader/testdata/test_anonymous_schedule.csv")
	prev, err := r1.Read()
	if err != nil {
		t.Fatalf("Read() v1 returned error: %v", err)
	}

	r2 := reader.NewScheduleReader("../../internal/reader/testdata/test_anonymous_schedule_v2.csv")
	curr, err := r2.Read()
	if err != nil {
		t.Fatalf("Read() v2 returned error: %v", err)
	}

	// Run comparison
	result := CompareSchedules(prev, curr)

	// Verify comparison completed
	if result.TotalTasks == 0 {
		t.Fatal("expected tasks in comparison")
	}

	// Verify scores are in valid range [0, 100]
	if result.OverallScore < 0 || result.OverallScore > 100 {
		t.Errorf("OverallScore out of range: %.2f", result.OverallScore)
	}
	if result.PillarAScore < 0 || result.PillarAScore > 40 {
		t.Errorf("PillarAScore out of range: %.2f", result.PillarAScore)
	}
	if result.PillarBScore < 0 || result.PillarBScore > 30 {
		t.Errorf("PillarBScore out of range: %.2f", result.PillarBScore)
	}
	if result.PillarCScore < 0 || result.PillarCScore > 30 {
		t.Errorf("PillarCScore out of range: %.2f", result.PillarCScore)
	}
}
