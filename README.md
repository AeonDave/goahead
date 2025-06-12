# GoAhead - Compile-time Code Generation Tool

**Compile-time code generation with intelligent placeholder replacement**

GoAhead is a powerful Go tool that automatically replaces placeholders in your code with the results of user-defined functions during compilation. It features intelligent replacement that preserves complex expressions while substituting only the specific placeholders.

## Quick Installation & Usage

### 1. Install from GitHub
```bash
go install github.com/AeonDave/goahead@latest
```

### 2. Use with Your Projects
```bash
# Automatic code generation during build (recommended)
go build -toolexec="goahead" ./...
go test -toolexec="goahead" ./...
go run -toolexec="goahead" main.go

# Enable verbose output
GOAHEAD_VERBOSE=1 go build -toolexec="goahead" ./...
```

This approach:
- ✅ Runs automatically before compilation
- ✅ Works with all Go commands (`build`, `test`, `run`)
- ✅ Requires no manual intervention
- ✅ Integrates seamlessly with CI/CD pipelines
- ✅ Processes only your project files (excludes system libraries)

### 1. Create Function Definitions
Create a file `functions.go` with your custom functions:

```go
//go:build exclude
//go:ahead functions

package main

import "strings"

// String functions
func getString() string {
    return "Hello World"
}

func toUpper(msg string) string {
    return strings.ToUpper(msg)
}

func concat(a, b string) string {
    return a + " " + b
}

// Numeric functions
func getInt() int {
    return 42
}

func addInt(a, b int) int {
    return a + b
}

func getFloat() float32 {
    return 3.14159
}

// Boolean functions
func getBool() bool {
    return true
}
```

### 2. Use Placeholders in Your Code
Create your main Go file with placeholders:

```go
package main

import (
    "fmt"
    "strings"
)

func main() {
    // ✨ Intelligent replacement preserves complex expressions
    
    //:getString
    result1 := strings.ToUpper("")  // → strings.ToUpper("Hello World")
    
    //:getInt
    result2 := int(0) + 10         // → int(42) + 10
    
    //:getBool
    result3 := !false              // → !true
    
    //:getFloat
    result4 := 0.0 * 2.5          // → 3.14159 * 2.5

    fmt.Printf("String: %s\n", result1)     // HELLO WORLD
    fmt.Printf("Int: %d\n", result2)        // 52
    fmt.Printf("Bool: %t\n", result3)       // false  
    fmt.Printf("Float: %.2f\n", result4)    // 7.85
}
```

### 3. Build with Automatic Code Generation
```bash
# Install GoAhead
go install github.com/AeonDave/goahead@latest

# Build your project with automatic code generation
go build -toolexec="goahead" .

# Run the generated code
./your-project
```

## Replacement Examples

GoAhead intelligently replaces only the placeholders while preserving your expressions:

| Original Code | After GoAhead | Result |
|---------------|---------------|---------|
| `strings.ToUpper("")` | `strings.ToUpper("Hello")` | `"HELLO"` |
| `int(0) + 5` | `int(42) + 5` | `47` |
| `!false` | `!true` | `false` |
| `0.0 * 3.14` | `2.5 * 3.14` | `7.85` |
| `len("") > 0` | `len("test") > 0` | `true` |
## Function Definition Rules

### Requirements
- Functions must be in files with both `//go:build exclude` and `//go:ahead functions` markers
- Functions must return exactly one value
- Function names are case-sensitive and must be unique across all files
- Use meaningful function names for better code readability

### Supported Types
| Type | Placeholder | Example |
|------|-------------|---------|
| `string` | `""` | `func getName() string { return "John" }` |
| `int`, `int8-64` | `0` | `func getAge() int { return 25 }` |
| `uint`, `uint8-64` | `0` | `func getCount() uint { return 100 }` |
| `float32`, `float64` | `0.0` or `0` | `func getPi() float32 { return 3.14 }` |
| `bool` | `false` | `func isValid() bool { return true }` |

### Multiple Function Files
Organize functions across multiple files for better structure:
```
project/
├── auth_functions.go      # Authentication related
├── data_functions.go      # Data processing  
├── config_functions.go    # Configuration
└── main.go                # Your main code
```

### Manual Usage
```bash
goahead [options]

Options:
  -dir string        Directory to process (default ".")
  -verbose          Enable verbose output  
  -help             Show help message
  -version          Show version information
```

### Environment Variables
```bash
GOAHEAD_VERBOSE=1    # Enable verbose output in toolexec mode
```