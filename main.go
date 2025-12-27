package main

import (
	"fmt"
	"os"

	"github.com/TechHutTV/pong/cmd"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		cmd.ShowHelp(version)
		return
	}

	command := os.Args[1]

	switch command {
	case "help", "-h", "--help":
		cmd.ShowHelp(version)
	case "version", "-v", "--version":
		fmt.Printf("pong version %s\n", version)
	case "local":
		cmd.RunLocal(os.Args[2:])
	case "out":
		cmd.RunOut(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Run 'pong help' for usage information.")
		os.Exit(1)
	}
}
