# GoAhead - Code Generation Tool

Simple compile-time user-defined functions evaluator tool.
Replaces special comments with the results of user-defined functions.

## Features

- **Automatic function discovery**: Automatically finds function files with `//go:ahead functions` marker
- **Multiple function files**: Support unlimited function files in your project
- **User-defined functions**: Define your own functions for code generation
- **Multiple return types**: Support for `string`, `int`, and `bool` return types
- **Type-safe replacements**: Preserves Go's type system during code generation
- **Robust error handling**: Clear error messages for debugging and duplicate detection
- **Automated workflow**: Integration with `go generate`, Makefiles, and `go build -toolexec`
- **Build integration**: Seamless integration with Go build process via `-toolexec`
- **Cross-platform**: Works on Windows, Linux, and macOS

## Installation

Install GoAhead using Go's package manager:

```bash
go install github.com/AeonDave/goahead@latest
```

## Integration Options

The recommended way to use GoAhead is through Go's `-toolexec` flag, which automatically runs code generation during builds:

```bash
# Use it with your builds (goahead must be in your PATH)
go build -toolexec="goahead" ./...
go test -toolexec="goahead" ./...
go run -toolexec="goahead" main.go
```
This approach automatically runs code generation before compilation, works with all Go commands, requires no manual intervention, and can be integrated into CI/CD pipelines. Verbose output can be enabled with `GOAHEAD_VERBOSE=1`.

Alternatively, you can run the tool manually (`goahead -dir ./path/to/your/project`) or use `//go:generate goahead -dir .` in your Go files and then run `go generate ./...`.

## Quick Start

### 1. Install

```bash
go install github.com/AeonDave/goahead@latest
```

### 2. Define Your Functions

Create one or more function files (e.g., `functions.go`) with these special markers:

```go
//go:build exclude
//go:ahead functions

package main

import "strings"

// String functions
func toUpper(msg string) string {
    return strings.ToUpper(msg)
}

func concat(a, b string) string {
    return a + "_" + b
}

// Int functions
func stringLength(s string) int {
    return len(s)
}

// Bool functions
func isEven(n int) bool {
    return n%2 == 0
}
```

### 3. Use in Your Code

Create a Go file with special comments:

```go
package main

import "fmt"

func main() {
    // This comment will be replaced with "HELLO WORLD"
    //:toUpper:"hello world"
    message := ""
    
    // This comment will be replaced with "Go_Lang"
    //:concat:"Go":"Lang"
    combined := ""
    
    // This comment will be replaced with 7
    //:stringLength:"example"
    length := 0
    
    // This comment will be replaced with true
    //:isEven:10
    even := false
    
    fmt.Printf("Message: %s, Combined: %s, Length: %d, Even: %t\n", 
               message, combined, length, even)
}
```

### 4. Generate Code

Using the recommended build integration:
```bash
go build -toolexec="goahead" main.go
```

Or manually:
```bash
goahead
```

After generation, your code becomes:

```go
package main

import "fmt"

func main() {
    // This comment will be replaced with "HELLO WORLD"
    message := "HELLO WORLD"
    
    // This comment will be replaced with "Go_Lang"
    combined := "Go_Lang"
    
    // This comment will be replaced with 7
    length := 7
    
    // This comment will be replaced with true
    even := true
    
    fmt.Printf("Message: %s, Combined: %s, Length: %d, Even: %t\n", 
               message, combined, length, even)
}
```

## Multiple Function Files

You can organize your functions across multiple files. Function names must be unique across all function files.

## Function Definition Rules

- Functions must be defined in files with both `//go:build exclude` and `//go:ahead functions` markers.
- Functions must return exactly one value of type `string`, `int`, or `bool`.
- Function names are case-sensitive and must be unique.
- Supported argument types: Strings (e.g. `"value"`), Integers (e.g. `42`), Booleans (e.g. `true`).

## Command Line Options

### GoAhead Tool
```bash
goahead [options]

Options:
  -dir string
        Directory to scan for Go files and function files (default ".")
  -verbose
        Enable verbose output
```

When used with `-toolexec`, environment variables can control behavior:
```bash
GOAHEAD_VERBOSE=1    Enable verbose output
```

## Error Handling

The tool provides clear error messages for issues like: function not found, type mismatch, parse errors, file errors, duplicate functions, and no function files found.

## License

MIT License - see LICENSE file for details.
