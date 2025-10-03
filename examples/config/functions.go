//go:build ignore

//go:ahead functions

package config

import (
	"os"
	"strings"
)

func serviceName() string {
	return "billing-api"
}

func servicePort() int {
	return 8080
}

func enableTLS() bool {
	return true
}

func env(key string) string {
	return os.Getenv(key)
}

func sanitizeCSV(input string) string {
	parts := strings.Split(input, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, ",")
}
