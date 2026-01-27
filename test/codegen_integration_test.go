package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

func TestRunCodegenIntegration(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func Greet(name string) string {
    return "Hello, " + strings.ToUpper(name)
}

func Add(a, b int) int {
    return a + b
}

func FlagValue() bool {
    return true
}
`)

	writeFile(t, dir, "main.go", `package main

import "fmt"

var (
    //:Greet:"gopher"
    welcome = ""

    //:Add:19:23
    total = 0

    //:Greet:=strings.TrimSpace(" gopher ")
    fancy = ""

    //:http.DetectContentType:=[]byte("plain text")
    mime = ""
)

var ready = false

func init() {
    //:FlagValue
    ready = false
}

func main() {
    fmt.Println(welcome, total, fancy, mime, ready)
}
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
		`welcome = "Hello, GOPHER"`,
		`total = 42`,
		`fancy = "Hello, GOPHER"`,
		`mime = "text/plain; charset=utf-8"`,
		`ready = true`,
	}

	for _, want := range expectations {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n---- got ----\n%s", want, got)
		}
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}
