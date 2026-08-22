package dcma

import (
	"fmt"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
)

// TaskException identifies a single task (or relationship) that contributed to
// a metric violation, along with a human-readable condition string that
// describes what is wrong and how to correct it.
type TaskException struct {
	TaskID    string
	Name      string
	Condition string
}

// ExclusionStep is one row of a metric's filtering funnel: a human-readable
// reason text paired with the count of items removed at that step.
//
// Example: {Reason: "Summary rows excluded", Count: 1067}
type ExclusionStep struct {
	Reason string
	Count  int
}

// Population describes the filtering funnel that produced a metric's
// denominator. It is metadata only — Population values never feed into the
// metric value or pass/fail logic; they exist solely so the Excel exceptions
// workbook can show the user what was in scope, what was excluded, and why.
//
// Fields:
//
//	Numerator   — count of items that violated the metric (same as len(Exceptions)
//	              for task-unit metrics, or the relationship-violator count for
//	              relationship-unit metrics)
//	Denominator — count of items the metric ratio is computed against
//	Unit        — "tasks", "relationships", or "schedule" (for schedule-level
//	              metrics like Critical Path Test and CPLI that have no ratio)
//	ScopeLabel  — one-line description of what is IN the denominator
//	Excluded    — ordered list of exclusion steps (top-down funnel)
type Population struct {
	Numerator   int
	Denominator int
	Unit        string
	ScopeLabel  string
	Excluded    []ExclusionStep
}

// MetricResult holds the result of a single metric assessment.
type MetricResult struct {
	Name          string
	Value         float64
	Threshold     float64
	Passing       bool
	NotApplicable bool
	Details       map[string]interface{}
	Exceptions    []TaskException
	// Population captures the metric's universe and filtering funnel for the
	// Excel exceptions report. nil for metrics that have not been instrumented
	// yet, so existing tests and callers keep working unchanged.
	Population *Population
}

// SchedulePopulation captures the file-wide task classification funnel for a
// single assessment run. It is computed once in DCMAAssessment.Assess and
// reused by the Excel writer to render the top section of the Universe sheet.
//
// All counts reflect the post-import view of the schedule. Tasks marked
// Active=No are silently dropped by the reader, so they are not represented
// in any of the counts below.
//
// Fields:
//
//	TotalRows                    — every task row (summary + milestone + work) in s.Tasks
//	SummaryRows                  — rows where IsSummary == true
//	Milestones                   — rows where IsSummary == false and IsMilestone == true
//	WorkTasks                    — rows where IsSummary == false and IsMilestone == false
//	CompletedWorkTasks           — work tasks with PercentComplete >= 100
//	IncompleteWorkTasks          — work tasks with PercentComplete <  100
//	WorkTasksWithBaselineFinish  — work tasks where BaselineFinish != nil
//	WorkTasksBaselineDueByStatus — work tasks whose BaselineFinish ≤ DataDate
//
// The remaining fields describe the "work + milestones" universe used by
// Missed Tasks (PAM 4.11) and BEI (DAU APMT-009) — these metrics include
// milestones because contractual phase gates are part of their universe per
// the standards.
//
//	AssessableTasks                    — work tasks + milestones (non-summary)
//	CompletedAssessableTasks           — of the above, PercentComplete >= 100
//	AssessableTasksWithBaselineFinish  — of the above, BaselineFinish != nil
//	AssessableTasksBaselineDueByStatus — of the above, BaselineFinish ≤ DataDate
type SchedulePopulation struct {
	TotalRows                          int
	SummaryRows                        int
	Milestones                         int
	WorkTasks                          int
	CompletedWorkTasks                 int
	IncompleteWorkTasks                int
	WorkTasksWithBaselineFinish        int
	WorkTasksBaselineDueByStatus       int
	AssessableTasks                    int
	CompletedAssessableTasks           int
	AssessableTasksWithBaselineFinish  int
	AssessableTasksBaselineDueByStatus int
}

// Metric is the interface for all DCMA metrics.
type Metric interface {
	ID() int
	Name() string
	Description() string
	Threshold() float64
	Assess(schedule *model.Schedule) MetricResult
}

// DCMAAssessment runs the 14-point assessment.
type DCMAAssessment struct {
	Metrics    []Metric
	Results    map[string]MetricResult
	// Population captures the file-wide task classification funnel that drives
	// every metric's denominator. Populated by Assess before any metric runs.
	Population SchedulePopulation
}

// NewDCMAAssessment creates a new assessment.
// If selectedMetrics is empty, all metrics are used.
func NewDCMAAssessment(selectedMetrics []int) *DCMAAssessment {
	allMetrics := []Metric{
		&LogicMetric{},
		&LeadsMetric{},
		&LagsMetric{},
		&RelationshipTypesMetric{},
		&HardConstraintsMetric{},
		&HighFloatMetric{},
		&NegativeFloatMetric{},
		&HighDurationMetric{},
		&InvalidDatesMetric{},
		&ResourcesMetric{},
		&MissedTasksMetric{},
		&CriticalPathTestMetric{},
		&CPLIMetric{},
		&BEIMetric{},
	}

	var metricsToRun []Metric
	if len(selectedMetrics) == 0 {
		metricsToRun = allMetrics
	} else {
		selectedMap := make(map[int]bool)
		for _, id := range selectedMetrics {
			selectedMap[id] = true
		}
		for _, m := range allMetrics {
			if selectedMap[m.ID()] {
				metricsToRun = append(metricsToRun, m)
			}
		}
	}

	return &DCMAAssessment{
		Metrics: metricsToRun,
		Results: make(map[string]MetricResult),
	}
}

// Assess runs the assessment on the schedule.
//
// Signature: (a *DCMAAssessment) Assess(schedule *model.Schedule)
// Arguments:
//   - schedule: parsed schedule (active tasks only)
//
// Side effects: populates a.Population with the file-wide task classification
// funnel, then runs each configured metric and stores the result in a.Results.
func (a *DCMAAssessment) Assess(schedule *model.Schedule) {
	a.Population = computeSchedulePopulation(schedule)
	for _, m := range a.Metrics {
		result := m.Assess(schedule)
		a.Results[m.Name()] = result
	}
}

// computeSchedulePopulation walks the schedule once and tallies every task
// classification used by the metrics so that the Excel writer can render the
// top section of the Universe sheet without duplicating the logic in metrics.go.
//
// Signature: computeSchedulePopulation(s *model.Schedule) SchedulePopulation
// Arguments:
//   - s: parsed schedule
//
// Returns: a SchedulePopulation value with one entry per task category.
func computeSchedulePopulation(s *model.Schedule) SchedulePopulation {
	var p SchedulePopulation
	for _, t := range s.Tasks {
		p.TotalRows++
		if t.IsSummary {
			p.SummaryRows++
			continue
		}
		// Anything below counts toward the "assessable" universe (used by
		// PAM 4.11 Missed Tasks and DAU APMT-009 BEI).
		p.AssessableTasks++
		if t.PercentComplete >= 100 {
			p.CompletedAssessableTasks++
		}
		if t.BaselineFinish != nil {
			p.AssessableTasksWithBaselineFinish++
			if !t.BaselineFinish.After(s.DataDate) {
				p.AssessableTasksBaselineDueByStatus++
			}
		}
		if t.IsMilestone {
			p.Milestones++
			continue
		}
		// Work-task-specific counts (used by Logic, Float, Duration, etc.).
		p.WorkTasks++
		if t.PercentComplete >= 100 {
			p.CompletedWorkTasks++
		} else {
			p.IncompleteWorkTasks++
		}
		if t.BaselineFinish != nil {
			p.WorkTasksWithBaselineFinish++
			if !t.BaselineFinish.After(s.DataDate) {
				p.WorkTasksBaselineDueByStatus++
			}
		}
	}
	return p
}

// LogicDebugLine describes why a specific task was or was not flagged by the
// Logic metric, showing which successor references were skipped due to task
// classification (summary, milestone) and which token formats prevented matching.
type LogicDebugLine struct {
	TaskID      string
	Name        string
	IsSummary   bool
	IsMilestone bool
	Predecessors string
	// SkippedBy lists the IDs whose predecessor references were ignored because
	// the referencing task is a summary row.
	SkippedBy []string
	// InactiveReferrers lists IDs of inactive tasks that reference the task as
	// a predecessor. Populated so auditors can see when the only successor of
	// a flagged task is disabled.
	InactiveReferrers []string
	HasSuccessor bool
	Flagged      bool
	FlagReason   string
}

// DebugLogic returns per-task diagnostic lines for the Logic metric so callers
// can trace exactly why a task was flagged or cleared.
//
// Signature: DebugLogic(s *model.Schedule) []LogicDebugLine
// Arguments:
//   - s: parsed schedule
//
// Returns: one LogicDebugLine per non-summary, incomplete work task.
func DebugLogic(s *model.Schedule) []LogicDebugLine {
	// Build successor map — same logic as LogicMetric.Assess.
	hasSuccessor := make(map[string]bool)
	// Track which tasks' links were skipped because they are summary rows.
	skippedLinks := make(map[string][]string) // predecessor ID → list of summary task IDs that tried to reference it
	for _, t := range s.Tasks {
		if t.IsSummary {
			if hasContent(t.Predecessors) {
				for _, p := range splitPredecessors(t.Predecessors) {
					if id := extractTaskID(strings.TrimSpace(p)); id != "" {
						skippedLinks[id] = append(skippedLinks[id], t.TaskID)
					}
				}
			}
			continue
		}
		if hasContent(t.Predecessors) {
			for _, p := range splitPredecessors(t.Predecessors) {
				if id := extractTaskID(strings.TrimSpace(p)); id != "" {
					hasSuccessor[id] = true
				}
			}
		}
	}

	// Track inactive references separately so the debug output can call out
	// the "only-referenced-by-inactive" pattern explicitly.
	inactiveRefs := make(map[string][]string)
	for _, t := range s.InactiveTasks {
		if !hasContent(t.Predecessors) {
			continue
		}
		for _, p := range splitPredecessors(t.Predecessors) {
			if id := extractTaskID(strings.TrimSpace(p)); id != "" {
				inactiveRefs[id] = append(inactiveRefs[id], t.TaskID)
			}
		}
	}

	var lines []LogicDebugLine
	for _, t := range s.Tasks {
		if t.IsSummary || t.IsMilestone || t.PercentComplete >= 100 {
			continue
		}
		noPred := !hasContent(t.Predecessors)
		noSucc := !hasSuccessor[t.TaskID]
		flagged := noPred || noSucc
		reason := ""
		if flagged {
			switch {
			case noPred && noSucc:
				reason = "island (no predecessor, no successor)"
			case noPred:
				reason = "no predecessor"
			default:
				reason = "no successor"
			}
		}
		lines = append(lines, LogicDebugLine{
			TaskID:            t.TaskID,
			Name:              t.Name,
			IsSummary:         t.IsSummary,
			IsMilestone:       t.IsMilestone,
			Predecessors:      t.Predecessors,
			SkippedBy:         skippedLinks[t.TaskID],
			InactiveReferrers: inactiveRefs[t.TaskID],
			HasSuccessor:      hasSuccessor[t.TaskID],
			Flagged:           flagged,
			FlagReason:        reason,
		})
	}
	return lines
}

// GetSummary returns a simplified text summary.
func (a *DCMAAssessment) GetSummary() string {
	var sb strings.Builder
	passedCount := 0
	for _, m := range a.Metrics {
		if a.Results[m.Name()].Passing {
			passedCount++
		}
	}

	sb.WriteString("\nDCMA 14-Point Assessment Results\n")
	sb.WriteString(strings.Repeat("=", 40) + "\n")
	sb.WriteString(fmt.Sprintf("Overall Score: %.1f%% (%d/%d passed)\n\n", float64(passedCount)/float64(len(a.Metrics))*100, passedCount, len(a.Metrics)))
	sb.WriteString("Metric Results:\n")

	for _, m := range a.Metrics {
		res := a.Results[m.Name()]
		status := "FAIL"
		if res.Passing {
			status = "PASS"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s: %.1f%% (Threshold: %.1f%%)\n", status, res.Name, res.Value*100, res.Threshold*100))
	}
	return sb.String()
}
