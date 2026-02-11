package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestArrayLiteralsPreserveCommas verifies that placeholder replacement in array literals preserves commas
func TestArrayLiteralsPreserveCommas(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Shadow(s string) string {
	return "hashed_" + s
}
`)

	writeFile(t, dir, "main.go", `package main

var patterns = []string{
	//:Shadow:"x64dbg"
	"",
	//:Shadow:"x32dbg"
	"",
	//:Shadow:"ollydbg"
	"",
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	result := string(content)

	// Verify all values are replaced
	if !strings.Contains(result, `"hashed_x64dbg"`) {
		t.Errorf("Expected hashed_x64dbg in result.\nGot:\n%s", result)
	}
	if !strings.Contains(result, `"hashed_x32dbg"`) {
		t.Errorf("Expected hashed_x32dbg in result.\nGot:\n%s", result)
	}
	if !strings.Contains(result, `"hashed_ollydbg"`) {
		t.Errorf("Expected hashed_ollydbg in result.\nGot:\n%s", result)
	}

	// Verify commas are preserved
	expectedLines := []string{
		`"hashed_x64dbg",`,
		`"hashed_x32dbg",`,
		`"hashed_ollydbg",`,
	}

	for _, expected := range expectedLines {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected line with trailing comma: %s\nGot:\n%s", expected, result)
		}
	}

	// Verify code compiles
	verifyCompiles(t, dir)
}

// TestArrayLiteralsLastElement verifies that the last element without comma also works
func TestArrayLiteralsLastElement(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Hash(s string) string {
	return "hash_" + s
}
`)

	writeFile(t, dir, "main.go", `package main

var items = []string{
	//:Hash:"first"
	"",
	//:Hash:"last"
	"",
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	result := string(content)

	// First element should have comma
	if !strings.Contains(result, `"hash_first",`) {
		t.Errorf("Expected hash_first with comma.\nGot:\n%s", result)
	}

	// Last element should also have comma (Go allows trailing comma)
	if !strings.Contains(result, `"hash_last",`) {
		t.Errorf("Expected hash_last with comma.\nGot:\n%s", result)
	}

	verifyCompiles(t, dir)
}

// TestArrayLiteralsIdempotent verifies that running GoAhead multiple times preserves commas
// This is the critical re-run scenario: first run replaces "" with "hash",
// second run must preserve the comma when replacing "hash" with same/new value
func TestArrayLiteralsIdempotent(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Shadow(s string) string {
	return "hashed_" + s
}
`)

	writeFile(t, dir, "main.go", `package main

var patterns = []string{
	//:Shadow:"x64dbg"
	"",
	//:Shadow:"x32dbg"
	"",
	//:Shadow:"ollydbg"
	"",
}

func main() {}
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First RunCodegen failed: %v", err)
	}

	// Verify first run has commas
	content1, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	result1 := string(content1)
	for _, expected := range []string{`"hashed_x64dbg",`, `"hashed_x32dbg",`, `"hashed_ollydbg",`} {
		if !strings.Contains(result1, expected) {
			t.Fatalf("First run: expected %s\nGot:\n%s", expected, result1)
		}
	}

	// Second run (re-process the already-replaced file)
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second RunCodegen failed: %v", err)
	}

	// Verify second run STILL has commas
	content2, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	result2 := string(content2)
	for _, expected := range []string{`"hashed_x64dbg",`, `"hashed_x32dbg",`, `"hashed_ollydbg",`} {
		if !strings.Contains(result2, expected) {
			t.Errorf("Second run: comma lost! Expected %s\nGot:\n%s", expected, result2)
		}
	}

	verifyCompiles(t, dir)
}

// TestArrayLiteralsNumeric verifies numeric array elements preserve commas
func TestArrayLiteralsNumeric(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetPort(name string) int {
	ports := map[string]int{"http": 80, "https": 443, "ssh": 22}
	return ports[name]
}
`)

	writeFile(t, dir, "main.go", `package main

var ports = []int{
	//:GetPort:"http"
	0,
	//:GetPort:"https"
	0,
	//:GetPort:"ssh"
	0,
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	result := string(content)

	// Verify values with commas
	if !strings.Contains(result, "80,") {
		t.Errorf("Expected 80 with comma.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "443,") {
		t.Errorf("Expected 443 with comma.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "22,") {
		t.Errorf("Expected 22 with comma.\nGot:\n%s", result)
	}

	verifyCompiles(t, dir)
}
