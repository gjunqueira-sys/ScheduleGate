package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/compare"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
)

// AppendCompareToCSV appends one summary row for a schedule comparison to a CSV file.
//
// Signature: AppendCompareToCSV(result *compare.BenchmarkResult, prevName, currName, customer, project, outputPath string) error
//
// Arguments:
//   - result: comparison outcome with scores and change counts
//   - prevName: previous schedule name (basename without extension)
//   - currName: current schedule name (basename without extension)
//   - customer: customer label for multi-project history (may be empty)
//   - project: project number/ID for multi-project history (may be empty)
//   - outputPath: path to the CSV database file
//
// Returns: nil on success, or an error if the file cannot be opened or written.
// If the file does not exist, it is created with a header row first.
// Each row represents one comparison cycle for trend/history analysis.
func AppendCompareToCSV(result *compare.BenchmarkResult, prevName, currName, customer, project, statusDate, outputPath string) (returnErr error) {
	fileExists := false
	if info, err := os.Stat(outputPath); err == nil && info.Size() > 0 {
		fileExists = true
	}

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open csv file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer func() {
		w.Flush()
		if returnErr == nil {
			if err := w.Error(); err != nil {
				returnErr = fmt.Errorf("csv flush error: %w", err)
			}
		}
	}()

	if !fileExists {
		header := []string{
			"Customer",
			"Project",
			"Previous Schedule",
			"Current Schedule",
			"Status Date",
			"Report Date",
			"Tool Version",
			"Overall Score",
			"Stability Score",
			"Reliability Score",
			"Scope Churn Score",
			"Total Tasks",
			"New Tasks",
			"Deleted Tasks",
			"Modified Tasks",
			"Unchanged Tasks",
			"Ghost Tasks",
			"Duration Inflated Count",
			"Duration Inflated Pct",
			"Churn Pct",
		}
		if err := w.Write(header); err != nil {
			return fmt.Errorf("failed to write csv header: %w", err)
		}
	}

	churnPct := 0.0
	if result.TotalTasks > 0 {
		churnPct = float64(result.NewTasks+result.DeletedTasks) / float64(result.TotalTasks) * 100
	}

	row := []string{
		customer,
		project,
		prevName,
		currName,
		statusDate,
		time.Now().Format("2006-01-02 15:04:05"),
		version.Display(),
		fmt.Sprintf("%.2f", result.OverallScore),
		fmt.Sprintf("%.2f", result.PillarAScore),
		fmt.Sprintf("%.2f", result.PillarBScore),
		fmt.Sprintf("%.2f", result.PillarCScore),
		fmt.Sprintf("%d", result.TotalTasks),
		fmt.Sprintf("%d", result.NewTasks),
		fmt.Sprintf("%d", result.DeletedTasks),
		fmt.Sprintf("%d", result.ModifiedTasks),
		fmt.Sprintf("%d", result.UnchangedTasks),
		fmt.Sprintf("%d", result.GhostTasksCount),
		fmt.Sprintf("%d", result.DurationInflatedCount),
		fmt.Sprintf("%.2f", result.DurationInflatedPct),
		fmt.Sprintf("%.2f", churnPct),
	}

	if err := w.Write(row); err != nil {
		return fmt.Errorf("failed to write csv row: %w", err)
	}

	return nil
}
