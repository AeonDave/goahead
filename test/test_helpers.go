package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/AeonDave/goahead/internal"
)

// writeFile is a helper function for creating test files
func writeFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// setupTestDir creates a temporary directory with the specified files
// Returns the directory path and a cleanup function
func setupTestDir(t *testing.T, files map[string]string) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "goahead_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create go.mod
	modContent := "module testmodule\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modContent), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test files
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			_ = os.RemoveAll(dir)
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	return dir, cleanup
}

// processAndReplace runs the full codegen pipeline on a file and returns the result
func processAndReplace(t *testing.T, dir string, fileName string) (string, error) {
	t.Helper()

	// Run codegen
	if err := RunCodegen(dir, false); err != nil {
		return "", err
	}

	// Read the processed file
	content, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatalf("Failed to read processed file: %v", err)
	}

	return string(content), nil
}

// verifyCompiles checks that the generated Go code actually compiles
// If input looks like a directory path, it reads main.go from that directory
// Otherwise, it treats input as Go source code directly
func verifyCompiles(t *testing.T, input string) {
	t.Helper()

	var code string

	// Check if input is a directory path
	if info, err := os.Stat(input); err == nil && info.IsDir() {
		// Input is a directory - read main.go from it
		content, err := os.ReadFile(filepath.Join(input, "main.go"))
		if err != nil {
			t.Fatalf("Failed to read main.go from directory %s: %v", input, err)
		}
		code = string(content)
	} else {
		// Input is the actual code
		code = input
	}

	// Create a temporary directory
	dir, err := os.MkdirTemp("", "goahead_compile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Create go.mod
	modContent := "module testmodule\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modContent), 0o644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Write the code to a file
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(code), 0o644); err != nil {
		t.Fatalf("Failed to write code: %v", err)
	}

	// Try to compile it
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "test_binary"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Generated code does not compile:\n%s\nError: %v", string(output), err)
	}
}
