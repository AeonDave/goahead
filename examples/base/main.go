package base

import "fmt"

func main() {
	//:getString
	msg := "Hello World"

	//:getInt
	num := 42

	//:ShadowStr:pippo
	secret := "4717317a03"

	fmt.Printf("Message: %s\n", msg)
	fmt.Printf("Number: %d\n", num)
	fmt.Printf("Secret: %s\n", secret)
}
