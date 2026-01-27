package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "github.com/AeonDave/goahead/internal"
)

func TestRunCodegenLiteralReplacements(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func MakeGreeting(name string) string {
    return "Hello, " + strings.ToUpper(strings.TrimSpace(name))
}

func Sum(a, b int) int { return a + b }

func Pi() float64 { return 3.1415 }

func Flag() bool { return true }
`)

	writeFile(t, dir, "main.go", `package main

import (
    "fmt"
    _ "strings"
)

var (
    //:MakeGreeting:"gopher"
    greeting = ""

    //:Sum:19:23
    result = 0

    //:Pi
    circumference = 0.0

    //:Flag
    ready = false

    //:MakeGreeting:=strings.TrimSpace(" dev ")
    trimmed = ""
)

func main() {
    //:MakeGreeting:"team"
    fmt.Println(MakeGreeting("team"))
    fmt.Println(greeting, result, circumference, ready, trimmed)
}

func MakeGreeting(name string) string { return "" }
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	expectations := []string{
		`greeting = "Hello, GOPHER"`,
		`result = 42`,
		`circumference = 3.1415`,
		`ready = true`,
		`trimmed = "Hello, DEV"`,
		`fmt.Println("Hello, TEAM")`,
	}

	for _, want := range expectations {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n---- got ----\n%s", want, got)
		}
	}

	// Verify generated code compiles
	verifyCompiles(t, got)
}

func TestRunCodegenSkipsBlankLinesAfterComment(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func MakeGreeting(name string) string { return "Hello, " + name }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:MakeGreeting:"gopher"

    greeting = ""
)
`)

	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	expected := "//:MakeGreeting:\"gopher\"\n\n    greeting = \"Hello, gopher\""
	if !strings.Contains(got, expected) {
		t.Fatalf("output missing expected block\nwant:\n%s\n---- got ----\n%s", expected, got)
	}
}

func TestRunCodegenHandlesMultiReturnFunctions(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func FetchValue() (string, error) { return "multi", nil }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:FetchValue
    result = ""
)
`)

	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, `result = "multi"`) {
		t.Fatalf("result assignment not replaced\n---- got ----\n%s", got)
	}
}
