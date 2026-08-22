package services

import (
	"fmt"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/rules"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type CheckPatternsService struct{}

func (s *CheckPatternsService) emit(app *application.App, data string) {
	app.Event.Emit("term-output", data)
}

func (s *CheckPatternsService) Run(p CheckPatternsRequest) *CheckPatternsResponse {
	app := application.Get()
	resp := &CheckPatternsResponse{}

	cmd := fmt.Sprintf("$ schedulegate check-patterns %s --rules %s", p.FilePath, p.RulesFile)
	if p.Detailed {
		cmd += " --detailed"
	}
	if p.PctFormat != "" {
		cmd += fmt.Sprintf(" --pct-format %s", p.PctFormat)
	}
	if p.DateLocale != "" && p.DateLocale != "US" {
		cmd += fmt.Sprintf(" --date-locale %s", p.DateLocale)
	}
	s.emit(app, cmd+"\n\n")

	app.Event.Emit("term-loading", true)
	defer app.Event.Emit("term-loading", false)

	s.emit(app, fmt.Sprintf("Loading rules: %s\n", p.RulesFile))
	ruleSet, err := rules.LoadRules(p.RulesFile)
	if err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("Error loading rules: %v", err))
		s.emit(app, fmt.Sprintf("\x1b[31mError loading rules: %v\x1b[0m\n", err))
		return resp
	}
	s.emit(app, fmt.Sprintf("Loaded %d rules\n", len(ruleSet.Rules)))

	dateLocale := strings.ToUpper(p.DateLocale)
	if dateLocale == "" {
		dateLocale = "US"
	}

	s.emit(app, fmt.Sprintf("Reading schedule: %s\n", p.FilePath))
	r := reader.NewScheduleReader(p.FilePath)
	r.DateLocale = dateLocale
	if p.PctFormat != "" {
		r.PctFormat = p.PctFormat
	}
	schedule, err := r.Read()
	if err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("Error reading schedule: %v", err))
		s.emit(app, fmt.Sprintf("\x1b[31mError reading schedule: %v\x1b[0m\n", err))
		return resp
	}
	s.emit(app, fmt.Sprintf("Loaded %d tasks\n\n", len(schedule.Tasks)))

	results := ruleSet.Evaluate(schedule)

	// Stop the loading spinner before writing results so its periodic
	// carriage-return redraw can't overwrite output lines.
	app.Event.Emit("term-loading", false)

	passedCount := 0
	var patternResults []PatternResult

	var sb strings.Builder
	// Clear any residual spinner artifact left on the current line.
	sb.WriteString("\r\x1b[K")

	for _, res := range results {
		var countInfo string
		if res.Rule.MaxCount > 0 {
			countInfo = fmt.Sprintf("Found: %d | Required: %d-%d", res.Count, res.Rule.MinCount, res.Rule.MaxCount)
		} else {
			countInfo = fmt.Sprintf("Found: %d | Required: %d+", res.Count, res.Rule.MinCount)
		}

		if res.Passing {
			passedCount++
			fmt.Fprintf(&sb, "  \x1b[32mPASS\x1b[0m  %-35s  %s\n", res.Rule.Name, countInfo)
		} else {
			fmt.Fprintf(&sb, "  \x1b[31mFAIL\x1b[0m  %-35s  %s\n", res.Rule.Name, countInfo)
		}

		patternResults = append(patternResults, PatternResult{
			Name:    res.Rule.Name,
			Count:   res.Count,
			Passing: res.Passing,
			Message: res.Message,
		})
	}

	if p.Detailed {
		for _, res := range results {
			if len(res.MatchingTasks) > 0 {
				fmt.Fprintf(&sb, "\nMATCHES: %s (%d tasks)\n", res.Rule.Name, len(res.MatchingTasks))
				for i, task := range res.MatchingTasks {
					if i >= 100 {
						fmt.Fprintf(&sb, "  ... and %d more\n", len(res.MatchingTasks)-100)
						break
					}
					fmt.Fprintf(&sb, "  • %s (%s)\n", task.Name, task.TaskID)
				}
			}
		}
	}

	allPassing := passedCount == len(results)
	status := "\x1b[31mNON-COMPLIANT\x1b[0m"
	if allPassing {
		status = "\x1b[32mCOMPLIANT\x1b[0m"
	}
	fmt.Fprintf(&sb, "\n\x1b[1mCOMPLIANCE STATUS: %s\x1b[0m\n", status)
	fmt.Fprintf(&sb, "%d/%d Rules Passed\n\n", passedCount, len(results))

	s.emit(app, sb.String())

	resp.Success = true
	resp.Results = patternResults
	resp.TaskCount = len(schedule.Tasks)

	return resp
}
