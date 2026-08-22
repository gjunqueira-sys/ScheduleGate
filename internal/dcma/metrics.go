package dcma

import (
	"fmt"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
)

// --- Logic Metric ---
type LogicMetric struct{}

func (m *LogicMetric) ID() int      { return 1 }
func (m *LogicMetric) Name() string { return "Logic" }
func (m *LogicMetric) Description() string {
	return "Percentage of incomplete tasks missing a predecessor and/or successor (PAM 4.1)"
}
func (m *LogicMetric) Threshold() float64 { return 0.05 }

// Assess evaluates PAM 4.1: missing predecessor or successor for incomplete tasks.
// Returns:
//
//	MetricResult with Value = missing/total, Passing = value ≤ 5%, and Exceptions
//	listing each offending task with a condition describing what is missing and what
//	to do to correct it.
func (m *LogicMetric) Assess(s *model.Schedule) MetricResult {
	// Build successor map by scanning every non-summary task's Predecessors field.
	// Milestones ARE included here: a work task whose only outgoing link leads to a
	// milestone phase gate is fully connected and must not be flagged. Summary
	// roll-up rows are still excluded because their inherited predecessor fields do
	// not represent real network relationships.
	hasSuccessor := make(map[string]bool)
	for _, t := range s.Tasks {
		if t.IsSummary {
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

	// Build an inactive-successor map so flagged tasks whose only referencer
	// is an inactive task can be self-explaining in the report. Each predID
	// maps to the inactive tasks that listed it as a predecessor.
	inactiveRefs := make(map[string][]*model.Task)
	for _, t := range s.InactiveTasks {
		if !hasContent(t.Predecessors) {
			continue
		}
		for _, p := range splitPredecessors(t.Predecessors) {
			if id := extractTaskID(strings.TrimSpace(p)); id != "" {
				inactiveRefs[id] = append(inactiveRefs[id], t)
			}
		}
	}

	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}

	missing := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		noPred := !hasContent(t.Predecessors)
		noSucc := !hasSuccessor[t.TaskID]
		if noPred || noSucc {
			missing++
			var cond string
			switch {
			case noPred && noSucc:
				cond = "No predecessor or successor — task is an island; add both incoming and outgoing Finish-to-Start logic links"
			case noPred:
				cond = "No predecessor — add a Finish-to-Start link from a preceding task"
			default:
				cond = "No successor — add a Finish-to-Start link to a following task"
			}
			if noSucc {
				if note := inactiveReferrerNote(inactiveRefs[t.TaskID]); note != "" {
					cond += "  " + note
				}
			}
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	val := float64(missing) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"missing": missing, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator:   missing,
			Denominator: len(incompleteTasks),
			Unit:        "tasks",
			ScopeLabel:  scope,
			Excluded:    exclusions,
		},
	}
}

// --- Leads Metric ---
type LeadsMetric struct{}

func (m *LeadsMetric) ID() int      { return 2 }
func (m *LeadsMetric) Name() string { return "Leads" }
func (m *LeadsMetric) Description() string {
	return "Percentage of logic links with leads (negative lag) for incomplete tasks (PAM 4.2)"
}
func (m *LeadsMetric) Threshold() float64 { return 0.00 }

// Assess evaluates PAM 4.2: logic links with negative lag on incomplete tasks.
// Returns:
//
//	MetricResult with Value = leads/totalRels, Passing = value == 0, and
//	Exceptions listing one entry per offending relationship (not per task) with
//	a condition identifying the predecessor and lag value plus corrective guidance.
func (m *LeadsMetric) Assess(s *model.Schedule) MetricResult {
	stats := scanRelationships(s)
	var exceptions []TaskException
	leads := 0
	for _, r := range stats.relationships {
		if hasNegativeLag(r.token) {
			leads++
			predID := extractTaskID(r.token)
			lagVal := extractLagToken(r.token)
			cond := fmt.Sprintf(
				"Predecessor %s has %s lead (negative lag) — replace with a true logic relationship or adjust task dates instead",
				predID, lagVal,
			)
			exceptions = append(exceptions, TaskException{TaskID: r.taskID, Name: r.taskName, Condition: cond})
		}
	}
	pop := stats.population(leads, "Predecessor relationships on incomplete non-summary, non-milestone work tasks")
	if stats.totalRels == 0 {
		return MetricResult{Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true, Population: pop}
	}
	val := float64(leads) / float64(stats.totalRels)
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"leads": leads, "total": stats.totalRels},
		Exceptions: exceptions,
		Population: pop,
	}
}

// --- Lags Metric ---
type LagsMetric struct{}

func (m *LagsMetric) ID() int      { return 3 }
func (m *LagsMetric) Name() string { return "Lags" }
func (m *LagsMetric) Description() string {
	return "Percentage of logic links with lags (positive lag) for incomplete tasks (PAM 4.3)"
}
func (m *LagsMetric) Threshold() float64 { return 0.10 }

// Assess evaluates PAM 4.3: logic links with positive lag on incomplete tasks.
// Returns:
//
//	MetricResult with Value = lags/totalRels, Passing = value ≤ 10%, and
//	Exceptions with one entry per offending relationship pointing to the
//	predecessor and suggesting insertion of an intermediate task.
func (m *LagsMetric) Assess(s *model.Schedule) MetricResult {
	stats := scanRelationships(s)
	var exceptions []TaskException
	lags := 0
	for _, r := range stats.relationships {
		if hasPositiveLag(r.token) {
			lags++
			predID := extractTaskID(r.token)
			lagVal := extractLagToken(r.token)
			cond := fmt.Sprintf(
				"Predecessor %s has %s lag — insert an intermediate task to represent the delay rather than using a lag",
				predID, lagVal,
			)
			exceptions = append(exceptions, TaskException{TaskID: r.taskID, Name: r.taskName, Condition: cond})
		}
	}
	pop := stats.population(lags, "Predecessor relationships on incomplete non-summary, non-milestone work tasks")
	if stats.totalRels == 0 {
		return MetricResult{Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true, Population: pop}
	}
	val := float64(lags) / float64(stats.totalRels)
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"lags": lags, "total": stats.totalRels},
		Exceptions: exceptions,
		Population: pop,
	}
}

// --- Relationship Types Metric ---
type RelationshipTypesMetric struct{}

func (m *RelationshipTypesMetric) ID() int      { return 4 }
func (m *RelationshipTypesMetric) Name() string { return "Relationship Types" }
func (m *RelationshipTypesMetric) Description() string {
	return "Percentage of FS relationship types across all logic links for incomplete tasks (PAM 4.4)"
}
func (m *RelationshipTypesMetric) Threshold() float64 { return 0.90 }

// Assess evaluates PAM 4.4: proportion of FS relationships among incomplete tasks.
// Returns:
//
//	MetricResult with Value = fsRels/totalRels, Passing = value ≥ 90%, and
//	Exceptions listing each non-FS relationship with its type and corrective note.
func (m *RelationshipTypesMetric) Assess(s *model.Schedule) MetricResult {
	stats := scanRelationships(s)
	fsRels := 0
	var exceptions []TaskException
	for _, r := range stats.relationships {
		relType := extractRelType(r.token)
		if relType == "FS" || relType == "" {
			fsRels++
			continue
		}
		predID := extractTaskID(r.token)
		cond := fmt.Sprintf(
			"%s dependency to Predecessor %s — review logic; ≥90%% of links should be Finish-to-Start",
			relType, predID,
		)
		exceptions = append(exceptions, TaskException{TaskID: r.taskID, Name: r.taskName, Condition: cond})
	}
	pop := stats.population(fsRels, "Predecessor relationships on incomplete non-summary, non-milestone work tasks (numerator = FS relationships)")
	if stats.totalRels == 0 {
		return MetricResult{Name: m.Name(), Value: 1.0, Threshold: m.Threshold(), Passing: true, Population: pop}
	}
	val := float64(fsRels) / float64(stats.totalRels)
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val >= m.Threshold(),
		Details:    map[string]interface{}{"fs": fsRels, "total": stats.totalRels},
		Exceptions: exceptions,
		Population: pop,
	}
}

// --- Hard Constraints Metric ---
type HardConstraintsMetric struct{}

func (m *HardConstraintsMetric) ID() int      { return 5 }
func (m *HardConstraintsMetric) Name() string { return "Hard Constraints" }
func (m *HardConstraintsMetric) Description() string {
	return "Percentage of incomplete tasks with hard constraints (PAM 4.5)"
}
func (m *HardConstraintsMetric) Threshold() float64 { return 0.05 }

// Assess evaluates PAM 4.5: hard constraints on incomplete non-summary tasks.
// Returns:
//
//	MetricResult with Value = constrained/total, Passing = value ≤ 5%, and
//	Exceptions listing each task with its constraint type, date, and a note
//	explaining that hard constraints override schedule logic.
func (m *HardConstraintsMetric) Assess(s *model.Schedule) MetricResult {
	// Milestones are excluded from the Hard Constraints population for the same
	// reason they are excluded from High Float, Negative Float, and High Duration:
	// contractual milestones legitimately carry "Must Finish On" constraints (that
	// is their scheduling purpose) and should not inflate the violation count.
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}

	hardTypes := []string{
		"Must Finish On", "MFO",
		"Must Start On", "MSO",
		"Start No Later Than", "SNLT",
		"Finish No Later Than", "FNLT",
	}

	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		for _, ht := range hardTypes {
			if strings.EqualFold(t.ConstraintType, ht) {
				count++
				dateStr := ""
				if t.ConstraintDate != nil {
					dateStr = " on " + t.ConstraintDate.Format("2006-01-02")
				}
				cond := fmt.Sprintf(
					"Hard constraint '%s'%s — remove or replace with ASAP/SNET; hard constraints override schedule logic and inflate float",
					t.ConstraintType, dateStr,
				)
				exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
				break
			}
		}
	}

	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(incompleteTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- High Float Metric ---
type HighFloatMetric struct{}

func (m *HighFloatMetric) ID() int      { return 6 }
func (m *HighFloatMetric) Name() string { return "High Float" }
func (m *HighFloatMetric) Description() string {
	return "Percentage of incomplete tasks with total float > 44 working days (PAM 4.6)"
}
func (m *HighFloatMetric) Threshold() float64 { return 0.05 }

// Assess evaluates PAM 4.6: total float exceeding 44 days for incomplete tasks.
// Returns:
//
//	MetricResult with Value = highFloat/total, Passing = value ≤ 5%, and
//	Exceptions listing each task's float value with guidance on root causes.
func (m *HighFloatMetric) Assess(s *model.Schedule) MetricResult {
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}
	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		if t.TotalSlack > 44 {
			count++
			cond := fmt.Sprintf(
				"Total Float %.0fd exceeds 44d limit — check for missing successors, a disconnected path, or an overly-late project finish milestone",
				t.TotalSlack,
			)
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(incompleteTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- Negative Float Metric ---
type NegativeFloatMetric struct{}

func (m *NegativeFloatMetric) ID() int      { return 7 }
func (m *NegativeFloatMetric) Name() string { return "Negative Float" }
func (m *NegativeFloatMetric) Description() string {
	return "Percentage of incomplete tasks with negative total float (PAM 4.7)"
}
func (m *NegativeFloatMetric) Threshold() float64 { return 0.05 }

// Assess evaluates PAM 4.7: negative total float for incomplete tasks.
// Returns:
//
//	MetricResult with Value = negFloat/total, Passing = value ≤ 5%, and
//	Exceptions listing each task's float value with guidance pointing to
//	likely root causes (hard constraints, long durations, compressed logic).
func (m *NegativeFloatMetric) Assess(s *model.Schedule) MetricResult {
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}
	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		if t.TotalSlack < 0 {
			count++
			cond := fmt.Sprintf(
				"Negative Float %.0fd — driven by hard constraint, long duration, or compressed logic; review constraints and predecessor links on this task",
				t.TotalSlack,
			)
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(incompleteTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- High Duration Metric ---
type HighDurationMetric struct{}

func (m *HighDurationMetric) ID() int      { return 8 }
func (m *HighDurationMetric) Name() string { return "High Duration" }
func (m *HighDurationMetric) Description() string {
	return "Percentage of incomplete tasks with baseline duration > 60 working days (PAM 4.8)"
}
func (m *HighDurationMetric) Threshold() float64 { return 0.10 }

// Assess evaluates PAM 4.8: baseline duration exceeding 44 days for incomplete tasks.
// Returns:
//
//	MetricResult with Value = highDuration/total, Passing = value ≤ 5%, and
//	Exceptions listing each task's duration with guidance to decompose the task.
func (m *HighDurationMetric) Assess(s *model.Schedule) MetricResult {
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}
	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		if t.BaselineDuration > 60 {
			count++
			cond := fmt.Sprintf(
				"Baseline Duration %.0fd exceeds 60d limit — break this work package into tasks of ≤60 working days each",
				t.BaselineDuration,
			)
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(incompleteTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- Invalid Dates Metric ---
type InvalidDatesMetric struct{}

func (m *InvalidDatesMetric) ID() int      { return 9 }
func (m *InvalidDatesMetric) Name() string { return "Invalid Dates" }
func (m *InvalidDatesMetric) Description() string {
	return "Percentage of activities with invalid dates"
}
func (m *InvalidDatesMetric) Threshold() float64 { return 0.00 }

// Assess evaluates PAM 4.9: invalid date combinations on incomplete work tasks.
// Returns:
//
//	MetricResult with Value = invalid/total, Passing = value == 0, and
//	Exceptions listing each task with a specific description of which date is
//	out of range relative to the data date, plus corrective guidance.
func (m *InvalidDatesMetric) Assess(s *model.Schedule) MetricResult {
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks (forecast/actual dates checked against status date)"
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}
	dd := s.DataDate
	ddStr := dd.Format("2006-01-02")
	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		var reasons []string
		// In MS Project, once a task starts the "Start" field is set to the actual
		// start date, which will always be before the data date. Flagging it as a
		// forecast-start violation is a false positive. Only check forecast start
		// when no actual start has been recorded (task is not yet started).
		if t.ActualStart == nil && t.Start != nil && t.Start.Before(dd) {
			reasons = append(reasons, fmt.Sprintf("Forecast Start %s is before data date %s — update the schedule or record an actual start date", t.Start.Format("2006-01-02"), ddStr))
		}
		// Forecast finish can still be checked for in-progress tasks: an incomplete
		// task whose forecast finish has already passed is genuinely overdue.
		if t.Finish != nil && t.Finish.Before(dd) {
			reasons = append(reasons, fmt.Sprintf("Forecast Finish %s is before data date %s — update the schedule or record an actual finish date", t.Finish.Format("2006-01-02"), ddStr))
		}
		if t.ActualStart != nil && t.ActualStart.After(dd) {
			reasons = append(reasons, fmt.Sprintf("Actual Start %s is after data date %s — correct the actual start or update the status date", t.ActualStart.Format("2006-01-02"), ddStr))
		}
		if t.ActualFinish != nil && t.ActualFinish.After(dd) {
			reasons = append(reasons, fmt.Sprintf("Actual Finish %s is after data date %s — correct the actual finish or update the status date", t.ActualFinish.Format("2006-01-02"), ddStr))
		}
		if len(reasons) > 0 {
			count++
			exceptions = append(exceptions, TaskException{
				TaskID:    t.TaskID,
				Name:      t.Name,
				Condition: strings.Join(reasons, "; "),
			})
		}
	}
	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(incompleteTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- Resources Metric ---
type ResourcesMetric struct{}

func (m *ResourcesMetric) ID() int      { return 10 }
func (m *ResourcesMetric) Name() string { return "Resources" }
func (m *ResourcesMetric) Description() string {
	return "Percentage of activities with resources assigned"
}
func (m *ResourcesMetric) Threshold() float64 { return 0.95 }

// Assess evaluates PAM 4.10: resource assignment on incomplete work tasks.
// Returns:
//
//	MetricResult with Value = resourced/total, Passing = value ≥ 95%, and
//	Exceptions listing each task missing a resource with a note to assign one
//	for cost and work tracking. Returns NotApplicable when no resource column
//	exists in the export.
func (m *ResourcesMetric) Assess(s *model.Schedule) MetricResult {
	incompleteTasks, exclusions := incompleteWorkTaskFunnel(s)
	scope := "Incomplete non-summary, non-milestone work tasks (numerator = tasks WITH resources assigned)"
	pop := &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions}
	if len(incompleteTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 1, Threshold: m.Threshold(), Passing: true,
			Population: pop,
		}
	}
	count := 0
	var exceptions []TaskException
	for _, t := range incompleteTasks {
		if hasContent(t.Resources) {
			count++
		} else {
			exceptions = append(exceptions, TaskException{
				TaskID:    t.TaskID,
				Name:      t.Name,
				Condition: "No resources assigned — assign at least one resource to enable cost and work tracking",
			})
		}
	}

	if count == 0 {
		anyResources := false
		for _, t := range filterWorkTasks(s.Tasks) {
			if hasContent(t.Resources) {
				anyResources = true
				break
			}
		}
		if !anyResources {
			pop.ScopeLabel = "Not Applicable — no Resource Names column present in this export"
			return MetricResult{
				Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true, NotApplicable: true,
				Population: pop,
			}
		}
	}

	pop.Numerator = count
	pop.Denominator = len(incompleteTasks)
	val := float64(count) / float64(len(incompleteTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val >= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(incompleteTasks)},
		Exceptions: exceptions,
		Population: pop,
	}
}

// --- Missed Tasks Metric ---
type MissedTasksMetric struct{}

func (m *MissedTasksMetric) ID() int      { return 11 }
func (m *MissedTasksMetric) Name() string { return "Missed Tasks" }
func (m *MissedTasksMetric) Description() string {
	return "Percentage of incomplete tasks past their finish date"
}
func (m *MissedTasksMetric) Threshold() float64 { return 0.05 }

// Assess evaluates PAM 4.11: tasks whose baseline finish has passed but are not complete.
// Returns:
//
//	MetricResult with Value = missed/dueCount, Passing = value ≤ 5%, and
//	Exceptions listing each task with its baseline finish date and current
//	completion percentage, prompting the user to update progress or revise baseline.
func (m *MissedTasksMetric) Assess(s *model.Schedule) MetricResult {
	// Per DCMA PAM 4.11 the universe is "tasks with baseline finish on or
	// before status date that are not complete". Milestones (contractual
	// phase gates, kickoffs, go-lives, etc.) are explicitly part of that
	// universe — a missed milestone is the most common DCMA-flagged item.
	// Only summary roll-up rows are excluded.
	summary, missingBaseline, futureBaseline := 0, 0, 0
	var dueTasks []*model.Task
	for _, t := range s.Tasks {
		switch {
		case t.IsSummary:
			summary++
		case t.BaselineFinish == nil:
			missingBaseline++
		case t.BaselineFinish.After(s.DataDate):
			futureBaseline++
		default:
			dueTasks = append(dueTasks, t)
		}
	}
	scope := "All tasks (work + milestones) whose Baseline Finish ≤ Status Date (per DCMA PAM 4.11)"
	exclusions := []ExclusionStep{
		{Reason: "Summary rows excluded", Count: summary},
		{Reason: "Tasks with no Baseline Finish", Count: missingBaseline},
		{Reason: "Tasks with Baseline Finish after Status Date (not yet due)", Count: futureBaseline},
	}
	if len(dueTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusions},
		}
	}
	count := 0
	var exceptions []TaskException
	for _, t := range dueTasks {
		if t.PercentComplete < 100 {
			count++
			cond := fmt.Sprintf(
				"Baseline Finish %s not met (%.0f%% complete) — update actual progress or revise the baseline to reflect the current plan",
				t.BaselineFinish.Format("2006-01-02"), t.PercentComplete,
			)
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	val := float64(count) / float64(len(dueTasks))
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val <= m.Threshold(),
		Details:    map[string]interface{}{"count": count, "total": len(dueTasks)},
		Exceptions: exceptions,
		Population: &Population{
			Numerator: count, Denominator: len(dueTasks),
			Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
		},
	}
}

// --- Critical Path Test Metric ---
type CriticalPathTestMetric struct{}

func (m *CriticalPathTestMetric) ID() int             { return 12 }
func (m *CriticalPathTestMetric) Name() string        { return "Critical Path Test" }
func (m *CriticalPathTestMetric) Description() string { return "Critical path validity test" }
func (m *CriticalPathTestMetric) Threshold() float64  { return 1.0 }

// Assess evaluates whether a valid critical path exists (any non-summary work task
// with TotalSlack ≤ 0). Summary tasks are excluded because their TotalSlack values
// are derived roll-ups, not independent network-driven float calculations.
// Returns:
//
//	MetricResult as a pass/fail with no per-task Exceptions; the metric is
//	binary — either a critical path exists or it does not.
func (m *CriticalPathTestMetric) Assess(s *model.Schedule) MetricResult {
	criticalCount := 0
	workTaskCount := 0
	for _, t := range filterWorkTasks(s.Tasks) {
		workTaskCount++
		if t.TotalSlack <= 0 {
			criticalCount++
		}
	}
	passing := criticalCount > 0
	val := 0.0
	if passing {
		val = 1.0
	}
	scope := fmt.Sprintf("Schedule-level binary test — passes when ≥1 work task has TotalSlack ≤ 0 (found %d of %d work tasks on the critical path)", criticalCount, workTaskCount)
	return MetricResult{
		Name:      m.Name(),
		Value:     val,
		Threshold: m.Threshold(),
		Passing:   passing,
		Details:   map[string]interface{}{"critical_tasks": criticalCount},
		Population: &Population{
			Numerator:   criticalCount,
			Denominator: workTaskCount,
			Unit:        "schedule",
			ScopeLabel:  scope,
		},
	}
}

// --- CPLI Metric ---
type CPLIMetric struct{}

func (m *CPLIMetric) ID() int      { return 13 }
func (m *CPLIMetric) Name() string { return "Critical Path Length Index" }
func (m *CPLIMetric) Description() string {
	return "Ratio of remaining to planned critical path duration"
}
func (m *CPLIMetric) Threshold() float64 { return 1.0 }

// Assess evaluates PAM 3.1.2.3: CPLI = (CPL + TF) / CPL.
//
// CPL  = Critical Path Length — remaining working days from data date to project
//
//	end, approximated as calendar days × 5/7.
//
// TF   = Total Float of the project completion task on the critical path.
//
// The completion task is selected in two passes:
//  1. Prefer the latest-finishing non-summary task with TotalSlack ≤ 0 (on the
//     critical path). This gives the most accurate TF for the CPLI formula.
//  2. If no critical task exists (all tasks have positive float), fall back to the
//     overall latest-finishing non-summary task. Without this fallback the metric
//     would always fail on schedules where the critical path test is passing.
//
// Returns:
//
//	MetricResult with a computed index value; no per-task Exceptions are
//	produced because CPLI is a schedule-level index, not a task-level flag.
func (m *CPLIMetric) Assess(s *model.Schedule) MetricResult {
	var criticalCompletion *model.Task  // latest-finish task with TotalSlack ≤ 0
	var overallCompletion *model.Task   // latest-finish task regardless of slack

	for _, t := range s.Tasks {
		if t.IsSummary || t.Finish == nil {
			continue
		}
		if overallCompletion == nil || t.Finish.After(*overallCompletion.Finish) {
			overallCompletion = t
		}
		if t.TotalSlack <= 0 {
			if criticalCompletion == nil || t.Finish.After(*criticalCompletion.Finish) {
				criticalCompletion = t
			}
		}
	}

	completionTask := criticalCompletion
	if completionTask == nil {
		completionTask = overallCompletion
	}
	scope := "Schedule-level index — computed from the project completion task (latest critical-path or latest non-summary task)"
	if completionTask == nil {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: false,
			Population: &Population{Unit: "schedule", ScopeLabel: scope + " — no completion task found"},
		}
	}

	calDays := completionTask.Finish.Sub(s.DataDate).Hours() / 24
	cpl := calDays * 5 / 7
	if cpl <= 0 {
		return MetricResult{
			Name: m.Name(), Value: 1.0, Threshold: m.Threshold(), Passing: true,
			Population: &Population{Unit: "schedule", ScopeLabel: scope + " — completion task is on or before the status date"},
		}
	}
	tf := completionTask.TotalSlack
	val := (cpl + tf) / cpl
	return MetricResult{
		Name:      m.Name(),
		Value:     val,
		Threshold: m.Threshold(),
		Passing:   val >= m.Threshold(),
		Details:   map[string]interface{}{"cpl": cpl, "tf": tf},
		Population: &Population{
			Unit:       "schedule",
			ScopeLabel: fmt.Sprintf("%s; CPL=%.0fd, TF=%.0fd", scope, cpl, tf),
		},
	}
}

// --- BEI Metric ---
type BEIMetric struct{}

func (m *BEIMetric) ID() int             { return 14 }
func (m *BEIMetric) Name() string        { return "Baseline Execution Index" }
func (m *BEIMetric) Description() string { return "Ratio of earned value to planned value" }
func (m *BEIMetric) Threshold() float64  { return 0.95 }

// Assess evaluates DCMA BEI = completed / tasks baselined to finish by status date.
// Returns:
//
//	MetricResult with Value = completed/baselineCount, Passing = value ≥ 95%, and
//	Exceptions listing each task that was planned to be done by the status date
//	but is not yet complete, with a note to update progress or re-baseline.
func (m *BEIMetric) Assess(s *model.Schedule) MetricResult {
	// Per DAU APMT-009 the BEI universe is "tasks baselined to finish by
	// status date" — milestones are explicitly part of that universe because
	// they represent contractual completion gates. Only summary roll-up rows
	// are excluded; milestones are kept in both numerator (completed tasks)
	// and denominator (baseline-due tasks).
	summaryRows := 0
	var assessableTasks []*model.Task
	for _, t := range s.Tasks {
		if t.IsSummary {
			summaryRows++
			continue
		}
		assessableTasks = append(assessableTasks, t)
	}
	scope := "Numerator: ALL completed tasks (work + milestones). Denominator: tasks whose Baseline Finish ≤ Status Date (per DAU APMT-009)."
	exclusionsBase := []ExclusionStep{
		{Reason: "Summary rows excluded", Count: summaryRows},
	}
	if len(assessableTasks) == 0 {
		return MetricResult{
			Name: m.Name(), Value: 0, Threshold: m.Threshold(), Passing: false,
			Population: &Population{Unit: "tasks", ScopeLabel: scope, Excluded: exclusionsBase},
		}
	}
	baselineCount := 0
	completed := 0
	missingBaseline := 0
	futureBaseline := 0
	var exceptions []TaskException
	for _, t := range assessableTasks {
		if t.PercentComplete >= 100 {
			completed++
		}
		if t.BaselineFinish == nil {
			missingBaseline++
			continue
		}
		if t.BaselineFinish.After(s.DataDate) {
			futureBaseline++
			continue
		}
		baselineCount++
		if t.PercentComplete < 100 {
			cond := fmt.Sprintf(
				"Planned finish %s not met (%.0f%% complete) — record actual progress or re-baseline; each incomplete task depresses BEI",
				t.BaselineFinish.Format("2006-01-02"), t.PercentComplete,
			)
			exceptions = append(exceptions, TaskException{TaskID: t.TaskID, Name: t.Name, Condition: cond})
		}
	}
	exclusions := append(exclusionsBase,
		ExclusionStep{Reason: "Tasks with no Baseline Finish (excluded from denominator)", Count: missingBaseline},
		ExclusionStep{Reason: "Tasks with Baseline Finish after Status Date (not yet due)", Count: futureBaseline},
	)
	pop := &Population{
		Numerator: completed, Denominator: baselineCount,
		Unit: "tasks", ScopeLabel: scope, Excluded: exclusions,
	}
	if baselineCount == 0 {
		return MetricResult{
			Name:       m.Name(),
			Value:      1.0,
			Threshold:  m.Threshold(),
			Passing:    true,
			Details:    map[string]interface{}{"completed": completed, "baseline_count": baselineCount, "missing_baseline": missingBaseline},
			Population: pop,
		}
	}
	val := float64(completed) / float64(baselineCount)
	return MetricResult{
		Name:       m.Name(),
		Value:      val,
		Threshold:  m.Threshold(),
		Passing:    val >= m.Threshold(),
		Details:    map[string]interface{}{"completed": completed, "baseline_count": baselineCount, "missing_baseline": missingBaseline},
		Exceptions: exceptions,
		Population: pop,
	}
}

// --- Helpers ---

// inactiveReferrerNote returns a parenthetical diagnostic note listing
// inactive tasks that reference the flagged task as a predecessor, or "" when
// there are none. The note is appended to Logic metric condition strings so
// auditors can immediately see the "duplicate-name, active twin replaced an
// inactive original" pattern without re-opening the schedule.
//
// Example output:
//
//	"(note: also referenced by inactive task 5135 'Mech. PreComm. CA-1-2 ...';
//	 inactive tasks are excluded from the network — re-link to an active task)"
//
// Signature: inactiveReferrerNote(refs []*model.Task) string
// Arguments:
//   - refs: inactive tasks that list the flagged task as a predecessor
//
// Returns: a single-line note, or "" when refs is empty.
func inactiveReferrerNote(refs []*model.Task) string {
	if len(refs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(refs))
	for _, t := range refs {
		name := t.Name
		if len(name) > 60 {
			name = name[:57] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s '%s'", t.TaskID, name))
	}
	return fmt.Sprintf(
		"(note: also referenced by inactive task %s; inactive tasks are excluded from the network — re-link this task to an active successor)",
		strings.Join(parts, ", "),
	)
}

func filterWorkTasks(tasks []*model.Task) []*model.Task {
	var res []*model.Task
	for _, t := range tasks {
		if !t.IsSummary && !t.IsMilestone {
			res = append(res, t)
		}
	}
	return res
}

// relationshipRecord is one parsed predecessor token paired with the owning
// task identity so relationship-based metrics can report which task contains
// the offending link.
type relationshipRecord struct {
	taskID   string
	taskName string
	token    string
}

// relationshipScan holds the result of one walk of the schedule's predecessor
// strings for relationship-based metrics (Leads, Lags, Relationship Types).
// The same data backs all three metrics so we walk the tasks exactly once per
// metric call instead of duplicating the loop body three times.
type relationshipScan struct {
	relationships             []relationshipRecord
	totalRels                 int
	summaryExcluded           int
	milestoneExcluded         int
	completedExcluded         int
	noPredecessorTasks        int
	tasksContributingRels     int
}

// scanRelationships walks every task in the schedule and collects all
// predecessor tokens that belong to incomplete non-summary, non-milestone work
// tasks. Tasks excluded from the relationship universe are tallied by class
// so the Universe sheet can display a clean exclusion funnel.
//
// Signature: scanRelationships(s *model.Schedule) relationshipScan
// Arguments:
//   - s: parsed schedule
//
// Returns: a relationshipScan with the in-scope relationship list and
// exclusion counts.
func scanRelationships(s *model.Schedule) relationshipScan {
	var scan relationshipScan
	for _, t := range s.Tasks {
		switch {
		case t.IsSummary:
			scan.summaryExcluded++
			continue
		case t.IsMilestone:
			scan.milestoneExcluded++
			continue
		case t.PercentComplete >= 100:
			scan.completedExcluded++
			continue
		}
		if !hasContent(t.Predecessors) {
			scan.noPredecessorTasks++
			continue
		}
		contributed := false
		for _, p := range splitPredecessors(t.Predecessors) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			scan.totalRels++
			scan.relationships = append(scan.relationships, relationshipRecord{
				taskID:   t.TaskID,
				taskName: t.Name,
				token:    p,
			})
			contributed = true
		}
		if contributed {
			scan.tasksContributingRels++
		}
	}
	return scan
}

// population builds a Population describing this relationship scan's funnel.
//
// Signature: (relationshipScan).population(numerator int, scope string) *Population
// Arguments:
//   - numerator: relationships counted as violations (leads, lags, or FS rels)
//   - scope:     one-line description of what's in scope
//
// Returns: a Population value ready to attach to a MetricResult.
func (r relationshipScan) population(numerator int, scope string) *Population {
	return &Population{
		Numerator:   numerator,
		Denominator: r.totalRels,
		Unit:        "relationships",
		ScopeLabel:  scope,
		Excluded: []ExclusionStep{
			{Reason: "Summary rows excluded", Count: r.summaryExcluded},
			{Reason: "Milestones excluded", Count: r.milestoneExcluded},
			{Reason: "Completed work tasks excluded (% Complete = 100)", Count: r.completedExcluded},
			{Reason: "Incomplete work tasks with no predecessor (contribute no relationships)", Count: r.noPredecessorTasks},
		},
	}
}

// incompleteWorkTaskFunnel walks the schedule once and returns the in-scope
// incomplete work tasks together with the exclusion counts for the three
// task-class filters shared by most DCMA metrics (summary, milestone,
// completed). Used by Hard Constraints, High Float, Negative Float, High
// Duration.
//
// Signature: incompleteWorkTaskFunnel(s *model.Schedule) (tasks []*model.Task, excluded []ExclusionStep)
// Arguments:
//   - s: parsed schedule
//
// Returns:
//   - tasks:    the in-scope incomplete non-summary, non-milestone work tasks
//   - excluded: ordered ExclusionStep list ready to attach to a Population
func incompleteWorkTaskFunnel(s *model.Schedule) ([]*model.Task, []ExclusionStep) {
	summary, milestone, completed := 0, 0, 0
	var incomplete []*model.Task
	for _, t := range s.Tasks {
		switch {
		case t.IsSummary:
			summary++
		case t.IsMilestone:
			milestone++
		case t.PercentComplete >= 100:
			completed++
		default:
			incomplete = append(incomplete, t)
		}
	}
	return incomplete, []ExclusionStep{
		{Reason: "Summary rows excluded", Count: summary},
		{Reason: "Milestones excluded", Count: milestone},
		{Reason: "Completed work tasks excluded (% Complete = 100)", Count: completed},
	}
}



func hasContent(s string) bool {
	return len(strings.TrimSpace(s)) > 0 && !strings.EqualFold(s, "nan")
}

// splitPredecessors splits a predecessor string on any of the delimiters that
// MS Project uses when exporting to Excel or CSV: comma, semicolon, and cell-
// embedded newlines (LF or CRLF). Excel cells can contain literal \n characters
// when a task has multiple predecessors, so splitting on \n is required to avoid
// treating "3416\n3417" as a single unrecognised token.
func splitPredecessors(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	})
}

// extractTaskID extracts the leading numeric task ID from a predecessor token
// such as "5FS+2d", "16FS-3w", "22", "125FS+23d", "5SS".
func extractTaskID(pred string) string {
	i := 0
	for i < len(pred) && pred[i] >= '0' && pred[i] <= '9' {
		i++
	}
	return pred[:i]
}

// hasNegativeLag returns true when a predecessor token contains a negative lag (a lead).
func hasNegativeLag(pred string) bool {
	i := 0
	for i < len(pred) && pred[i] >= '0' && pred[i] <= '9' {
		i++
	}
	for i < len(pred) && (pred[i] >= 'A' && pred[i] <= 'Z' || pred[i] >= 'a' && pred[i] <= 'z') {
		i++
	}
	return i < len(pred) && pred[i] == '-'
}

// hasPositiveLag returns true when a predecessor token contains a positive lag.
func hasPositiveLag(pred string) bool {
	i := 0
	for i < len(pred) && pred[i] >= '0' && pred[i] <= '9' {
		i++
	}
	for i < len(pred) && (pred[i] >= 'A' && pred[i] <= 'Z' || pred[i] >= 'a' && pred[i] <= 'z') {
		i++
	}
	return i < len(pred) && pred[i] == '+'
}

// extractRelType returns the relationship-type letters from a predecessor token
// (e.g. "FS", "SS", "FF", "SF") in upper case, or "" if none are present.
func extractRelType(pred string) string {
	i := 0
	for i < len(pred) && pred[i] >= '0' && pred[i] <= '9' {
		i++
	}
	start := i
	for i < len(pred) && (pred[i] >= 'A' && pred[i] <= 'Z' || pred[i] >= 'a' && pred[i] <= 'z') {
		i++
	}
	return strings.ToUpper(pred[start:i])
}

// extractLagToken returns the lag portion of a predecessor token as a string,
// including its sign and unit (e.g. "-1d", "+3w", "-5d"). Returns "" if no
// lag is present. Used to build human-readable condition strings for Leads and
// Lags exceptions.
func extractLagToken(pred string) string {
	i := 0
	// skip task ID digits
	for i < len(pred) && pred[i] >= '0' && pred[i] <= '9' {
		i++
	}
	// skip optional relationship type letters
	for i < len(pred) && (pred[i] >= 'A' && pred[i] <= 'Z' || pred[i] >= 'a' && pred[i] <= 'z') {
		i++
	}
	if i >= len(pred) {
		return ""
	}
	return pred[i:] // everything after the rel-type is the sign + magnitude + unit
}
