package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

const version = "1.0.0"

type UserFunction struct {
	Name       string
	InputTypes []string
	OutputType string
	FilePath   string
}

type ProcessorContext struct {
	Functions   map[string]*UserFunction
	FileSet     *token.FileSet
	CurrentFile string
	FuncFiles   []string
	TempDir     string
}

func main() {
	if len(os.Args) >= 2 && !strings.HasPrefix(os.Args[1], "-") &&
		(strings.Contains(os.Args[1], "compile") || strings.Contains(os.Args[1], "link") || strings.Contains(os.Args[1], "asm")) {
		runAsToolexec()
		return
	}

	var (
		dir     = flag.String("dir", ".", "Directory to process")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
		help    = flag.Bool("help", false, "Show help")
		ver     = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *ver {
		fmt.Printf("goahead version %s\n", version)
		return
	}

	if *verbose {
		fmt.Printf("Running goahead in standalone mode\n")
		fmt.Printf("Processing directory: %s\n", *dir)
	}

	if err := runCodegen(*dir, *verbose); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func showHelp() {
	fmt.Printf(`goahead - Go code generation tool
	
	USAGE:
	    # Install goahead
	    go install github.com/AeonDave/goahead@latest
	
	    # Use with go build
	    go build -toolexec="goahead" main.go
	
	    # Use standalone
	    goahead -dir=.
	
	SETUP:
	    1. Create function files with markers:
	       //go:build exclude
	       //go:ahead functions
	
	    2. Add generation comments in your code:
	       //:functionName:arg1:arg2
	
	    3. Build with toolexec integration:
	       go build -toolexec="goahead" ./...
	
	OPTIONS:
	    -dir string     Directory to process (default ".")
	    -verbose        Enable verbose output
	    -help           Show this help
	    -version        Show version
	
	ENVIRONMENT:
	    GOAHEAD_VERBOSE=1    Enable verbose output in toolexec mode
	
	VERSION: %s
	`, version)
}

func runAsToolexec() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <original-tool> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	originalTool := os.Args[1]
	originalArgs := os.Args[2:]

	if !strings.HasSuffix(originalTool, "compile") && !strings.Contains(originalTool, "compile") {
		runOriginalTool(originalTool, originalArgs)
		return
	}

	var goFiles []string
	var outputDir string

	for i, arg := range originalArgs {
		if strings.HasSuffix(arg, ".go") {
			goFiles = append(goFiles, arg)
		}
		if arg == "-o" && i+1 < len(originalArgs) {
			outputPath := originalArgs[i+1]
			outputDir = filepath.Dir(outputPath)
		}
	}

	if len(goFiles) > 0 {
		workDir := findCommonDir(goFiles)
		if workDir == "" {
			workDir = outputDir
		}
		if workDir == "" {
			workDir = "."
		}

		verbose := os.Getenv("GOAHEAD_VERBOSE") == "1"
		if verbose {
			fmt.Fprintf(os.Stderr, "[goahead] Running codegen in %s\n", workDir)
		}

		if err := runCodegen(workDir, verbose); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[goahead] Codegen failed: %v\n", err)
			}
		}
	}
	runOriginalTool(originalTool, originalArgs)
}

func findCommonDir(files []string) string {
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

func runOriginalTool(tool string, args []string) {
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

func runCodegen(dir string, verbose bool) error {
	if verbose {
		fmt.Printf("Parsed flags:\n")
		fmt.Printf("  dir: '%s'\n", dir)
		fmt.Printf("  verbose: %t\n", verbose)
	}

	ctx := &ProcessorContext{
		Functions: make(map[string]*UserFunction),
		FileSet:   token.NewFileSet(),
	}

	tempDir, err := os.MkdirTemp("", "codegen-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	ctx.TempDir = tempDir

	if err := ctx.findFunctionFiles(dir); err != nil {
		return fmt.Errorf("failed to find function files: %v", err)
	}
	if len(ctx.FuncFiles) == 0 {
		if verbose {
			log.Printf("No function files found (looking for files with '//go:ahead functions' marker)")
		}
		return nil
	}

	if err := ctx.loadUserFunctions(); err != nil {
		return fmt.Errorf("failed to load user functions: %v", err)
	}

	if verbose {
		fmt.Printf("Found %d function files:\n", len(ctx.FuncFiles))
		for _, file := range ctx.FuncFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Printf("Loaded %d user functions:\n", len(ctx.Functions))
		for name, fn := range ctx.Functions {
			fmt.Printf("  - %s(%s) %s\n", name, strings.Join(fn.InputTypes, ", "), fn.OutputType)
		}
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if ctx.isFunctionFile(path) || strings.Contains(path, ctx.TempDir) {
			return nil
		}
		ctx.CurrentFile = path
		return ctx.processGoFile(path, verbose)
	})
	if err != nil {
		return fmt.Errorf("error processing directory: %v", err)
	}

	if verbose {
		fmt.Println("Code generation completed successfully")
	}
	return nil
}

func (ctx *ProcessorContext) findFunctionFiles(dir string) error {
	ctx.FuncFiles = []string{}

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineCount := 0
		hasGoAheadMarker := false

		for scanner.Scan() && lineCount < 10 {
			line := strings.TrimSpace(scanner.Text())
			if line == "//go:ahead functions" {
				hasGoAheadMarker = true
				break
			}
			lineCount++
		}
		if hasGoAheadMarker {
			ctx.FuncFiles = append(ctx.FuncFiles, path)
		}
		return nil
	})
}

func (ctx *ProcessorContext) isFunctionFile(path string) bool {
	return slices.Contains(ctx.FuncFiles, path)
}

func (ctx *ProcessorContext) loadUserFunctions() error {
	for _, funcFile := range ctx.FuncFiles {
		if err := ctx.loadUserFunctionsFromFile(funcFile); err != nil {
			return fmt.Errorf("error loading functions from %s: %v", funcFile, err)
		}
	}
	return nil
}

func (ctx *ProcessorContext) loadUserFunctionsFromFile(filePath string) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read functions file: %v", err)
	}

	node, err := parser.ParseFile(ctx.FileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse functions file: %v", err)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Name.IsExported() || fn.Name.Name[0] >= 'a' && fn.Name.Name[0] <= 'z' {
				funcName := fn.Name.Name
				if existingFunc, exists := ctx.Functions[funcName]; exists {
					log.Fatalf("ERROR: Duplicate function '%s' found!\n"+
						"  - First definition: %s\n"+
						"  - Second definition: %s\n"+
						"Please rename one of the functions to avoid conflicts.",
						funcName, existingFunc.FilePath, filePath)
				}

				userFunc := &UserFunction{
					Name:       funcName,
					InputTypes: []string{},
					FilePath:   filePath,
				}

				if fn.Type.Params != nil {
					for _, param := range fn.Type.Params.List {
						if len(param.Names) == 0 {
							userFunc.InputTypes = append(userFunc.InputTypes, typeToString(param.Type))
						} else {
							typeStr := typeToString(param.Type)
							for range param.Names {
								userFunc.InputTypes = append(userFunc.InputTypes, typeStr)
							}
						}
					}
				}

				if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
					userFunc.OutputType = typeToString(fn.Type.Results.List[0].Type)
				} else {
					userFunc.OutputType = "void"
				}
				ctx.Functions[fn.Name.Name] = userFunc
			}
		}
		return true
	})
	return nil
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
	default:
		return "unknown"
	}
}

func (ctx *ProcessorContext) processGoFile(filePath string, verbose bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	modified := false

	commentPattern := regexp.MustCompile(`^\s*//\s*:([^:]+)(?::(.*))?`)
	for scanner.Scan() {
		line := scanner.Text()
		if commentMatch := commentPattern.FindStringSubmatch(line); commentMatch != nil {
			funcName := strings.TrimSpace(commentMatch[1])
			argsStr := ""
			if len(commentMatch) > 2 && commentMatch[2] != "" {
				argsStr = strings.TrimSpace(commentMatch[2])
			}

			lines = append(lines, line)
			if scanner.Scan() {
				nextLine := scanner.Text()

				result, err := ctx.executeUserFunction(funcName, argsStr)
				if err != nil {
					log.Printf("Warning: Could not execute function '%s': %v", funcName, err)
					lines = append(lines, nextLine)
					continue
				}

				userFunc := ctx.Functions[funcName]
				outputType := userFunc.OutputType
				var newLine string
				var replaced bool

				switch outputType {
				case "string":
					stringPattern := regexp.MustCompile(`"([^"]*)"`)
					if stringPattern.MatchString(nextLine) {
						if firstMatch := stringPattern.FindStringIndex(nextLine); firstMatch != nil {
							escapedString := escapeString(result)
							newLine = nextLine[:firstMatch[0]] + escapedString + nextLine[firstMatch[1]:]
							replaced = true
						}
					}
				case "int", "int32", "int64":
					intPattern := regexp.MustCompile(`\b\d+\b`)
					if firstMatch := intPattern.FindStringIndex(nextLine); firstMatch != nil {
						newLine = nextLine[:firstMatch[0]] + result + nextLine[firstMatch[1]:]
						replaced = true
					}
				case "bool":
					boolPattern := regexp.MustCompile(`\b(true|false)\b`)
					if firstMatch := boolPattern.FindStringIndex(nextLine); firstMatch != nil {
						newLine = nextLine[:firstMatch[0]] + result + nextLine[firstMatch[1]:]
						replaced = true
					}
				default:
					stringPattern := regexp.MustCompile(`"([^"]*)"`)
					if stringPattern.MatchString(nextLine) {
						if firstMatch := stringPattern.FindStringIndex(nextLine); firstMatch != nil {
							escapedString := escapeString(result)
							newLine = nextLine[:firstMatch[0]] + escapedString + nextLine[firstMatch[1]:]
							replaced = true
						}
					}
				}

				if replaced {
					lines = append(lines, newLine)
					modified = true
					if verbose {
						log.Printf("Processed in %s: %s(%s) -> %s", filePath, funcName, argsStr, result)
					}
				} else {
					lines = append(lines, nextLine)
				}
			}
		} else {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	if modified {
		return ctx.writeFile(filePath, lines)
	}
	return nil
}

func (ctx *ProcessorContext) executeUserFunction(funcName string, argsStr string) (string, error) {
	userFunc, exists := ctx.Functions[funcName]
	if !exists {
		return "", fmt.Errorf("function %s not found", funcName)
	}

	args := []string{}
	if argsStr != "" {
		args = strings.Split(argsStr, ":")
		for i, arg := range args {
			args[i] = strings.TrimSpace(arg)
		}
	}

	if len(args) != len(userFunc.InputTypes) {
		return "", fmt.Errorf("function %s expects %d arguments, got %d",
			funcName, len(userFunc.InputTypes), len(args))
	}

	evaluatedArgs := make([]string, len(args))
	for i, arg := range args {
		val, err := ctx.evaluateExpression(arg)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate argument %d (%s): %v", i, arg, err)
		}
		evaluatedArgs[i] = val
	}
	return ctx.callUserFunction(funcName, evaluatedArgs)
}

func (ctx *ProcessorContext) evaluateExpression(expression string) (string, error) {
	if strings.HasPrefix(expression, `"`) && strings.HasSuffix(expression, `"`) {
		return strings.Trim(expression, `"`), nil
	}
	if _, err := strconv.Atoi(expression); err == nil {
		return expression, nil
	}
	if expression == "true" || expression == "false" {
		return expression, nil
	}
	return expression, nil
}

func (ctx *ProcessorContext) callUserFunction(funcName string, args []string) (string, error) {
	userFunc := ctx.Functions[funcName]

	tmplStr := `package main
	
	import (
	 "fmt"
	{{.AdditionalImports}}
	)
	
	{{.UserFunctions}}
	
	func main() {
	 result := {{.FuncName}}({{.Args}})
	 fmt.Print(result)
	}
	`
	var allFuncLines []string
	importSet := make(map[string]bool)

	for _, funcFile := range ctx.FuncFiles {
		userFuncContent, err := os.ReadFile(funcFile)
		if err != nil {
			return "", fmt.Errorf("failed to read user functions file %s: %v", funcFile, err)
		}

		userFuncStr := string(userFuncContent)
		lines := strings.Split(userFuncStr, "\n")
		var funcLines []string
		inFunction := false
		braceCount := 0
		skipNext := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//go:build") ||
				strings.HasPrefix(trimmed, "//go:ahead") ||
				strings.HasPrefix(trimmed, "package ") {
				continue
			}
			if strings.HasPrefix(trimmed, "import") {
				if strings.Contains(trimmed, "(") {
					skipNext = true
				} else {
					if parts := strings.Fields(trimmed); len(parts) >= 2 {
						importPath := strings.Trim(parts[1], `"`)
						importSet[importPath] = true
					}
				}
				continue
			}
			if skipNext {
				if strings.Contains(trimmed, ")") {
					skipNext = false
				} else if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
					importPath := strings.Trim(trimmed, `" 	`)
					if importPath != "" {
						importSet[importPath] = true
					}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "//") && !inFunction {
				continue
			}
			if strings.Contains(trimmed, "func ") && !inFunction {
				inFunction = true
				braceCount = 0
			}
			if inFunction {
				funcLines = append(funcLines, line)
				braceCount += strings.Count(line, "{")
				braceCount -= strings.Count(line, "}")
				if braceCount == 0 && strings.Contains(line, "}") {
					inFunction = false
				}
			}
		}
		allFuncLines = append(allFuncLines, funcLines...)
	}

	formattedArgs := make([]string, len(args))
	for i, arg := range args {
		inputType := userFunc.InputTypes[i]
		switch inputType {
		case "string":
			formattedArgs[i] = "`" + arg + "`"
		case "int", "int32", "int64":
			formattedArgs[i] = arg
		case "bool":
			formattedArgs[i] = arg
		default:
			formattedArgs[i] = "`" + arg + "`"
		}
	}
	var additionalImports []string
	for importPath := range importSet {
		if importPath != "fmt" {
			additionalImports = append(additionalImports, "\t\""+importPath+"\"")
		}
	}

	tmpl := template.Must(template.New("program").Parse(tmplStr))
	data := struct {
		UserFunctions     string
		FuncName          string
		Args              string
		AdditionalImports string
	}{
		UserFunctions:     strings.Join(allFuncLines, "\n"),
		FuncName:          funcName,
		Args:              strings.Join(formattedArgs, ", "),
		AdditionalImports: strings.Join(additionalImports, "\n"),
	}
	tempFile := filepath.Join(ctx.TempDir, "temp_program.go")
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	if err := tmpl.Execute(file, data); err != nil {
		file.Close()
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	file.Close()

	cmd := exec.Command("go", "run", tempFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute temp program: %v\nOutput:\n%s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func (ctx *ProcessorContext) writeFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to file %s: %v", filePath, err)
		}
	}
	return nil
}

func escapeString(s string) string {
	if strings.Contains(s, "\\") {
		return "`" + s + "`"
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
