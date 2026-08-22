package reader

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
	"github.com/xuri/excelize/v2"
)

// ScheduleReader reads and parses Microsoft Project schedule exports.
type ScheduleReader struct {
	filePath      string
	PctFormat     string   // "0-100" (default, MS Project) or "fraction" (0.0-1.0 scale)
	DateLocale    string   // "US" (default, MM/DD/YYYY first) or "EU" (DD/MM/YYYY first)
	dateWarnings    []string
	seenAmbiguous   map[string]bool
	seenUnparseable map[string]bool
}

// NewScheduleReader creates a new ScheduleReader.
func NewScheduleReader(filePath string) *ScheduleReader {
	return &ScheduleReader{filePath: filePath}
}

// Read reads and parses the schedule file.
func (r *ScheduleReader) Read() (*model.Schedule, error) {
	r.seenAmbiguous = make(map[string]bool)
	r.seenUnparseable = make(map[string]bool)
	r.dateWarnings = nil

	ext := strings.ToLower(filepath.Ext(r.filePath))
	var rows [][]string
	var err error

	if ext == ".xlsx" || ext == ".xls" {
		rows, err = r.readExcel()
	} else if ext == ".csv" {
		rows, err = r.readCSV()
	} else {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	headers, warnings := r.normalizeHeaders(rows[0])
	var missing []string
	for _, col := range RequiredColumns {
		if _, ok := headers[col]; !ok {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}
	tasks, inactive, err := r.parseTasks(headers, rows[1:])
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 && len(inactive) == 0 {
		return nil, fmt.Errorf("no tasks found in file: header row found but all data rows were empty or invalid")
	}

	// Calculate project metadata
	// DataDate: try filename first, fall back to today. Override with --status-date flag if needed.
	dataDate := extractDataDateFromFilename(r.filePath)
	dataDate = time.Date(dataDate.Year(), dataDate.Month(), dataDate.Day(), 0, 0, 0, 0, time.UTC)
	var validStarts []time.Time
	var validFinishes []time.Time

	for _, t := range tasks {
		if t.Start != nil {
			validStarts = append(validStarts, *t.Start)
		}
		if t.Finish != nil {
			validFinishes = append(validFinishes, *t.Finish)
		}
	}

	projectStart := time.Now()
	if len(validStarts) > 0 {
		projectStart = validStarts[0]
		for _, t := range validStarts {
			if t.Before(projectStart) {
				projectStart = t
			}
		}
	}

	projectFinish := time.Now()
	if len(validFinishes) > 0 {
		projectFinish = validFinishes[0]
		for _, t := range validFinishes {
			if t.After(projectFinish) {
				projectFinish = t
			}
		}
	}

	allWarnings := append(warnings, r.dateWarnings...)

	return &model.Schedule{
		Name:          strings.TrimSuffix(filepath.Base(r.filePath), ext),
		DataDate:      dataDate,
		ProjectStart:  projectStart,
		ProjectFinish: projectFinish,
		Tasks:         tasks,
		InactiveTasks: inactive,
		Warnings:      allWarnings,
	}, nil
}

// extractDataDateFromFilename tries to parse an IMS data date embedded in the filename.
// Strategy (highest-confidence first):
//  1. 8-digit sequences → try MMDDYYYY then YYYYMMDD
//  2. "st" prefix + 4-digit MMDD (e.g. "st0410" → April 10 of the current year)
//  3. 5-digit MDDYY / 6-digit MMDDYY compressed date (e.g. "41026" → 4/10/26)
//
// Falls back to time.Now() if no valid date is found.
func extractDataDateFromFilename(filePath string) time.Time {
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// 1. Try 8-digit sequences first (most specific)
	re8 := regexp.MustCompile(`\d{8}`)
	for _, match := range re8.FindAllString(base, -1) {
		if t, err := time.Parse("01022006", match); err == nil {
			return t
		}
		if t, err := time.Parse("20060102", match); err == nil {
			return t
		}
	}

	// 2. Try "st" + 4-digit MMDD pattern (e.g. "st0410" → April 10 of current year)
	// The "st" prefix is a common IMS convention for "status date".
	reSt := regexp.MustCompile(`(?i)st(\d{4})`)
	if m := reSt.FindStringSubmatch(base); m != nil {
		year := time.Now().Year()
		if t, err := time.Parse("01022006", m[1]+strconv.Itoa(year)); err == nil {
			return t
		}
	}

	// 3. Try 5-digit MDDYY (e.g. "41026" → 04/10/2026) or 6-digit MMDDYY
	re56 := regexp.MustCompile(`\d{5,6}`)
	for _, match := range re56.FindAllString(base, -1) {
		if len(match) == 5 {
			// Pad month to 2 digits: treat first digit as month
			padded := "0" + match // → MMDDYY (6 chars)
			if t, err := time.Parse("010206", padded); err == nil && t.Year() >= 2020 {
				return t
			}
		} else if len(match) == 6 {
			if t, err := time.Parse("010206", match); err == nil && t.Year() >= 2020 {
				return t
			}
		}
	}

	return time.Now()
}

func (r *ScheduleReader) readExcel() ([][]string, error) {
	f, err := excelize.OpenFile(r.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get first sheet name
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ScheduleReader) readCSV() ([][]string, error) {
	f, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	// Handle large fields if necessary, but default settings usually work
	return reader.ReadAll()
}

var columnMappings = map[string][]string{
	"task_id":           {"Task ID", "ID", "Task_ID", "TaskID"},
	"unique_id":         {"Unique ID", "UniqueID"},
	"name":              {"Task Name", "Name", "Task_Name", "TaskName", "Activity Name"},
	"duration":          {"Duration", "Duration (days)", "Dur", "Original Duration"},
	"baseline_duration": {"Duration1", "Baseline Duration", "Baseline_Duration", "BL Duration"},

	"start":                   {"Start", "Start Date", "Start_Date", "StartDate", "Early Start"},
	"finish":                  {"Finish", "Finish Date", "Finish_Date", "FinishDate", "Early Finish"},
	"predecessors":            {"Predecessors", "Pred", "Predecessor", "Predecessor IDs"},
	"percent_complete":        {"% Complete", "Percent Complete", "Percent_Complete", "Pct Complete"},
	"resources":               {"Resource Names", "Resources", "Resource", "Assigned Resources"},
	"wbs":                     {"WBS", "WBS Code", "Work Breakdown Structure"},
	"constraint_type":         {"Constraint Type", "Constraint_Type", "ConstraintType"},
	"constraint_date":         {"Constraint Date", "Constraint_Date", "ConstraintDate"},
	"total_slack":             {"Total Slack", "Total_Slack", "TotalSlack", "Float", "Total Float", "Total_Float"},
	"finish_variance":         {"Finish_Variance", "Finish Variance", "FinishVariance"},
	// task_variance_indicator and free_slack are mapped for cosmetic reasons only
	// (to prevent EXTRA warnings) but are NOT consumed by any metric/compare/report code.
	"task_variance_indicator": {"Task_Variance_Indicator", "Task Variance Indicator", "TaskVarianceIndicator", "Variance Indicator"},
	"free_slack":              {"Free Slack", "Free_Slack", "FreeSlack"},
	"outline_level":           {"Outline Level", "Outline_Level", "OutlineLevel", "Level"},
	"discipline":              {"Task Discipline", "Discipline", "Task_Discipline", "TaskDiscipline"},
	"mechanical_segment_nbr":  {"Mechanical_Segment_Nbr", "Mechanical Segment Nbr", "MechanicalSegmentNbr", "Mech Segment", "Mech_Seg", "Mech Seg"},
	"control_segment_nbr":     {"Control_Segment_Nbr", "Control Segment Nbr", "ControlSegmentNbr", "Controls Segment"},
	// "summary" maps the MS Project read-only boolean that marks a WBS parent row.
	// This is the authoritative source for IsSummary; it must take precedence over
	// "rollup", which is a different MS Project field controlling visual bar roll-up.
	"summary": {"Summary", "Is Summary", "IsSummary", "Is_Summary"},
	// "rollup" is the MS Project visual display field; used as a fallback for
	// IsSummary only when no dedicated "summary" column is present in the export.
	"rollup": {"Rollup"},
	"active":                  {"Active", "Is Active", "IsActive", "Is_Active", "Enabled"},
	"actual_start":            {"Actual_Start", "Actual Start", "ActualStart"},
	"actual_finish":           {"Actual_Finish", "Actual Finish", "ActualFinish"},
	// baseline_start is used only as intermediate input to derive baseline_duration
	// when the baseline_duration column is absent; baseline_finish is the column
	// actually consumed by Metrics 11 (Missed Tasks) and 14 (BEI).
	"baseline_start":          {"Baseline_Start", "Baseline Start", "BL Start", "BaselineStart"},
	"baseline_finish":         {"Baseline_Finish", "Baseline Finish", "BL Finish", "BaselineFinish"},
	// MS Project also exports Baseline 1 and Baseline 2 in the same CSV export.
	// We don't use them for assessment (Baseline 0 is the contractual baseline),
	// but mapping them prevents them from appearing as unrecognised "EXTRA" columns.
	"baseline1_start":         {"Baseline1_Start", "Baseline 1 Start", "BL1 Start"},
	"baseline1_finish":        {"Baseline1_Finish", "Baseline 1 Finish", "BL1 Finish"},
	"baseline2_start":         {"Baseline2_Start", "Baseline 2 Start", "BL2 Start"},
	"baseline2_finish":        {"Baseline2_Finish", "Baseline 2 Finish", "BL2 Finish"},
	"baseline2_duration":      {"Duration3", "Baseline2 Duration", "BL2 Duration", "Baseline2_Duration"},
	"segment_name":            {"Segment_Name", "Segment Name", "SegmentName"},
	"segment_type":            {"Segment_Type", "Segment Type", "SegmentType"},
}

// RequiredColumns are columns that must be present for DCMA assessment to work correctly.
var RequiredColumns = []string{
	"task_id",
	"name",
	"duration",
	"start",
	"finish",
	"predecessors",
	"percent_complete",
}

// OptionalColumns enhance analysis or are required only for specific commands (e.g. check-patterns).
// Columns are grouped by the DCMA metrics they enable.
var OptionalColumns = []string{
	// General logic / float / constraint metrics
	"constraint_type",
	"total_slack",
	"finish_variance",

	// Metric 8 (High Baseline Duration) — also used as fallback for baseline_finish
	"baseline_duration",

	// Metrics 9 (Invalid Dates): actual dates are compared to the status date
	"actual_start",
	"actual_finish",

	// Metrics 11 (Missed Tasks) and 14 (BEI): baseline_finish drives both denominators.
	// baseline_start is used only as intermediate input to derive baseline_duration
	// when the baseline_duration column is absent.
	// These are the Baseline 0 (original baseline) columns exported by MS Project.
	// Without baseline_finish, BEI and Missed Tasks cannot be assessed against the baseline.
	"baseline_start",
	"baseline_finish",

	// Task activation state — tasks with Active=No are excluded from all metrics
	"active",
	// Summary/rollup classification for WBS parent rows
	"summary",
	"rollup",

	// Labelling / grouping / reporting
	"unique_id",
	"resources",
	"discipline",
	"mechanical_segment_nbr",
	"control_segment_nbr",
	"wbs",
	"constraint_date",
	"outline_level",
	"segment_name",
	"segment_type",
	"baseline2_duration",
}

// ColumnValidationResult holds the results of column validation.
type ColumnValidationResult struct {
	Found    map[string]string // normalized key → original header name
	Missing  []string          // normalized keys not found
	Extra    []string          // unmapped column names from the file
	Warnings []string          // non-fatal issues encountered during parsing
}

// GetColumnMappings returns the column mappings for external use.
func GetColumnMappings() map[string][]string {
	return columnMappings
}

// ValidateColumns reads only the headers from a file and validates them.
func ValidateColumns(filePath string) (*ColumnValidationResult, error) {
	r := NewScheduleReader(filePath)

	ext := strings.ToLower(filepath.Ext(filePath))
	var rows [][]string
	var err error

	if ext == ".xlsx" || ext == ".xls" {
		rows, err = r.readExcel()
	} else if ext == ".csv" {
		rows, err = r.readCSV()
	} else {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	headers := rows[0]
	headerMap, warnings := r.normalizeHeaders(headers)

	result := &ColumnValidationResult{
		Found:    make(map[string]string),
		Missing:  []string{},
		Extra:    []string{},
		Warnings: warnings,
	}

	// Check which expected columns were found
	for key := range columnMappings {
		if idx, ok := headerMap[key]; ok {
			result.Found[key] = strings.TrimSpace(headers[idx])
		} else {
			result.Missing = append(result.Missing, key)
		}
	}

	// Find extra columns not in our mappings
	for _, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		found := false
		for key := range columnMappings {
			if idx, ok := headerMap[key]; ok && idx < len(headers) && strings.TrimSpace(headers[idx]) == h {
				found = true
				break
			}
		}
		if !found {
			result.Extra = append(result.Extra, h)
		}
	}

	// Warn when neither summary nor rollup column is present — this silently
	// degrades all 14 DCMA metrics because IsSummary defaults to false for all
	// tasks, causing summary roll-up rows to be included in metric calculations.
	_, hasSummary := headerMap["summary"]
	_, hasRollup := headerMap["rollup"]
	if !hasSummary && !hasRollup {
		result.Warnings = append(result.Warnings,
			"Neither 'Summary' nor 'Rollup' column found — all tasks will be treated as non-summary; this affects all 14 DCMA metrics which exclude summary rows")
	}

	return result, nil
}

func (r *ScheduleReader) normalizeHeaders(headers []string) (map[string]int, []string) {
	headerMap := make(map[string]int)
	seen := make(map[string]bool)
	var duplicateWarnings []string
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if i == 0 {
			h = strings.TrimPrefix(h, "\xEF\xBB\xBF")
			h = strings.TrimSpace(h)
		}
		found := false
		for key, variants := range columnMappings {
			for _, v := range variants {
				if strings.EqualFold(h, v) {
					if seen[key] {
						duplicateWarnings = append(duplicateWarnings, fmt.Sprintf(
							"ambiguous column %q (column %d) maps to %q but already mapped from earlier column; skipping", h, i+1, key))
					} else {
						headerMap[key] = i
						seen[key] = true
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	return headerMap, duplicateWarnings
}

// parseTasks parses the body rows into active and inactive task slices.
//
// Signature: parseTasks(headerMap map[string]int, rows [][]string) ([]*model.Task, []*model.Task, error)
// Arguments:
//   - headerMap: column index map from normalizeHeaders
//   - rows:      data rows (header row already removed)
//
// Returns:
//   - active:   tasks with Active=Yes (or no Active column present); these
//               drive every DCMA metric
//   - inactive: tasks explicitly marked Active=No; excluded from every metric
//               and exposed via Schedule.InactiveTasks for diagnostic reports
//   - err:      reserved for future parser errors (currently always nil)
func (r *ScheduleReader) parseTasks(headerMap map[string]int, rows [][]string) ([]*model.Task, []*model.Task, error) {
	needsPctNormalization := r.PctFormat == "fraction"

	var tasks []*model.Task
	var inactive []*model.Task
	for i, row := range rows {
		// Skip empty rows logic if needed, but for now specific empty checks
		if len(row) == 0 {
			continue
		}

		taskID := r.getValue(row, headerMap, "task_id")
		uniqueID := r.getValue(row, headerMap, "unique_id")
		name := r.getValue(row, headerMap, "name")
		if taskID == "" && uniqueID == "" && name == "" {
			continue
		}

		if name == "" || strings.EqualFold(name, "nan") {
			name = "(Unnamed)"
		}
		// Prefer the visible "Task ID" / "ID" column so the report value always
		// matches what the user sees in MS Project. Fall back to "Unique ID" only
		// when no primary ID column is present, and finally to a 1-based row index.
		if taskID == "" {
			if uniqueID != "" {
				taskID = uniqueID
			} else {
				taskID = strconv.Itoa(i + 1)
			}
		}

		// Helper to safely parse strings
		getStr := func(key string) string {
			return r.getValue(row, headerMap, key)
		}

		duration := r.parseDuration(getStr("duration"))
		baselineDuration := r.parseDuration(getStr("baseline_duration"))
		if baselineDuration == 0 {
			// Fallback 1: derive from baseline dates if available (more accurate than scheduled duration)
			bsRaw := getStr("baseline_start")
			bfRaw := getStr("baseline_finish")
			if bsRaw != "" && bfRaw != "" {
				bsDate := r.parseDate(bsRaw)
				bfDate := r.parseDate(bfRaw)
				if bsDate != nil && bfDate != nil && bfDate.After(*bsDate) {
					calDays := bfDate.Sub(*bsDate).Hours() / 24
					baselineDuration = calDays * 5 / 7
				}
			}
			// Fallback 2: scheduled duration
			if baselineDuration == 0 {
				baselineDuration = duration
			}
		}
		start := r.parseDate(getStr("start"))
		finish := r.parseDate(getStr("finish"))
		actualStart := r.parseDate(getStr("actual_start"))
		actualFinish := r.parseDate(getStr("actual_finish"))
		baselineStart := r.parseDate(getStr("baseline_start"))
		baselineFinish := r.parseDate(getStr("baseline_finish"))
		// Fallback: derive baseline_finish from finish - finish_variance when column absent
		if baselineFinish == nil {
			_, hasBaselineFinish := headerMap["baseline_finish"]
			_, hasFinishVar := headerMap["finish_variance"]
			if !hasBaselineFinish && hasFinishVar && finish != nil {
				varianceDays := r.parseDuration(getStr("finish_variance"))
				derived := finish.AddDate(0, 0, -int(varianceDays))
				baselineFinish = &derived
			}
		}
		pctComplete := r.parseFloat(getStr("percent_complete"))
		// PctFormat controls the scale of the percent_complete column.
		// Default "0-100": values are used as-is (MS Project CSV export standard).
		// "fraction": multiply by 100 to convert 0.0-1.0 to 0-100 (Primavera / other tools).
		if needsPctNormalization {
			pctComplete *= 100
		}
		constraintDate := r.parseDate(getStr("constraint_date"))
		_, hasTotalSlack := headerMap["total_slack"]
		_, hasFinishVariance := headerMap["finish_variance"]
		// Slack exports are often duration strings (e.g., "121d", "3w"), so parse as duration.
		totalSlack := r.parseDuration(getStr("total_slack"))
		if !hasTotalSlack && hasFinishVariance {
			// Finish_Variance proxy: negate so that behind-baseline (FV>0) → negative slack,
			// on-baseline (FV=0) → critical (slack=0), ahead-of-baseline (FV<0) → positive float.
			totalSlack = -r.parseDuration(getStr("finish_variance"))
		}
		freeSlack := r.parseDuration(getStr("free_slack"))
		outlineLevel := r.parseInt(getStr("outline_level"))

		// Is Milestone logic: duration must be explicitly present AND equal to zero.
		// An empty or unparseable duration cell returns 0.0 from parseDuration, so
		// we guard against misclassifying tasks whose duration column is simply blank.
		durationRaw := getStr("duration")
		durationPresent := len(strings.TrimSpace(durationRaw)) > 0 && !strings.EqualFold(strings.TrimSpace(durationRaw), "nan")
		isMilestone := durationPresent && duration == 0

		// IsSummary is derived from the MS Project "Summary" column when present —
		// that field is a read-only boolean MS Project sets automatically on WBS
		// parent rows. When the export omits "Summary" but includes "Rollup", fall
		// back to Rollup as a proxy (summary tasks always have Rollup=Yes, though
		// some work tasks may also have it set for visual purposes).
		_, hasSummaryCol := headerMap["summary"]
		summaryRaw := strings.ToLower(strings.TrimSpace(getStr("summary")))
		rollupRaw := strings.ToLower(strings.TrimSpace(getStr("rollup")))
		isSummaryVal := summaryRaw
		if !hasSummaryCol {
			isSummaryVal = rollupRaw
		}
		isSummary := isSummaryVal == "yes" || isSummaryVal == "true" || isSummaryVal == "1"

		// Active logic: missing column or blank cell defaults to active.
		// Explicit "Yes"/"true"/"1" means active; "No"/"false"/"0" means inactive.
		_, hasActiveCol := headerMap["active"]
		activeVal := strings.ToLower(strings.TrimSpace(getStr("active")))
		isActive := !hasActiveCol || activeVal == "" || activeVal == "yes" || activeVal == "true" || activeVal == "1"

		task := &model.Task{
			TaskID:               taskID,
			UniqueID:             uniqueID,
			Name:                 name,
			Duration:             duration,
			Start:                start,
			Finish:               finish,
			Predecessors:         getStr("predecessors"),
			PercentComplete:      pctComplete,
			Resources:            getStr("resources"),
			WBS:                  getStr("wbs"),
			ConstraintType:       getStr("constraint_type"),
			ConstraintDate:       constraintDate,
			IsMilestone:          isMilestone,
			IsSummary:            isSummary,
			Active:               isActive,
			TotalSlack:           totalSlack,
			FreeSlack:            freeSlack,
			OutlineLevel:         outlineLevel,
			BaselineDuration:     baselineDuration,
			ActualStart:          actualStart,
			ActualFinish:         actualFinish,
			BaselineStart:        baselineStart,
			BaselineFinish:       baselineFinish,
			Discipline:           getStr("discipline"),
			MechanicalSegmentNbr: getStr("mechanical_segment_nbr"),
			ControlSegmentNbr:    getStr("control_segment_nbr"),
		}
		if isActive {
			tasks = append(tasks, task)
		} else {
			// Disabled tasks must not appear in any metric assessment, but the
			// schedule keeps them on the side so diagnostics (e.g. Logic
			// metric notes about inactive successors) can reference them.
			inactive = append(inactive, task)
		}
	}
	return tasks, inactive, nil
}

func (r *ScheduleReader) getValue(row []string, headerMap map[string]int, key string) string {
	idx, ok := headerMap[key]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// daysPerMonth converts month-unit durations (e.g. "1mo", "1.2mo") to working
// days. 20 is the MS Project default ("Days per month" in Project Options) and
// matches the DCMA PAM convention. Centralized here so it can be promoted to a
// config flag later if a customer's calendar differs.
const daysPerMonth = 20.0

// durationUnit pairs a textual suffix (e.g. "mo") with the multiplier that
// converts the numeric magnitude to working days.
type durationUnit struct {
	suffix string
	factor float64
}

// durationUnits enumerates every MS Project duration suffix the reader
// understands, **in longest-suffix-first order**. Ordering matters because
// HasSuffix is greedy from the right: "mos" must be tried before "mo", and
// "days"/"day" before the bare "d". Bare "m" (minutes) is intentionally
// omitted — it collides visually with "mo" and is not used in DCMA schedules.
var durationUnits = []durationUnit{
	{"mos", daysPerMonth}, {"mon", daysPerMonth}, {"mo", daysPerMonth},
	{"days", 1}, {"day", 1},
	{"wks", 5}, {"wk", 5}, {"w", 5},
	{"hrs", 1.0 / 8.0}, {"hr", 1.0 / 8.0}, {"h", 1.0 / 8.0},
	{"d", 1},
}

// parseDuration converts a Microsoft Project duration string to working days.
//
// Signature: (r *ScheduleReader) parseDuration(val string) float64
// Arguments:
//   - val: raw duration cell, e.g. "5d", "12w", "1mo", "1.2mo", "793d?",
//     "4h", "" (whitespace-tolerant, case-insensitive).
//
// Returns:
//   - float64 number of working days. Unparseable / empty values return 0.
//
// Conversion table:
//   - mo / mos / mon → 20 working days  (MS Project "Days per month" default)
//   - w  / wk  / wks → 5  working days
//   - d  / day / days → 1 day (unit-less inputs are also treated as days)
//   - h  / hr  / hrs → 0.125 day (8-hour working day)
//
// MS Project appends "?" to estimated durations (e.g. "1d?", "793d?"); the
// suffix is stripped before unit matching so estimated tasks parse to a
// non-zero magnitude and are not misclassified as zero-duration milestones.
func (r *ScheduleReader) parseDuration(val string) float64 {
	v := strings.ToLower(strings.TrimSpace(val))
	if v == "" {
		return 0
	}
	v = strings.TrimSpace(strings.TrimSuffix(v, "?"))
	for _, u := range durationUnits {
		if strings.HasSuffix(v, u.suffix) {
			num := strings.TrimSpace(strings.TrimSuffix(v, u.suffix))
			if f, err := strconv.ParseFloat(num, 64); err == nil {
				return f * u.factor
			}
			return 0
		}
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func (r *ScheduleReader) parseFloat(val string) float64 {
	if val == "" {
		return 0
	}
	val = strings.ReplaceAll(val, "%", "")
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

func (r *ScheduleReader) parseInt(val string) int {
	if val == "" {
		return 1 // Default level 1
	}
	i, _ := strconv.Atoi(val)
	if i == 0 {
		return 1
	}
	return i
}

func (r *ScheduleReader) parseDate(val string) *time.Time {
	if val == "" || strings.EqualFold(val, "NA") || strings.EqualFold(val, "N/A") {
		return nil
	}

	usSlash := "01/02/2006"
	euSlash := "02/01/2006"
	usSlashShort := "01/02/06"
	euSlashShort := "02/01/06"
	usSlashDT := "01/02/2006 15:04:05"
	euSlashDT := "02/01/2006 15:04:05"

	firstSlash, secondSlash := usSlash, euSlash
	firstShort, secondShort := usSlashShort, euSlashShort
	firstDT, secondDT := usSlashDT, euSlashDT
	if r.DateLocale == "EU" {
		firstSlash, secondSlash = secondSlash, firstSlash
		firstShort, secondShort = secondShort, firstShort
		firstDT, secondDT = secondDT, firstDT
	}

	formats := []string{
		"2006-01-02",
		firstSlash,
		secondSlash,
		"2006/01/02",
		"January 02, 2006 3:04 PM",
		"January 2, 2006 3:04 PM",
		"Jan 02, 2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
		firstShort,
		secondShort,
		"Mon 01/02/06",
		"Mon 01/2/06",
		"Mon 1/02/06",
		"Mon 1/2/06",
		"2006-01-02 15:04:05",
		firstDT,
		secondDT,
	}

	var result *time.Time
	var matchedSlash string
	for _, fmtStr := range formats {
		t, err := time.Parse(fmtStr, val)
		if err == nil {
			result = &t
			switch fmtStr {
			case usSlash, euSlash, usSlashShort, euSlashShort, usSlashDT, euSlashDT:
				matchedSlash = fmtStr
			}
			break
		}
	}

	if result == nil {
		if !r.seenUnparseable[val] {
			r.seenUnparseable[val] = true
			r.dateWarnings = append(r.dateWarnings, fmt.Sprintf(
				"unparseable date %q: no format matched; date treated as nil",
				val,
			))
		}
		return nil
	}

	if matchedSlash != "" {
		opposite := ""
		switch matchedSlash {
		case usSlash:
			opposite = euSlash
		case euSlash:
			opposite = usSlash
		case usSlashShort:
			opposite = euSlashShort
		case euSlashShort:
			opposite = usSlashShort
		case usSlashDT:
			opposite = euSlashDT
		case euSlashDT:
			opposite = usSlashDT
		}
		if opposite != "" {
			if alt, err := time.Parse(opposite, val); err == nil && !alt.Equal(*result) {
				altLabel := "EU"
				usedLabel := "US"
				if r.DateLocale == "EU" {
					altLabel = "US"
					usedLabel = "EU"
				}
				if !r.seenAmbiguous[val] {
					r.seenAmbiguous[val] = true
					r.dateWarnings = append(r.dateWarnings, fmt.Sprintf(
						"ambiguous date %q: parsed as %s (%s locale); alternative is %s (%s locale)",
						val, result.Format("2006-01-02"), usedLabel,
						alt.Format("2006-01-02"), altLabel,
					))
				}
			}
		}
	}

	return result
}
