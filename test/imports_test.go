package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestImportCornerCases tests import alias and stdlib resolution
func TestImportCornerCases(t *testing.T) {
	t.Run("MultipleImportAliases", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
//go:ahead import str=strings
//go:ahead import fmt2=fmt
//go:ahead import os2=os

package main
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:str.ToUpper:"hello"
    upper = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `upper = "HELLO"`) {
			t.Fatalf("aliased import not working\n%s", string(content))
		}
	})

	t.Run("ImportAliasWithSpecialPath", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
//go:ahead import nethttp=net/http

package main
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:nethttp.StatusText:200
    statusText = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `statusText = "OK"`) {
			t.Fatalf("nested package alias not working\n%s", string(content))
		}
	})

	t.Run("StdlibWithoutAlias", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:strings.Repeat:"ab":3
    repeated = ""
    //:strings.Count:"hello":"l"
    count = 0
    //:strings.Contains:"hello":"ell"
    contains = false
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, `repeated = "ababab"`) {
			t.Fatalf("strings.Repeat not working\n%s", got)
		}
		if !strings.Contains(got, `count = 2`) {
			t.Fatalf("strings.Count not working\n%s", got)
		}
		if !strings.Contains(got, `contains = true`) {
			t.Fatalf("strings.Contains not working\n%s", got)
		}
	})

	t.Run("PathPackage", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:path.Base:"/a/b/c.txt"
    base = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `base = "c.txt"`) {
			t.Fatalf("path.Base not working\n%s", string(content))
		}
	})
}
