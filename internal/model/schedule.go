package model

import (
	"fmt"
	"time"
)

// Task represents a single schedule task.
type Task struct {
	// TaskID is the visible sequential ID from the "Task ID" / "ID" column —
	// the number the user sees in the left-hand column of MS Project.
	TaskID string
	// UniqueID is the permanent MS Project "Unique ID" — stable across insertions
	// and deletions. Populated only when a "Unique ID" column is present alongside
	// a "Task ID" column; otherwise empty.
	UniqueID             string
	Name                 string
	Duration             float64
	Start                *time.Time
	Finish               *time.Time
	Predecessors         string
	PercentComplete      float64
	Resources            string
	WBS                  string
	ConstraintType       string
	ConstraintDate       *time.Time
	IsMilestone          bool
	IsSummary            bool
	// Active indicates whether the task is enabled in the schedule.
	// Tasks with Active == false (sourced from an "Active = No" column) are
	// disabled/inactive and must be excluded from all metric assessments.
	Active               bool
	TotalSlack           float64
	FreeSlack            float64
	OutlineLevel         int
	Discipline           string
	MechanicalSegmentNbr string
	ControlSegmentNbr    string
	BaselineDuration     float64
	ActualStart          *time.Time
	ActualFinish         *time.Time
	BaselineStart        *time.Time
	BaselineFinish       *time.Time
}

// String returns a string representation of the Task.
func (t Task) String() string {
	start := "nil"
	if t.Start != nil {
		start = t.Start.Format("2006-01-02")
	}
	finish := "nil"
	if t.Finish != nil {
		finish = t.Finish.Format("2006-01-02")
	}
	return fmt.Sprintf("Task(id=%s, name='%s', start=%s, finish=%s)", t.TaskID, t.Name, start, finish)
}

// Schedule represents a complete project schedule.
type Schedule struct {
	Name          string
	DataDate      time.Time
	ProjectStart  time.Time
	ProjectFinish time.Time
	Tasks         []*Task
	// InactiveTasks holds tasks that were marked Active=No in the source
	// export. They are intentionally excluded from every metric so the
	// assessment honours the "active network only" convention, but the reader
	// still parses them so diagnostics (e.g. the Logic metric's enriched
	// condition strings) can explain when a flagged task's only referrer is
	// an inactive task.
	InactiveTasks []*Task
	// Warnings collects non-fatal issues (e.g., duplicate column mappings)
	// encountered during file parsing.
	Warnings []string
}

// String returns a string representation of the Schedule.
func (s Schedule) String() string {
	return fmt.Sprintf("Schedule(name='%s', tasks=%d, data_date=%s)", s.Name, len(s.Tasks), s.DataDate.Format("2006-01-02"))
}
