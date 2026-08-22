package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/report"
	"github.com/spf13/cobra"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// sanitizeError strips ANSI styling and PASS/FAIL badge words from a message
// so JSON error payloads stay clean and machine-readable.
func sanitizeError(msg string) string {
	clean := ansiRe.ReplaceAllString(msg, "")
	clean = strings.TrimSpace(clean)
	clean = strings.TrimPrefix(clean, "FAIL")
	clean = strings.TrimPrefix(clean, "PASS")
	return strings.TrimSpace(clean)
}

// jsonEnabled reports whether --json (stdout JSON) was requested.
func jsonEnabled(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// jsonOutputPath returns the --json-output file path, or "" if not set.
func jsonOutputPath(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("json-output")
	return v
}

// writeJSONOutput emits v as JSON to --json-output (if set) and, when --json
// is also present, to stdout. Used by every command's --json handling.
func writeJSONOutput(cmd *cobra.Command, v interface{}) error {
	if outPath := jsonOutputPath(cmd); outPath != "" {
		if err := report.WriteJSONFile(outPath, v); err != nil {
			return err
		}
	}
	if jsonEnabled(cmd) {
		return report.WriteJSON(os.Stdout, v)
	}
	return nil
}

// jsonErrorPayload writes a {"error": msg} payload honoring the active output
// mode: the --json-output file when set (without --json), otherwise stdout.
func jsonErrorPayload(cmd *cobra.Command, msg string) error {
	payload := map[string]string{"error": sanitizeError(msg)}
	if outPath := jsonOutputPath(cmd); outPath != "" && !jsonEnabled(cmd) {
		return report.WriteJSONFile(outPath, payload)
	}
	return report.WriteJSON(os.Stdout, payload)
}

// fail prints an error (as JSON when --json is active) and exits 1.
func fail(cmd *cobra.Command, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if jsonEnabled(cmd) {
		if err := jsonErrorPayload(cmd, msg); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	} else {
		fmt.Println(msg)
	}
	os.Exit(1)
}

// autoOpen suppresses file auto-opening in JSON mode, which is inappropriate
// for headless CI/CD runs.
func autoOpen(cmd *cobra.Command, path string) {
	if jsonEnabled(cmd) {
		return
	}
	_ = report.OpenFile(path)
}
