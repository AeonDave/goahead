package internal

import (
	"fmt"
	"go/token"
	"log"
	"os"
	"strings"
)

func RunCodegen(dir string, verbose bool) error {
	if verbose {
		fmt.Printf("Parsed flags:\n")
		fmt.Printf("  dir: '%s'\n", dir)
		fmt.Printf("  verbose: %t\n", verbose)
	}
	ctx := &ProcessorContext{
		Functions:       make(map[string]*UserFunction),
		FileSet:         token.NewFileSet(),
		ImportOverrides: make(map[string]string),
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
	if err := fileProcessor.FindFunctionFiles(dir); err != nil {
		return fmt.Errorf("failed to find function files: %v", err)
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
	if err := fileProcessor.ProcessDirectory(dir, verbose, codeProcessor); err != nil {
		return fmt.Errorf("error processing directory: %v", err)
	}
	if verbose {
		fmt.Println("Code generation completed successfully")
	}

	return nil
}

func printLoadedInfo(ctx *ProcessorContext) {
	fmt.Printf("Found %d function files:\n", len(ctx.FuncFiles))
	for _, file := range ctx.FuncFiles {
		fmt.Printf("  - %s\n", file)
	}

	fmt.Printf("Loaded %d user functions:\n", len(ctx.Functions))
	for name, fn := range ctx.Functions {
		fmt.Printf("  - %s(%s) %s\n", name, strings.Join(fn.InputTypes, ", "), fn.OutputType)
	}
}
