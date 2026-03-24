BINARY_NAME=gnmi-collector
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build build-linux test lint clean

# Build for current platform
build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/gnmi-collector/

# Cross-compile for Linux (NX-OS switch)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINARY_NAME) ./cmd/gnmi-collector/

# Run all tests
test:
	go test -v -count=1 ./...

# Run tests with coverage
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) coverage.out
