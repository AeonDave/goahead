package types

import "fmt"

var (
	// Custom type as return
	//:getDefaultStatus
	status = 0

	// Status name from custom type value
	//:statusName:2
	statusStr = ""

	// Using struct internally
	//:getDefaultConfig
	serverAddr = ""

	// Nested struct calculation
	//:getRectArea
	area = 0
)

func main() {
	fmt.Printf("Status: %d\n", status)
	fmt.Printf("Status Name: %s\n", statusStr)
	fmt.Printf("Server: %s\n", serverAddr)
	fmt.Printf("Rectangle Area: %d\n", area)
}
