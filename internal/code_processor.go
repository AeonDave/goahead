package internal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type CodeProcessor struct {
	ctx      *ProcessorContext
	executor *FunctionExecutor
}

func NewCodeProcessor(ctx *ProcessorContext, executor *FunctionExecutor) *CodeProcessor {
	return &CodeProcessor{
		ctx:      ctx,
		executor: executor,
	}
}

func (cp *CodeProcessor) ProcessFile(filePath string, verbose bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
		}
	}(file)

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

func (cp *CodeProcessor) processCodeLine(line, funcName, argsStr, filePath string, verbose bool) (string, bool) {
	result, err := cp.executor.ExecuteFunction(funcName, argsStr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not execute function '%s' in %s: %v\n", funcName, filePath, err)
		return line, false
	}

	userFunc, ok := cp.ctx.Functions[funcName]
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Function definition for '%s' not found in context while processing %s.\n", funcName, filePath)
		return line, false
	}

	trimmedLine := strings.TrimSpace(line)
	leadingWhitespace := ""
	if len(line) > len(trimmedLine) {
		leadingWhitespace = line[:len(line)-len(trimmedLine)]
	}
	isAssignment := regexp.MustCompile(`^\s*(var\s+\w+(\s+[\w.\[\]]+)?\s*=|[\w.,\s]+\s*:=|[\w.]+\s*=)\s*`).MatchString(line)

	var formattedResult string
	switch userFunc.OutputType {
	case "string":
		formattedResult = escapeString(result)
	case "int", "int8", "int16", "int32", "int64":
		if strings.Contains(result, ".") {
			parts := strings.Split(result, ".")
			formattedResult = parts[0]
		} else {
			formattedResult = result
		}
	case "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		if strings.HasPrefix(result, "-") {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: Negative value %s used for unsigned type %s, using absolute value\n",
				result, userFunc.OutputType)
			formattedResult = strings.TrimPrefix(result, "-")
		} else if strings.Contains(result, ".") {
			parts := strings.Split(result, ".")
			formattedResult = parts[0]
		} else {
			formattedResult = result
		}
	case "float32", "float64":
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
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Unknown output type '%s' for function '%s' in %s. Treating as string.\n", userFunc.OutputType, funcName, filePath)
		formattedResult = escapeString(result)
	}

	var newLine string
	if isAssignment {
		varAssignPattern := regexp.MustCompile(`^(\s*(?:var\s+\w+(?:\s+[\w.\[\]]+)?\s*=|[\w.,\s]+\s*:=|[\w.]+\s*=)\s*)(.*)$`)
		matches := varAssignPattern.FindStringSubmatch(line)

		if len(matches) >= 3 {
			varAssignPart := matches[1]
			expressionPart := matches[2]
			replacedExpression := cp.replaceFirstPlaceholder(expressionPart, formattedResult, userFunc.OutputType)
			if replacedExpression != expressionPart {
				newLine = varAssignPart + replacedExpression
			} else {
				newLine = varAssignPart + formattedResult
			}
		} else {
			if argsStr != "" {
				re := regexp.MustCompile(fmt.Sprintf(`%s\(\s*%s\s*\)`, regexp.QuoteMeta(funcName), regexp.QuoteMeta(argsStr)))
				newLine = re.ReplaceAllString(line, formattedResult)
			}
			if newLine == line {
				re := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\)`, regexp.QuoteMeta(funcName)))
				newLine = re.ReplaceAllString(line, formattedResult)
			}
			if newLine == line {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not replace function call for '%s' in line: %s\n",
					funcName, strings.TrimSpace(line))
				return line, false
			}
		}
	} else {
		newLine = leadingWhitespace + formattedResult
	}
	replaced := newLine != line
	if replaced {
		_, _ = fmt.Fprintf(os.Stderr, "[goahead] Replaced in %s: %s(%s) -> %s\n", filePath, funcName, argsStr, result)
		if verbose {
			_, _ = fmt.Fprintf(os.Stderr, "  Original: '%s'\n  New: '%s'\n", strings.TrimSpace(line), strings.TrimSpace(newLine))
		}
	}
	return newLine, replaced
}

func (cp *CodeProcessor) writeFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filePath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
		}
	}(file)

	writer := bufio.NewWriter(file)
	defer func(writer *bufio.Writer) {
		err := writer.Flush()
		if err != nil {
		}
	}(writer)

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

func (cp *CodeProcessor) replaceFirstPlaceholder(expression, replacement, outputType string) string {
	switch outputType {
	case "string":
		re := regexp.MustCompile(`""`)
		if re.MatchString(expression) {
			return re.ReplaceAllStringFunc(expression, func(match string) string {
				// Sostituisce solo la prima occorrenza, poi ritorna l'originale
				return replacement
			})
		}

	case "int", "int8", "int16", "int32", "int64":
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
	return expression
}
