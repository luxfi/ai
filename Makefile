.PHONY: all build build-desktop clean test dev install fmt lint

VERSION := 0.1.0
BUILD_DIR := ./bin

all: build

# Build the lux-ai binary
build:
	@echo "Building lux-ai..."
	go build -o $(BUILD_DIR)/lux-ai ./cmd/lux-ai

# Build the desktop app
build-desktop:
	@echo "Building lux-ai-desktop..."
	cd desktop && pnpm install && pnpm tauri:build

# Development mode for desktop
dev-desktop:
	cd desktop && pnpm install && pnpm tauri:dev

# Run lux-ai
run: build
	$(BUILD_DIR)/lux-ai -port 9090

# Install dependencies
install:
	go mod tidy
	cd desktop && pnpm install

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf desktop/dist
	rm -rf desktop/src-tauri/target

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run ./...
