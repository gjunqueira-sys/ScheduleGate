package report

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/rules"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
)

// WriteJSON marshals v with 2-space indentation to the given writer.
func WriteJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteJSONFile marshals v to a file, creating (or truncating) the path.
func WriteJSONFile(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteJSON(f, v)
}

// BuildAssessJSON converts an assessment into its machine-readable form.
// The overall score excludes N/A metrics, matching the terminal and HTML views.
func BuildAssessJSON(assessment *dcma.DCMAAssessment, scheduleName, statusDate string, warnings []string) *AssessJSONOutput {
	results := make([]MetricJSONResult, 0, len(assessment.Metrics))
	passed, total := 0, 0
	for _, m := range assessment.Metrics {
		res := assessment.Results[m.Name()]
		if !res.NotApplicable {
			total++
			if res.Passing {
				passed++
			}
		}
		results = append(results, MetricJSONResult{
			ID:             m.ID(),
			Name:           m.Name(),
			Description:    m.Description(),
			Value:          res.Value,
			Threshold:      res.Threshold,
			Passing:        res.Passing,
			NotApplicable:  res.NotApplicable,
			ExceptionCount: len(res.Exceptions),
		})
	}

	score := 0.0
	if total > 0 {
		score = float64(passed) / float64(total) * 100
	}

	pop := assessment.Population
	return &AssessJSONOutput{
		ScheduleName: scheduleName,
		StatusDate:   statusDate,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		ToolVersion:  version.Display(),
		OverallScore: score,
		PassedCount:  passed,
		TotalCount:   total,
		Results:      results,
		Population: SchedulePopulation{
			TotalRows:                          pop.TotalRows,
			SummaryRows:                        pop.SummaryRows,
			Milestones:                         pop.Milestones,
			WorkTasks:                          pop.WorkTasks,
			CompletedWorkTasks:                 pop.CompletedWorkTasks,
			IncompleteWorkTasks:                pop.IncompleteWorkTasks,
			WorkTasksWithBaselineFinish:        pop.WorkTasksWithBaselineFinish,
			WorkTasksBaselineDueByStatus:       pop.WorkTasksBaselineDueByStatus,
			AssessableTasks:                    pop.AssessableTasks,
			CompletedAssessableTasks:           pop.CompletedAssessableTasks,
			AssessableTasksWithBaselineFinish:  pop.AssessableTasksWithBaselineFinish,
			AssessableTasksBaselineDueByStatus: pop.AssessableTasksBaselineDueByStatus,
		},
		Warnings: warnings,
	}
}

// BuildCompareJSON converts a benchmark result into its machine-readable form.
func BuildCompareJSON(result *compare.BenchmarkResult, prevFile, currFile, statusDate string) *CompareJSONOutput {
	deltas := make([]TaskDeltaJSON, 0, len(result.TaskDeltas))
	for _, d := range result.TaskDeltas {
		deltas = append(deltas, TaskDeltaJSON{
			TaskID:         d.TaskID,
			Name:           d.Name,
			WBS:            d.WBS,
			Status:         string(d.Status),
			FinishVariance: d.FinishVariance,
			DurationDelta:  d.DurationDelta,
			PrevDuration:   d.PrevDuration,
			ExecutionDelta: d.ExecutionDelta,
			IsGhostTask:    d.IsGhostTask,
			IsMilestone:    d.IsMilestone,
			ImpactType:     d.ImpactType,
			ImpactMsg:      d.ImpactMsg,
		})
	}

	friction := make([]FrictionItemJSON, 0, len(result.FrictionIndex))
	for _, f := range result.FrictionIndex {
		friction = append(friction, FrictionItemJSON{WBS: f.WBS, GhostTaskCount: f.GhostTaskCount})
	}

	return &CompareJSONOutput{
		PreviousFile:          prevFile,
		CurrentFile:           currFile,
		StatusDate:            statusDate,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
		ToolVersion:           version.Display(),
		OverallScore:          result.OverallScore,
		StabilityScore:        result.PillarAScore,
		ReliabilityScore:      result.PillarBScore,
		ScopeChurnScore:       result.PillarCScore,
		TotalTasks:            result.TotalTasks,
		NewTasks:              result.NewTasks,
		DeletedTasks:          result.DeletedTasks,
		ModifiedTasks:         result.ModifiedTasks,
		UnchangedTasks:        result.UnchangedTasks,
		GhostTasksCount:       result.GhostTasksCount,
		DurationInflatedCount: result.DurationInflatedCount,
		DurationInflatedPct:   result.DurationInflatedPct,
		TaskDeltas:            deltas,
		FrictionIndex:         friction,
		Warnings:              result.Warnings,
	}
}

// BuildValidateJSON converts a column-validation result into its machine-readable form.
func BuildValidateJSON(result *reader.ColumnValidationResult, sourceFile string) *ValidateJSONOutput {
	requiredFound := make(map[string]string)
	var requiredMissing []string
	for _, col := range reader.RequiredColumns {
		if original, ok := result.Found[col]; ok {
			requiredFound[col] = original
		} else {
			requiredMissing = append(requiredMissing, col)
		}
	}

	optionalFound := make(map[string]string)
	for _, col := range reader.OptionalColumns {
		if original, ok := result.Found[col]; ok {
			optionalFound[col] = original
		}
	}

	status := "READY"
	if len(requiredMissing) > 0 {
		status = "INCOMPLETE"
	}

	extra := append([]string(nil), result.Extra...)
	sort.Strings(extra)

	return &ValidateJSONOutput{
		SourceFile:      sourceFile,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		ToolVersion:     version.Display(),
		Status:          status,
		RequiredFound:   requiredFound,
		RequiredMissing: requiredMissing,
		OptionalFound:   optionalFound,
		ExtraColumns:    extra,
		Warnings:        result.Warnings,
	}
}

// BuildPatternsJSON converts rule-evaluation results into machine-readable form.
func BuildPatternsJSON(results []rules.RuleResult, scheduleFile, rulesFile string) *PatternsJSONOutput {
	passed := 0
	out := make([]PatternJSONResult, 0, len(results))
	for _, r := range results {
		if r.Passing {
			passed++
		}
		ids := make([]string, 0, len(r.MatchingTasks))
		for _, t := range r.MatchingTasks {
			ids = append(ids, t.TaskID)
		}
		out = append(out, PatternJSONResult{
			RuleName:        r.Rule.Name,
			Match:           r.Rule.Match,
			MinCount:        r.Rule.MinCount,
			MaxCount:        r.Rule.MaxCount,
			MatchingCount:   r.Count,
			Passing:         r.Passing,
			Message:         r.Message,
			MatchingTaskIDs: ids,
		})
	}

	status := "COMPLIANT"
	if passed != len(results) {
		status = "NON-COMPLIANT"
	}

	return &PatternsJSONOutput{
		ScheduleFile: scheduleFile,
		RulesFile:    rulesFile,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		ToolVersion:  version.Display(),
		Status:       status,
		PassedCount:  passed,
		TotalCount:   len(results),
		Results:      out,
	}
}
