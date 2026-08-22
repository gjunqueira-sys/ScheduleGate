package services

import (
	"fmt"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ValidateService struct{}

func (s *ValidateService) emit(app *application.App, data string) {
	app.Event.Emit("term-output", data)
}

func (s *ValidateService) Run(p ValidateRequest) *ValidateResponse {
	app := application.Get()

	s.emit(app, fmt.Sprintf("$ schedulegate validate %s\n\n", p.FilePath))

	app.Event.Emit("term-loading", true)
	defer app.Event.Emit("term-loading", false)

	s.emit(app, fmt.Sprintf("Reading: %s\n", p.FilePath))
	s.emit(app, "Validating column schema...\n\n")

	result, err := reader.ValidateColumns(p.FilePath)
	if err != nil {
		s.emit(app, fmt.Sprintf("\x1b[31mError: %v\x1b[0m\n", err))
		return &ValidateResponse{}
	}

	// Stop the loading spinner before writing results so its periodic
	// carriage-return redraw can't overwrite output lines.
	app.Event.Emit("term-loading", false)

	var found []string
	for _, canonical := range reader.RequiredColumns {
		if orig, ok := result.Found[canonical]; ok {
			found = append(found, orig)
		}
	}
	for _, canonical := range reader.OptionalColumns {
		if orig, ok := result.Found[canonical]; ok {
			found = append(found, orig)
		}
	}

	var sb strings.Builder
	// Clear any residual spinner artifact left on the current line.
	sb.WriteString("\r\x1b[K")

	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			fmt.Fprintf(&sb, "  \x1b[33mWarning: %s\x1b[0m\n", w)
		}
	}

	fmt.Fprintf(&sb, "  \x1b[32mFound:   %d columns\x1b[0m\n", len(result.Found))
	for _, col := range found {
		fmt.Fprintf(&sb, "    \x1b[32m✓\x1b[0m %s\n", col)
	}

	if len(result.Missing) > 0 {
		fmt.Fprintf(&sb, "\n  \x1b[31mMissing: %d required columns\x1b[0m\n", len(result.Missing))
		for _, col := range result.Missing {
			fmt.Fprintf(&sb, "    \x1b[31m✗\x1b[0m %s\n", col)
		}
	}

	if len(result.Extra) > 0 {
		fmt.Fprintf(&sb, "\n  \x1b[90mExtra:   %d unrecognized columns\x1b[0m\n", len(result.Extra))
		for _, col := range result.Extra {
			fmt.Fprintf(&sb, "    - %s\n", col)
		}
	}

	passing := len(result.Missing) == 0
	if passing {
		sb.WriteString("\n\x1b[1m\x1b[32mVALIDATION: PASSED\x1b[0m\n")
	} else {
		sb.WriteString("\n\x1b[1m\x1b[31mVALIDATION: FAILED\x1b[0m\n")
	}

	s.emit(app, sb.String())

	return &ValidateResponse{
		Success: true,
		Found:   len(result.Found),
		Missing: len(result.Missing),
		Extra:   len(result.Extra),
	}
}
