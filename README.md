# GoAhead -> Compile-time Code Generation for Go

GoAhead plugs into the Go toolchain and replaces lightweight placeholder comments with real values **before** your code is compiled. Helpers that you keep alongside your project are executed at build time, so your runtime code stays clean while still benefiting from generated constants, strings, configuration, or any other deterministic value.

[![CodeQL Advanced](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml/badge.svg)](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AeonDave/goahead)](https://goreportcard.com/report/github.com/AeonDave/goahead)
![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/AeonDave/goahead)
![GitHub last commit](https://img.shields.io/github/last-commit/AeonDave/goahead)
![GitHub License](https://img.shields.io/github/license/AeonDave/goahead)


## Highlights
- Works with every Go command via `-toolexec` (build, test, run, generate)
- Intelligent placeholder replacement keeps surrounding expressions intact
- Understands parameterised helpers, raw Go expressions, and simple type inference
- Supports standard-library packages and custom aliases through `//go:ahead import`
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
    fmt.Printf("%s on %d
", greeting, listenPort)
}
```
Run your normal build and let GoAhead do the rest:
```bash
go build -toolexec="goahead" ./...
```

## Placeholder Grammar
```
//:functionName[:arg1[:argN]]
```
- Strings use double quotes: `"production"`
- Numbers and booleans are unquoted: `42`, `3.14`, `true`
- Prefix with `=` to pass a raw Go expression that should not be quoted: `//:hash:=strings.TrimSpace(input)`
- Arguments must match the helper signature exactly (GoAhead reports mismatches)
- The first literal placeholder of the matching type in the following statement/expression is replaced (e.g. the first `""`, `0`, `0.0`, or `false`)

## Using Standard Library & External Packages
You can call into packages without copying wrappers:
```go
//:os.Getenv:"HOME"
homeDir := ""
```
GoAhead auto-detects most standard aliases, and you can declare your own for clarity:
```go
//go:ahead import http=net/http
//go:ahead import uuid=github.com/google/uuid
```
Place these directives inside any helper file. The alias is available to every placeholder invocation during the same build.

## Intelligent Replacement
- GoAhead rewrites only the minimal literal placeholder, leaving the surrounding expression untouched.
- Strings, ints, floats, uints, and bools are formatted automatically; complex output is quoted so it compiles immediately.
- Results are cached for the duration of the build to avoid re-running identical helpers.

## Project Layout Examples
Examples live under `examples/` and can be executed with `go run -toolexec="goahead" ./examples/<name>`.

| Directory | Demonstrates |
|-----------|--------------|
| `examples/base`   | Minimal helpers returning strings, ints, and derived values |
| `examples/config` | Generating a configuration struct, sanitising CSV input, and reading env values |
| `examples/stdlib` | Calling `os.Getenv`, `http.DetectContentType`, and other stdlib helpers via alias directives |
| `examples/report` | Multi-argument helpers to build status messages, percentages, and slugs |

## Testing
Integration tests under `test/` spin up temporary modules and assert the emitted Go source. Run them with:
```bash
go test ./test/...
```

## Command-line Reference
```
goahead -dir=<path> [-verbose] [-version]
```
| Flag      | Description                                   |
|-----------|-----------------------------------------------|
| `-dir`    | Directory to process (default `.`)            |
| `-verbose`| Print detailed progress information           |
| `-version`| Print the tool version and exit               |
| `-help`   | Show the extended usage banner                |

GoAhead also respects these environment variables:
```
GOAHEAD_VERBOSE=1   # Force verbose output when invoked via toolexec
```

## Tips
- Keep helpers deterministic and side-effect free for reproducible builds.
- Use multiple helper files to organise different domains (config, copy, test data, etc.).
- Commit the generated placeholders only if you need deterministic review diffs; otherwise regenerate during CI.
