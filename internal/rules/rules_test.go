package rules

import (
	"strings"
	"testing"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func workTask(id, name, discipline string) *model.Task {
	return workTaskWithDuration(id, name, discipline, 5)
}

func workTaskWithDuration(id, name, discipline string, duration float64) *model.Task {
	return &model.Task{
		TaskID:     id,
		Name:       name,
		Discipline: discipline,
		Duration:   duration,
		IsSummary:  false,
		IsMilestone: false,
	}
}

func summaryTask(id, name, discipline string) *model.Task {
	return &model.Task{
		TaskID:     id,
		Name:       name,
		Discipline: discipline,
		Duration:   5,
		IsSummary:  true,
	}
}

func milestoneTask(id, name, discipline string) *model.Task {
	return &model.Task{
		TaskID:      id,
		Name:        name,
		Discipline:  discipline,
		Duration:    0,
		IsMilestone: true,
	}
}

func newSchedule(tasks ...*model.Task) *model.Schedule {
	return &model.Schedule{Name: "test", Tasks: tasks}
}

// ---------------------------------------------------------------------------
// filterWorkTasks (package-level helper)
// ---------------------------------------------------------------------------

// TestFilterWorkTasks_ExcludesSummaryAndMilestone verifies that both summary
// and milestone tasks are removed and only work tasks remain.
func TestFilterWorkTasks_ExcludesSummaryAndMilestone(t *testing.T) {
	tasks := []*model.Task{
		{TaskID: "1", IsSummary: false, IsMilestone: false},
		{TaskID: "s1", IsSummary: true, IsMilestone: false},
		{TaskID: "m1", IsSummary: false, IsMilestone: true},
		{TaskID: "2", IsSummary: false, IsMilestone: false},
	}
	got := filterWorkTasks(tasks)
	if len(got) != 2 {
		t.Fatalf("filterWorkTasks returned %d tasks, want 2", len(got))
	}
	for _, task := range got {
		if task.IsSummary || task.IsMilestone {
			t.Errorf("filterWorkTasks kept a summary/milestone task (id=%s)", task.TaskID)
		}
	}
}

// ---------------------------------------------------------------------------
// Evaluate — summary/milestone exclusion
// ---------------------------------------------------------------------------

// TestEvaluate_SummaryTaskNotCountedByRule verifies that a summary task matching
// a rule's pattern is not included in the match count.
func TestEvaluate_SummaryTaskNotCountedByRule(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name:     "Mechanical tasks",
				Match:    map[string]string{"discipline": "*echanical*"},
				MinCount: 1,
			},
		},
	}

	s := newSchedule(
		workTask("1", "Install Pump", "05 - Mechanical"),
		summaryTask("s1", "Mechanical Summary", "05 - Mechanical"), // must be excluded
	)

	results := rs.Evaluate(s)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Count != 1 {
		t.Errorf("rule Count = %d, want 1; summary task must not be counted", r.Count)
	}
}

// TestEvaluate_SummaryOnlyMatchFails verifies that if only a summary task matches
// a rule, the count is 0 and the rule fails the min_count constraint.
func TestEvaluate_SummaryOnlyMatchFails(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name:     "Need mechanical",
				Match:    map[string]string{"discipline": "*echanical*"},
				MinCount: 1,
			},
		},
	}

	s := newSchedule(
		workTask("1", "Electrical Install", "03 - Electrical"),
		summaryTask("s1", "Mechanical Summary", "05 - Mechanical"), // only match — but is summary
	)

	results := rs.Evaluate(s)
	r := results[0]

	if r.Count != 0 {
		t.Errorf("rule Count = %d, want 0; summary-only match must yield zero work-task count", r.Count)
	}
	if r.Passing {
		t.Error("rule should FAIL when only a summary task matches the pattern")
	}
}

// TestEvaluate_MilestoneNotCountedByRule confirms milestones are also excluded
// from rule evaluation alongside summary tasks.
func TestEvaluate_MilestoneNotCountedByRule(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name:     "Mechanical tasks",
				Match:    map[string]string{"discipline": "*echanical*"},
				MinCount: 1,
			},
		},
	}

	s := newSchedule(
		workTask("1", "Install Pump", "05 - Mechanical"),
		milestoneTask("m1", "Mechanical Complete", "05 - Mechanical"),
	)

	results := rs.Evaluate(s)
	r := results[0]

	if r.Count != 1 {
		t.Errorf("rule Count = %d, want 1; milestone must not be counted", r.Count)
	}
}

func TestEvaluate_ConstraintViolationTasksStillShowAsMatches(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name: "Long mechanical tasks",
				Match: map[string]string{"discipline": "*echanical*"},
				Constraints: Constraints{
					MinDuration: 10,
				},
				MinCount: 2,
			},
		},
	}

	s := newSchedule(
		workTaskWithDuration("1", "Short Mech", "05 - Mechanical", 5),
		workTaskWithDuration("2", "Long Mech A", "05 - Mechanical", 15),
		workTaskWithDuration("3", "Long Mech B", "05 - Mechanical", 20),
	)

	results := rs.Evaluate(s)
	r := results[0]

	if len(r.MatchingTasks) != 3 {
		t.Errorf("MatchingTasks length = %d, want 3; all pattern-matched tasks should appear", len(r.MatchingTasks))
	}
	if r.Count != 2 {
		t.Errorf("Count = %d, want 2; short task should be excluded from effective count", r.Count)
	}
	if !strings.Contains(r.Message, "excluded by duration constraints") {
		t.Errorf("Message should mention constraint exclusions, got: %q", r.Message)
	}
}

func TestEvaluate_NoConstraintsBehaviorUnchanged(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name:     "Mechanical tasks",
				Match:    map[string]string{"discipline": "*echanical*"},
				MinCount: 2,
			},
		},
	}

	s := newSchedule(
		workTask("1", "Mech A", "05 - Mechanical"),
		workTask("2", "Mech B", "05 - Mechanical"),
	)

	results := rs.Evaluate(s)
	r := results[0]

	if r.Count != 2 {
		t.Errorf("Count = %d, want 2", r.Count)
	}
	if r.Message != "OK" {
		t.Errorf("Message = %q, want OK", r.Message)
	}
	if len(r.MatchingTasks) != 2 {
		t.Errorf("MatchingTasks = %d, want 2", len(r.MatchingTasks))
	}
}

func TestEvaluate_MaxDurationConstraintExcludesMatches(t *testing.T) {
	rs := &RuleSet{
		Rules: []Rule{
			{
				Name: "Short tasks only",
				Match: map[string]string{"discipline": "*echanical*"},
				Constraints: Constraints{
					MaxDuration: 10,
				},
				MinCount: 1,
			},
		},
	}

	s := newSchedule(
		workTaskWithDuration("1", "Short Mech", "05 - Mechanical", 5),
		workTaskWithDuration("2", "Long Mech", "05 - Mechanical", 15),
	)

	results := rs.Evaluate(s)
	r := results[0]

	if len(r.MatchingTasks) != 2 {
		t.Errorf("MatchingTasks length = %d, want 2; both tasks match the pattern", len(r.MatchingTasks))
	}
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1; long task excluded by max duration", r.Count)
	}
}
