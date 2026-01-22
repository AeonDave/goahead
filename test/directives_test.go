package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestGoDirectiveCornerCases tests preservation of Go directives
func TestGoDirectiveCornerCases(t *testing.T) {
	t.Run("PreserveGoGenerate", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getGeneratedValue() string { return "generated" }
`)
		writeFile(t, dir, "main.go", `package main

//go:generate echo "Hello from go generate"
//go:generate go run ./gen/...

var (
    //:getGeneratedValue
    val = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `//go:generate echo`) {
			t.Fatalf("go:generate directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `//go:generate go run`) {
			t.Fatalf("second go:generate directive not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveGoEmbed", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getEmbedName() string { return "config.json" }
`)
		writeFile(t, dir, "main.go", `package main

import (
    _ "embed"
)

//go:embed config.json
var configData string

//go:embed templates/*
var templateDir embed.FS

var (
    //:getEmbedName
    embedFile = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `//go:embed config.json`) {
			t.Fatalf("go:embed directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `//go:embed templates/*`) {
			t.Fatalf("go:embed with wildcard not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveCompilerDirectives", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getOptLevel() int { return 3 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:getOptLevel
    optLevel = 0
)

//go:noinline
func criticalFunction() int {
    return optLevel * 2
}

//go:nosplit
func lowLevelFunc() {}

//go:norace
func raceExempt() {}

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `//go:noinline`) {
			t.Fatalf("go:noinline directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `//go:nosplit`) {
			t.Fatalf("go:nosplit directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `//go:norace`) {
			t.Fatalf("go:norace directive not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveGoLinkname", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getLinkTarget() string { return "runtime.throw" }
`)
		writeFile(t, dir, "main.go", `package main

import (
    _ "unsafe"
)

//go:linkname runtimeThrow runtime.throw
func runtimeThrow(s string)

var (
    //:getLinkTarget
    linkTarget = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `//go:linkname runtimeThrow runtime.throw`) {
			t.Fatalf("go:linkname directive not preserved\n%s", string(content))
		}
	})

	t.Run("PreserveBuildConstraints", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getPlatform() string { return "windows" }
`)
		writeFile(t, dir, "platform_windows.go", `//go:build windows && amd64
// +build windows,amd64

package main

var (
    //:getPlatform
    platform = ""
)
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "platform_windows.go"))
		if !strings.Contains(string(content), `//go:build windows && amd64`) {
			t.Fatalf("go:build directive not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `// +build windows,amd64`) {
			t.Fatalf("legacy +build directive not preserved\n%s", string(content))
		}
	})

	t.Run("MixedGoAheadAndGoDirectives", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func getMsg() string { return "hello" }
`)
		writeFile(t, dir, "main.go", `package main

//go:generate stringer -type=MyType

var (
    //:getMsg
    msg = ""
    //:strings.ToUpper:"world"
    upperWorld = "WORLD"

//go:noinline
func process() string {
    return msg
}

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), `//go:generate stringer`) {
			t.Fatalf("go:generate not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `//go:noinline`) {
			t.Fatalf("go:noinline not preserved\n%s", string(content))
		}
		if !strings.Contains(string(content), `msg = "hello"`) {
			t.Fatalf("local function placeholder not replaced\n%s", string(content))
		}
		if !strings.Contains(string(content), `upperWorld = "WORLD"`) {
			t.Fatalf("stdlib function placeholder not replaced\n%s", string(content))
		}
	})
}
