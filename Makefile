# Makefile for GoAhead - Unified Code Generation Tool

# Cross-compilation targets
PLATFORMS := linux/amd64 windows/amd64 darwin/amd64 darwin/arm64
BINARY_NAME := goahead

# Detect platform
ifeq ($(OS),Windows_NT)
    EXE_EXT = .exe
else
    EXE_EXT = 
endif

# Default target
.PHONY: all
all: build

# Build for current platform with auto version increment
.PHONY: build
build: increment-version
	@echo "Building goahead for current platform..."
	go build -o $(BINARY_NAME)$(EXE_EXT)

# Increment build/patch version automatically
.PHONY: increment-version
increment-version:
	@echo "Incrementing build version..."
	@go run - <<'EOF'
package main

import (
    "fmt"
    "os"
    "regexp"
    "strconv"
)

func main() {
    const file = "internal/constants.go"
    data, err := os.ReadFile(file)
    if err != nil {
        panic(err)
    }

    re := regexp.MustCompile(`Version\s*=\s*"(\d+)\.(\d+)\.(\d+)"`)
    matches := re.FindSubmatch(data)
    if matches == nil {
        panic("could not find version in internal/constants.go")
    }

    major, _ := strconv.Atoi(string(matches[1]))
    minor, _ := strconv.Atoi(string(matches[2]))
    patch, _ := strconv.Atoi(string(matches[3]))
    newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch+1)

    fmt.Printf("Updating version from %d.%d.%d to %s\n", major, minor, patch, newVersion)

    updated := re.ReplaceAll(data, []byte(fmt.Sprintf(`Version = "%s"`, newVersion)))
    if err := os.WriteFile(file, updated, 0o644); err != nil {
        panic(err)
    }
}
EOF

# Build without version increment
.PHONY: build-no-version
build-no-version:
	@echo "Building goahead for current platform (no version increment)..."
	go build -o $(BINARY_NAME)$(EXE_EXT)

# Install locally
.PHONY: install
install:
	@echo "Installing goahead..."
	go install

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
ifeq ($(OS),Windows_NT)
	del /f $(BINARY_NAME).exe 2>nul || echo "Binary already clean"
	rmdir /s /q dist 2>nul || echo "dist directory already clean"
else
	rm -f $(BINARY_NAME)
	rm -rf dist/
endif
	go clean

# Test the tool (create a simple test)
.PHONY: test
test:
	@echo "Running unit and integration tests..."
	go test ./...

# Test as toolexec (requires installation)
.PHONY: test-toolexec
test-toolexec: install
	@echo "Testing goahead as toolexec..."
	@echo "GoAhead installed. You can now use: go build -toolexec=\"goahead\" ."

# Cross-compilation
.PHONY: build-cross
build-cross:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p dist || md dist 2>nul || echo "dist directory exists"
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(BINARY_NAME)-$$OS-$$ARCH; \
		if [ "$$OS" = "windows" ]; then \
			OUTPUT_NAME=$$OUTPUT_NAME.exe; \
		fi; \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build -o dist/$$OUTPUT_NAME; \
	done

# Setup
.PHONY: setup
setup:
	go mod tidy

# Help
.PHONY: help
help:
	@echo "GoAhead - Unified Code Generation Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  install       - Install goahead locally"
	@echo "  test          - Run go test ./..."
	@echo "  test-toolexec - Test goahead as toolexec"
	@echo "  build-cross   - Cross-compile for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  setup         - Setup project dependencies"
	@echo "  help          - Show this help"
	@echo ""
	@echo "Usage after installation:"
	@echo "  go install github.com/AeonDave/goahead@latest"
	@echo "  go build -toolexec=\"goahead\" ./..."
