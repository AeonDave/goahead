package constants

import "fmt"

var (
	// Using helper that references package constants
	//:prefixed:"DATABASE_URL"
	envKey = ""

	// Formatted key-value
	//:formatted:"config":"production"
	configEntry = ""

	// Get version constant
	//:getVersion
	appVersion = ""

	// Using custom type
	//:getDefaultLevel
	logLevel = 0

	// Level name from custom type
	//:levelName:2
	levelStr = ""

	// Package variable
	//:getTimeout
	timeout = 0
)

func main() {
	fmt.Printf("Env Key: %s\n", envKey)
	fmt.Printf("Config: %s\n", configEntry)
	fmt.Printf("Version: %s\n", appVersion)
	fmt.Printf("Log Level: %d\n", logLevel)
	fmt.Printf("Level Name: %s\n", levelStr)
	fmt.Printf("Timeout: %d\n", timeout)
}
