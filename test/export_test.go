package test

import (
	"strings"
	"testing"
)

// TestUnexportedFunctionNotAvailable verifies that lowercase (unexported) functions
// cannot be used in placeholder comments
func TestUnexportedFunctionNotAvailable(t *testing.T) {
	helpers := `//go:build exclude
//go:ahead functions

package main

// unexported function (lowercase)
func echo(s string) string {
	return s
}

// Exported function (uppercase)
func Echo(s string) string {
	return echo(s)
}
`

	main := `package main

//:echo:"test"
var result = ""

func main() {
	println(result)
}
`

	dir, cleanup := setupTestDir(t, map[string]string{
		"helpers.go": helpers,
		"main.go":    main,
	})
	defer cleanup()

	result, err := processAndReplace(t, dir, "main.go")

	// Should produce a warning because echo is unexported
	// The placeholder won't be replaced
	if err != nil {
		t.Fatalf("processAndReplace failed: %v", err)
	}

	// Verify the placeholder was NOT replaced
	if !strings.Contains(result, `result = ""`) {
		t.Errorf("Expected placeholder to not be replaced for unexported function, got:\n%s", result)
	}
}

// TestExportedFunctionAvailable verifies that uppercase (exported) functions work
func TestExportedFunctionAvailable(t *testing.T) {
	helpers := `//go:build exclude
//go:ahead functions

package main

// Exported function (uppercase)
func Echo(s string) string {
	return s
}
`

	main := `package main

//:Echo:"test"
var result = ""

func main() {
	println(result)
}
`

	dir, cleanup := setupTestDir(t, map[string]string{
		"helpers.go": helpers,
		"main.go":    main,
	})
	defer cleanup()

	result, err := processAndReplace(t, dir, "main.go")
	if err != nil {
		t.Fatalf("processAndReplace failed: %v", err)
	}

	verifyCompiles(t, result)

	if !strings.Contains(result, `result = "test"`) {
		t.Errorf("Expected result = \"test\", got:\n%s", result)
	}
}

// TestUnexportedVariableNotAvailable verifies variables follow export rules
func TestUnexportedVariableNotAvailable(t *testing.T) {
	helpers := `//go:build exclude
//go:ahead functions

package main

// unexported variable
var seed = "abc123"

// Exported function using unexported variable
func GetSeed() string {
	return seed
}
`

	main := `package main

//:GetSeed:
var result = ""

func main() {
	println(result)
}
`

	dir, cleanup := setupTestDir(t, map[string]string{
		"helpers.go": helpers,
		"main.go":    main,
	})
	defer cleanup()

	result, err := processAndReplace(t, dir, "main.go")
	if err != nil {
		t.Fatalf("processAndReplace failed: %v", err)
	}

	verifyCompiles(t, result)

	// GetSeed should work (exported), even though it uses unexported seed internally
	if !strings.Contains(result, `result = "abc123"`) {
		t.Errorf("Expected result = \"abc123\", got:\n%s", result)
	}
}

// TestMixedExportUnexportFunctions verifies mixing works correctly
func TestMixedExportUnexportFunctions(t *testing.T) {
	helpers := `//go:build exclude
//go:ahead functions

package main

// unexported helper
func formatInternal(s string) string {
	return "[" + s + "]"
}

// Exported function using unexported helper
func Format(s string) string {
	return formatInternal(s)
}
`

	main := `package main

//:Format:"test"
var result = ""

func main() {
	println(result)
}
`

	dir, cleanup := setupTestDir(t, map[string]string{
		"helpers.go": helpers,
		"main.go":    main,
	})
	defer cleanup()

	result, err := processAndReplace(t, dir, "main.go")
	if err != nil {
		t.Fatalf("processAndReplace failed: %v", err)
	}

	verifyCompiles(t, result)

	// Format should work and call formatInternal internally
	if !strings.Contains(result, `result = "[test]"`) {
		t.Errorf("Expected result = \"[test]\", got:\n%s", result)
	}
}

// TestUnexportedFunctionStillExecutableInternally verifies that unexported functions
// can still be called by exported functions within helpers
func TestUnexportedFunctionStillExecutableInternally(t *testing.T) {
	helpers := `//go:build exclude
//go:ahead functions

package main

// unexported utility
func multiply(a, b int) int {
	return a * b
}

// unexported utility
func add(a, b int) int {
	return a + b
}

// Exported function using multiple unexported helpers
func Calculate(a, b, c int) int {
	return add(multiply(a, b), c)
}
`

	main := `package main

//:Calculate:2:3:4
var result = 0

func main() {
	println(result)
}
`

	dir, cleanup := setupTestDir(t, map[string]string{
		"helpers.go": helpers,
		"main.go":    main,
	})
	defer cleanup()

	result, err := processAndReplace(t, dir, "main.go")
	if err != nil {
		t.Fatalf("processAndReplace failed: %v", err)
	}

	verifyCompiles(t, result)

	// Calculate(2, 3, 4) = multiply(2, 3) + 4 = 6 + 4 = 10
	if !strings.Contains(result, "result = 10") {
		t.Errorf("Expected result = 10, got:\n%s", result)
	}
}
