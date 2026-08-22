package services

type MetricResult struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Passing     bool    `json:"passing"`
	NotApplicable bool  `json:"notApplicable"`
}

type AssessRequest struct {
	FilePath         string `json:"filePath"`
	Metrics          []int  `json:"metrics"`
	HTMLOutput       string `json:"htmlOutput"`
	CSVOutput        string `json:"csvOutput"`
	ExceptionsReport string `json:"exceptionsReport"`
	Customer         string `json:"customer"`
	Project          string `json:"project"`
	Verbose          bool   `json:"verbose"`
	StatusDate       string `json:"statusDate"`
	DebugLogic       bool   `json:"debugLogic"`
	PctFormat        string `json:"pctFormat"`
	DateLocale       string `json:"dateLocale"`
}

type AssessResponse struct {
	Success      bool           `json:"success"`
	OverallScore float64        `json:"overallScore"`
	Passed       int            `json:"passed"`
	Total        int            `json:"total"`
	ScheduleName string         `json:"scheduleName"`
	DataDate     string         `json:"dataDate"`
	TaskCount    int            `json:"taskCount"`
	Metrics      []MetricResult `json:"metrics"`
	OutputFiles  []string       `json:"outputFiles"`
	Error        string         `json:"error,omitempty"`
	Errors       []string       `json:"errors,omitempty"`
}

type CompareRequest struct {
	PreviousFile string `json:"previousFile"`
	CurrentFile  string `json:"currentFile"`
	HTMLOutput   string `json:"htmlOutput"`
	CSVOutput    string `json:"csvOutput"`
	Detailed     bool   `json:"detailed"`
	Customer     string `json:"customer"`
	Project      string `json:"project"`
	PctFormat    string `json:"pctFormat"`
	DateLocale   string `json:"dateLocale"`
}

type TaskDelta struct {
	TaskID         string  `json:"taskId"`
	Name           string  `json:"name"`
	WBS            string  `json:"wbs"`
	Status         string  `json:"status"`
	FinishVariance float64 `json:"finishVariance"`
	DurationDelta  float64 `json:"durationDelta"`
	IsGhostTask    bool    `json:"isGhostTask"`
	IsMilestone    bool    `json:"isMilestone"`
}

type CompareResponse struct {
	Success              bool        `json:"success"`
	OverallScore         float64     `json:"overallScore"`
	PillarAScore         float64     `json:"pillarAScore"`
	PillarBScore         float64     `json:"pillarBScore"`
	PillarCScore         float64     `json:"pillarCScore"`
	TotalTasks           int         `json:"totalTasks"`
	NewTasks             int         `json:"newTasks"`
	DeletedTasks         int         `json:"deletedTasks"`
	ModifiedTasks        int         `json:"modifiedTasks"`
	UnchangedTasks       int         `json:"unchangedTasks"`
	GhostTasksCount      int         `json:"ghostTasksCount"`
	DurationInflatedPct  float64     `json:"durationInflatedPct"`
	PrevScheduleName     string      `json:"prevScheduleName"`
	CurrScheduleName     string      `json:"currScheduleName"`
	TaskDeltas           []TaskDelta `json:"taskDeltas"`
	OutputFiles          []string    `json:"outputFiles"`
	Error                string      `json:"error,omitempty"`
	Errors               []string    `json:"errors,omitempty"`
}

type ValidateRequest struct {
	FilePath   string `json:"filePath"`
	HTMLOutput string `json:"htmlOutput"`
	CSVOutput  string `json:"csvOutput"`
}

type ValidateResponse struct {
	Success     bool     `json:"success"`
	Found       int      `json:"found"`
	Missing     int      `json:"missing"`
	Extra       int      `json:"extra"`
	ColumnNames []string `json:"columnNames"`
	OutputFiles []string `json:"outputFiles"`
	Errors      []string `json:"errors,omitempty"`
}

type CheckPatternsRequest struct {
	FilePath   string `json:"filePath"`
	RulesFile  string `json:"rulesFile"`
	HTMLOutput string `json:"htmlOutput"`
	CSVOutput  string `json:"csvOutput"`
	Detailed   bool   `json:"detailed"`
	PctFormat  string `json:"pctFormat"`
	DateLocale string `json:"dateLocale"`
}

type PatternResult struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Passing bool   `json:"passing"`
	Message string `json:"message"`
}

type CheckPatternsResponse struct {
	Success     bool            `json:"success"`
	Results     []PatternResult `json:"results"`
	TaskCount   int             `json:"taskCount"`
	OutputFiles []string        `json:"outputFiles"`
	Errors      []string        `json:"errors,omitempty"`
}
