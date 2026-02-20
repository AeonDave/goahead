package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	gotoken "go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

const evalFmtAlias = "goaheadfmt"

var executionTemplate = template.Must(template.New("program").Parse(ExecutionTemplate))
var executionBatchTemplate = template.Must(template.New("programBatch").Parse(ExecutionBatchTemplate))

type invocationKind int

const (
	invocationUser invocationKind = iota
	invocationExternal
)

type callTarget struct {
	kind           invocationKind
	userFunc       *UserFunction
	callExpr       string
	packageAlias   string
	packagePath    string
	importResolved bool
}

type argumentKind int

const (
	argumentExpression argumentKind = iota
	argumentString
	argumentBool
	argumentInt
	argumentFloat
)

type argument struct {
	Raw             string
	Normalized      string
	Kind            argumentKind
	AutoQuote       bool
	ForceExpression bool
}

type FunctionExecutor struct {
	ctx *ProcessorContext

	cache map[string]string

	// Cache for prepared code per directory
	preparedByDir map[string]*preparedCode

	// Cache helper files by depth to avoid repeated scans
	helperFilesByDepth map[int][]string

	stdImportMap map[string]string
	stdListErr   error
}

type BatchCall struct {
	FuncName string
	ArgsStr  string
}

type BatchResult struct {
	Result   string
	UserFunc *UserFunction
	Err      error
}

type preparedCode struct {
	source    string
	importSet map[string]struct{}
}

func NewFunctionExecutor(ctx *ProcessorContext) *FunctionExecutor {
	return &FunctionExecutor{
		ctx:           ctx,
		cache:         make(map[string]string),
		preparedByDir: make(map[string]*preparedCode),
	}
}

func (fe *FunctionExecutor) Prepare() error {
	// No-op now, preparation happens on demand per directory
	return nil
}

func (fe *FunctionExecutor) ExecuteFunction(funcName string, argsStr string, sourceDir string) (string, *UserFunction, error) {
	args, err := fe.parseArguments(argsStr)
	if err != nil {
		return "", nil, err
	}

	target, err := fe.determineTarget(funcName, sourceDir)
	if err != nil {
		return "", nil, err
	}

	// Include sourceDir in cache key for hierarchical resolution
	key, err := fe.cacheKeyWithDir(target, args, sourceDir)
	if err != nil {
		return "", nil, err
	}
	if cached, ok := fe.cache[key]; ok {
		return cached, target.userFunc, nil
	}

	formattedArgs, err := fe.formatArguments(target, args)
	if err != nil {
		return "", nil, err
	}

	callExpr := target.callExpr
	if len(formattedArgs) > 0 {
		callExpr = fmt.Sprintf("%s(%s)", target.callExpr, strings.Join(formattedArgs, ", "))
	} else {
		callExpr = fmt.Sprintf("%s()", target.callExpr)
	}

	program, err := fe.buildProgramForDir(target, callExpr, sourceDir)
	if err != nil {
		return "", nil, err
	}

	result, err := fe.executeProgram(program)
	if err != nil {
		if target.kind == invocationExternal && !target.importResolved {
			suggestion := fmt.Sprintf("%s=%s", target.packageAlias, target.packagePath)
			if target.packagePath == target.packageAlias && fe.stdListErr != nil {
				suggestion = fmt.Sprintf("%s=<import path>", target.packageAlias)
			}
			extra := ""
			if fe.stdListErr != nil {
				extra = fmt.Sprintf(" (automatic standard library resolution failed: %v)", fe.stdListErr)
			}
			return "", nil, fmt.Errorf("%w. Add //go:ahead import %s in a function file to declare the package alias%s", err, suggestion, extra)
		}
		return "", nil, err
	}

	fe.cache[key] = result
	return result, target.userFunc, nil
}

func (fe *FunctionExecutor) ExecuteBatch(calls []BatchCall, sourceDir string) []BatchResult {
	results := make([]BatchResult, len(calls))
	if len(calls) == 0 {
		return results
	}

	type pendingCall struct {
		index    int
		callExpr string
		target   callTarget
		cacheKey string
	}

	var pending []pendingCall
	callExprs := make([]string, 0, len(calls))
	targets := make([]callTarget, 0, len(calls))

	for i, call := range calls {
		args, err := fe.parseArguments(call.ArgsStr)
		if err != nil {
			results[i].Err = err
			continue
		}

		target, err := fe.determineTarget(call.FuncName, sourceDir)
		if err != nil {
			results[i].Err = err
			continue
		}

		key, err := fe.cacheKeyWithDir(target, args, sourceDir)
		if err != nil {
			results[i].Err = err
			continue
		}
		if cached, ok := fe.cache[key]; ok {
			results[i] = BatchResult{Result: cached, UserFunc: target.userFunc}
			continue
		}

		formattedArgs, err := fe.formatArguments(target, args)
		if err != nil {
			results[i].Err = err
			continue
		}

		callExpr := target.callExpr
		if len(formattedArgs) > 0 {
			callExpr = fmt.Sprintf("%s(%s)", target.callExpr, strings.Join(formattedArgs, ", "))
		} else {
			callExpr = fmt.Sprintf("%s()", target.callExpr)
		}

		pending = append(pending, pendingCall{
			index:    i,
			callExpr: callExpr,
			target:   target,
			cacheKey: key,
		})
		callExprs = append(callExprs, callExpr)
		targets = append(targets, target)
	}

	if len(pending) == 0 {
		return results
	}

	program, err := fe.buildProgramForDirBatch(targets, callExprs, sourceDir)
	if err != nil {
		for _, call := range pending {
			results[call.index].Err = err
		}
		return results
	}

	output, err := fe.executeProgram(program)
	if err != nil {
		for _, call := range pending {
			results[call.index].Err = err
		}
		return results
	}

	lines := splitOutputLines(output)
	if len(lines) != len(pending) {
		err := fmt.Errorf("unexpected batch output lines: expected %d got %d", len(pending), len(lines))
		for _, call := range pending {
			results[call.index].Err = err
		}
		return results
	}

	for i, call := range pending {
		result := lines[i]
		fe.cache[call.cacheKey] = result
		results[call.index] = BatchResult{Result: result, UserFunc: call.target.userFunc}
	}

	return results
}

func (fe *FunctionExecutor) parseArguments(argsStr string) ([]argument, error) {
	if strings.TrimSpace(argsStr) == "" {
		return nil, nil
	}

	rawArgs, err := splitArguments(argsStr)
	if err != nil {
		return nil, err
	}

	args := make([]argument, len(rawArgs))
	for i, token := range rawArgs {
		args[i] = classifyArgument(token)
	}
	return args, nil
}

func (fe *FunctionExecutor) determineTarget(funcName string, sourceDir string) (callTarget, error) {
	// Use hierarchical resolution: walk up from sourceDir to find the function
	if fn, helperPath := fe.ctx.ResolveFunction(funcName, sourceDir); fn != nil {
		_ = helperPath // Used for logging in caller
		return callTarget{
			kind:     invocationUser,
			userFunc: fn,
			callExpr: funcName,
		}, nil
	}

	alias, remainder, ok := strings.Cut(funcName, ".")
	if !ok || alias == "" || remainder == "" {
		// Provide helpful error message
		// Check if it's a lowercase function (unexported)
		if len(funcName) > 0 && funcName[0] >= 'a' && funcName[0] <= 'z' {
			return callTarget{}, fmt.Errorf("function '%s' not found (note: only exported/uppercase functions are available)", funcName)
		}
		return callTarget{}, fmt.Errorf("function '%s' not found; define it in a //go:ahead functions file", funcName)
	}

	path, resolved := fe.resolveImportPath(alias)

	return callTarget{
		kind:           invocationExternal,
		callExpr:       funcName,
		packageAlias:   alias,
		packagePath:    path,
		importResolved: resolved,
	}, nil
}

func (fe *FunctionExecutor) resolveImportPath(alias string) (string, bool) {
	if alias == "" {
		return "", false
	}
	fe.ensureStdImportMap()
	if path, ok := fe.stdImportMap[alias]; ok && path != "" {
		return path, true
	}
	return alias, false
}

func (fe *FunctionExecutor) formatArguments(target callTarget, args []argument) ([]string, error) {
	if target.kind != invocationUser {
		return formatExternalArguments(args), nil
	}
	return formatUserArguments(target.userFunc, args)
}

func formatExternalArguments(args []argument) []string {
	formatted := make([]string, len(args))
	for i, arg := range args {
		formatted[i] = argDisplayForExternal(arg)
	}
	return formatted
}

func formatUserArguments(fn *UserFunction, args []argument) ([]string, error) {
	expected := fn.InputTypes

	// Check for variadic function (last param starts with ...)
	isVariadic := len(expected) > 0 && strings.HasPrefix(expected[len(expected)-1], "...")

	if isVariadic {
		// For variadic functions, we need at least (len(expected) - 1) arguments
		minArgs := len(expected) - 1
		if len(args) < minArgs {
			return nil, fmt.Errorf("function %s expects at least %d arguments, got %d", fn.Name, minArgs, len(args))
		}
	} else {
		if len(expected) != len(args) {
			return nil, fmt.Errorf("function %s expects %d arguments, got %d", fn.Name, len(expected), len(args))
		}
	}

	formatted := make([]string, len(args))
	for i, arg := range args {
		var typ string
		if i < len(expected)-1 || !isVariadic {
			typ = expected[i]
		} else {
			// Variadic argument: extract element type from "...T"
			typ = strings.TrimPrefix(expected[len(expected)-1], "...")
		}

		value, err := formatArgumentForType(arg, typ)
		if err != nil {
			return nil, fmt.Errorf("argument %d for %s: %w", i, fn.Name, err)
		}
		formatted[i] = value
	}
	return formatted, nil
}

func (fe *FunctionExecutor) buildProgramForDir(target callTarget, callExpr string, sourceDir string) (string, error) {
	prepared, err := fe.ensurePreparedForDir(sourceDir)
	if err != nil {
		return "", err
	}

	importSet := make(map[string]struct{})
	for spec := range prepared.importSet {
		importSet[spec] = struct{}{}
	}

	if target.packagePath != "" {
		if spec := buildImportSpec(target.packageAlias, target.packagePath); spec != "" {
			importSet[spec] = struct{}{}
		}
	}

	imports := make([]string, 0, len(importSet))
	for spec := range importSet {
		imports = append(imports, spec)
	}
	sort.Strings(imports)

	data := struct {
		Imports  []string
		UserCode string
		CallExpr string
		FmtAlias string
	}{
		Imports:  imports,
		UserCode: strings.TrimSpace(prepared.source),
		CallExpr: callExpr,
		FmtAlias: evalFmtAlias,
	}

	var builder strings.Builder
	if err := executionTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return "", fmt.Errorf("failed to format generated program: %v", err)
	}

	return string(formatted), nil
}

func (fe *FunctionExecutor) buildProgramForDirBatch(targets []callTarget, callExprs []string, sourceDir string) (string, error) {
	prepared, err := fe.ensurePreparedForDir(sourceDir)
	if err != nil {
		return "", err
	}

	importSet := make(map[string]struct{})
	for spec := range prepared.importSet {
		importSet[spec] = struct{}{}
	}

	for _, target := range targets {
		if target.packagePath != "" {
			if spec := buildImportSpec(target.packageAlias, target.packagePath); spec != "" {
				importSet[spec] = struct{}{}
			}
		}
	}

	imports := make([]string, 0, len(importSet))
	for spec := range importSet {
		imports = append(imports, spec)
	}
	sort.Strings(imports)

	data := struct {
		Imports  []string
		UserCode string
		Calls    []string
		FmtAlias string
	}{
		Imports:  imports,
		UserCode: strings.TrimSpace(prepared.source),
		Calls:    callExprs,
		FmtAlias: evalFmtAlias,
	}

	var builder strings.Builder
	if err := executionBatchTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return "", fmt.Errorf("failed to format generated program: %v", err)
	}

	return string(formatted), nil
}

// ensurePreparedForDir prepares code with only the declarations visible from sourceDir
func (fe *FunctionExecutor) ensurePreparedForDir(sourceDir string) (*preparedCode, error) {
	if prepared, ok := fe.preparedByDir[sourceDir]; ok {
		return prepared, nil
	}

	// Collect visible helper files by walking up from sourceDir
	visibleFiles := fe.collectVisibleHelperFiles(sourceDir)

	var pieces []string
	importSet := make(map[string]struct{})
	seenIdentifiers := make(map[string]bool)

	// Process files in order from closest to furthest (local shadows global)
	for _, file := range visibleFiles {
		code, imports, identifiers := fe.processFunctionFileWithNames(file)

		// Filter out declarations that are already defined (shadowed)
		filteredCode := fe.filterShadowedDeclarations(code, identifiers, seenIdentifiers)

		if filteredCode != "" {
			pieces = append(pieces, filteredCode)
		}
		for spec := range imports {
			importSet[spec] = struct{}{}
		}
		// Mark these identifiers as seen
		for _, id := range identifiers {
			seenIdentifiers[id] = true
		}
	}

	prepared := &preparedCode{
		source:    strings.Join(pieces, "\n\n"),
		importSet: importSet,
	}
	fe.preparedByDir[sourceDir] = prepared

	return prepared, nil
}

// collectVisibleHelperFiles returns helper files visible from sourceDir using depth-based resolution.
// All project helper files are visible everywhere; depth determines shadowing priority.
// Files are ordered: closest depth first (for shadowing), then deeper depths (lower priority).
func (fe *FunctionExecutor) collectVisibleHelperFiles(sourceDir string) []string {
	var result []string
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		absSourceDir = sourceDir
	}
	sourceDepth := fe.ctx.CalculateDepth(absSourceDir)

	depthToFiles := fe.helperFilesByDepth
	if depthToFiles == nil {
		depthToFiles = fe.buildHelperFilesByDepth()
		fe.helperFilesByDepth = depthToFiles
	}

	// Collect files from sourceDepth down to 0 (closest definitions take priority)
	for depth := sourceDepth; depth >= 0; depth-- {
		if files, ok := depthToFiles[depth]; ok {
			result = append(result, files...)
		}
	}

	// Also include files from deeper depths (available project-wide, lower priority)
	maxDepth := fe.ctx.GetMaxDepth()
	for depth := sourceDepth + 1; depth <= maxDepth; depth++ {
		if files, ok := depthToFiles[depth]; ok {
			result = append(result, files...)
		}
	}

	return result
}

func (fe *FunctionExecutor) buildHelperFilesByDepth() map[int][]string {
	depthToFiles := make(map[int][]string)
	for _, file := range fe.ctx.FuncFiles {
		dir := filepath.Dir(file)
		absDir, err := filepath.Abs(dir)
		if err != nil {
			absDir = dir
		}
		depth := fe.ctx.CalculateDepth(absDir)
		depthToFiles[depth] = append(depthToFiles[depth], file)
	}
	for depth := range depthToFiles {
		sort.Strings(depthToFiles[depth])
	}
	return depthToFiles
}

// filterShadowedDeclarations removes declarations (func/var/const/type) that are already in seenIdentifiers
func (fe *FunctionExecutor) filterShadowedDeclarations(code string, identifiers []string, seen map[string]bool) string {
	// Quick check: if no overlap, return original code
	hasOverlap := false
	for _, id := range identifiers {
		if seen[id] {
			hasOverlap = true
			break
		}
	}
	if !hasOverlap {
		return code
	}

	// Need to filter - parse and remove shadowed declarations
	lines := strings.Split(code, "\n")
	var result []string
	inBlock := false
	skipBlock := false
	braceCount := 0
	parenCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			// Check if this starts a declaration we should skip
			if strings.HasPrefix(trimmed, "func ") {
				name := extractFuncName(trimmed)
				if name != "" && seen[name] {
					skipBlock = true
					inBlock = true
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
					continue
				}
				inBlock = true
				braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			} else if strings.HasPrefix(trimmed, "var ") {
				names := extractVarNames(trimmed)
				if namesOverlap(names, seen) {
					skipBlock = true
					inBlock = true
					parenCount = strings.Count(line, "(") - strings.Count(line, ")")
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
					if parenCount == 0 && braceCount == 0 && !strings.HasSuffix(trimmed, "(") {
						// Single-line var declaration
						inBlock = false
						skipBlock = false
					}
					continue
				}
				inBlock = true
				parenCount = strings.Count(line, "(") - strings.Count(line, ")")
				braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			} else if strings.HasPrefix(trimmed, "const ") {
				names := extractConstNames(trimmed)
				if namesOverlap(names, seen) {
					skipBlock = true
					inBlock = true
					parenCount = strings.Count(line, "(") - strings.Count(line, ")")
					if parenCount == 0 && !strings.HasSuffix(trimmed, "(") {
						// Single-line const declaration
						inBlock = false
						skipBlock = false
					}
					continue
				}
				inBlock = true
				parenCount = strings.Count(line, "(") - strings.Count(line, ")")
			} else if strings.HasPrefix(trimmed, "type ") {
				names := extractTypeNames(trimmed)
				if namesOverlap(names, seen) {
					skipBlock = true
					inBlock = true
					parenCount = strings.Count(line, "(") - strings.Count(line, ")")
					braceCount = strings.Count(line, "{") - strings.Count(line, "}")
					if parenCount == 0 && braceCount == 0 && !strings.HasSuffix(trimmed, "(") && !strings.HasSuffix(trimmed, "{") {
						// Single-line type declaration
						inBlock = false
						skipBlock = false
					}
					continue
				}
				inBlock = true
				parenCount = strings.Count(line, "(") - strings.Count(line, ")")
				braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			}
		} else {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			parenCount += strings.Count(line, "(") - strings.Count(line, ")")
			if braceCount <= 0 && parenCount <= 0 {
				inBlock = false
				if skipBlock {
					skipBlock = false
					continue
				}
			}
		}

		if !skipBlock {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// extractFuncName extracts the function name from a "func name(" line
func extractFuncName(line string) string {
	// Remove "func " prefix
	rest := strings.TrimPrefix(line, "func ")
	// Find opening paren
	parenIdx := strings.Index(rest, "(")
	if parenIdx == -1 {
		return ""
	}
	return strings.TrimSpace(rest[:parenIdx])
}

// extractVarNames extracts variable names from a "var" declaration line
// Handles: "var x int", "var x, y int", "var x = 1", "var ("
func extractVarNames(line string) []string {
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimPrefix(trimmed, "var ")
	rest = strings.TrimSpace(rest)

	// Check for block start
	if rest == "(" || rest == "" {
		return nil
	}

	// Find the end of the name(s) - could be space, =, or type
	var names []string
	// Split by comma for multiple names
	if eqIdx := strings.Index(rest, "="); eqIdx != -1 {
		rest = rest[:eqIdx]
	}
	// Remove type annotation
	for _, sep := range []string{" int", " string", " bool", " float", " byte", " rune", " uint", " ["} {
		if idx := strings.Index(rest, sep); idx != -1 {
			rest = rest[:idx]
			break
		}
	}
	// Also handle custom types (anything after space)
	if spaceIdx := strings.Index(rest, " "); spaceIdx != -1 {
		rest = rest[:spaceIdx]
	}

	for _, name := range strings.Split(rest, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// extractConstNames extracts constant names from a "const" declaration line
func extractConstNames(line string) []string {
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimPrefix(trimmed, "const ")
	rest = strings.TrimSpace(rest)

	// Check for block start
	if rest == "(" || rest == "" {
		return nil
	}

	// Find the end of the name(s)
	var names []string
	if eqIdx := strings.Index(rest, "="); eqIdx != -1 {
		rest = rest[:eqIdx]
	}
	// Remove type annotation
	if spaceIdx := strings.Index(rest, " "); spaceIdx != -1 {
		rest = rest[:spaceIdx]
	}

	for _, name := range strings.Split(rest, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// extractTypeNames extracts type names from a "type" declaration line
// Handles: "type X struct", "type X = int", "type ("
func extractTypeNames(line string) []string {
	trimmed := strings.TrimSpace(line)
	rest := strings.TrimPrefix(trimmed, "type ")
	rest = strings.TrimSpace(rest)

	// Check for block start
	if rest == "(" || rest == "" {
		return nil
	}

	// Find the type name (first identifier)
	var name string
	for i, r := range rest {
		if r == ' ' || r == '=' || r == '[' {
			name = rest[:i]
			break
		}
	}
	if name == "" {
		name = rest
	}
	name = strings.TrimSpace(name)
	if name != "" {
		return []string{name}
	}
	return nil
}

// namesOverlap checks if any name in the list is in the seen map
func namesOverlap(names []string, seen map[string]bool) bool {
	for _, name := range names {
		if seen[name] {
			return true
		}
	}
	return false
}

func (fe *FunctionExecutor) executeProgram(program string) (string, error) {
	tempFile := filepath.Join(fe.ctx.TempDir, "goahead_eval.go")
	if err := os.WriteFile(tempFile, []byte(program), 0o600); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}

	cmd := exec.Command("go", "run", tempFile)
	cmd.Env = sanitizeGoEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		// On Windows, "go run" may fail to clean up temp executables
		// (e.g. "go: unlinkat ... Access is denied.") causing a non-zero
		// exit even though the program itself executed successfully.
		// If the only stderr content is cleanup errors, use stdout.
		if stdoutStr != "" && IsGoCleanupError(stderrStr) {
			return strings.TrimSpace(stdoutStr), nil
		}
		return "", fmt.Errorf("failed to execute temp program: %v\nOutput:\n%s%s", err, stdoutStr, stderrStr)
	}

	return strings.TrimSpace(stdoutStr), nil
}

// IsGoCleanupError returns true when every non-blank line in stderr is a
// Go toolchain file-cleanup message (e.g. "go: unlinkat … Access is denied."
// or "go: removing … Access is denied.").  These arise on Windows when
// antivirus or filesystem locks prevent deletion of temp build artefacts
// and are harmless when the program itself already produced its output.
func IsGoCleanupError(stderr string) bool {
	if strings.TrimSpace(stderr) == "" {
		return false
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "go: unlinkat") || strings.HasPrefix(line, "go: removing") {
			continue
		}
		return false // non-cleanup error present
	}
	return true
}

func splitOutputLines(output string) []string {
	if output == "" {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	lines := make([]string, 0, 8)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func (fe *FunctionExecutor) processFunctionFile(path string) (string, map[string]struct{}) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", make(map[string]struct{})
	}

	lines := strings.Split(string(content), "\n")
	var builder strings.Builder
	imports := make(map[string]struct{})

	inBlock := false // inside func, const, var, type block
	inImportBlock := false
	braceCount := 0
	parenCount := 0 // for const/var/type blocks with ()

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip build tags and package declaration
		if strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "// +build") {
			continue
		}
		if strings.HasPrefix(trimmed, "//go:ahead") {
			continue
		}
		if strings.HasPrefix(trimmed, "package ") {
			continue
		}

		// Handle import statements
		if strings.HasPrefix(trimmed, "import ") {
			if strings.HasSuffix(trimmed, "(") {
				inImportBlock = true
			} else {
				spec := strings.TrimSpace(strings.TrimPrefix(trimmed, "import"))
				if spec != "" {
					imports[spec] = struct{}{}
				}
			}
			continue
		}
		if inImportBlock {
			if trimmed == ")" {
				inImportBlock = false
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			imports[trimmed] = struct{}{}
			continue
		}

		// Skip standalone comments when not in a block
		if strings.HasPrefix(trimmed, "//") && !inBlock {
			continue
		}

		// Detect start of top-level declarations
		if !inBlock {
			if strings.HasPrefix(trimmed, "func ") {
				inBlock = true
				braceCount = 0
			} else if strings.HasPrefix(trimmed, "const ") ||
				strings.HasPrefix(trimmed, "var ") ||
				strings.HasPrefix(trimmed, "type ") {
				inBlock = true
				parenCount = 0
				braceCount = 0
			}
		}

		if inBlock {
			builder.WriteString(line)
			builder.WriteByte('\n')

			// Count braces and parens to detect end of block
			for _, r := range line {
				switch r {
				case '{':
					braceCount++
				case '}':
					braceCount--
				case '(':
					parenCount++
				case ')':
					parenCount--
				}
			}

			// Check if block is complete
			// For func: ends when braceCount returns to 0 after being > 0
			// For const/var/type: ends when line doesn't end with ( and parenCount == 0, or single line
			if braceCount == 0 && parenCount == 0 {
				// Check if it's a complete declaration
				if strings.Contains(line, "}") ||
					strings.Contains(line, ")") ||
					(!strings.HasSuffix(trimmed, "(") && !strings.HasSuffix(trimmed, "{") && !strings.HasSuffix(trimmed, ",")) {
					inBlock = false
					builder.WriteByte('\n')
				}
			}
		}
	}

	return strings.TrimSpace(builder.String()), imports
}

// processFunctionFileWithNames is like processFunctionFile but also returns all identifier names
// (functions, variables, constants, types) for proper shadowing support
func (fe *FunctionExecutor) processFunctionFileWithNames(path string) (string, map[string]struct{}, []string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", make(map[string]struct{}), nil
	}

	lines := strings.Split(string(content), "\n")
	var builder strings.Builder
	imports := make(map[string]struct{})
	var identifiers []string

	inBlock := false
	inImportBlock := false
	braceCount := 0
	parenCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "// +build") {
			continue
		}
		if strings.HasPrefix(trimmed, "//go:ahead") {
			continue
		}
		if strings.HasPrefix(trimmed, "package ") {
			continue
		}

		if strings.HasPrefix(trimmed, "import ") {
			if strings.HasSuffix(trimmed, "(") {
				inImportBlock = true
			} else {
				spec := strings.TrimSpace(strings.TrimPrefix(trimmed, "import"))
				if spec != "" {
					imports[spec] = struct{}{}
				}
			}
			continue
		}
		if inImportBlock {
			if trimmed == ")" {
				inImportBlock = false
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			imports[trimmed] = struct{}{}
			continue
		}

		if strings.HasPrefix(trimmed, "//") && !inBlock {
			continue
		}

		if !inBlock {
			if strings.HasPrefix(trimmed, "func ") {
				inBlock = true
				braceCount = 0
				funcName := extractFuncName(trimmed)
				// Only include exported functions for placeholder usage
				if funcName != "" && gotoken.IsExported(funcName) {
					identifiers = append(identifiers, funcName)
				}
			} else if strings.HasPrefix(trimmed, "var ") {
				inBlock = true
				parenCount = 0
				braceCount = 0
				varNames := extractVarNames(trimmed)
				// Only include exported variables
				for _, name := range varNames {
					if gotoken.IsExported(name) {
						identifiers = append(identifiers, name)
					}
				}
			} else if strings.HasPrefix(trimmed, "const ") {
				inBlock = true
				parenCount = 0
				braceCount = 0
				constNames := extractConstNames(trimmed)
				// Only include exported constants
				for _, name := range constNames {
					if gotoken.IsExported(name) {
						identifiers = append(identifiers, name)
					}
				}
			} else if strings.HasPrefix(trimmed, "type ") {
				inBlock = true
				parenCount = 0
				braceCount = 0
				typeNames := extractTypeNames(trimmed)
				// Only include exported types
				for _, name := range typeNames {
					if gotoken.IsExported(name) {
						identifiers = append(identifiers, name)
					}
				}
			}
		}

		if inBlock {
			builder.WriteString(line)
			builder.WriteByte('\n')

			for _, r := range line {
				switch r {
				case '{':
					braceCount++
				case '}':
					braceCount--
				case '(':
					parenCount++
				case ')':
					parenCount--
				}
			}

			if braceCount == 0 && parenCount == 0 {
				if strings.Contains(line, "}") ||
					strings.Contains(line, ")") ||
					(!strings.HasSuffix(trimmed, "(") && !strings.HasSuffix(trimmed, "{") && !strings.HasSuffix(trimmed, ",")) {
					inBlock = false
					builder.WriteByte('\n')
				}
			}
		}
	}

	return strings.TrimSpace(builder.String()), imports, identifiers
}

func splitArguments(input string) ([]string, error) {
	var (
		parts      []string
		current    strings.Builder
		inQuote    bool
		quote      rune
		escape     bool
		braceDepth int
		parenDepth int
		brackDepth int
	)

	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\' && inQuote:
			current.WriteRune(r)
			escape = true
		case inQuote:
			current.WriteRune(r)
			if r == quote {
				inQuote = false
			}
		case r == '"' || r == '\'' || r == '`':
			inQuote = true
			quote = r
			current.WriteRune(r)
		case r == '{':
			braceDepth++
			current.WriteRune(r)
		case r == '}':
			braceDepth--
			current.WriteRune(r)
		case r == '(':
			parenDepth++
			current.WriteRune(r)
		case r == ')':
			parenDepth--
			current.WriteRune(r)
		case r == '[':
			brackDepth++
			current.WriteRune(r)
		case r == ']':
			brackDepth--
			current.WriteRune(r)
		case r == ':' && braceDepth == 0 && parenDepth == 0 && brackDepth == 0:
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if escape {
		return nil, fmt.Errorf("unterminated escape sequence in %q", input)
	}

	parts = append(parts, strings.TrimSpace(current.String()))
	return parts, nil
}

func classifyArgument(raw string) argument {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "=") {
		expr := strings.TrimSpace(trimmed[1:])
		return argument{
			Raw:             expr,
			Normalized:      expr,
			Kind:            argumentExpression,
			ForceExpression: true,
		}
	}

	if trimmed == "" {
		return argument{
			Raw:        "",
			Normalized: "",
			Kind:       argumentString,
			AutoQuote:  true,
		}
	}

	if unquoted, err := strconv.Unquote(trimmed); err == nil {
		return argument{
			Raw:        unquoted,
			Normalized: unquoted,
			Kind:       argumentString,
		}
	}

	lower := strings.ToLower(trimmed)
	if lower == "true" || lower == "false" {
		return argument{
			Raw:        lower,
			Normalized: lower,
			Kind:       argumentBool,
		}
	}

	if _, err := strconv.ParseInt(trimmed, 0, 64); err == nil {
		return argument{
			Raw:        trimmed,
			Normalized: trimmed,
			Kind:       argumentInt,
		}
	}

	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return argument{
			Raw:        trimmed,
			Normalized: trimmed,
			Kind:       argumentFloat,
		}
	}

	if gotoken.IsIdentifier(trimmed) {
		return argument{
			Raw:        trimmed,
			Normalized: trimmed,
			Kind:       argumentString,
			AutoQuote:  true,
		}
	}

	return argument{
		Raw:        trimmed,
		Normalized: trimmed,
		Kind:       argumentExpression,
	}
}

func argDisplayForExternal(arg argument) string {
	if arg.ForceExpression || arg.Kind == argumentExpression {
		return arg.Raw
	}

	if arg.Kind == argumentString {
		return strconv.Quote(arg.Normalized)
	}

	return arg.Raw
}

func formatArgumentForType(arg argument, expected string) (string, error) {
	if arg.ForceExpression {
		return arg.Raw, nil
	}

	switch expected {
	case "string":
		return strconv.Quote(arg.Normalized), nil
	case "bool":
		return arg.Raw, nil
	case "int", "uint", "uintptr":
		return arg.Raw, nil
	case "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"byte", "rune":
		return fmt.Sprintf("%s(%s)", expected, arg.Raw), nil
	case "float32", "float64":
		return fmt.Sprintf("%s(%s)", expected, arg.Raw), nil
	default:
		return arg.Raw, nil
	}
}

func buildImportSpec(alias, path string) string {
	if path == "" {
		return ""
	}
	if alias == "" || alias == "_" || alias == "." {
		if alias == "_" || alias == "." {
			return fmt.Sprintf("%s %q", alias, path)
		}
		return fmt.Sprintf("%q", path)
	}
	if filepath.Base(path) == alias {
		return fmt.Sprintf("%q", path)
	}
	return fmt.Sprintf("%s %q", alias, path)
}

func (fe *FunctionExecutor) cacheKey(target callTarget, args []argument) (string, error) {
	type argKey struct {
		Kind            argumentKind `json:"k"`
		ForceExpression bool         `json:"f"`
		AutoQuote       bool         `json:"a"`
		Value           string       `json:"v"`
	}

	payload := struct {
		Call string   `json:"c"`
		Args []argKey `json:"a"`
	}{
		Call: target.callExpr,
		Args: make([]argKey, len(args)),
	}

	for i, arg := range args {
		value := arg.Normalized
		if arg.Kind == argumentExpression || arg.ForceExpression {
			value = arg.Raw
		}
		payload.Args[i] = argKey{
			Kind:            arg.Kind,
			ForceExpression: arg.ForceExpression,
			AutoQuote:       arg.AutoQuote,
			Value:           value,
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (fe *FunctionExecutor) cacheKeyWithDir(target callTarget, args []argument, sourceDir string) (string, error) {
	baseKey, err := fe.cacheKey(target, args)
	if err != nil {
		return "", err
	}
	// Include sourceDir in key to handle shadowing properly
	return fmt.Sprintf("%s|%s", sourceDir, baseKey), nil
}

func sanitizeGoEnv(env []string) []string {
	clean := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "GOFLAGS=") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func (fe *FunctionExecutor) ensureStdImportMap() {
	if fe.stdImportMap != nil {
		return
	}

	fe.stdImportMap = make(map[string]string)

	cmd := exec.Command("go", "list", "std")
	cmd.Env = sanitizeGoEnv(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			fe.stdListErr = fmt.Errorf("go list std: %w: %s", err, trimmed)
		} else {
			fe.stdListErr = fmt.Errorf("go list std: %w", err)
		}
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		base := filepath.Base(line)
		if existing, ok := fe.stdImportMap[base]; ok && existing != line {
			fe.stdImportMap[base] = ""
			continue
		}
		fe.stdImportMap[base] = line
	}
}
