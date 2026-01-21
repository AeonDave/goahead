package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// InjectPattern matches //:inject:FuncName
const InjectPattern = `^\s*//\s*:inject:(\w+)\s*$`

// InjectionResult contains the extracted function and its dependencies
type InjectionResult struct {
	FunctionCode string
	Imports      []string
	Constants    string
	Variables    string
	Types        string
}

// Injector handles function injection from helper files
type Injector struct {
	ctx *ProcessorContext
}

// NewInjector creates a new Injector
func NewInjector(ctx *ProcessorContext) *Injector {
	return &Injector{ctx: ctx}
}

// ExtractFunction extracts a function and its dependencies from helper files
func (inj *Injector) ExtractFunction(funcName, sourceDir string) (*InjectionResult, error) {
	// Find the function using hierarchical resolution
	userFunc, helperPath := inj.ctx.ResolveFunction(funcName, sourceDir)
	if userFunc == nil {
		return nil, fmt.Errorf("function '%s' not found in any helper file", funcName)
	}

	// Parse the helper file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, helperPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse helper file %s: %v", helperPath, err)
	}

	result := &InjectionResult{}

	// Extract imports
	for _, imp := range node.Imports {
		var importSpec string
		if imp.Name != nil {
			importSpec = imp.Name.Name + " " + imp.Path.Value
		} else {
			importSpec = imp.Path.Value
		}
		result.Imports = append(result.Imports, importSpec)
	}

	// Build map of helper functions
	funcDecls := make(map[string]*ast.FuncDecl)
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			funcDecls[fn.Name.Name] = fn
		}
	}

	// Ensure the target exists
	funcDecl, ok := funcDecls[funcName]
	if !ok {
		return nil, fmt.Errorf("function '%s' not found in %s", funcName, helperPath)
	}

	// Collect dependent helper functions recursively
	included := make(map[string]*ast.FuncDecl)
	queue := []string{funcName}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if _, already := included[name]; already {
			continue
		}
		fn := funcDecls[name]
		if fn == nil {
			continue
		}
		included[name] = fn

		used := inj.collectUsedIdentifiers(fn)
		for ident := range used {
			if ident == name {
				continue
			}
			if _, exists := funcDecls[ident]; exists {
				queue = append(queue, ident)
			}
		}
	}

	// Collect identifiers used by all included functions
	usedIdents := make(map[string]bool)
	for _, fn := range included {
		for ident := range inj.collectUsedIdentifiers(fn) {
			usedIdents[ident] = true
		}
	}

	// Extract dependencies (const, var, type) that are used
	result.Constants, result.Variables, result.Types = inj.extractDependencies(node, fset, usedIdents)

	// Build function code: target first, then other dependencies (stable order)
	var funcBuf strings.Builder
	if err := printer.Fprint(&funcBuf, fset, funcDecl); err != nil {
		return nil, fmt.Errorf("failed to print function: %v", err)
	}

	// Add dependent helper functions (excluding target), sorted for stability
	var otherNames []string
	for name := range included {
		if name == funcName {
			continue
		}
		otherNames = append(otherNames, name)
	}
	sort.Strings(otherNames)

	for _, name := range otherNames {
		funcBuf.WriteString("\n\n")
		if err := printer.Fprint(&funcBuf, fset, included[name]); err != nil {
			return nil, fmt.Errorf("failed to print dependent function: %v", err)
		}
	}

	result.FunctionCode = funcBuf.String()
	return result, nil
}

// collectUsedIdentifiers finds all identifiers used in a function
func (inj *Injector) collectUsedIdentifiers(fn *ast.FuncDecl) map[string]bool {
	used := make(map[string]bool)

	// Include identifiers from signature (params/results/receiver)
	if fn.Recv != nil {
		ast.Inspect(fn.Recv, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				used[ident.Name] = true
			}
			return true
		})
	}
	if fn.Type != nil {
		ast.Inspect(fn.Type, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				used[ident.Name] = true
			}
			return true
		})
	}

	// Include identifiers from body
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				used[ident.Name] = true
			}
			return true
		})
	}

	return used
}

// extractDependencies extracts const/var/type declarations used by the function
func (inj *Injector) extractDependencies(file *ast.File, fset *token.FileSet, usedIdents map[string]bool) (constants, variables, types string) {
	var constBuf, varBuf, typeBuf strings.Builder

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		switch genDecl.Tok {
		case token.CONST:
			for _, spec := range genDecl.Specs {
				vs := spec.(*ast.ValueSpec)
				for _, name := range vs.Names {
					if usedIdents[name.Name] {
						// Extract this const
						var buf strings.Builder
						printer.Fprint(&buf, fset, &ast.GenDecl{
							Tok:   token.CONST,
							Specs: []ast.Spec{vs},
						})
						constBuf.WriteString(buf.String())
						constBuf.WriteString("\n")
						break
					}
				}
			}
		case token.VAR:
			for _, spec := range genDecl.Specs {
				vs := spec.(*ast.ValueSpec)
				for _, name := range vs.Names {
					if usedIdents[name.Name] {
						var buf strings.Builder
						printer.Fprint(&buf, fset, &ast.GenDecl{
							Tok:   token.VAR,
							Specs: []ast.Spec{vs},
						})
						varBuf.WriteString(buf.String())
						varBuf.WriteString("\n")
						break
					}
				}
			}
		case token.TYPE:
			for _, spec := range genDecl.Specs {
				ts := spec.(*ast.TypeSpec)
				if usedIdents[ts.Name.Name] {
					var buf strings.Builder
					printer.Fprint(&buf, fset, &ast.GenDecl{
						Tok:   token.TYPE,
						Specs: []ast.Spec{ts},
					})
					typeBuf.WriteString(buf.String())
					typeBuf.WriteString("\n")
				}
			}
		}
	}

	return constBuf.String(), varBuf.String(), typeBuf.String()
}

// ProcessFileInjections handles all //:inject: directives in a file
func (inj *Injector) ProcessFileInjections(filePath string, verbose bool) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	sourceDir := filepath.Dir(filePath)
	absSourceDir, _ := filepath.Abs(sourceDir)

	injectRe := regexp.MustCompile(InjectPattern)
	lines := strings.Split(string(content), "\n")

	var newLines []string
	var importsToAdd []string
	var depsToAdd []string
	modified := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		match := injectRe.FindStringSubmatch(line)

		if match != nil {
			funcName := match[1]

			result, err := inj.ExtractFunction(funcName, absSourceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not inject function '%s' in %s: %v\n", funcName, filePath, err)
				newLines = append(newLines, line)
				continue
			}

			// Collect imports
			importsToAdd = append(importsToAdd, result.Imports...)

			// Collect dependencies
			if result.Types != "" {
				depsToAdd = append(depsToAdd, result.Types)
			}
			if result.Constants != "" {
				depsToAdd = append(depsToAdd, result.Constants)
			}
			if result.Variables != "" {
				depsToAdd = append(depsToAdd, result.Variables)
			}

			// Replace the inject marker with the function
			newLines = append(newLines, result.FunctionCode)

			fmt.Fprintf(os.Stderr, "[goahead] Injected function '%s' in %s\n", funcName, filePath)
			modified = true
			continue
		}

		newLines = append(newLines, line)
	}

	if !modified {
		return nil
	}

	// Add imports and dependencies at the appropriate places
	finalContent := inj.insertImportsAndDeps(newLines, importsToAdd, depsToAdd)

	return os.WriteFile(filePath, []byte(finalContent), 0o644)
}

// insertImportsAndDeps adds imports and dependencies to the file content
func (inj *Injector) insertImportsAndDeps(lines []string, imports []string, deps []string) string {
	if len(imports) == 0 && len(deps) == 0 {
		return strings.Join(lines, "\n")
	}

	importSet := make(map[string]bool)
	for _, imp := range imports {
		importSet[imp] = true
	}

	packageLineIdx := -1
	importStart := -1
	importEnd := -1
	importSingle := -1
	inImport := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") && packageLineIdx == -1 {
			packageLineIdx = i
		}
		if strings.HasPrefix(trimmed, "import (") {
			importStart = i
			inImport = true
		}
		if inImport && trimmed == ")" {
			importEnd = i
			inImport = false
		}
		if strings.HasPrefix(trimmed, "import ") && !strings.HasSuffix(trimmed, "(") {
			if importStart == -1 {
				importSingle = i
			}
		}
	}

	var result []string
	insertedDeps := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Convert single-line import into block if needed
		if i == importSingle && len(importSet) > 0 {
			spec := strings.TrimSpace(strings.TrimPrefix(trimmed, "import"))
			result = append(result, "import (")
			if spec != "" {
				result = append(result, "\t"+spec)
			}
			for imp := range importSet {
				if spec == imp {
					continue
				}
				result = append(result, "\t"+imp)
			}
			result = append(result, ")")
			continue
		}

		result = append(result, line)

		// Insert imports after package if none exist
		if i == packageLineIdx && importStart == -1 && importSingle == -1 && len(importSet) > 0 {
			result = append(result, "")
			result = append(result, "import (")
			for imp := range importSet {
				result = append(result, "\t"+imp)
			}
			result = append(result, ")")
		}

		// Extend import block before closing )
		if i == importEnd && len(importSet) > 0 {
			result = result[:len(result)-1]
			for imp := range importSet {
				found := false
				for j := importStart; j <= importEnd; j++ {
					if strings.Contains(lines[j], imp) {
						found = true
						break
					}
				}
				if !found {
					result = append(result, "\t"+imp)
				}
			}
			result = append(result, ")")
		}

		// Insert dependencies after imports
		if !insertedDeps && len(deps) > 0 {
			if i == importEnd || (importStart == -1 && importSingle == -1 && i == packageLineIdx) || i == importSingle {
				result = append(result, "")
				for _, dep := range deps {
					result = append(result, strings.TrimSpace(dep))
				}
				insertedDeps = true
			}
		}
	}

	return strings.Join(result, "\n")
}
