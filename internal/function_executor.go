package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// FunctionExecutor gestisce l'esecuzione delle funzioni utente
type FunctionExecutor struct {
	ctx *ProcessorContext
}

// NewFunctionExecutor crea un nuovo esecutore di funzioni
func NewFunctionExecutor(ctx *ProcessorContext) *FunctionExecutor {
	return &FunctionExecutor{ctx: ctx}
}

// ExecuteFunction esegue una funzione utente con i parametri specificati
func (fe *FunctionExecutor) ExecuteFunction(funcName string, argsStr string) (string, error) {
	userFunc, exists := fe.ctx.Functions[funcName]
	if !exists {
		return "", fmt.Errorf("function %s not found", funcName)
	}

	args, err := fe.parseArguments(argsStr)
	if err != nil {
		return "", err
	}

	if len(args) != len(userFunc.InputTypes) {
		return "", fmt.Errorf("function %s expects %d arguments, got %d",
			funcName, len(userFunc.InputTypes), len(args))
	}

	evaluatedArgs, err := fe.evaluateArguments(args)
	if err != nil {
		return "", err
	}

	return fe.callUserFunction(funcName, evaluatedArgs)
}

// parseArguments analizza la stringa degli argomenti
func (fe *FunctionExecutor) parseArguments(argsStr string) ([]string, error) {
	if argsStr == "" {
		return []string{}, nil
	}

	args := strings.Split(argsStr, ":")
	for i, arg := range args {
		args[i] = strings.TrimSpace(arg)
	}

	return args, nil
}

// evaluateArguments valuta tutti gli argomenti
func (fe *FunctionExecutor) evaluateArguments(args []string) ([]string, error) {
	evaluatedArgs := make([]string, len(args))

	for i, arg := range args {
		val, err := fe.evaluateExpression(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate argument %d (%s): %v", i, arg, err)
		}
		evaluatedArgs[i] = val
	}

	return evaluatedArgs, nil
}

// evaluateExpression valuta una singola espressione
func (fe *FunctionExecutor) evaluateExpression(expression string) (string, error) {
	// Stringa tra virgolette
	if strings.HasPrefix(expression, `"`) && strings.HasSuffix(expression, `"`) {
		return strings.Trim(expression, `"`), nil
	}

	// Numero intero
	if _, err := strconv.Atoi(expression); err == nil {
		return expression, nil
	}

	// Valore booleano
	if expression == "true" || expression == "false" {
		return expression, nil
	}

	return expression, nil
}

// callUserFunction chiama effettivamente la funzione utente
func (fe *FunctionExecutor) callUserFunction(funcName string, args []string) (string, error) {
	userFunc := fe.ctx.Functions[funcName]

	// Prepara il codice per l'esecuzione
	code, err := fe.prepareExecutionCode(funcName, args, userFunc)
	if err != nil {
		return "", err
	}

	// Esegue il codice
	return fe.executeCode(code)
}

// prepareExecutionCode prepara il codice per l'esecuzione
func (fe *FunctionExecutor) prepareExecutionCode(funcName string, args []string, userFunc *UserFunction) (string, error) {
	allFuncLines, importSet := fe.extractFunctionsAndImports()
	formattedArgs := fe.formatArguments(args, userFunc.InputTypes)
	additionalImports := fe.formatImports(importSet)

	tmpl := template.Must(template.New("program").Parse(ExecutionTemplate))

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

	tempFile := filepath.Join(fe.ctx.TempDir, "temp_program.go")
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return tempFile, nil
}

// extractFunctionsAndImports estrae le funzioni e gli import dai file
func (fe *FunctionExecutor) extractFunctionsAndImports() ([]string, map[string]bool) {
	var allFuncLines []string
	importSet := make(map[string]bool)

	for _, funcFile := range fe.ctx.FuncFiles {
		funcLines, imports := fe.processFunctionFile(funcFile)
		allFuncLines = append(allFuncLines, funcLines...)

		for imp := range imports {
			importSet[imp] = true
		}
	}

	return allFuncLines, importSet
}

// processFunctionFile elabora un singolo file di funzioni
func (fe *FunctionExecutor) processFunctionFile(funcFile string) ([]string, map[string]bool) {
	userFuncContent, err := os.ReadFile(funcFile)
	if err != nil {
		return nil, nil
	}

	userFuncStr := string(userFuncContent)
	lines := strings.Split(userFuncStr, "\n")

	var funcLines []string
	importSet := make(map[string]bool)

	inFunction := false
	braceCount := 0
	skipNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Salta le direttive del compilatore e package
		if fe.shouldSkipLine(trimmed) {
			continue
		}

		// Gestisce gli import
		if fe.handleImportLine(trimmed, &skipNext, importSet) {
			continue
		}

		if skipNext {
			if fe.handleImportBlock(trimmed, &skipNext, importSet) {
				continue
			}
		}

		// Gestisce le funzioni
		if fe.handleFunctionLine(line, trimmed, &inFunction, &braceCount, &funcLines) {
			continue
		}
	}

	return funcLines, importSet
}

// shouldSkipLine verifica se una riga deve essere saltata
func (fe *FunctionExecutor) shouldSkipLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//go:build") ||
		strings.HasPrefix(trimmed, "//go:ahead") ||
		strings.HasPrefix(trimmed, "package ")
}

// handleImportLine gestisce le righe di import
func (fe *FunctionExecutor) handleImportLine(trimmed string, skipNext *bool, importSet map[string]bool) bool {
	if strings.HasPrefix(trimmed, "import") {
		if strings.Contains(trimmed, "(") {
			*skipNext = true
		} else {
			if parts := strings.Fields(trimmed); len(parts) >= 2 {
				importPath := strings.Trim(parts[1], `"`)
				importSet[importPath] = true
			}
		}
		return true
	}
	return false
}

// handleImportBlock gestisce i blocchi di import
func (fe *FunctionExecutor) handleImportBlock(trimmed string, skipNext *bool, importSet map[string]bool) bool {
	if strings.Contains(trimmed, ")") {
		*skipNext = false
	} else if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
		importPath := strings.Trim(trimmed, `" 	`)
		if importPath != "" {
			importSet[importPath] = true
		}
	}
	return true
}

// handleFunctionLine gestisce le righe delle funzioni
func (fe *FunctionExecutor) handleFunctionLine(line, trimmed string, inFunction *bool, braceCount *int, funcLines *[]string) bool {
	if strings.HasPrefix(trimmed, "//") && !*inFunction {
		return true
	}

	if strings.Contains(trimmed, "func ") && !*inFunction {
		*inFunction = true
		*braceCount = 0
	}

	if *inFunction {
		*funcLines = append(*funcLines, line)
		*braceCount += strings.Count(line, "{")
		*braceCount -= strings.Count(line, "}")
		if *braceCount == 0 && strings.Contains(line, "}") {
			*inFunction = false
		}
	}

	return *inFunction
}

// formatArguments formatta gli argomenti in base ai loro tipi
func (fe *FunctionExecutor) formatArguments(args []string, inputTypes []string) []string {
	formattedArgs := make([]string, len(args))

	for i, arg := range args {
		inputType := inputTypes[i]
		switch inputType {
		case "string":
			formattedArgs[i] = `"` + arg + `"`
		case "int":
			formattedArgs[i] = arg
		case "int8":
			formattedArgs[i] = "int8(" + arg + ")"
		case "int16":
			formattedArgs[i] = "int16(" + arg + ")"
		case "int32":
			formattedArgs[i] = "int32(" + arg + ")"
		case "int64":
			formattedArgs[i] = "int64(" + arg + ")"
		case "uint":
			formattedArgs[i] = "uint(" + arg + ")"
		case "uint8":
			formattedArgs[i] = "uint8(" + arg + ")"
		case "uint16":
			formattedArgs[i] = "uint16(" + arg + ")"
		case "uint32":
			formattedArgs[i] = "uint32(" + arg + ")"
		case "uint64":
			formattedArgs[i] = "uint64(" + arg + ")"
		case "float32":
			formattedArgs[i] = "float32(" + arg + ")"
		case "float64":
			formattedArgs[i] = "float64(" + arg + ")"
		case "bool", "byte", "rune":
			formattedArgs[i] = arg
		default:
			formattedArgs[i] = `"` + arg + `"`
		}
	}

	return formattedArgs
}

// formatImports formatta gli import aggiuntivi
func (fe *FunctionExecutor) formatImports(importSet map[string]bool) []string {
	var additionalImports []string

	for importPath := range importSet {
		if importPath != "fmt" {
			additionalImports = append(additionalImports, "\t\""+importPath+"\"")
		}
	}

	return additionalImports
}

// executeCode esegue il codice generato
func (fe *FunctionExecutor) executeCode(tempFile string) (string, error) {
	cmd := exec.Command("go", "run", tempFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute temp program: %v\nOutput:\n%s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}
