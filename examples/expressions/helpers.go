//go:build ignore

//go:ahead functions
//go:ahead import http=net/http
//go:ahead import filepath=path/filepath
//go:ahead import base64=encoding/base64

package expressions

// Helper function using map literals with colons inside
func getMapLen() int {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	return len(m)
}

// Helper using struct literals with colons inside
func getPointSum() int {
	type Point struct{ X, Y int }
	p := Point{X: 10, Y: 20}
	return p.X + p.Y
}

// Helper with slice containing colons in string
func getURLs() []string {
	return []string{
		"http://localhost:8080",
		"https://api.example.com:443/v1",
	}
}

func countURLs() int {
	return len(getURLs())
}
