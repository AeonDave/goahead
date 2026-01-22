package constants

import "fmt"

var (
	// Using helper that references package constants
	//:prefixed:"DATABASE_URL"
	envKey = "APP_DATABASE_URL"

	// Formatted key-value
	//:formatted:"config":"production"
	configEntry = "config::production"

	// Get version constant
	//:getVersion
	appVersion = "1.0.0"

	// Using custom type
	//:getDefaultLevel
	logLevel = 1

	// Level name from custom type
	//:levelName:2
	levelStr = "WARN"

	// Package variable
	//:getTimeout
	timeout = 30
)

func main() {
	fmt.Printf("Env Key: %s\n", envKey)
	fmt.Printf("Config: %s\n", configEntry)
	fmt.Printf("Version: %s\n", appVersion)
	fmt.Printf("Log Level: %d\n", logLevel)
	fmt.Printf("Level Name: %s\n", levelStr)
	fmt.Printf("Timeout: %d\n", timeout)
}
