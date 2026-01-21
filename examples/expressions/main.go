package expressions

import "fmt"

var (
	// Map literal - colons inside {} are preserved
	//:getMapLen
	mapSize = 0

	// Struct literal - colons inside {} are preserved
	//:getPointSum
	pointSum = 0

	// Count URLs (with colons in strings)
	//:countURLs
	urlCount = 0

	// Raw expression with slice literal
	//:http.DetectContentType:=[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pngMime = ""

	// Raw expression with nested function calls
	//:filepath.Base:="/usr/local/bin/myapp"
	baseName = ""

	// Raw expression with base64
	//:base64.StdEncoding.EncodeToString:=[]byte("Hello, GoAhead!")
	encoded = ""
)

func main() {
	fmt.Printf("Map size: %d\n", mapSize)
	fmt.Printf("Point sum: %d\n", pointSum)
	fmt.Printf("URL count: %d\n", urlCount)
	fmt.Printf("PNG MIME: %s\n", pngMime)
	fmt.Printf("Base name: %s\n", baseName)
	fmt.Printf("Base64: %s\n", encoded)
}
