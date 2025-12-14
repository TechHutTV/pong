# Pong

A collection of CLI tools to help you use Linux and Unix systems efficiently from the command line.

## Features

- **Network Discovery** - Scan your local subnet to find other machines and resources
- **Modular Design** - Easy to extend with new commands
- **Clean Output** - Human-readable formatted output

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/TechHutTV/pong.git
cd pong

# Build the project
make

# Install system-wide (optional)
sudo make install
```

### Package Managers

*Coming soon*

## Usage

### Display Help

```bash
pong
```

Shows all available commands and usage information.

### Scan Local Network

```bash
pong local
```

Discovers machines on your local subnet and displays:
- IP addresses
- Hostnames (when resolvable)
- Connection status

**Example output:**
```
Scanning subnet 192.168.1.0/24...

IP Address       Hostname              Status
───────────────────────────────────────────────
192.168.1.1      router.local          Online
192.168.1.10     workstation.local     Online
192.168.1.15     nas.local             Online

Found 3 hosts on the network.
```

## Build Requirements

- GCC or Clang compiler
- Make
- POSIX-compliant system (Linux, macOS, BSD)

### Optional Dependencies

- `libpcap` - For advanced network scanning features

## Building

```bash
# Standard build
make

# Debug build
make debug

# Clean build artifacts
make clean
```

## Project Structure

```
pong/
├── CLAUDE.md       # Project documentation
├── LICENSE         # GPLv3 license
├── README.md       # This file
├── Makefile        # Build configuration
└── src/            # Source code
    ├── main.c      # Entry point
    ├── commands/   # Command implementations
    │   └── local.c # Network scanner
    └── utils/      # Shared utilities
```

## Contributing

Contributions are welcome! Please read the project guidelines in `CLAUDE.md` before submitting changes.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
