package report

// AssessJSONOutput is the machine-readable form of a DCMA assessment,
// designed for CI/CD consumption. All values mirror the terminal/HTML views.
type AssessJSONOutput struct {
	ScheduleName string             `json:"scheduleName"`
	StatusDate   string             `json:"statusDate"`
	GeneratedAt  string             `json:"generatedAt"`
	ToolVersion  string             `json:"toolVersion"`
	OverallScore float64            `json:"overallScore"`
	PassedCount  int                `json:"passedCount"`
	TotalCount   int                `json:"totalCount"`
	Results      []MetricJSONResult `json:"results"`
	Population   SchedulePopulation `json:"population"`
	Warnings     []string           `json:"warnings,omitempty"`
}

// MetricJSONResult is the per-metric view inside an assess JSON payload.
type MetricJSONResult struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Value          float64 `json:"value"`
	Threshold      float64 `json:"threshold"`
	Passing        bool    `json:"passing"`
	NotApplicable  bool    `json:"notApplicable"`
	ExceptionCount int     `json:"exceptionCount"`
}

// SchedulePopulation is the JSON-tagged mirror of dcma.SchedulePopulation.
type SchedulePopulation struct {
	TotalRows                          int `json:"totalRows"`
	SummaryRows                        int `json:"summaryRows"`
	Milestones                         int `json:"milestones"`
	WorkTasks                          int `json:"workTasks"`
	CompletedWorkTasks                 int `json:"completedWorkTasks"`
	IncompleteWorkTasks                int `json:"incompleteWorkTasks"`
	WorkTasksWithBaselineFinish        int `json:"workTasksWithBaselineFinish"`
	WorkTasksBaselineDueByStatus       int `json:"workTasksBaselineDueByStatus"`
	AssessableTasks                    int `json:"assessableTasks"`
	CompletedAssessableTasks           int `json:"completedAssessableTasks"`
	AssessableTasksWithBaselineFinish  int `json:"assessableTasksWithBaselineFinish"`
	AssessableTasksBaselineDueByStatus int `json:"assessableTasksBaselineDueByStatus"`
}

// CompareJSONOutput is the machine-readable form of a two-version benchmark.
type CompareJSONOutput struct {
	PreviousFile          string             `json:"previousFile"`
	CurrentFile           string             `json:"currentFile"`
	StatusDate            string             `json:"statusDate"`
	GeneratedAt           string             `json:"generatedAt"`
	ToolVersion           string             `json:"toolVersion"`
	OverallScore          float64            `json:"overallScore"`
	StabilityScore        float64            `json:"stabilityScore"`
	ReliabilityScore      float64            `json:"reliabilityScore"`
	ScopeChurnScore       float64            `json:"scopeChurnScore"`
	TotalTasks            int                `json:"totalTasks"`
	NewTasks              int                `json:"newTasks"`
	DeletedTasks          int                `json:"deletedTasks"`
	ModifiedTasks         int                `json:"modifiedTasks"`
	UnchangedTasks        int                `json:"unchangedTasks"`
	GhostTasksCount       int                `json:"ghostTasksCount"`
	DurationInflatedCount int                `json:"durationInflatedCount"`
	DurationInflatedPct   float64            `json:"durationInflatedPct"`
	TaskDeltas            []TaskDeltaJSON    `json:"taskDeltas,omitempty"`
	FrictionIndex         []FrictionItemJSON `json:"frictionIndex,omitempty"`
	Warnings              []string           `json:"warnings,omitempty"`
}

// TaskDeltaJSON is the flat, JSON-safe subset of compare.TaskDelta.
type TaskDeltaJSON struct {
	TaskID         string  `json:"taskId"`
	Name           string  `json:"name"`
	WBS            string  `json:"wbs"`
	Status         string  `json:"status"`
	FinishVariance float64 `json:"finishVariance"`
	DurationDelta  float64 `json:"durationDelta"`
	PrevDuration   float64 `json:"prevDuration"`
	ExecutionDelta float64 `json:"executionDelta"`
	IsGhostTask    bool    `json:"isGhostTask"`
	IsMilestone    bool    `json:"isMilestone"`
	ImpactType     string  `json:"impactType"`
	ImpactMsg      string  `json:"impactMsg"`
}

// FrictionItemJSON is the JSON mirror of compare.FrictionItem.
type FrictionItemJSON struct {
	WBS            string `json:"wbs"`
	GhostTaskCount int    `json:"ghostTaskCount"`
}

// ValidateJSONOutput is the machine-readable form of a column validation run.
type ValidateJSONOutput struct {
	SourceFile      string            `json:"sourceFile"`
	GeneratedAt     string            `json:"generatedAt"`
	ToolVersion     string            `json:"toolVersion"`
	Status          string            `json:"status"` // "READY" | "INCOMPLETE"
	RequiredFound   map[string]string `json:"requiredFound"`
	RequiredMissing []string          `json:"requiredMissing"`
	OptionalFound   map[string]string `json:"optionalFound"`
	ExtraColumns    []string          `json:"extraColumns"`
	Warnings        []string          `json:"warnings,omitempty"`
}

// PatternsJSONOutput is the machine-readable form of a pattern compliance run.
type PatternsJSONOutput struct {
	ScheduleFile string              `json:"scheduleFile"`
	RulesFile    string              `json:"rulesFile"`
	GeneratedAt  string              `json:"generatedAt"`
	ToolVersion  string              `json:"toolVersion"`
	Status       string              `json:"status"` // "COMPLIANT" | "NON-COMPLIANT"
	PassedCount  int                 `json:"passedCount"`
	TotalCount   int                 `json:"totalCount"`
	Results      []PatternJSONResult `json:"results"`
	Warnings     []string            `json:"warnings,omitempty"`
}

// PatternJSONResult is the per-rule view inside a patterns JSON payload.
type PatternJSONResult struct {
	RuleName        string            `json:"ruleName"`
	Match           map[string]string `json:"match"`
	MinCount        int               `json:"minCount"`
	MaxCount        int               `json:"maxCount"`
	MatchingCount   int               `json:"matchingCount"`
	Passing         bool              `json:"passing"`
	Message         string            `json:"message"`
	MatchingTaskIDs []string          `json:"matchingTaskIds,omitempty"`
}
