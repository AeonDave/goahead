# GoAhead

Compile-time code generation for Go. Replace placeholder comments with computed values during build.

[![CodeQL Advanced](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml/badge.svg)](https://github.com/AeonDave/goahead/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AeonDave/goahead)](https://goreportcard.com/report/github.com/AeonDave/goahead)
![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/AeonDave/goahead)
![GitHub last commit](https://img.shields.io/github/last-commit/AeonDave/goahead)
![GitHub License](https://img.shields.io/github/license/AeonDave/goahead)

![GoAhead](logo.png)

**Features:**
- Subcommand integration (`goahead build`, `goahead test`, `goahead run`)
- Depth-based symbol resolution with shadowing
- Function injection for runtime code generation
- Variadic functions and raw expressions
- Standard library auto-resolution
- CGO compatible
- Cross-platform

## Quick Start

**1. Create a helper file** (excluded from normal builds):

```go
//go:build exclude
//go:ahead functions

package helpers

// Exported function (uppercase = available for placeholders)
func Welcome(name string) string {
    return "Hello, " + name + "!"
}

// unexported function (lowercase = NOT available)
func formatInternal(s string) string {
    return "[" + s + "]"
}
```

**2. Use placeholder in source**:

```go
package main

//:Welcome:"gopher"
var greeting = ""

func main() {
    println(greeting) // Prints: Hello, gopher!
}
```

**3. Build**:

```bash
goahead build ./...
```

---

## Export Requirements

**Only exported (uppercase) symbols are available for placeholder replacement:**

```go
//go:build exclude
//go:ahead functions

package helpers

// ✅ Available for placeholders
func PublicAPI() string { return "public" }
var ExportedVar = "visible"
const ExportedConst = 42
type ExportedType struct {}

// ❌ NOT available (private implementation)
func internalHelper() string { return "private" }
var privateVar = "hidden"
const privateConst = 99
type privateType struct {}
```

**Rationale:**
- Aligns with Go conventions (export = public API)
- Private functions remain for internal helper use only
- Clearer separation of interface vs implementation
- Injection requires exported functions anyway

---

## Usage Modes

**Subcommands** (recommended, required for CGO):
```bash
goahead build ./...      # Process + build
goahead test ./...       # Process + test  
goahead run ./cmd/app    # Process + run
```

**Toolexec** (not for CGO):
```bash
go build -toolexec="goahead" ./...
```

**Standalone**:
```bash
goahead -dir=./mypackage -verbose
```

---

## Placeholder Syntax

```
//:functionName[:arg1[:arg2[:argN]]]
targetStatement = literalPlaceholder
```

The placeholder comment must appear **immediately before** the target statement. GoAhead replaces the first matching literal.

**Argument types:**

| Type | Example |
|------|---------|
| String | `"hello"`, `` `raw` `` |
| Integer | `42`, `0xFF`, `0o755`, `0b1010` |
| Float | `3.14`, `-2.5`, `1.5e10` |
| Boolean | `true`, `false` |
| Expression | `=strings.TrimSpace(" hi ")` |

**Examples:**

```go
//:version:
const Version = ""  // → "1.0.0"

//:add:10:20
const Sum = 0  // → 30

//:hash:=[]byte{0x89, 0x50, 0x4E, 0x47}
var h = ""  // → "hash_result"
```

> **Note**: Both `//:func` and `// :func` are valid (space-tolerant for formatters).

---

## Depth-Based Symbol Resolution

GoAhead resolves symbols (functions, variables, constants, types) using depth-based inheritance:

**Rules:**
1. **Only exported symbols** (uppercase) are tracked and available
2. Symbols resolve from source depth **down to depth 0** (root)
3. Same-depth symbols are **pooled and visible** across siblings
4. **Deeper shadows shallower** - definitions at greater depth override parent definitions
5. **Duplicates at same depth = FATAL ERROR**

**Example:**

```
project/
├── helpers.go            # var Seed = "root", Version() → "1.0" (depth 0)
├── main.go               # uses Version() → "1.0"
├── obfuscation/          # depth 1
│   ├── helpers.go        # var Seed = "obf", Shadow() → "OBF"
│   └── main.go           # uses Shadow() → "OBF"
└── evasion/              # depth 1
    └── obfuscation/      # depth 2
        ├── helpers.go    # var Seed = "eva", Shadow() → "EVA" (shadows parent)
        └── main.go       # uses Shadow() → "EVA"
```

**Console output shows resolution:**
```
[goahead] Replaced in main.go: Version() → "1.0" (from helpers.go)
[goahead] WARNING: Symbol 'Shadow' at depth 2 shadows symbol at depth 1
```

This prevents "redeclared" errors when multiple helper files define the same variable/constant/type at different depths.

---

## Function Injection

Inject entire functions from helpers into source files. Useful for obfuscation where you need both build-time encoding and runtime decoding.

**Syntax:**

```go
//:inject:MethodName
type InterfaceName interface {
    MethodName(args) returnType
}
```

**Example:**

```go
// helpers.go
//go:build exclude
//go:ahead functions

package main

const key = 0x42

func Shadow(s string) string {
    out := ""
    for _, c := range s {
        out += string(byte(c) ^ key)
    }
    return out
}

func Unshadow(s string) string {
    return Shadow(s) // XOR is reversible
}
```

```go
// main.go
package main

//:Shadow:"secret"
var encoded = ""

//:inject:Unshadow
type Decoder interface {
    Unshadow(s string) string
}

func main() {
    println(Unshadow(encoded))
}
```

**What gets injected:**
- Function implementation
- Required imports (unused imports filtered out)
- Required constants/variables/types
- Helper-to-helper dependencies

**Behavior:**
- Markers **stay** in source (repeatable injection)
- Previous injected code is **removed and re-injected** on each build
- Updates to helpers **propagate automatically**

---

## Standard Library

Auto-detected stdlib packages work directly:

```go
//:strings.ToUpper:"hello"
var upper = ""  // → "HELLO"

//:os.Getenv:"HOME"
var home = ""

//:strconv.Itoa:42
var str = ""  // → "42"

//:http.DetectContentType:=[]byte("data")
var mime = ""
```

---

## Variadic Functions

```go
// Helper
func join(sep string, parts ...string) string {
    return strings.Join(parts, sep)
}

// Usage
//:join:"-":"a":"b":"c"
var result = ""  // → "a-b-c"
```

---

## Installation

```bash
go install github.com/AeonDave/goahead@latest
```

---

## Command Reference

**Subcommands:**
```bash
goahead build [flags] <packages>    # Process then build
goahead test [flags] <packages>     # Process then test
goahead run [flags] <package>       # Process then run
```

**Standalone:**
```bash
goahead -dir=<path> [-verbose] [-version] [-help]
```

**Environment:**
```bash
GOAHEAD_VERBOSE=1    # Enable verbose output
```

---

## CGO Projects

**Always use subcommands** for CGO projects:

```bash
goahead build -tags production ./...
```

**Why?** Toolexec mode (`-toolexec="goahead"`) creates race conditions with parallel CGO compilation, causing corrupted binaries.

---

## Submodule Isolation

GoAhead automatically detects and isolates subdirectories with their own `go.mod` files:

```
monorepo/
├── go.mod                    # Main module
├── helpers.go                # Version() → "1.0-main"
├── main.go                   # Uses Version() → "1.0-main"
└── subproject/               # ← Has its own go.mod
    ├── go.mod                # Separate module
    ├── helpers.go            # Version() → "2.0-sub" (independent)
    └── main.go               # Uses Version() → "2.0-sub"
```

**Behavior:**
- Submodules are **completely isolated** - they don't see parent helpers
- Parent project doesn't see submodule helpers
- Each submodule is processed as a **separate tree** starting at depth 0
- **Single `goahead` invocation** processes everything (recursively)
- Works with **nested submodules** (submodules containing submodules)

**Use case:** Monorepos with multiple Go modules that need independent code generation.

**Console output:**
```
[goahead] Found submodule: subproject
[goahead] Processing submodule: subproject
```

---

## Limitations

**Placeholder replacement is one-time:**
- After replacement, the literal stays in source
- Re-running on processed code has no effect
- To update, restore the original placeholder comment

**Exception:** Injection markers (`//:inject:`) are repeatable and stay in source.

---

## Troubleshooting

**Function not found:**
- Verify `//go:build exclude` and `//go:ahead functions` tags
- Check function name is exact match (case-sensitive)
- Ensure argument count matches signature

**Type mismatch:**
- Match placeholder to return type: `0` for int, `""` for string, etc.

**Colons in arguments:**
- Wrap strings: `"http://localhost:8080"`
- Use expressions: `=map[string]int{"key": 1}`

---

## Examples

See [`examples/`](examples/) directory:
- `base/` - Basic placeholder usage
- `config/` - Configuration generation
- `constants/` - Constant definitions
- `directives/` - Build directives
- `expressions/` - Raw expressions
- `stdlib_e/` - Standard library integration
- `types/` - Custom types
- `variadic/` - Variadic functions

---

## Contributing

See [AGENTS.md](AGENTS.md) for development guidelines.

**Before submitting:**
1. Run `go test ./...` - all tests must pass
2. Add tests for new features
3. Update AGENTS.md for structural changes
4. Update README.md for user-facing changes

---

## License

MIT License - see [LICENSE](LICENSE) file.


