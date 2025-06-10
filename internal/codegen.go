package internal

import (
	"fmt"
	"go/token"
	"log"
	"os"
	"strings"
)

// RunCodegen esegue la generazione del codice
func RunCodegen(dir string, verbose bool) error {
	if verbose {
		fmt.Printf("Parsed flags:\n")
		fmt.Printf("  dir: '%s'\n", dir)
		fmt.Printf("  verbose: %t\n", verbose)
	}

	// Crea il contesto del processore
	ctx := &ProcessorContext{
		Functions: make(map[string]*UserFunction),
		FileSet:   token.NewFileSet(),
	}

	// Crea directory temporanea
	tempDir, err := os.MkdirTemp("", "codegen-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	ctx.TempDir = tempDir

	// Inizializza i processori
	fileProcessor := NewFileProcessor(ctx)
	executor := NewFunctionExecutor(ctx)
	codeProcessor := NewCodeProcessor(ctx, executor)

	// Trova e carica le funzioni
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

	// Mostra informazioni se verbose
	if verbose {
		printLoadedInfo(ctx)
	}

	// Elabora tutti i file
	if err := fileProcessor.ProcessDirectory(dir, verbose, codeProcessor); err != nil {
		return fmt.Errorf("error processing directory: %v", err)
	}

	if verbose {
		fmt.Println("Code generation completed successfully")
	}

	return nil
}

// printLoadedInfo stampa le informazioni sui file e funzioni caricate
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
