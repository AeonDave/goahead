# GoAhead Agents Guide

## What GoAhead Does

GoAhead is a compile-time code generation tool for Go. It replaces placeholder comments with computed values at build time.

**Toolexec mode** (recommended):
```bash
go build -toolexec="goahead" ./...
```

**Standalone mode**:
```bash
goahead -dir=./mypackage
```

## Placeholder Grammar

### File Marker (required in files using placeholders)
```
//go:ahead [helpers.go] [stdlib:strings,fmt]
```
- `[helpers.go]`: Optional helper file containing functions to execute
- `[stdlib:pkg1,pkg2]`: Optional stdlib packages to use directly

### Placeholder Syntax
```
//:functionName:arg1:arg2:...
nextLineValue
```

The placeholder comment MUST be immediately followed by a literal value on the next line. GoAhead replaces that literal with the function's return value.

### Argument Types
| Type | Example | Notes |
|------|---------|-------|
| String | `"hello"` or `` `raw` `` | Quoted or backtick |
| Int | `42`, `-5`, `0x1F`, `0b101` | Decimal, hex, binary, octal |
| Float | `3.14`, `-2.5`, `1e-10` | Scientific notation supported |
| Bool | `true`, `false` | |

### Examples

**Helper file** (`helpers.go`):
```go
//go:build ignore

package main

func version() string { return "1.0.0" }
func add(a, b int) int { return a + b }
func greet(name string) string { return "Hello, " + name }
```

**Source file** (`main.go`):
```go
//go:ahead helpers.go

package main

const Version = //:version:
"placeholder"

const Sum = //:add:10:20
0

const Greeting = //:greet:"World"
""
```

**After processing**:
```go
const Version = "1.0.0"
const Sum = 30
const Greeting = "Hello, World"
```

### Stdlib Integration
```go
//go:ahead stdlib:strings,strconv

const Upper = //:strings.ToUpper:"hello"
""

const Num = //:strconv.Itoa:42
""
```

## Repository Overview

- `go.mod`: Module `github.com/AeonDave/goahead`, Go 1.22+
- `main.go`: CLI entry point for toolexec and standalone modes
- `internal/`: Core logic (codegen, toolexec, function execution)
- `examples/`: Feature examples:
  - `base/`: Simple helper functions
  - `config/`: Configuration injection
  - `report/`: Multiple helpers
  - `stdlib_e/`: Stdlib integration
  - `variadic/`: Variadic functions (`joinAll`, `sum`, `maxOf`)
  - `constants/`: Constants and types in helpers
  - `expressions/`: Maps, structs, slices
  - `types/`: Custom type definitions
  - `directives/`: Go directives (`//go:embed`, `//go:noinline`)
- `test/`: Tests by category:
  - `test_helpers.go`: Shared utilities
  - `grammar_test.go`: Placeholder syntax
  - `arguments_test.go`: Argument parsing
  - `expressions_test.go`: Complex expressions
  - `imports_test.go`: Import aliases
  - `helpers_file_test.go`: Helper features
  - `strings_test.go`: Unicode, JSON, special strings
  - `cgo_test.go`: CGO compatibility
  - `directives_test.go`: Go directive preservation
  - `platform_test.go`: Cross-platform paths

## Coding Standards

- Standard library only unless documented otherwise
- Run `gofmt`; group and sort imports
- Return errors with `fmt.Errorf` and `%w`; no panics
- Deterministic: identical inputs → identical outputs
- Reusable code goes in `internal/`

## Implementation Notes

- `internal/` packages: side-effect free at import, use constructors
- Marker syntax: `//go:ahead ...` (file), `//:func:args` (placeholder)
- CLI flags wired through `internal.Config`
- Tests required for behaviour changes

## Testing

```bash
go test ./...           # Full suite
go test ./internal/...  # Core logic
go test ./test/...      # Integration tests
go test -race ./...     # Race detection
```

## Pre-Submission Checks

```bash
go vet ./...
go build ./...
```

## Pull Request Expectations

- Summarise behavioural changes
- Update `README.md` for user-facing changes
- Update this AGENTS.md for structural changes
- Include regression tests
