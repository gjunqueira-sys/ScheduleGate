package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type AssessService struct{}

func (s *AssessService) buildCommand(p AssessRequest) string {
	var parts []string
	parts = append(parts, "schedulegate", "assess", p.FilePath)
	if len(p.Metrics) > 0 {
		ids := make([]string, len(p.Metrics))
		for i, m := range p.Metrics {
			ids[i] = strconv.Itoa(m)
		}
		parts = append(parts, "--metrics", strings.Join(ids, ","))
	}
	if p.StatusDate != "" {
		parts = append(parts, "--status-date", p.StatusDate)
	}
	if p.HTMLOutput != "" {
		parts = append(parts, "--html", p.HTMLOutput)
	}
	if p.CSVOutput != "" {
		parts = append(parts, "--csv", p.CSVOutput)
	}
	if p.ExceptionsReport != "" {
		parts = append(parts, "--exceptions-report", p.ExceptionsReport)
	}
	if p.Customer != "" {
		parts = append(parts, "--customer", p.Customer)
	}
	if p.Project != "" {
		parts = append(parts, "--project", p.Project)
	}
	if p.Verbose {
		parts = append(parts, "--verbose")
	}
	if p.DebugLogic {
		parts = append(parts, "--debug-logic")
	}
	if p.PctFormat != "" {
		parts = append(parts, "--pct-format", p.PctFormat)
	}
	if p.DateLocale != "" && p.DateLocale != "US" {
		parts = append(parts, "--date-locale", p.DateLocale)
	}
	return strings.Join(parts, " ")
}

func (s *AssessService) emit(app *application.App, data string) {
	app.Event.Emit("term-output", data)
}

func (s *AssessService) Run(p AssessRequest) *AssessResponse {
	app := application.Get()
	resp := &AssessResponse{}

	cmd := s.buildCommand(p)
	s.emit(app, fmt.Sprintf("$ %s\n\n", cmd))

	app.Event.Emit("term-loading", true)
	defer app.Event.Emit("term-loading", false)

	dateLocale := strings.ToUpper(p.DateLocale)
	if dateLocale == "" {
		dateLocale = "US"
	}

	r := reader.NewScheduleReader(p.FilePath)
	r.DateLocale = dateLocale
	if p.PctFormat != "" {
		r.PctFormat = p.PctFormat
	}

	s.emit(app, "Reading schedule file...\n")
	schedule, err := r.Read()
	if err != nil {
		resp.Error = fmt.Sprintf("Error reading schedule: %v", err)
		s.emit(app, fmt.Sprintf("\x1b[31mError: %v\x1b[0m\n", err))
		return resp
	}
	s.emit(app, "done.\n")

	if p.StatusDate != "" {
		parsed, parseErr := parseStatusDate(p.StatusDate, dateLocale)
		if parseErr != nil {
			resp.Errors = append(resp.Errors, parseErr.Error())
			s.emit(app, fmt.Sprintf("\x1b[33mWarning: %v\x1b[0m\n", parseErr))
		} else {
			schedule.DataDate = parsed
		}
	}

	for _, w := range schedule.Warnings {
		s.emit(app, fmt.Sprintf("  \x1b[33mWarning: %s\x1b[0m\n", w))
	}

	s.emit(app, fmt.Sprintf("\nSchedule: %s\n", schedule.Name))
	s.emit(app, fmt.Sprintf("Data Date: %s\n", schedule.DataDate.Format("2006-01-02")))
	s.emit(app, fmt.Sprintf("Tasks: %d\n\n", len(schedule.Tasks)))

	assessment := dcma.NewDCMAAssessment(p.Metrics)
	assessment.Assess(schedule)

	passedCount := 0
	totalCount := 0
	var metricResults []MetricResult

	// Stop the loading spinner before writing results so its periodic
	// carriage-return redraw can't overwrite metric lines.
	app.Event.Emit("term-loading", false)

	var sb strings.Builder
	// Clear any residual spinner artifact left on the current line.
	sb.WriteString("\r\x1b[K")
	for _, m := range assessment.Metrics {
		res := assessment.Results[m.Name()]
		if res.NotApplicable {
			metricResults = append(metricResults, MetricResult{
				ID:            m.ID(),
				Name:          m.Name(),
				Description:   m.Description(),
				Value:         res.Value,
				Threshold:     res.Threshold,
				Passing:       res.Passing,
				NotApplicable: true,
			})
			fmt.Fprintf(&sb, "  Metric %2d — %-24s \x1b[90mN/A\x1b[0m\n", m.ID(), m.Name())
			continue
		}
		totalCount++
		if res.Passing {
			passedCount++
		}

		status := "PASS"
		color := "\x1b[32m"
		if !res.Passing {
			status = "FAIL"
			color = "\x1b[31m"
		}

		line := fmt.Sprintf("  %s%-5s\x1b[0m  Metric %2d — %-24s %5.1f%%  (threshold: %.0f%%)",
			color, status, m.ID(), m.Name(), res.Value*100, res.Threshold*100)

		if p.Verbose {
			if count, ok := res.Details["count"]; ok {
				line += fmt.Sprintf("  [%v/%v]", count, res.Details["total"])
			} else if completed, ok := res.Details["completed"]; ok {
				line += fmt.Sprintf("  [completed=%v baseline=%v]", completed, res.Details["baseline_count"])
			}
		}

		sb.WriteString(line + "\n")

		metricResults = append(metricResults, MetricResult{
			ID:          m.ID(),
			Name:        m.Name(),
			Description: m.Description(),
			Value:       res.Value * 100,
			Threshold:   res.Threshold * 100,
			Passing:     res.Passing,
		})
	}

	score := 0.0
	if totalCount > 0 {
		score = float64(passedCount) / float64(totalCount) * 100
	}

	fmt.Fprintf(&sb, "\n%sOVERALL SCORE: %.1f%%  (%d/%d passed)%s\n\n",
		"\x1b[1m", score, passedCount, totalCount, "\x1b[0m")

	s.emit(app, sb.String())

	if p.DebugLogic {
		s.emit(app, "\nLOGIC METRIC DEBUG:\n")
		lines := dcma.DebugLogic(schedule)
		flagCount := 0
		for _, l := range lines {
			if !l.Flagged {
				continue
			}
			flagCount++
			s.emit(app, fmt.Sprintf("  [FAIL] ID %-6s  %s\n", l.TaskID, l.Name))
			s.emit(app, fmt.Sprintf("         Reason: %s\n", l.FlagReason))
			if l.Predecessors != "" {
				s.emit(app, fmt.Sprintf("         Predecessors: %s\n", l.Predecessors))
			}
			s.emit(app, "\n")
		}
		if flagCount == 0 {
			s.emit(app, "  No Logic violations to trace.\n")
		}
	}

	var outputFiles []string

	if p.HTMLOutput != "" {
		err := report.GenerateHTML(assessment, schedule.Name, p.Customer, p.Project, schedule.DataDate.Format("2006-01-02"), p.HTMLOutput)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("HTML report: %v", err))
			s.emit(app, fmt.Sprintf("\x1b[31mError generating HTML: %v\x1b[0m\n", err))
		} else {
			outputFiles = append(outputFiles, p.HTMLOutput)
			s.emit(app, fmt.Sprintf("\x1b[32mHTML report: %s\x1b[0m\n", p.HTMLOutput))
		}
	}

	if p.CSVOutput != "" {
		err := report.AppendToCSV(assessment, schedule.Name, p.Customer, p.Project, schedule.DataDate.Format("2006-01-02"), p.CSVOutput)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("CSV: %v", err))
			s.emit(app, fmt.Sprintf("\x1b[31mError appending CSV: %v\x1b[0m\n", err))
		} else {
			outputFiles = append(outputFiles, p.CSVOutput)
			s.emit(app, fmt.Sprintf("\x1b[32mCSV appended: %s\x1b[0m\n", p.CSVOutput))
		}
	}

	if p.ExceptionsReport != "" {
		err := report.GenerateExcelExceptions(assessment, schedule, schedule.Name, p.Customer, p.Project, schedule.DataDate.Format("2006-01-02"), p.ExceptionsReport)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("Excel: %v", err))
			s.emit(app, fmt.Sprintf("\x1b[31mError generating Excel: %v\x1b[0m\n", err))
		} else {
			outputFiles = append(outputFiles, p.ExceptionsReport)
			s.emit(app, fmt.Sprintf("\x1b[32mExcel report: %s\x1b[0m\n", p.ExceptionsReport))
		}
	}

	resp.Success = true
	resp.OverallScore = score
	resp.Passed = passedCount
	resp.Total = totalCount
	resp.ScheduleName = schedule.Name
	resp.DataDate = schedule.DataDate.Format("2006-01-02")
	resp.TaskCount = len(schedule.Tasks)
	resp.Metrics = metricResults
	resp.OutputFiles = outputFiles

	return resp
}

func parseStatusDate(s, dateLocale string) (time.Time, error) {
	usSlash := "01/02/2006"
	euSlash := "02/01/2006"
	usDash := "01-02-2006"
	euDash := "02-01-2006"
	usShortSlash := "01/02/06"
	euShortSlash := "02/01/06"
	usShortDash := "01-02-06"
	euShortDash := "02-01-06"

	firstSlash, secondSlash := usSlash, euSlash
	firstDash, secondDash := usDash, euDash
	firstShortSlash, secondShortSlash := usShortSlash, euShortSlash
	firstShortDash, secondShortDash := usShortDash, euShortDash
	if dateLocale == "EU" {
		firstSlash, secondSlash = secondSlash, firstSlash
		firstDash, secondDash = secondDash, firstDash
		firstShortSlash, secondShortSlash = secondShortSlash, firstShortSlash
		firstShortDash, secondShortDash = secondShortDash, firstShortDash
	}

	formats := []string{
		"2006-01-02",
		firstSlash, secondSlash,
		firstDash, secondDash,
		firstShortSlash, secondShortSlash,
		firstShortDash, secondShortDash,
	}

	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q — use YYYY-MM-DD, MM/DD/YYYY, or MM/DD/YY", s)
}
