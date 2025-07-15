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
	return len(os.Args) >= 2 &&
		!strings.HasPrefix(os.Args[1], "-") &&
		(strings.Contains(os.Args[1], "compile") ||
			strings.Contains(os.Args[1], "link") ||
			strings.Contains(os.Args[1], "asm"))
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
	fmt.Printf(`goahead - Go code generation tool
	
	USAGE:
	    # Install goahead
	    go install github.com/AeonDave/goahead@latest
	
	    # Use with go build
	    go build -toolexec="goahead" main.go
	
	    # Use standalone
	    goahead -dir=.
	
	SETUP:
	    1. Create function files with markers:
	       //go:build exclude
	       //go:ahead functions
	
	    2. Add generation comments in your code:
	       //:functionName:arg1:arg2
	
	    3. Build with toolexec integration:
	       go build -toolexec="goahead" ./...
	
	OPTIONS:
	    -dir string     Directory to process (default ".")
	    -verbose        Enable verbose output
	    -help           Show this help
	    -version        Show version
	
	ENVIRONMENT:
	    GOAHEAD_VERBOSE=1    Enable verbose output in toolexec mode
	
	VERSION: %s
	`, internal.Version)
}
