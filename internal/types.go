package internal

import (
	"go/token"
)

type UserFunction struct {
	Name       string
	InputTypes []string
	OutputType string
	FilePath   string
}

type ProcessorContext struct {
	Functions   map[string]*UserFunction
	FileSet     *token.FileSet
	CurrentFile string
	FuncFiles   []string
	TempDir     string
}

type Config struct {
	Dir     string
	Verbose bool
	Help    bool
	Version bool
}
