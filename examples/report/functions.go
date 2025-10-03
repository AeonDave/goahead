//go:build ignore

//go:ahead functions

package report

import (
	"fmt"
	"strings"
	"time"
)

func incidentSummary(team string, resolved, total int) string {
	return fmt.Sprintf("%s team resolved %d of %d incidents", strings.ToUpper(team), resolved, total)
}

func resolutionRate(resolved, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", float64(resolved)/float64(total)*100)
}

func generatedAt(layout string) string {
	return time.Now().Format(layout)
}

func slugify(input string) string {
	sanitized := strings.ToLower(strings.TrimSpace(input))
	return strings.ReplaceAll(sanitized, " ", "-")
}
