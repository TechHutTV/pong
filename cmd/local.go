package cmd

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Host represents a discovered network host
type Host struct {
	IP       string
	Hostname string
	Status   string
}

// RunLocal executes the local network scan command
func RunLocal(args []string) {
	fs := flag.NewFlagSet("local", flag.ExitOnError)
	timeout := fs.Int("t", 1000, "Timeout in milliseconds per host")
	workers := fs.Int("w", 100, "Number of concurrent workers")
	showHelp := fs.Bool("h", false, "Show help for this command")

	fs.Usage = func() {
		fmt.Println(`Usage: pong local [options]

Scan the local subnet to discover other machines on the network.

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Examples:
  pong local            Scan local network with default settings
  pong local -t 500     Scan with 500ms timeout per host
  pong local -w 50      Scan with 50 concurrent workers`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *showHelp {
		fs.Usage()
		return
	}

	// Get local network interface information
	localIP, ipNet, err := getLocalNetwork()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting local network: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning local network: %s\n", ipNet.String())
	fmt.Printf("Local IP: %s\n", localIP)
	fmt.Println()

	// Generate list of IPs to scan
	ips := generateIPRange(ipNet)
	if len(ips) == 0 {
		fmt.Fprintln(os.Stderr, "No IP addresses to scan")
		os.Exit(1)
	}

	if len(ips) > 1024 {
		fmt.Printf("Scanning %d hosts (this may take a while)...\n\n", len(ips))
	}

	// Scan the network
	hosts := scanNetwork(ips, time.Duration(*timeout)*time.Millisecond, *workers)

	// Display results
	displayResults(hosts, localIP)
}

// getLocalNetwork returns the local IP address and network
func getLocalNetwork() (string, *net.IPNet, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6 and loopback addresses
			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}

			return ip4.String(), &net.IPNet{IP: ip4.Mask(ipNet.Mask), Mask: ipNet.Mask}, nil
		}
	}

	return "", nil, fmt.Errorf("no suitable network interface found")
}

// generateIPRange generates all IP addresses in the given network
func generateIPRange(ipNet *net.IPNet) []string {
	var ips []string

	ip := ipNet.IP.To4()
	if ip == nil {
		return ips
	}

	mask := ipNet.Mask
	ones, bits := mask.Size()
	hostBits := bits - ones

	// For very small subnets (/30, /31, /32), assume a /24 network instead
	// This handles point-to-point links and other edge cases
	if hostBits < 4 {
		hostBits = 8 // Assume /24
		// Recalculate network address for /24
		ip = net.IPv4(ip[0], ip[1], ip[2], 0).To4()
	}

	// Limit scan to /16 networks maximum (65534 hosts)
	if hostBits > 16 {
		hostBits = 16
	}

	numHosts := (1 << hostBits) - 2 // Exclude network and broadcast addresses
	if numHosts <= 0 {
		numHosts = 254 // Fallback to /24
	}

	// Start from the network address + 1
	start := ipToUint32(ip)
	for i := 1; i <= numHosts; i++ {
		ips = append(ips, uint32ToIP(start+uint32(i)))
	}

	return ips
}

// ipToUint32 converts an IP address to uint32
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// uint32ToIP converts uint32 to IP address string
func uint32ToIP(n uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", n>>24, (n>>16)&0xFF, (n>>8)&0xFF, n&0xFF)
}

// scanNetwork scans the given IP addresses concurrently
func scanNetwork(ips []string, timeout time.Duration, workers int) []Host {
	var (
		hosts   []Host
		hostsMu sync.Mutex
		wg      sync.WaitGroup
	)

	// Create a buffered channel for work distribution
	ipChan := make(chan string, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				if isHostAlive(ip, timeout) {
					hostname := resolveHostname(ip, timeout)
					hostsMu.Lock()
					hosts = append(hosts, Host{
						IP:       ip,
						Hostname: hostname,
						Status:   "Online",
					})
					hostsMu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	return hosts
}

// isHostAlive checks if a host is reachable
func isHostAlive(ip string, timeout time.Duration) bool {
	// Try common ports to check if host is alive
	ports := []string{"80", "443", "22", "445", "139", "21", "23", "25", "53", "8080"}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan bool, len(ports))

	for _, port := range ports {
		go func(p string) {
			dialer := net.Dialer{Timeout: timeout}
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, p))
			if err == nil {
				conn.Close()
				resultChan <- true
				return
			}
			// Connection refused means host is alive but port is closed
			if strings.Contains(err.Error(), "refused") {
				resultChan <- true
				return
			}
			resultChan <- false
		}(port)
	}

	// Wait for first success or all failures
	successCount := 0
	for i := 0; i < len(ports); i++ {
		select {
		case result := <-resultChan:
			if result {
				return true
			}
			successCount++
		case <-ctx.Done():
			return false
		}
	}

	return false
}

// resolveHostname attempts to resolve the hostname for an IP
func resolveHostname(ip string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := net.Resolver{}
	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return "-"
	}

	// Clean up the hostname (remove trailing dot)
	hostname := strings.TrimSuffix(names[0], ".")
	return hostname
}

// displayResults prints the scan results in a formatted table
func displayResults(hosts []Host, localIP string) {
	if len(hosts) == 0 {
		fmt.Println("No hosts found on the network.")
		return
	}

	// Sort hosts by IP address
	sort.Slice(hosts, func(i, j int) bool {
		return ipToUint32(net.ParseIP(hosts[i].IP)) < ipToUint32(net.ParseIP(hosts[j].IP))
	})

	// Calculate column widths
	ipWidth := 15
	hostnameWidth := 30
	statusWidth := 10

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s\n", ipWidth, "IP Address", hostnameWidth, "Hostname", statusWidth, "Status")
	fmt.Println(strings.Repeat("─", ipWidth+hostnameWidth+statusWidth+4))

	// Print hosts
	for _, host := range hosts {
		hostname := host.Hostname
		if len(hostname) > hostnameWidth {
			hostname = hostname[:hostnameWidth-3] + "..."
		}

		// Mark local IP
		status := host.Status
		if host.IP == localIP {
			status = "Online (You)"
		}

		fmt.Printf("%-*s  %-*s  %-*s\n", ipWidth, host.IP, hostnameWidth, hostname, statusWidth, status)
	}

	fmt.Printf("\nFound %d host(s) on the network.\n", len(hosts))
}
