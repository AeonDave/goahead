package config

import (
	"fmt"
	"time"
)

type Config struct {
	Name           string
	Port           int
	TLS            bool
	AllowedOrigins string
	Timeout        time.Duration
}

var (
	//:serviceName
	name = ""

	//:servicePort
	port = 0

	//:enableTLS
	tlsEnabled = false

	//:sanitizeCSV:"https://app.example.com , https://admin.example.com "
	origins = ""
)

func main() {
	cfg := Config{
		Name:           name,
		Port:           port,
		TLS:            tlsEnabled,
		AllowedOrigins: origins,
		Timeout:        30 * time.Second,
	}

	//:env:"ADMIN_EMAIL"
	adminEmail := ""

	fmt.Printf("Config: %#v\n", cfg)
	fmt.Printf("Admin email: %s\n", adminEmail)
}
