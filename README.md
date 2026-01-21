# GoAhead → Compile-time Code Generation for Go

GoAhead plugs into the Go toolchain and replaces lightweight placeholder comments with real values **before** your code is compiled. Helpers that you keep alongside your project are executed at build time, so your runtime code stays clean while still benefiting from generated constants, strings, configuration, or any other deterministic value.

[![CodeQL Advanced](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml/badge.svg)](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AeonDave/goahead)](https://goreportcard.com/report/github.com/AeonDave/goahead)
![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/AeonDave/goahead)
![GitHub last commit](https://img.shields.io/github/last-commit/AeonDave/goahead)
![GitHub License](https://img.shields.io/github/license/AeonDave/goahead)

![GoAhead](logo.png)

## Highlights

- Works with every Go command via `-toolexec` (build, test, run, generate)
- Intelligent placeholder replacement keeps surrounding expressions intact
- Understands parameterised helpers, raw Go expressions, and simple type inference
- Supports standard-library packages and custom aliases through `//go:ahead import`
- Full support for variadic functions, constants, type definitions, and complex expressions
- Cross-platform compatible (Linux, macOS, Windows)
- Ships with ready-to-run examples and an integration test suite

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

func port() int { return 8080 }
```

### 2. Reference helpers from your code

Placeholders use the form `//:functionName[:arg1[:argN]]` and attach to the statement they decorate.

```go
package main

import "fmt"

var (
    //:welcome:"gopher"
    greeting = ""

    //:port
    listenPort = 0
)

func main() {
    fmt.Printf("%s on %d\n", greeting, listenPort)
}
```

### 3. Build with GoAhead

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

Organize helpers across multiple files - all functions are available:

```
project/
├── helpers/
│   ├── config.go      # //go:ahead functions
│   ├── formatting.go  # //go:ahead functions
│   └── crypto.go      # //go:ahead functions
└── main.go
```

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
```

### Import Aliases

Declare aliases for any package in your helper file:

```go
//go:ahead import http=net/http
//go:ahead import filepath=path/filepath
//go:ahead import b64=encoding/base64
```

Then use them in placeholders:

```go
//:http.DetectContentType:=[]byte("PNG data")
mime = ""

//:filepath.Base:"/usr/local/bin/app"
name = ""

//:b64.StdEncoding.EncodeToString:=[]byte("secret")
encoded = ""
```

---

## Compatibility with Go Directives

GoAhead preserves all standard Go directives:

| Directive | Preserved | Notes |
|-----------|-----------|-------|
| `//go:build` | ✅ | Build constraints |
| `// +build` | ✅ | Legacy build tags |
| `//go:generate` | ✅ | Code generation |
| `//go:embed` | ✅ | Embed files |
| `//go:noinline` | ✅ | Compiler hints |
| `//go:nosplit` | ✅ | Stack management |
| `//go:linkname` | ✅ | Symbol linking |

Example mixing GoAhead with Go directives:

```go
//go:build linux || darwin

package main

//go:generate stringer -type=Status

//go:embed config.json
var configData string

var (
    //:getVersion
    version = ""
)

//go:noinline
func criticalPath() { /* ... */ }
```

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

## Project Layout Examples

| Directory | Demonstrates |
|-----------|--------------|
| `examples/base` | Basic helpers returning strings, ints |
| `examples/config` | Configuration struct, CSV sanitization, env values |
| `examples/stdlib_e` | Standard library calls via aliases |
| `examples/report` | Multi-argument helpers, formatting |
| `examples/variadic` | Variadic function support |
| `examples/constants` | Constants, variables, custom types in helpers |
| `examples/expressions` | Complex expressions with maps, structs, slices |
| `examples/types` | Custom type definitions and usage |
| `examples/directives` | Mixing GoAhead with Go directives |

Run any example:

```bash
go run -toolexec="goahead" ./examples/variadic
```

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

## Testing

Integration tests under `test/` spin up temporary modules and assert the emitted Go source:

```bash
go test ./test/...           # All tests
go test ./test/... -v        # Verbose output
go test ./test/... -race     # Race detection
```

Test categories:
- **Grammar tests**: Placeholder syntax variations
- **Argument parsing**: All numeric formats, strings, escaping
- **Expression tests**: Maps, structs, slices with colons
- **Import tests**: Aliases, stdlib resolution
- **Directive tests**: Preserving //go: directives
- **CGO tests**: CGO compatibility
- **Cross-platform**: Windows/Unix path handling

---

## Best Practices

1. **Keep helpers deterministic** - No random values, no time-dependent output unless intentional
2. **Avoid side effects** - Helpers should be pure functions for reproducible builds
3. **Use multiple helper files** - Organize by domain (config, crypto, formatting)
4. **Prefer explicit types** - Return concrete types, avoid `interface{}`
5. **Test your helpers** - Write unit tests for complex helper logic
6. **Document placeholders** - Add comments explaining what each placeholder generates

---

## Troubleshooting

### Placeholder not replaced

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

---

## License

MIT License - see [LICENSE](LICENSE) for details.
