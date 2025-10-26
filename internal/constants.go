package internal

const (
	Version           = "1.1.0"
	FunctionMarker    = "//go:ahead functions"
	CommentPattern    = `^\s*//\s*:([^:]+)(?::(.*))?`
	ExecutionTemplate = `package main

import (
	{{.FmtAlias}} "fmt"
{{- range .Imports}}
	{{.}}
{{- end}}
)

{{- if .UserCode}}
{{.UserCode}}

{{- end}}
func goaheadFirst[T any](v T, rest ...any) T {
        for _, candidate := range rest {
                if candidate == nil {
                        continue
                }
                if err, ok := candidate.(error); ok && err != nil {
                        panic(err)
                }
        }
        return v
}

func main() {
	result := goaheadFirst({{.CallExpr}})
	{{.FmtAlias}}.Printf("%#v", result)
}
`
)

var (
	GoInstallPaths = []string{
		"/usr/lib/go",
		"/usr/local/go",
		"/opt/go",
		"\\Go\\", // Windows
	}
	SystemPaths = []string{
		"/runtime/",
		"/internal/",
		"/vendor/",
		"/pkg/mod/",
	}
)
