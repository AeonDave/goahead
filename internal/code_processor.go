package internal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CodeProcessor gestisce la sostituzione del codice
type CodeProcessor struct {
	ctx      *ProcessorContext
	executor *FunctionExecutor
}

// NewCodeProcessor crea un nuovo processore di codice
func NewCodeProcessor(ctx *ProcessorContext, executor *FunctionExecutor) *CodeProcessor {
	return &CodeProcessor{
		ctx:      ctx,
		executor: executor,
	}
}

// ProcessFile elabora un singolo file Go per le sostituzioni
func (cp *CodeProcessor) ProcessFile(filePath string, verbose bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	lines, modified, err := cp.processLines(file, filePath, verbose)
	if err != nil {
		return err
	}

	if modified {
		return cp.writeFile(filePath, lines)
	}
	return nil
}

// processLines elabora tutte le righe di un file
func (cp *CodeProcessor) processLines(file *os.File, filePath string, verbose bool) ([]string, bool, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	modified := false

	commentPattern := regexp.MustCompile(CommentPattern)

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
				newLine, wasModified := cp.processCodeLine(nextLine, funcName, argsStr, filePath, verbose)
				lines = append(lines, newLine)
				if wasModified {
					modified = true
				}
			}
		} else {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	return lines, modified, nil
}

// processCodeLine elabora una singola riga di codice per le sostituzioni
func (cp *CodeProcessor) processCodeLine(line, funcName, argsStr, filePath string, verbose bool) (string, bool) {
	// Execute the function to get the result value
	result, err := cp.executor.ExecuteFunction(funcName, argsStr)
	if err != nil {
		// Log to stderr, but don't modify the file content
		fmt.Fprintf(os.Stderr, "Warning: Could not execute function '%s' in %s: %v\n", funcName, filePath, err)
		return line, false // Return original line, no modification
	}

	userFunc, ok := cp.ctx.Functions[funcName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Warning: Function definition for '%s' not found in context while processing %s.\n", funcName, filePath)
		return line, false
	}

	// Analyze the original line to determine if it's a variable assignment
	// Detect the variable assignment pattern: "var x type = func()" or "x := func()" or "x = func()"
	trimmedLine := strings.TrimSpace(line)
	leadingWhitespace := ""
	if len(line) > len(trimmedLine) {
		leadingWhitespace = line[:len(line)-len(trimmedLine)]
	}

	// Check if line is a variable assignment - improved regex to handle more patterns
	isAssignment := regexp.MustCompile(`^\s*(var\s+\w+(\s+[\w\.\[\]]+)?\s*=|[\w\.,\s]+\s*:=|[\w\.]+\s*=)\s*`).MatchString(line)

	// Format the result according to its type for Go source code
	var formattedResult string
	switch userFunc.OutputType {
	case "string":
		formattedResult = escapeString(result)
	case "int", "int8", "int16", "int32", "int64":
		// Ensure numeric types have the correct formatting for Go
		// For int types, remove any decimal parts if present
		if strings.Contains(result, ".") {
			parts := strings.Split(result, ".")
			formattedResult = parts[0]
		} else {
			formattedResult = result
		}
	case "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		// For unsigned int types, ensure positive values only
		if strings.HasPrefix(result, "-") {
			fmt.Fprintf(os.Stderr, "Warning: Negative value %s used for unsigned type %s, using absolute value\n",
				result, userFunc.OutputType)
			formattedResult = strings.TrimPrefix(result, "-")
		} else if strings.Contains(result, ".") {
			parts := strings.Split(result, ".")
			formattedResult = parts[0]
		} else {
			formattedResult = result
		}
	case "float32", "float64":
		// Ensure it has a decimal point for float literals
		if !strings.Contains(result, ".") {
			formattedResult = result + ".0"
		} else {
			formattedResult = result
		}
	case "bool":
		if result == "1" || strings.ToLower(result) == "true" {
			formattedResult = "true"
		} else {
			formattedResult = "false"
		}
	default:
		fmt.Fprintf(os.Stderr, "Warning: Unknown output type '%s' for function '%s' in %s. Treating as string.\n", userFunc.OutputType, funcName, filePath)
		formattedResult = escapeString(result)
	}

	// For variable assignments, we need to replace only the function call, not the entire line
	var newLine string
	if isAssignment {
		// Analyze the line to extract the variable assignment part - improved regex to handle := correctly
		varAssignPattern := regexp.MustCompile(`^(\s*(?:var\s+\w+(?:\s+[\w\.\[\]]+)?\s*=|[\w\.,\s]+\s*:=|[\w\.]+\s*=)\s*)(.*)$`)
		matches := varAssignPattern.FindStringSubmatch(line)

		if len(matches) >= 3 {
			varAssignPart := matches[1]
			expressionPart := matches[2]
			// Try to replace only the first placeholder of the correct type in the expression
			replacedExpression := cp.replaceFirstPlaceholder(expressionPart, formattedResult, userFunc.OutputType)

			if replacedExpression != expressionPart {
				// Successfully replaced a placeholder
				newLine = varAssignPart + replacedExpression
			} else {
				// No placeholder found, fallback to replacing the entire expression
				newLine = varAssignPart + formattedResult
			}
		} else {
			// If the regex didn't match, fallback to the old behavior
			if argsStr != "" {
				// Try exact match with arguments first
				re := regexp.MustCompile(fmt.Sprintf(`%s\(\s*%s\s*\)`, regexp.QuoteMeta(funcName), regexp.QuoteMeta(argsStr)))
				newLine = re.ReplaceAllString(line, formattedResult)
			}

			// If no replacement occurred or no args provided, try more flexible patterns
			if newLine == line {
				// Match function name with any arguments
				re := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\)`, regexp.QuoteMeta(funcName)))
				newLine = re.ReplaceAllString(line, formattedResult)
			}

			// If still no replacement, log a warning and return the original line
			if newLine == line {
				fmt.Fprintf(os.Stderr, "Warning: Could not replace function call for '%s' in line: %s\n",
					funcName, strings.TrimSpace(line))
				return line, false
			}
		}
	} else {
		// For regular files or non-assignment lines, just replace with the formatted result
		newLine = leadingWhitespace + formattedResult
	}

	replaced := newLine != line

	if replaced && verbose {
		fmt.Fprintf(os.Stderr, "Processed in %s: %s(%s) -> %s\n", filePath, funcName, argsStr, result)
		fmt.Fprintf(os.Stderr, "  Original: '%s'\n  New: '%s'\n", line, newLine)
	}

	return newLine, replaced
}

// writeFile scrive le righe elaborate in un file
func (cp *CodeProcessor) writeFile(filePath string, lines []string) error {
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

// escapeString gestisce l'escape delle stringhe
func escapeString(s string) string {
	if strings.Contains(s, "\\") {
		return "`" + s + "`"
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// replaceFirstPlaceholder sostituisce il primo placeholder del tipo corretto in un'espressione
func (cp *CodeProcessor) replaceFirstPlaceholder(expression, replacement, outputType string) string {
	switch outputType {
	case "string":
		// Cerca e sostituisce la prima stringa vuota ""
		re := regexp.MustCompile(`""`)
		if re.MatchString(expression) {
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				// Sostituisce solo la prima occorrenza, poi ritorna l'originale
				return replacement
			})
		}

	case "int", "int8", "int16", "int32", "int64":
		// Cerca e sostituisce il primo 0 intero
		re := regexp.MustCompile(`\b0\b`)
		if re.MatchString(expression) {
			replaced := false
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				if !replaced {
					replaced = true
					return replacement
				}
				return match
			})
		}

	case "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		// Cerca e sostituisce il primo 0 intero senza segno
		re := regexp.MustCompile(`\b0\b`)
		if re.MatchString(expression) {
			replaced := false
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				if !replaced {
					replaced = true
					return replacement
				}
				return match
			})
		}

	case "float32", "float64":
		// Cerca e sostituisce il primo 0.0 float
		re := regexp.MustCompile(`\b0\.0\b`)
		if re.MatchString(expression) {
			replaced := false
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				if !replaced {
					replaced = true
					return replacement
				}
				return match
			})
		}
		// Se non trova 0.0, prova con 0
		re2 := regexp.MustCompile(`\b0\b`)
		if re2.MatchString(expression) {
			replaced := false
			return re2.ReplaceAllStringFunc(expression, func(match string) string {
				if !replaced {
					replaced = true
					return replacement
				}
				return match
			})
		}

	case "bool":
		// Cerca e sostituisce il primo false
		re := regexp.MustCompile(`\bfalse\b`)
		if re.MatchString(expression) {
			replaced := false
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				if !replaced {
					replaced = true
					return replacement
				}
				return match
			})
		}
	}

	// Se non trova nessun placeholder specifico, ritorna l'espressione originale
	return expression
}
