//go:build linux || darwin || windows

// This example demonstrates GoAhead working alongside Go directives.
// All //go: directives are preserved while //: placeholders are replaced.

package directives

import (
	_ "embed"
	"fmt"
)

//go:generate echo "This generate directive is preserved"

// Build-time configuration via GoAhead
var (
	//:getBuildMode
	buildMode = ""

	//:getOptimizationLevel
	optLevel = 0

	//:getFeatureFlag
	newUIEnabled = false
)

//go:embed helpers.go
var helperSource string

//go:noinline
func criticalPath() string {
	return buildMode
}

func main() {
	fmt.Printf("Build Mode: %s\n", buildMode)
	fmt.Printf("Optimization: O%d\n", optLevel)
	fmt.Printf("Critical: %s\n", criticalPath())
	fmt.Printf("Helper source length: %d bytes\n", len(helperSource))
}
