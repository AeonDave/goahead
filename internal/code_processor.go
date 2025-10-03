package internal

import (
	"bufio"
	"errors"
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
	stringLiteralPattern   = regexp.MustCompile(`""`)
	numericZeroPattern     = regexp.MustCompile(`\b0\b`)
	floatZeroPattern       = regexp.MustCompile(`\b0\.0\b`)
	boolFalsePattern       = regexp.MustCompile(`\bfalse\b`)
	errNoReplacement       = errors.New("no replacement performed")
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

	typeHint := cp.typeHintFor(funcName, result)
	formattedResult := formatResultForReplacement(result, typeHint)

	leadingWhitespace, _ := splitLeadingWhitespace(line)
	newLine, replaced, buildErr := cp.buildReplacementLine(line, leadingWhitespace, funcName, argsStr, formattedResult, typeHint)
	if buildErr != nil {
		if errors.Is(buildErr, errNoReplacement) {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: Could not replace function call for '%s' in line: %s\n", funcName, strings.TrimSpace(line))
		}
		return line, false
	}

	if replaced {
		_, _ = fmt.Fprintf(os.Stderr, "[goahead] Replaced in %s: %s(%s) -> %s\n", filePath, funcName, argsStr, result)
		if verbose {
			_, _ = fmt.Fprintf(os.Stderr, "  Original: '%s'\n  New: '%s'\n", strings.TrimSpace(line), strings.TrimSpace(newLine))
		}
	}

	return newLine, replaced
}

func splitLeadingWhitespace(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if len(line) == len(trimmed) {
		return "", line
	}
	return line[:len(line)-len(trimmed)], trimmed
}

func (cp *CodeProcessor) buildReplacementLine(originalLine, leadingWhitespace, funcName, argsStr, formattedResult, typeHint string) (string, bool, error) {
	if assignmentPattern.MatchString(originalLine) {
		return cp.replaceInAssignment(originalLine, funcName, argsStr, formattedResult, typeHint)
	}

	if replacedLine, ok := cp.replaceFunctionCall(originalLine, funcName, argsStr, formattedResult); ok {
		return replacedLine, true, nil
	}

	newLine := leadingWhitespace + formattedResult
	return newLine, newLine != originalLine, nil
}

func (cp *CodeProcessor) replaceInAssignment(originalLine, funcName, argsStr, formattedResult, typeHint string) (string, bool, error) {
	matches := assignmentSplitPattern.FindStringSubmatch(originalLine)
	if len(matches) < 3 {
		replacedLine, ok := cp.replaceFunctionCall(originalLine, funcName, argsStr, formattedResult)
		if !ok {
			return "", false, errNoReplacement
		}
		return replacedLine, true, nil
	}

	varAssignPart := matches[1]
	expressionPart := matches[2]

	replacedExpression, replaced := cp.replaceFirstPlaceholder(expressionPart, formattedResult, typeHint)
	if replaced {
		newLine := varAssignPart + replacedExpression
		return newLine, newLine != originalLine, nil
	}

	newLine := varAssignPart + formattedResult
	return newLine, newLine != originalLine, nil
}

func (cp *CodeProcessor) typeHintFor(funcName, result string) string {
	if userFunc, ok := cp.ctx.Functions[funcName]; ok {
		hint := mapOutputType(userFunc.OutputType)
		if hint != "other" {
			return hint
		}
	}
	return inferResultKind(result)
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

func (cp *CodeProcessor) replaceFirstPlaceholder(expression, replacement, typeHint string) (string, bool) {
	switch typeHint {
	case "string":
		return replaceFirstMatch(stringLiteralPattern, expression, replacement)
	case "int", "uint":
		return replaceFirstMatch(numericZeroPattern, expression, replacement)
	case "float":
		if updated, ok := replaceFirstMatch(floatZeroPattern, expression, replacement); ok {
			return updated, true
		}
		return replaceFirstMatch(numericZeroPattern, expression, replacement)
	case "bool":
		return replaceFirstMatch(boolFalsePattern, expression, replacement)
	default:
		return expression, false
	}
}

func replaceFirstMatch(re *regexp.Regexp, expression, replacement string) (string, bool) {
	replaced := false
	updated := re.ReplaceAllStringFunc(expression, func(match string) string {
		if replaced {
			return match
		}
		replaced = true
		return replacement
	})
	return updated, replaced
}

func (cp *CodeProcessor) replaceFunctionCall(line, funcName, argsStr, replacement string) (string, bool) {
	if argsStr != "" {
		re := regexp.MustCompile(fmt.Sprintf(`%s\(\s*%s\s*\)`, regexp.QuoteMeta(funcName), regexp.QuoteMeta(argsStr)))
		if re.MatchString(line) {
			return re.ReplaceAllString(line, replacement), true
		}
	}

	re := regexp.MustCompile(fmt.Sprintf(`%s\([^)]*\)`, regexp.QuoteMeta(funcName)))
	if re.MatchString(line) {
		return re.ReplaceAllString(line, replacement), true
	}
	return line, false
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
