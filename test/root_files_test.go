package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/AeonDave/goahead/internal"
)

// TestRootFilesProcessed verifies that files in the root directory are processed
func TestRootFilesProcessed(t *testing.T) {
	dir, err := os.MkdirTemp("", "goahead_root_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Root go.mod
	writeFile(t, dir, "go.mod", "module testmod\ngo 1.22\n")

	// Helper file in root
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetVersion() string {
	return "1.0.0"
}
`)

	// Source file in root that uses placeholder
	writeFile(t, dir, "main.go", `package main

//:GetVersion:
var version = ""

func main() {
	println(version)
}
`)

	// Run codegen
	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Verify root file was processed
	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	if !strings.Contains(string(content), `version = "1.0.0"`) {
		t.Errorf("Root file should have placeholder replaced.\nExpected: version = \"1.0.0\"\nGot:\n%s", string(content))
	}
}

// TestRootFilesProcessedWithDotSlash verifies that ".\\" path works correctly
// This reproduces the real-world usage: goahead -dir .\
func TestRootFilesProcessedWithDotSlash(t *testing.T) {
	// Create temp dir and chdir to it to test relative paths
	dir, err := os.MkdirTemp("", "goahead_dotslash_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	writeFile(t, dir, "go.mod", "module testmod\ngo 1.22\n")

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetVersion() string {
	return "1.0.0"
}
`)

	writeFile(t, dir, "main.go", `package main

//:GetVersion:
var version = ""

func main() {
	println(version)
}
`)

	// Save and restore working directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Change to the temp dir and use ".\" as path (like the user does)
	os.Chdir(dir)
	if err := RunCodegen(".\\", false); err != nil {
		t.Fatalf("RunCodegen with .\\ failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	if !strings.Contains(string(content), `version = "1.0.0"`) {
		t.Errorf("Root file should have placeholder replaced with .\\ path.\nExpected: version = \"1.0.0\"\nGot:\n%s", string(content))
	}
}

// TestRootFilesWithSubdirAndDotPath verifies root + subdirectory processing with "." path
func TestRootFilesWithSubdirAndDotPath(t *testing.T) {
	dir, err := os.MkdirTemp("", "goahead_rootsub_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	writeFile(t, dir, "go.mod", "module testmod\ngo 1.22\n")

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetEnv() string {
	return "production"
}
`)

	// Root file
	writeFile(t, dir, "main.go", `package main

//:GetEnv:
var env = ""

func main() {
	println(env)
}
`)

	// Subdirectory file (should also be processed using root helper)
	writeFile(t, dir, "sub/main.go", `package sub

//:GetEnv:
var subEnv = ""
`)

	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Root file should be replaced
	rootContent, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read root main.go: %v", err)
	}
	if !strings.Contains(string(rootContent), `env = "production"`) {
		t.Errorf("Root file not processed.\nGot:\n%s", string(rootContent))
	}

	// Sub file should also be replaced
	subContent, err := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	if err != nil {
		t.Fatalf("Failed to read sub/main.go: %v", err)
	}
	if !strings.Contains(string(subContent), `subEnv = "production"`) {
		t.Errorf("Subdirectory file not processed.\nGot:\n%s", string(subContent))
	}
}
