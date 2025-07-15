//go:build ignore

package main

import "fmt"

func main() {
	//:getString
	msg := "Hello World"

	//:getInt
	num := 42

	//:ShadowStr:pippo
	secret := ""

	fmt.Printf("Message: %s\n", msg)
	fmt.Printf("Number: %d\n", num)
	fmt.Printf("Secret: %s\n", secret)
}
