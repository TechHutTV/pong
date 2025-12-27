package cmd

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// PingResult holds the result of a single ping attempt
type PingResult struct {
	Success  bool
	Duration time.Duration
	Error    string
}

// PingStats holds statistics for the ping session
type PingStats struct {
	Sent       int
	Received   int
	Lost       int
	MinTime    time.Duration
	MaxTime    time.Duration
	TotalTime  time.Duration
	StartTime  time.Time
}

// RunOut executes the out (ping-like) command
func RunOut(args []string) {
	fs := flag.NewFlagSet("out", flag.ExitOnError)
	count := fs.Int("c", 0, "Number of pings to send (0 = unlimited)")
	timeout := fs.Int("t", 2000, "Timeout in milliseconds per ping")
	interval := fs.Float64("i", 1.0, "Interval between pings in seconds")
	port := fs.Int("p", 80, "Port to ping (default: 80)")
	showHelp := fs.Bool("h", false, "Show help for this command")
	quiet := fs.Bool("q", false, "Quiet mode - only show summary")
	ipv4Only := fs.Bool("4", false, "Force IPv4")
	ipv6Only := fs.Bool("6", false, "Force IPv6")

	fs.Usage = func() {
		fmt.Println(`Usage: pong out [options] <host>

Check connectivity to a remote host using TCP connections.
Similar to ping, but uses TCP instead of ICMP (no root required).

Examples:
  pong out google.com             Ping google.com continuously
  pong out -c 5 google.com        Send 5 pings to google.com
  pong out -p 443 google.com      Ping google.com on port 443 (HTTPS)
  pong out -t 500 192.168.1.1     Ping with 500ms timeout
  pong out -i 0.5 google.com      Ping every 0.5 seconds
  pong out -q -c 10 google.com    Quiet mode, show only summary

Options:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *showHelp {
		fs.Usage()
		return
	}

	if *ipv4Only && *ipv6Only {
		fmt.Fprintln(os.Stderr, "Error: Cannot use both -4 and -6 flags")
		os.Exit(1)
	}

	remaining := fs.Args()
	if len(remaining) < 1 {
		fmt.Fprintln(os.Stderr, "Error: No host specified")
		fmt.Fprintln(os.Stderr, "Usage: pong out [options] <host>")
		os.Exit(1)
	}

	host := remaining[0]
	timeoutDuration := time.Duration(*timeout) * time.Millisecond
	intervalDuration := time.Duration(*interval * float64(time.Second))

	// Resolve the host
	network := "ip"
	if *ipv4Only {
		network = "ip4"
	} else if *ipv6Only {
		network = "ip6"
	}

	ips, err := net.DefaultResolver.LookupIP(context.Background(), network, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving host %s: %v\n", host, err)
		os.Exit(1)
	}

	if len(ips) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No IP addresses found for %s\n", host)
		os.Exit(1)
	}

	targetIP := ips[0].String()

	// Determine if we should show the hostname
	displayHost := host
	if host != targetIP {
		displayHost = fmt.Sprintf("%s (%s)", host, targetIP)
	}

	fmt.Printf("PONG %s port %d\n", displayHost, *port)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	stats := PingStats{
		StartTime: time.Now(),
		MinTime:   time.Duration(1<<63 - 1), // Max duration
	}

	// Run the ping loop
	seq := 0
	done := false

	for !done {
		select {
		case <-sigChan:
			done = true
			fmt.Println() // New line after ^C
			continue
		default:
		}

		seq++
		result := tcpPing(targetIP, *port, timeoutDuration)
		stats.Sent++

		if result.Success {
			stats.Received++
			stats.TotalTime += result.Duration
			if result.Duration < stats.MinTime {
				stats.MinTime = result.Duration
			}
			if result.Duration > stats.MaxTime {
				stats.MaxTime = result.Duration
			}

			if !*quiet {
				fmt.Printf("Connected to %s:%d - seq=%d time=%.2fms\n",
					targetIP, *port, seq, float64(result.Duration.Microseconds())/1000)
			}
		} else {
			stats.Lost++
			if !*quiet {
				fmt.Printf("Failed to connect to %s:%d - seq=%d %s\n",
					targetIP, *port, seq, result.Error)
			}
		}

		// Check if we've reached the count limit
		if *count > 0 && seq >= *count {
			done = true
			continue
		}

		// Wait for interval (but allow interrupt)
		if !done {
			select {
			case <-sigChan:
				done = true
				fmt.Println()
			case <-time.After(intervalDuration):
			}
		}
	}

	// Print statistics
	printStats(displayHost, *port, stats)
}

// tcpPing attempts a TCP connection to the specified host and port
func tcpPing(host string, port int, timeout time.Duration) PingResult {
	address := fmt.Sprintf("%s:%d", host, port)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	duration := time.Since(start)

	if err != nil {
		errMsg := "timeout"
		if strings.Contains(err.Error(), "refused") {
			errMsg = "connection refused"
		} else if strings.Contains(err.Error(), "no route") {
			errMsg = "no route to host"
		} else if strings.Contains(err.Error(), "network is unreachable") {
			errMsg = "network unreachable"
		}
		return PingResult{
			Success: false,
			Error:   errMsg,
		}
	}

	conn.Close()
	return PingResult{
		Success:  true,
		Duration: duration,
	}
}

// printStats prints the ping statistics summary
func printStats(host string, port int, stats PingStats) {
	elapsed := time.Since(stats.StartTime)

	fmt.Println()
	fmt.Printf("--- %s:%d ping statistics ---\n", host, port)

	lossPercent := float64(0)
	if stats.Sent > 0 {
		lossPercent = float64(stats.Lost) / float64(stats.Sent) * 100
	}

	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss, time %.0fms\n",
		stats.Sent, stats.Received, lossPercent, float64(elapsed.Milliseconds()))

	if stats.Received > 0 {
		avgTime := stats.TotalTime / time.Duration(stats.Received)
		fmt.Printf("rtt min/avg/max = %.2f/%.2f/%.2f ms\n",
			float64(stats.MinTime.Microseconds())/1000,
			float64(avgTime.Microseconds())/1000,
			float64(stats.MaxTime.Microseconds())/1000)
	}
}
