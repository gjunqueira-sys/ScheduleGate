package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
	"github.com/gjunqueira-sys/ScheduleGate/internal/ui"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
	"github.com/xuri/excelize/v2"
)

// excelSheetName maps each metric name to the sheet tab label used in the
// exceptions workbook. Metrics that produce no per-task exceptions (CPLI,
// Critical Path Test) are intentionally absent from this map.
var excelSheetName = map[string]string{
	"Logic":                      "Logic",
	"Leads":                      "Leads",
	"Lags":                       "Lags",
	"Relationship Types":         "Relationship Types",
	"Hard Constraints":           "Hard Constraints",
	"High Float":                 "High Float",
	"Negative Float":             "Negative Float",
	"High Duration":              "High Duration",
	"Invalid Dates":              "Invalid Dates",
	"Resources":                  "Resources",
	"Missed Tasks":               "Missed Tasks",
	"Baseline Execution Index":   "BEI Incomplete Tasks",
}

// palette holds the hex color codes that mirror internal/report/styles.go so
// the Excel workbook matches the HTML report's visual identity.
const (
	palHeaderDark  = "0F172A" // dark navy — sheet title rows
	palCardBg      = "1E293B" // dark slate — column header rows & alt rows
	palAccentBlue  = "3B82F6" // brand blue  — section headers
	palPassGreen   = "10B981" // success green
	palFailRed     = "EF4444" // danger red
	palMuted       = "94A3B8" // muted slate — N/A cells
	palWhite       = "F1F5F9" // near-white text on dark backgrounds
	palBorder      = "334155" // border/separator colour
	palRowAlt      = "1A2744" // subtle alternate row tint
)

// styleIDs bundles all pre-created excelize style IDs so they are allocated
// once and reused across all sheets.
type styleIDs struct {
	titleCell    int // dark navy fill, white bold large text, centred
	metaLabel    int // normal weight, muted text
	metaValue    int // normal weight, white text
	colHeader    int // card-bg fill, accent-blue bold text
	passCell     int // green fill, white bold text, centred
	failCell     int // red fill, white bold text, centred
	naCell       int // muted fill, white text, centred
	numCell      int // white text, centred, monospace-ish
	normalCell   int // white text, left-aligned
	altRowCell   int // subtle tint, white text, left-aligned
	overallPass  int // accent-blue fill, white bold centred
	overallFail  int // red fill, white bold centred
	detailHeader int // card-bg fill, accent-blue bold
	detailAlt    int // alternate row for detail sheets
	detailNormal int // normal row for detail sheets
	wrapCell     int // wrap-text for Condition column
	wrapAltCell  int // wrap-text + alt background for Condition column
	logoLines    []int // monospace dark-navy fills with a blue→purple gradient,
	// one style per ASCII-art logo line (matches CLI logo)
}

// buildStyles allocates all required excelize styles once and returns their IDs.
// Arguments:
//
//	f (*excelize.File) — the workbook to register styles against
//
// Returns:
//
//	(styleIDs, error)
func buildStyles(f *excelize.File) (styleIDs, error) {
	var s styleIDs
	var err error

	makeStyle := func(style *excelize.Style) int {
		if err != nil {
			return 0
		}
		id, e := f.NewStyle(style)
		if e != nil {
			err = e
		}
		return id
	}

	s.titleCell = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palHeaderDark}},
		Font: &excelize.Font{Bold: true, Size: 14, Color: palWhite},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.metaLabel = makeStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palCardBg}},
		Font:      &excelize.Font{Bold: true, Size: 10, Color: palMuted},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.metaValue = makeStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palCardBg}},
		Font:      &excelize.Font{Size: 10, Color: palWhite},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.colHeader = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palCardBg}},
		Font: &excelize.Font{Bold: true, Size: 10, Color: palAccentBlue},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: palAccentBlue, Style: 2},
		},
	})

	s.passCell = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D1FAE5"}},
		Font: &excelize.Font{Bold: true, Size: 10, Color: "065F46"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.failCell = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEE2E2"}},
		Font: &excelize.Font{Bold: true, Size: 10, Color: "991B1B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.naCell = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E2E8F0"}},
		Font: &excelize.Font{Size: 10, Color: "64748B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.numCell = makeStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.normalCell = makeStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.altRowCell = makeStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8FAFC"}},
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.overallPass = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"DCFCE7"}},
		Font: &excelize.Font{Bold: true, Size: 11, Color: "166534"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.overallFail = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEE2E2"}},
		Font: &excelize.Font{Bold: true, Size: 11, Color: "991B1B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	s.detailHeader = makeStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palCardBg}},
		Font: &excelize.Font{Bold: true, Size: 10, Color: palAccentBlue},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: palAccentBlue, Style: 2},
		},
	})

	s.detailAlt = makeStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8FAFC"}},
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: false},
	})

	s.detailNormal = makeStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	s.wrapCell = makeStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})

	s.wrapAltCell = makeStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8FAFC"}},
		Font:      &excelize.Font{Size: 10, Color: "1E293B"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})

	// ASCII-logo gradient: top→bottom mirrors the CLI logo's blue→purple
	// gradient defined in internal/ui/logo.go. Courier New keeps the ASCII
	// columns aligned on every platform Excel runs on.
	logoGradient := []string{
		"60A5FA", // Blue 400
		"3B82F6", // Blue 500
		"818CF8", // Indigo 400
		"A78BFA", // Purple 400
		"C084FC", // Purple 500
	}
	s.logoLines = make([]int, len(logoGradient))
	for i, c := range logoGradient {
		s.logoLines[i] = makeStyle(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{palHeaderDark}},
			Font:      &excelize.Font{Family: "Courier New", Bold: true, Size: 10, Color: c},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})
	}

	return s, err
}

// setCellStr is a nil-safe helper that sets a cell value and style, ignoring
// the error from SetCellStyle when the range is trivially valid.
func setCellStr(f *excelize.File, sheet, cell, value string, styleID int) {
	_ = f.SetCellValue(sheet, cell, value)
	_ = f.SetCellStyle(sheet, cell, cell, styleID)
}

func setCellInt(f *excelize.File, sheet, cell string, value int, styleID int) {
	_ = f.SetCellValue(sheet, cell, value)
	_ = f.SetCellStyle(sheet, cell, cell, styleID)
}

func setCellFloat(f *excelize.File, sheet, cell string, value float64, styleID int) {
	_ = f.SetCellValue(sheet, cell, value)
	_ = f.SetCellStyle(sheet, cell, cell, styleID)
}

// colName converts a 1-based column index to an Excel column letter (A, B, …, Z, AA …).
func colName(col int) string {
	name := ""
	for col > 0 {
		col--
		name = string(rune('A'+col%26)) + name
		col /= 26
	}
	return name
}

// cellAddr returns a cell address string given 1-based row and column indices.
func cellAddr(row, col int) string {
	return fmt.Sprintf("%s%d", colName(col), row)
}

// formatTaskDate renders a schedule date for an Excel cell. It returns an empty
// string for a nil pointer so missing dates leave the cell blank rather than
// printing a zero value.
//
// Signature: formatTaskDate(t *time.Time) string
//
// Arguments:
//
//	t (*time.Time) — the date to format; may be nil
//
// Returns:
//
//	string — the date formatted as "2006-01-02", or "" when t is nil
func formatTaskDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// logoLines returns the non-empty content rows of the CLI ASCII logo so they
// can be rendered as individual Excel rows. Splitting at this layer keeps the
// raw constant in internal/ui authoritative for both the terminal and the
// workbook.
//
// Signature: logoLines() []string
//
// Returns:
//
//	[]string — one entry per visible logo line, in top-to-bottom order
func logoLines() []string {
	raw := strings.Split(ui.LogoRaw, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// writeLogoBanner renders the CLI ASCII logo at the top of the given sheet,
// occupying one row per logo line merged from column A through endCol. Each
// line uses a monospace bold font on the workbook's dark-navy header fill,
// coloured with the blue→purple gradient that mirrors the terminal output.
//
// Signature:
//
//	writeLogoBanner(f *excelize.File, sheet string, s styleIDs, startRow int, endCol string) int
//
// Arguments:
//
//	f        (*excelize.File) — destination workbook
//	sheet    (string)         — target sheet name
//	s        (styleIDs)       — pre-built styles, must include logoLines
//	startRow (int)            — 1-based first row to write the logo into
//	endCol   (string)         — last column letter to merge each logo row to
//
// Returns:
//
//	int — the first row below the banner that is free for subsequent content
func writeLogoBanner(f *excelize.File, sheet string, s styleIDs, startRow int, endCol string) int {
	lines := logoLines()
	for i, line := range lines {
		row := startRow + i
		start := fmt.Sprintf("A%d", row)
		end := fmt.Sprintf("%s%d", endCol, row)
		_ = f.MergeCell(sheet, start, end)
		_ = f.SetRowHeight(sheet, row, 14)
		style := s.logoLines[i%len(s.logoLines)]
		setCellStr(f, sheet, start, line, style)
	}
	return startRow + len(lines)
}

// GenerateExcelExceptions writes a multi-sheet DCMA exceptions workbook to
// outputPath. The workbook mirrors the visual identity of the HTML assessment
// report (dark header, green/red status cells, accent-blue column headers) and
// uses a structured 14-Point Assessment export format.
//
// Sheet layout:
//
//	"Summary"        — metadata block + per-metric pass/fail table with
//	                   exception counts
//	"Critical Path — Work" / "Critical Milestones" — remaining incomplete tasks
//	                                               with Total Slack ≤ 0 (work vs
//	                                               milestones; excludes summaries
//	                                               and project rollup)
//
// Arguments:
//
//	assessment   (*dcma.DCMAAssessment) — completed assessment with Results
//	schedule     (*model.Schedule)      — parsed schedule whose tasks feed the
//	                                       Critical Path sheet (dates, slack, WBS)
//	scheduleName (string)               — name shown in the workbook header
//	customer     (string)               — customer name (may be empty)
//	project      (string)               — project number (may be empty)
//	outputPath   (string)               — destination .xlsx file path
//
// Returns:
//
//	error — non-nil if style creation, sheet population, or file saving fails
func GenerateExcelExceptions(
	assessment *dcma.DCMAAssessment,
	schedule *model.Schedule,
	scheduleName, customer, project, statusDate, outputPath string,
) error {
	f := excelize.NewFile()
	defer f.Close()

	styles, err := buildStyles(f)
	if err != nil {
		return fmt.Errorf("excel: failed to create styles: %w", err)
	}

	// Rename the default "Sheet1" to "Summary"
	if err := f.SetSheetName("Sheet1", "Summary"); err != nil {
		return fmt.Errorf("excel: failed to rename sheet: %w", err)
	}

	if err := writeSummarySheet(f, styles, assessment, scheduleName, customer, project, statusDate); err != nil {
		return fmt.Errorf("excel: summary sheet: %w", err)
	}

	if err := writeUniverseSheet(f, styles, assessment); err != nil {
		return fmt.Errorf("excel: universe sheet: %w", err)
	}

	if err := writeCriticalPathSheet(f, styles, schedule); err != nil {
		return fmt.Errorf("excel: critical path sheet: %w", err)
	}

	if err := writeDetailSheets(f, styles, assessment); err != nil {
		return fmt.Errorf("excel: detail sheets: %w", err)
	}

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("excel: save failed: %w", err)
	}
	return nil
}

// writeSummarySheet populates the "Summary" sheet with a metadata block and
// the per-metric results table.
func writeSummarySheet(
	f *excelize.File,
	s styleIDs,
	assessment *dcma.DCMAAssessment,
	scheduleName, customer, project, statusDate string,
) error {
	const sheet = "Summary"

	// ── Column widths ────────────────────────────────────────────────────────
	_ = f.SetColWidth(sheet, "A", "A", 32)
	_ = f.SetColWidth(sheet, "B", "B", 6)
	_ = f.SetColWidth(sheet, "C", "C", 10)
	_ = f.SetColWidth(sheet, "D", "D", 14)
	_ = f.SetColWidth(sheet, "E", "E", 14)
	_ = f.SetColWidth(sheet, "F", "F", 12)
	_ = f.SetColWidth(sheet, "G", "G", 12)
	_ = f.SetColWidth(sheet, "H", "H", 12)
	_ = f.SetColWidth(sheet, "I", "I", 14)

	// ── CLI ASCII logo banner ────────────────────────────────────────────────
	row := writeLogoBanner(f, sheet, s, 1, "I")

	// ── Workbook title ───────────────────────────────────────────────────────
	_ = f.MergeCell(sheet, cellAddr(row, 1), cellAddr(row, 9))
	_ = f.SetRowHeight(sheet, row, 30)
	setCellStr(f, sheet, cellAddr(row, 1), "DCMA 14-Point Assessment — "+scheduleName, s.titleCell)
	row++

	// ── Metadata block ───────────────────────────────────────────────────────
	metaRows := []struct{ label, value string }{
		{"Customer:", customer},
		{"Project:", project},
		{"Status Date:", statusDate},
		{"Report Date:", time.Now().Format("2006-01-02 15:04")},
		{"Tool Version:", version.Display()},
	}
	for _, mr := range metaRows {
		setCellStr(f, sheet, cellAddr(row, 1), mr.label, s.metaLabel)
		setCellStr(f, sheet, cellAddr(row, 2), mr.value, s.metaValue)
		_ = f.MergeCell(sheet, cellAddr(row, 2), cellAddr(row, 9))
		row++
	}

	// ── Blank separator ──────────────────────────────────────────────────────
	_ = f.SetRowHeight(sheet, row, 8)
	row++

	// ── Column headers ───────────────────────────────────────────────────────
	_ = f.SetRowHeight(sheet, row, 22)
	headers := []string{"Metric", "ID", "Status", "Value", "Threshold", "Numerator", "Denominator", "Exceptions", "Unit"}
	for col, h := range headers {
		setCellStr(f, sheet, cellAddr(row, col+1), h, s.colHeader)
	}
	row++

	// ── One row per metric ───────────────────────────────────────────────────
	passedCount := 0
	totalCount := 0
	for _, m := range assessment.Metrics {
		res := assessment.Results[m.Name()]
		_ = f.SetRowHeight(sheet, row, 18)

		setCellStr(f, sheet, cellAddr(row, 1), m.Name(), s.normalCell)
		setCellInt(f, sheet, cellAddr(row, 2), m.ID(), s.numCell)

		unitStr, numStr, denStr := metricCountStrings(res)
		if res.NotApplicable {
			setCellStr(f, sheet, cellAddr(row, 3), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 4), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 5), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 6), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 7), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 8), "N/A", s.naCell)
			setCellStr(f, sheet, cellAddr(row, 9), unitStr, s.naCell)
		} else {
			totalCount++
			statusStyle := s.failCell
			statusText := "FAIL"
			if res.Passing {
				passedCount++
				statusStyle = s.passCell
				statusText = "PASS"
			}
			setCellStr(f, sheet, cellAddr(row, 3), statusText, statusStyle)
			setCellStr(f, sheet, cellAddr(row, 4), fmt.Sprintf("%.2f%%", res.Value*100), s.numCell)
			setCellStr(f, sheet, cellAddr(row, 5), formatThreshold(m.Threshold(), res), s.numCell)
			setCellStr(f, sheet, cellAddr(row, 6), numStr, s.numCell)
			setCellStr(f, sheet, cellAddr(row, 7), denStr, s.numCell)
			setCellInt(f, sheet, cellAddr(row, 8), len(res.Exceptions), s.numCell)
			setCellStr(f, sheet, cellAddr(row, 9), unitStr, s.numCell)
		}
		row++
	}

	// ── Overall score row ────────────────────────────────────────────────────
	_ = f.SetRowHeight(sheet, row, 22)
	overallScore := 0.0
	if totalCount > 0 {
		overallScore = float64(passedCount) / float64(totalCount) * 100
	}
	overallStyle := s.overallFail
	if overallScore >= 80 {
		overallStyle = s.overallPass
	}
	_ = f.MergeCell(sheet, cellAddr(row, 1), cellAddr(row, 2))
	setCellStr(f, sheet, cellAddr(row, 1), "Overall Score", s.colHeader)
	setCellStr(f, sheet, cellAddr(row, 3), fmt.Sprintf("%d/%d Passed", passedCount, totalCount), overallStyle)
	setCellStr(f, sheet, cellAddr(row, 4), fmt.Sprintf("%.1f%%", overallScore), overallStyle)
	_ = f.MergeCell(sheet, cellAddr(row, 4), cellAddr(row, 9))

	return nil
}

// formatThreshold returns a human-readable threshold string such as "≤ 5%" or
// "≥ 95%" consistent with the HTML report.
func formatThreshold(threshold float64, res dcma.MetricResult) string {
	if threshold == 0.0 {
		return "= 0%"
	}
	if res.Passing && res.Value >= res.Threshold {
		// metric passes when value is at-or-above threshold (e.g. Resources, BEI)
		return fmt.Sprintf("≥ %.0f%%", threshold*100)
	}
	if threshold >= 0.90 {
		return fmt.Sprintf("≥ %.0f%%", threshold*100)
	}
	return fmt.Sprintf("≤ %.0f%%", threshold*100)
}

// writeCriticalPathSheet adds two PM-focused critical-path views to the workbook:
//
//   - "Critical Path — Work"     — incomplete work tasks (matches Metric 12 scope)
//   - "Critical Milestones"      — incomplete zero-duration gates/checkpoints
//
// Both sheets apply the same membership filters: Total Slack ≤ 0, exclude summary
// rollups and the MS Project project-summary row, and exclude 100%-complete tasks
// so the list reflects remaining driving work rather than actualized history.
//
// Signature: writeCriticalPathSheet(f *excelize.File, s styleIDs, schedule *model.Schedule) error
func writeCriticalPathSheet(f *excelize.File, s styleIDs, schedule *model.Schedule) error {
	hasOutline := scheduleHasOutlineLevels(schedule.Tasks)
	work := collectCriticalPathTasks(schedule.Tasks, hasOutline, func(t *model.Task) bool {
		return !t.IsMilestone
	})
	milestones := collectCriticalPathTasks(schedule.Tasks, hasOutline, func(t *model.Task) bool {
		return t.IsMilestone
	})

	if err := writeCriticalPathTableSheet(f, s, criticalPathSheetSpec{
		SheetName:   "Critical Path — Work",
		Title:       "Critical Path — Remaining Work (Total Slack ≤ 0, incomplete)",
		ScopeNote:   "Incomplete work tasks with Total Slack ≤ 0. Excludes summaries, project rollup, and 100%-complete tasks.",
		EmptyMessage: "No remaining critical work tasks (no incomplete non-summary work task has Total Slack ≤ 0).",
		Tasks:       work,
	}); err != nil {
		return err
	}

	return writeCriticalPathTableSheet(f, s, criticalPathSheetSpec{
		SheetName:   "Critical Milestones",
		Title:       "Critical Milestones — Remaining Gates (Total Slack ≤ 0, incomplete)",
		ScopeNote:   "Incomplete milestones with Total Slack ≤ 0. Excludes summaries, project rollup, and 100%-complete gates.",
		EmptyMessage: "No remaining critical milestones (no incomplete non-summary milestone has Total Slack ≤ 0).",
		Tasks:       milestones,
	})
}

// criticalPathSheetSpec holds the per-sheet metadata and task rows for one
// critical-path table.
type criticalPathSheetSpec struct {
	SheetName    string
	Title        string
	ScopeNote    string
	EmptyMessage string
	Tasks        []*model.Task
}

// scheduleHasOutlineLevels reports whether the parsed schedule carries a
// meaningful Outline Level column (at least one task above level 0). When false,
// outline-level-based project-summary detection is skipped so tasks are not
// wrongly excluded because the column was absent and defaulted to zero.
//
// Signature: scheduleHasOutlineLevels(tasks []*model.Task) bool
func scheduleHasOutlineLevels(tasks []*model.Task) bool {
	for _, t := range tasks {
		if t != nil && t.OutlineLevel > 0 {
			return true
		}
	}
	return false
}

// isProjectSummaryTask identifies the MS Project project-summary rollup row
// that sometimes survives summary detection when Rollup=No and no Summary
// column is present (e.g. Task ID 0 spanning the whole project).
//
// Signature: isProjectSummaryTask(t *model.Task, hasOutlineLevels bool) bool
func isProjectSummaryTask(t *model.Task, hasOutlineLevels bool) bool {
	if t == nil {
		return true
	}
	if t.TaskID == "0" {
		return true
	}
	if hasOutlineLevels && t.OutlineLevel == 0 {
		return true
	}
	return false
}

// isRemainingCriticalTask applies the shared critical-path membership rule used
// by both work and milestone sheets before the type-specific predicate runs.
//
// Signature: isRemainingCriticalTask(t *model.Task, hasOutlineLevels bool) bool
func isRemainingCriticalTask(t *model.Task, hasOutlineLevels bool) bool {
	if t == nil || t.IsSummary || isProjectSummaryTask(t, hasOutlineLevels) {
		return false
	}
	if t.TotalSlack > 0 || t.PercentComplete >= 100 {
		return false
	}
	return true
}

// collectCriticalPathTasks returns incomplete, non-summary tasks with Total
// Slack ≤ 0 that match kind (work vs milestone), sorted by Start ascending.
//
// Signature:
//
//	collectCriticalPathTasks(
//	    tasks []*model.Task,
//	    hasOutlineLevels bool,
//	    kind func(*model.Task) bool,
//	) []*model.Task
func collectCriticalPathTasks(
	tasks []*model.Task,
	hasOutlineLevels bool,
	kind func(*model.Task) bool,
) []*model.Task {
	out := make([]*model.Task, 0)
	for _, t := range tasks {
		if !isRemainingCriticalTask(t, hasOutlineLevels) || !kind(t) {
			continue
		}
		out = append(out, t)
	}
	sortTasksByStart(out)
	return out
}

// sortTasksByStart orders tasks by Start ascending; tasks without Start sort last.
//
// Signature: sortTasksByStart(tasks []*model.Task)
func sortTasksByStart(tasks []*model.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i].Start, tasks[j].Start
		switch {
		case a == nil && b == nil:
			return false
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return a.Before(*b)
		}
	})
}

// writeCriticalPathTableSheet renders one critical-path table (title, scope,
// headers, and data rows) into a new worksheet.
//
// Signature:
//
//	writeCriticalPathTableSheet(
//	    f *excelize.File,
//	    s styleIDs,
//	    spec criticalPathSheetSpec,
//	) error
func writeCriticalPathTableSheet(f *excelize.File, s styleIDs, spec criticalPathSheetSpec) error {
	if _, err := f.NewSheet(spec.SheetName); err != nil {
		return fmt.Errorf("failed to create sheet %q: %w", spec.SheetName, err)
	}
	sheet := spec.SheetName

	_ = f.SetColWidth(sheet, "A", "A", 12)
	_ = f.SetColWidth(sheet, "B", "B", 42)
	_ = f.SetColWidth(sheet, "C", "C", 18)
	_ = f.SetColWidth(sheet, "D", "D", 10)
	_ = f.SetColWidth(sheet, "E", "E", 14)
	_ = f.SetColWidth(sheet, "F", "F", 14)
	_ = f.SetColWidth(sheet, "G", "G", 12)
	_ = f.SetColWidth(sheet, "H", "H", 16)
	_ = f.SetColWidth(sheet, "I", "I", 14)

	_ = f.MergeCell(sheet, "A1", "I1")
	_ = f.SetRowHeight(sheet, 1, 24)
	setCellStr(f, sheet, "A1", spec.Title, s.titleCell)

	_ = f.MergeCell(sheet, "A2", "I2")
	_ = f.SetRowHeight(sheet, 2, 18)
	setCellStr(f, sheet, "A2", criticalPathSummary(spec.Tasks, spec.ScopeNote), s.metaLabel)

	_ = f.SetRowHeight(sheet, 3, 20)
	headers := []string{
		"Task ID", "Task Name", "WBS", "Duration",
		"Start", "Finish", "Total Slack", "Baseline Finish", "Finish Variance (d)",
	}
	for i, h := range headers {
		setCellStr(f, sheet, cellAddr(3, i+1), h, s.detailHeader)
	}

	if len(spec.Tasks) == 0 {
		_ = f.MergeCell(sheet, "A4", "I4")
		_ = f.SetRowHeight(sheet, 4, 20)
		setCellStr(f, sheet, "A4", spec.EmptyMessage, s.detailNormal)
		return nil
	}

	for i, t := range spec.Tasks {
		row := i + 4
		alt := i%2 == 1
		textStyle := s.detailNormal
		numStyle := s.numCell
		if alt {
			textStyle = s.detailAlt
			numStyle = s.altRowCell
		}
		_ = f.SetRowHeight(sheet, row, 18)

		setCellStr(f, sheet, cellAddr(row, 1), t.TaskID, textStyle)
		setCellStr(f, sheet, cellAddr(row, 2), t.Name, textStyle)
		setCellStr(f, sheet, cellAddr(row, 3), t.WBS, textStyle)
		setCellFloat(f, sheet, cellAddr(row, 4), t.Duration, numStyle)
		setCellStr(f, sheet, cellAddr(row, 5), formatTaskDate(t.Start), numStyle)
		setCellStr(f, sheet, cellAddr(row, 6), formatTaskDate(t.Finish), numStyle)
		setCellFloat(f, sheet, cellAddr(row, 7), t.TotalSlack, numStyle)
		setCellStr(f, sheet, cellAddr(row, 8), formatTaskDate(t.BaselineFinish), numStyle)
		if t.Finish != nil && t.BaselineFinish != nil {
			varianceDays := t.Finish.Sub(*t.BaselineFinish).Hours() / 24
			setCellFloat(f, sheet, cellAddr(row, 9), varianceDays, numStyle)
		} else {
			setCellStr(f, sheet, cellAddr(row, 9), "", numStyle)
		}
	}

	return nil
}

// criticalPathSummary builds the scope banner above a critical-path table:
// filter description, task count, and earliest/latest dates when available.
//
// Signature: criticalPathSummary(tasks []*model.Task, scopeNote string) string
func criticalPathSummary(tasks []*model.Task, scopeNote string) string {
	if len(tasks) == 0 {
		return fmt.Sprintf("%s  |  0 tasks", scopeNote)
	}
	var earliest, latest *time.Time
	for _, t := range tasks {
		if t.Start != nil && (earliest == nil || t.Start.Before(*earliest)) {
			earliest = t.Start
		}
		if t.Finish != nil && (latest == nil || t.Finish.After(*latest)) {
			latest = t.Finish
		}
	}
	summary := fmt.Sprintf("%s  |  %d tasks", scopeNote, len(tasks))
	if earliest != nil && latest != nil {
		summary += fmt.Sprintf("  |  Earliest start %s  →  Latest finish %s",
			formatTaskDate(earliest), formatTaskDate(latest))
	}
	return summary
}

// writeDetailSheets creates one sheet per metric that has at least one
// exception entry. Sheet names use the metric's canonical name (e.g. "Logic",
// "Leads") keeping terminology consistent with the
// CLI output and HTML report.
func writeDetailSheets(f *excelize.File, s styleIDs, assessment *dcma.DCMAAssessment) error {
	for _, m := range assessment.Metrics {
		res := assessment.Results[m.Name()]
		if len(res.Exceptions) == 0 {
			continue
		}
		sheetName, ok := excelSheetName[m.Name()]
		if !ok {
			continue
		}

		if _, err := f.NewSheet(sheetName); err != nil {
			return fmt.Errorf("failed to create sheet %q: %w", sheetName, err)
		}

		if err := populateDetailSheet(f, s, sheetName, m.Name(), res); err != nil {
			return err
		}
	}
	return nil
}

// populateDetailSheet writes the header banner, the per-metric scope banner,
// and the exception rows for a single metric detail sheet.
//
// Layout (rows):
//
//	1: Sheet title banner "<metric> — Exceptions"
//	2: Scope banner — universe, violations, rate, and exclusions for this metric
//	3: Column headers (Task ID | Task Name | Condition)
//	4+: One row per exception
//
// Arguments:
//
//	f          (*excelize.File)    — workbook
//	s          (styleIDs)          — pre-built style IDs
//	sheetName  (string)            — Excel sheet tab name
//	metricName (string)            — metric display name for the title banner
//	res        (dcma.MetricResult) — full result; provides exceptions plus the
//	                                  Population funnel rendered in row 2
func populateDetailSheet(
	f *excelize.File,
	s styleIDs,
	sheetName, metricName string,
	res dcma.MetricResult,
) error {
	_ = f.SetColWidth(sheetName, "A", "A", 12)
	_ = f.SetColWidth(sheetName, "B", "B", 42)
	_ = f.SetColWidth(sheetName, "C", "C", 72)

	_ = f.MergeCell(sheetName, "A1", "C1")
	_ = f.SetRowHeight(sheetName, 1, 24)
	setCellStr(f, sheetName, "A1", metricName+" — Exceptions", s.titleCell)

	// ── Row 2: scope / universe banner ───────────────────────────────────────
	scopeText := detailScopeBanner(res)
	_ = f.MergeCell(sheetName, "A2", "C2")
	lineCount := (len(scopeText) / 100) + 1
	bannerH := float64(16 * lineCount)
	if bannerH < 22 {
		bannerH = 22
	}
	_ = f.SetRowHeight(sheetName, 2, bannerH)
	setCellStr(f, sheetName, "A2", scopeText, s.wrapAltCell)

	// ── Row 3: column headers ────────────────────────────────────────────────
	_ = f.SetRowHeight(sheetName, 3, 20)
	setCellStr(f, sheetName, "A3", "Task ID", s.detailHeader)
	setCellStr(f, sheetName, "B3", "Task Name", s.detailHeader)
	setCellStr(f, sheetName, "C3", "Condition / Corrective Action", s.detailHeader)

	// ── Data rows ────────────────────────────────────────────────────────────
	for i, ex := range res.Exceptions {
		row := i + 4
		alt := i%2 == 1
		baseStyle := s.detailNormal
		condStyle := s.wrapCell
		if alt {
			baseStyle = s.detailAlt
			condStyle = s.wrapAltCell
		}
		lineCount := (len(ex.Condition) / 72) + 1
		rowH := float64(15 * lineCount)
		if rowH < 18 {
			rowH = 18
		}
		_ = f.SetRowHeight(sheetName, row, rowH)

		setCellStr(f, sheetName, fmt.Sprintf("A%d", row), ex.TaskID, baseStyle)
		setCellStr(f, sheetName, fmt.Sprintf("B%d", row), ex.Name, baseStyle)
		setCellStr(f, sheetName, fmt.Sprintf("C%d", row), ex.Condition, condStyle)
	}

	return nil
}

// detailScopeBanner formats the per-metric scope banner shown at row 2 of
// each detail sheet. Format:
//
//	"Universe: <denominator> <unit>  |  Numerator: <numerator>  |  Exceptions:
//	 <n>  |  Rate: <value>%
//	 Scope: <scope>
//	 Excluded: <reason>=<n>; <reason>=<n>; …"
//
// The banner uses "Numerator" rather than "Violations" because for some
// metrics (Relationship Types, Resources, BEI) the numerator represents the
// GOOD count, not violations; "Numerator" is unambiguous and matches the
// Summary sheet's column header. The separate "Exceptions" count is the
// number of rows the user will see below on the detail sheet itself.
//
// When res.Population is nil (legacy results) the function returns a brief
// fallback line so the row is never blank.
//
// Signature: detailScopeBanner(res dcma.MetricResult) string
func detailScopeBanner(res dcma.MetricResult) string {
	exceptionsCount := len(res.Exceptions)
	if res.Population == nil {
		return fmt.Sprintf("Exceptions: %d  |  Rate: %.2f%%", exceptionsCount, res.Value*100)
	}
	p := res.Population
	var sb strings.Builder
	if p.Denominator > 0 {
		fmt.Fprintf(&sb, "Universe: %d %s  |  Numerator: %d  |  Exceptions: %d  |  Rate: %.2f%%",
			p.Denominator, p.Unit, p.Numerator, exceptionsCount, res.Value*100)
	} else {
		fmt.Fprintf(&sb, "Numerator: %d  |  Exceptions: %d  |  Rate: %.2f%%",
			p.Numerator, exceptionsCount, res.Value*100)
	}
	if p.ScopeLabel != "" {
		fmt.Fprintf(&sb, "\nScope: %s", p.ScopeLabel)
	}
	if len(p.Excluded) > 0 {
		parts := make([]string, 0, len(p.Excluded))
		for _, e := range p.Excluded {
			parts = append(parts, fmt.Sprintf("%s=%d", e.Reason, e.Count))
		}
		fmt.Fprintf(&sb, "\nExcluded: %s", strings.Join(parts, "; "))
	}
	return sb.String()
}

// writeUniverseSheet writes the "Universe" sheet, a single transparency view
// of every metric's denominator funnel. The sheet has two stacked sections:
//
//  1. File-wide task funnel — derived from assessment.Population — showing how
//     the raw row count is decomposed into summaries, milestones, work tasks,
//     and the completed/incomplete and baseline-finish breakdowns inside the
//     work-task slice. A footnote calls out that inactive (Active=No) tasks
//     are excluded at import time and therefore are not represented in any of
//     these counts.
//  2. Per-metric scope — one row per metric showing Unit, Numerator,
//     Denominator, Value, Threshold, the scope description text, and a
//     compact serialisation of the exclusion funnel for that metric.
//
// Signature: writeUniverseSheet(f *excelize.File, s styleIDs, assessment *dcma.DCMAAssessment) error
//
// Returns: error if any cell write or column-width call fails.
func writeUniverseSheet(f *excelize.File, s styleIDs, assessment *dcma.DCMAAssessment) error {
	const sheet = "Universe"
	if _, err := f.NewSheet(sheet); err != nil {
		return fmt.Errorf("failed to create universe sheet: %w", err)
	}

	_ = f.SetColWidth(sheet, "A", "A", 48)
	_ = f.SetColWidth(sheet, "B", "B", 12)
	_ = f.SetColWidth(sheet, "C", "C", 12)
	_ = f.SetColWidth(sheet, "D", "D", 12)
	_ = f.SetColWidth(sheet, "E", "E", 12)
	_ = f.SetColWidth(sheet, "F", "F", 14)
	_ = f.SetColWidth(sheet, "G", "G", 80)

	_ = f.MergeCell(sheet, "A1", "G1")
	_ = f.SetRowHeight(sheet, 1, 28)
	setCellStr(f, sheet, "A1", "Assessment Universe — how each metric's denominator was derived", s.titleCell)

	pop := assessment.Population
	_ = f.MergeCell(sheet, "A2", "G2")
	_ = f.SetRowHeight(sheet, 2, 18)
	setCellStr(f, sheet, "A2",
		"Counts reflect active tasks only; rows marked Active=No are excluded at import time and are NOT represented below.",
		s.metaLabel,
	)

	_ = f.SetRowHeight(sheet, 3, 6)
	_ = f.MergeCell(sheet, "A4", "G4")
	_ = f.SetRowHeight(sheet, 4, 22)
	setCellStr(f, sheet, "A4", "1. File-wide task classification", s.detailHeader)

	_ = f.SetRowHeight(sheet, 5, 20)
	setCellStr(f, sheet, "A5", "Category", s.colHeader)
	setCellStr(f, sheet, "B5", "Count", s.colHeader)
	for col := 'C'; col <= 'G'; col++ {
		setCellStr(f, sheet, fmt.Sprintf("%c5", col), "", s.colHeader)
	}

	funnel := []struct {
		label string
		count int
	}{
		{"Total active rows in schedule", pop.TotalRows},
		{"  Summary rows", pop.SummaryRows},
		{"  Milestones (Duration = 0)", pop.Milestones},
		{"  Work tasks", pop.WorkTasks},
		{"    Completed work tasks (% Complete = 100)", pop.CompletedWorkTasks},
		{"    Incomplete work tasks (% Complete < 100)", pop.IncompleteWorkTasks},
		{"    Work tasks with Baseline Finish populated", pop.WorkTasksWithBaselineFinish},
		{"    Work tasks with Baseline Finish ≤ Status Date (baseline-due)", pop.WorkTasksBaselineDueByStatus},
		{"  Assessable tasks: work + milestones (used by Missed Tasks PAM 4.11 and BEI DAU APMT-009)", pop.AssessableTasks},
		{"    Completed assessable tasks (% Complete = 100)", pop.CompletedAssessableTasks},
		{"    Assessable tasks with Baseline Finish populated", pop.AssessableTasksWithBaselineFinish},
		{"    Assessable tasks with Baseline Finish ≤ Status Date (BEI/Missed denominator)", pop.AssessableTasksBaselineDueByStatus},
	}
	row := 6
	for i, entry := range funnel {
		_ = f.SetRowHeight(sheet, row, 18)
		labelStyle := s.detailNormal
		countStyle := s.numCell
		if i%2 == 1 {
			labelStyle = s.detailAlt
			countStyle = s.altRowCell
		}
		setCellStr(f, sheet, cellAddr(row, 1), entry.label, labelStyle)
		setCellInt(f, sheet, cellAddr(row, 2), entry.count, countStyle)
		row++
	}

	row++ // visual gap
	_ = f.MergeCell(sheet, cellAddr(row, 1), cellAddr(row, 7))
	_ = f.SetRowHeight(sheet, row, 22)
	setCellStr(f, sheet, cellAddr(row, 1), "2. Per-metric scope and filtering funnel", s.detailHeader)
	row++

	headerRow := row
	_ = f.SetRowHeight(sheet, headerRow, 20)
	headers := []string{"Metric", "Unit", "Numerator", "Denominator", "Value", "Threshold", "Scope / Excluded"}
	for i, h := range headers {
		setCellStr(f, sheet, cellAddr(headerRow, i+1), h, s.colHeader)
	}
	row++

	for i, metric := range assessment.Metrics {
		res := assessment.Results[metric.Name()]
		alt := i%2 == 1
		baseStyle := s.detailNormal
		numStyle := s.numCell
		wrapStyle := s.wrapCell
		if alt {
			baseStyle = s.detailAlt
			numStyle = s.altRowCell
			wrapStyle = s.wrapAltCell
		}

		unitStr, numStr, denStr := metricCountStrings(res)
		valStr := metricValueString(res)
		thrStr := formatThreshold(metric.Threshold(), res)
		scopeText := scopeAndExclusionsText(res)
		lineCount := (len(scopeText) / 80) + 1
		rowH := float64(15 * lineCount)
		if rowH < 18 {
			rowH = 18
		}
		_ = f.SetRowHeight(sheet, row, rowH)

		setCellStr(f, sheet, cellAddr(row, 1), metric.Name(), baseStyle)
		setCellStr(f, sheet, cellAddr(row, 2), unitStr, baseStyle)
		setCellStr(f, sheet, cellAddr(row, 3), numStr, numStyle)
		setCellStr(f, sheet, cellAddr(row, 4), denStr, numStyle)
		setCellStr(f, sheet, cellAddr(row, 5), valStr, numStyle)
		setCellStr(f, sheet, cellAddr(row, 6), thrStr, numStyle)
		setCellStr(f, sheet, cellAddr(row, 7), scopeText, wrapStyle)
		row++
	}

	return nil
}

// metricCountStrings returns the Unit / Numerator / Denominator cell strings
// for the Universe and Summary tables. Returns "N/A" or empty strings when
// the metric has no Population (legacy results) or is Not Applicable.
//
// Signature: metricCountStrings(res dcma.MetricResult) (unit, numerator, denominator string)
func metricCountStrings(res dcma.MetricResult) (string, string, string) {
	if res.NotApplicable {
		return "N/A", "N/A", "N/A"
	}
	if res.Population == nil {
		return "tasks", "", ""
	}
	p := res.Population
	if p.Unit == "schedule" {
		// Schedule-level metrics (Critical Path Test, CPLI) don't have a
		// ratio; show "—" so the column is not misread as zero.
		num := "—"
		den := "—"
		if p.Denominator > 0 {
			num = fmt.Sprintf("%d", p.Numerator)
			den = fmt.Sprintf("%d", p.Denominator)
		}
		return p.Unit, num, den
	}
	return p.Unit, fmt.Sprintf("%d", p.Numerator), fmt.Sprintf("%d", p.Denominator)
}

// metricValueString returns the metric value as a percent string, or "N/A".
//
// Signature: metricValueString(res dcma.MetricResult) string
func metricValueString(res dcma.MetricResult) string {
	if res.NotApplicable {
		return "N/A"
	}
	return fmt.Sprintf("%.2f%%", res.Value*100)
}

// scopeAndExclusionsText returns the human-readable scope and exclusion line
// shown in the Universe sheet's per-metric table and in the per-metric detail
// sheet banner. Format:
//
//	"<scope> | Excluded: <reason>=<n>; <reason>=<n>; …"
//
// When res.Population is nil the function returns an empty string.
//
// Signature: scopeAndExclusionsText(res dcma.MetricResult) string
func scopeAndExclusionsText(res dcma.MetricResult) string {
	if res.Population == nil {
		return ""
	}
	p := res.Population
	if len(p.Excluded) == 0 {
		return p.ScopeLabel
	}
	parts := make([]string, 0, len(p.Excluded))
	for _, e := range p.Excluded {
		parts = append(parts, fmt.Sprintf("%s=%d", e.Reason, e.Count))
	}
	return fmt.Sprintf("%s  |  Excluded: %s", p.ScopeLabel, strings.Join(parts, "; "))
}
