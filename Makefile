# Pong CLI Makefile
# Build and manage the pong CLI application

BINARY_NAME := pong
MODULE := github.com/TechHutTV/pong
VERSION := $(shell grep 'const version' main.go | cut -d'"' -f2)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet
GOFMT := gofmt

# Build directories
BUILD_DIR := build
DIST_DIR := dist

# Build flags
LDFLAGS := -s -w

# Installation directory
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin

.PHONY: all build clean test install uninstall fmt vet lint run help tidy cross-compile

# Default target
all: build

# Build the application
build:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Build with debug symbols
build-debug:
	$(GOBUILD) -o $(BINARY_NAME) .

# Run the application
run: build
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Install the binary to system
install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

# Uninstall the binary from system
uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

# Format code
fmt:
	$(GOFMT) -s -w .

# Check code formatting
fmt-check:
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

# Run go vet
vet:
	$(GOVET) ./...

# Tidy go modules
tidy:
	$(GOMOD) tidy

# Download dependencies
deps:
	$(GOMOD) download

# Cross-compile for multiple platforms
cross-compile: clean
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=freebsd GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-freebsd-amd64 .

# Show help
help:
	@echo "Pong CLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the application"
	@echo "  make build        Build the application"
	@echo "  make build-debug  Build with debug symbols"
	@echo "  make run          Build and run the application"
	@echo "  make clean        Remove build artifacts"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make install      Install binary to $(BINDIR)"
	@echo "  make uninstall    Remove binary from $(BINDIR)"
	@echo "  make fmt          Format source code"
	@echo "  make fmt-check    Check code formatting"
	@echo "  make vet          Run go vet"
	@echo "  make tidy         Tidy go modules"
	@echo "  make deps         Download dependencies"
	@echo "  make cross-compile Build for multiple platforms"
	@echo "  make help         Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX=$(PREFIX)  Installation prefix"
	@echo "  DESTDIR=$(DESTDIR) Staging directory for packaging"
