package internal

import (
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"time"
)

func RunCodegen(dir string, verbose bool) error {
	startTotal := time.Now()

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
	// This also detects and records submodules (directories with their own go.mod)
	startWalk := time.Now()
	allFiles, err := fileProcessor.CollectAllGoFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to collect files: %v", err)
	}
	if verbose {
		fmt.Printf("[goahead] Walk completed in %v\n", time.Since(startWalk))
	}
	if len(ctx.Submodules) > 0 {
		// Always show submodules found (important info)
		fmt.Printf("[goahead] Found %d submodule(s) to process separately:\n", len(ctx.Submodules))
		for _, sub := range ctx.Submodules {
			relPath, _ := filepath.Rel(ctx.RootDir, sub)
			if relPath == "" {
				relPath = sub
			}
			fmt.Printf("  - %s\n", relPath)
		}
	}

	// Track if we have work to do in this project
	hasLocalWork := len(ctx.FuncFiles) > 0

	if !hasLocalWork {
		if verbose {
			log.Printf("No function files found in this project (looking for files with '%s' marker)", FunctionMarker)
		}
		// Don't return - we still need to process submodules below
	}

	if hasLocalWork {
		startLoad := time.Now()
		if err := fileProcessor.LoadUserFunctions(); err != nil {
			return fmt.Errorf("failed to load user functions: %v", err)
		}
		if err := executor.Prepare(); err != nil {
			return fmt.Errorf("failed to prepare executor: %v", err)
		}
		if verbose {
			fmt.Printf("[goahead] Load functions completed in %v\n", time.Since(startLoad))
			printLoadedInfo(ctx)
		}

		// Fast-check: identify which files need processing (have markers)
		startFilter := time.Now()
		filesToProcess := fileProcessor.FilterFilesWithMarkers(allFiles)
		if verbose {
			fmt.Printf("[goahead] Filter completed in %v\n", time.Since(startFilter))
			fmt.Printf("[goahead] Found %d files with markers out of %d total .go files\n", len(filesToProcess), len(allFiles))
		}

		// Process files sequentially to avoid race conditions on caches
		startProcess := time.Now()
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
			fmt.Printf("[goahead] Process completed in %v\n", time.Since(startProcess))
		}
	}

	if verbose {
		fmt.Printf("[goahead] Total time: %v\n", time.Since(startTotal))
		fmt.Println("[goahead] Code generation completed successfully")
	}

	// Process submodules recursively (each submodule is treated as an independent project)
	// This happens AFTER the main project is done, so submodules are completely isolated
	submodules := ctx.Submodules // Copy before ctx is garbage collected
	for _, submodule := range submodules {
		relPath, _ := filepath.Rel(ctx.RootDir, submodule)
		if relPath == "" {
			relPath = submodule
		}
		fmt.Printf("\n[goahead] Processing submodule: %s\n", relPath)
		if err := RunCodegen(submodule, verbose); err != nil {
			return fmt.Errorf("error processing submodule %s: %v", submodule, err)
		}
	}

	return nil
}

func printLoadedInfo(ctx *ProcessorContext) {
	fmt.Printf("Found %d function file(s):\n", len(ctx.FuncFiles))
	for _, file := range ctx.FuncFiles {
		relPath, _ := filepath.Rel(ctx.RootDir, file)
		if relPath == "" {
			relPath = file
		}
		fmt.Printf("  - %s\n", relPath)
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
	fmt.Printf("Loaded %d exported function(s) across %d %s:\n", totalFuncs, maxDepth+1, depthWord)
	fmt.Print(ctx.FormatDepthInfo())
}
