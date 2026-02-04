package internal

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

type FileProcessor struct {
	ctx *ProcessorContext
}

func NewFileProcessor(ctx *ProcessorContext) *FileProcessor {
	return &FileProcessor{ctx: ctx}
}

// CollectAllGoFiles walks the directory tree once and collects all .go files
// It also identifies function files and stores them in ctx.FuncFiles
// Submodules (directories with their own go.mod) are detected and stored separately
func (fp *FileProcessor) CollectAllGoFiles(dir string) ([]string, error) {
	var allFiles []string
	fp.ctx.FuncFiles = []string{}
	fp.ctx.Submodules = []string{}

	// Get absolute path of root dir to compare
	absRootDir, err := filepath.Abs(dir)
	if err != nil {
		absRootDir = dir
	}
	absRootDir = filepath.Clean(absRootDir)

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check for submodule (directory with go.mod that's not the root)
		if d.IsDir() {
			absPath, _ := filepath.Abs(path)
			absPath = filepath.Clean(absPath)
			if !pathsEqual(absPath, absRootDir) {
				goModPath := filepath.Join(path, "go.mod")
				if _, statErr := os.Stat(goModPath); statErr == nil {
					// Found a submodule - record it and skip this directory tree
					fp.ctx.Submodules = append(fp.ctx.Submodules, absPath)
					return filepath.SkipDir
				}
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Check if it's a function file (fast check - reads only first 10 lines)
		if fp.hasFunctionMarker(path) {
			fp.ctx.FuncFiles = append(fp.ctx.FuncFiles, path)
		} else {
			allFiles = append(allFiles, path)
		}
		return nil
	})

	return allFiles, err
}

// FilterFilesWithMarkers quickly checks which files contain placeholder or inject markers
// Uses parallel scanning for speed
func (fp *FileProcessor) FilterFilesWithMarkers(files []string) []string {
	if len(files) == 0 {
		return nil
	}

	// Compile patterns once
	commentRe := regexp.MustCompile(CommentPattern)
	injectRe := regexp.MustCompile(InjectPattern)

	type result struct {
		path      string
		hasMarker bool
	}

	results := make(chan result, len(files))
	var wg sync.WaitGroup

	// Process files in parallel
	semaphore := make(chan struct{}, 32) // Limit concurrent file opens

	for _, path := range files {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			hasMarker := fp.fileHasMarkers(p, commentRe, injectRe)
			results <- result{path: p, hasMarker: hasMarker}
		}(path)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var filtered []string
	for r := range results {
		if r.hasMarker {
			filtered = append(filtered, r.path)
		}
	}

	return filtered
}

// fileHasMarkers quickly scans a file for placeholder or inject markers
func (fp *FileProcessor) fileHasMarkers(path string, commentRe, injectRe *regexp.Regexp) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if commentRe.MatchString(line) || injectRe.MatchString(line) {
			return true
		}
	}
	return false
}

// FindFunctionFiles is kept for backward compatibility but now just wraps CollectAllGoFiles
func (fp *FileProcessor) FindFunctionFiles(dir string) error {
	_, err := fp.CollectAllGoFiles(dir)
	return err
}

func (fp *FileProcessor) hasFunctionMarker(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

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

func (fp *FileProcessor) LoadUserFunctions() error {
	for _, funcFile := range fp.ctx.FuncFiles {
		if err := fp.loadFunctionsFromFile(funcFile); err != nil {
			return fmt.Errorf("error loading functions from %s: %v", funcFile, err)
		}
	}
	return nil
}

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

func (fp *FileProcessor) processFunctionDeclaration(fn *ast.FuncDecl, filePath string) {
	if !fp.isValidFunction(fn) {
		return
	}

	funcName := fn.Name.Name

	// Only exported (uppercase) functions are available for placeholder replacement
	if !gotoken.IsExported(funcName) {
		return
	}

	// Get directory of the helper file
	funcDir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(funcDir)
	if err != nil {
		absDir = funcDir
	}

	// Calculate depth relative to RootDir
	depth := fp.ctx.CalculateDepth(absDir)

	userFunc := &UserFunction{
		Name:       funcName,
		InputTypes: fp.extractInputTypes(fn),
		OutputType: fp.extractOutputType(fn),
		FilePath:   filePath,
		Depth:      depth,
	}

	// Initialize maps if needed
	if fp.ctx.FunctionsByDir[absDir] == nil {
		fp.ctx.FunctionsByDir[absDir] = make(map[string]*UserFunction)
	}
	if fp.ctx.FunctionsByDepth[depth] == nil {
		fp.ctx.FunctionsByDepth[depth] = make(map[string]*UserFunction)
	}

	// Check for duplicate in same directory (this is an error)
	if existingFunc, exists := fp.ctx.FunctionsByDir[absDir][funcName]; exists {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: Duplicate function '%s' in same directory!\n"+
			"  - First definition: %s\n"+
			"  - Second definition: %s\n",
			funcName, existingFunc.FilePath, filePath)
		os.Exit(1)
	}

	// Check for duplicate at same depth (different directories) - this is now an error
	if existingFunc, exists := fp.ctx.FunctionsByDepth[depth][funcName]; exists {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: Duplicate function '%s' at same depth level %d!\n"+
			"  - First definition: %s\n"+
			"  - Second definition: %s\n"+
			"  Hint: Functions at the same depth level must have unique names.\n",
			funcName, depth, existingFunc.FilePath, filePath)
		os.Exit(1)
	}

	// Check for shadowing (function at deeper level shadows one at shallower level)
	fp.checkShadowing(funcName, depth, filePath)

	// Store in depth-specific map
	fp.ctx.FunctionsByDepth[depth][funcName] = userFunc

	// Store in directory-specific map
	fp.ctx.FunctionsByDir[absDir][funcName] = userFunc
}

// checkShadowing warns if this function shadows one from a shallower depth level
func (fp *FileProcessor) checkShadowing(funcName string, funcDepth int, filePath string) {
	// Check all shallower depths (0 to funcDepth-1)
	for depth := 0; depth < funcDepth; depth++ {
		if funcs, ok := fp.ctx.FunctionsByDepth[depth]; ok {
			if existingFunc, exists := funcs[funcName]; exists {
				relPath, _ := filepath.Rel(fp.ctx.RootDir, filePath)
				existingRelPath, _ := filepath.Rel(fp.ctx.RootDir, existingFunc.FilePath)
				if relPath == "" {
					relPath = filePath
				}
				if existingRelPath == "" {
					existingRelPath = existingFunc.FilePath
				}
				_, _ = fmt.Fprintf(os.Stderr, "[goahead] WARNING: '%s' at depth %d (%s) shadows depth %d (%s)\n",
					funcName, funcDepth, relPath, depth, existingRelPath)
				return
			}
		}
	}
}

func (fp *FileProcessor) isValidFunction(fn *ast.FuncDecl) bool {
	return fn.Name.IsExported() || (fn.Name.Name[0] >= 'a' && fn.Name.Name[0] <= 'z')
}

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

func (fp *FileProcessor) extractOutputType(fn *ast.FuncDecl) string {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return typeToString(fn.Type.Results.List[0].Type)
	}
	return ""
}

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
	case *ast.Ellipsis:
		return "..." + typeToString(t.Elt)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + typeToString(t.Value)
		case ast.RECV:
			return "<-chan " + typeToString(t.Value)
		default:
			return "chan " + typeToString(t.Value)
		}
	case *ast.FuncType:
		return "func"
	case *ast.StructType:
		return "struct{}"
	default:
		return "unknown"
	}
}

func (fp *FileProcessor) IsFunctionFile(path string) bool {
	return slices.Contains(fp.ctx.FuncFiles, path)
}

func FilterUserFiles(files []string) []string {
	ctx := newFilterContext(os.Getenv("GOAHEAD_VERBOSE") == "1")
	var userFiles []string

	for _, file := range files {
		include, message := ctx.includeFile(file)
		if include {
			userFiles = append(userFiles, file)
		}
		if ctx.verbose && message != "" {
			_, _ = fmt.Fprintln(os.Stderr, message)
		}
	}

	return userFiles
}

type filterContext struct {
	verbose    bool
	gopath     string
	goroot     string
	absCwd     string
	moduleRoot string
}

func newFilterContext(verbose bool) *filterContext {
	ctx := &filterContext{verbose: verbose}
	ctx.gopath = determineGoPath()
	ctx.goroot = determineGoRoot()
	ctx.absCwd, ctx.moduleRoot = determineWorkspace()
	return ctx
}

func determineGoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return gopath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, "go")
}

func determineGoRoot() string {
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		return goroot
	}
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func determineWorkspace() (string, string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		absCwd = cwd
	}
	return absCwd, findModuleRoot(cwd)
}

func (c *filterContext) includeFile(file string) (bool, string) {
	absFile := c.absolutePath(file)
	if shouldExcludeFile(absFile, c.goroot, c.gopath) {
		return false, fmt.Sprintf("[goahead] Skipping system file: %s", file)
	}
	if isVendorPath(file) {
		return false, fmt.Sprintf("[goahead] Skipping vendor file: %s", file)
	}
	if isStdlibPath(file) {
		return false, fmt.Sprintf("[goahead] Skipping standard library file: %s", file)
	}
	if containsTestDirectory(file) {
		return true, fmt.Sprintf("[goahead] Including test directory file: %s", file)
	}
	if strings.HasSuffix(file, "_test.go") {
		return true, fmt.Sprintf("[goahead] Including test file: %s", file)
	}
	if isLocalPath(file) {
		return true, fmt.Sprintf("[goahead] Including local file: %s", file)
	}
	if isUserFile(absFile, c.absCwd, c.moduleRoot) {
		return true, fmt.Sprintf("[goahead] Including user file: %s", file)
	}
	return false, fmt.Sprintf("[goahead] Skipping non-user file: %s", file)
}

func (c *filterContext) absolutePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

func isVendorPath(path string) bool {
	if strings.Contains(path, "/vendor/") || strings.Contains(path, "\\vendor\\") {
		return true
	}
	return strings.HasPrefix(path, "vendor/") || strings.HasPrefix(path, "vendor\\")
}

func isStdlibPath(path string) bool {
	if !strings.Contains(path, "/src/") {
		return false
	}
	for _, segment := range []string{"/runtime/", "/internal/", "/crypto/", "/encoding/", "/net/", "/os/", "/fmt/"} {
		if strings.Contains(path, segment) {
			return true
		}
	}
	return false
}

func containsTestDirectory(path string) bool {
	return strings.Contains(path, "/test/") || strings.Contains(path, "\\test\\")
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, "./") || filepath.Base(path) == path
}

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

func shouldExcludeFile(absFile, goroot, gopath string) bool {
	if goroot != "" && strings.HasPrefix(absFile, goroot) {
		return true
	}
	if gopath != "" && strings.Contains(absFile, filepath.Join(gopath, "pkg/mod")) {
		return true
	}
	for _, path := range GoInstallPaths {
		if strings.Contains(absFile, path) {
			return true
		}
	}
	for _, path := range SystemPaths {
		if strings.Contains(absFile, path) {
			return true
		}
	}

	if strings.Contains(absFile, "_obj") {
		return true
	}
	if strings.Contains(absFile, "_test") && !strings.HasSuffix(absFile, "_test.go") {
		return true
	}

	return false
}

func isUserFile(absFile, absCwd, moduleRoot string) bool {
	if strings.HasPrefix(absFile, absCwd) {
		return true
	}
	if moduleRoot != "" && strings.HasPrefix(absFile, moduleRoot) {
		return true
	}
	if strings.HasSuffix(absFile, "_test.go") {
		return true
	}
	if !filepath.IsAbs(absFile) {
		return true
	}
	return false
}

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

func (fp *FileProcessor) ProcessDirectory(dir string, verbose bool, codeProcessor *CodeProcessor) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || fp.IsFunctionFile(path) {
			return nil
		}
		if err := codeProcessor.ProcessFile(path, verbose); err != nil {
			return fmt.Errorf("error processing file %s: %v", path, err)
		}
		return nil
	})
}

func (fp *FileProcessor) ProcessDirectoryInjections(dir string, verbose bool, injector *Injector) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || fp.IsFunctionFile(path) {
			return nil
		}
		if err := injector.ProcessFileInjections(path, verbose); err != nil {
			return fmt.Errorf("error processing injections in %s: %v", path, err)
		}
		return nil
	})
}
