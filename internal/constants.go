package internal

import "runtime/debug"

var Version = getVersion()

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

const (
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
func goaheadFirst[T any](v T, _ ...any) T {
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
		// Windows paths
		"\\runtime\\",
		"\\internal\\",
		"\\vendor\\",
		"\\pkg\\mod\\",
	}
)
