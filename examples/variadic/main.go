package variadic

import "fmt"

var (
	// Variadic string function with separator
	//:joinAll:"-":"a":"b":"c":"d"
	dashed = ""

	// Variadic without separator
	//:concat:"Hello":" ":"World":"!"
	message = ""

	// Variadic numbers
	//:sum:1:2:3:4:5
	total = 0

	// Find maximum
	//:maxOf:42:17:99:8:73
	maximum = 0
)

func main() {
	fmt.Printf("Dashed: %s\n", dashed)
	fmt.Printf("Message: %s\n", message)
	fmt.Printf("Sum 1-5: %d\n", total)
	fmt.Printf("Max: %d\n", maximum)
}
