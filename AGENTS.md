# GoAhead Agents Guide

## Repository Overview

- `go.mod`: Declares the `github.com/AeonDave/goahead` module targeting Go 1.22+.
- `main.go`: Command-line entry point that orchestrates toolexec or standalone execution.
- `internal/`: Core library for code generation, toolexec integration, and helper execution.
- `examples/`: Minimal projects demonstrating how to invoke GoAhead during builds.
- `test/`: Integration and unit tests covering filters, helper execution, and code generation outputs.
- `Makefile`: Convenience targets mirroring common Go toolchain commands.
- `README.md`: Usage overview, installation instructions, and quick-start guide.

## Coding Standards

- Write Go code and stick to the standard library unless an exception is documented.
- Run `gofmt` on any touched Go source; keep imports grouped and sorted.
- Prefer explicit error handling over panics in library code and return contextual errors (`fmt.Errorf` with `%w`).
- Maintain deterministic behaviour: helper execution and code rewriting should be reproducible for identical inputs.
- Keep functions focused; extract reusable utilities into the `internal` package when they span multiple entry points.

## Implementation Notes

- `internal/` packages should remain side-effect free at import time and expose explicit constructors (e.g., `NewToolexecManager`).
- Preserve marker comment conventions (`//go:ahead ...`, `//:helperName:arg`) when transforming source files.
- When extending CLI flags, wire them through `internal.Config` so both standalone and toolexec modes remain aligned.
- Extend `test/` with unit or integration coverage alongside any behaviour changes; create descriptive subdirectories for fixtures when needed.

## Testing

Run Go test commands from the repository root:

```bash
go test ./...           # Full suite
go test ./internal/...  # Package-level checks for core logic
go test ./test/...      # Integration-focused tests
```

Use `go test -race ./...` when modifying concurrent code or file processing pipelines.

## Pre-Submission Checks

Before opening a pull request, execute:

```bash
go vet ./...    # Static analysis
go build ./...  # Ensure binaries compile
```

Include any additional project-specific scripts (e.g., `make test`, `make lint`) if they become relevant to your change.

## Pull Request Expectations

- Summarise behavioural changes, highlighting impacts on code generation, CLI flags, or helper execution.
- Document user-facing updates in `README.md` or example projects when behaviour shifts.
- Update this AGENTS.md guide with any relevant changes.
- Ensure new helpers or transformations include regression tests and, where necessary, example coverage.
