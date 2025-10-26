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

func makeGreeting(name string) string {
    return "Hello, " + strings.ToUpper(strings.TrimSpace(name))
}

func sum(a, b int) int { return a + b }

func pi() float64 { return 3.1415 }

func flag() bool { return true }
`)

	writeFile(t, dir, "main.go", `package main

import (
    "fmt"
    "strings"
)

var (
    //:makeGreeting:"gopher"
    greeting = ""

    //:sum:19:23
    result = 0

    //:pi
    circumference = 0.0

    //:flag
    ready = false

    //:makeGreeting:=strings.TrimSpace(" dev ")
    trimmed = ""
)

func main() {
    //:makeGreeting:"team"
    fmt.Println(makeGreeting("team"))
    fmt.Println(greeting, result, circumference, ready, trimmed)
}
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
}

func TestRunCodegenSkipsBlankLinesAfterComment(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func makeGreeting(name string) string { return "Hello, " + name }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:makeGreeting:"gopher"

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

	expected := "//:makeGreeting:\"gopher\"\n\n    greeting = \"Hello, gopher\""
	if !strings.Contains(got, expected) {
		t.Fatalf("output missing expected block\nwant:\n%s\n---- got ----\n%s", expected, got)
	}
}

func TestRunCodegenHandlesMultiReturnFunctions(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func fetchValue() (string, error) { return "multi", nil }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:fetchValue
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

func TestRunCodegenFailsWhenHelperReturnsError(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "errors"

func fetchValue() (string, error) { return "", errors.New("boom") }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:fetchValue
    result = ""
)
`)

	err := internal.RunCodegen(dir, false)
	if err == nil {
		t.Fatalf("expected RunCodegen to fail when helper returns non-nil error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected helper error to propagate, got: %v", err)
	}
}
