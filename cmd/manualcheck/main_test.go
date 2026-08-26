package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdroot "github.com/gjunqueira-sys/ScheduleGate/cmd"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/spf13/pflag"
)

func buildMinimalFixture() string {
	root := cmdroot.RootCmd()
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html><html><body>")

	for _, c := range root.Commands() {
		if c.Hidden {
			continue
		}
		useName := strings.Fields(c.Use)[0]
		sb.WriteString("<code>" + useName + "</code>")
		sb.WriteString("schedulegate " + useName)

		c.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			sb.WriteString("--" + f.Name + " ")
		})

		for _, sub := range c.Commands() {
			if sub.Hidden {
				continue
			}
			subName := strings.Fields(sub.Use)[0]
			sb.WriteString("schedulegate " + useName + " " + subName)
		}
	}

	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		sb.WriteString("--" + f.Name + " ")
	})

	for _, m := range metricsUnderTest() {
		id := m.ID()
		sb.WriteString(fmt.Sprintf(`<section id="metric-%d">`, id))
		sb.WriteString("<strong>" + m.Name() + "</strong>")
		sb.WriteString(thresholdString(id, m))
		sb.WriteString("</section>")
	}

	for _, col := range reader.RequiredColumns {
		sb.WriteString(fmt.Sprintf("<td><code>%s</code></td>", col))
	}

	tiers := []string{"Community", "Pro", "Team", "Enterprise", "Lifetime"}
	for _, tier := range tiers {
		sb.WriteString(tier + " ")
	}

	sb.WriteString("schedulegate v1.0.4")
	sb.WriteString("schedulegate v1.0.4")
	sb.WriteString("</body></html>")

	return sb.String()
}

func TestCheckCompleteFixture(t *testing.T) {
	fixture := buildMinimalFixture()
	problems, stats := Check(fixture, "v1.0.4")

	if len(problems) > 0 {
		t.Errorf("Expected no problems, got: %v", problems)
	}
	if stats.Metrics != 14 {
		t.Errorf("Expected 14 metrics, got %d", stats.Metrics)
	}
	if stats.Columns != len(reader.RequiredColumns) {
		t.Errorf("Expected %d columns, got %d", len(reader.RequiredColumns), stats.Columns)
	}
}

func TestCheckDetectsMissingFlag(t *testing.T) {
	fixture := buildMinimalFixture()
	fixture = strings.ReplaceAll(fixture, "--html", "")
	problems, _ := Check(fixture, "v1.0.4")

	found := false
	for _, p := range problems {
		if strings.Contains(p, "--html") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected missing --html to be detected, got: %v", problems)
	}
}

func TestCheckDetectsMissingMetric(t *testing.T) {
	fixture := buildMinimalFixture()
	fixture = strings.Replace(fixture, `id="metric-1"`, "", 1)
	problems, _ := Check(fixture, "v1.0.4")

	found := false
	for _, p := range problems {
		if strings.Contains(p, "Metric 1") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected missing metric 1 to be detected, got: %v", problems)
	}
}

func TestCheckDetectsVersionMismatch(t *testing.T) {
	fixture := buildMinimalFixture()
	problems, _ := Check(fixture, "v1.0.5")

	found := false
	for _, p := range problems {
		if strings.Contains(p, "Version badge") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected version mismatch to be detected, got: %v", problems)
	}
}

func TestRealManualUpToDate(t *testing.T) {
	manualPath := filepath.Join("..", "..", "docs", "user-manual.html")
	content, err := os.ReadFile(manualPath)
	if err != nil {
		t.Skipf("Manual not found at %s: %v", manualPath, err)
	}

	problems, stats := Check(string(content), "")

	if len(problems) > 0 {
		t.Errorf("Real manual is out of date:\n")
		for _, p := range problems {
			t.Errorf("  - %s\n", p)
		}
	}

	if stats.Metrics != 14 {
		t.Errorf("Expected 14 metrics checked, got %d", stats.Metrics)
	}
	if stats.Columns != len(reader.RequiredColumns) {
		t.Errorf("Expected %d columns checked, got %d", len(reader.RequiredColumns), stats.Columns)
	}
}
