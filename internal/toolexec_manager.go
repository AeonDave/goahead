package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolexecManager gestisce l'integrazione con toolexec
type ToolexecManager struct{}

// Variable to track if version has been shown
var versionShown = false

// NewToolexecManager crea un nuovo gestore toolexec
func NewToolexecManager() *ToolexecManager {
	return &ToolexecManager{}
}

// RunAsToolexec esegue goahead come wrapper toolexec
func (tm *ToolexecManager) RunAsToolexec() {
	// Show version info only once
	if !versionShown {
		fmt.Fprintf(os.Stderr, "[goahead] GoAhead Code Generator v%s\n", Version)
		fmt.Fprintf(os.Stderr, "[goahead] Processing Go compilation with intelligent code generation\n")
		versionShown = true
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <original-tool> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	originalTool := os.Args[1]
	originalArgs := os.Args[2:]

	// Se non è il compilatore, esegui lo strumento originale
	if !tm.isCompilerTool(originalTool) {
		tm.runOriginalTool(originalTool, originalArgs)
		return
	}

	// Estrae i file Go e la directory di output
	goFiles, outputDir := tm.extractFilesAndOutputDir(originalArgs)

	if len(goFiles) > 0 {
		// Filtra solo i file utente
		userFiles := FilterUserFiles(goFiles)

		if len(userFiles) == 0 {
			tm.runOriginalTool(originalTool, originalArgs)
			return
		}

		// Determina la directory di lavoro
		workDir := tm.determineWorkDir(userFiles, outputDir)

		// Esegue la generazione del codice se richiesta
		tm.runCodegenIfVerbose(workDir, goFiles, userFiles)
	}

	// Esegue sempre lo strumento originale alla fine
	tm.runOriginalTool(originalTool, originalArgs)
}

// isCompilerTool verifica se lo strumento è un compilatore
func (tm *ToolexecManager) isCompilerTool(tool string) bool {
	return strings.HasSuffix(tool, "compile") || strings.Contains(tool, "compile")
}

// extractFilesAndOutputDir estrae i file Go e la directory di output dagli argomenti
func (tm *ToolexecManager) extractFilesAndOutputDir(args []string) ([]string, string) {
	var goFiles []string
	var outputDir string

	for i, arg := range args {
		if strings.HasSuffix(arg, ".go") {
			goFiles = append(goFiles, arg)
		}
		if arg == "-o" && i+1 < len(args) {
			outputPath := args[i+1]
			outputDir = filepath.Dir(outputPath)
		}
	}

	return goFiles, outputDir
}

// determineWorkDir determina la directory di lavoro
func (tm *ToolexecManager) determineWorkDir(userFiles []string, outputDir string) string {
	workDir := FindCommonDir(userFiles)
	if workDir == "" {
		workDir = outputDir
	}
	if workDir == "" {
		workDir = "."
	}
	return workDir
}

// runCodegenIfVerbose esegue la generazione del codice se il modo verbose è attivo
func (tm *ToolexecManager) runCodegenIfVerbose(workDir string, goFiles, userFiles []string) {
	verbose := os.Getenv("GOAHEAD_VERBOSE") == "1"

	if verbose {
		fmt.Fprintf(os.Stderr, "[goahead] Files detected: %v\n", goFiles)
		fmt.Fprintf(os.Stderr, "[goahead] User files after filtering: %v\n", userFiles)
		fmt.Fprintf(os.Stderr, "[goahead] Running codegen in %s\n", workDir)

		// Print current directory for debugging
		cwd, err := os.Getwd()
		if err == nil {
			fmt.Fprintf(os.Stderr, "[goahead] Current working directory: %s\n", cwd)
		}
	}

	// Log file types for better debugging
	if verbose {
		tm.logFileTypes(goFiles)
	}

	if err := RunCodegen(workDir, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[goahead] Codegen failed: %v\n", err)
		}
	}
}

// logFileTypes logs detailed information about file types
func (tm *ToolexecManager) logFileTypes(files []string) {
	for _, file := range files {
		fileType := "unknown"
		if strings.Contains(file, "_test.go") {
			fileType = "test file"
		} else if strings.HasSuffix(file, ".go") {
			fileType = "go file"
		}

		location := "user file"
		if tm.isSystemFile(file) {
			location = "system file"
		}

		fmt.Fprintf(os.Stderr, "[goahead] File: %s (Type: %s, Location: %s)\n", file, fileType, location)
	}
}

// isSystemFile checks if the file is a system file
func (tm *ToolexecManager) isSystemFile(file string) bool {
	// Check common system paths
	systemPaths := []string{
		"/usr/lib/go",
		"/usr/local/go",
		os.Getenv("GOROOT"),
		filepath.Join(os.Getenv("GOPATH"), "pkg/mod"),
	}

	for _, path := range systemPaths {
		if path != "" && strings.Contains(file, path) {
			return true
		}
	}

	return false
}

// runOriginalTool esegue lo strumento originale
func (tm *ToolexecManager) runOriginalTool(tool string, args []string) {
	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}
