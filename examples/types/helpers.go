//go:build ignore

//go:ahead functions

package types

import (
	"fmt"
	"strings"
)

// Custom type definitions supported in helper files
type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusCompleted
	StatusFailed
)

type Config struct {
	Host string
	Port int
}

// Type alias
type StringList = []string

// Function returning custom type
func getDefaultStatus() Status {
	return StatusActive
}

// Function using custom struct
func getDefaultConfig() string {
	cfg := Config{Host: "localhost", Port: 8080}
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

// Function using type alias
func joinList(items StringList, sep string) string {
	return strings.Join(items, sep)
}

// Function accepting custom type
func statusName(s Status) string {
	names := []string{"Pending", "Active", "Completed", "Failed"}
	if int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

// Nested type usage
type Point struct{ X, Y int }
type Rectangle struct{ TopLeft, BottomRight Point }

func getRectArea() int {
	r := Rectangle{
		TopLeft:     Point{X: 0, Y: 0},
		BottomRight: Point{X: 10, Y: 5},
	}
	return (r.BottomRight.X - r.TopLeft.X) * (r.BottomRight.Y - r.TopLeft.Y)
}
