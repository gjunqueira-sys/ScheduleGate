package report

import (
	"fmt"
	"html/template"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
)

// CompareReportData holds data for the Compare HTML template.
type CompareReportData struct {
	PreviousFile string
	CurrentFile  string
	StatusDate   string
	GeneratedAt  string
	ToolVersion  string
	Result       *compare.BenchmarkResult
	OverallScore int
	Logo         string
	ShowDetailed bool
}

const compareHtmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Schedule Comparison</title>
    <style>
		` + cssStyles + `
		/* Detailed Table Styles */
		.detailed-table {
			width: 100%;
			border-collapse: collapse;
			margin-top: 1rem;
			font-size: 0.85rem;
		}
		.detailed-table th {
			text-align: left;
			padding: 0.75rem;
			background-color: var(--card-bg);
			border-bottom: 1px solid var(--border-color);
			color: var(--text-muted);
			font-weight: 500;
		}
		.detailed-table td {
			padding: 0.6rem 0.75rem;
			border-bottom: 1px solid var(--border-color);
		}
		.detailed-table tr:hover {
			background-color: rgba(255,255,255,0.02);
		}
		.symbol-cell {
			font-size: 1.1rem;
			text-align: center;
		}
		.impact-stability { color: var(--color-danger); }
		.impact-reliability { color: var(--color-warning); }
		.impact-scope { color: var(--color-accent); }
		.impact-neutral { color: var(--text-muted); }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
				<pre class="ascii-logo">{{.Logo}}</pre>
                <h1>ScheduleGate</h1>
            </div>
            <div class="meta-info">
                <div class="meta-item"><span class="meta-label">Previous:</span> {{.PreviousFile}}</div>
                <div class="meta-item"><span class="meta-label">Current:</span> {{.CurrentFile}}</div>
                {{if .StatusDate}}<div class="meta-item"><span class="meta-label">Status Date:</span> {{.StatusDate}}</div>{{end}}
                <div class="meta-item"><span class="meta-label">Tool:</span> {{.ToolVersion}}</div>
                <div class="meta-item">{{.GeneratedAt}}</div>
            </div>
        </header>

        <!-- Score Card -->
        <div class="card score-container">
            <div class="score-value {{if ge .OverallScore 80}}text-success{{else}}{{if ge .OverallScore 60}}text-warning{{else}}text-danger{{end}}{{end}}">
                {{.OverallScore}}
            </div>
            <div class="score-subtitle">
                Stability Score (0-100)
            </div>
        </div>

        <!-- Pillars -->
        <div class="card">
            <div class="section-title">Scoring Pillars</div>
            
            <div class="progress-container">
                <div class="progress-label">
                    <span>Schedule Stability (Dates)</span>
                    <span>{{printf "%.1f" .Result.PillarAScore}}/40</span>
                </div>
                <div class="progress-bg">
                    <div class="progress-fill" style="width: {{printf "%.1f" (mul .Result.PillarAScore 2.5)}}%; background: #10b981;"></div>
                </div>
            </div>

            <div class="progress-container">
                <div class="progress-label">
                    <span>Duration Reliability (Bloat)</span>
                    <span>{{printf "%.1f" .Result.PillarBScore}}/30</span>
                </div>
                <div class="progress-bg">
                    <div class="progress-fill" style="width: {{printf "%.1f" (mul .Result.PillarBScore 3.33)}}%; background: #3b82f6;"></div>
                </div>
            </div>

            <div class="progress-container">
                <div class="progress-label">
                    <span>Scope Churn</span>
                    <span>{{printf "%.1f" .Result.PillarCScore}}/30</span>
                </div>
                <div class="progress-bg">
                    <div class="progress-fill" style="width: {{printf "%.1f" (mul .Result.PillarCScore 3.33)}}%; background: #f59e0b;"></div>
                </div>
            </div>
        </div>

        <!-- Metrics Grid -->
        <div class="grid-2">
            <div class="card">
                <div class="section-title">Change Metrics</div>
                <table>
                    <tr>
                        <td>New Tasks</td>
                        <td class="cell-num text-warning">{{.Result.NewTasks}}</td>
                    </tr>
                    <tr>
                        <td>Deleted Tasks</td>
                        <td class="cell-num text-danger">{{.Result.DeletedTasks}}</td>
                    </tr>
                    <tr>
                        <td>Modified Tasks</td>
                        <td class="cell-num">{{.Result.ModifiedTasks}}</td>
                    </tr>
                    <tr>
                        <td>Unchanged Tasks</td>
                        <td class="cell-num">{{.Result.UnchangedTasks}}</td>
                    </tr>
                </table>
            </div>

            <div class="card">
                <div class="section-title">Critical Issues</div>
                <table>
                    <tr>
                        <td>Ghost Tasks</td>
                        <td class="cell-num {{if gt .Result.GhostTasksCount 0}}text-danger{{else}}text-success{{end}}">{{.Result.GhostTasksCount}}</td>
                    </tr>
                    <tr>
                        <td>Duration Bloat (>10%)</td>
                        <td class="cell-num {{if gt .Result.DurationInflatedCount 0}}text-danger{{else}}text-success{{end}}">
                            {{.Result.DurationInflatedCount}} <span style="font-size: 0.8em; opacity: 0.7;">({{printf "%.1f" .Result.DurationInflatedPct}}%)</span>
                        </td>
                    </tr>
                </table>
            </div>
        </div>

        <!-- Friction Index -->
        {{if .Result.FrictionIndex}}
        <div class="card">
            <div class="section-title">Friction Index (Bottlenecks)</div>
            <p style="color: var(--text-muted); margin-bottom: 1rem;">
                Top WBS areas accumulating "Ghost Tasks" (Planned start in past, 0% complete).
            </p>
            <table>
                <thead>
                    <tr>
                        <th>WBS Area</th>
                        <th>Ghost Task Count</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Result.FrictionIndex}}
                    <tr>
                        <td>{{.WBS}}</td>
                        <td class="cell-num text-danger">{{.GhostTaskCount}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

        <!-- Detailed Task Analysis -->
        {{if .ShowDetailed}}
        <div class="card">
            <div class="section-title">Detailed Task Analysis</div>
            <table class="detailed-table">
                <thead>
                    <tr>
                        <th width="50">Sym</th>
                        <th>WBS</th>
                        <th>Task Name</th>
                        <th>Status</th>
                        <th>Impact</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Result.TaskDeltas}}
                    <tr>
                        <td class="symbol-cell">{{.Symbol}}</td>
                        <td>{{.WBS}}</td>
                        <td>{{.Name}}</td>
                        <td>
                            {{if eq .Status "New"}}<span class="text-success">New</span>{{end}}
                            {{if eq .Status "Deleted"}}<span class="text-danger">Deleted</span>{{end}}
                            {{if eq .Status "Modified"}}<span class="text-warning">Modified</span>{{end}}
                            {{if eq .Status "Unchanged"}}<span class="text-muted">Unchanged</span>{{end}}
                        </td>
                        <td class="impact-{{.ImpactType | toLower}}">
                            {{if .ImpactMsg}}
                                <strong>{{.ImpactType}}</strong>: {{.ImpactMsg}}
                            {{else}}
                                <span class="text-muted">-</span>
                            {{end}}
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

		<!-- Help / Definitions -->
        <div class="card">
            <div class="section-title">Definitions & Logic</div>
            <div class="grid-2">
				<div>
					<h4 style="margin-bottom: 0.5rem; color: #fff;">Stability Score (0-100)</h4>
					<p style="font-size: 0.9rem; color: var(--text-muted);">
						A composite score measuring the health of schedule changes.
						<ul style="padding-left: 1.2rem; margin-top: 0.5rem;">
							<li><strong>Stability (40%):</strong> Penalizes task finish dates slipping > 2 days.</li>
							<li><strong>Reliability (30%):</strong> Penalizes task durations increasing by > 10% (Bloat).</li>
							<li><strong>Churn (30%):</strong> Penalizes extensive adding/removing of tasks.</li>
						</ul>
					</p>
				</div>
				<div>
					<h4 style="margin-bottom: 0.5rem; color: #fff;">Ghost Tasks (Friction)</h4>
					<p style="font-size: 0.9rem; color: var(--text-muted);">
						Tasks that had a planned Start Date in the past (relative to status date) but have <strong>0% Progress</strong>.
						These represent immediate execution bottlenecks or "sliding" work.
					</p>
					<h4 style="margin-bottom: 0.5rem; color: #fff; margin-top: 1rem;">Duration Bloat</h4>
					<p style="font-size: 0.9rem; color: var(--text-muted);">
						Tasks where the duration has increased by more than 10% compared to the previous version. 
						Excessive bloat crashes the Reliability pillar score.
					</p>
				</div>
			</div>
        </div>

        <footer style="text-align: center; color: var(--border-color); font-size: 0.8rem; margin-top: 3rem;">
			Generated by {{.ToolVersion}}
		</footer>
    </div>
</body>
</html>
`

// GenerateCompareHTML creates an HTML comparison report.
func GenerateCompareHTML(result *compare.BenchmarkResult, prevFile, currFile, statusDate, outputPath string, showDetailed bool) error {
	data := CompareReportData{
		PreviousFile: prevFile,
		CurrentFile:  currFile,
		StatusDate:   statusDate,
		GeneratedAt:  time.Now().Format("2006-01-02 15:04:05"),
		ToolVersion:  version.Display(),
		Result:       result,
		OverallScore: int(math.Round(result.OverallScore)),
		Logo:         ui.LogoRaw,
		ShowDetailed: showDetailed,
	}

	tmpl, err := template.New("compare_report").Funcs(template.FuncMap{
		"mul":     func(a float64, b float64) float64 { return a * b },
		"toLower": strings.ToLower,
	}).Parse(compareHtmlTemplate)

	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}
