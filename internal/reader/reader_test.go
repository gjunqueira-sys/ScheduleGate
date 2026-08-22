package reader

import (
	"strings"
	"testing"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
)

func buildHeaderMap(keys ...string) map[string]int {
	m := make(map[string]int)
	for i, k := range keys {
		m[k] = i
	}
	return m
}

func makeRow(vals ...string) []string {
	return vals
}

// ---------------------------------------------------------------------------
// parseDuration
// ---------------------------------------------------------------------------

func TestParseDuration(t *testing.T) {
	r := &ScheduleReader{}

	cases := []struct {
		input string
		want  float64
		desc  string
	}{
		// Normal day values
		{"5d", 5, "plain days"},
		{"0d", 0, "zero days"},
		{"10", 10, "no unit"},
		{"23d", 23, "23 days"},
		{"103d", 103, "103 days"},

		// Week conversion (1w = 5 working days)
		{"2w", 10, "2 weeks"},
		{"12w", 60, "12 weeks"},

		// MS Project estimated-duration suffix '?' — bug fixed in this session.
		// Before the fix, parseDuration stripped 'd' but not '?', so
		// ParseFloat("793?") returned 0, misclassifying the task as a milestone.
		{"793d?", 793, "estimated duration suffix (project summary)"},
		{"783d?", 783, "estimated duration suffix (summary task)"},
		{"694d?", 694, "estimated duration suffix (main milestones summary)"},
		{"1d?", 1, "estimated 1-day task (Customer Delay task)"},
		{"724d?", 724, "estimated duration suffix (procurement summary)"},

		// Month units — BUG-012 fix. MS Project exports "Nmo" for tasks whose
		// Duration is entered in months. Prior to this fix the tokenizer
		// silently returned 0, misclassifying multi-week activities (e.g. the
		// 24 invoice-payment tasks in a sample schedule) as milestones.
		// 20 working days/month is the MS Project default and matches DCMA PAM.
		{"1mo", 20, "1 month (sample invoice-payment task)"},
		{"2mo", 40, "2 months"},
		{"1.2mo", 24, "fractional months (1.2mo)"},
		{"1mo?", 20, "estimated month-unit duration"},
		{"1mos", 20, "month alias: mos"},
		{"1mon", 20, "month alias: mon"},
		{"1 mo", 20, "month with whitespace separator"},
		{"12mo", 240, "12 months"},

		// Hour units — defensive addition; converts to working days using an
		// 8-hour day so an hour-unit duration is not parsed as a milestone.
		{"4h", 0.5, "4 hours (half working day)"},
		{"8hrs", 1, "8 hours (one working day)"},
		{"2hr", 0.25, "2 hours"},

		// Week alias regression guards
		{"3wk", 15, "week alias: wk"},
		{"2wks", 10, "week alias: wks"},

		// Edge cases
		{"", 0, "empty string"},
		{"0d?", 0, "estimated zero duration"},
		{"0mo", 0, "zero months parses as zero"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := r.parseDuration(tc.input)
			if got != tc.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// Milestone detection depends on parseDuration returning 0 only for true
// zero-duration tasks, not for tasks with estimated suffixes.
func TestMilestoneDetectionWithEstimatedDuration(t *testing.T) {
	r := &ScheduleReader{}

	// These should NOT be milestones (duration > 0). The "*mo" entries are
	// drawn from a sample schedule (task ids 35-60: contract execution,
	// monthly invoice payments, final payment) — these were misclassified as
	// milestones by the pre-BUG-012 parser, shifting the incomplete-work
	// denominator from the DCMA-correct 391 down to 387.
	nonMilestones := []string{"1d?", "793d?", "12w", "5d", "1mo", "2mo", "1.2mo", "4h"}
	for _, v := range nonMilestones {
		if r.parseDuration(v) == 0 {
			t.Errorf("parseDuration(%q) returned 0 — task would be wrongly classified as a milestone", v)
		}
	}

	// These SHOULD be milestones (duration == 0)
	milestones := []string{"0d", "0", "", "0mo"}
	for _, v := range milestones {
		if r.parseDuration(v) != 0 {
			t.Errorf("parseDuration(%q) returned non-zero — task should be a milestone", v)
		}
	}
}

// ---------------------------------------------------------------------------
// extractDataDateFromFilename
// ---------------------------------------------------------------------------

func TestExtractDataDateFromFilename(t *testing.T) {
	currentYear := time.Now().Year()

	cases := []struct {
		filename string
		wantYear int
		wantMon  time.Month
		wantDay  int
		desc     string
	}{
		// 8-digit MMDDYYYY — original behaviour preserved
		{
			"schedule_04102026.csv",
			2026, time.April, 10,
			"8-digit MMDDYYYY",
		},
		// 8-digit YYYYMMDD
		{
			"schedule_20260410.csv",
			2026, time.April, 10,
			"8-digit YYYYMMDD",
		},
		// 'st' + 4-digit MMDD — new pattern added this session.
		// Filename convention: st0410 → April 10 of the current year.
		{
			"9167495_grainger_41026_st0410 (1).csv",
			currentYear, time.April, 10,
			"st+MMDD pattern (sample file)",
		},
		{
			"project_st0312.csv",
			currentYear, time.March, 12,
			"st+MMDD pattern (March 12)",
		},
		{
			"report_ST0101.csv",
			currentYear, time.January, 1,
			"st+MMDD case-insensitive",
		},
		// 5-digit MDDYY — new pattern: '41026' → 4/10/26 → April 10 2026
		{
			"9167495_grainger_41026.csv",
			2026, time.April, 10,
			"5-digit MDDYY",
		},
		{
			"project_31226.csv",
			2026, time.March, 12,
			"5-digit MDDYY (March 12 2026)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := extractDataDateFromFilename(tc.filename)
			if got.Year() != tc.wantYear || got.Month() != tc.wantMon || got.Day() != tc.wantDay {
				t.Errorf("extractDataDateFromFilename(%q) = %s, want %04d-%02d-%02d",
					tc.filename, got.Format("2006-01-02"), tc.wantYear, int(tc.wantMon), tc.wantDay)
			}
		})
	}
}

func TestExtractDataDateFromFilename_Fallback(t *testing.T) {
	// When no recognisable date is in the filename, the function must fall back
	// to time.Now(). We accept any result within 5 seconds of now.
	before := time.Now().Add(-5 * time.Second)
	got := extractDataDateFromFilename("my_schedule_no_date.csv")
	after := time.Now().Add(5 * time.Second)

	if got.Before(before) || got.After(after) {
		t.Errorf("expected fallback to time.Now(), got %s", got)
	}
}

// ---------------------------------------------------------------------------
// OptionalColumns — baseline columns must be declared
// ---------------------------------------------------------------------------

func TestOptionalColumnsIncludeBaselineColumns(t *testing.T) {
	// These columns drive BEI, Missed Tasks, High Duration, and Invalid Dates.
	// Prior to this session's fix they were silently consumed by the reader but
	// never surfaced in the validate command. They must now be in OptionalColumns.
	required := []string{
		"baseline_start",
		"baseline_finish",
		"baseline_duration",
		"actual_start",
		"actual_finish",
	}

	optSet := make(map[string]bool, len(OptionalColumns))
	for _, c := range OptionalColumns {
		optSet[c] = true
	}

	for _, col := range required {
		if !optSet[col] {
			t.Errorf("OptionalColumns is missing %q — validate command will not report its presence", col)
		}
	}
}

// ---------------------------------------------------------------------------
// parseDate
// ---------------------------------------------------------------------------

func TestPctFormatDefault_KeepsValuesAsIs(t *testing.T) {
	// Default PctFormat ("") means 0-100 scale — values are kept as-is.
	headerMap := buildHeaderMap("name", "percent_complete")
	rows := [][]string{
		makeRow("Task A", "37"),
		makeRow("Task B", "100"),
		makeRow("Task C", "1"),
	}

	r := &ScheduleReader{}
	tasks, _, err := r.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].PercentComplete != 37 {
		t.Errorf("Task A: expected 37, got %.0f", tasks[0].PercentComplete)
	}
	if tasks[1].PercentComplete != 100 {
		t.Errorf("Task B: expected 100, got %.0f", tasks[1].PercentComplete)
	}
	if tasks[2].PercentComplete != 1 {
		t.Errorf("Task C: expected 1 (not 100), got %.0f", tasks[2].PercentComplete)
	}
}

func TestPctFormatFraction_MultipliesBy100(t *testing.T) {
	headerMap := buildHeaderMap("name", "percent_complete")
	rows := [][]string{
		makeRow("Task A", "0.37"),
		makeRow("Task B", "1.0"),
		makeRow("Task C", "0"),
	}

	r := &ScheduleReader{PctFormat: "fraction"}
	tasks, _, err := r.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].PercentComplete != 37 {
		t.Errorf("Task A: expected 37, got %.0f", tasks[0].PercentComplete)
	}
	if tasks[1].PercentComplete != 100 {
		t.Errorf("Task B: expected 100, got %.0f", tasks[1].PercentComplete)
	}
	if tasks[2].PercentComplete != 0 {
		t.Errorf("Task C: expected 0, got %.0f", tasks[2].PercentComplete)
	}
}

func TestPctFormatExplicit0To100_KeepsValuesAsIs(t *testing.T) {
	headerMap := buildHeaderMap("name", "percent_complete")
	rows := [][]string{
		makeRow("Task A", "50"),
		makeRow("Task B", "1"),
	}

	r := &ScheduleReader{PctFormat: "0-100"}
	tasks, _, err := r.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].PercentComplete != 50 {
		t.Errorf("Task A: expected 50, got %.0f", tasks[0].PercentComplete)
	}
	if tasks[1].PercentComplete != 1 {
		t.Errorf("Task B: expected 1, got %.0f", tasks[1].PercentComplete)
	}
}

func TestPctFormatUnknown_DefaultsTo0To100(t *testing.T) {
	headerMap := buildHeaderMap("name", "percent_complete")
	rows := [][]string{
		makeRow("Task A", "42"),
	}

	r := &ScheduleReader{PctFormat: "garbage"}
	tasks, _, err := r.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if tasks[0].PercentComplete != 42 {
		t.Errorf("expected 42 (default 0-100 behavior), got %.0f", tasks[0].PercentComplete)
	}
}

func TestPctFormatFraction_EarlyProjectWithOnly0And1(t *testing.T) {
	// The old heuristic would fail here: all values ≤ 1.0 would be treated as
	// fraction, multiplying 1 → 100. With explicit PctFormat="fraction", this
	// is correct behavior. With default (no flag), these stay as 0 and 1.
	headerMap := buildHeaderMap("name", "percent_complete")
	rows := [][]string{
		makeRow("Task A", "0"),
		makeRow("Task B", "1"),
	}

	// Fails the old heuristic, correct with explicit flag
	r := &ScheduleReader{PctFormat: "fraction"}
	tasks, _, err := r.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if tasks[0].PercentComplete != 0 {
		t.Errorf("Task A: expected 0, got %.0f", tasks[0].PercentComplete)
	}
	if tasks[1].PercentComplete != 100 {
		t.Errorf("Task B: expected 100 (fraction 1.0), got %.0f", tasks[1].PercentComplete)
	}

	// Default behavior — values stay as-is (no false promotion)
	r2 := &ScheduleReader{}
	tasks2, _, err := r2.parseTasks(headerMap, rows)
	if err != nil {
		t.Fatal(err)
	}
	if tasks2[1].PercentComplete != 1 {
		t.Errorf("Default: Task B expected 1 (not 100), got %.0f", tasks2[1].PercentComplete)
	}
}

func TestParseDate(t *testing.T) {
	r := &ScheduleReader{}

	cases := []struct {
		input   string
		wantNil bool
		wantY   int
		wantM   time.Month
		wantD   int
		desc    string
	}{
		{"", true, 0, 0, 0, "empty string"},
		{"NA", true, 0, 0, 0, "NA sentinel"},
		{"N/A", true, 0, 0, 0, "N/A sentinel"},
		{"2026-04-10", false, 2026, time.April, 10, "ISO format"},
		{"01/22/2025", false, 2025, time.January, 22, "MM/DD/YYYY"},
		{"January 22, 2025 12:00 AM", false, 2025, time.January, 22, "long MS Project format"},
		{"Wed 1/22/25", false, 2025, time.January, 22, "short MS Project dow format"},
		{"Fri 4/25/25", false, 2025, time.April, 25, "short MS Project dow format 2"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := r.parseDate(tc.input)
			if tc.wantNil {
				if got != nil {
					t.Errorf("parseDate(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseDate(%q) = nil, want %04d-%02d-%02d", tc.input, tc.wantY, int(tc.wantM), tc.wantD)
			}
			if got.Year() != tc.wantY || got.Month() != tc.wantM || got.Day() != tc.wantD {
				t.Errorf("parseDate(%q) = %s, want %04d-%02d-%02d",
					tc.input, got.Format("2006-01-02"), tc.wantY, int(tc.wantM), tc.wantD)
			}
		})
	}
}

func TestParseDate_EULocale(t *testing.T) {
	r := &ScheduleReader{DateLocale: "EU", seenAmbiguous: make(map[string]bool)}

	cases := []struct {
		input   string
		wantNil bool
		wantY   int
		wantM   time.Month
		wantD   int
		desc    string
	}{
		{"22/01/2025", false, 2025, time.January, 22, "DD/MM/YYYY (EU primary)"},
		{"22/01/25", false, 2025, time.January, 22, "DD/MM/YY (EU short)"},
		{"05/04/2026", false, 2026, time.April, 5, "ambiguous date using EU locale → April 5"},
		{"2026-04-10", false, 2026, time.April, 10, "ISO format (unaffected by locale)"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := r.parseDate(tc.input)
			if tc.wantNil {
				if got != nil {
					t.Errorf("parseDate(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseDate(%q) = nil, want %04d-%02d-%02d", tc.input, tc.wantY, int(tc.wantM), tc.wantD)
			}
			if got.Year() != tc.wantY || got.Month() != tc.wantM || got.Day() != tc.wantD {
				t.Errorf("parseDate(%q) = %s, want %04d-%02d-%02d",
					tc.input, got.Format("2006-01-02"), tc.wantY, int(tc.wantM), tc.wantD)
			}
		})
	}
}

func TestParseDate_AmbiguousWarning(t *testing.T) {
	r := &ScheduleReader{DateLocale: "US", seenAmbiguous: make(map[string]bool)}

	got := r.parseDate("05/04/2026")
	if got == nil {
		t.Fatal("parseDate returned nil")
	}
	if got.Month() != time.May || got.Day() != 4 {
		t.Errorf("expected May 4 (US locale), got %s", got.Format("2006-01-02"))
	}

	if len(r.dateWarnings) != 1 {
		t.Fatalf("expected 1 ambiguous date warning, got %d: %v", len(r.dateWarnings), r.dateWarnings)
	}
	if !strings.Contains(r.dateWarnings[0], "ambiguous date") {
		t.Errorf("warning should mention ambiguous date, got: %q", r.dateWarnings[0])
	}
	if !strings.Contains(r.dateWarnings[0], "US locale") {
		t.Errorf("warning should mention US locale, got: %q", r.dateWarnings[0])
	}
	if !strings.Contains(r.dateWarnings[0], "EU locale") {
		t.Errorf("warning should mention EU locale, got: %q", r.dateWarnings[0])
	}

	parsedSameAgain := r.parseDate("05/04/2026")
	if parsedSameAgain == nil || parsedSameAgain.Month() != time.May {
		t.Error("same ambiguous date should still parse correctly on second call")
	}
	if len(r.dateWarnings) != 1 {
		t.Errorf("should not produce duplicate warnings, got %d", len(r.dateWarnings))
	}
}

func TestParseDate_AmbiguousWarning_ELocale(t *testing.T) {
	r := &ScheduleReader{DateLocale: "EU", seenAmbiguous: make(map[string]bool)}

	got := r.parseDate("05/04/2026")
	if got == nil {
		t.Fatal("parseDate returned nil")
	}
	if got.Month() != time.April || got.Day() != 5 {
		t.Errorf("expected April 5 (EU locale), got %s", got.Format("2006-01-02"))
	}

	if len(r.dateWarnings) != 1 {
		t.Fatalf("expected 1 ambiguous date warning, got %d", len(r.dateWarnings))
	}
	if !strings.Contains(r.dateWarnings[0], "EU locale") {
		t.Errorf("warning should name EU as the used locale, got: %q", r.dateWarnings[0])
	}
	if !strings.Contains(r.dateWarnings[0], "US locale") {
		t.Errorf("warning should name US as the alternative, got: %q", r.dateWarnings[0])
	}
}

func TestRead_HeaderOnlyFileReturnsError(t *testing.T) {
	r := NewScheduleReader("testdata/header_only.csv")
	_, err := r.Read()
	if err == nil {
		t.Fatal("expected error for file with header only and no data rows")
	}
}

func TestRead_AllEmptyDataRowsReturnsError(t *testing.T) {
	r := NewScheduleReader("testdata/empty_data_rows.csv")
	_, err := r.Read()
	if err == nil {
		t.Fatal("expected error for file with header and all-empty data rows")
	}
}

// ---------------------------------------------------------------------------
// Anonymous Schedule Integration Tests
// ---------------------------------------------------------------------------

func TestRead_AnonymousSchedule(t *testing.T) {
	// Test reading the large anonymous test schedule fixture
	r := NewScheduleReader("testdata/test_anonymous_schedule.csv")
	sched, err := r.Read()
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	if sched.Name != "test_anonymous_schedule" {
		t.Errorf("Name = %q, want test_anonymous_schedule", sched.Name)
	}

	// Verify we have tasks (the schedule has 6146 data rows)
	if len(sched.Tasks) == 0 {
		t.Fatal("expected tasks in schedule")
	}

	// Verify no identifying references to "Ross" or "Dematic" exist in task names
	for _, task := range sched.Tasks {
		if len(task.Name) > 100 {
			t.Errorf("task %s name too long: %d chars", task.TaskID, len(task.Name))
		}
		// Check task names don't contain identifying strings
		nameLower := strings.ToLower(task.Name)
		if strings.Contains(nameLower, "ross") {
			t.Errorf("task %s name contains 'Ross': %q", task.TaskID, task.Name)
		}
		if strings.Contains(nameLower, "demat") {
			t.Errorf("task %s name contains 'Dematic': %q", task.TaskID, task.Name)
		}
	}
}

func TestRead_AnonymousScheduleV2(t *testing.T) {
	// Test reading the modified v2 schedule (for compare testing)
	r := NewScheduleReader("testdata/test_anonymous_schedule_v2.csv")
	sched, err := r.Read()
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	if sched.Name != "test_anonymous_schedule_v2" {
		t.Errorf("Name = %q, want test_anonymous_schedule_v2", sched.Name)
	}

	// Should have similar task count as v1
	if len(sched.Tasks) == 0 {
		t.Fatal("expected tasks in schedule")
	}
}

func TestResourcesColumnMatching(t *testing.T) {
	// Test that reader correctly loads Resources column from test fixture
	// The fixture uses "Resource Names" header (canonical MS Project)
	r := NewScheduleReader("testdata/test_resources_schedule.csv")
	sched, err := r.Read()
	if err != nil {
		t.Fatalf("Failed to load test_resources_schedule.csv: %v", err)
	}

	// Verify tasks with various resource assignments
	tests := []struct {
		taskID     string
		wantResources string
		desc       string
	}{
		{"3", "Alice; Bob", "Task with multiple resources"},
		{"22", "Frank", "Task with single resource"},
		{"23", "Frank; Grace; Henry", "Task with three resources"},
		{"21", "", "Task with no resources"},
		{"26", "Irene", "Task incomplete with resources"},
	}

	for _, tt := range tests {
		task := findTaskByTaskID(sched, tt.taskID)
		if task == nil {
			t.Errorf("Task %s not found in schedule", tt.taskID)
			continue
		}
		if task.Resources != tt.wantResources {
			t.Errorf("Task %s (%s): Resources = %q, want %q", 
				tt.taskID, tt.desc, task.Resources, tt.wantResources)
		}
	}
}

func TestResources_IntegrationWithFixture(t *testing.T) {
	// Integration test: load test_resources_schedule.csv and verify Resources field is populated
	r := NewScheduleReader("testdata/test_resources_schedule.csv")
	sched, err := r.Read()
	if err != nil {
		t.Fatalf("Failed to load test_resources_schedule.csv: %v", err)
	}

	// Verify tasks with resources
	resourceCounts := 0
	noResourceCounts := 0
	for _, task := range sched.Tasks {
		if task.Active && !task.IsSummary && !task.IsMilestone {
			if task.Resources != "" {
				resourceCounts++
			} else {
				noResourceCounts++
			}
		}
	}

	if resourceCounts == 0 {
		t.Error("Expected some tasks with resources assigned")
	}
	if noResourceCounts == 0 {
		t.Error("Expected some tasks without resources assigned")
	}

	// Verify specific tasks from fixture
	// Task 3 should have "Alice; Bob"
	task3 := findTaskByTaskID(sched, "3")
	if task3 == nil {
		t.Fatal("Task 3 not found")
	}
	if task3.Resources != "Alice; Bob" {
		t.Errorf("Task 3 Resources = %q, want 'Alice; Bob'", task3.Resources)
	}

	// Task 21 should have no resources
	task21 := findTaskByTaskID(sched, "21")
	if task21 == nil {
		t.Fatal("Task 21 not found")
	}
	if task21.Resources != "" {
		t.Errorf("Task 21 Resources = %q, want '' (empty)", task21.Resources)
	}
}

func findTaskByTaskID(sched *model.Schedule, taskID string) *model.Task {
	for _, t := range sched.Tasks {
		if t.TaskID == taskID {
			return t
		}
	}
	return nil
}
