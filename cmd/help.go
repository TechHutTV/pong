package cmd

import "fmt"

// ShowHelp displays the help page with available commands
func ShowHelp(version string) {
	fmt.Printf(`Pong - CLI utilities for Linux and Unix systems

Version: %s

Usage:
  pong                  Display this help page
  pong <command>        Execute a specific command
  pong <command> -h     Show help for a specific command

Available Commands:
  local                 Scan local subnet for network resources
  out                   Check connectivity to a remote host (ping-like)
  help                  Display this help page
  version               Display version information

Examples:
  pong local            Scan local network for active hosts
  pong local -t 500     Scan with 500ms timeout per host
  pong out google.com   Check connectivity to google.com
  pong out -c 5 8.8.8.8 Send 5 connection checks to 8.8.8.8

For more information about a command, run:
  pong <command> -h

`, version)
}
