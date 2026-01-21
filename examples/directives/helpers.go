//go:build ignore

//go:ahead functions

package directives

// This file demonstrates that GoAhead helpers work alongside
// other Go directives without interfering with them.

func getBuildMode() string {
	return "release"
}

func getOptimizationLevel() int {
	return 3
}

func getFeatureFlag() bool {
	return true
}

func getEmbedPattern() string {
	return "templates/*.html"
}
