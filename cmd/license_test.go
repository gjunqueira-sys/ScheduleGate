package cmd

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gjunqueira-sys/ScheduleGate/internal/license"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(out)
}

func TestLicenseGenerateKeyOnly(t *testing.T) {
	oldSecret := license.SecretKey
	license.SecretKey = "test-secret-key-only"
	t.Cleanup(func() { license.SecretKey = oldSecret })

	// Point viper at an empty config so a real ~/.schedulegate.yaml is ignored.
	_ = rootCmd.PersistentFlags().Set("config", t.TempDir()+"/nonexistent.yaml")

	rootCmd.SetArgs([]string{
		"license", "generate",
		"--tier", "pro",
		"--email", "ci@schedulegate.dev",
		"--expiry", "2099-01-01",
		"--key-only",
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	out := captureStdout(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	key := strings.TrimSpace(out)
	if !regexp.MustCompile(`^SG-[A-Za-z0-9_-]+$`).MatchString(key) {
		t.Fatalf("expected a single bare SG- key on stdout, got %q", out)
	}
}
