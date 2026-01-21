package internal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ToolexecManager struct{}

var versionShown = false

func NewToolexecManager() *ToolexecManager {
	return &ToolexecManager{}
}

// RunAsToolexec esegue goahead come wrapper toolexec
func (tm *ToolexecManager) RunAsToolexec() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s <original-tool> [args...]\n", os.Args[0])
		os.Exit(1)
	}
	originalTool := os.Args[1]
	originalArgs := os.Args[2:]
	if !tm.isCompilerTool(originalTool) {
		tm.runOriginalTool(originalTool, originalArgs)
		return
	}
	goFiles, outputDir := tm.extractFilesAndOutputDir(originalArgs)

	if len(goFiles) > 0 {
		userFiles := FilterUserFiles(goFiles)

		if len(userFiles) == 0 {
			tm.runOriginalTool(originalTool, originalArgs)
			return
		}

		if !versionShown {
			_, _ = fmt.Fprintf(os.Stderr, "[goahead] GoAhead Code Generator %s\n", Version)
			_, _ = fmt.Fprintf(os.Stderr, "[goahead] Processing user code with intelligent code generation\n")
			versionShown = true
		}
		workDir := tm.determineWorkDir(userFiles, outputDir)
		tm.runCodegenIfVerbose(workDir, goFiles, userFiles)
	}
	tm.runOriginalTool(originalTool, originalArgs)
}

func (tm *ToolexecManager) isCompilerTool(tool string) bool {
	base := filepath.Base(tool)
	// Only process the Go compiler, skip cgo, asm, link, etc.
	return base == "compile" || base == "compile.exe"
}

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

func (tm *ToolexecManager) runCodegenIfVerbose(workDir string, goFiles, userFiles []string) {
	verbose := os.Getenv("GOAHEAD_VERBOSE") == "1"

	if verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[goahead] Files detected: %v\n", goFiles)
		_, _ = fmt.Fprintf(os.Stderr, "[goahead] User files after filtering: %v\n", userFiles)
		_, _ = fmt.Fprintf(os.Stderr, "[goahead] Running codegen in %s\n", workDir)
		cwd, err := os.Getwd()
		if err == nil {
			_, _ = fmt.Fprintf(os.Stderr, "[goahead] Current working directory: %s\n", cwd)
		}
	}
	if verbose {
		tm.logFileTypes(goFiles)
	}

	if err := RunCodegen(workDir, verbose); err != nil {
		if verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[goahead] Codegen failed: %v\n", err)
		}
	}
}

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

		_, _ = fmt.Fprintf(os.Stderr, "[goahead] File: %s (Type: %s, Location: %s)\n", file, fileType, location)
	}
}

func (tm *ToolexecManager) isSystemFile(file string) bool {
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

func (tm *ToolexecManager) runOriginalTool(tool string, args []string) {
	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}
