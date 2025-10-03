package internal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type CodeProcessor struct {
	ctx      *ProcessorContext
	executor *FunctionExecutor
}

var (
	assignmentPattern      = regexp.MustCompile(`^\s*(var\s+\w+(\s+[\w.\[\]]+)?\s*=|[\w.,\s]+\s*:=|[\w.]+\s*=)\s*`)
	assignmentSplitPattern = regexp.MustCompile(`^(\s*(?:var\s+\w+(?:\s+[\w.\[\]]+)?\s*=|[\w.,\s]+\s*:=|[\w.]+\s*=)\s*)(.*)$`)
)

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

	typeHint := "other"
	if userFunc, ok := cp.ctx.Functions[funcName]; ok {
		typeHint = mapOutputType(userFunc.OutputType)
	}

	inferred := inferResultKind(result)
	if typeHint == "other" {
		typeHint = inferred
	}

	formattedResult := formatResultForReplacement(result, typeHint)

	trimmedLine := strings.TrimSpace(line)
	leadingWhitespace := ""
	if len(line) > len(trimmedLine) {
		leadingWhitespace = line[:len(line)-len(trimmedLine)]
	}

	isAssignment := assignmentPattern.MatchString(line)

	var newLine string
	if isAssignment {
		matches := assignmentSplitPattern.FindStringSubmatch(line)

		if len(matches) >= 3 {
			varAssignPart := matches[1]
			expressionPart := matches[2]
			replacedExpression := cp.replaceFirstPlaceholder(expressionPart, formattedResult, typeHint)
			if replacedExpression != expressionPart {
				newLine = varAssignPart + replacedExpression
			} else {
				newLine = varAssignPart + formattedResult
			}
		} else {
			replacement := cp.replaceFunctionCall(line, funcName, argsStr, formattedResult)
			if replacement == line {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not replace function call for '%s' in line: %s\n", funcName, strings.TrimSpace(line))
				return line, false
			}
			newLine = replacement
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

func (cp *CodeProcessor) replaceFirstPlaceholder(expression, replacement, typeHint string) string {
	switch typeHint {
	case "string":
		re := regexp.MustCompile(`""`)
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

	case "int":
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

	case "uint":
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

	case "float":
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
		reZero := regexp.MustCompile(`\b0\b`)
		if reZero.MatchString(expression) {
			replaced := false
			return reZero.ReplaceAllStringFunc(expression, func(match string) string {
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

func (cp *CodeProcessor) replaceFunctionCall(line, funcName, argsStr, replacement string) string {
	updated := line
	if argsStr != "" {
		re := regexp.MustCompile(fmt.Sprintf(`%s\(\s*%s\s*\)`, regexp.QuoteMeta(funcName), regexp.QuoteMeta(argsStr)))
		updated = re.ReplaceAllString(updated, replacement)
	}
	if updated == line {
		re := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\)`, regexp.QuoteMeta(funcName)))
		updated = re.ReplaceAllString(updated, replacement)
	}
	return updated
}

func mapOutputType(outputType string) string {
	switch outputType {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "float32", "float64":
		return "float"
	case "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune":
		return "uint"
	case "int", "int8", "int16", "int32", "int64":
		return "int"
	default:
		return "other"
	}
}

func inferResultKind(result string) string {
	trimmed := strings.TrimSpace(result)
	lower := strings.ToLower(trimmed)
	if lower == "true" || lower == "false" {
		return "bool"
	}
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		return "string"
	}
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return "float"
	}
	return "other"
}

func formatResultForReplacement(result string, typeHint string) string {
	trimmed := strings.TrimSpace(result)
	switch typeHint {
	case "string":
		if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
			return trimmed
		}
		return escapeString(trimmed)
	case "bool":
		if strings.EqualFold(trimmed, "true") {
			return "true"
		}
		if strings.EqualFold(trimmed, "false") {
			return "false"
		}
		return "false"
	default:
		return trimmed
	}
}
