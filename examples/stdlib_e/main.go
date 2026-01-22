package stdlib_e

import "fmt"

var (
	//:os.Getenv:"HOME"
	homeDir = "C:\\Users\\novad\\"

	//:http.DetectContentType:=[]byte("plain text payload")
	mime = "text/plain; charset=utf-8"
)

func main() {
	//:strings.ToUpper:"detected"
	status := "DETECTED"

	fmt.Printf("HOME: %s\n", homeDir)
	fmt.Printf("MIME: %s\n", mime)
	fmt.Printf("Status: %s\n", status)
}
