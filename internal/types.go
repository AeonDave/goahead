package internal

import (
	"fmt"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

type UserFunction struct {
	Name       string
	InputTypes []string
	OutputType string
	FilePath   string
	Depth      int // Depth relative to RootDir (0 = root)
}

type ProcessorContext struct {
	// FunctionsByDepth maps depth level to functions defined at that depth
	// Key is the depth (0 = root), value is map of function name to function
	FunctionsByDepth map[int]map[string]*UserFunction

	// FunctionsByDir maps directory path to functions defined in that directory
	// Key is the absolute directory path, value is map of function name to function
	FunctionsByDir map[string]map[string]*UserFunction

	// RootDir is the root directory being processed (for hierarchy resolution)
	RootDir string

	// Verbose enables detailed logging
	Verbose bool

	// Submodules contains paths to directories with their own go.mod (treated as separate projects)
	Submodules []string

	FileSet     *token.FileSet
	CurrentFile string
	FuncFiles   []string
	TempDir     string
}

// CalculateDepth returns the depth of a directory relative to RootDir
func (ctx *ProcessorContext) CalculateDepth(dir string) int {
	// Normalize paths
	rootClean := filepath.Clean(ctx.RootDir)
	dirClean := filepath.Clean(dir)

	// If same as root, depth is 0
	if rootClean == dirClean {
		return 0
	}

	// Get relative path
	rel, err := filepath.Rel(rootClean, dirClean)
	if err != nil {
		return 0
	}
	if rel == "" || rel == "." {
		return 0
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return 0
	}

	// Count separators + 1 for the final component
	depth := 1
	for _, c := range rel {
		if c == filepath.Separator || c == '/' || c == '\\' {
			depth++
		}
	}
	return depth
}

// ResolveFunction finds a function using depth-based resolution.
// It searches from the source file's depth upward to depth 0.
// Returns the function and the helper file path it came from.
func (ctx *ProcessorContext) ResolveFunction(name, sourceDir string) (*UserFunction, string) {
	sourceDepth := ctx.CalculateDepth(sourceDir)

	// Search from sourceDepth down to 0
	for depth := sourceDepth; depth >= 0; depth-- {
		if funcs, ok := ctx.FunctionsByDepth[depth]; ok {
			if fn, ok := funcs[name]; ok {
				return fn, fn.FilePath
			}
		}
	}

	return nil, ""
}

// GetMaxDepth returns the maximum depth with functions defined
func (ctx *ProcessorContext) GetMaxDepth() int {
	maxDepth := 0
	for depth := range ctx.FunctionsByDepth {
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

// GetFunctionCountByDepth returns total functions at a specific depth
func (ctx *ProcessorContext) GetFunctionCountByDepth(depth int) int {
	if funcs, ok := ctx.FunctionsByDepth[depth]; ok {
		return len(funcs)
	}
	return 0
}

// FormatDepthInfo returns a formatted string showing functions by depth
func (ctx *ProcessorContext) FormatDepthInfo() string {
	var sb strings.Builder
	maxDepth := ctx.GetMaxDepth()

	for depth := 0; depth <= maxDepth; depth++ {
		if funcs, ok := ctx.FunctionsByDepth[depth]; ok && len(funcs) > 0 {
			sb.WriteString(fmt.Sprintf("  Depth %d:\n", depth))
			names := make([]string, 0, len(funcs))
			for name := range funcs {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				fn := funcs[name]
				relPath, _ := filepath.Rel(ctx.RootDir, fn.FilePath)
				if relPath == "" {
					relPath = fn.FilePath
				}
				if fn.OutputType != "" {
					sb.WriteString(fmt.Sprintf("    - %s(%s) %s [%s]\n",
						name, strings.Join(fn.InputTypes, ", "), fn.OutputType, relPath))
				} else {
					sb.WriteString(fmt.Sprintf("    - %s(%s) [%s]\n",
						name, strings.Join(fn.InputTypes, ", "), relPath))
				}
			}
		}
	}
	return sb.String()
}

type Config struct {
	Dir     string
	Verbose bool
	Help    bool
	Version bool
}
