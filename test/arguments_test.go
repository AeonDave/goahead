package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestArgumentParsingCornerCases tests argument parsing edge cases
func TestArgumentParsingCornerCases(t *testing.T) {
	t.Run("EscapedQuotesInString", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func echo(s string) string { return s }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:echo:"hello \"world\""
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, `hello \"world\"`) && !strings.Contains(got, `hello "world"`) {
			t.Fatalf("escaped quotes not preserved\n%s", got)
		}
	})

	t.Run("BacktickString", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func echo(s string) string { return s }
`)
		writeFile(t, dir, "main.go", "package main\n\nvar (\n    //:echo:`raw string`\n    val = \"\"\n)\n\nfunc main() {}\n")
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("EmptyStringArgument", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func echo(s string) string { 
    if s == "" {
        return "empty"
    }
    return s 
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:echo:""
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `val = "empty"`) {
			t.Fatalf("empty string arg not handled correctly\n%s", string(content))
		}
	})

	t.Run("HexadecimalNumber", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func addOne(n int) int { return n + 1 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:addOne:0xFF
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		// 0xFF = 255, +1 = 256
		if !strings.Contains(string(content), "256") && !strings.Contains(string(content), "0x100") {
			t.Fatalf("hex number not handled\n%s", string(content))
		}
	})

	t.Run("OctalNumber", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func double(n int) int { return n * 2 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:double:0o10
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		// 0o10 = 8, *2 = 16
		if !strings.Contains(string(content), "16") {
			t.Fatalf("octal number not handled\n%s", string(content))
		}
	})

	t.Run("BinaryNumber", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func identity(n int) int { return n }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:identity:0b1010
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		// 0b1010 = 10
		if !strings.Contains(string(content), "10") {
			t.Fatalf("binary number not handled\n%s", string(content))
		}
	})

	t.Run("NegativeFloat", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func negate(f float64) float64 { return -f }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:negate:-3.14
    val = 0.0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		// -(-3.14) = 3.14
		if !strings.Contains(string(content), "3.14") {
			t.Fatalf("negative float not handled\n%s", string(content))
		}
	})

	t.Run("ScientificNotation", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func identity(f float64) float64 { return f }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:identity:1.5e10
    val = 0.0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("BooleanValues", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func negate(b bool) bool { return !b }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:negate:true
    a = false
    //:negate:false
    b = false
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, "a = false") || !strings.Contains(got, "b = true") {
			t.Fatalf("boolean processing failed\n%s", got)
		}
	})
}
