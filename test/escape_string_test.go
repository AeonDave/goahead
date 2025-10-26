package test

import (
	"strconv"
	"testing"
	_ "unsafe"
)

//go:linkname escapeString github.com/AeonDave/goahead/internal.escapeString
func escapeString(string) string

func TestEscapeStringPrefersQuotedLiteralWhenBacktickPresent(t *testing.T) {
	input := "path\\`dir"
	got := escapeString(input)
	want := strconv.Quote(input)
	if got != want {
		t.Fatalf("escapeString(%q) = %q, want %q", input, got, want)
	}
}
