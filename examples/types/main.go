package types

import "fmt"

var (
	// Custom type as return
	//:getDefaultStatus
	status = 1

	// Status name from custom type value
	//:statusName:2
	statusStr = "Completed"

	// Using struct internally
	//:getDefaultConfig
	serverAddr = "localhost:8080"

	// Nested struct calculation
	//:getRectArea
	area = 50
)

func main() {
	fmt.Printf("Status: %d\n", status)
	fmt.Printf("Status Name: %s\n", statusStr)
	fmt.Printf("Server: %s\n", serverAddr)
	fmt.Printf("Rectangle Area: %d\n", area)
}
