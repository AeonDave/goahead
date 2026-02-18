package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

func TestInjectionWhitespaceStableAcrossRuns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "go.mod", "module testmodule\ngo 1.21\n")

	// Helper implementing the injected method
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func ShadowRuntime(seedHex string, input string) string {
	if input == "" {
		return ""
	}
	return seedHex + input
}
`)

	// Target file with inject marker and interface
	writeFile(t, dir, "main.go", `package main

// :inject:ShadowRuntime
type RuntimeDecoder interface {
	ShadowRuntime(seedHex string, input string) string
}

func main() {}
`)

	// First run
	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen (1st) failed: %v", err)
	}
	c1, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go after 1st run: %v", err)
	}

	// Second run
	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen (2nd) failed: %v", err)
	}
	c2, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go after 2nd run: %v", err)
	}

	if string(c1) != string(c2) {
		t.Fatalf("expected idempotent reinjection; file changed across runs\n--- after 1st ---\n%s\n--- after 2nd ---\n%s", string(c1), string(c2))
	}

	got := string(c2)
	endMarker := "// End of goahead generated code."

	// No blank line immediately before the end marker
	if strings.Contains(got, "\n\n"+endMarker) {
		t.Fatalf("unexpected blank line before end marker\n%s", got)
	}

	// Exactly one blank line after the end marker
	if !strings.Contains(got, endMarker+"\n\n") {
		t.Fatalf("expected exactly one blank line after end marker\n%s", got)
	}
	if strings.Contains(got, endMarker+"\n\n\n") {
		t.Fatalf("unexpected extra blank line after end marker\n%s", got)
	}
}
