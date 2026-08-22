package report

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenFile opens the specified file path using the default application
// determined by the operating system.
func OpenFile(path string) error {
	var cmd *exec.Cmd
	var args []string

	switch runtime.GOOS {
	case "darwin":
		args = []string{"open", path}
	case "windows":
		args = []string{"cmd", "/c", "start", "", path}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		args = []string{"xdg-open", path}
	}

	cmd = exec.Command(args[0], args[1:]...)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	return nil
}
