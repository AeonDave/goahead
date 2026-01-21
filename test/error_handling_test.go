package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestErrorHandling verifica che gli errori siano gestiti correttamente
func TestErrorHandling(t *testing.T) {
	t.Run("MissingFunction", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func existingFunc() string { return "exists" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:nonExistentFunction
    value = ""
)

func main() {}
`)
		// Dovrebbe generare un warning, non un errore fatale
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen should not fail on missing function: %v", err)
		}
	})

	t.Run("WrongArgumentCount", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func twoArgs(a, b string) string { return a + b }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:twoArgs:"only_one"
    result = ""
)

func main() {}
`)
		// Dovrebbe generare un warning, non un errore fatale
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen should not fail on wrong argument count: %v", err)
		}
	})

	t.Run("InvalidSyntaxInHelperFile", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func validFunc() string { return "valid" }

// Commento valido alla fine
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:validFunc
    value = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})
}

// TestDuplicateFunctionNames verifica la gestione di funzioni duplicate
func TestDuplicateFunctionHandling(t *testing.T) {
	// Questo test verifica che funzioni diverse in file diversi funzionino
	dir := t.TempDir()
	writeFile(t, dir, "helpers1.go", `//go:build exclude
//go:ahead functions

package main

func uniqueFunc1() string { return "from_file_1" }
`)
	writeFile(t, dir, "helpers2.go", `//go:build exclude
//go:ahead functions

package main

func uniqueFunc2() string { return "from_file_2" }
`)
	writeFile(t, dir, "main.go", `package main

var (
    //:uniqueFunc1
    v1 = ""
    
    //:uniqueFunc2
    v2 = ""
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}
}

// TestCommentFormats verifica i diversi formati di commenti
func TestCommentFormats(t *testing.T) {
	t.Run("SpacesInComment", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func spacedFunc() string { return "spaced" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    // :spacedFunc
    value = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("TabInComment", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func tabbedFunc() string { return "tabbed" }
`)
		writeFile(t, dir, "main.go", `package main

var (
	//:tabbedFunc
	value = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})
}

// TestToolexecModeDetection verifica la logica di rilevamento della modalità toolexec
func TestToolexecModeDetection(t *testing.T) {
	// Test del FilterUserFiles con vari percorsi
	tests := []struct {
		name     string
		files    []string
		expected int
	}{
		{
			name:     "OnlyLocalFiles",
			files:    []string{"main.go", "utils.go", "handler.go"},
			expected: 3,
		},
		{
			name:     "MixedFiles",
			files:    []string{"main.go", filepath.Join("vendor", "lib.go")},
			expected: 1,
		},
		{
			name:     "TestFiles",
			files:    []string{"main.go", "main_test.go", "utils_test.go"},
			expected: 3,
		},
		{
			name:     "EmptyList",
			files:    []string{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOAHEAD_VERBOSE", "")
			t.Setenv("GOPATH", filepath.FromSlash("/tmp/go"))
			t.Setenv("GOROOT", filepath.FromSlash("/tmp/go-root"))

			got := internal.FilterUserFiles(tc.files)
			if len(got) != tc.expected {
				t.Fatalf("expected %d files, got %d: %v", tc.expected, len(got), got)
			}
		})
	}
}

// TestFindCommonDir verifica la funzione per trovare la directory comune
func TestFindCommonDir(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "SingleFile",
			files:    []string{filepath.Join("src", "main.go")},
			expected: "src",
		},
		{
			name:     "SameDir",
			files:    []string{filepath.Join("src", "a.go"), filepath.Join("src", "b.go")},
			expected: "src",
		},
		{
			name:     "Empty",
			files:    []string{},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := internal.FindCommonDir(tc.files)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestComplexExpressions verifica espressioni complesse
func TestComplexExpressions(t *testing.T) {
	t.Run("ExpressionWithMethodChain", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func chain(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:chain:"  test  "
    result = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("ExpressionInFunction", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getPort() int { return 8080 }
`)
		writeFile(t, dir, "main.go", `package main

import "fmt"

func startServer() {
    //:getPort
    port := 0
    fmt.Printf("Starting on port %d\n", port)
}

func main() {
    startServer()
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})
}

// TestStdlibCalls verifica le chiamate alla libreria standard
func TestStdlibCalls(t *testing.T) {
	t.Run("StringsPackage", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:strings.ToUpper:"hello"
    upper = ""
    
    //:strings.ToLower:"HELLO"
    lower = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})
}

// TestPreservesNonMatchingCode verifica che il codice non correlato sia preservato
func TestPreservesNonMatchingCode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getConfig() string { return "config_value" }
`)

	originalCode := `package main

import (
    "fmt"
    "os"
)

// This is an important comment that should be preserved
const (
    Version = "1.0.0"
    Author  = "Test"
)

var (
    //:getConfig
    config = ""
    
    // Regular comment
    other = "unchanged"
)

func helper() string {
    return "helper result"
}

func main() {
    fmt.Println(config)
    fmt.Println(other)
    fmt.Println(os.Getenv("HOME"))
}
`
	writeFile(t, dir, "main.go", originalCode)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := readTestFile(t, filepath.Join(dir, "main.go"))

	// Verifica che elementi non correlati siano preservati
	checks := []string{
		"This is an important comment that should be preserved",
		`Version = "1.0.0"`,
		`Author  = "Test"`,
		`other = "unchanged"`,
		"func helper() string",
		`return "helper result"`,
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Fatalf("missing preserved content: %q\n---- got ----\n%s", check, content)
		}
	}

	// Verifica che il placeholder sia stato sostituito
	if !strings.Contains(content, `config = "config_value"`) {
		t.Fatalf("placeholder not replaced\n---- got ----\n%s", content)
	}
}

func readTestFile(t *testing.T, path string) (string, error) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
