package compare

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
)

// filterNonSummaryTasks returns only tasks whose IsSummary flag is false.
// Summary (rollup) tasks carry no independent predecessor/successor logic and
// must not be included in delta scoring or pillar calculations.
//
// Signature: filterNonSummaryTasks(tasks []*model.Task) []*model.Task
// Arguments:
//   - tasks: full task slice from a parsed Schedule
//
// Returns: new slice containing only work/detail tasks (IsSummary == false)
func filterNonSummaryTasks(tasks []*model.Task) []*model.Task {
	result := make([]*model.Task, 0, len(tasks))
	for _, t := range tasks {
		if !t.IsSummary {
			result = append(result, t)
		}
	}
	return result
}

// CompareSchedules compares two schedules and returns a benchmark result.
// Summary (rollup) tasks are excluded from both snapshots before comparison
// because they aggregate child values and have no independent logic links.
func CompareSchedules(prev, curr *model.Schedule) *BenchmarkResult {
	// 1. Indexing — work/detail tasks only
	prevMap := make(map[string]*model.Task)
	duplicates := make(map[string]bool)
	for _, t := range filterNonSummaryTasks(prev.Tasks) {
		if _, exists := prevMap[t.TaskID]; exists {
			duplicates[t.TaskID] = true
		}
		prevMap[t.TaskID] = t
	}

	currMap := make(map[string]*model.Task)
	for _, t := range filterNonSummaryTasks(curr.Tasks) {
		if _, exists := currMap[t.TaskID]; exists {
			duplicates[t.TaskID] = true
		}
		currMap[t.TaskID] = t
	}

	var warnings []string
	for id := range duplicates {
		warnings = append(warnings, fmt.Sprintf("duplicate TaskID %q found — only last occurrence used", id))
	}

	// 2. Identify All Unique IDs
	allIDs := make(map[string]bool)
	for id := range prevMap {
		allIDs[id] = true
	}
	for id := range currMap {
		allIDs[id] = true
	}

	// 3. Calculate Deltas
	var deltas []*TaskDelta
	newTasks := 0
	deletedTasks := 0
	modifiedTasks := 0
	unchangedTasks := 0
	ghostTasks := 0

	// For Pillar Calculation
	totalTasks := len(allIDs)
	if totalTasks == 0 {
		return &BenchmarkResult{
			TotalTasks: 0,
			Warnings:   warnings,
		}
	}
	countFVGT2 := 0.0   // Count for Finish Variance > 2 days (weighted if milestone)
	countDurGrowth := 0 // Count for Duration increased > 10%

	// Friction Aggregation
	frictionMap := make(map[string]int)

	for id := range allIDs {
		p, inPrev := prevMap[id]
		c, inCurr := currMap[id]

		delta := &TaskDelta{
			TaskID: id,
			Status: StatusUnchanged,
		}

		if inCurr {
			delta.Name = c.Name
			delta.WBS = c.WBS
			delta.IsMilestone = c.IsMilestone
			delta.CurrStart = c.Start
		} else {
			delta.Name = p.Name
			delta.WBS = p.WBS
			delta.IsMilestone = p.IsMilestone
			delta.PrevStart = p.Start
		}

		if !inPrev && inCurr {
			delta.Status = StatusNew
			newTasks++
			// Check Ghost Task
			if isGhost(c, curr.DataDate) {
				delta.IsGhostTask = true
				ghostTasks++
				addToFriction(frictionMap, c.WBS)
			}
		} else if inPrev && !inCurr {
			delta.Status = StatusDeleted
			deletedTasks++
		} else {
			// In Both
			delta.PrevStart = p.Start
			if diff(p, c) {
				delta.Status = StatusModified
				modifiedTasks++
			} else {
				unchangedTasks++
			}

			// Metrics
			delta.DurationDelta = c.Duration - p.Duration
			delta.PrevDuration = p.Duration
			delta.ExecutionDelta = c.PercentComplete - p.PercentComplete

			// Finish Variance
			if c.Finish != nil && p.Finish != nil {
				// Duration in hours usually, but here model says float64 Days?
				// Wait, model.go doesn't specify unit, but usually duration is days.
				// Let's assume Finish is Time.
				diffDuration := c.Finish.Sub(*p.Finish).Hours() / 24.0
				delta.FinishVariance = diffDuration
			}

			// Pillar A Checks
			if delta.FinishVariance > 2.0 {
				weight := 1.0
				if c.IsMilestone {
					weight = 2.0
				}
				countFVGT2 += weight
			}

			// Pillar B Checks
			if c.Duration > p.Duration {
				growth := (c.Duration - p.Duration) / p.Duration
				if p.Duration == 0 {
					if c.Duration > 0 {
						growth = 1.0 // Infinite growth effectively
					} else {
						growth = 0
					}
				}
				if growth > 0.10 {
					countDurGrowth++
				}
			}

			// Check Ghost Task
			if isGhost(c, curr.DataDate) {
				delta.IsGhostTask = true
				ghostTasks++
				addToFriction(frictionMap, c.WBS)
			}
		}

		// Assign Visual Symbology & Impact
		AssignSymbology(delta)

		deltas = append(deltas, delta)
	}

	// 4. Calculate Scores

	// Pillar A: Schedule Stability (40%)
	// Penalty: 1 pt per 1% of tasks with FV > 2d
	pctFV := (float64(countFVGT2) / float64(totalTasks)) * 100
	pillarAPenalty := pctFV * 1.0
	// Clamp penalty
	pillarAScore := math.Max(0, 40-pillarAPenalty)

	// Pillar B: Duration Reliability (30%)
	// Penalty: 1.5 pt per 1% of tasks with Duration Increase > 10%
	// Note: Logic says "tasks" not "total tasks", but usually means total tasks in scope.
	pctDur := (float64(countDurGrowth) / float64(totalTasks)) * 100
	pillarBPenalty := pctDur * 1.5
	pillarBScore := math.Max(0, 30-pillarBPenalty)

	// Pillar C: Scope Churn (30%)
	// Penalty: 2 pts per 1% Task Churn (Added + Deleted / Total)
	churnCount := newTasks + deletedTasks
	pctChurn := (float64(churnCount) / float64(totalTasks)) * 100
	pillarCPenalty := pctChurn * 2.0
	pillarCScore := math.Max(0, 30-pillarCPenalty)

	overallScore := pillarAScore + pillarBScore + pillarCScore

	// 5. Friction Index (Sort by count)
	var frictionItems []FrictionItem
	for wbs, count := range frictionMap {
		// Aggregate to top level WBS if possible?
		// Specification says: "Rank WBS levels"
		// Let's just output the stored map keys for now.
		// Ideally we would roll up numbers (e.g. 1.1 ghost counts towards 1), but sticking to direct for now.
		frictionItems = append(frictionItems, FrictionItem{WBS: wbs, GhostTaskCount: count})
	}
	sort.Slice(frictionItems, func(i, j int) bool {
		return frictionItems[i].GhostTaskCount > frictionItems[j].GhostTaskCount
	})

	return &BenchmarkResult{
		OverallScore:          overallScore,
		PillarAScore:          pillarAScore,
		PillarBScore:          pillarBScore,
		PillarCScore:          pillarCScore,
		TotalTasks:            totalTasks,
		NewTasks:              newTasks,
		DeletedTasks:          deletedTasks,
		ModifiedTasks:         modifiedTasks,
		UnchangedTasks:        unchangedTasks,
		GhostTasksCount:       ghostTasks,
		DurationInflatedCount: countDurGrowth,
		DurationInflatedPct:   pctDur,
		TaskDeltas:            deltas,
		FrictionIndex:         frictionItems,
		Warnings:              warnings,
	}
}

func diff(p, c *model.Task) bool {
	// Simple check for modification
	if p.Name != c.Name {
		return true
	}
	if p.Duration != c.Duration {
		return true
	}
	if p.PercentComplete != c.PercentComplete {
		return true
	}
	// Dates
	pStart := time.Time{}
	if p.Start != nil {
		pStart = *p.Start
	}
	cStart := time.Time{}
	if c.Start != nil {
		cStart = *c.Start
	}
	if !pStart.Equal(cStart) {
		return true
	}

	pFinish := time.Time{}
	if p.Finish != nil {
		pFinish = *p.Finish
	}
	cFinish := time.Time{}
	if c.Finish != nil {
		cFinish = *c.Finish
	}
	if !pFinish.Equal(cFinish) {
		return true
	}

	return false
}

func isGhost(t *model.Task, statusDate time.Time) bool {
	// Start Date < Status Date AND % Complete == 0
	if t.Start == nil {
		return false
	}
	return t.Start.Before(statusDate) && t.PercentComplete == 0
}

func addToFriction(m map[string]int, wbs string) {
	if wbs == "" {
		m["(No WBS)"]++
		return
	}
	// Innovation: Aggregate by Top-Level WBS for better reporting
	// Assuming WBS is like "1", "1.2", "1.2.3". Top level is "1".
	parts := strings.Split(wbs, ".")
	topLevel := parts[0]
	m[topLevel]++
}
