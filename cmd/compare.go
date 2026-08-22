package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/spf13/cobra"
)

// compareCmd represents the compare command
var compareCmd = &cobra.Command{
	Use:   "compare [previous_file] [current_file]",
	Short: "Benchmark two schedule versions to understand changes",
	Long: `Compares a previous schedule file against the current version.
Calculates Stability Score, Friction Index, and detailed change metrics.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		prevPath := args[0]
		currPath := args[1]
		jsonMode := jsonEnabled(cmd)

		// --- License gating ---
		if tierOf() == license.TierCommunity {
			fail(cmd, "%s The compare engine requires Pro tier. Upgrade: https://schedulegate.dev/pricing", ui.FailBadge.String())
		}

		if !jsonMode {
			fmt.Println(ui.RenderLogo())
			fmt.Println(ui.RenderTitle("SCHEDULE BENCHMARK"))
			fmt.Println(ui.RenderVersion())
		}

		pctFormat, _ := cmd.Flags().GetString("pct-format")
		dateLocale, _ := cmd.Flags().GetString("date-locale")
		dateLocale = strings.ToUpper(dateLocale)
		if dateLocale == "" {
			dateLocale = "US"
		}
		if dateLocale != "US" && dateLocale != "EU" {
			fail(cmd, "Error: --date-locale must be \"US\" or \"EU\", got %q", dateLocale)
		}

		// Determine status date — use flag if provided, otherwise use today
		statusDateStr, _ := cmd.Flags().GetString("status-date")
		var statusDate time.Time
		if statusDateStr != "" {
			parsed, parseErr := parseStatusDate(statusDateStr, dateLocale)
			if parseErr != nil {
				fail(cmd, "Error: %v", parseErr)
			}
			statusDate = parsed
		} else {
			statusDate = time.Now()
		}
		statusDateFormatted := statusDate.Format("2006-01-02")

		// Load Previous
		if !jsonMode {
			fmt.Printf("Loading Previous: %s\n", prevPath)
		}
		rPrev := reader.NewScheduleReader(prevPath)
		rPrev.DateLocale = dateLocale
		if pctFormat != "" {
			rPrev.PctFormat = pctFormat
		}
		prevSched, err := rPrev.Read()
		if err != nil {
			fail(cmd, "%s Error reading previous schedule: %v", ui.FailBadge.String(), err)
		}

		// Load Current
		if !jsonMode {
			fmt.Printf("Loading Current:  %s\n", currPath)
		}
		rCurr := reader.NewScheduleReader(currPath)
		rCurr.DateLocale = dateLocale
		if pctFormat != "" {
			rCurr.PctFormat = pctFormat
		}
		currSched, err := rCurr.Read()
		if err != nil {
			fail(cmd, "%s Error reading current schedule: %v", ui.FailBadge.String(), err)
		}

		if !jsonMode {
			fmt.Println()
			for _, w := range prevSched.Warnings {
				fmt.Printf("Warning (previous): %s\n", w)
			}
			for _, w := range currSched.Warnings {
				fmt.Printf("Warning (current): %s\n", w)
			}
			if len(prevSched.Warnings)+len(currSched.Warnings) > 0 {
				fmt.Println()
			}
		}

		// Run Comparison
		if !jsonMode {
			fmt.Println("Running comparison engine...")
		}
		result := compare.CompareSchedules(prevSched, currSched)

		if !jsonMode {
			for _, w := range result.Warnings {
				fmt.Println(lipgloss.NewStyle().Foreground(ui.ColorWarning).Render("⚠ " + w))
			}

			// 1. Stability Score Card
			scoreStr := fmt.Sprintf("%.0f", result.OverallScore)
			scoreColor := ui.ColorSuccess
			if result.OverallScore < 80 {
				scoreColor = ui.ColorWarning
			}
			if result.OverallScore < 60 {
				scoreColor = ui.ColorDanger
			}

			fmt.Println(ui.CardStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					ui.SectionHeaderStyle.Render("STABILITY SCORE"),
					lipgloss.NewStyle().Bold(true).Foreground(scoreColor).Padding(1).Render(scoreStr+"/100"),
				),
			))

			// 2. Pillars Breakdown
			fmt.Println(ui.SectionHeaderStyle.Render("SCORING PILLARS"))
			printPillar("Stability", result.PillarAScore, 40)
			printPillar("Reliability", result.PillarBScore, 30)
			printPillar("Scope Churn", result.PillarCScore, 30)
			fmt.Println()

			// 3. Change Summary
			fmt.Println(ui.SectionHeaderStyle.Render("CHANGE METRICS"))
			fmt.Printf("  New Tasks:      %d\n", result.NewTasks)
			fmt.Printf("  Deleted Tasks:  %d\n", result.DeletedTasks)
			fmt.Printf("  Modified Tasks: %d\n", result.ModifiedTasks)
			fmt.Printf("  Ghost Tasks:    %d\n", result.GhostTasksCount)
			if result.DurationInflatedCount > 0 {
				fmt.Printf("  Duration Bloat: %d (%.1f%% of tasks grew >10%%)\n", result.DurationInflatedCount, result.DurationInflatedPct)
			}
			fmt.Println()

			// 4. Friction Index (Top 5)
			if len(result.FrictionIndex) > 0 {
				fmt.Println(ui.SectionHeaderStyle.Render("FRICTION INDEX (Top Bottlenecks)"))
				limit := 5
				if len(result.FrictionIndex) < 5 {
					limit = len(result.FrictionIndex)
				}
				for i := 0; i < limit; i++ {
					item := result.FrictionIndex[i]
					fmt.Printf("  WBS %-10s : %d Ghost Tasks\n", item.WBS, item.GhostTaskCount)
				}
				fmt.Println()
			}
		}

		// HTML Report
		htmlPath, _ := cmd.Flags().GetString("html")
		if htmlPath != "" {
			showDetailed, _ := cmd.Flags().GetBool("detailed")
			err := report.GenerateCompareHTML(result, prevPath, currPath, statusDateFormatted, htmlPath, showDetailed)
			if err != nil {
				fmt.Printf("%s Error generating HTML report: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s HTML report generated: %s\n", ui.PassBadge.String(), htmlPath)
			}
			autoOpen(cmd, htmlPath)
		}

		// CSV Database export (one summary row per comparison)
		csvPath, _ := cmd.Flags().GetString("csv")
		if csvPath != "" {
			customer, _ := cmd.Flags().GetString("customer")
			project, _ := cmd.Flags().GetString("project")

			err := report.AppendCompareToCSV(result, prevSched.Name, currSched.Name, customer, project, statusDateFormatted, csvPath)
			if err != nil {
				fmt.Printf("%s Error updating CSV database: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s Appended comparison results to CSV database: %s\n", ui.PassBadge.String(), csvPath)
			}
		}

		// JSON output
		if jsonMode || jsonOutputPath(cmd) != "" {
			jsonData := report.BuildCompareJSON(result, prevPath, currPath, statusDateFormatted)
			if err := writeJSONOutput(cmd, jsonData); err != nil {
				fail(cmd, "Error writing JSON output: %v", err)
			}
		}
	},
}

func printPillar(name string, score, max float64) {
	barWidth := 20
	// Avoid division by zero if max is 0 (unlikely here)
	if max == 0 {
		max = 1
	}
	ratio := score / max
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	col := ui.ColorSuccess
	if score < max*0.7 {
		col = ui.ColorWarning
	}
	if score < max*0.5 {
		col = ui.ColorDanger
	}

	fmt.Printf("  %-15s %s %.1f/%d\n", name, lipgloss.NewStyle().Foreground(col).Render(bar), score, int(max))
}

func init() {
	rootCmd.AddCommand(compareCmd)
	compareCmd.Flags().String("html", "", "Path to generate HTML report (e.g. compare_report.html)")
	compareCmd.Flags().Bool("detailed", false, "Include detailed task-level analysis in the report")
	compareCmd.Flags().String("csv", "", "Path to append comparison results to a CSV database file")
	compareCmd.Flags().String("customer", "", "Customer name for the report")
	compareCmd.Flags().String("project", "", "Project number/ID for the report")
	compareCmd.Flags().String("pct-format", "", "Percent complete column scale: \"0-100\" (default, MS Project) or \"fraction\" (0.0-1.0 scale from Primavera, etc.)")
	compareCmd.Flags().String("date-locale", "US", "Date format locale for interpreting DD/MM vs MM/DD ambiguity: \"US\" (MM/DD/YYYY first) or \"EU\" (DD/MM/YYYY first)")
	compareCmd.Flags().String("status-date", "", "Override the schedule status date (YYYY-MM-DD, MM/DD/YYYY, or MM/DD/YY).\nDefaults to today's date.")
	compareCmd.Flags().Bool("json", false, "Output results as JSON to stdout (CI/CD friendly)")
	compareCmd.Flags().String("json-output", "", "Write JSON results to a file")
}
