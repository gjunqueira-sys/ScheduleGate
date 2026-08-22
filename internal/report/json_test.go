package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/rules"
)

func TestWriteJSON_IndentedAndValid(t *testing.T) {
	var buf bytes.Buffer
	payload := map[string]int{"a": 1}
	if err := WriteJSON(&buf, payload); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("  \"a\": 1")) {
		t.Errorf("WriteJSON output not indented: %q", buf.String())
	}
	var back map[string]int
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Errorf("WriteJSON output is not valid JSON: %v", err)
	}
}

func TestBuildAssessJSON(t *testing.T) {
	m := &dcma.LogicMetric{}
	assessment := &dcma.DCMAAssessment{
		Metrics: []dcma.Metric{m},
		Results: map[string]dcma.MetricResult{
			m.Name(): {
				Name:      m.Name(),
				Value:     0.92,
				Threshold: 0.95,
				Passing:   false,
				Exceptions: []dcma.TaskException{
					{TaskID: "42", Name: "Task 42", Condition: "no successor"},
				},
			},
		},
		Population: dcma.SchedulePopulation{
			WorkTasks:           100,
			CompletedWorkTasks:  40,
			IncompleteWorkTasks: 60,
		},
	}

	out := BuildAssessJSON(assessment, "Test Schedule", "2026-08-16", []string{"parse note"})
	if out.ScheduleName != "Test Schedule" {
		t.Errorf("ScheduleName = %q, want Test Schedule", out.ScheduleName)
	}
	if out.StatusDate != "2026-08-16" {
		t.Errorf("StatusDate = %q, want 2026-08-16", out.StatusDate)
	}
	if out.OverallScore != 0 {
		t.Errorf("OverallScore = %v, want 0 (0/1 passed)", out.OverallScore)
	}
	if out.PassedCount != 0 || out.TotalCount != 1 {
		t.Errorf("counts = (%d, %d), want (0, 1)", out.PassedCount, out.TotalCount)
	}
	if len(out.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(out.Results))
	}
	r := out.Results[0]
	if r.ID != m.ID() || r.Name != m.Name() || r.Description != m.Description() {
		t.Errorf("metric meta mismatch: got %+v", r)
	}
	if r.ExceptionCount != 1 {
		t.Errorf("ExceptionCount = %d, want 1", r.ExceptionCount)
	}
	if out.Population.WorkTasks != 100 || out.Population.IncompleteWorkTasks != 60 {
		t.Errorf("Population = %+v, want WorkTasks=100 Incomplete=60", out.Population)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("Warnings = %v, want 1 entry", out.Warnings)
	}
	if out.ToolVersion == "" || out.GeneratedAt == "" {
		t.Error("ToolVersion and GeneratedAt must be populated")
	}
}

func TestBuildAssessJSON_ScoreExcludesNA(t *testing.T) {
	pass := &dcma.LogicMetric{}
	na := &dcma.BEIMetric{}
	assessment := &dcma.DCMAAssessment{
		Metrics: []dcma.Metric{pass, na},
		Results: map[string]dcma.MetricResult{
			pass.Name(): {Name: pass.Name(), Value: 0.9, Threshold: 0.95, Passing: true},
			na.Name():   {Name: na.Name(), Value: 0, Threshold: 0.8, Passing: false, NotApplicable: true},
		},
	}
	out := BuildAssessJSON(assessment, "S", "2026-08-16", nil)
	if out.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1 (N/A excluded)", out.TotalCount)
	}
	if out.OverallScore != 100 {
		t.Errorf("OverallScore = %v, want 100", out.OverallScore)
	}
}

func TestBuildCompareJSON(t *testing.T) {
	result := &compare.BenchmarkResult{
		OverallScore:          82.5,
		PillarAScore:          33,
		PillarBScore:          25,
		PillarCScore:          24.5,
		TotalTasks:            100,
		NewTasks:              10,
		DeletedTasks:          5,
		ModifiedTasks:         20,
		UnchangedTasks:        65,
		GhostTasksCount:       3,
		DurationInflatedCount: 2,
		DurationInflatedPct:   2.0,
		TaskDeltas: []*compare.TaskDelta{
			{
				TaskID:         "T1",
				Name:           "Foundation",
				WBS:            "WBS.1",
				Status:         compare.StatusModified,
				FinishVariance: 3.5,
				DurationDelta:  2,
				PrevDuration:   20,
				IsGhostTask:    true,
				ImpactType:     "Reliability",
				ImpactMsg:      "duration grew >10%",
			},
		},
		FrictionIndex: []compare.FrictionItem{{WBS: "WBS.1", GhostTaskCount: 3}},
		Warnings:      []string{"duplicate TaskID T1"},
	}

	out := BuildCompareJSON(result, "old.xlsx", "new.xlsx", "2026-08-16")
	if out.OverallScore != 82.5 {
		t.Errorf("OverallScore = %v, want 82.5", out.OverallScore)
	}
	if out.StabilityScore != 33 || out.ReliabilityScore != 25 || out.ScopeChurnScore != 24.5 {
		t.Errorf("pillar scores = (%v,%v,%v)", out.StabilityScore, out.ReliabilityScore, out.ScopeChurnScore)
	}
	if len(out.TaskDeltas) != 1 {
		t.Fatalf("len(TaskDeltas) = %d, want 1", len(out.TaskDeltas))
	}
	d := out.TaskDeltas[0]
	if d.TaskID != "T1" || d.Status != "Modified" || d.FinishVariance != 3.5 {
		t.Errorf("TaskDelta = %+v", d)
	}
	if len(out.FrictionIndex) != 1 || out.FrictionIndex[0].WBS != "WBS.1" {
		t.Errorf("FrictionIndex = %+v", out.FrictionIndex)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("Warnings = %v, want 1 entry", out.Warnings)
	}
}

func TestBuildValidateJSON(t *testing.T) {
	result := &reader.ColumnValidationResult{
		Found: map[string]string{
			"task_id": "Task ID",
			"name":    "Task Name",
		},
		Warnings: []string{"header case normalized"},
	}

	out := BuildValidateJSON(result, "test.xlsx")
	if out.Status != "INCOMPLETE" {
		t.Errorf("Status = %q, want INCOMPLETE (most required columns missing)", out.Status)
	}
	if out.RequiredFound["task_id"] != "Task ID" {
		t.Errorf("RequiredFound[task_id] = %q, want Task ID", out.RequiredFound["task_id"])
	}
	// Every required column except the two found should be in RequiredMissing.
	wantMissing := len(reader.RequiredColumns) - 2
	if len(out.RequiredMissing) != wantMissing {
		t.Errorf("len(RequiredMissing) = %d, want %d", len(out.RequiredMissing), wantMissing)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("Warnings = %v, want 1 entry", out.Warnings)
	}
}

func TestBuildValidateJSON_Ready(t *testing.T) {
	allFound := make(map[string]string)
	for _, col := range reader.RequiredColumns {
		allFound[col] = col
	}
	result := &reader.ColumnValidationResult{Found: allFound}
	out := BuildValidateJSON(result, "test.xlsx")
	if out.Status != "READY" {
		t.Errorf("Status = %q, want READY", out.Status)
	}
	if len(out.RequiredMissing) != 0 {
		t.Errorf("RequiredMissing = %v, want empty", out.RequiredMissing)
	}
}

func TestBuildPatternsJSON(t *testing.T) {
	results := []rules.RuleResult{
		{
			Rule: rules.Rule{
				Name:     "Design packages present",
				Match:    map[string]string{"WBS": "DESIGN*"},
				MinCount: 1,
				MaxCount: 0,
			},
			Count:   3,
			Passing: true,
			MatchingTasks: []*model.Task{
				{TaskID: "10", Name: "Design Pkg 1"},
				{TaskID: "11", Name: "Design Pkg 2"},
			},
		},
		{
			Rule:    rules.Rule{Name: "No orphan tasks", MinCount: 1},
			Count:   0,
			Passing: false,
			Message: "no matching tasks found",
		},
	}

	out := BuildPatternsJSON(results, "sched.xlsx", "rules.yaml")
	if out.Status != "NON-COMPLIANT" {
		t.Errorf("Status = %q, want NON-COMPLIANT", out.Status)
	}
	if out.PassedCount != 1 || out.TotalCount != 2 {
		t.Errorf("counts = (%d, %d), want (1, 2)", out.PassedCount, out.TotalCount)
	}
	if len(out.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(out.Results))
	}
	r := out.Results[0]
	if r.RuleName != "Design packages present" || r.MatchingCount != 3 {
		t.Errorf("first result = %+v", r)
	}
	if len(r.MatchingTaskIDs) != 2 || r.MatchingTaskIDs[0] != "10" {
		t.Errorf("MatchingTaskIDs = %v, want [10 11]", r.MatchingTaskIDs)
	}
	if r.Match["WBS"] != "DESIGN*" {
		t.Errorf("Match = %v", r.Match)
	}
}

// ---------------------------------------------------------------------------
// Anonymous Schedule Integration Tests
// ---------------------------------------------------------------------------

func TestBuildAssessJSON_AnonymousSchedule(t *testing.T) {
	// Test JSON output generation with the large anonymous test schedule
	r := reader.NewScheduleReader("../../internal/reader/testdata/test_anonymous_schedule.csv")
	sched, err := r.Read()
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	// Run assessment
	assessment := dcma.NewDCMAAssessment(nil)
	assessment.Assess(sched)

	// Build JSON output
	statusDate := "2026-04-10"
	out := BuildAssessJSON(assessment, sched.Name, statusDate, sched.Warnings)

	// Verify JSON structure
	if out.ScheduleName != sched.Name {
		t.Errorf("ScheduleName = %q, want %q", out.ScheduleName, sched.Name)
	}
	if out.StatusDate != statusDate {
		t.Errorf("StatusDate = %q, want %q", out.StatusDate, statusDate)
	}
	if out.ToolVersion == "" {
		t.Error("ToolVersion should be populated")
	}
	if out.GeneratedAt == "" {
		t.Error("GeneratedAt should be populated")
	}

	// Verify all metrics are present
	if len(out.Results) != 14 {
		t.Errorf("expected 14 results, got %d", len(out.Results))
	}

	// Verify no identifying references in JSON
	jsonStr := out.ScheduleName + out.StatusDate
	if strings.Contains(strings.ToLower(jsonStr), "ross") {
		t.Error("JSON contains identifying reference to 'Ross'")
	}
}
