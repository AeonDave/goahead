package stdlib_e

import "fmt"

var (
	//:os.Getenv:"HOME"
	homeDir = ""

	//:http.DetectContentType:=[]byte("plain text payload")
	mime = ""
)

func main() {
	//:strings.ToUpper:"detected"
	status := ""

	fmt.Printf("HOME: %s\n", homeDir)
	fmt.Printf("MIME: %s\n", mime)
	fmt.Printf("Status: %s\n", status)
}
