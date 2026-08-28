package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebsiteProOnlyStorefront(t *testing.T) {
	root := filepath.Join("..", "..")
	html, err := os.ReadFile(filepath.Join(root, "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)

	for _, want := range []string{
		`href="https://junqueira5.gumroad.com/l/schedulegate" class="price-btn">Buy Pro</a>`,
		`unzip schedulegate.zip`,
		`<code>schedulegate</code> (macOS)`,
		`<code>schedulegate.exe</code> (Windows)`,
		`<code>schedulegate-linux</code> (Linux)`,
		`Community: skip this step`,
		`schedulegate license set SG-`,
		`https://github.com/gjunqueira-sys/ScheduleGate/releases/latest/download/schedulegate-v`,
		`unzip -j sg.zip schedulegate-linux`,
		`<a href="/docs">Docs</a>`,
		`>Buy Pro</a>`,
		`github.com/gjunqueira-sys/ScheduleGate/issues`,
		`<a href="/terms">Terms</a>`,
		`<a href="/privacy">Privacy</a>`,
		`<a href="/refund">Refund</a>`,
		`mailto:support@schedulegate.dev`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("index.html missing %q", want)
		}
	}

	for _, forbid := range []string{
		"Buy Team",
		"Buy Lifetime",
		"Buy Enterprise",
		"USER_MANUAL.md",
		"unzip schedulegate &&",
		"https://github.com/.../schedulegate-linux",
	} {
		if strings.Contains(page, forbid) {
			t.Errorf("index.html must not contain %q", forbid)
		}
	}

	if strings.Count(page, "gumroad.com") != 2 {
		t.Errorf("expected exactly 2 Gumroad links (Pro CTA + footer), got %d", strings.Count(page, "gumroad.com"))
	}
}

func TestVercelPricingRedirect(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "web", "vercel.json"))
	if err != nil {
		t.Fatal(err)
	}

	var cfg struct {
		Redirects []struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"redirects"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("vercel.json: %v", err)
	}

	found := false
	for _, r := range cfg.Redirects {
		if r.Source == "/pricing" && r.Destination == "/#pricing" {
			found = true
		}
	}
	if !found {
		t.Fatal("vercel.json must redirect /pricing → /#pricing")
	}
}

func TestLegalPagesPublished(t *testing.T) {
	root := filepath.Join("..", "..")
	files := []string{"terms.html", "privacy.html", "refund.html"}
	combined := ""
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(root, "web", name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		page := string(raw)
		combined += page
		if !strings.Contains(page, "support@schedulegate.dev") {
			t.Errorf("%s missing support@schedulegate.dev", name)
		}
		for _, forbid := range []string{
			"DRAFT — not in force",
			`name="robots" content="noindex, nofollow"`,
			"Draft dated",
			"not published policy",
			"has no effect until published",
		} {
			if strings.Contains(page, forbid) {
				t.Errorf("%s still contains draft marker %q", name, forbid)
			}
		}
	}

	for _, want := range []string{
		"AGPLv3",
		"Community",
		"Commercial License",
		"merchant of record",
		"Resend",
		"no telemetry",
		"14 days",
		"future purchases",
		"Effective 27 August 2026",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("legal pages missing required topic %q", want)
		}
	}
}

func TestGumroadDescriptionHasLegalURLs(t *testing.T) {
	root := filepath.Join("..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "scripts", "release", "gumroad", "description-template.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	for _, want := range []string{
		"https://schedulegate.dev/terms",
		"https://schedulegate.dev/privacy",
		"https://schedulegate.dev/refund",
		"support@schedulegate.dev",
		"schedulegate-linux",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("gumroad description-template.md missing %q", want)
		}
	}
	for _, forbid := range []string{
		"schedulegate-gui",
		"MIT License",
	} {
		if strings.Contains(page, forbid) {
			t.Errorf("gumroad description-template.md must not contain %q", forbid)
		}
	}
}
