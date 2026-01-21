//go:build ignore

//go:ahead functions

package constants

import "fmt"

// Constants defined at package level - now properly extracted
const (
	Prefix    = "APP_"
	Separator = "::"
	Version   = "1.0.0"
)

// Type definitions - now properly supported
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Variables at package level
var defaultTimeout = 30

// Functions using package-level constants and types
func prefixed(key string) string {
	return Prefix + key
}

func formatted(key, value string) string {
	return key + Separator + value
}

func getVersion() string {
	return Version
}

func getDefaultLevel() Level {
	return LevelInfo
}

func levelName(l Level) string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", l)
	}
}

func getTimeout() int {
	return defaultTimeout
}
