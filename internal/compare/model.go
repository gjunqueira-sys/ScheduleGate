package compare

import (
	"fmt"
	"time"
)

// TaskStatus represents the change state of a task.
type TaskStatus string

const (
	StatusNew       TaskStatus = "New"
	StatusDeleted   TaskStatus = "Deleted"
	StatusModified  TaskStatus = "Modified"
	StatusUnchanged TaskStatus = "Unchanged"
)

// TaskDelta represents the difference between two versions of a task.
type TaskDelta struct {
	TaskID         string
	Name           string
	WBS            string
	Status         TaskStatus
	FinishVariance float64 // Current - Previous (days)
	DurationDelta  float64 // Current - Previous (days)
	PrevDuration   float64 // Previous duration (days), for growth % calculation
	ExecutionDelta float64 // Current % - Previous %
	IsMilestone    bool
	IsGhostTask    bool // Started in past but 0% complete
	PrevStart      *time.Time
	CurrStart      *time.Time

	// Detailed Reporting Fields
	Symbol     string
	ImpactType string // "Stability", "Reliability", "Scope", "Neutral"
	ImpactMsg  string
}

// FrictionItem represents a WBS element with high friction.
type FrictionItem struct {
	WBS            string
	GhostTaskCount int
}

// BenchmarkResult holds the results of the schedule comparison.
type BenchmarkResult struct {
	// Scores (0-100)
	OverallScore float64
	PillarAScore float64 // Schedule Stability
	PillarBScore float64 // Duration Reliability
	PillarCScore float64 // Scope Churn

	// Raw Data for Reporting
	TotalTasks            int
	NewTasks              int
	DeletedTasks          int
	ModifiedTasks         int
	UnchangedTasks        int
	GhostTasksCount       int
	DurationInflatedCount int     // Count of tasks with duration growth > 10%
	DurationInflatedPct   float64 // Percentage of tasks with duration growth > 10%

	// Detailed Deltas
	TaskDeltas []*TaskDelta

	// Friction Index (Bottlenecks)
	FrictionIndex []FrictionItem

	// Warnings for non-fatal data issues (e.g. duplicate TaskIDs)
	Warnings []string
}

func (r *BenchmarkResult) String() string {
	return fmt.Sprintf("Score: %.1f (Stability: %.1f, Duration: %.1f, Scope: %.1f)",
		r.OverallScore, r.PillarAScore, r.PillarBScore, r.PillarCScore)
}
