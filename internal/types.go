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
	// FunctionsByDir maps directory path to functions defined in that directory
	// Key is the absolute directory path, value is map of function name to function
	FunctionsByDir map[string]map[string]*UserFunction

	// RootDir is the root directory being processed (for hierarchy resolution)
	RootDir string

	// Verbose enables detailed logging
	Verbose bool

	// Legacy: flat map for backward compatibility during transition
	Functions   map[string]*UserFunction
	FileSet     *token.FileSet
	CurrentFile string
	FuncFiles   []string
	TempDir     string
}

// ResolveFunction finds a function by walking up the directory tree from sourceDir.
// Returns the function and the helper file path it came from.
func (ctx *ProcessorContext) ResolveFunction(name, sourceDir string) (*UserFunction, string) {
	// Walk up from sourceDir to RootDir
	for dir := sourceDir; ; dir = parentDir(dir) {
		if funcs, ok := ctx.FunctionsByDir[dir]; ok {
			if fn, ok := funcs[name]; ok {
				return fn, fn.FilePath
			}
		}

		// Stop at root or when we can't go up anymore
		if dir == ctx.RootDir || dir == "." || dir == "" || dir == parentDir(dir) {
			break
		}
	}

	// Check root directory explicitly (in case sourceDir didn't traverse through it)
	if funcs, ok := ctx.FunctionsByDir[ctx.RootDir]; ok {
		if fn, ok := funcs[name]; ok {
			return fn, fn.FilePath
		}
	}

	return nil, ""
}

// parentDir returns the parent directory of a path
func parentDir(dir string) string {
	if dir == "" || dir == "." {
		return ""
	}
	// Find last separator
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			if i == 0 {
				return string(dir[0])
			}
			return dir[:i]
		}
	}
	return "."
}

type Config struct {
	Dir     string
	Verbose bool
	Help    bool
	Version bool
}
