package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestExpressionCornerCases tests complex expression handling
func TestExpressionCornerCases(t *testing.T) {
	t.Run("ExpressionWithNestedParens", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func Process(s string) string { return strings.ToUpper(s) }
`)
		writeFile(t, dir, "main.go", `package main

import "strings"

var (
    //:Process:=strings.TrimSpace(strings.ToLower("  HELLO  "))
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `"HELLO"`) {
			t.Fatalf("nested parens expression not handled\n%s", string(content))
		}
	})

	t.Run("ExpressionWithSlice", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main
`)
		// Full PNG header (first 8 bytes) for proper detection
		writeFile(t, dir, "main.go", `package main

var (
    //:http.DetectContentType:=[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
    mime = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "image/png") {
			t.Fatalf("slice expression not handled\n%s", string(content))
		}
	})

	t.Run("ExpressionWithMap", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetMapLen() int { 
    m := map[string]int{"a": 1, "b": 2}
    return len(m) 
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetMapLen
    count = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "count = 2") {
			t.Fatalf("map in helper not working\n%s", string(content))
		}
	})

	t.Run("ExpressionWithStruct", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

type Point struct { X, Y int }

func GetPointSum() int { 
    p := Point{X: 10, Y: 20}
    return p.X + p.Y 
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetPointSum
    sum = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "sum = 30") {
			t.Fatalf("struct in helper not working\n%s", string(content))
		}
	})
}

// TestReplacementCornerCases tests literal replacement patterns
func TestReplacementCornerCases(t *testing.T) {
	t.Run("MultipleZerosOnLine", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetValue() int { return 42 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetValue
    val = 0 + 0 + 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "val = 42 + 0 + 0") {
			t.Fatalf("multiple zeros not handled correctly\n%s", string(content))
		}
	})

	t.Run("MultipleEmptyStringsOnLine", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetValue() string { return "replaced" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetValue
    val = "" + "" + ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `val = "replaced" + "" + ""`) {
			t.Fatalf("multiple empty strings not handled correctly\n%s", string(content))
		}
	})

	t.Run("BoolInComplexExpression", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func IsEnabled() bool { return true }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:IsEnabled
    val = false && true || false
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "val = true && true || false") {
			t.Fatalf("bool in complex expression not handled\n%s", string(content))
		}
	})

	t.Run("PreserveIndentation", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetValue() string { return "indented" }
`)
		writeFile(t, dir, "main.go", `package main

func main() {
    if true {
        if true {
            //:GetValue
            deeplyIndented := ""
            _ = deeplyIndented
        }
    }
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `            deeplyIndented := "indented"`) {
			t.Fatalf("indentation not preserved\n%s", string(content))
		}
	})
}
