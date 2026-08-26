package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type target struct {
	RelPath     string
	Name        string
	MinMatches  int
	ExtractRe   *regexp.Regexp
	ReplaceRe   *regexp.Regexp
	Replacement func(bare, monthYear string) string
}

func targets() []target {
	return []target{
		{
			RelPath:    "Makefile",
			Name:       "Makefile VERSION",
			MinMatches: 1,
			ExtractRe:  regexp.MustCompile(`(?m)^VERSION \?= (\d+\.\d+\.\d+)\r?$`),
			ReplaceRe:  regexp.MustCompile(`(?m)^VERSION \?= \d+\.\d+\.\d+\r?$`),
			Replacement: func(bare, _ string) string {
				return "VERSION ?= " + bare
			},
		},
		{
			RelPath:    "README.md",
			Name:       "README public release",
			MinMatches: 1,
			ExtractRe:  regexp.MustCompile(`\*\*Public release:\*\* v(\d+\.\d+\.\d+)`),
			ReplaceRe:  regexp.MustCompile(`\*\*Public release:\*\* v\d+\.\d+\.\d+ — [A-Za-z]+ \d{4}`),
			Replacement: func(bare, monthYear string) string {
				return "**Public release:** v" + bare + " — " + monthYear
			},
		},
		{
			RelPath:    "web/index.html",
			Name:       "website download URL",
			MinMatches: 1,
			ExtractRe:  regexp.MustCompile(`schedulegate-v(\d+\.\d+\.\d+)\.zip`),
			ReplaceRe:  regexp.MustCompile(`schedulegate-v\d+\.\d+\.\d+\.zip`),
			Replacement: func(bare, _ string) string {
				return "schedulegate-v" + bare + ".zip"
			},
		},
		{
			RelPath:    "internal/version/version.go",
			Name:       "version.go default",
			MinMatches: 1,
			ExtractRe:  regexp.MustCompile(`var Version = "(\d+\.\d+\.\d+)"`),
			ReplaceRe:  regexp.MustCompile(`var Version = "\d+\.\d+\.\d+"`),
			Replacement: func(bare, _ string) string {
				return `var Version = "` + bare + `"`
			},
		},
		{
			RelPath:    "desktop/build/config.yml",
			Name:       "desktop info.version",
			MinMatches: 1,
			ExtractRe:  regexp.MustCompile(`(?m)^  version: "(\d+\.\d+\.\d+)"\r?$`),
			ReplaceRe:  regexp.MustCompile(`(?m)^  version: "\d+\.\d+\.\d+"\r?$`),
			Replacement: func(bare, _ string) string {
				return `  version: "` + bare + `"`
			},
		},
		{
			RelPath:    "docs/user-manual.html",
			Name:       "user-manual badges",
			MinMatches: 2,
			ExtractRe:  regexp.MustCompile(`schedulegate v(\d+\.\d+\.\d+)`),
			ReplaceRe:  regexp.MustCompile(`schedulegate v\d+\.\d+\.\d+`),
			Replacement: func(bare, _ string) string {
				return "schedulegate v" + bare
			},
		},
		{
			RelPath:    "docs/assess-manual.html",
			Name:       "assess-manual badges",
			MinMatches: 2,
			ExtractRe:  regexp.MustCompile(`schedulegate v(\d+\.\d+\.\d+)`),
			ReplaceRe:  regexp.MustCompile(`schedulegate v\d+\.\d+\.\d+`),
			Replacement: func(bare, _ string) string {
				return "schedulegate v" + bare
			},
		},
	}
}

func normalize(version string) (string, error) {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(v, "v")
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(v) {
		return "", fmt.Errorf("invalid version %q (expected X.Y.Z or vX.Y.Z)", version)
	}
	return v, nil
}

func monthYear() string {
	return time.Now().Format("January 2006")
}

func readFile(root, rel string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n"), nil
}

func extract(t target, content string) []string {
	matches := t.ExtractRe.FindAllStringSubmatch(content, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

type fileStatus struct {
	Target   target
	Versions []string
	Err      error
}

func inspect(root string) []fileStatus {
	var out []fileStatus
	for _, t := range targets() {
		st := fileStatus{Target: t}
		content, err := readFile(root, t.RelPath)
		if err != nil {
			st.Err = err
			out = append(out, st)
			continue
		}
		st.Versions = extract(t, content)
		out = append(out, st)
	}
	return out
}

func check(root, version string) []string {
	bare, err := normalize(version)
	if err != nil {
		return []string{err.Error()}
	}
	var problems []string
	for _, st := range inspect(root) {
		name := st.Target.Name
		if st.Err != nil {
			problems = append(problems, fmt.Sprintf("%s (%s): %v", name, st.Target.RelPath, st.Err))
			continue
		}
		if len(st.Versions) < st.Target.MinMatches {
			problems = append(problems, fmt.Sprintf("%s (%s): found %d version(s), expected >= %d",
				name, st.Target.RelPath, len(st.Versions), st.Target.MinMatches))
			continue
		}
		for _, v := range st.Versions {
			if v != bare {
				problems = append(problems, fmt.Sprintf("%s (%s): has %s, expected %s",
					name, st.Target.RelPath, v, bare))
				break
			}
		}
	}
	return problems
}

func apply(root, version string) error {
	bare, err := normalize(version)
	if err != nil {
		return err
	}
	my := monthYear()
	for _, t := range targets() {
		path := filepath.Join(root, t.RelPath)
		content, err := readFile(root, t.RelPath)
		if err != nil {
			return fmt.Errorf("%s: %w", t.RelPath, err)
		}
		found := t.ReplaceRe.FindAllStringIndex(content, -1)
		if len(found) < t.MinMatches {
			return fmt.Errorf("%s: pattern matched %d time(s), expected >= %d", t.RelPath, len(found), t.MinMatches)
		}
		updated := t.ReplaceRe.ReplaceAllString(content, t.Replacement(bare, my))
		if updated == content {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
			return fmt.Errorf("writing %s: %w", t.RelPath, err)
		}
		fmt.Printf("✓ %s → v%s\n", t.RelPath, bare)
	}
	return nil
}

func printList(root string) {
	fmt.Printf("%-28s %-32s %s\n", "FILE", "ROLE", "VERSION")
	for _, st := range inspect(root) {
		ver := "(unreadable)"
		if st.Err != nil {
			ver = st.Err.Error()
		} else if len(st.Versions) == 0 {
			ver = "(none found)"
		} else {
			seen := map[string]bool{}
			var uniq []string
			for _, v := range st.Versions {
				if !seen[v] {
					seen[v] = true
					uniq = append(uniq, "v"+v)
				}
			}
			ver = strings.Join(uniq, ", ")
			if len(st.Versions) > 1 {
				ver += fmt.Sprintf(" (%dx)", len(st.Versions))
			}
		}
		fmt.Printf("%-28s %-32s %s\n", st.Target.RelPath, st.Target.Name, ver)
	}
}

func main() {
	applyVer := flag.String("apply", "", "rewrite all version-bearing files to this version (e.g. v1.0.5)")
	checkVer := flag.String("check", "", "fail unless every version-bearing file shows this version")
	list := flag.Bool("list", false, "print the version currently recorded in each file")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if *applyVer == "" && *checkVer == "" && !*list {
		*list = true
	}

	if *list && *applyVer == "" && *checkVer == "" {
		printList(absRoot)
		return
	}

	if *applyVer != "" {
		if err := apply(absRoot, *applyVer); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying version: %v\n", err)
			os.Exit(2)
		}
	}

	want := *checkVer
	if want == "" {
		want = *applyVer
	}
	if want != "" {
		problems := check(absRoot, want)
		if len(problems) > 0 {
			fmt.Fprintln(os.Stderr, "Version drift detected:")
			for _, p := range problems {
				fmt.Fprintf(os.Stderr, "  - %s\n", p)
			}
			os.Exit(1)
		}
		fmt.Printf("✓ All version-bearing files match %s\n", want)
	}
}
