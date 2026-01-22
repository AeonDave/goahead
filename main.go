package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AeonDave/goahead/internal"
)

func main() {
	if isToolexecMode() {
		toolexecManager := internal.NewToolexecManager()
		toolexecManager.RunAsToolexec()
		return
	}

	// Check for subcommands: goahead build, goahead run, goahead test
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "build", "run", "test":
			runGoCommandWithCodegen(os.Args[1], os.Args[2:])
			return
		}
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

// runGoCommandWithCodegen runs codegen first, then executes go build/run/test
func runGoCommandWithCodegen(command string, args []string) {
	verbose := os.Getenv("GOAHEAD_VERBOSE") == "1"

	// Determine working directory from args or use current
	workDir := "."
	for i, arg := range args {
		// Look for package path arguments (not flags)
		if !strings.HasPrefix(arg, "-") && (strings.HasPrefix(arg, "./") || arg == "." || strings.HasSuffix(arg, "...")) {
			// Extract directory from pattern like ./cmd/... or ./pkg
			dir := strings.TrimSuffix(arg, "/...")
			dir = strings.TrimSuffix(dir, "...")
			if dir == "" || dir == "." {
				dir = "."
			}
			// For patterns like ./... we want to process from current dir
			if strings.Contains(args[i], "...") {
				workDir = "."
			} else {
				workDir = dir
			}
			break
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[goahead] Running codegen in %s before 'go %s'\n", workDir, command)
	}

	// Run codegen first
	if err := internal.RunCodegen(workDir, verbose); err != nil {
		log.Fatalf("[goahead] Codegen failed: %v", err)
	}

	// Now run go command WITHOUT toolexec
	goArgs := append([]string{command}, args...)
	cmd := exec.Command("go", goArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if verbose {
		fmt.Fprintf(os.Stderr, "[goahead] Running: go %s\n", strings.Join(goArgs, " "))
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func isToolexecMode() bool {
	if len(os.Args) < 2 {
		return false
	}
	// In toolexec mode, Go passes the tool path as an argument. Scan for the first
	// non-flag argument and accept any executable that looks like a Go tool.
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if looksLikeGoTool(arg) {
			return true
		}
	}
	return false
}

func looksLikeGoTool(arg string) bool {
	if strings.Contains(arg, "go"+string(os.PathSeparator)+"pkg"+string(os.PathSeparator)+"tool") {
		return true
	}
	base := filepath.Base(arg)
	if strings.HasSuffix(strings.ToLower(base), ".exe") {
		base = base[:len(base)-4]
	}
	switch strings.ToLower(base) {
	case "compile", "link", "asm", "cgo", "pack", "buildid",
		"addr2line", "api", "cover", "dist", "doc", "fix", "nm",
		"objdump", "pprof", "test2json", "trace", "vet":
		return true
	default:
		return false
	}
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
	Subcommands (recommended for CGO):
		goahead build ./...          Process + build
		goahead run ./cmd/app        Process + run  
		goahead test ./...           Process + test

	Toolexec mode:
		go build -toolexec="goahead" ./...

	Standalone (process only):
		goahead -dir=./mypackage

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
		goahead build ./...

	Result: greeting becomes "Hello, gopher"

OPTIONS
	-dir <path>    Directory to process (default: current)
	-verbose       Enable verbose output
	-help          Show this help
	-version       Show version

ENVIRONMENT
	GOAHEAD_VERBOSE=1    Enable verbose output

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
