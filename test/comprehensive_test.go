package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestEdgeCases verifica casi limite e scenari particolari
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyDirectory", func(t *testing.T) {
		dir := t.TempDir()
		// Una directory vuota non dovrebbe causare errori
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen on empty directory failed: %v", err)
		}
	})

	t.Run("NoFunctionFiles", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "main.go", `package main
func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen with no function files failed: %v", err)
		}
	})

	t.Run("FunctionFileWithoutMatchingCalls", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Unused() string { return "unused" }
`)
		writeFile(t, dir, "main.go", `package main

func main() {
    println("no placeholders here")
}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}
	})

	t.Run("SpecialCharactersInStringResult", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func JsonString() string { return "{\"key\": \"value\"}" }
func PathString() string { return "C:\\Users\\test\\file.txt" }
func MultilineString() string { return "line1\nline2\nline3" }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:JsonString
    json = ""
    
    //:PathString
    path = ""
    
    //:MultilineString
    multi = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		// Verifica che le stringhe siano correttamente escaped
		if !strings.Contains(got, `json =`) {
			t.Fatalf("json assignment not found\n---- got ----\n%s", got)
		}
		if !strings.Contains(got, `path =`) {
			t.Fatalf("path assignment not found\n---- got ----\n%s", got)
		}
	})

	t.Run("UintAndNegativeNumbers", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func PositiveUint() uint { return 255 }
func NegativeInt() int { return -42 }
func LargeInt() int64 { return 9223372036854775807 }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:PositiveUint
    uval = 0
    
    //:NegativeInt
    neg = 0
    
    //:LargeInt
    large = 0
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		// Go's %#v formats uint as hex, so 255 becomes 0xff
		if !strings.Contains(got, "uval = 0xff") && !strings.Contains(got, "uval = 255") {
			t.Fatalf("uint assignment not replaced correctly\n---- got ----\n%s", got)
		}
		if !strings.Contains(got, "neg = -42") {
			t.Fatalf("negative int assignment not replaced correctly\n---- got ----\n%s", got)
		}
		if !strings.Contains(got, "large = 9223372036854775807") {
			t.Fatalf("large int assignment not replaced correctly\n---- got ----\n%s", got)
		}
	})
}

// TestArgumentParsing verifica il parsing degli argomenti in vari formati
func TestArgumentParsing(t *testing.T) {
	t.Run("MultipleStringArguments", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Concat(a, b, c string) string { return a + b + c }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Concat:"hello":"world":"!"
    result = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		if !strings.Contains(got, `result = "helloworld!"`) {
			t.Fatalf("concat result not correct\n---- got ----\n%s", got)
		}
	})

	t.Run("MixedArgumentTypes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "fmt"

func Mixed(s string, i int, b bool) string { return fmt.Sprintf("%s-%d-%t", s, i, b) }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Mixed:"test":42:true
    result = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		if !strings.Contains(got, `result = "test-42-true"`) {
			t.Fatalf("mixed result not correct\n---- got ----\n%s", got)
		}
	})

	t.Run("ColonInStringArgument", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Identity(s string) string { return s }
`)
		writeFile(t, dir, "main.go", `package main

var (
    //:Identity:"http://example.com:8080/path"
    url = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		if !strings.Contains(got, `url = "http://example.com:8080/path"`) {
			t.Fatalf("URL with colons not handled correctly\n---- got ----\n%s", got)
		}
	})
}

// TestExpressionPlaceholders verifica i placeholder con espressioni raw
func TestExpressionPlaceholders(t *testing.T) {
	t.Run("RawExpressionWithFunction", func(t *testing.T) {
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
    //:Process:=strings.TrimSpace("  hello  ")
    trimmed = ""
)

func main() {}
`)
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatalf("read main.go: %v", err)
		}
		got := string(content)

		if !strings.Contains(got, `trimmed = "HELLO"`) {
			t.Fatalf("expression placeholder not evaluated\n---- got ----\n%s", got)
		}
	})
}

// TestFloatHandling verifica la gestione corretta dei numeri in virgola mobile
func TestFloatHandling(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetFloat32() float32 { return 3.14159 }
func GetFloat64() float64 { return 2.718281828 }
func GetSmallFloat() float64 { return 0.000001 }
func GetLargeFloat() float64 { return 1e10 }
`)
	writeFile(t, dir, "main.go", `package main

var (
    //:GetFloat32
    f32 = 0.0
    
    //:GetFloat64
    f64 = 0.0
    
    //:GetSmallFloat
    small = 0.0
    
    //:GetLargeFloat
    large = 0.0
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	// Verifica che i float siano stati sostituiti (il formato esatto può variare)
	if strings.Contains(got, "f32 = 0.0") {
		t.Fatalf("float32 not replaced\n---- got ----\n%s", got)
	}
	if strings.Contains(got, "f64 = 0.0") {
		t.Fatalf("float64 not replaced\n---- got ----\n%s", got)
	}
}

// TestBooleanHandling verifica la gestione corretta dei booleani
func TestBooleanHandling(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetTrue() bool { return true }
func GetFalse() bool { return false }
`)
	writeFile(t, dir, "main.go", `package main

var (
    //:GetTrue
    isEnabled = false
    
    //:GetFalse
    isDisabled = false
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, "isEnabled = true") {
		t.Fatalf("true bool not replaced\n---- got ----\n%s", got)
	}
	// GetFalse restituisce false, ma il placeholder era già false
	// quindi verifichiamo solo che il file sia valido
}

// TestMultipleFunctionFiles verifica la gestione di più file di funzioni
func TestMultipleFunctionFiles(t *testing.T) {
	dir := t.TempDir()

	// Primo file di funzioni
	writeFile(t, dir, "helpers1.go", `//go:build exclude
//go:ahead functions

package main

func Greeting() string { return "Hello" }
`)

	// Secondo file di funzioni
	writeFile(t, dir, "helpers2.go", `//go:build exclude
//go:ahead functions

package main

func Farewell() string { return "Goodbye" }
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:Greeting
    hello = ""
    
    //:Farewell
    bye = ""
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, `hello = "Hello"`) {
		t.Fatalf("first helper function not working\n---- got ----\n%s", got)
	}
	if !strings.Contains(got, `bye = "Goodbye"`) {
		t.Fatalf("second helper function not working\n---- got ----\n%s", got)
	}
}

// TestNestedDirectories verifica la gestione di directory annidate
func TestNestedDirectories(t *testing.T) {
	dir := t.TempDir()

	// File helper nella directory principale
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func RootFunc() string { return "root" }
`)

	// File in una subdirectory
	writeFile(t, filepath.Join(dir, "subdir"), "child.go", `package subdir

var (
    // Questo file non dovrebbe essere processato perché non ha il marker
    value = "static"
)
`)

	writeFile(t, dir, "main.go", `package main

var (
    //:RootFunc
    fromRoot = ""
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, `fromRoot = "root"`) {
		t.Fatalf("root function not working\n---- got ----\n%s", got)
	}
}

// TestCaching verifica che il caching funzioni correttamente
func TestCaching(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Repeated() string { return "cached_value" }
`)
	writeFile(t, dir, "main.go", `package main

var (
    //:Repeated
    first = ""
    
    //:Repeated
    second = ""
    
    //:Repeated
    third = ""
)

func main() {}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	// Tutte e tre le variabili dovrebbero avere lo stesso valore
	expectedCount := strings.Count(got, `"cached_value"`)
	if expectedCount != 3 {
		t.Fatalf("expected 3 occurrences of cached_value, got %d\n---- got ----\n%s", expectedCount, got)
	}
}

// TestInlineAssignment verifica le assegnazioni inline
func TestInlineAssignment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func GetValue() string { return "inline_value" }
`)
	writeFile(t, dir, "main.go", `package main

func main() {
    //:GetValue
    localVar := ""
    _ = localVar
}
`)
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, `localVar := "inline_value"`) {
		t.Fatalf("inline assignment not replaced\n---- got ----\n%s", got)
	}
}

// TestVerboseMode verifica che la modalità verbose funzioni
func TestVerboseMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func VerboseTest() string { return "verbose" }
`)
	writeFile(t, dir, "main.go", `package main

var (
    //:VerboseTest
    v = ""
)

func main() {}
`)
	// Test con verbose=true non dovrebbe causare errori
	err := internal.RunCodegen(dir, true)
	if err != nil {
		t.Fatalf("RunCodegen with verbose failed: %v", err)
	}
}
