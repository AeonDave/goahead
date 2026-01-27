package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestGrammarCornerCases tests various grammatical variations of placeholders
func TestGrammarCornerCases(t *testing.T) {
	t.Run("PlaceholderAtEndOfFile", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func lastFunc() string { return "last" }
`)
		writeFile(t, dir, "main.go", `package main

//:lastFunc`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("ConsecutivePlaceholders", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func First() string { return "1st" }
func Second() string { return "2nd" }
func Third() string { return "3rd" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:First
    a = ""
    //:Second
    b = ""
    //:Third
    c = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, `a = "1st"`) || !strings.Contains(got, `b = "2nd"`) || !strings.Contains(got, `c = "3rd"`) {
			t.Fatalf("consecutive placeholders not all replaced\n%s", got)
		}
	})

	t.Run("PlaceholderWithSpaces", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func SpaceTest() string { return "space" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //   :SpaceTest
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("PlaceholderWithTab", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func TabTest() string { return "tab" }
`)
		content := "package main\n\nvar (\n\t//\t:TabTest\n\tval = \"\"\n)\n\nfunc main() {}\n"
		writeFile(t, dir, "main.go", content)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("FunctionNameWithNumbers", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Get123Value() string { return "123" }
func GetValue456() string { return "456" }
func V7() int { return 7 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Get123Value
    a = ""
    //:GetValue456
    b = ""
    //:V7
    c = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, `a = "123"`) || !strings.Contains(got, `b = "456"`) || !strings.Contains(got, `c = 7`) {
			t.Fatalf("function names with numbers not handled\n%s", got)
		}
	})

	t.Run("FunctionNameWithUnderscore", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Get_value() string { return "underscore" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Get_value
    a = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `a = "underscore"`) {
			t.Fatalf("underscore function name not handled\n%s", string(content))
		}
	})

	t.Run("EmptyArgumentList", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func NoArgs() string { return "no args" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:NoArgs:
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("VeryLongFunctionName", func(t *testing.T) {
		dir := t.TempDir()
		longName := "ThisIsAVeryLongFunctionNameThatExceedsNormalExpectationsForReadability"
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func `+longName+`() string { return "long" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:`+longName+`
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `val = "long"`) {
			t.Fatalf("long function name not handled")
		}
	})
}
