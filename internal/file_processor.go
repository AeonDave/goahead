package internal

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// FileProcessor gestisce la ricerca e il caricamento delle funzioni
type FileProcessor struct {
	ctx *ProcessorContext
}

// NewFileProcessor crea un nuovo processore di file
func NewFileProcessor(ctx *ProcessorContext) *FileProcessor {
	return &FileProcessor{ctx: ctx}
}

// FindFunctionFiles trova tutti i file che contengono funzioni definite dall'utente
func (fp *FileProcessor) FindFunctionFiles(dir string) error {
	fp.ctx.FuncFiles = []string{}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		if fp.hasFunctionMarker(path) {
			fp.ctx.FuncFiles = append(fp.ctx.FuncFiles, path)
		}
		return nil
	})
}

// hasFunctionMarker verifica se un file contiene il marker delle funzioni
func (fp *FileProcessor) hasFunctionMarker(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && lineCount < 10 {
		line := strings.TrimSpace(scanner.Text())
		if line == FunctionMarker {
			return true
		}
		lineCount++
	}
	return false
}

// LoadUserFunctions carica tutte le funzioni dai file trovati
func (fp *FileProcessor) LoadUserFunctions() error {
	for _, funcFile := range fp.ctx.FuncFiles {
		if err := fp.loadFunctionsFromFile(funcFile); err != nil {
			return fmt.Errorf("error loading functions from %s: %v", funcFile, err)
		}
	}
	return nil
}

// loadFunctionsFromFile carica le funzioni da un singolo file
func (fp *FileProcessor) loadFunctionsFromFile(filePath string) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read functions file: %v", err)
	}

	node, err := parser.ParseFile(fp.ctx.FileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse functions file: %v", err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			fp.processFunctionDeclaration(fn, filePath)
		}
		return true
	})

	return nil
}

// processFunctionDeclaration elabora una dichiarazione di funzione
func (fp *FileProcessor) processFunctionDeclaration(fn *ast.FuncDecl, filePath string) {
	if !fp.isValidFunction(fn) {
		return
	}

	funcName := fn.Name.Name
	if fp.isDuplicateFunction(funcName, filePath) {
		return
	}

	userFunc := &UserFunction{
		Name:       funcName,
		InputTypes: fp.extractInputTypes(fn),
		OutputType: fp.extractOutputType(fn),
		FilePath:   filePath,
	}

	fp.ctx.Functions[funcName] = userFunc
}

// isValidFunction verifica se una funzione è valida per l'elaborazione
func (fp *FileProcessor) isValidFunction(fn *ast.FuncDecl) bool {
	return fn.Name.IsExported() || (fn.Name.Name[0] >= 'a' && fn.Name.Name[0] <= 'z')
}

// isDuplicateFunction verifica e gestisce le funzioni duplicate
func (fp *FileProcessor) isDuplicateFunction(funcName, filePath string) bool {
	if existingFunc, exists := fp.ctx.Functions[funcName]; exists {
		log.Fatalf("ERROR: Duplicate function '%s' found!\n"+
			"  - First definition: %s\n"+
			"  - Second definition: %s\n"+
			"Please rename one of the functions to avoid conflicts.",
			funcName, existingFunc.FilePath, filePath)
		return true
	}
	return false
}

// extractInputTypes estrae i tipi di input da una funzione
func (fp *FileProcessor) extractInputTypes(fn *ast.FuncDecl) []string {
	var inputTypes []string

	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			if len(param.Names) == 0 {
				inputTypes = append(inputTypes, typeToString(param.Type))
			} else {
				typeStr := typeToString(param.Type)
				for range param.Names {
					inputTypes = append(inputTypes, typeStr)
				}
			}
		}
	}

	return inputTypes
}

// extractOutputType estrae il tipo di output da una funzione
func (fp *FileProcessor) extractOutputType(fn *ast.FuncDecl) string {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return typeToString(fn.Type.Results.List[0].Type)
	}
	return "void"
}

// typeToString converte un ast.Expr in stringa
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return "[" + typeToString(t.Len) + "]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}

// IsFunctionFile verifica se un percorso è un file di funzioni
func (fp *FileProcessor) IsFunctionFile(path string) bool {
	return slices.Contains(fp.ctx.FuncFiles, path)
}

// FilterUserFiles filtra i file utente escludendo quelli di sistema
func FilterUserFiles(files []string) []string {
	var userFiles []string
	verbose := os.Getenv("GOAHEAD_VERBOSE") == "1"

	// Get GOPATH and GOROOT to better detect system files
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH if not set
		homeDir, err := os.UserHomeDir()
		if err == nil {
			gopath = filepath.Join(homeDir, "go")
		}
	}

	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		// Try to detect GOROOT using 'go env'
		cmd := exec.Command("go", "env", "GOROOT")
		output, err := cmd.Output()
		if err == nil {
			goroot = strings.TrimSpace(string(output))
		}
	}

	// Get the current working directory for detecting user files
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "." // Fallback
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		absCwd = cwd // Fallback
	}

	// Get the module root directory (go.mod location)
	moduleRoot := findModuleRoot(cwd)

	for _, file := range files {
		// SPECIAL CASE: Any file in the test directory should be kept
		if strings.Contains(file, "/test/") || strings.Contains(file, "\\test\\") {
			userFiles = append(userFiles, file)
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Including test directory file: %s\n", file)
			}
			continue
		}

		// Always include test files
		if strings.HasSuffix(file, "_test.go") {
			userFiles = append(userFiles, file)
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Including test file: %s\n", file)
			}
			continue
		}

		// Always include files in current directory
		if strings.HasPrefix(file, "./") || filepath.Base(file) == file {
			userFiles = append(userFiles, file)
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Including local file: %s\n", file)
			}
			continue
		}

		absFile, err := filepath.Abs(file)
		if err != nil {
			absFile = file // Fallback if we can't get absolute path
		}

		// Skip system files
		if shouldExcludeFile(absFile, goroot, gopath) {
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Skipping system file: %s\n", file)
			}
			continue
		}

		// Include files in the current working directory or module
		if isUserFile(absFile, absCwd, moduleRoot) {
			userFiles = append(userFiles, file)
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Including user file: %s\n", file)
			}
		} else if verbose {
			fmt.Fprintf(os.Stderr, "[goahead] Skipping non-user file: %s\n", file)
		}
	}

	return userFiles
}

// findModuleRoot finds the directory containing go.mod
func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root directory
			return ""
		}
		dir = parent
	}
}

// shouldExcludeFile verifica se un file deve essere escluso
func shouldExcludeFile(absFile, goroot, gopath string) bool {
	// Skip Go installation files
	if goroot != "" && strings.HasPrefix(absFile, goroot) {
		return true
	}

	// Skip standard GOPATH module files
	if gopath != "" && strings.Contains(absFile, filepath.Join(gopath, "pkg/mod")) {
		return true
	}

	// Check against known system paths
	for _, path := range GoInstallPaths {
		if strings.Contains(absFile, path) {
			return true
		}
	}

	// Check against known system directory patterns
	for _, path := range SystemPaths {
		if strings.Contains(absFile, path) {
			return true
		}
	}

	// Skip files from go test temporary directory (_obj and similar)
	if strings.Contains(absFile, "_obj") {
		return true
	}

	// Special handling for test files - only exclude test files in system paths,
	// but allow user project test files
	if strings.Contains(absFile, "_test") && !strings.HasSuffix(absFile, "_test.go") {
		return true
	}

	return false
}

// isUserFile verifica se un file è un file utente
func isUserFile(absFile, absCwd, moduleRoot string) bool {
	// Files in the current working directory
	if strings.HasPrefix(absFile, absCwd) {
		return true
	}

	// Files in the module directory
	if moduleRoot != "" && strings.HasPrefix(absFile, moduleRoot) {
		return true
	}

	// Always include test files
	if strings.HasSuffix(absFile, "_test.go") {
		return true
	}

	// If the file is not absolute, consider it a user file
	if !filepath.IsAbs(absFile) {
		return true
	}

	return false
}

// FindCommonDir trova la directory comune di un set di file
func FindCommonDir(files []string) string {
	if len(files) == 0 {
		return ""
	}

	commonDir := filepath.Dir(files[0])
	for _, file := range files[1:] {
		dir := filepath.Dir(file)
		for !strings.HasPrefix(dir, commonDir) && commonDir != "." && commonDir != "/" {
			commonDir = filepath.Dir(commonDir)
		}
	}
	return commonDir
}

// ProcessDirectory elabora tutti i file in una directory
func (fp *FileProcessor) ProcessDirectory(dir string, verbose bool, codeProcessor *CodeProcessor) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || fp.IsFunctionFile(path) {
			return nil
		}

		if verbose {
			fmt.Printf("Processing file: %s\n", path)
		}

		if err := codeProcessor.ProcessFile(path, verbose); err != nil {
			return fmt.Errorf("error processing file %s: %v", path, err)
		}

		return nil
	})
}
