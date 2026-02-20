package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "github.com/AeonDave/goahead/internal"
)

func TestIsGoCleanupError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "empty stderr",
			stderr: "",
			want:   false,
		},
		{
			name:   "only whitespace",
			stderr: "   \n  \n",
			want:   false,
		},
		{
			name:   "single unlinkat line",
			stderr: `go: unlinkat C:\Users\ninja\AppData\Local\Temp\go-build123\b001\exe\goahead_eval.exe: Access is denied.` + "\n",
			want:   true,
		},
		{
			name:   "multiple unlinkat lines",
			stderr: "go: unlinkat /tmp/go-build1/a.exe: Access is denied.\ngo: unlinkat /tmp/go-build1/b.exe: Access is denied.\n",
			want:   true,
		},
		{
			name:   "removing variant",
			stderr: "go: removing /tmp/go-build1/a.exe: Access is denied.\n",
			want:   true,
		},
		{
			name:   "mixed cleanup variants",
			stderr: "go: unlinkat /tmp/a.exe: Access is denied.\ngo: removing /tmp/b.exe: Access is denied.\n",
			want:   true,
		},
		{
			name:   "real compilation error",
			stderr: "./main.go:5:2: undefined: foo\n",
			want:   false,
		},
		{
			name:   "cleanup plus compilation error",
			stderr: "go: unlinkat /tmp/a.exe: Access is denied.\n./main.go:5:2: undefined: foo\n",
			want:   false,
		},
		{
			name:   "unrelated go error",
			stderr: "go: cannot find main module\n",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := internal.IsGoCleanupError(tt.stderr)
			if got != tt.want {
				t.Errorf("IsGoCleanupError(%q) = %v, want %v", tt.stderr, got, tt.want)
			}
		})
	}
}

func TestExecuteProgramWindowsCleanupRecovery(t *testing.T) {
	// Integration test: verifies the execution path works correctly
	// with a simple helper function. Also validates that the stdout/stderr
	// separation in executeProgram doesn't break normal execution.
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "fmt"

func Echo(s string) string {
	return fmt.Sprintf("echo:%s", s)
}
`)
	writeFile(t, dir, "main.go", `package main

//:Echo:"hello"
var x = ""

func main() {}
`)

	if err := internal.RunCodegen(dir, false); err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	result := string(content)

	verifyCompiles(t, result)

	if !strings.Contains(result, `x = "echo:hello"`) {
		t.Errorf("expected Echo result in output, got:\n%s", result)
	}
}
