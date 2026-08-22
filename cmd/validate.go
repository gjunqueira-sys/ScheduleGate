package cmd

import (
	"encoding/csv"
	"fmt"
	"html"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate schedule columns against expected format",
	Long: `Reads a schedule file (Excel or CSV) and checks if it contains the expected columns.
Reports which columns are found, missing, and any extra columns present.

Example:
  schedulegate validate my_schedule.xlsx`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		jsonMode := jsonEnabled(cmd)

		if !jsonMode {
			fmt.Println(ui.RenderLogo())
			fmt.Println(ui.RenderTitle("COLUMN VALIDATION"))
			fmt.Println(ui.RenderVersion())
			fmt.Printf("Validating: %s\n\n", filePath)
		}

		result, err := reader.ValidateColumns(filePath)
		if err != nil {
			fail(cmd, "%s Error reading file: %v", ui.FailBadge.String(), err)
		}

		reportTimestamp := time.Now().Format("2006-01-02 15:04:05")

		if !jsonMode {
			for _, w := range result.Warnings {
				fmt.Printf("Warning: %s\n", w)
			}
			if len(result.Warnings) > 0 {
				fmt.Println()
			}
		}

		// Calculate stats
		requiredFound := 0
		requiredMissing := 0
		optionalFound := 0
		optionalMissing := 0

		for _, col := range reader.RequiredColumns {
			if _, ok := result.Found[col]; ok {
				requiredFound++
			} else {
				requiredMissing++
			}
		}

		for _, col := range reader.OptionalColumns {
			if _, ok := result.Found[col]; ok {
				optionalFound++
			} else {
				optionalMissing++
			}
		}

		allRequiredPresent := requiredMissing == 0

		// Render Score Card
		var statusText, statusColor string
		if allRequiredPresent {
			statusText = "READY"
			statusColor = "42" // Green
		} else {
			statusText = "INCOMPLETE"
			statusColor = "196" // Red
		}

		if !jsonMode {
			scoreLabel := fmt.Sprintf("Required: %d/%d | Optional: %d/%d",
				requiredFound, len(reader.RequiredColumns),
				optionalFound, len(reader.OptionalColumns))

			fmt.Println(ui.CardStyle.Render(
				lipgloss.JoinVertical(lipgloss.Center,
					ui.SectionHeaderStyle.Render("VALIDATION STATUS"),
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor)).Padding(1).Render(statusText),
					lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(scoreLabel),
				),
			))

			// Required Columns Section
			fmt.Println(ui.SectionHeaderStyle.Render("REQUIRED COLUMNS"))
			for _, col := range reader.RequiredColumns {
				var badge, detail string
				if original, ok := result.Found[col]; ok {
					badge = ui.PassBadge.String()
					detail = fmt.Sprintf("→ \"%s\"", original)
				} else {
					badge = ui.FailBadge.String()
					detail = "(not found)"
				}
				row := fmt.Sprintf("%s  %-20s  %s", badge, col, lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(detail))
				fmt.Println(row)
			}
			fmt.Println()

			// Optional Columns Section
			fmt.Println(ui.SectionHeaderStyle.Render("OPTIONAL COLUMNS"))
			for _, col := range reader.OptionalColumns {
				var badge, detail string
				if original, ok := result.Found[col]; ok {
					badge = ui.PassBadge.String()
					detail = fmt.Sprintf("→ \"%s\"", original)
				} else {
					badge = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorLight).Background(ui.ColorWarning).Padding(0, 1).Render("SKIP")
					detail = "(not found)"
				}
				row := fmt.Sprintf("%s  %-20s  %s", badge, col, lipgloss.NewStyle().Foreground(ui.ColorNeutral).Render(detail))
				fmt.Println(row)
			}
			fmt.Println()

			// Extra Columns Section
			if len(result.Extra) > 0 {
				fmt.Println(ui.SectionHeaderStyle.Render("EXTRA COLUMNS (unmapped)"))
				sort.Strings(result.Extra)
				for _, col := range result.Extra {
					fmt.Printf("  • %s\n", col)
				}
				fmt.Println()
			}
		}

		// HTML Report
		htmlPath, _ := cmd.Flags().GetString("html")
		if htmlPath != "" {
			err := generateValidationHTML(result, filePath, htmlPath, reportTimestamp)
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
			err := appendValidationCSV(result, filePath, csvPath, reportTimestamp)
			if err != nil {
				fmt.Printf("%s Error updating CSV: %v\n", ui.FailBadge.String(), err)
			} else if !jsonMode {
				fmt.Printf("%s Appended results to CSV: %s\n", ui.PassBadge.String(), csvPath)
			}
		}

		// JSON output
		if jsonMode || jsonOutputPath(cmd) != "" {
			jsonData := report.BuildValidateJSON(result, filePath)
			if err := writeJSONOutput(cmd, jsonData); err != nil {
				fail(cmd, "Error writing JSON output: %v", err)
			}
		}
	},
}

func generateValidationHTML(result *reader.ColumnValidationResult, sourceFile, outputPath, reportTimestamp string) error {
	requiredFound := 0
	for _, col := range reader.RequiredColumns {
		if _, ok := result.Found[col]; ok {
			requiredFound++
		}
	}
	optionalFound := 0
	for _, col := range reader.OptionalColumns {
		if _, ok := result.Found[col]; ok {
			optionalFound++
		}
	}

	allRequiredPresent := requiredFound == len(reader.RequiredColumns)

	var statusBadge, statusColor string
	if allRequiredPresent {
		statusBadge = "READY"
		statusColor = "#2ecc71"
	} else {
		statusBadge = "INCOMPLETE"
		statusColor = "#e74c3c"
	}

	var requiredRows, optionalRows strings.Builder
	for _, col := range reader.RequiredColumns {
		if original, ok := result.Found[col]; ok {
			requiredRows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td class="found">✓ Found</td><td>%s</td></tr>`, html.EscapeString(col), html.EscapeString(original)))
		} else {
			requiredRows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td class="missing">✗ Missing</td><td>-</td></tr>`, html.EscapeString(col)))
		}
	}
	for _, col := range reader.OptionalColumns {
		if original, ok := result.Found[col]; ok {
			optionalRows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td class="found">✓ Found</td><td>%s</td></tr>`, html.EscapeString(col), html.EscapeString(original)))
		} else {
			optionalRows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td class="skipped">– Skipped</td><td>-</td></tr>`, html.EscapeString(col)))
		}
	}

	var extraList strings.Builder
	if len(result.Extra) > 0 {
		for _, col := range result.Extra {
			extraList.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(col)))
		}
	} else {
		extraList.WriteString("<li>None</li>")
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Column Validation Report</title>
<style>
:root { --bg: #0d1117; --card: #161b22; --border: #30363d; --text: #c9d1d9; --accent: #58a6ff; --success: #2ecc71; --danger: #e74c3c; --warning: #f39c12; }
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: var(--bg); color: var(--text); padding: 2rem; }
.container { max-width: 900px; margin: 0 auto; }
h1 { color: var(--accent); margin-bottom: 0.5rem; }
.subtitle { color: #8b949e; margin-bottom: 2rem; }
.status-card { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 2rem; text-align: center; margin-bottom: 2rem; }
.status-badge { display: inline-block; padding: 0.5rem 1.5rem; border-radius: 6px; font-weight: bold; font-size: 1.5rem; color: white; background: %s; }
.stats { color: #8b949e; margin-top: 1rem; }
h2 { color: var(--accent); margin: 1.5rem 0 1rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
table { width: 100%%; border-collapse: collapse; background: var(--card); border-radius: 8px; overflow: hidden; margin-bottom: 1rem; }
th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid var(--border); }
th { background: #21262d; color: var(--accent); }
.found { color: var(--success); font-weight: bold; }
.missing { color: var(--danger); font-weight: bold; }
.skipped { color: var(--warning); }
ul { list-style: none; padding-left: 1rem; }
li { padding: 0.25rem 0; }
li::before { content: "•"; color: var(--accent); margin-right: 0.5rem; }
</style>
</head>
<body>
<div class="container">
<h1>Column Validation Report</h1>
<p class="subtitle">Source: %s | %s | %s</p>
<div class="status-card">
<div class="status-badge">%s</div>
<p class="stats">Required: %d/%d | Optional: %d/%d</p>
</div>
<h2>Required Columns</h2>
<table><thead><tr><th>Column</th><th>Status</th><th>Matched Header</th></tr></thead><tbody>%s</tbody></table>
<h2>Optional Columns</h2>
<table><thead><tr><th>Column</th><th>Status</th><th>Matched Header</th></tr></thead><tbody>%s</tbody></table>
<h2>Extra Columns (Unmapped)</h2>
<ul>%s</ul>
<footer style="text-align: center; color: #8b949e; font-size: 0.8rem; margin-top: 3rem;">Generated by %s</footer>
</div>
</body>
</html>`,
		statusColor, html.EscapeString(sourceFile), reportTimestamp, version.Display(), statusBadge,
		requiredFound, len(reader.RequiredColumns),
		optionalFound, len(reader.OptionalColumns),
		requiredRows.String(), optionalRows.String(), extraList.String(), version.Display())

	return os.WriteFile(outputPath, []byte(html), 0644)
}

func appendValidationCSV(result *reader.ColumnValidationResult, sourceFile, csvPath, reportTimestamp string) (returnErr error) {
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
		header := []string{"Source File", "Report Date", "Tool Version", "Status", "Required Found", "Required Total", "Optional Found", "Optional Total", "Missing Columns"}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	requiredFound := 0
	var missing []string
	for _, col := range reader.RequiredColumns {
		if _, ok := result.Found[col]; ok {
			requiredFound++
		} else {
			missing = append(missing, col)
		}
	}

	optionalFound := 0
	for _, col := range reader.OptionalColumns {
		if _, ok := result.Found[col]; ok {
			optionalFound++
		}
	}

	status := "READY"
	if len(missing) > 0 {
		status = "INCOMPLETE"
	}

	row := []string{
		sourceFile,
		reportTimestamp,
		version.Display(),
		status,
		fmt.Sprintf("%d", requiredFound),
		fmt.Sprintf("%d", len(reader.RequiredColumns)),
		fmt.Sprintf("%d", optionalFound),
		fmt.Sprintf("%d", len(reader.OptionalColumns)),
		strings.Join(missing, "; "),
	}

	return writer.Write(row)
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().String("html", "", "Path to generate HTML report (e.g. validation.html)")
	validateCmd.Flags().String("csv", "", "Path to append results to a CSV file")
	validateCmd.Flags().Bool("json", false, "Output results as JSON to stdout (CI/CD friendly)")
	validateCmd.Flags().String("json-output", "", "Write JSON results to a file")
}
