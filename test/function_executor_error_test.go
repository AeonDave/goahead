package test

import (
	"strings"
	"testing"

	internal "github.com/AeonDave/goahead/internal"
)

func TestExecuteFunctionReportsStdlibResolutionFailure(t *testing.T) {
	ctx := &internal.ProcessorContext{
		Functions:      make(map[string]*internal.UserFunction),
		FunctionsByDir: make(map[string]map[string]*internal.UserFunction),
		RootDir:        t.TempDir(),
		FuncFiles:      nil,
		TempDir:        t.TempDir(),
	}
	executor := internal.NewFunctionExecutor(ctx)

	t.Setenv("PATH", "")

	_, _, err := executor.ExecuteFunction("http.DetectContentType", `"data"`, ctx.RootDir)
	if err == nil {
		t.Fatalf("expected error when go toolchain is unavailable")
	}

	message := err.Error()
	if !strings.Contains(message, "automatic standard library resolution failed") {
		t.Fatalf("missing stdlib resolution hint in error: %s", message)
	}
	if !strings.Contains(message, "http=<import path>") {
		t.Fatalf("missing placeholder import hint: %s", message)
	}
}
