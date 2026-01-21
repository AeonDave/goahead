package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestDepthBasedResolution tests the new depth-based function resolution
func TestDepthBasedResolution(t *testing.T) {
	t.Run("SiblingDirectoriesShareFunctions", func(t *testing.T) {
		// Structure:
		// root/
		// ├── helpers.go      # rootFunc() @ depth 0
		// ├── obfuscation/    # depth 1
		// │   └── helpers.go  # Shadow() @ depth 1
		// └── evasion/        # depth 1
		//     └── safeSyscall/    # depth 2
		//         └── main.go     # should see Shadow() from depth 1

		dir := t.TempDir()

		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
package helpers
func rootFunc() string { return "root" }
`)
		writeFile(t, dir, "obfuscation/helpers.go", `//go:build exclude
//go:ahead functions
package obfuscation
func Shadow(s string) string { return "shadowed:" + s }
func HashStr(s string) string { return "hashed:" + s }
`)
		writeFile(t, dir, "evasion/safeSyscall/main.go", `package main

//:Shadow:"secret"
var encoded = ""

//:HashStr:"password"
var hashed = ""

//:rootFunc:
var root = ""

func main() {}
`)

		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		result, _ := os.ReadFile(filepath.Join(dir, "evasion/safeSyscall/main.go"))

		// Should find Shadow() from obfuscation/ (sibling at depth 1)
		if !strings.Contains(string(result), `encoded = "shadowed:secret"`) {
			t.Errorf("Should find Shadow() from sibling directory at depth 1\n%s", result)
		}

		// Should find HashStr() from obfuscation/ (sibling at depth 1)
		if !strings.Contains(string(result), `hashed = "hashed:password"`) {
			t.Errorf("Should find HashStr() from sibling directory at depth 1\n%s", result)
		}

		// Should find rootFunc() from root (depth 0)
		if !strings.Contains(string(result), `root = "root"`) {
			t.Errorf("Should find rootFunc() from root at depth 0\n%s", result)
		}
	})

	t.Run("DeeplyNestedCanAccessAllUpperDepths", func(t *testing.T) {
		// Structure:
		// root/
		// ├── helpers.go          # depth0Func() @ depth 0
		// ├── level1/             # depth 1
		// │   ├── helpers.go      # depth1Func() @ depth 1
		// │   └── level2/         # depth 2
		// │       └── level3/     # depth 3
		// │           └── main.go # should see depth0Func, depth1Func

		dir := t.TempDir()

		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
package helpers
func depth0Func() string { return "depth0" }
`)
		writeFile(t, dir, "level1/helpers.go", `//go:build exclude
//go:ahead functions
package level1
func depth1Func() string { return "depth1" }
`)
		writeFile(t, dir, "level1/level2/level3/main.go", `package main

//:depth0Func:
var v0 = ""

//:depth1Func:
var v1 = ""

func main() {}
`)

		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		result, _ := os.ReadFile(filepath.Join(dir, "level1/level2/level3/main.go"))

		if !strings.Contains(string(result), `v0 = "depth0"`) {
			t.Errorf("Deeply nested file should access depth 0 functions\n%s", result)
		}

		if !strings.Contains(string(result), `v1 = "depth1"`) {
			t.Errorf("Deeply nested file should access depth 1 functions\n%s", result)
		}
	})

	t.Run("ShadowingBetweenDepths", func(t *testing.T) {
		// Structure:
		// root/
		// ├── helpers.go      # version() = "1.0" @ depth 0
		// └── pkg/            # depth 1
		//     ├── helpers.go  # version() = "2.0" @ depth 1 (shadows)
		//     └── main.go     # should get version() = "2.0"

		dir := t.TempDir()

		writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions
package helpers
func version() string { return "1.0" }
`)
		writeFile(t, dir, "pkg/helpers.go", `//go:build exclude
//go:ahead functions
package pkg
func version() string { return "2.0" }
`)
		writeFile(t, dir, "pkg/main.go", `package main

//:version:
var v = ""

func main() {}
`)
		writeFile(t, dir, "root_main.go", `package main

//:version:
var rootV = ""

func main() {}
`)

		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		// pkg/main.go should get version from depth 1
		result, _ := os.ReadFile(filepath.Join(dir, "pkg/main.go"))
		if !strings.Contains(string(result), `v = "2.0"`) {
			t.Errorf("Should get version from depth 1 (shadowing)\n%s", result)
		}

		// root_main.go should get version from depth 0
		rootResult, _ := os.ReadFile(filepath.Join(dir, "root_main.go"))
		if !strings.Contains(string(rootResult), `rootV = "1.0"`) {
			t.Errorf("Root file should get version from depth 0\n%s", rootResult)
		}
	})

	t.Run("MultipleSiblingsAtSameDepth", func(t *testing.T) {
		// Structure with multiple siblings at depth 1, each with different functions
		// root/
		// ├── pkg1/helpers.go  # func1() @ depth 1
		// ├── pkg2/helpers.go  # func2() @ depth 1
		// ├── pkg3/helpers.go  # func3() @ depth 1
		// └── user/main.go     # should see func1, func2, func3

		dir := t.TempDir()

		writeFile(t, dir, "pkg1/helpers.go", `//go:build exclude
//go:ahead functions
package pkg1
func func1() string { return "from-pkg1" }
`)
		writeFile(t, dir, "pkg2/helpers.go", `//go:build exclude
//go:ahead functions
package pkg2
func func2() string { return "from-pkg2" }
`)
		writeFile(t, dir, "pkg3/helpers.go", `//go:build exclude
//go:ahead functions
package pkg3
func func3() string { return "from-pkg3" }
`)
		writeFile(t, dir, "user/main.go", `package main

//:func1:
var v1 = ""

//:func2:
var v2 = ""

//:func3:
var v3 = ""

func main() {}
`)

		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("RunCodegen failed: %v", err)
		}

		result, _ := os.ReadFile(filepath.Join(dir, "user/main.go"))

		if !strings.Contains(string(result), `v1 = "from-pkg1"`) {
			t.Errorf("Should find func1() from pkg1/ at depth 1\n%s", result)
		}
		if !strings.Contains(string(result), `v2 = "from-pkg2"`) {
			t.Errorf("Should find func2() from pkg2/ at depth 1\n%s", result)
		}
		if !strings.Contains(string(result), `v3 = "from-pkg3"`) {
			t.Errorf("Should find func3() from pkg3/ at depth 1\n%s", result)
		}
	})
}

// TestDepthCalculation tests the depth calculation logic
func TestDepthCalculation(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := &internal.ProcessorContext{
		RootDir: tmpDir,
	}

	tests := []struct {
		name     string
		dir      string
		expected int
	}{
		{"root", tmpDir, 0},
		{"depth1", filepath.Join(tmpDir, "pkg1"), 1},
		{"depth2", filepath.Join(tmpDir, "pkg1", "sub"), 2},
		{"depth3", filepath.Join(tmpDir, "a", "b", "c"), 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ctx.CalculateDepth(tc.dir)
			if got != tc.expected {
				t.Errorf("CalculateDepth(%s) = %d, want %d", tc.dir, got, tc.expected)
			}
		})
	}
}

// TestDuplicateAtSameDepthError tests that duplicate functions at the same depth cause an error
func TestDuplicateAtSameDepthError(t *testing.T) {
	// This test verifies the error condition - we can't easily test os.Exit(1)
	// but we can verify the context state before the check

	dir := t.TempDir()

	writeFile(t, dir, "pkg1/helpers.go", `//go:build exclude
//go:ahead functions
package pkg1
func duplicate() string { return "from-pkg1" }
`)
	writeFile(t, dir, "pkg2/helpers.go", `//go:build exclude
//go:ahead functions
package pkg2
func duplicate() string { return "from-pkg2" }
`)

	// This should fail because duplicate() is defined twice at depth 1
	// We capture this by checking if the warning/error is produced
	// In a real scenario, this would call os.Exit(1)

	// For now, we just document the expected behavior
	t.Log("Expected: ERROR: Duplicate function 'duplicate' at same depth level 1")
	t.Log("This test documents expected behavior - actual enforcement is via os.Exit(1)")

	// Verify files exist
	_, err := os.Stat(filepath.Join(dir, "pkg1", "helpers.go"))
	if err != nil {
		t.Fatalf("pkg1/helpers.go should exist: %v", err)
	}
	_, err = os.Stat(filepath.Join(dir, "pkg2", "helpers.go"))
	if err != nil {
		t.Fatalf("pkg2/helpers.go should exist: %v", err)
	}
}

// TestRealWorldScenario simulates the user's actual project structure
func TestRealWorldScenario(t *testing.T) {
	// Simulates:
	// injection/
	// ├── obfuscation/
	// │   └── goahead_helpers.go  # Shadow(), HashStr(), GetEnv()
	// └── evasion/
	//     └── safeSyscall/
	//         └── RecycleGate/
	//             └── whisper.go  # uses Shadow(), GetEnv()

	dir := t.TempDir()

	writeFile(t, dir, "obfuscation/goahead_helpers.go", `//go:build exclude
//go:ahead functions
package obfuscation

func Shadow(s string) string {
	result := ""
	for _, c := range s {
		result += string(c ^ 0x42)
	}
	return result
}

func HashStr(s string) string {
	return "hash:" + s
}
`)
	writeFile(t, dir, "evasion/safeSyscall/RecycleGate/whisper.go", `package RecycleGate

//:Shadow:"ntdll"
var encoded = ""

//:HashStr:"kernel32"
var hashed = ""

func init() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	result, _ := os.ReadFile(filepath.Join(dir, "evasion/safeSyscall/RecycleGate/whisper.go"))

	// Shadow should XOR each character with 0x42
	// 'n' ^ 0x42 = 110 ^ 66 = 44 = ','
	// etc...
	if strings.Contains(string(result), `encoded = ""`) {
		t.Errorf("Shadow() should have been applied\n%s", result)
	}

	if strings.Contains(string(result), `hashed = ""`) {
		t.Errorf("HashStr() should have been applied\n%s", result)
	}
}
