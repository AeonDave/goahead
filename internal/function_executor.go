package internal

import (
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

	userFunctionsSource string
	userImportSet       map[string]struct{}
	prepared            bool

	stdImportMap map[string]string
}

func NewFunctionExecutor(ctx *ProcessorContext) *FunctionExecutor {
	return &FunctionExecutor{
		ctx:   ctx,
		cache: make(map[string]string),
	}
}

func (fe *FunctionExecutor) Prepare() error {
	return fe.ensureUserFunctionsPrepared()
}

func (fe *FunctionExecutor) ExecuteFunction(funcName string, argsStr string) (string, error) {
	args, err := fe.parseArguments(argsStr)
	if err != nil {
		return "", err
	}

	target, err := fe.determineTarget(funcName)
	if err != nil {
		return "", err
	}

	key, err := fe.cacheKey(target, args)
	if err != nil {
		return "", err
	}
	if cached, ok := fe.cache[key]; ok {
		return cached, nil
	}

	formattedArgs, err := fe.formatArguments(target, args)
	if err != nil {
		return "", err
	}

	callExpr := target.callExpr
	if len(formattedArgs) > 0 {
		callExpr = fmt.Sprintf("%s(%s)", target.callExpr, strings.Join(formattedArgs, ", "))
	} else {
		callExpr = fmt.Sprintf("%s()", target.callExpr)
	}

	program, err := fe.buildProgram(target, callExpr)
	if err != nil {
		return "", err
	}

	result, err := fe.executeProgram(program)
	if err != nil {
		if target.kind == invocationExternal && !target.importResolved {
			return "", fmt.Errorf("%w. Add //go:ahead import %s=%s in a function file to declare the package alias", err, target.packageAlias, target.packagePath)
		}
		return "", err
	}

	fe.cache[key] = result
	return result, nil
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

func (fe *FunctionExecutor) determineTarget(funcName string) (callTarget, error) {
	if fn, ok := fe.ctx.Functions[funcName]; ok {
		return callTarget{
			kind:     invocationUser,
			userFunc: fn,
			callExpr: funcName,
		}, nil
	}

	alias, remainder, ok := strings.Cut(funcName, ".")
	if !ok || alias == "" || remainder == "" {
		return callTarget{}, fmt.Errorf("function %s not found; define it in a //go:ahead functions file or reference it as package.Func", funcName)
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
	if path, ok := fe.ctx.ImportOverrides[alias]; ok {
		return path, true
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
	if len(expected) != len(args) {
		return nil, fmt.Errorf("function %s expects %d arguments, got %d", fn.Name, len(expected), len(args))
	}

	formatted := make([]string, len(args))
	for i, typ := range expected {
		value, err := formatArgumentForType(args[i], typ)
		if err != nil {
			return nil, fmt.Errorf("argument %d for %s: %w", i, fn.Name, err)
		}
		formatted[i] = value
	}
	return formatted, nil
}

func (fe *FunctionExecutor) buildProgram(target callTarget, callExpr string) (string, error) {
	if err := fe.ensureUserFunctionsPrepared(); err != nil {
		return "", err
	}

	importSet := make(map[string]struct{})
	for spec := range fe.userImportSet {
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
		UserCode: strings.TrimSpace(fe.userFunctionsSource),
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

func (fe *FunctionExecutor) executeProgram(program string) (string, error) {
	tempFile := filepath.Join(fe.ctx.TempDir, "goahead_eval.go")
	if err := os.WriteFile(tempFile, []byte(program), 0o600); err != nil {
		return "", fmt.Errorf("failed to write temp file: %v", err)
	}

	cmd := exec.Command("go", "run", tempFile)
	cmd.Env = sanitizeGoEnv(os.Environ())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute temp program: %v\nOutput:\n%s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func (fe *FunctionExecutor) ensureUserFunctionsPrepared() error {
	if fe.prepared {
		return nil
	}

	var pieces []string
	importSet := make(map[string]struct{})

	for _, file := range fe.ctx.FuncFiles {
		code, imports := fe.processFunctionFile(file)
		if code != "" {
			pieces = append(pieces, code)
		}
		for spec := range imports {
			importSet[spec] = struct{}{}
		}
	}

	fe.userFunctionsSource = strings.Join(pieces, "\n\n")
	fe.userImportSet = importSet
	fe.prepared = true

	return nil
}

func (fe *FunctionExecutor) processFunctionFile(path string) (string, map[string]struct{}) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", make(map[string]struct{})
	}

	lines := strings.Split(string(content), "\n")
	var builder strings.Builder
	imports := make(map[string]struct{})

	inFunction := false
	inImportBlock := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//go:ahead ") {
			fe.handleDirective(trimmed)
			continue
		}
		if strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "package ") {
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
		if strings.HasPrefix(trimmed, "//") && !inFunction {
			continue
		}
		if strings.HasPrefix(trimmed, "func ") && !inFunction {
			inFunction = true
			braceCount = 0
		}
		if inFunction {
			builder.WriteString(line)
			builder.WriteByte('\n')
			braceCount += strings.Count(line, "{")
			braceCount -= strings.Count(line, "}")
			if braceCount == 0 && strings.Contains(line, "}") {
				inFunction = false
				builder.WriteByte('\n')
			}
		}
	}

	return strings.TrimSpace(builder.String()), imports
}

func (fe *FunctionExecutor) handleDirective(line string) {
	directive := strings.TrimSpace(strings.TrimPrefix(line, "//go:ahead"))
	if !strings.HasPrefix(directive, "import") {
		return
	}
	spec := strings.TrimSpace(strings.TrimPrefix(directive, "import"))
	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		return
	}
	alias := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])
	path = strings.Trim(path, "`\"'")
	if alias != "" && path != "" {
		fe.ctx.ImportOverrides[alias] = path
	}
}

func splitArguments(input string) ([]string, error) {
	var (
		parts   []string
		current strings.Builder
		inQuote bool
		quote   rune
		escape  bool
	)

	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\':
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
		case r == ':':
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
