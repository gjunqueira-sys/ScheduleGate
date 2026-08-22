package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type CompareService struct{}

func (s *CompareService) emit(app *application.App, data string) {
	app.Event.Emit("term-output", data)
}

func (s *CompareService) buildCommand(p CompareRequest) string {
	var parts []string
	parts = append(parts, "schedulegate", "compare", p.PreviousFile, p.CurrentFile)
	if p.HTMLOutput != "" {
		parts = append(parts, "--html", p.HTMLOutput)
	}
	if p.CSVOutput != "" {
		parts = append(parts, "--csv", p.CSVOutput)
	}
	if p.Detailed {
		parts = append(parts, "--detailed")
	}
	if p.Customer != "" {
		parts = append(parts, "--customer", p.Customer)
	}
	if p.Project != "" {
		parts = append(parts, "--project", p.Project)
	}
	if p.PctFormat != "" {
		parts = append(parts, "--pct-format", p.PctFormat)
	}
	if p.DateLocale != "" && p.DateLocale != "US" {
		parts = append(parts, "--date-locale", p.DateLocale)
	}
	return strings.Join(parts, " ")
}

func (s *CompareService) Run(p CompareRequest) *CompareResponse {
	app := application.Get()
	resp := &CompareResponse{}

	cmd := s.buildCommand(p)
	s.emit(app, fmt.Sprintf("$ %s\n\n", cmd))

	app.Event.Emit("term-loading", true)
	defer app.Event.Emit("term-loading", false)

	dateLocale := strings.ToUpper(p.DateLocale)
	if dateLocale == "" {
		dateLocale = "US"
	}

	s.emit(app, "Reading previous schedule...\n")
	prevReader := reader.NewScheduleReader(p.PreviousFile)
	prevReader.DateLocale = dateLocale
	if p.PctFormat != "" {
		prevReader.PctFormat = p.PctFormat
	}
	prevSchedule, err := prevReader.Read()
	if err != nil {
		resp.Error = fmt.Sprintf("Error reading previous schedule: %v", err)
		s.emit(app, fmt.Sprintf("\x1b[31mError: %v\x1b[0m\n", err))
		return resp
	}

	s.emit(app, fmt.Sprintf("  Previous: %s (%d tasks)\n", prevSchedule.Name, len(prevSchedule.Tasks)))
	s.emit(app, "Reading current schedule...\n")

	currReader := reader.NewScheduleReader(p.CurrentFile)
	currReader.DateLocale = dateLocale
	if p.PctFormat != "" {
		currReader.PctFormat = p.PctFormat
	}
	currSchedule, err := currReader.Read()
	if err != nil {
		resp.Error = fmt.Sprintf("Error reading current schedule: %v", err)
		s.emit(app, fmt.Sprintf("\x1b[31mError: %v\x1b[0m\n", err))
		return resp
	}

	s.emit(app, fmt.Sprintf("  Current:  %s (%d tasks)\n\n", currSchedule.Name, len(currSchedule.Tasks)))
	s.emit(app, "Benchmarking...\n")

	result := compare.CompareSchedules(prevSchedule, currSchedule)

	// Stop the loading spinner before writing results so its periodic
	// carriage-return redraw can't overwrite output lines.
	app.Event.Emit("term-loading", false)

	var sb strings.Builder
	// Clear any residual spinner artifact left on the current line.
	sb.WriteString("\r\x1b[K")

	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			fmt.Fprintf(&sb, "  \x1b[33mWarning: %s\x1b[0m\n", w)
		}
	}

	fmt.Fprintf(&sb, "\n\x1b[1mCOMPARISON SCORE: %.1f%%\x1b[0m\n\n", result.OverallScore)
	fmt.Fprintf(&sb, "  Pillar A (Stability 40%%):      %s%.1f/40\x1b[0m\n", scoreColor(result.PillarAScore, 40), result.PillarAScore)
	fmt.Fprintf(&sb, "  Pillar B (Reliability 30%%):    %s%.1f/30\x1b[0m\n", scoreColor(result.PillarBScore, 30), result.PillarBScore)
	fmt.Fprintf(&sb, "  Pillar C (Scope Churn 30%%):    %s%.1f/30\x1b[0m\n\n", scoreColor(result.PillarCScore, 30), result.PillarCScore)

	fmt.Fprintf(&sb, "  Total Tasks:     %d\n", result.TotalTasks)
	fmt.Fprintf(&sb, "  New:             %d\n", result.NewTasks)
	fmt.Fprintf(&sb, "  Deleted:         %d\n", result.DeletedTasks)
	fmt.Fprintf(&sb, "  Modified:        %d\n", result.ModifiedTasks)
	fmt.Fprintf(&sb, "  Unchanged:       %d\n", result.UnchangedTasks)
	fmt.Fprintf(&sb, "  Ghost Tasks:     %d\n", result.GhostTasksCount)
	fmt.Fprintf(&sb, "  Duration Infl.:  %.1f%%\n\n", result.DurationInflatedPct)

	s.emit(app, sb.String())

	var deltas []TaskDelta
	for _, d := range result.TaskDeltas {
		if d.Status == compare.StatusUnchanged {
			continue
		}
		deltas = append(deltas, TaskDelta{
			TaskID:         d.TaskID,
			Name:           d.Name,
			WBS:            d.WBS,
			Status:         string(d.Status),
			FinishVariance: d.FinishVariance,
			DurationDelta:  d.DurationDelta,
			IsGhostTask:    d.IsGhostTask,
			IsMilestone:    d.IsMilestone,
		})
	}

	var outputFiles []string

	if p.HTMLOutput != "" {
		err := report.GenerateCompareHTML(result, prevSchedule.Name, currSchedule.Name, time.Now().Format("2006-01-02"), p.HTMLOutput, p.Detailed)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("HTML report: %v", err))
			s.emit(app, fmt.Sprintf("\x1b[31mError generating HTML: %v\x1b[0m\n", err))
		} else {
			outputFiles = append(outputFiles, p.HTMLOutput)
			s.emit(app, fmt.Sprintf("\x1b[32mHTML report: %s\x1b[0m\n", p.HTMLOutput))
		}
	}

	if p.CSVOutput != "" {
		err := report.AppendCompareToCSV(result, prevSchedule.Name, currSchedule.Name, p.Customer, p.Project, time.Now().Format("2006-01-02"), p.CSVOutput)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("CSV: %v", err))
			s.emit(app, fmt.Sprintf("\x1b[31mError appending CSV: %v\x1b[0m\n", err))
		} else {
			outputFiles = append(outputFiles, p.CSVOutput)
			s.emit(app, fmt.Sprintf("\x1b[32mCSV appended: %s\x1b[0m\n", p.CSVOutput))
		}
	}

	resp.Success = true
	resp.OverallScore = result.OverallScore
	resp.PillarAScore = result.PillarAScore
	resp.PillarBScore = result.PillarBScore
	resp.PillarCScore = result.PillarCScore
	resp.TotalTasks = result.TotalTasks
	resp.NewTasks = result.NewTasks
	resp.DeletedTasks = result.DeletedTasks
	resp.ModifiedTasks = result.ModifiedTasks
	resp.UnchangedTasks = result.UnchangedTasks
	resp.GhostTasksCount = result.GhostTasksCount
	resp.DurationInflatedPct = result.DurationInflatedPct
	resp.PrevScheduleName = prevSchedule.Name
	resp.CurrScheduleName = currSchedule.Name
	resp.TaskDeltas = deltas
	resp.OutputFiles = outputFiles

	return resp
}

func scoreColor(score, max float64) string {
	pct := score / max
	if pct >= 0.7 {
		return "\x1b[32m"
	}
	if pct >= 0.4 {
		return "\x1b[33m"
	}
	return "\x1b[31m"
}
