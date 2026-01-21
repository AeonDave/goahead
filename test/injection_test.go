package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestInjectionBasic tests basic interface method injection
func TestInjectionBasic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Decode(s string) string {
	result := ""
	for _, c := range s {
		result += string(c ^ 0x42)
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Decode
type Decoder interface {
	Decode(s string) string
}

func main() {
	_ = Decode("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	// Verify function was injected
	if !strings.Contains(contentStr, "func Decode(s string) string") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}
	// Verify marker is preserved (not removed)
	if !strings.Contains(contentStr, "//:inject:Decode") {
		t.Errorf("Inject marker should be preserved, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithSpaceAfterSlashes tests "// :inject:" format (space after //)
func TestInjectionWithSpaceAfterSlashes(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Transform(s string) string {
	return "transformed:" + s
}
`)

	// Note: "// :inject:" with space - common after gofmt
	writeFile(t, dir, "main.go", `package main

// :inject:Transform
type Transformer interface {
	Transform(s string) string
}

func main() {
	_ = Transform("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "func Transform(s string) string") {
		t.Errorf("Function not injected with space format, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionMultipleMethods tests injecting multiple interface methods
func TestInjectionMultipleMethods(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Encrypt(s string) string {
	return "enc:" + s
}

func Decrypt(s string) string {
	return "dec:" + s
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Encrypt
//:inject:Decrypt
type Security interface {
	Encrypt(s string) string
	Decrypt(s string) string
}

func main() {
	_ = Encrypt("test")
	_ = Decrypt("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "func Encrypt") {
		t.Errorf("Encrypt not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func Decrypt") {
		t.Errorf("Decrypt not injected, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionMethodNotInInterface tests error when method not in interface
func TestInjectionMethodNotInInterface(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func NotAMethod(s string) string { return s }
`)

	writeFile(t, dir, "main.go", `package main

//:inject:NotAMethod
type MyInterface interface {
	SomeOtherMethod()
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err == nil {
		t.Fatal("Expected error when method not in interface")
	}
	if !strings.Contains(err.Error(), "not found in interface") {
		t.Errorf("Expected 'not found in interface' error, got: %v", err)
	}
}

// TestInjectionNoInterface tests error when marker not followed by interface
func TestInjectionNoInterface(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func SomeFunc() string { return "x" }
`)

	writeFile(t, dir, "main.go", `package main

//:inject:SomeFunc
var x = 1

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err == nil {
		t.Fatal("Expected error when inject not followed by interface")
	}
	if !strings.Contains(err.Error(), "must be followed by an interface") {
		t.Errorf("Expected 'must be followed by interface' error, got: %v", err)
	}
}

// TestInjectionUnusedImportsNotAdded tests that unused imports are NOT injected
func TestInjectionUnusedImportsNotAdded(t *testing.T) {
	dir := t.TempDir()

	// Helper file with multiple imports, but function only uses one
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import (
	"encoding/hex"
	"io"
	"net/http"
)

// HexEncode only uses hex package
func HexEncode(s string) string {
	return hex.EncodeToString([]byte(s))
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:HexEncode
type Encoder interface {
	HexEncode(s string) string
}

func main() {
	_ = HexEncode("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	// hex should be added (it's used)
	if !strings.Contains(contentStr, `"encoding/hex"`) {
		t.Errorf("Used import 'hex' should be added, got:\n%s", contentStr)
	}

	// io should NOT be in the import block (unused)
	if strings.Contains(contentStr, `"io"`) {
		t.Errorf("Unused import 'io' should NOT be added, got:\n%s", contentStr)
	}

	// net/http should NOT be in the import block (unused)
	if strings.Contains(contentStr, `"net/http"`) {
		t.Errorf("Unused import 'net/http' should NOT be added, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithImports tests injection includes required imports
func TestInjectionWithImports(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "encoding/base64"

func DecodeBase64(s string) string {
	data, _ := base64.StdEncoding.DecodeString(s)
	return string(data)
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:DecodeBase64
type Base64Decoder interface {
	DecodeBase64(s string) string
}

func main() {
	_ = DecodeBase64("dGVzdA==")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, `"encoding/base64"`) {
		t.Errorf("Import not added, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithConstants tests injection includes used constants
func TestInjectionWithConstants(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const xorKey = 0x42

func Unscramble(s string) string {
	result := ""
	for _, c := range s {
		result += string(c ^ xorKey)
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Unscramble
type Unscrambler interface {
	Unscramble(s string) string
}

func main() {
	_ = Unscramble("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "xorKey") {
		t.Errorf("Constant not added, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithHelperDependency tests dependent helper functions are injected
func TestInjectionWithHelperDependency(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const key byte = 0x2A

func xorByte(b byte) byte {
	return b ^ key
}

func Unshadow(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		result += string(xorByte(s[i]))
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Unshadow
type Shadower interface {
	Unshadow(s string) string
}

func main() {
	_ = Unshadow("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "func Unshadow") {
		t.Errorf("Unshadow not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func xorByte") {
		t.Errorf("Dependent helper function not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "const key") {
		t.Errorf("Dependent constant not injected, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionImplementationNotFound tests error when implementation not in helpers
func TestInjectionImplementationNotFound(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func SomethingElse() string { return "x" }
`)

	writeFile(t, dir, "main.go", `package main

//:inject:DoWork
type Worker interface {
	DoWork() string
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err == nil {
		t.Fatal("Expected error when implementation not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestInjectionWithValueReplacement tests injection works alongside value replacement
func TestInjectionWithValueReplacement(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const xorKey byte = 0x42

func Shadow(s string) string {
	result := ""
	for _, c := range s {
		result += string(byte(c) ^ xorKey)
	}
	return result
}

func Unshadow(s string) string {
	result := ""
	for _, c := range s {
		result += string(byte(c) ^ xorKey)
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

import "fmt"

//:Shadow:"secret"
var encrypted = ""

//:inject:Unshadow
type Shadower interface {
	Unshadow(s string) string
}

func main() {
	decrypted := Unshadow(encrypted)
	fmt.Println(decrypted)
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	// Verify value was replaced
	if strings.Contains(contentStr, `encrypted = ""`) {
		t.Errorf("Value not replaced, got:\n%s", contentStr)
	}
	// Verify function was injected
	if !strings.Contains(contentStr, "func Unshadow") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionHierarchy tests injection respects hierarchical resolution
func TestInjectionHierarchy(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Process() string {
	return "root-process"
}
`)

	writeFile(t, dir, "sub/helpers.go", `//go:build exclude
//go:ahead functions

package main

func Process() string {
	return "sub-process"
}
`)

	writeFile(t, dir, "sub/main.go", `package main

//:inject:Process
type Processor interface {
	Process() string
}

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	contentStr := string(content)

	// Should have sub-process, not root-process
	if !strings.Contains(contentStr, `"sub-process"`) {
		t.Errorf("Expected sub-process from hierarchy, got:\n%s", contentStr)
	}
}

// TestInjectionPreservesExistingImports tests injection with existing imports block
func TestInjectionPreservesExistingImports(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func Process(s string) string {
	return strings.ToUpper(s)
}
`)

	writeFile(t, dir, "main.go", `package main

import "fmt"

//:inject:Process
type Processor interface {
	Process(s string) string
}

func main() {
	fmt.Println(Process("test"))
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, `"fmt"`) {
		t.Errorf("Original import fmt not preserved, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `"strings"`) {
		t.Errorf("Injected import strings not added, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithVariables tests injection includes dependent variables
func TestInjectionWithVariables(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

var defaultKey = []byte{0x01, 0x02, 0x03, 0x04}

func Decrypt(s string) string {
	result := ""
	for i, c := range s {
		result += string(c ^ rune(defaultKey[i%len(defaultKey)]))
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Decrypt
type Decrypter interface {
	Decrypt(s string) string
}

func main() {
	_ = Decrypt("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "defaultKey") {
		t.Errorf("Variable dependency not injected, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithTypeInSignature tests type dependencies in function signature
func TestInjectionWithTypeInSignature(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

type Token struct {
	Value string
}

func FormatToken(t Token) string {
	return "token:" + t.Value
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:FormatToken
type TokenFormatter interface {
	FormatToken(t Token) string
}

func main() {
	_ = FormatToken(Token{Value: "x"})
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "type Token struct") {
		t.Errorf("Type dependency not injected, got:\n%s", contentStr)
	}

	verifyCompiles(t, dir)
}

// TestInjectionDanglingMarker tests error for marker at end of file
func TestInjectionDanglingMarker(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Foo() string { return "foo" }
`)

	writeFile(t, dir, "main.go", `package main

func main() {}

//:inject:Foo
`)

	err := internal.RunCodegen(dir, false)
	if err == nil {
		t.Fatal("Expected error for dangling inject marker")
	}
	if !strings.Contains(err.Error(), "must be followed by an interface") {
		t.Errorf("Expected interface error, got: %v", err)
	}
}

// TestInjectionIdempotent tests that running codegen twice replaces functions (not duplicates)
func TestInjectionIdempotent(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Decode(s string) string {
	result := ""
	for _, c := range s {
		result += string(c ^ 0x42)
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Decode
type Decoder interface {
	Decode(s string) string
}

func main() {
	_ = Decode("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First RunCodegen failed: %v", err)
	}

	content1, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr1 := string(content1)

	// Verify function was injected
	if !strings.Contains(contentStr1, "func Decode(s string) string") {
		t.Fatalf("Function not injected on first run")
	}

	// Marker should still be present
	if !strings.Contains(contentStr1, "//:inject:Decode") {
		t.Fatalf("Marker should remain after injection")
	}

	// Count occurrences
	count1 := strings.Count(contentStr1, "func Decode(s string) string")
	if count1 != 1 {
		t.Fatalf("Expected 1 Decode function after first run, got %d", count1)
	}

	// Second run (simulating subsequent build)
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second RunCodegen failed: %v", err)
	}

	content2, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr2 := string(content2)

	// Count occurrences again - should still be 1 (replaced, not duplicated)
	count2 := strings.Count(contentStr2, "func Decode(s string) string")
	if count2 != 1 {
		t.Fatalf("Expected 1 Decode function after second run, got %d (duplication occurred!)", count2)
	}

	// Marker should still be present
	if !strings.Contains(contentStr2, "//:inject:Decode") {
		t.Fatalf("Marker should remain after second injection")
	}

	// Third run for good measure
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Third RunCodegen failed: %v", err)
	}

	content3, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr3 := string(content3)

	count3 := strings.Count(contentStr3, "func Decode(s string) string")
	if count3 != 1 {
		t.Fatalf("Expected 1 Decode function after third run, got %d (duplication occurred!)", count3)
	}

	verifyCompiles(t, dir)
}

// TestInjectionUpdatesOnHelperChange tests that changing helper updates the injected function
func TestInjectionUpdatesOnHelperChange(t *testing.T) {
	dir := t.TempDir()

	// Initial helper with version 1
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Process() string {
	return "version1"
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Process
type Processor interface {
	Process() string
}

func main() {
	_ = Process()
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	content1, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(content1), `"version1"`) {
		t.Fatalf("Expected version1 in first run")
	}

	// Update helper to version 2
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Process() string {
	return "version2"
}
`)

	// Second run - should update the function
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	content2, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr2 := string(content2)

	// Should have version2, not version1
	if !strings.Contains(contentStr2, `"version2"`) {
		t.Fatalf("Expected version2 after helper update, got:\n%s", contentStr2)
	}
	if strings.Contains(contentStr2, `"version1"`) {
		t.Fatalf("version1 should be replaced by version2")
	}

	// Should still have only one Process function
	count := strings.Count(contentStr2, "func Process() string")
	if count != 1 {
		t.Fatalf("Expected 1 Process function, got %d", count)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithPlaceholderMultipleRuns tests multiple runs with both placeholder and inject
func TestInjectionWithPlaceholderMultipleRuns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const key byte = 0x42

func Encode(s string) string {
	result := ""
	for _, c := range s {
		result += string(byte(c) ^ key)
	}
	return result
}

func Decode(s string) string {
	result := ""
	for _, c := range s {
		result += string(byte(c) ^ key)
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

import "fmt"

//:Encode:"password123"
var secret = ""

//:inject:Decode
type Decoder interface {
	Decode(s string) string
}

func main() {
	fmt.Println(Decode(secret))
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	content1, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr1 := string(content1)

	// Placeholder should be replaced
	if strings.Contains(contentStr1, `secret = ""`) {
		t.Errorf("Placeholder not replaced on first run")
	}
	// Inject marker should remain
	if !strings.Contains(contentStr1, "//:inject:Decode") {
		t.Errorf("Inject marker should remain")
	}
	// Function should be injected
	if !strings.Contains(contentStr1, "func Decode") {
		t.Errorf("Function not injected")
	}

	verifyCompiles(t, dir)

	// Second run - placeholder already replaced, inject should still work
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	content2, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr2 := string(content2)

	// Should still have exactly one Decode function
	count := strings.Count(contentStr2, "func Decode(s string) string")
	if count != 1 {
		t.Errorf("Expected 1 Decode function after second run, got %d", count)
	}

	verifyCompiles(t, dir)

	// Third run
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Third run failed: %v", err)
	}

	content3, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	count3 := strings.Count(string(content3), "func Decode(s string) string")
	if count3 != 1 {
		t.Errorf("Expected 1 Decode function after third run, got %d", count3)
	}

	verifyCompiles(t, dir)
}

// TestInjectionMultipleInterfacesMultipleRuns tests multiple interfaces with inject over multiple builds
func TestInjectionMultipleInterfacesMultipleRuns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Encrypt(s string) string { return "enc:" + s }
func Decrypt(s string) string { return "dec:" + s }
func Hash(s string) string { return "hash:" + s }
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Encrypt
//:inject:Decrypt
type Crypto interface {
	Encrypt(s string) string
	Decrypt(s string) string
}

//:inject:Hash
type Hasher interface {
	Hash(s string) string
}

func main() {
	_ = Encrypt("x")
	_ = Decrypt("y")
	_ = Hash("z")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	content1, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr1 := string(content1)

	if strings.Count(contentStr1, "func Encrypt") != 1 {
		t.Errorf("Expected 1 Encrypt")
	}
	if strings.Count(contentStr1, "func Decrypt") != 1 {
		t.Errorf("Expected 1 Decrypt")
	}
	if strings.Count(contentStr1, "func Hash") != 1 {
		t.Errorf("Expected 1 Hash")
	}

	verifyCompiles(t, dir)

	// Second run
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	content2, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr2 := string(content2)

	if strings.Count(contentStr2, "func Encrypt") != 1 {
		t.Errorf("Expected 1 Encrypt after second run, got %d", strings.Count(contentStr2, "func Encrypt"))
	}
	if strings.Count(contentStr2, "func Decrypt") != 1 {
		t.Errorf("Expected 1 Decrypt after second run")
	}
	if strings.Count(contentStr2, "func Hash") != 1 {
		t.Errorf("Expected 1 Hash after second run")
	}

	verifyCompiles(t, dir)
}

// TestInjectionDuplicateImportsHandled tests that duplicate imports are handled correctly
func TestInjectionDuplicateImportsHandled(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import "strings"

func ToUpper(s string) string {
	return strings.ToUpper(s)
}
`)

	// Source already has strings import
	writeFile(t, dir, "main.go", `package main

import (
	"fmt"
	"strings"
)

//:inject:ToUpper
type Upper interface {
	ToUpper(s string) string
}

func main() {
	fmt.Println(strings.ToLower("TEST"))
	fmt.Println(ToUpper("test"))
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// First run
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	verifyCompiles(t, dir)

	// Second run - should not create duplicate imports
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	verifyCompiles(t, dir)

	// Third run
	err = internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("Third run failed: %v", err)
	}

	verifyCompiles(t, dir)
}

// TestInjectionWithDependenciesMultipleRuns tests that dependencies are handled across runs
func TestInjectionWithDependenciesMultipleRuns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const secretKey = 0xAB

var salt = []byte{0x01, 0x02}

func Scramble(s string) string {
	result := ""
	for i, c := range s {
		result += string(byte(c) ^ secretKey ^ salt[i%len(salt)])
	}
	return result
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Scramble
type Scrambler interface {
	Scramble(s string) string
}

func main() {
	_ = Scramble("test")
}
`)

	writeFile(t, dir, "go.mod", `module testmod
go 1.22
`)

	// Run 3 times
	for i := 1; i <= 3; i++ {
		err := internal.RunCodegen(dir, false)
		if err != nil {
			t.Fatalf("Run %d failed: %v", i, err)
		}

		content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		contentStr := string(content)

		// Should have exactly one of each
		if strings.Count(contentStr, "func Scramble") != 1 {
			t.Errorf("Run %d: Expected 1 Scramble function", i)
		}
		if strings.Count(contentStr, "secretKey") < 1 {
			t.Errorf("Run %d: Expected secretKey constant", i)
		}
		if strings.Count(contentStr, "salt") < 1 {
			t.Errorf("Run %d: Expected salt variable", i)
		}

		verifyCompiles(t, dir)
	}
}
