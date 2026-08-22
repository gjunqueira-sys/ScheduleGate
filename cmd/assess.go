package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/spf13/cobra"
)

// parseStatusDate attempts to parse a user-supplied status date string.
// Accepted formats: YYYY-MM-DD, MM/DD/YYYY, MM-DD-YYYY, MM/DD/YY, MM-DD-YY.
// dateLocale controls which ambiguous format (MM/DD vs DD/MM) is tried first.
func parseStatusDate(s string, dateLocale string) (time.Time, error) {
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
		firstSlash,
		secondSlash,
		firstDash,
		secondDash,
		firstShortSlash,
		secondShortSlash,
		firstShortDash,
		secondShortDash,
	}

	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"unrecognised date %q — use YYYY-MM-DD, MM/DD/YYYY, or MM/DD/YY", s,
	)
}

// assessCmd represents the assess command
var assessCmd = &cobra.Command{
	Use:   "assess [file]",
	Short: "Perform DCMA 14-point assessment on a schedule file",
	Long: `Reads a Microsoft Project schedule (Excel or CSV) and performs the DCMA 14-point assessment metrics.
	
Example:
  schedulegate assess my_schedule.xlsx --metrics 1,5,12`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		jsonMode := jsonEnabled(cmd)

		// --- License gating ---
		tier := tierOf()
		if tier == license.TierCommunity {
			if jsonMode || jsonOutputPath(cmd) != "" {
				fail(cmd, "%s JSON output requires Pro tier. Upgrade: https://schedulegate.dev/pricing", ui.FailBadge.String())
			}
			for _, flagName := range []string{"html", "csv", "exceptions-report"} {
				if v, _ := cmd.Flags().GetString(flagName); v != "" {
					fail(cmd, "%s HTML/CSV/Excel output requires Pro tier. Upgrade: https://schedulegate.dev/pricing", ui.FailBadge.String())
				}
			}
			allowed, used, err := license.CheckMonthlyUsage()
			if err != nil {
				fail(cmd, "%s usage tracking error: %v", ui.FailBadge.String(), err)
			}
			if !allowed {
				fail(cmd, "%s Community tier allows 1 assessment/month (already used %d this month). Upgrade: https://schedulegate.dev/pricing", ui.FailBadge.String(), used)
			}
		}

		var selectedMetrics []int
		metricsStr, _ := cmd.Flags().GetString("metrics")
		if metricsStr != "" {
			parts := strings.Split(metricsStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				id, err := strconv.Atoi(p)
				if err != nil {
					fail(cmd, "Invalid metric ID: %s. IDs must be integers 1-14.", p)
				}
				selectedMetrics = append(selectedMetrics, id)
			}
		}

		if !jsonMode {
			fmt.Println(ui.RenderLogo())
			fmt.Println(ui.RenderTitle("SCHEDULE ASSESSMENT"))
			fmt.Println(ui.RenderVersion())
			fmt.Printf("Reading schedule: %s\n", filePath)
		}
		dateLocale, _ := cmd.Flags().GetString("date-locale")
		dateLocale = strings.ToUpper(dateLocale)
		if dateLocale == "" {
			dateLocale = "US"
		}
		if dateLocale != "US" && dateLocale != "EU" {
			fail(cmd, "Error: --date-locale must be \"US\" or \"EU\", got %q", dateLocale)
		}
		r := reader.NewScheduleReader(filePath)
		r.DateLocale = dateLocale
		if pctFormat, _ := cmd.Flags().GetString("pct-format"); pctFormat != "" {
			r.PctFormat = pctFormat
		}
		schedule, err := r.Read()
		if err != nil {
			fail(cmd, "Error reading schedule: %v", err)
		}

		// --status-date overrides the default status date (today).
		statusDateStr, _ := cmd.Flags().GetString("status-date")
		dateSource := "today"
		if statusDateStr != "" {
			parsed, parseErr := parseStatusDate(statusDateStr, dateLocale)
			if parseErr != nil {
				fail(cmd, "Error: %v", parseErr)
			}
			schedule.DataDate = parsed
			dateSource = "flag"
		}

		if !jsonMode {
			for _, w := range schedule.Warnings {
				fmt.Printf("Warning: %s\n", w)
			}
			if len(schedule.Warnings) > 0 {
				fmt.Println()
			}
			fmt.Printf("Successfully loaded schedule: %s\n", schedule.Name)
			fmt.Printf("Status date: %s  (source: %s)\n", schedule.DataDate.Format("2006-01-02"), dateSource)
			fmt.Printf("running assessment...\n\n")
		}

		assessment := dcma.NewDCMAAssessment(selectedMetrics)
		assessment.Assess(schedule)

		// Styled Summary — exclude N/A metrics from the pass/fail tally
		passedCount := 0
		totalCount := 0
		for _, m := range assessment.Metrics {
			res := assessment.Results[m.Name()]
			if res.NotApplicable {
				continue
			}
			totalCount++
			if res.Passing {
				passedCount++
			}
		}

		score := 0.0
		if totalCount > 0 {
			score = float64(passedCount) / float64(totalCount) * 100
		}
		scoreStr := fmt.Sprintf("%.0f%%", score)

		if !jsonMode {
			// Render Score Card
			scoreLabel := fmt.Sprintf("%d/%d Metrics Passed", passedCount, totalCount)
			fmt.Println(ui.CardStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					ui.SectionHeaderStyle.Render("OVERALL SCORE"),
					lipgloss.NewStyle().Bold(true).Foreground(ui.ColorPrimary).Padding(1).Render(scoreStr),
					lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(scoreLabel),
				),
			))

			// Render Metrics
			fmt.Println(ui.SectionHeaderStyle.Render("METRIC RESULTS"))
			for _, m := range assessment.Metrics {
				res := assessment.Results[m.Name()]
				var valStr, threshStr string
				var badge string
				if res.NotApplicable {
					valStr = "N/A"
					threshStr = ""
					badge = lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render("[N/A]")
				} else {
					valStr = fmt.Sprintf("%.1f%%", res.Value*100)
					threshStr = fmt.Sprintf("(Threshold: %.1f%%)", res.Threshold*100)
					badge = ui.Status(res.Passing)
				}

				// Row: Badge | Name | Value | Threshold
				row := fmt.Sprintf("%s  %-25s  %s  %s",
					badge,
					m.Name(),
					lipgloss.NewStyle().Bold(true).Render(valStr),
					lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(threshStr),
				)
				if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
					if count, ok := res.Details["count"]; ok {
						total := res.Details["total"]
						row += fmt.Sprintf("  [%v / %v]", count, total)
					} else if completed, ok := res.Details["completed"]; ok {
						bc := res.Details["baseline_count"]
						row += fmt.Sprintf("  [completed=%v baseline=%v]", completed, bc)
					}
				}
				fmt.Println(row)
			}
			fmt.Println()

			// Logic debug trace — prints per-task successor resolution for all flagged tasks
			if debugLogic, _ := cmd.Flags().GetBool("debug-logic"); debugLogic {
				fmt.Println(ui.SectionHeaderStyle.Render("LOGIC METRIC DEBUG"))
				lines := dcma.DebugLogic(schedule)
				flaggedCount := 0
				for _, l := range lines {
					if !l.Flagged {
						continue
					}
					flaggedCount++
					fmt.Printf("  [%s] ID %-6s  %s\n", ui.FailBadge.String(), l.TaskID, l.Name)
					fmt.Printf("           Reason     : %s\n", l.FlagReason)
					if l.Predecessors != "" {
						fmt.Printf("           Predecessors: %s\n", l.Predecessors)
					}
					if len(l.SkippedBy) > 0 {
						fmt.Printf("           Skipped by summary tasks: %s\n", strings.Join(l.SkippedBy, ", "))
					}
					if len(l.InactiveReferrers) > 0 {
						fmt.Printf("           Inactive referrers (excluded from network): %s\n", strings.Join(l.InactiveReferrers, ", "))
					}
					fmt.Println()
				}
				if flaggedCount == 0 {
					fmt.Println("  No Logic violations to trace.")
				}
			}
		}

		// HTML Report
		htmlPath, _ := cmd.Flags().GetString("html")
		if htmlPath != "" {
			customer, _ := cmd.Flags().GetString("customer")
			project, _ := cmd.Flags().GetString("project")

			err := report.GenerateHTML(assessment, schedule.Name, customer, project, schedule.DataDate.Format("2006-01-02"), htmlPath)
			if err != nil {
				if jsonMode {
					fail(cmd, "%s Error generating HTML report: %v", ui.FailBadge.String(), err)
				} else {
					fmt.Printf("%s Error generating HTML report: %v\n", ui.FailBadge.String(), err)
				}
			} else if !jsonMode {
				fmt.Printf("%s HTML report generated: %s\n", ui.PassBadge.String(), htmlPath)
			}
			autoOpen(cmd, htmlPath)
		}

		// CSV Database export
		csvPath, _ := cmd.Flags().GetString("csv")
		if csvPath != "" {
			customer, _ := cmd.Flags().GetString("customer")
			project, _ := cmd.Flags().GetString("project")

			err := report.AppendToCSV(assessment, schedule.Name, customer, project, schedule.DataDate.Format("2006-01-02"), csvPath)
			if err != nil {
				fmt.Printf("%s Error updating CSV database: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s Appended results to CSV database: %s\n", ui.PassBadge.String(), csvPath)
			}
		}

		// Excel exceptions report
		excelPath, _ := cmd.Flags().GetString("exceptions-report")
		if excelPath != "" {
			customer, _ := cmd.Flags().GetString("customer")
			project, _ := cmd.Flags().GetString("project")

			err := report.GenerateExcelExceptions(assessment, schedule, schedule.Name, customer, project, schedule.DataDate.Format("2006-01-02"), excelPath)
			if err != nil {
				if jsonMode {
					fail(cmd, "%s Error generating exceptions report: %v", ui.FailBadge.String(), err)
				} else {
					fmt.Printf("%s Error generating exceptions report: %v\n", ui.FailBadge.String(), err)
				}
			} else if !jsonMode {
				fmt.Printf("%s Exceptions report: %s\n", ui.PassBadge.String(), excelPath)
			}
			autoOpen(cmd, excelPath)
		}

		// JSON output
		if jsonMode || jsonOutputPath(cmd) != "" {
			jsonData := report.BuildAssessJSON(assessment, schedule.Name, schedule.DataDate.Format("2006-01-02"), schedule.Warnings)
			if err := writeJSONOutput(cmd, jsonData); err != nil {
				fail(cmd, "Error writing JSON output: %v", err)
			}
		}

		// Record usage only after a successful run on the Community tier.
		if tier == license.TierCommunity {
			if err := license.IncrementMonthlyUsage(); err != nil {
				fmt.Printf("Warning: could not record usage: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(assessCmd)
	assessCmd.Flags().StringP("metrics", "m", "", "Comma-separated list of metric IDs to run (1-14)")
	assessCmd.Flags().String("html", "", "Path to generate HTML report (e.g. report.html)")
	assessCmd.Flags().String("csv", "", "Path to append results to a CSV database file")
	assessCmd.Flags().String("customer", "", "Customer name for the report")
	assessCmd.Flags().String("project", "", "Project number/ID for the report")
	assessCmd.Flags().BoolP("verbose", "v", false, "Show raw numerator/denominator counts for each metric")
	assessCmd.Flags().String("status-date", "", "Override the schedule status date (YYYY-MM-DD, MM/DD/YYYY, or MM/DD/YY).\nDefaults to today's date.")
	assessCmd.Flags().String("exceptions-report", "", "Path to generate an Excel workbook listing every task that caused a metric violation, with corrective guidance (e.g. exceptions.xlsx)")
	assessCmd.Flags().Bool("debug-logic", false, "Print per-task Logic metric trace showing why each flagged task has no successor (useful for verifying false positives)")
	assessCmd.Flags().String("pct-format", "", "Percent complete column scale: \"0-100\" (default, MS Project) or \"fraction\" (0.0-1.0 scale from Primavera, etc.)")
	assessCmd.Flags().String("date-locale", "US", "Date format locale for interpreting DD/MM vs MM/DD ambiguity: \"US\" (MM/DD/YYYY first) or \"EU\" (DD/MM/YYYY first)")
	assessCmd.Flags().Bool("json", false, "Output results as JSON to stdout (CI/CD friendly)")
	assessCmd.Flags().String("json-output", "", "Write JSON results to a file")
}
