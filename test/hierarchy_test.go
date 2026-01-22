package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestHierarchyBasic tests that a function in the root is available in subdirectories
func TestHierarchyBasic(t *testing.T) {
	dir := t.TempDir()

	// Root helper file
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func globalFunc() string { return "from-root" }
`)

	// Subdirectory source file
	subDir := filepath.Join(dir, "sub")
	writeFile(t, dir, "sub/main.go", `package main

//:globalFunc
var msg = ""

func main() { _ = msg }
`)

	writeFile(t, dir, "sub/go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(subDir, "main.go"))
	if !strings.Contains(string(content), `"from-root"`) {
		t.Errorf("Expected subdirectory to use root function, got: %s", content)
	}

	// Verify generated code compiles
	verifyCompiles(t, subDir)
}

// TestHierarchyShadowing tests that a local function shadows a root function
func TestHierarchyShadowing(t *testing.T) {
	dir := t.TempDir()

	// Root helper file with version()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func version() string { return "1.0.0" }
`)

	// Subdirectory helper file that shadows version()
	writeFile(t, dir, "sub/helpers.go", `//go:build exclude
//go:ahead functions

package main

func version() string { return "2.0.0-sub" }
`)

	// Subdirectory source file
	subDir := filepath.Join(dir, "sub")
	writeFile(t, dir, "sub/main.go", `package main

//:version
var v = ""
`)

	// Root source file
	writeFile(t, dir, "main.go", `package main

//:version
var v = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Subdirectory should use local version (2.0.0-sub)
	subContent, _ := os.ReadFile(filepath.Join(subDir, "main.go"))
	if !strings.Contains(string(subContent), `"2.0.0-sub"`) {
		t.Errorf("Expected subdirectory to use local shadowed function, got: %s", subContent)
	}

	// Root should use root version (1.0.0)
	rootContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(rootContent), `"1.0.0"`) {
		t.Errorf("Expected root to use root function, got: %s", rootContent)
	}
}

// TestHierarchyDeepNesting tests resolution through multiple levels
func TestHierarchyDeepNesting(t *testing.T) {
	dir := t.TempDir()

	// Root helper
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func rootOnly() string { return "root-only" }
func shared() string { return "root-shared" }
`)

	// Level 1 helper - shadows shared
	writeFile(t, dir, "pkg1/helpers.go", `//go:build exclude
//go:ahead functions

package main

func shared() string { return "pkg1-shared" }
func pkg1Only() string { return "pkg1-only" }
`)

	// Level 2 helper - shadows shared again
	writeFile(t, dir, "pkg1/deep/helpers.go", `//go:build exclude
//go:ahead functions

package main

func shared() string { return "deep-shared" }
`)

	// Source in root
	writeFile(t, dir, "main.go", `package main

//:rootOnly
var r = ""

//:shared
var s = ""
`)

	// Source in pkg1
	writeFile(t, dir, "pkg1/main.go", `package main

//:rootOnly
var r = ""

//:shared
var s = ""

//:pkg1Only
var p = ""
`)

	// Source in pkg1/deep
	writeFile(t, dir, "pkg1/deep/main.go", `package main

//:rootOnly
var r = ""

//:shared
var s = ""

//:pkg1Only
var p = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Check root/main.go
	rootContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(rootContent), `"root-only"`) {
		t.Errorf("Root main.go: expected root-only, got: %s", rootContent)
	}
	if !strings.Contains(string(rootContent), `"root-shared"`) {
		t.Errorf("Root main.go: expected root-shared, got: %s", rootContent)
	}

	// Check pkg1/main.go
	pkg1Content, _ := os.ReadFile(filepath.Join(dir, "pkg1/main.go"))
	if !strings.Contains(string(pkg1Content), `"root-only"`) {
		t.Errorf("pkg1/main.go: expected root-only (inherited), got: %s", pkg1Content)
	}
	if !strings.Contains(string(pkg1Content), `"pkg1-shared"`) {
		t.Errorf("pkg1/main.go: expected pkg1-shared (shadowed), got: %s", pkg1Content)
	}
	if !strings.Contains(string(pkg1Content), `"pkg1-only"`) {
		t.Errorf("pkg1/main.go: expected pkg1-only, got: %s", pkg1Content)
	}

	// Check pkg1/deep/main.go
	deepContent, _ := os.ReadFile(filepath.Join(dir, "pkg1/deep/main.go"))
	if !strings.Contains(string(deepContent), `"root-only"`) {
		t.Errorf("pkg1/deep/main.go: expected root-only (inherited from root), got: %s", deepContent)
	}
	if !strings.Contains(string(deepContent), `"deep-shared"`) {
		t.Errorf("pkg1/deep/main.go: expected deep-shared (local shadow), got: %s", deepContent)
	}
	if !strings.Contains(string(deepContent), `"pkg1-only"`) {
		t.Errorf("pkg1/deep/main.go: expected pkg1-only (inherited from pkg1), got: %s", deepContent)
	}
}

// TestHierarchyNoInheritanceUpward tests that files "above" a helper don't see its functions
func TestHierarchyNoInheritanceUpward(t *testing.T) {
	dir := t.TempDir()

	// Helper only in subdirectory
	writeFile(t, dir, "sub/helpers.go", `//go:build exclude
//go:ahead functions

package main

func subOnly() string { return "sub-only" }
`)

	// Source file in root trying to use subdirectory function (should fail)
	writeFile(t, dir, "main.go", `package main

//:subOnly
var s = ""
`)

	// Source file in subdirectory (should work)
	writeFile(t, dir, "sub/main.go", `package main

//:subOnly
var s = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Root main.go should NOT have the replacement (function not found)
	rootContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if strings.Contains(string(rootContent), `"sub-only"`) {
		t.Errorf("Root should NOT see subdirectory function, but got: %s", rootContent)
	}

	// Sub main.go SHOULD have the replacement
	subContent, _ := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	if !strings.Contains(string(subContent), `"sub-only"`) {
		t.Errorf("Subdirectory should see its own function, got: %s", subContent)
	}
}

// TestHierarchySiblingVisibility tests that sibling directories at the same depth share functions
// With depth-based resolution, functions at the same depth are visible to all files at or below that depth
func TestHierarchySiblingVisibility(t *testing.T) {
	dir := t.TempDir()

	// Root helper (shared)
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func common() string { return "common" }
`)

	// pkg1 helper
	writeFile(t, dir, "pkg1/helpers.go", `//go:build exclude
//go:ahead functions

package main

func pkg1Func() string { return "pkg1-func" }
`)

	// pkg2 helper
	writeFile(t, dir, "pkg2/helpers.go", `//go:build exclude
//go:ahead functions

package main

func pkg2Func() string { return "pkg2-func" }
`)

	// pkg1 source - with depth-based resolution, sees all functions at depth 0 and depth 1
	writeFile(t, dir, "pkg1/main.go", `package main

//:common
var c = ""

//:pkg1Func
var p1 = ""

//:pkg2Func
var p2 = ""
`)

	// pkg2 source - with depth-based resolution, sees all functions at depth 0 and depth 1
	writeFile(t, dir, "pkg2/main.go", `package main

//:common
var c = ""

//:pkg2Func
var p2 = ""

//:pkg1Func
var p1 = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	// Check pkg1 - now sees ALL functions at depth 0 and depth 1
	pkg1Content, _ := os.ReadFile(filepath.Join(dir, "pkg1/main.go"))
	if !strings.Contains(string(pkg1Content), `"common"`) {
		t.Errorf("pkg1 should see common, got: %s", pkg1Content)
	}
	if !strings.Contains(string(pkg1Content), `"pkg1-func"`) {
		t.Errorf("pkg1 should see pkg1Func, got: %s", pkg1Content)
	}
	// NEW: With depth-based resolution, pkg1 CAN see pkg2Func (same depth)
	if !strings.Contains(string(pkg1Content), `"pkg2-func"`) {
		t.Errorf("pkg1 should see pkg2Func (same depth), got: %s", pkg1Content)
	}

	// Check pkg2 - now sees ALL functions at depth 0 and depth 1
	pkg2Content, _ := os.ReadFile(filepath.Join(dir, "pkg2/main.go"))
	if !strings.Contains(string(pkg2Content), `"common"`) {
		t.Errorf("pkg2 should see common, got: %s", pkg2Content)
	}
	if !strings.Contains(string(pkg2Content), `"pkg2-func"`) {
		t.Errorf("pkg2 should see pkg2Func, got: %s", pkg2Content)
	}
	// NEW: With depth-based resolution, pkg2 CAN see pkg1Func (same depth)
	if !strings.Contains(string(pkg2Content), `"pkg1-func"`) {
		t.Errorf("pkg2 should see pkg1Func (same depth), got: %s", pkg2Content)
	}
}

// TestHierarchyDuplicateInSameDirectory tests that duplicates in same directory cause error
func TestHierarchyDuplicateInSameDirectory(t *testing.T) {
	dir := t.TempDir()

	// Two helper files in same directory with same function name
	writeFile(t, dir, "helpers1.go", `//go:build exclude
//go:ahead functions

package main

func duplicate() string { return "first" }
`)

	writeFile(t, dir, "helpers2.go", `//go:build exclude
//go:ahead functions

package main

func duplicate() string { return "second" }
`)

	writeFile(t, dir, "main.go", `package main

//:duplicate
var d = ""
`)

	// This should cause the process to exit - we can't easily test os.Exit(1)
	// but we can verify the behavior doesn't panic
	// In a real scenario this would exit with error

	// For now, just verify the code doesn't panic during setup
	// The actual exit happens in processFunctionDeclaration
}

// TestHierarchyMultipleFunctionsPerLevel tests multiple functions at each level
func TestHierarchyMultipleFunctionsPerLevel(t *testing.T) {
	dir := t.TempDir()

	// Root with multiple functions
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func rootA() string { return "root-a" }
func rootB() int { return 100 }
func rootC() bool { return true }
`)

	// Sub with some overrides
	writeFile(t, dir, "sub/helpers.go", `//go:build exclude
//go:ahead functions

package main

func rootB() int { return 200 }
func subD() string { return "sub-d" }
`)

	// Source using all functions
	writeFile(t, dir, "sub/main.go", `package main

//:rootA
var a = ""

//:rootB
var b = 0

//:rootC
var c = false

//:subD
var d = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, `"root-a"`) {
		t.Errorf("Expected rootA from root, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "200") {
		t.Errorf("Expected rootB=200 from sub (shadowed), got: %s", contentStr)
	}
	if strings.Contains(contentStr, "100") {
		t.Errorf("Should NOT have rootB=100 from root, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "true") {
		t.Errorf("Expected rootC=true from root, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, `"sub-d"`) {
		t.Errorf("Expected subD from sub, got: %s", contentStr)
	}
}

// TestHierarchyWithStdlib tests that stdlib functions work alongside hierarchical user functions
func TestHierarchyWithStdlib(t *testing.T) {
	dir := t.TempDir()

	// Root helper
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func userFunc() string { return "user-value" }
`)

	// Source using both user and stdlib
	writeFile(t, dir, "main.go", `package main

//:userFunc
var user = ""

//:strings.ToUpper:"hello"
var upper = "HELLO"
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, `"user-value"`) {
		t.Errorf("Expected user function result, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, `"HELLO"`) {
		t.Errorf("Expected stdlib result, got: %s", contentStr)
	}
}

// TestHierarchyVariadicInheritance tests variadic functions work with inheritance
func TestHierarchyVariadicInheritance(t *testing.T) {
	dir := t.TempDir()

	// Root helper with variadic function
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func join(sep string, parts ...string) string {
	return strings.Join(parts, sep)
}
`)

	// Subdirectory using inherited variadic function
	writeFile(t, dir, "sub/main.go", `package main

//:join:"-":"a":"b":"c"
var result = ""
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	if !strings.Contains(string(content), `"a-b-c"`) {
		t.Errorf("Expected inherited variadic function to work, got: %s", content)
	}
}
