# GoAhead → Compile-time Code Generation for Go

GoAhead is a compile-time code generation tool for Go. It replaces placeholder comments with computed values at build time.

[![CodeQL Advanced](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml/badge.svg)](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AeonDave/goahead)](https://goreportcard.com/report/github.com/AeonDave/goahead)
![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/AeonDave/goahead)
![GitHub last commit](https://img.shields.io/github/last-commit/AeonDave/goahead)
![GitHub License](https://img.shields.io/github/license/AeonDave/goahead)

![GoAhead](logo.png)

## Highlights

- Works with Go commands via `-toolexec` (build, test, run, generate)
- Placeholder replacement keeps surrounding expressions intact
- Supports parameterised helpers, raw Go expressions, and simple type inference
- Supports standard-library packages directly (strings, strconv, os, http, etc.)
- Hierarchical function inheritance: subdirectories can shadow parent functions
- Supports variadic functions, constants, type definitions, and complex expressions
- Cross-platform compatible (Linux, macOS, Windows)
- Includes examples and an integration test suite

## Installation

```bash
go install github.com/AeonDave/goahead@latest
```

## Quick Start

### 1. Declare build-time helpers

Helpers live in files marked with both build tags. They are excluded from normal compilation but executed by GoAhead.

```go
//go:build exclude
//go:ahead functions

package helpers

import "strings"

func welcome(name string) string {
    return "Hello, " + strings.ToUpper(name)
}
```

### 2. Reference helpers + build

```go
package main

//:welcome:"gopher"
var greeting = ""
```

```bash
go build -toolexec="goahead" ./...
```

---

## Placeholder Grammar Reference

### Basic Syntax

```
//:functionName[:arg1[:arg2[:argN]]]
```

The placeholder comment must appear on the line **immediately before** the target statement or expression.

### Argument Types

| Type | Syntax | Examples |
|------|--------|----------|
| **String** | Double quotes | `"hello"`, `"production"`, `""` |
| **Raw String** | Backticks | `` `raw\nstring` `` |
| **Integer** | Unquoted number | `42`, `-17`, `0xFF`, `0o755`, `0b1010` |
| **Float** | Decimal or scientific | `3.14`, `-2.5`, `1.5e10` |
| **Boolean** | Unquoted | `true`, `false`, `True`, `False` |
| **Raw Expression** | Prefix with `=` | `=strings.TrimSpace(input)` |

### Number Formats

GoAhead supports all Go number literal formats:

```go
//:process:42          // Decimal
//:process:0xFF        // Hexadecimal  
//:process:0o755       // Octal
//:process:0b1010      // Binary
//:process:-3.14       // Negative float
//:process:1.5e10      // Scientific notation
```

### Raw Expressions

Prefix an argument with `=` to pass a raw Go expression that will be evaluated at build time:

```go
//:hash:=strings.TrimSpace(input)
//:detect:=[]byte{0x89, 0x50, 0x4E, 0x47}
//:process:=map[string]int{"a": 1, "b": 2}
```

Raw expressions can contain colons (e.g., in map literals, struct literals, slice expressions) - GoAhead correctly parses nested brackets.

### Variadic Functions

GoAhead fully supports variadic helper functions:

```go
// Helper definition
func joinAll(sep string, parts ...string) string {
    return strings.Join(parts, sep)
}

func sum(nums ...int) int {
    total := 0
    for _, n := range nums { total += n }
    return total
}
```

```go
// Usage - pass any number of arguments
//:joinAll:"-":"a":"b":"c":"d"
result = ""  // → "a-b-c-d"

//:sum:1:2:3:4:5
total = 0    // → 15
```

### Placeholder Targets

The placeholder replaces the **first matching literal** in the following statement:

| Return Type | Placeholder | Example Target |
|-------------|-------------|----------------|
| `string` | `""` or `` ` ` `` | `msg = ""`  |
| `int` | `0` | `count = 0` |
| `float64` | `0.0` | `rate = 0.0` |
| `bool` | `false` | `enabled = false` |
| `uint` | `0xff` (hex) | `flags = 0xff` |

---

## Helper File Features

### Constants and Variables

Helper files can define constants and variables at package level:

```go
//go:build exclude
//go:ahead functions

package helpers

const (
    Prefix    = "APP_"
    Separator = "::"
)

var defaultTimeout = 30

func prefixed(key string) string {
    return Prefix + key
}
```

### Custom Types

Define and use custom types within helper files:

```go
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusCompleted
)

func getDefaultStatus() Status {
    return StatusActive
}

func statusName(s Status) string {
    names := []string{"Pending", "Active", "Completed"}
    return names[s]
}
```

### Multiple Helper Files

GoAhead scans the entire directory tree and resolves functions using **hierarchical inheritance**:

- Functions resolve **bottom-up** from source file's directory to root
- A source file "sees" functions from its own directory + all ancestors
- **Local shadows parent**: A subdirectory's function overrides the parent's function of the same name
- **Siblings are isolated**: `pkg1/` cannot see `pkg2/`'s functions (only common ancestors)

```
project/
├── helpers.go        # version() -> "1.0.0", common() -> "shared"
├── main.go           # uses version() → "1.0.0", common() → "shared"
├── pkg1/
│   ├── helpers.go    # version() -> "2.0.0-pkg1"  ← shadows root
│   └── main.go       # uses version() → "2.0.0-pkg1", common() → "shared"
└── pkg2/
    └── main.go       # uses version() → "1.0.0", common() → "shared"
```

**Console output** shows which helper file was used:
```
[goahead] Replaced in pkg1/main.go: version() -> "2.0.0-pkg1" (from pkg1/helpers.go)
[goahead] WARNING: Function 'version' in pkg1/helpers.go shadows function in helpers.go
```

⚠️ **Duplicate function names in the same directory cause a fatal error** - GoAhead will exit with an error showing both file locations.

---

## Standard Library Integration

### Auto-detected Packages

GoAhead automatically resolves common standard library packages:

```go
//:strings.ToUpper:"hello"
upper = ""  // → "HELLO"

//:os.Getenv:"HOME"
home = ""   // → "/home/user"

//:strconv.Itoa:42
str = ""    // → "42"

//:http.DetectContentType:=[]byte("PNG data")
mime = ""   // → "application/octet-stream"
```

---

## Function Injection

GoAhead can **inject entire functions** from helper files into your source code. This is useful for obfuscation scenarios where you need both an encoding function (executed at build time) and a decoding function (included in runtime).

### Basic Syntax

```
//:inject:FunctionName
```

### Example: String Obfuscation

**Helper file** (`helpers.go`):
```go
//go:build exclude
//go:ahead functions

package main

const xorKey byte = 0x42

func Shadow(s string) string {
    result := ""
    for _, c := range s {
        result += string(byte(c) ^ xorKey)
    }
    return result
}

func Unshadow(s string) string {
    result := ""
    for _, c := range s {
        result += string(byte(c) ^ xorKey)
    }
    return result
}
```

**Source file** (`main.go`):
```go
package main

import "fmt"

//:Shadow:"secret password"
var encrypted = ""

//:inject:Unshadow

func main() {
    fmt.Println(Unshadow(encrypted))
}
```

**After processing**:
```go
package main

import "fmt"

const xorKey byte = 0x42  // Dependency copied automatically

var encrypted = "1'!0'6r2#115/0&"

func Unshadow(s string) string {
    result := ""
    for _, c := range s {
        result += string(byte(c) ^ xorKey)
    }
    return result
}

func main() {
    fmt.Println(Unshadow(encrypted))
}
```

### What Gets Injected

When you use `//:inject:FunctionName`, GoAhead:

1. **Copies the function** from the helper file (the marker line is replaced)
2. **Adds required imports** used by the function (single-line imports are upgraded to a block when needed)
3. **Includes dependencies** (constants, variables, types) that the function uses
4. **Includes helper-to-helper dependencies** (other helper functions called by the injected function)
5. **Respects hierarchy** - uses the function from the nearest helper file

If the function is not found, GoAhead leaves the marker line untouched and prints a warning.

---

## CGO Compatibility

GoAhead works correctly with CGO files:

```go
package main

/*
#include <stdio.h>
#cgo LDFLAGS: -lm

// C comments with colons: like:this are preserved
*/
import "C"

var (
    //:getLibVersion
    libVersion = ""
)
```

CGO preambles, directives, and comments are fully preserved.

---

## Command-line Reference

```
goahead -dir=<path> [-verbose] [-version]
```

| Flag | Description |
|------|-------------|
| `-dir` | Directory to process (default `.`) |
| `-verbose` | Print detailed progress information |
| `-version` | Print the tool version and exit |
| `-help` | Show the extended usage banner |

### Environment Variables

```bash
GOAHEAD_VERBOSE=1   # Force verbose output when invoked via toolexec
```

---

- Ensure helper file has both `//go:build exclude` and `//go:ahead functions`
- Check function name matches exactly (case-sensitive)
- Verify argument count matches function signature

### Type mismatch

- Ensure the literal placeholder matches the return type
- Use `0` for int, `0.0` for float, `""` for string, `false` for bool

### Colon in argument

- Wrap strings in quotes: `"http://localhost:8080"`
- For raw expressions, use `=`: `//:process:=map[string]int{"a": 1}`

### CGO errors

- GoAhead preserves CGO preambles - check C code syntax separately
- Ensure `import "C"` appears alone after the preamble
