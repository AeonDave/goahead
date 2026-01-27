package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSubcommandBuild tests that 'goahead build' runs codegen then go build
func TestSubcommandBuild(t *testing.T) {
	dir := t.TempDir()

	// Create helper file
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetVersion() string { return "1.0.0" }
`)

	// Create main file with placeholder
	writeFile(t, dir, "main.go", `package main

import "fmt"

//:GetVersion:
var version = ""

func main() {
	fmt.Println(version)
}
`)

	// Create go.mod
	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// Build goahead first
	goaheadExe := buildGoahead(t)

	// Run goahead build
	cmd := exec.Command(goaheadExe, "build", "-o", filepath.Join(dir, "testapp.exe"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goahead build failed: %v\nOutput: %s", err, output)
	}

	// Verify the source file was modified
	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(content), `"1.0.0"`) {
		t.Errorf("Expected placeholder to be replaced with '1.0.0', got:\n%s", content)
	}

	// Verify binary was created
	binaryPath := filepath.Join(dir, "testapp.exe")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("Expected binary to be created at %s", binaryPath)
	}
}

// TestSubcommandBuildWithTags tests that build flags are passed through
func TestSubcommandBuildWithTags(t *testing.T) {
	dir := t.TempDir()

	// Create helper file
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetMode() string { return "debug" }
`)

	// Create main file
	writeFile(t, dir, "main.go", `package main

//:GetMode:
var mode = ""

func main() {}
`)

	// Create go.mod
	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	goaheadExe := buildGoahead(t)

	// Run goahead build with -tags flag
	cmd := exec.Command(goaheadExe, "build", "-tags", "integration", "-o", filepath.Join(dir, "app.exe"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goahead build with tags failed: %v\nOutput: %s", err, output)
	}

	// Verify source was processed
	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(content), `"debug"`) {
		t.Errorf("Expected placeholder to be replaced, got:\n%s", content)
	}
}

// TestSubcommandTest tests that 'goahead test' works
func TestSubcommandTest(t *testing.T) {
	dir := t.TempDir()

	// Create helper file
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetTestValue() string { return "test-value" }
`)

	// Create main file
	writeFile(t, dir, "main.go", `package main

//:GetTestValue:
var testVal = ""
`)

	// Create test file
	writeFile(t, dir, "main_test.go", `package main

import "testing"

func TestValue(t *testing.T) {
	if testVal != "test-value" {
		t.Errorf("expected 'test-value', got '%s'", testVal)
	}
}
`)

	// Create go.mod
	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	goaheadExe := buildGoahead(t)

	// Run goahead test
	cmd := exec.Command(goaheadExe, "test", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goahead test failed: %v\nOutput: %s", err, output)
	}

	// Verify test passed (output should contain PASS)
	if !strings.Contains(string(output), "PASS") && !strings.Contains(string(output), "ok") {
		t.Errorf("Expected test to pass, got output:\n%s", output)
	}
}

// TestSubcommandWithRecursivePattern tests ./... pattern
func TestSubcommandWithRecursivePattern(t *testing.T) {
	dir := t.TempDir()

	// Create root helper
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package helpers

func RootFunc() string { return "root" }
`)

	// Create subpackage
	writeFile(t, dir, "pkg/main.go", `package pkg

//:RootFunc:
var Val = ""
`)

	// Create go.mod
	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	goaheadExe := buildGoahead(t)

	// Run goahead build with ./...
	cmd := exec.Command(goaheadExe, "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Build might fail due to no main package, but codegen should still run
		t.Logf("Build output (may fail, checking codegen): %s", output)
	}

	// Verify the subpackage file was processed
	content, _ := os.ReadFile(filepath.Join(dir, "pkg/main.go"))
	if !strings.Contains(string(content), `"root"`) {
		t.Errorf("Expected subpackage placeholder to be replaced, got:\n%s", content)
	}
}

// TestSubcommandVerbose tests verbose output with GOAHEAD_VERBOSE
func TestSubcommandVerbose(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getVal() string { return "verbose-test" }
`)

	writeFile(t, dir, "main.go", `package main

//:getVal:
var v = ""

func main() {}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	goaheadExe := buildGoahead(t)

	cmd := exec.Command(goaheadExe, "build", "-o", filepath.Join(dir, "app.exe"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOAHEAD_VERBOSE=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goahead build failed: %v\nOutput: %s", err, output)
	}

	// Verify verbose output
	if !strings.Contains(string(output), "[goahead]") {
		t.Errorf("Expected verbose output with [goahead] prefix, got:\n%s", output)
	}
}

// buildGoahead builds the goahead binary and returns its path
func buildGoahead(t *testing.T) string {
	t.Helper()

	// Get the module root (parent of test directory)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	moduleRoot := filepath.Dir(wd)

	// Build goahead to a temp location
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "goahead.exe")

	cmd := exec.Command("go", "build", "-o", exePath, ".")
	cmd.Dir = moduleRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build goahead: %v\nOutput: %s", err, output)
	}

	return exePath
}
