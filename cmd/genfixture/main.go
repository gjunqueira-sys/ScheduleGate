// Command genfixture writes a small, valid MS Project-style schedule export
// (.xlsx) to the given path. It exists so CI can build and smoke-test the real
// binary end-to-end (reader → engine → report) on every target OS without
// committing binary fixtures.
package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

var headers = []string{
	"Task ID", "Unique ID", "Task Name", "Duration", "Start", "Finish",
	"Predecessors", "% Complete", "Total Slack", "Baseline Start", "Baseline Finish",
	"Active", "Resource Names",
}

var rows = [][]string{
	{"1", "1", "Design", "10d", "2026-01-05", "2026-01-16", "", "100", "0d", "2026-01-05", "2026-01-16", "Yes", "Alice"},
	{"2", "2", "Build", "20d", "2026-01-19", "2026-02-13", "1", "50", "5d", "2026-01-19", "2026-02-13", "Yes", "Alice; Bob"},
	{"3", "3", "Test", "10d", "2026-02-16", "2026-02-27", "2", "0", "2d", "2026-02-16", "2026-02-27", "Yes", ""},
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genfixture <output.xlsx>")
		os.Exit(2)
	}
	path := os.Args[1]

	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}

	if err := f.SaveAs(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
}
