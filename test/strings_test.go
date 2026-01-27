package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestSpecialStringContent tests handling of special string content
func TestSpecialStringContent(t *testing.T) {
	t.Run("UnicodeString", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetEmoji() string { return "Hello 🌍🚀✨" }
func GetChinese() string { return "你好世界" }
func GetArabic() string { return "مرحبا" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetEmoji
    emoji = ""
    //:GetChinese
    chinese = ""
    //:GetArabic
    arabic = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		got := string(content)
		if !strings.Contains(got, "🌍") {
			t.Fatalf("emoji not preserved\n%s", got)
		}
		if !strings.Contains(got, "你好") {
			t.Fatalf("chinese not preserved\n%s", got)
		}
	})

	t.Run("NewlinesAndTabs", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetMultiline() string { return "line1\nline2\nline3" }
func GetTabs() string { return "col1\tcol2\tcol3" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetMultiline
    multi = ""
    //:GetTabs
    tabs = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("VeryLongString", func(t *testing.T) {
		dir := t.TempDir()
		longString := strings.Repeat("a", 10000)
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func GetLongString() string { return strings.Repeat("a", 10000) }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetLongString
    long = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), longString[:100]) {
			t.Fatalf("long string not handled\n")
		}
	})

	t.Run("SQLInjectionLikeString", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetSQLish() string { return "'; DROP TABLE users; --" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetSQLish
    sql = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "DROP TABLE") {
			t.Fatalf("SQL-like string not preserved\n%s", string(content))
		}
	})

	t.Run("JSONString", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetJSON() string { 
    return "{\"name\": \"John\", \"age\": 30, \"nested\": {\"key\": \"value\"}}" 
}
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:GetJSON
    jsonStr = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		if !strings.Contains(string(content), "nested") {
			t.Fatalf("JSON string not preserved\n%s", string(content))
		}
	})
}
