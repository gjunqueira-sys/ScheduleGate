package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/version"
)

// AppendToCSV appends the assessment results to a CSV file.
// If the file does not exist, it creates it and writes the header.
// The CSV is structured as a flat database where each row is an assessment.
func AppendToCSV(assessment *dcma.DCMAAssessment, scheduleName, customer, project, statusDate, outputPath string) (returnErr error) {
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

	// Get specific list of all metrics to ensure consistent column ordering
	// We use a temporary assessment to get the canonical list of metrics
	allMetrics := dcma.NewDCMAAssessment(nil).Metrics

	if !fileExists {
		// Write Header
		header := []string{
			"Customer",
			"Project",
			"Schedule Name",
			"Status Date",
			"Report Date",
			"Tool Version",
			"Overall Score",
		}
		for _, m := range allMetrics {
			header = append(header, m.Name())
		}
		if err := w.Write(header); err != nil {
			return fmt.Errorf("failed to write csv header: %w", err)
		}
	}

	// Calculate scores — exclude N/A metrics from tally (matches CLI/HTML/Excel)
	passedCount := 0
	totalRun := 0
	for _, res := range assessment.Results {
		if res.NotApplicable {
			continue
		}
		totalRun++
		if res.Passing {
			passedCount++
		}
	}
	
	// Avoid divide by zero if no metrics run, though unlikely in practice
	overallScore := 0.0
	if totalRun > 0 {
		overallScore = float64(passedCount) / float64(totalRun)
	}

	// Build Row
	row := []string{
		customer,
		project,
		scheduleName,
		statusDate,
		time.Now().Format("2006-01-02 15:04:05"),
		version.Display(),
		fmt.Sprintf("%.2f%%", overallScore*100),
	}

	for _, m := range allMetrics {
		if res, ok := assessment.Results[m.Name()]; ok {
			row = append(row, fmt.Sprintf("%.2f%%", res.Value*100))
		} else {
			// Metric was not run in this assessment
			row = append(row, "")
		}
	}

	if err := w.Write(row); err != nil {
		return fmt.Errorf("failed to write csv row: %w", err)
	}

	return nil
}
