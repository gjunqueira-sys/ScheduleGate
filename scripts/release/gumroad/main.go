package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// GumroadBrowserAutomation handles browser-based updates to Gumroad products.
// This script is designed to be called from the release pipeline.
//
// Usage:
//
//	go run main.go --zip path/to/schedulegate-v1.0.3.zip \
//	               --description path/to/description.md \
//	               --version v1.0.3
//
// The script uses the Playwright browser tools available in the opencode session
// to navigate Gumroad, login, upload files, and update product descriptions.
//
// Credentials are read from:
//   - Environment variables: GUMROAD_EMAIL, GUMROAD_PASSWORD, GUMROAD_PRODUCT_URL
//   - Interactive prompts if not set

func main() {
	args := parseArgs()

	fmt.Println("=== Gumroad Product Update ===")
	fmt.Printf("Version: %s\n", args.version)
	fmt.Printf("Zip file: %s\n", args.zipPath)
	fmt.Printf("Description: %s\n", args.descriptionPath)
	fmt.Println()

	// Get credentials
	creds, err := getCredentials(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting credentials: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Email: %s\n", creds.email)
	fmt.Printf("Product URL: %s\n", creds.productURL)
	fmt.Println()

	// Read the description file
	description, err := os.ReadFile(args.descriptionPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading description file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Description loaded successfully")
	fmt.Println()

	// Execute the browser automation steps
	// Note: This script is designed to be run within the opencode session
	// which has Playwright browser tools available. The actual browser
	// automation will be performed by the assistant using the available tools.
	fmt.Println("=== Browser Automation Steps ===")
	fmt.Println()
	fmt.Println("The following steps will be performed via Playwright:")
	fmt.Println("1. Navigate to Gumroad login page")
	fmt.Println("2. Enter credentials and login")
	fmt.Println("3. Navigate to product edit page")
	fmt.Println("4. Upload new zip file")
	fmt.Println("5. Replace product description")
	fmt.Println("6. Save/publish changes")
	fmt.Println("7. Verify success")
	fmt.Println()

	// Generate step-by-step instructions for the assistant
	generateInstructions(creds, args, string(description))

	fmt.Println()
	fmt.Println("=== Instructions Generated ===")
	fmt.Println("The assistant will now execute these steps using Playwright browser tools.")
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

func getCredentials(args Args) (Credentials, error) {
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
		// Note: In a real implementation, you'd want to use terminal
		// password input (no echo). For now, we read from stdin.
		password, err := reader.ReadString('\n')
		if err != nil {
			return creds, err
		}
		creds.password = strings.TrimSpace(password)
	}

	return creds, nil
}

func generateInstructions(creds Credentials, args Args, description string) {
	fmt.Println("=== Playwright Instructions ===")
	fmt.Println()
	fmt.Printf("# Step 1: Navigate to Gumroad login\n")
	fmt.Printf("# URL: https://gumroad.com/login\n")
	fmt.Println()
	fmt.Printf("# Step 2: Login\n")
	fmt.Printf("# Email: %s\n", creds.email)
	fmt.Printf("# Password: [hidden]\n")
	fmt.Println()
	fmt.Printf("# Step 3: Navigate to product edit page\n")
	fmt.Printf("# URL: %s\n", creds.productURL)
	fmt.Println()
	fmt.Printf("# Step 4: Upload file\n")
	fmt.Printf("# File: %s\n", args.zipPath)
	fmt.Println()
	fmt.Printf("# Step 5: Update description\n")
	fmt.Printf("# Length: %d characters\n", len(description))
	fmt.Println()

	// Save instructions to a file for the assistant
	instructionFile := fmt.Sprintf("/tmp/gumroad-instructions-%s.txt", time.Now().Format("20060102-150405"))
	f, err := os.Create(instructionFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not create instruction file: %v\n", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "Gumroad Update Instructions for %s\n", args.version)
	fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "Credentials:\n")
	fmt.Fprintf(f, "  Email: %s\n", creds.email)
	fmt.Fprintf(f, "  Password: [use environment variable or prompt]\n")
	fmt.Fprintf(f, "  Product URL: %s\n\n", creds.productURL)
	fmt.Fprintf(f, "Files:\n")
	fmt.Fprintf(f, "  Zip: %s\n", args.zipPath)
	fmt.Fprintf(f, "  Description: %s\n\n", args.descriptionPath)
	fmt.Fprintf(f, "Steps:\n")
	fmt.Fprintf(f, "1. Navigate to https://gumroad.com/login\n")
	fmt.Fprintf(f, "2. Login with credentials\n")
	fmt.Fprintf(f, "3. Navigate to product edit page: %s\n", creds.productURL)
	fmt.Fprintf(f, "4. Find file upload section\n")
	fmt.Fprintf(f, "5. Remove old file if exists\n")
	fmt.Fprintf(f, "6. Upload: %s\n", args.zipPath)
	fmt.Fprintf(f, "7. Find description editor\n")
	fmt.Fprintf(f, "8. Replace entire description with content from: %s\n", args.descriptionPath)
	fmt.Fprintf(f, "9. Click Save/Publish\n")
	fmt.Fprintf(f, "10. Verify success\n")

	fmt.Printf("Instructions saved to: %s\n", instructionFile)
}
