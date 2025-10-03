package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

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

func TestRunCodegenIntegration(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
//go:ahead import http=net/http

package main

import "strings"

func greet(name string) string {
    return "Hello, " + strings.ToUpper(name)
}

func add(a, b int) int {
    return a + b
}

func flagValue() bool {
    return true
}
`)

	writeFile(t, dir, "main.go", `package main

import "fmt"

var (
    //:greet:"gopher"
    welcome = ""

    //:add:19:23
    total = 0

    //:greet:=strings.TrimSpace(" gopher ")
    fancy = ""

    //:http.DetectContentType:=[]byte("plain text")
    mime = ""
)

var ready = false

func init() {
    //:flagValue
    ready = false
}

func main() {
    fmt.Println(welcome, total, fancy, mime, ready)
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
}
