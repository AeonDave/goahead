package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestCrossPlatformPaths tests cross-platform path handling
func TestCrossPlatformPaths(t *testing.T) {
	t.Run("WindowsStylePaths", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getWindowsPath() string { return "C:\\Users\\test\\file.txt" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:getWindowsPath
    path = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `C:\\Users\\test\\file.txt`) {
			t.Fatalf("Windows path not correctly escaped\n%s", string(content))
		}
	})

	t.Run("UnixStylePaths", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getUnixPath() string { return "/usr/local/bin/app" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:getUnixPath
    path = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `/usr/local/bin/app`) {
			t.Fatalf("Unix path not correctly handled\n%s", string(content))
		}
	})

	t.Run("PathWithSpaces", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getSpacePath() string { return "C:\\Program Files\\My App\\config.txt" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:getSpacePath
    path = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `Program Files`) {
			t.Fatalf("path with spaces not correctly handled\n%s", string(content))
		}
	})
}

// TestContextPreservation tests that context is preserved during processing
func TestContextPreservation(t *testing.T) {
	t.Run("PreservePackageComments", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getValue() string { return "value" }
`)
		writeFile(t, dir, "main.go", `// Package main provides the main entry point.
// This is a multi-line package comment
// that should be preserved.
package main

var (
    //:getValue
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "Package main provides") {
			t.Fatalf("package comments not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveBuildTags", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getValue() string { return "value" }
`)
		writeFile(t, dir, "main.go", `//go:build linux && amd64

package main

var (
    //:getValue
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "//go:build linux && amd64") {
			t.Fatalf("build tags not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveImportGroups", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getValue() string { return "value" }
`)
		writeFile(t, dir, "main.go", `package main

import (
    "fmt"
    "os"

    "encoding/json"
)

var (
    //:getValue
    val = ""
)

func main() {
    fmt.Println(val)
    _ = os.Args
    _ = json.Marshal
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "\"fmt\"") || !strings.Contains(string(content), "\"encoding/json\"") {
			t.Fatalf("import structure not preserved\n%s", string(content))
		}
	})
}
