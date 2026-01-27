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

func Echo(s string) string { return s }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Echo:"hello \"world\""
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

func Echo(s string) string { return s }
`)
		writeFile(t, dir, "main.go", "package main\n\nvar (\n    //:Echo:`raw string`\n    val = \"\"\n)\n\nfunc main() {}\n")
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

func Echo(s string) string { 
    if s == "" {
        return "empty"
    }
    return s 
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Echo:""
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

func AddOne(n int) int { return n + 1 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:AddOne:0xFF
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

func Double(n int) int { return n * 2 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Double:0o10
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

func Identity(n int) int { return n }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Identity:0b1010
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

func Negate(f float64) float64 { return -f }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Negate:-3.14
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

func Identity(f float64) float64 { return f }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Identity:1.5e10
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

func Negate(b bool) bool { return !b }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Negate:true
    a = false
    //:Negate:false
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

	t.Run("SpaceAfterSlashes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func AddOne(n int) int { return n + 1 }
`)
		// Note: "// :" with space - common after gofmt
		writeFile(t, dir, "main.go", `package main

var (
    // :AddOne:10
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "val = 11") {
			t.Fatalf("space after // not handled\n%s", string(content))
		}
	})

	t.Run("MultipleSpacesAfterSlashes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Double(n int) int { return n * 2 }
`)
		// Multiple spaces after //
		writeFile(t, dir, "main.go", `package main

var (
    //  :Double:5
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "val = 10") {
			t.Fatalf("multiple spaces after // not handled\n%s", string(content))
		}
	})
}
