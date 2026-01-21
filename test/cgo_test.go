package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestCgoCornerCases tests CGO compatibility
func TestCgoCornerCases(t *testing.T) {
	t.Run("FileWithCgoImport", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getVersion() string { return "1.0.0" }
`)
		writeFile(t, dir, "main.go", `package main

/*
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

import "fmt"

var (
    //:getVersion
    version = ""
)

func main() {
    fmt.Println(version)
    C.puts(C.CString("Hello from C"))
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `#include <stdlib.h>`) {
			t.Fatalf("cgo header not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `import "C"`) {
			t.Fatalf("cgo import not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `version = "1.0.0"`) {
			t.Fatalf("placeholder not replaced\n%s", string(content))
		}
	})

	t.Run("FileWithCgoDirectives", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getLibPath() string { return "/usr/local/lib" }
`)
		writeFile(t, dir, "main.go", `package main

/*
#cgo LDFLAGS: -L/usr/local/lib -lsomelib
#cgo CFLAGS: -I/usr/local/include
#include <someheader.h>
*/
import "C"

import "fmt"

var (
    //:getLibPath
    libPath = ""
)

func main() {
    fmt.Println(libPath)
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `#cgo LDFLAGS`) {
			t.Fatalf("cgo LDFLAGS directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `#cgo CFLAGS`) {
			t.Fatalf("cgo CFLAGS directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `libPath = "/usr/local/lib"`) {
			t.Fatalf("placeholder not replaced\n%s", string(content))
		}
	})

	t.Run("MixedCgoAndGoAheadMarkers", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getCConfig() string { return "debug" }
`)
		writeFile(t, dir, "main.go", `package main

/*
#include <stdio.h>

// This is a C comment with colons: test:test:test
static void helper() {
    printf("Hello\n");
}
*/
import "C"

var (
    //:getCConfig
    config = ""
)

func main() {
    C.helper()
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "test:test:test") {
			t.Fatalf("C comment with colons was modified\n%s", string(content))
		}
		if !strings.Contains(string(content), `config = "debug"`) {
			t.Fatalf("placeholder not replaced\n%s", string(content))
		}
	})
}
