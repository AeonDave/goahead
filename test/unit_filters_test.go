package test

import (
	"path/filepath"
	"testing"

	internal "github.com/AeonDave/goahead/internal"
)

func TestFilterUserFiles(t *testing.T) {
	t.Setenv("GOAHEAD_VERBOSE", "")
	t.Setenv("GOPATH", filepath.FromSlash("/tmp/go"))
	t.Setenv("GOROOT", filepath.FromSlash("/tmp/go-root"))

	files := []string{
		"main.go",
		filepath.Join("vendor", "dep.go"),
		filepath.Join("pkg", "handler_test.go"),
		filepath.Join("pkg", "test", "data.go"),
		filepath.Join(filepath.FromSlash("/tmp/go-root"), "src", "fmt", "print.go"),
	}

	got := internal.FilterUserFiles(files)
	want := map[string]struct{}{
		"main.go":                               {},
		filepath.Join("pkg", "handler_test.go"): {},
		filepath.Join("pkg", "test", "data.go"): {},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d (%v)", len(want), len(got), got)
	}

	for _, file := range got {
		if _, ok := want[file]; !ok {
			t.Fatalf("unexpected file %q in result %v", file, got)
		}
		delete(want, file)
	}

	if len(want) != 0 {
		t.Fatalf("missing expected files: %v", want)
	}
}

func TestFilterUserFilesVerbose(t *testing.T) {
	t.Setenv("GOAHEAD_VERBOSE", "1")
	t.Setenv("GOPATH", filepath.FromSlash("/tmp/go"))
	t.Setenv("GOROOT", filepath.FromSlash("/tmp/go-root"))

	files := []string{filepath.Join("vendor", "dep.go")}
	got := internal.FilterUserFiles(files)
	if len(got) != 0 {
		t.Fatalf("expected vendor file to be skipped, got %v", got)
	}
}
