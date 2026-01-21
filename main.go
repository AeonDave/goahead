package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AeonDave/goahead/internal"
)

func main() {
	if isToolexecMode() {
		toolexecManager := internal.NewToolexecManager()
		toolexecManager.RunAsToolexec()
		return
	}
	// If no arguments are provided, show help/usage instead of running codegen
	if len(os.Args) == 1 {
		showHelp()
		return
	}
	config := parseFlags()

	if config.Help {
		showHelp()
		return
	}

	if config.Version {
		fmt.Printf("goahead version %s\n", internal.Version)
		return
	}

	if config.Verbose {
		fmt.Printf("Running goahead in standalone mode\n")
		fmt.Printf("Processing directory: %s\n", config.Dir)
	}

	if err := internal.RunCodegen(config.Dir, config.Verbose); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func isToolexecMode() bool {
	if len(os.Args) < 2 {
		return false
	}
	// If first arg starts with "-", it's a flag, not toolexec mode
	if strings.HasPrefix(os.Args[1], "-") {
		return false
	}
	// In toolexec mode, Go passes the tool path as first argument
	// The tool is typically in GOROOT/pkg/tool/GOOS_GOARCH/
	// Accept any executable that looks like a Go tool
	arg := os.Args[1]
	return strings.Contains(arg, "go"+string(os.PathSeparator)+"pkg"+string(os.PathSeparator)+"tool") ||
		strings.HasSuffix(arg, "compile") || strings.HasSuffix(arg, "compile.exe") ||
		strings.HasSuffix(arg, "link") || strings.HasSuffix(arg, "link.exe") ||
		strings.HasSuffix(arg, "asm") || strings.HasSuffix(arg, "asm.exe") ||
		strings.HasSuffix(arg, "cgo") || strings.HasSuffix(arg, "cgo.exe") ||
		strings.HasSuffix(arg, "pack") || strings.HasSuffix(arg, "pack.exe") ||
		strings.HasSuffix(arg, "buildid") || strings.HasSuffix(arg, "buildid.exe")
}

func parseFlags() *internal.Config {
	config := &internal.Config{}

	flag.StringVar(&config.Dir, "dir", ".", "Directory to process")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&config.Help, "help", false, "Show help")
	flag.BoolVar(&config.Version, "version", false, "Show version")
	flag.Parse()

	return config
}

func showHelp() {
	const boxInnerWidth = 79

	center := func(s string, width int) string {
		r := []rune(s)
		if len(r) >= width {
			return string(r[:width])
		}
		pad := width - len(r)
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}

	shortVersionForBanner := func(version string) string {
		version = strings.TrimSpace(version)
		if version == "" {
			return "dev"
		}
		if version == "(devel)" {
			return "dev"
		}

		// Preserve a dirty marker, but drop long pseudo-version details.
		meta := ""
		if i := strings.Index(version, "+"); i >= 0 {
			meta = version[i:]
			version = version[:i]
		}
		if meta != "+dirty" {
			// Only keep a concise dirty marker in the banner.
			meta = ""
		}

		// Trim Go pseudo-version suffixes like:
		//   v0.1.1-0.20260121133307-33bc43fdf900
		// keeping just v0.1.1
		if i := strings.Index(version, "-0."); i >= 0 {
			// Ensure it looks like -0.<14digits>-<hash>
			if len(version) >= i+3+14 {
				isDigits := true
				for _, c := range version[i+3 : i+3+14] {
					if c < '0' || c > '9' {
						isDigits = false
						break
					}
				}
				if isDigits {
					version = version[:i]
				}
			}
		}

		if version == "" {
			version = "dev"
		}
		return version + meta
	}

	short := shortVersionForBanner(internal.Version)
	if !strings.HasPrefix(short, "v") {
		short = "v" + short
	}
	title := "GOAHEAD " + short
	header := "╔" + strings.Repeat("═", boxInnerWidth) + "╗\n" +
		"║" + center(title, boxInnerWidth) + "║\n" +
		"║" + center("Compile-Time Code Generation for Go", boxInnerWidth) + "║\n" +
		"╚" + strings.Repeat("═", boxInnerWidth) + "╝\n"

	body := `

  Replace placeholder comments with computed values at build time.

INSTALL
	go install github.com/AeonDave/goahead@latest

USAGE
	Toolexec (recommended):    go build -toolexec="goahead" ./...
	Standalone:                goahead -dir=./mypackage

QUICK START
	1. Create a helper file (helpers.go):
		//go:build exclude
		//go:ahead functions
		package helpers
		func welcome(name string) string { return "Hello, " + name }

	2. Use placeholders in your code (main.go):
		package main
		//:welcome:"gopher"
		var greeting = ""

	3. Build with goahead:
		go build -toolexec="goahead" ./...

	Result: greeting becomes "Hello, gopher"

OPTIONS
	-dir <path>    Directory to process (default: current)
	-verbose       Enable verbose output
	-help          Show this help
	-version       Show version

ENVIRONMENT
	GOAHEAD_VERBOSE=1    Enable verbose output in toolexec mode

DOCUMENTATION
	https://github.com/AeonDave/goahead

`

	// The raw string above is indented in source code; convert leading tabs into
	// spaces so the CLI output is aligned consistently across terminals.
	body = strings.NewReplacer(
		"\n\t\t", "\n    ",
		"\n\t", "\n  ",
	).Replace(body)

	fmt.Print("\n" + header + body)
}
