package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// GumroadBrowserAutomation generates structured instructions for updating
// a Gumroad product via Playwright browser automation.
//
// This script is designed to be run from within the opencode session
// (which has Playwright browser tools available). It outputs a JSON
// manifest that the assistant reads to execute the browser steps.
//
// Usage:
//
//	go run main.go --zip path/to/schedulegate-v1.0.3.zip \
//	               --description path/to/description.md \
//	               --version v1.0.3
//
// Credentials are read from:
//   - Environment variables: GUMROAD_EMAIL, GUMROAD_PASSWORD, GUMROAD_PRODUCT_URL
//   - Interactive prompts if not set

func main() {
	args := parseArgs()

	creds, err := getCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting credentials: %v\n", err)
		os.Exit(1)
	}

	description, err := os.ReadFile(args.descriptionPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading description file: %v\n", err)
		os.Exit(1)
	}

	manifest := Manifest{
		Version:              args.version,
		ZipPath:              args.zipPath,
		Description:          string(description),
		DescriptionPath:      args.descriptionPath,
		GumroadLoginURL:      "https://gumroad.com/login",
		GumroadEmail:         creds.email,
		GumroadPasswordHint:  "use GUMROAD_PASSWORD env var",
		ProductURL:           creds.productURL,
		InstructionsPath:     fmt.Sprintf("/tmp/gumroad-instructions-%s.txt", time.Now().Format("20060102-150405")),
		Steps: []Step{
			{Number: 1, Action: "navigate", Target: "https://gumroad.com/login", Description: "Navigate to Gumroad login page"},
			{Number: 2, Action: "login", Target: "login form", Description: "Enter credentials and login"},
			{Number: 3, Action: "navigate", Target: creds.productURL, Description: "Navigate to product edit page"},
			{Number: 4, Action: "click", Target: "Upload files button", Description: "Click upload/add file button"},
			{Number: 5, Action: "upload", Target: args.zipPath, Description: "Upload the release zip file"},
			{Number: 6, Action: "wait", Target: "upload progress bar", Description: "Wait for upload to complete"},
			{Number: 7, Action: "replace_text", Target: "product description editor", Description: "Replace entire product description"},
			{Number: 8, Action: "click", Target: "Save/Publish button", Description: "Click Save or Publish changes"},
			{Number: 9, Action: "verify", Target: "product page", Description: "Verify version and file are updated"},
		},
	}

	// Write human-readable instructions file
	saveInstructions(creds, args, manifest.InstructionsPath)

	// Output JSON manifest for the assistant to consume
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(manifest)
}

type Step struct {
	Number      int    `json:"number"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

type Manifest struct {
	Version             string `json:"version"`
	ZipPath             string `json:"zip_path"`
	Description         string `json:"description"`
	DescriptionPath     string `json:"description_path"`
	GumroadLoginURL     string `json:"gumroad_login_url"`
	GumroadEmail        string `json:"gumroad_email"`
	GumroadPasswordHint string `json:"gumroad_password_hint"`
	ProductURL          string `json:"product_url"`
	InstructionsPath    string `json:"instructions_path"`
	Steps               []Step `json:"steps"`
}

type Args struct {
	zipPath         string
	descriptionPath string
	version         string
}

type Credentials struct {
	email      string
	password   string
	productURL string
}

func parseArgs() Args {
	args := Args{}

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--zip":
			if i+1 < len(os.Args) {
				args.zipPath = os.Args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(os.Args) {
				args.descriptionPath = os.Args[i+1]
				i++
			}
		case "--version":
			if i+1 < len(os.Args) {
				args.version = os.Args[i+1]
				i++
			}
		}
	}

	if args.zipPath == "" {
		args.zipPath = "release/schedulegate.zip"
	}
	if args.descriptionPath == "" {
		args.descriptionPath = "/tmp/gumroad-description.md"
	}
	if args.version == "" {
		args.version = "v1.0.3"
	}

	return args
}

func getCredentials() (Credentials, error) {
	creds := Credentials{
		email:      os.Getenv("GUMROAD_EMAIL"),
		password:   os.Getenv("GUMROAD_PASSWORD"),
		productURL: os.Getenv("GUMROAD_PRODUCT_URL"),
	}

	if creds.productURL == "" {
		creds.productURL = "https://junqueira5.gumroad.com/l/schedulegate"
	}

	reader := bufio.NewReader(os.Stdin)

	if creds.email == "" {
		fmt.Print("Gumroad Email: ")
		email, err := reader.ReadString('\n')
		if err != nil {
			return creds, err
		}
		creds.email = strings.TrimSpace(email)
	}

	if creds.password == "" {
		fmt.Print("Gumroad Password: ")
		password, err := reader.ReadString('\n')
		if err != nil {
			return creds, err
		}
		creds.password = strings.TrimSpace(password)
	}

	return creds, nil
}

func saveInstructions(creds Credentials, args Args, path string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create instruction file: %v\n", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "Gumroad Update Instructions for %s\n", args.version)
	fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "Credentials:\n")
	fmt.Fprintf(f, "  Email: %s\n", creds.email)
	fmt.Fprintf(f, "  Password: [use environment variable]\n")
	fmt.Fprintf(f, "  Product URL: %s\n\n", creds.productURL)
	fmt.Fprintf(f, "Files:\n")
	fmt.Fprintf(f, "  Zip: %s\n", args.zipPath)
	fmt.Fprintf(f, "  Description: %s\n\n", args.descriptionPath)
	fmt.Fprintf(f, "Steps:\n")
	fmt.Fprintf(f, "1. Navigate to https://gumroad.com/login\n")
	fmt.Fprintf(f, "2. Login with credentials\n")
	fmt.Fprintf(f, "3. Navigate to product edit page: %s\n", creds.productURL)
	fmt.Fprintf(f, "4. Find file upload section, remove old file if exists\n")
	fmt.Fprintf(f, "5. Upload: %s\n", args.zipPath)
	fmt.Fprintf(f, "6. Wait for upload to complete\n")
	fmt.Fprintf(f, "7. Find description editor, replace entire content\n")
	fmt.Fprintf(f, "8. Click Save/Publish\n")
	fmt.Fprintf(f, "9. Verify success\n")
}
