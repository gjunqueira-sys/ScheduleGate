# Binary name
BINARY_NAME=schedulegate
BIN_DIR=bin
VERSIONS_DIR=bin/versions

# Version metadata (override VERSION= on the command line)
VERSION ?= 1.0.5
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PKG := github.com/gjunqueira-sys/ScheduleGate/internal/version
LICENSE_PKG := github.com/gjunqueira-sys/ScheduleGate/internal/license

# License signing secret. Override at release time:
#   make build LICENSE_SECRET=my-secret
# Leave empty for dev builds (runs as free Community tier).
LICENSE_SECRET ?=

# Safeguard: warn if a secret is provided in a local (non-CI) build.
# The production secret should only be injected via GitHub Actions or the
# release pipeline. Local builds with the prod secret can generate valid
# Pro licenses, defeating the licensing model.
ifneq ($(LICENSE_SECRET),)
  ifeq ($(CI),)
    $(warning ⚠️  LICENSE_SECRET is set in a local build. This binary can generate valid Pro licenses.)
    $(warning    Only use this for testing with a non-production secret.)
    $(warning    The prod secret should only be injected via GitHub Actions or release pipeline.)
  endif
endif

LDFLAGS=-ldflags "-s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE) \
	-X $(LICENSE_PKG).SecretKey=$(LICENSE_SECRET)"

.PHONY: all build build-windows build-mac build-linux build-all build-versioned build-license-server clean test

all: clean build-mac build-windows build-linux

# Build for the current platform (defaulting to macOS based on dev environment)
build: build-mac

# Build for Windows (amd64)
build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME).exe main.go

# Build for macOS (arm64 for Apple Silicon, change to amd64 if needed)
build-mac:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) main.go

# Build for Linux (amd64)
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux main.go

# Build all three platform targets
build-all: clean build-mac build-windows build-linux

# Build the license minting server (store webhook delivery backend)
# Requires SG_SECRET / SG_ADMIN_TOKEN at runtime, see docs/LICENSING_DEPLOYMENT.md
build-license-server:
	@echo "Building license server..."
	go build -o $(BIN_DIR)/license-server ./cmd/license-server

# Build versioned binary (preserves in bin/versions/)
build-versioned: build-mac
	@mkdir -p $(VERSIONS_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME) $(VERSIONS_DIR)/$(BINARY_NAME)-v$(VERSION)
	@echo "Saved: $(VERSIONS_DIR)/$(BINARY_NAME)-v$(VERSION)"

# Run all tests
test:
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)/schedulegate $(BIN_DIR)/schedulegate.exe $(BIN_DIR)/schedulegate-linux $(BIN_DIR)/license-server
