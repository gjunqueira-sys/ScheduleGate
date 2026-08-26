package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"

	cmdroot "github.com/gjunqueira-sys/ScheduleGate/cmd"
	"github.com/gjunqueira-sys/ScheduleGate/internal/dcma"
	"github.com/gjunqueira-sys/ScheduleGate/internal/reader"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var defaultManualPath = "docs/user-manual.html"

type Stats struct {
	Commands int
	Flags    int
	Metrics  int
	Columns  int
}

func metricsUnderTest() []dcma.Metric {
	return []dcma.Metric{
		&dcma.LogicMetric{}, &dcma.LeadsMetric{}, &dcma.LagsMetric{},
		&dcma.RelationshipTypesMetric{}, &dcma.HardConstraintsMetric{},
		&dcma.HighFloatMetric{}, &dcma.NegativeFloatMetric{},
		&dcma.HighDurationMetric{}, &dcma.InvalidDatesMetric{},
		&dcma.ResourcesMetric{}, &dcma.MissedTasksMetric{},
		&dcma.CriticalPathTestMetric{}, &dcma.CPLIMetric{},
		&dcma.BEIMetric{},
	}
}

func thresholdString(id int, m dcma.Metric) string {
	if id == 12 {
		return "Pass / Fail"
	}
	if id == 13 {
		return "&ge; 1.0"
	}
	pct := int(math.Round(m.Threshold() * 100))
	return fmt.Sprintf("%d%%", pct)
}

func Check(manualHTML, expectVersion string) ([]string, Stats) {
	var problems []string
	stats := Stats{}

	htmlLower := strings.ToLower(manualHTML)

	root := cmdroot.RootCmd()
	skipFlags := map[string]bool{"help": true}

	var collectCommands func(c *cobra.Command, path string)
	collectCommands = func(c *cobra.Command, path string) {
		if c.Hidden {
			return
		}
		stats.Commands++
		useName := strings.Fields(c.Use)[0]

		if path == "" {
			if !strings.Contains(htmlLower, "<code>"+useName+"</code>") &&
				!strings.Contains(strings.ToLower(manualHTML), "schedulegate "+useName) {
				problems = append(problems, fmt.Sprintf("Command '%s' not documented", useName))
			}
		} else {
			search := path + " " + useName
			if !strings.Contains(strings.ToLower(manualHTML), search) {
				problems = append(problems, fmt.Sprintf("Command '%s' not documented", search))
			}
		}

		c.Flags().VisitAll(func(f *pflag.Flag) {
			if skipFlags[f.Name] {
				return
			}
			stats.Flags++
			token := "--" + f.Name
			if !strings.Contains(manualHTML, token) {
				problems = append(problems, fmt.Sprintf("Flag '%s' of command '%s' not documented", token, pathOrDefault(path, useName)))
			}
		})

		for _, sub := range c.Commands() {
			collectCommands(sub, pathOrDefault(path, useName))
		}
	}

	for _, c := range root.Commands() {
		collectCommands(c, "")
	}

	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if skipFlags[f.Name] {
			return
		}
		stats.Flags++
		token := "--" + f.Name
		if !strings.Contains(manualHTML, token) {
			problems = append(problems, fmt.Sprintf("Root persistent flag '%s' not documented", token))
		}
	})

	metrics := metricsUnderTest()
	stats.Metrics = len(metrics)
	for _, m := range metrics {
		id := m.ID()
		name := m.Name()
		sectionStart := strings.Index(manualHTML, fmt.Sprintf(`id="metric-%d"`, id))
		if sectionStart == -1 {
			problems = append(problems, fmt.Sprintf("Metric %d section not found", id))
			continue
		}
		sectionEnd := strings.Index(manualHTML[sectionStart+1:], `id="metric-`)
		section := manualHTML[sectionStart:]
		if sectionEnd != -1 {
			section = section[:sectionEnd]
		}

		if !strings.Contains(section, name) {
			problems = append(problems, fmt.Sprintf("Metric %d name '%s' not found in section", id, name))
		}

		threshold := thresholdString(id, m)
		if !strings.Contains(section, threshold) {
			problems = append(problems, fmt.Sprintf("Metric %d threshold '%s' not found in section", id, threshold))
		}
	}

	stats.Columns = len(reader.RequiredColumns)
	for _, col := range reader.RequiredColumns {
		pattern := fmt.Sprintf("<td><code>%s</code></td>", col)
		if !strings.Contains(manualHTML, pattern) {
			problems = append(problems, fmt.Sprintf("Required column '%s' not in table", col))
		}
	}

	tiers := []string{"Community", "Pro", "Team", "Enterprise", "Lifetime"}
	for _, tier := range tiers {
		if !strings.Contains(manualHTML, tier) {
			problems = append(problems, fmt.Sprintf("License tier '%s' not documented", tier))
		}
	}

	if expectVersion != "" {
		bare := strings.TrimPrefix(expectVersion, "v")
		token := "v" + bare
		count := strings.Count(manualHTML, token)
		if count < 2 {
			problems = append(problems, fmt.Sprintf("Version badge '%s' appears %d times (expected >= 2)", token, count))
		}
	}

	return problems, stats
}

func pathOrDefault(path, useName string) string {
	if path == "" {
		return useName
	}
	return path
}

func updateVersion(manualPath, newVersion string) error {
	content, err := os.ReadFile(manualPath)
	if err != nil {
		return err
	}
	bare := strings.TrimPrefix(newVersion, "v")
	re := regexp.MustCompile(`v\d+\.\d+\.\d+`)
	updated := re.ReplaceAllString(string(content), "v"+bare)
	return os.WriteFile(manualPath, []byte(updated), 0644)
}

func main() {
	manualPath := flag.String("manual", defaultManualPath, "path to the user manual HTML")
	updateVer := flag.String("update-version", "", "rewrite version badges in-place to this version (e.g. v1.0.4)")
	expectVer := flag.String("expect-version", "", "fail unless badges show this version (e.g. v1.0.4)")
	flag.Parse()

	if *updateVer != "" {
		if err := updateVersion(*manualPath, *updateVer); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating version: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("✓ Version badges updated to %s\n", *updateVer)
	}

	content, err := os.ReadFile(*manualPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading manual: %v\n", err)
		os.Exit(2)
	}

	problems, stats := Check(string(content), *expectVer)
	if len(problems) > 0 {
		fmt.Fprintln(os.Stderr, "Manual is out of date:")
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "  - %s\n", p)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Manual up to date (%d commands, %d flags, %d metrics, %d columns)\n",
		stats.Commands, stats.Flags, stats.Metrics, stats.Columns)
}
