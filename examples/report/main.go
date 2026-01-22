package report

import "fmt"

func main() {
	//:incidentSummary:"Platform":29:37
	summary := "PLATFORM team resolved 29 of 37 incidents"

	//:resolutionRate:29:37
	rate := "78.4%"

	//:generatedAt:"2006-01-02 15:04"
	generatedAt := "2026-01-22 17:49"

	//:slugify:"Weekly Platform Update"
	slug := "weekly-platform-update"

	fmt.Println(summary)
	fmt.Printf("Resolution rate: %s\n", rate)
	fmt.Printf("Generated at: %s\n", generatedAt)
	fmt.Printf("Slug: %s\n", slug)
}
