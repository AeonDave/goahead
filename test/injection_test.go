package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AeonDave/goahead/internal"
)

// TestInjectionBasic tests basic function injection
func TestInjectionBasic(t *testing.T) {
	dir := t.TempDir()

	// Helper file with function to inject
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

	// Source file requesting injection
	writeFile(t, dir, "main.go", `package main

//:inject:Decode

func main() {
	_ = Decode("test")
}
`)

	// Need go.mod for compilation
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
	if !strings.Contains(contentStr, "c ^ 0x42") {
		t.Errorf("Function body not injected correctly, got:\n%s", contentStr)
	}

	// Verify generated code compiles
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

	// Verify import was added
	if !strings.Contains(contentStr, `"encoding/base64"`) {
		t.Errorf("Import not added, got:\n%s", contentStr)
	}
	// Verify function was injected
	if !strings.Contains(contentStr, "func DecodeBase64") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
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

	// Verify constant was added
	if !strings.Contains(contentStr, "xorKey") && !strings.Contains(contentStr, "0x42") {
		t.Errorf("Constant not added, got:\n%s", contentStr)
	}
	// Verify function was injected
	if !strings.Contains(contentStr, "func Unscramble") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
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

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionWithTypeSignature tests type dependencies used only in signature
func TestInjectionWithTypeSignature(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

type Token struct {
	Value string
}

func FormatToken(t Token) string {
	return "token"
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:FormatToken

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
	if !strings.Contains(contentStr, "func FormatToken") {
		t.Errorf("FormatToken not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionWithComplexImports tests imports for crypto packages
func TestInjectionWithComplexImports(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import (
	"crypto/aes"
	"crypto/cipher"
)

func NewGCM() cipher.AEAD {
	block, _ := aes.NewCipher(make([]byte, 16))
	gcm, _ := cipher.NewGCM(block)
	return gcm
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:NewGCM

func main() {
	_ = NewGCM()
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

	if !strings.Contains(contentStr, `"crypto/aes"`) || !strings.Contains(contentStr, `"crypto/cipher"`) {
		t.Errorf("Complex imports not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func NewGCM") {
		t.Errorf("NewGCM not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionWithChaCha20Poly1305 tests imports for chacha20-poly1305
func TestInjectionWithChaCha20Poly1305(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

import (
	"crypto/cipher"
	"golang.org/x/crypto/chacha20poly1305"
)

func NewChaCha20() cipher.AEAD {
	key := make([]byte, chacha20poly1305.KeySize)
	aead, _ := chacha20poly1305.New(key)
	return aead
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:NewChaCha20

func main() {
	_ = NewChaCha20()
}
`)

	// Note: This test verifies injection only, not compilation
	// because golang.org/x/crypto is external
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, `"golang.org/x/crypto/chacha20poly1305"`) || !strings.Contains(contentStr, `"crypto/cipher"`) {
		t.Errorf("ChaCha20 imports not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func NewChaCha20") {
		t.Errorf("NewChaCha20 not injected, got:\n%s", contentStr)
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

	// Verify value was replaced (encrypted should not be "")
	if strings.Contains(contentStr, `encrypted = ""`) {
		t.Errorf("Value not replaced, got:\n%s", contentStr)
	}
	// Verify function was injected
	if !strings.Contains(contentStr, "func Unshadow") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionHierarchy tests injection respects hierarchical resolution
func TestInjectionHierarchy(t *testing.T) {
	dir := t.TempDir()

	// Root helper
	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Decoder() string {
	return "root-decoder"
}
`)

	// Sub helper (shadows root)
	writeFile(t, dir, "sub/helpers.go", `//go:build exclude
//go:ahead functions

package main

func Decoder() string {
	return "sub-decoder"
}
`)

	// Sub source file - should get sub's Decoder
	writeFile(t, dir, "sub/main.go", `package main

//:inject:Decoder

func main() {}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "sub/main.go"))
	contentStr := string(content)

	// Should have sub-decoder, not root-decoder
	if !strings.Contains(contentStr, `"sub-decoder"`) {
		t.Errorf("Expected sub-decoder from hierarchy, got:\n%s", contentStr)
	}
}

// TestInjectionNotFound tests error handling when function not found
func TestInjectionNotFound(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Exists() string { return "exists" }
`)

	writeFile(t, dir, "main.go", `package main

//:inject:NonExistent

func main() {}
`)

	// Should not fail, just warn
	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen should not fail: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	// Marker should remain (not replaced)
	if !strings.Contains(contentStr, "//:inject:NonExistent") {
		t.Errorf("Marker should remain when function not found, got:\n%s", contentStr)
	}
}

// TestMultipleInjections tests injecting multiple functions
func TestMultipleInjections(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

func Encode(s string) string {
	return "encoded:" + s
}

func Decode(s string) string {
	return "decoded:" + s
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Encode

//:inject:Decode

func main() {
	_ = Encode("test")
	_ = Decode("test")
}
`)

	err := internal.RunCodegen(dir, false)
	if err != nil {
		t.Fatalf("RunCodegen failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "func Encode") {
		t.Errorf("Encode not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func Decode") {
		t.Errorf("Decode not injected, got:\n%s", contentStr)
	}
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

	if !strings.Contains(contentStr, "func Decrypt") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "defaultKey") {
		t.Errorf("Variable dependency not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionWithMultipleDependencies tests deep dependency chains
func TestInjectionWithMultipleDependencies(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

const prefix = "pre_"
const suffix = "_suf"

func addPrefix(s string) string {
	return prefix + s
}

func addSuffix(s string) string {
	return s + suffix
}

func Transform(s string) string {
	return addSuffix(addPrefix(s))
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:Transform

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

	// Transform depends on addPrefix and addSuffix which depend on prefix and suffix
	if !strings.Contains(contentStr, "func Transform") {
		t.Errorf("Main function not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func addPrefix") {
		t.Errorf("addPrefix helper not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func addSuffix") {
		t.Errorf("addSuffix helper not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `prefix = "pre_"`) {
		t.Errorf("prefix constant not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `suffix = "_suf"`) {
		t.Errorf("suffix constant not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
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
	if !strings.Contains(contentStr, "func Process") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}

// TestInjectionInNestedStruct tests injection doesn't break struct fields
func TestInjectionWithStructType(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "helpers.go", `//go:build exclude
//go:ahead functions

package main

type Config struct {
	Key   string
	Value int
}

func NewConfig() Config {
	return Config{Key: "default", Value: 42}
}
`)

	writeFile(t, dir, "main.go", `package main

//:inject:NewConfig

func main() {
	cfg := NewConfig()
	_ = cfg
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

	if !strings.Contains(contentStr, "type Config struct") {
		t.Errorf("Type not injected, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "func NewConfig") {
		t.Errorf("Function not injected, got:\n%s", contentStr)
	}

	// Verify generated code compiles
	verifyCompiles(t, dir)
}
