package internal

const (
	Version           = "1.0.2"
	FunctionMarker    = "//go:ahead functions"
	CommentPattern    = `^\s*//\s*:([^:]+)(?::(.*))?`
	ExecutionTemplate = `package main
	
	import (
	 "fmt"
	{{.AdditionalImports}}
	)
	
	{{.UserFunctions}}
	
	func main() {
	 result := {{.FuncName}}({{.Args}})
	 fmt.Print(result)
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
	//SupportedTypes = []string{
	//	"string",
	//	"int",
	//	"int8",
	//	"int16",
	//	"int32",
	//	"int64",
	//	"uint",
	//	"uint8",
	//	"uint16",
	//	"uint32",
	//	"uint64",
	//	"bool",
	//	"float32",
	//	"float64",
	//	"byte",
	//	"rune",
	//}
	//ExcludedPaths = append(GoInstallPaths, SystemPaths...)
)
