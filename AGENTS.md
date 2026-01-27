# GoAhead - Development Guide

Compile-time code generation for Go. Executes helper functions at build time and replaces placeholders with results.

**Core concept:** Placeholder comment (`//:func:args`) + literal value on next line → GoAhead replaces literal with computed result.

---

## Project Structure

```
goahead/
├── main.go                    # CLI entry point
├── internal/                  # All business logic
│   ├── codegen.go            # Orchestration
│   ├── code_processor.go     # Placeholder replacement
│   ├── file_processor.go     # File I/O, parsing
│   ├── function_executor.go  # Helper execution, depth resolution
│   ├── injector.go           # Function injection
│   ├── toolexec_manager.go   # Toolexec mode
│   ├── types.go              # Core types: ProcessorContext, UserFunction, Config
│   └── constants.go          # Version, patterns
├── test/                      # All tests
│   ├── test_helpers.go       # setupTestDir, verifyCompiles, processAndReplace
│   └── *_test.go             # Tests by feature
└── examples/                  # Feature examples
```

---

## Execution Flow

1. **Scan** - `file_processor.CollectAllGoFiles()` walks tree, categorizes files
2. **Load** - `file_processor.LoadUserFunctions()` parses helpers, registers in `ProcessorContext.FunctionsByDepth`
3. **Prepare** - `function_executor.ensurePreparedForDir()` builds executable code for each source directory using depth resolution
4. **Process** - `code_processor.ProcessFile()` finds placeholders, calls helpers, replaces literals
5. **Inject** - `injector.ProcessFileInjections()` copies functions from helpers to source

---

## Key Mechanisms

### Depth-Based Resolution

**Purpose:** Allow different implementations of same symbol at different tree depths without conflicts.

**Algorithm:**
```
sourceDepth = calculateDepth(sourceFile)
for depth = sourceDepth down to 0:
    if symbol exists at depth:
        return symbol
```

**Rules:**
- **Only exported symbols** (uppercase) are tracked/available for placeholders
- Same-depth symbols pool and share (siblings see each other)
- Deeper shadows shallower (child overrides parent)
- Duplicate at same depth = FATAL
- No upward inheritance (root can't see child-only symbols)

**Implementation:** `internal/function_executor.go`:
- `collectVisibleHelperFiles()` - gathers helpers from source depth to 0
- `filterShadowedDeclarations()` - removes shadowed funcs/vars/consts/types
- `processFunctionFileWithNames()` - extracts all **exported** identifiers using `token.IsExported()`

**Critical:** 
- Variables, constants, and types follow same shadowing as functions
- Unexported symbols (lowercase) are ignored for placeholder/shadowing but still executable within helpers
- This prevents "redeclared" errors and aligns with Go export conventions

### Submodule Isolation

**Purpose:** Directories with their own `go.mod` are independent projects - they don't inherit parent helpers and are processed as separate trees.

**Detection:** During `CollectAllGoFiles()`, if a subdirectory contains `go.mod`, it's:
1. Added to `ctx.Submodules`
2. Skipped with `filepath.SkipDir` (not processed as part of parent)

**Processing:** After main project completes, `RunCodegen()` recursively processes each submodule:
```go
for _, submodule := range ctx.Submodules {
    RunCodegen(submodule, verbose)  // Fresh context, isolated tree
}
```

**Behavior:**
- Submodule helpers are **NOT visible** to parent project
- Parent helpers are **NOT visible** to submodule
- Each submodule has its own depth-based resolution tree starting at depth 0
- Works recursively (submodules can contain submodules)
- Single `goahead` invocation processes entire workspace including all nested submodules

**Use case:** Monorepos with multiple Go modules that need independent code generation.

---

### Function Injection

**Purpose:** Copy runtime functions from helpers to source (e.g., obfuscation: encode at build time, decode at runtime).

**Markers:**
```go
//:inject:MethodName
type Interface interface {
    MethodName(args) returnType
}
```

**Behavior:**
- Validates method exists in interface
- Removes existing injected code
- Copies function + dependencies from helper
- Preserves marker (repeatable on subsequent builds)

**Implementation:** `internal/injector.go`

---

## Testing Philosophy

**Critical:** Tests verify the SPECIFICATION, not the implementation.

**Wrong approach:**
```go
// Bad: models current buggy behavior
result := process(input)
if strings.Contains(result, "something") { ... } // Just checks output
```

**Correct approach:**
```go
// Good: verifies specification
result := processAndReplace(t, dir, "main.go")
verifyCompiles(t, result)  // Compiles = valid Go
if !strings.Contains(result, `expected = "value"`) {
    t.Errorf("Should replace placeholder with value")
}
```

**Must test:**
- Positive cases (valid inputs work)
- Negative cases (invalid inputs fail with correct error)
- Edge cases (empty, nil, boundary, Unicode)
- Generated code compiles (`verifyCompiles`)

**Test utilities** (`test/test_helpers.go`):
- `setupTestDir(t, files)` - temp dir with test files
- `verifyCompiles(t, code)` - compile check
- `processAndReplace(t, dir, file)` - full pipeline

---

## Coding Standards

**Stdlib only** - no external dependencies
**Deterministic** - same input = same output always
**Error wrapping** - `fmt.Errorf("context: %w", err)`
**No panics** - return errors (except truly unrecoverable)
**Config-driven** - CLI flags → `internal.Config`

---

## Common Pitfalls

1. **Placeholder needs literal on next line** - comment alone does nothing
2. **Helper files need TWO tags** - `//go:build exclude` AND `//go:ahead functions`
3. **Case-sensitive** - `Version` ≠ `version`
4. **Strings need quotes** - `//:greet:"World"` not `//:greet:World`
5. **Toolexec + CGO = race condition** - use subcommands for CGO

---

## Development Workflow

**Before changes:**
```bash
go test ./...         # Verify tests pass
```

**After changes:**
```bash
gofmt -w .           # Format
go vet ./...         # Static analysis
go build ./...       # Build check
go test ./...        # All tests
go test -race ./...  # Race detection
```

**Adding features:**
1. Read relevant tests to understand expected behavior
2. Add tests for new functionality FIRST
3. Implement feature
4. Verify generated code compiles (`verifyCompiles`)
5. Run full test suite

**Fixing bugs:**
1. Add failing test demonstrating bug
2. Fix bug
3. Verify test passes
4. Run full suite

---

## PR Requirements

1. All tests pass (`go test ./...`)
2. No race conditions (`go test -race ./...`)
3. Update README.md for user-facing changes
4. Update AGENTS.md for structural/flow changes
5. Include tests for new features
6. Include regression tests for bug fixes
