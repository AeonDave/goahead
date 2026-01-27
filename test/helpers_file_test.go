package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestFunctionFileCornerCases tests helper file features
func TestFunctionFileCornerCases(t *testing.T) {
	t.Run("HelperWithImports", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import (
    "crypto/sha256"
    "encoding/hex"
    "strings"
)

func HashString(s string) string {
    h := sha256.Sum256([]byte(strings.TrimSpace(s)))
    return hex.EncodeToString(h[:])
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:HashString:"hello"
    hash = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		// SHA256 of "hello"
		if !strings.Contains(string(content), "2cf24dba") {
			t.Fatalf("hash function with imports not working\n%s", string(content))
		}
	})

	t.Run("HelperWithMultipleFunctions", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Helper1() string { return Helper2() + Helper3() }
func Helper2() string { return "hello" }
func Helper3() string { return "world" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Helper1
    combined = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `combined = "helloworld"`) {
			t.Fatalf("helper calling other helpers not working\n%s", string(content))
		}
	})

	t.Run("HelperWithTypeDefinition", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

type MyInt int

func GetMyInt() MyInt { return 42 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetMyInt
    val = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "val = 42") {
			t.Fatalf("custom type not working\n%s", string(content))
		}
	})

	t.Run("HelperWithConstants", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const Prefix = "PREFIX_"

func Prefixed(s string) string { return Prefix + s }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Prefixed:"value"
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `val = "PREFIX_value"`) {
			t.Fatalf("helper with constants not working\n%s", string(content))
		}
	})

	t.Run("HelperWithVariadicFunction", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func JoinAll(sep string, parts ...string) string {
    return strings.Join(parts, sep)
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:JoinAll:"-":"a":"b":"c"
    joined = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `joined = "a-b-c"`) {
			t.Fatalf("variadic function not working\n%s", string(content))
		}
	})

	t.Run("MarkerOnDifferentLines", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `// This is a copyright notice
// Author: Test
// License: MIT

//go:build exclude

//go:ahead functions

package main

func GetValue() string { return "value" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetValue
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `val = "value"`) {
			t.Fatalf("marker on different line not detected\n%s", string(content))
		}
	})
}
