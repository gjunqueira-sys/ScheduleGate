package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func fixtureTree() map[string]string {
	return map[string]string{
		"Makefile":                    "VERSION ?= 1.0.2\nCOMMIT := abc\n",
		"README.md":                   "hello\n**Public release:** v1.0.2 — January 2026\n",
		"web/index.html":              `curl .../schedulegate-v1.0.2.zip -o x.zip`,
		"internal/version/version.go": "package version\nvar Version = \"1.0.2\"\n",
		"desktop/build/config.yml":    "version: '3'\ninfo:\n  productName: ScheduleGate\n  version: \"1.0.2\"\n",
		"docs/user-manual.html":       "schedulegate v1.0.2 badge\nschedulegate v1.0.2 footer\n",
		"docs/assess-manual.html":     "schedulegate v1.0.2 badge\nschedulegate v1.0.2 footer\n",
	}
}

func TestNormalize(t *testing.T) {
	bare, err := normalize("v1.2.3")
	if err != nil || bare != "1.2.3" {
		t.Fatalf("v-prefix: got %q %v", bare, err)
	}
	bare, err = normalize("1.2.3")
	if err != nil || bare != "1.2.3" {
		t.Fatalf("bare: got %q %v", bare, err)
	}
	if _, err := normalize("1.2"); err == nil {
		t.Fatal("expected error for 1.2")
	}
	if _, err := normalize("v1.2.3-beta"); err == nil {
		t.Fatal("expected error for prerelease")
	}
}

func TestApplyThenCheck(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, fixtureTree())

	if err := apply(root, "v2.3.4"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	problems := check(root, "v2.3.4")
	if len(problems) > 0 {
		t.Fatalf("check after apply: %v", problems)
	}
	problems = check(root, "v2.3.5")
	if len(problems) == 0 {
		t.Fatal("expected check to fail for a different version")
	}

	cfg, err := os.ReadFile(filepath.Join(root, "desktop/build/config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "version: '3'") {
		t.Fatal("must not rewrite Wails schema version")
	}
	if !strings.Contains(string(cfg), `version: "2.3.4"`) {
		t.Fatalf("desktop info.version not updated: %s", cfg)
	}

	readme, _ := os.ReadFile(filepath.Join(root, "README.md"))
	if !strings.Contains(string(readme), "**Public release:** v2.3.4 — ") {
		t.Fatalf("README not updated: %s", readme)
	}
}

func TestCheckDetectsDrift(t *testing.T) {
	root := t.TempDir()
	files := fixtureTree()
	files["Makefile"] = "VERSION ?= 9.9.9\n"
	writeTree(t, root, files)

	problems := check(root, "1.0.2")
	found := false
	for _, p := range problems {
		if strings.Contains(p, "Makefile") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Makefile drift, got %v", problems)
	}
}

func TestCheckDetectsMissingPattern(t *testing.T) {
	root := t.TempDir()
	files := fixtureTree()
	files["web/index.html"] = "<html>no zip url here</html>"
	writeTree(t, root, files)

	problems := check(root, "1.0.2")
	found := false
	for _, p := range problems {
		if strings.Contains(p, "website") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected website pattern miss, got %v", problems)
	}
}

func TestExtractorsOnRealRepo(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, st := range inspect(root) {
		if st.Err != nil {
			t.Errorf("%s: %v", st.Target.RelPath, st.Err)
			continue
		}
		if len(st.Versions) < st.Target.MinMatches {
			t.Errorf("%s: extractor found %d version(s), expected >= %d",
				st.Target.RelPath, len(st.Versions), st.Target.MinMatches)
		}
	}
}

func TestDesktopSchemaVersionUntouchedOnRealFile(t *testing.T) {
	root := filepath.Join("..", "..")
	content, err := readFile(root, "desktop/build/config.yml")
	if err != nil {
		t.Skip(err)
	}
	var desktop target
	for _, tgt := range targets() {
		if tgt.RelPath == "desktop/build/config.yml" {
			desktop = tgt
			break
		}
	}
	updated := desktop.ReplaceRe.ReplaceAllString(content, `  version: "9.9.9"`)
	if strings.Count(updated, `version: "9.9.9"`) != 1 {
		t.Fatalf("expected exactly one info.version replacement, content:\n%s", updated)
	}
	if !strings.Contains(updated, "version: '3'") {
		t.Fatal("Wails schema version: '3' must be preserved")
	}
}
