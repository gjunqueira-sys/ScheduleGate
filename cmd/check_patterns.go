package cmd

import (
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/gjunqueira-sys/ScheduleGate/internal/rules"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
	"github.com/spf13/cobra"
)

var checkPatternsCmd = &cobra.Command{
	Use:   "check-patterns [schedule-file]",
	Short: "Check schedule tasks against pattern rules",
	Long: `Validates that a schedule contains tasks matching specified patterns.
Rules are defined in a YAML file with glob patterns and count requirements.

Example:
  schedulegate check-patterns schedule.xlsx --rules rules.yaml`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		jsonMode := jsonEnabled(cmd)
		rulesPath, _ := cmd.Flags().GetString("rules")

		// --- License gating ---
		if tierOf() == license.TierCommunity {
			fail(cmd, "%s Custom YAML pattern rules require Pro tier. Upgrade: https://schedulegate.dev/pricing", ui.FailBadge.String())
		}

		if rulesPath == "" {
			fail(cmd, "Error: --rules flag is required")
		}

		if !jsonMode {
			fmt.Println(ui.RenderLogo())
			fmt.Println(ui.RenderTitle("PATTERN COMPLIANCE CHECK"))
			fmt.Println(ui.RenderVersion())
			fmt.Printf("Loading rules: %s\n", rulesPath)
		}

		// Load rules
		ruleSet, err := rules.LoadRules(rulesPath)
		if err != nil {
			fail(cmd, "%s Error loading rules: %v", ui.FailBadge.String(), err)
		}
		if !jsonMode {
			fmt.Printf("Loaded %d rules\n\n", len(ruleSet.Rules))
		}

		// Load schedule
		if !jsonMode {
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
			fail(cmd, "%s Error reading schedule: %v", ui.FailBadge.String(), err)
		}
		if !jsonMode {
			for _, w := range schedule.Warnings {
				fmt.Printf("Warning: %s\n", w)
			}
			if len(schedule.Warnings) > 0 {
				fmt.Println()
			}
			fmt.Printf("Loaded %d tasks\n\n", len(schedule.Tasks))
		}

		reportTimestamp := time.Now().Format("2006-01-02 15:04:05")

		// Evaluate rules
		results := ruleSet.Evaluate(schedule)

		// Calculate summary
		passedCount := 0
		for _, res := range results {
			if res.Passing {
				passedCount++
			}
		}

		allPassing := passedCount == len(results)

		if !jsonMode {
			// Render Score Card
			var statusText, statusColor string
			if allPassing {
				statusText = "COMPLIANT"
				statusColor = "42"
			} else {
				statusText = "NON-COMPLIANT"
				statusColor = "196"
			}

			scoreLabel := fmt.Sprintf("%d/%d Rules Passed", passedCount, len(results))
			fmt.Println(ui.CardStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					ui.SectionHeaderStyle.Render("COMPLIANCE STATUS"),
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor)).Padding(1).Render(statusText),
					lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(scoreLabel),
				),
			))

			// Render Results
			fmt.Println(ui.SectionHeaderStyle.Render("RULE RESULTS"))
			for _, res := range results {
				var countInfo string
				if res.Rule.MaxCount > 0 {
					countInfo = fmt.Sprintf("Found: %d | Required: %d-%d", res.Count, res.Rule.MinCount, res.Rule.MaxCount)
				} else {
					countInfo = fmt.Sprintf("Found: %d | Required: %d+", res.Count, res.Rule.MinCount)
				}

				badge := ui.Status(res.Passing)
				row := fmt.Sprintf("%s  %-35s  %s",
					badge,
					res.Rule.Name,
					lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(countInfo),
				)
				fmt.Println(row)
			}
			fmt.Println()

			// Show detailed flag
			detailed, _ := cmd.Flags().GetBool("detailed")
			if detailed {
				for _, res := range results {
					if len(res.MatchingTasks) > 0 {
						fmt.Println(ui.SectionHeaderStyle.Render(fmt.Sprintf("MATCHES: %s", res.Rule.Name)))
						for i, task := range res.MatchingTasks {
							if i >= 100 {
								fmt.Printf("  ... and %d more\n", len(res.MatchingTasks)-100)
								break
							}
							fmt.Printf("  • %s (%s)\n", task.Name, task.TaskID)
						}
						fmt.Println()
					}
				}
			}
		}

		// HTML Report
		htmlPath, _ := cmd.Flags().GetString("html")
		if htmlPath != "" {
			detailed, _ := cmd.Flags().GetBool("detailed")
			err := generatePatternsHTML(results, filePath, rulesPath, htmlPath, detailed, reportTimestamp)
			if err != nil {
				fmt.Printf("%s Error generating HTML report: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s HTML report generated: %s\n", ui.PassBadge.String(), htmlPath)
			}
			autoOpen(cmd, htmlPath)
		}

		// CSV export
		csvPath, _ := cmd.Flags().GetString("csv")
		if csvPath != "" {
			err := appendPatternsCSV(results, filePath, csvPath, reportTimestamp)
			if err != nil {
				fmt.Printf("%s Error updating CSV: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s Appended results to CSV: %s\n", ui.PassBadge.String(), csvPath)
			}
		}

		// JSON output
		if jsonMode || jsonOutputPath(cmd) != "" {
			jsonData := report.BuildPatternsJSON(results, filePath, rulesPath)
			if err := writeJSONOutput(cmd, jsonData); err != nil {
				fail(cmd, "Error writing JSON output: %v", err)
			}
		}
	},
}

func generatePatternsHTML(results []rules.RuleResult, scheduleFile, rulesFile, outputPath string, detailed bool, reportTimestamp string) error {
	passedCount := 0
	for _, res := range results {
		if res.Passing {
			passedCount++
		}
	}
	allPassing := passedCount == len(results)

	var statusBadge, statusColor string
	if allPassing {
		statusBadge = "COMPLIANT"
		statusColor = "#2ecc71"
	} else {
		statusBadge = "NON-COMPLIANT"
		statusColor = "#e74c3c"
	}

	var tableRows strings.Builder
	for _, res := range results {
		var status, statusClass string
		if res.Passing {
			status = "✓ PASS"
			statusClass = "pass"
		} else {
			status = "✗ FAIL"
			statusClass = "fail"
		}

		var countReq string
		if res.Rule.MaxCount > 0 {
			countReq = fmt.Sprintf("%d-%d", res.Rule.MinCount, res.Rule.MaxCount)
		} else {
			countReq = fmt.Sprintf("%d+", res.Rule.MinCount)
		}

		// Build pattern display
		var patterns []string
		for field, pat := range res.Rule.Match {
			patterns = append(patterns, fmt.Sprintf("%s: %s", field, pat))
		}
		patternStr := strings.Join(patterns, "<br>")

		tableRows.WriteString(fmt.Sprintf(
			`<tr><td>%s</td><td><code>%s</code></td><td>%d</td><td>%s</td><td class="%s">%s</td></tr>`,
			html.EscapeString(res.Rule.Name), html.EscapeString(patternStr), res.Count, countReq, statusClass, status,
		))
	}

	// Build detailed matching tasks section
	var detailedSection strings.Builder
	if detailed {
		for _, res := range results {
			if len(res.MatchingTasks) > 0 {
				detailedSection.WriteString(fmt.Sprintf(`<h2>Matches: %s (%d tasks)</h2><div class="task-list">`, html.EscapeString(res.Rule.Name), len(res.MatchingTasks)))
				maxShow := 100
				for i, task := range res.MatchingTasks {
					if i >= maxShow {
						detailedSection.WriteString(fmt.Sprintf(`<div class="task-item more">... and %d more tasks</div>`, len(res.MatchingTasks)-maxShow))
						break
					}
					detailedSection.WriteString(fmt.Sprintf(`<div class="task-item"><span class="task-id">%s</span> %s</div>`, html.EscapeString(task.TaskID), html.EscapeString(task.Name)))
				}
				detailedSection.WriteString(`</div>`)
			}
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Pattern Compliance Report</title>
<style>
:root { --bg: #0d1117; --card: #161b22; --border: #30363d; --text: #c9d1d9; --accent: #58a6ff; --success: #2ecc71; --danger: #e74c3c; }
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); padding: 2rem; }
.container { max-width: 1000px; margin: 0 auto; }
h1 { color: var(--accent); margin-bottom: 0.5rem; }
.subtitle { color: #8b949e; margin-bottom: 2rem; }
.status-card { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 2rem; text-align: center; margin-bottom: 2rem; }
.status-badge { display: inline-block; padding: 0.5rem 1.5rem; border-radius: 6px; font-weight: bold; font-size: 1.5rem; color: white; background: %s; }
.stats { color: #8b949e; margin-top: 1rem; }
h2 { color: var(--accent); margin: 1.5rem 0 1rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
table { width: 100%%; border-collapse: collapse; background: var(--card); border-radius: 8px; overflow: hidden; }
th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid var(--border); }
th { background: #21262d; color: var(--accent); }
code { background: #21262d; padding: 0.2rem 0.4rem; border-radius: 4px; font-size: 0.85rem; }
.pass { color: var(--success); font-weight: bold; }
.fail { color: var(--danger); font-weight: bold; }
.task-list { background: var(--card); border-radius: 8px; padding: 1rem; margin-bottom: 1.5rem; }
.task-item { padding: 0.5rem 0; border-bottom: 1px solid var(--border); }
.task-item:last-child { border-bottom: none; }
.task-id { color: var(--accent); font-family: monospace; margin-right: 0.5rem; }
.task-item.more { color: #8b949e; font-style: italic; }
</style>
</head>
<body>
<div class="container">
<h1>Pattern Compliance Report</h1>
<p class="subtitle">Schedule: %s | Rules: %s | %s | %s</p>
<div class="status-card">
<div class="status-badge">%s</div>
<p class="stats">%d/%d Rules Passed</p>
</div>
<h2>Rule Results</h2>
<table>
<thead><tr><th>Rule Name</th><th>Pattern</th><th>Found</th><th>Required</th><th>Status</th></tr></thead>
<tbody>%s</tbody>
</table>
%s
<footer style="text-align: center; color: #8b949e; font-size: 0.8rem; margin-top: 3rem;">Generated by %s</footer>
</div>
</body>
</html>`,
		statusColor, html.EscapeString(scheduleFile), html.EscapeString(rulesFile), reportTimestamp, version.Display(), statusBadge,
		passedCount, len(results), tableRows.String(), detailedSection.String(), version.Display())

	return os.WriteFile(outputPath, []byte(html), 0644)
}

func appendPatternsCSV(results []rules.RuleResult, scheduleFile, csvPath, reportTimestamp string) (returnErr error) {
	fileExists := true
	if info, err := os.Stat(csvPath); err != nil || info.Size() == 0 {
		fileExists = false
	}

	f, err := os.OpenFile(csvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer func() {
		writer.Flush()
		if err := writer.Error(); err != nil {
			returnErr = fmt.Errorf("csv flush error: %w", err)
		}
	}()

	if !fileExists {
		header := []string{"Schedule File", "Report Date", "Tool Version", "Rule Name", "Pattern", "Found Count", "Min Required", "Max Required", "Status"}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	for _, res := range results {
		var patterns []string
		for field, pat := range res.Rule.Match {
			patterns = append(patterns, fmt.Sprintf("%s:%s", field, pat))
		}

		status := "PASS"
		if !res.Passing {
			status = "FAIL"
		}

		maxStr := ""
		if res.Rule.MaxCount > 0 {
			maxStr = fmt.Sprintf("%d", res.Rule.MaxCount)
		}

		row := []string{
			scheduleFile,
			reportTimestamp,
			version.Display(),
			res.Rule.Name,
			strings.Join(patterns, "; "),
			fmt.Sprintf("%d", res.Count),
			fmt.Sprintf("%d", res.Rule.MinCount),
			maxStr,
			status,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(checkPatternsCmd)
	checkPatternsCmd.Flags().String("rules", "", "Path to YAML rules file (required)")
	checkPatternsCmd.Flags().String("html", "", "Path to generate HTML report")
	checkPatternsCmd.Flags().String("csv", "", "Path to append results to CSV")
	checkPatternsCmd.Flags().Bool("detailed", false, "Show matching task details")
	checkPatternsCmd.Flags().String("pct-format", "", "Percent complete column scale: \"0-100\" (default, MS Project) or \"fraction\" (0.0-1.0 scale from Primavera, etc.)")
	checkPatternsCmd.Flags().String("date-locale", "US", "Date format locale for interpreting DD/MM vs MM/DD ambiguity: \"US\" (MM/DD/YYYY first) or \"EU\" (DD/MM/YYYY first)")
	checkPatternsCmd.Flags().Bool("json", false, "Output results as JSON to stdout (CI/CD friendly)")
	checkPatternsCmd.Flags().String("json-output", "", "Write JSON results to a file")
}
