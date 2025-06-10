package internal

const (
	// Version dell'applicazione
	Version = "1.0.2"

	// Marker per identificare i file di funzioni
	FunctionMarker = "//go:ahead functions"

	// Pattern per i commenti di generazione
	CommentPattern = `^\s*//\s*:([^:]+)(?::(.*))?`

	// Template per l'esecuzione delle funzioni
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

// Variabili di configurazione per l'esclusione dei file
var (
	// Percorsi di installazione Go da escludere
	GoInstallPaths = []string{
		"/usr/lib/go",
		"/usr/local/go",
		"/opt/go",
		"\\Go\\", // Windows
	}

	// Percorsi di sistema da escludere
	SystemPaths = []string{
		"/runtime/",
		"/internal/",
		"/vendor/",
		"/pkg/mod/",
	}

	// Tipi supportati per i parametri delle funzioni
	SupportedTypes = []string{
		"string",
		"int",
		"int8",
		"int16",
		"int32",
		"int64",
		"uint",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"bool",
		"float32",
		"float64",
		"byte",
		"rune",
	}

	// Percorsi combinati da escludere
	ExcludedPaths = append(GoInstallPaths, SystemPaths...)
)
