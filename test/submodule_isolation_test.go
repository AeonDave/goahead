package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/AeonDave/goahead/internal"
)

// TestSubmoduleIsolation verifies that submodules (directories with go.mod) are treated as separate projects
func TestSubmoduleIsolation(t *testing.T) {
	// Create a root project with a submodule
	// Root has its own helper with Version() returning "1.0"
	// Submodule has its own helper with Version() returning "2.0"
	// They should NOT interfere with each other

	dir, err := os.MkdirTemp("", "goahead_submodule_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Root go.mod
	writeTestFile(t, dir, "go.mod", "module rootproject\ngo 1.21\n")

	// Root helper
	writeTestFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Version() string {
	return "1.0-root"
}
`)

	// Root main.go
	writeTestFile(t, dir, "main.go", `package main

//:Version:
var version = ""

func main() {
	println(version)
}
`)

	// Submodule directory with its own go.mod
	subDir := filepath.Join(dir, "subproject")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subproject dir: %v", err)
	}

	// Submodule go.mod
	writeTestFile(t, subDir, "go.mod", "module subproject\ngo 1.21\n")

	// Submodule helper - different Version!
	writeTestFile(t, subDir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Version() string {
	return "2.0-sub"
}
`)

	// Submodule main.go
	writeTestFile(t, subDir, "main.go", `package main

//:Version:
var version = ""

func main() {
	println(version)
}
`)

	// Run codegen on root - should process root AND submodule separately
	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Check root main.go - should have "1.0-root"
	rootContent, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read root main.go: %v", err)
	}
	if !strings.Contains(string(rootContent), `version = "1.0-root"`) {
		t.Errorf("Root main.go should have version = \"1.0-root\", got:\n%s", string(rootContent))
	}
	// CRITICAL: Root should NOT have submodule's version
	if strings.Contains(string(rootContent), `version = "2.0-sub"`) {
		t.Errorf("Root main.go should NOT have submodule's version, but got:\n%s", string(rootContent))
	}

	// Check submodule main.go - should have "2.0-sub" (not "1.0-root"!)
	subContent, err := os.ReadFile(filepath.Join(subDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read submodule main.go: %v", err)
	}
	if !strings.Contains(string(subContent), `version = "2.0-sub"`) {
		t.Errorf("Submodule main.go should have version = \"2.0-sub\", got:\n%s", string(subContent))
	}
	// CRITICAL: Submodule should NOT have root's version
	if strings.Contains(string(subContent), `version = "1.0-root"`) {
		t.Errorf("Submodule main.go should NOT have root's version, but got:\n%s", string(subContent))
	}
}

// TestSubmoduleShadowingPrevention verifies that submodules don't see parent helpers
func TestSubmoduleShadowingPrevention(t *testing.T) {
	dir, err := os.MkdirTemp("", "goahead_shadow_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Root with helper
	writeTestFile(t, dir, "go.mod", "module rootproject\ngo 1.21\n")
	writeTestFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func RootOnly() string {
	return "from-root"
}
`)

	// Root uses its own function - should work
	writeTestFile(t, dir, "main.go", `package main

//:RootOnly:
var rootValue = ""

func main() {
	println(rootValue)
}
`)

	// Submodule WITHOUT helpers - should NOT see RootOnly
	subDir := filepath.Join(dir, "subproject")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subproject dir: %v", err)
	}
	writeTestFile(t, subDir, "go.mod", "module subproject\ngo 1.21\n")

	// Submodule tries to use RootOnly - should fail (not find it)
	writeTestFile(t, subDir, "main.go", `package main

//:RootOnly:
var value = ""

func main() {
	println(value)
}
`)

	// Run codegen
	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// CRITICAL: Root should have its placeholder replaced
	rootContent, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read root main.go: %v", err)
	}
	if !strings.Contains(string(rootContent), `rootValue = "from-root"`) {
		t.Errorf("Root should see its own RootOnly function, but got:\n%s", string(rootContent))
	}

	// CRITICAL: Submodule should NOT have the value replaced (RootOnly not visible)
	subContent, err := os.ReadFile(filepath.Join(subDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read submodule main.go: %v", err)
	}
	// The placeholder should NOT be replaced because RootOnly is not visible to submodule
	if strings.Contains(string(subContent), `value = "from-root"`) {
		t.Errorf("Submodule should NOT see parent's RootOnly function, but got:\n%s", string(subContent))
	}
	// Original placeholder should remain
	if !strings.Contains(string(subContent), `value = ""`) {
		t.Errorf("Submodule should keep original empty value, but got:\n%s", string(subContent))
	}
}

// TestNestedSubmodules verifies that nested submodules work correctly
func TestNestedSubmodules(t *testing.T) {
	dir, err := os.MkdirTemp("", "goahead_nested_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Root
	writeTestFile(t, dir, "go.mod", "module root\ngo 1.21\n")
	writeTestFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Level() string { return "root" }
`)
	writeTestFile(t, dir, "main.go", `package main

//:Level:
var level = ""

func main() { println(level) }
`)

	// Level 1 submodule
	sub1 := filepath.Join(dir, "sub1")
	_ = os.MkdirAll(sub1, 0o755)
	writeTestFile(t, sub1, "go.mod", "module sub1\ngo 1.21\n")
	writeTestFile(t, sub1, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Level() string { return "sub1" }
`)
	writeTestFile(t, sub1, "main.go", `package main

//:Level:
var level = ""

func main() { println(level) }
`)

	// Level 2 submodule (nested inside sub1)
	sub2 := filepath.Join(sub1, "sub2")
	_ = os.MkdirAll(sub2, 0o755)
	writeTestFile(t, sub2, "go.mod", "module sub2\ngo 1.21\n")
	writeTestFile(t, sub2, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Level() string { return "sub2" }
`)
	writeTestFile(t, sub2, "main.go", `package main

//:Level:
var level = ""

func main() { println(level) }
`)

	// Run codegen from root
	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Verify each level has its own value
	checkContent := func(path, expected string) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", path, err)
		}
		if !strings.Contains(string(content), expected) {
			t.Errorf("%s should contain %q, got:\n%s", path, expected, string(content))
		}
	}

	checkContent(filepath.Join(dir, "main.go"), `level = "root"`)
	checkContent(filepath.Join(sub1, "main.go"), `level = "sub1"`)
	checkContent(filepath.Join(sub2, "main.go"), `level = "sub2"`)
}

// TestSiblingWithoutSubmodule verifies that siblings without go.mod share parent's helpers
func TestSiblingWithoutSubmodule(t *testing.T) {
	dir, err := os.MkdirTemp("", "goahead_sibling_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)

	// Root with helper
	writeTestFile(t, dir, "go.mod", "module root\ngo 1.21\n")
	writeTestFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func SharedFunc() string { return "shared" }
`)

	// Sibling 1 - NO go.mod, should use parent's helper
	sib1 := filepath.Join(dir, "sibling1")
	_ = os.MkdirAll(sib1, 0o755)
	writeTestFile(t, sib1, "main.go", `package main

//:SharedFunc:
var value = ""

func main() { println(value) }
`)

	// Sibling 2 - HAS go.mod, should NOT see parent's helper
	sib2 := filepath.Join(dir, "sibling2")
	_ = os.MkdirAll(sib2, 0o755)
	writeTestFile(t, sib2, "go.mod", "module sibling2\ngo 1.21\n")
	writeTestFile(t, sib2, "main.go", `package main

//:SharedFunc:
var value = ""

func main() { println(value) }
`)

	// Run codegen
	if err := RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Sibling 1 should have value replaced (same project)
	sib1Content, err := os.ReadFile(filepath.Join(sib1, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read sibling1 main.go: %v", err)
	}
	if !strings.Contains(string(sib1Content), `value = "shared"`) {
		t.Errorf("Sibling1 (same project) should have value = \"shared\", got:\n%s", string(sib1Content))
	}

	// Sibling 2 should NOT have value replaced (separate project)
	sib2Content, err := os.ReadFile(filepath.Join(sib2, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read sibling2 main.go: %v", err)
	}
	if strings.Contains(string(sib2Content), `value = "shared"`) {
		t.Errorf("Sibling2 (submodule) should NOT see parent's SharedFunc, but got:\n%s", string(sib2Content))
	}
}

// Helper to write test files
func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write %s: %v", path, err)
	}
}
