package internal

import (
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"
)

func RunCodegen(dir string, verbose bool) error {
	if verbose {
		fmt.Printf("Parsed flags:\n")
		fmt.Printf("  dir: '%s'\n", dir)
		fmt.Printf("  verbose: %t\n", verbose)
	}

	// Get absolute path for RootDir
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	ctx := &ProcessorContext{
		FunctionsByDir:   make(map[string]map[string]*UserFunction),
		FunctionsByDepth: make(map[int]map[string]*UserFunction),
		RootDir:          absDir,
		Verbose:          verbose,
		FileSet:          token.NewFileSet(),
	}
	tempDir, err := os.MkdirTemp("", "codegen-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	ctx.TempDir = tempDir
	fileProcessor := NewFileProcessor(ctx)
	executor := NewFunctionExecutor(ctx)
	codeProcessor := NewCodeProcessor(ctx, executor)
	injector := NewInjector(ctx)

	// Single walk: collect all .go files and categorize them
	allFiles, err := fileProcessor.CollectAllGoFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to collect files: %v", err)
	}

	if len(ctx.FuncFiles) == 0 {
		if verbose {
			log.Printf("No function files found (looking for files with '%s' marker)", FunctionMarker)
		}
		return nil
	}

	if err := fileProcessor.LoadUserFunctions(); err != nil {
		return fmt.Errorf("failed to load user functions: %v", err)
	}
	if err := executor.Prepare(); err != nil {
		return fmt.Errorf("failed to prepare executor: %v", err)
	}
	if verbose {
		printLoadedInfo(ctx)
	}

	// Fast-check: identify which files need processing (have markers)
	filesToProcess := fileProcessor.FilterFilesWithMarkers(allFiles)
	if verbose {
		fmt.Printf("[goahead] Found %d files with markers out of %d total .go files\n", len(filesToProcess), len(allFiles))
	}

	// Process files sequentially to avoid race conditions on caches
	for _, filePath := range filesToProcess {
		// Process injections first
		if err := injector.ProcessFileInjections(filePath, verbose); err != nil {
			return fmt.Errorf("error processing injections in %s: %v", filePath, err)
		}
		// Then process placeholders
		if err := codeProcessor.ProcessFile(filePath, verbose); err != nil {
			return fmt.Errorf("error processing %s: %v", filePath, err)
		}
	}

	if verbose {
		fmt.Println("[goahead] Code generation completed successfully")
	}

	return nil
}

func printLoadedInfo(ctx *ProcessorContext) {
	fmt.Printf("Found %d function files:\n", len(ctx.FuncFiles))
	for _, file := range ctx.FuncFiles {
		fmt.Printf("  - %s\n", file)
	}

	// Show functions organized by depth
	totalFuncs := 0
	for _, funcs := range ctx.FunctionsByDepth {
		totalFuncs += len(funcs)
	}
	maxDepth := ctx.GetMaxDepth()
	depthWord := "depth levels"
	if maxDepth == 0 {
		depthWord = "depth level"
	}
	fmt.Printf("Loaded %d user functions across %d %s:\n", totalFuncs, maxDepth+1, depthWord)
	fmt.Print(ctx.FormatDepthInfo())
}
