.PHONY: all build build-node build-miner build-desktop clean test dev

VERSION := 0.1.0
BUILD_DIR := ./bin

all: build

# Build all components
build: build-node build-miner

# Build the AI node server
build-node:
	@echo "Building lux-ai-node..."
	go build -o $(BUILD_DIR)/lux-ai-node ./cmd/lux-ai-node

# Build the AI miner
build-miner:
	@echo "Building lux-ai-miner..."
	go build -o $(BUILD_DIR)/lux-ai-miner ./cmd/lux-ai-miner

# Build the desktop app
build-desktop:
	@echo "Building lux-ai-desktop..."
	cd desktop && pnpm install && pnpm tauri:build

# Development mode for desktop
dev-desktop:
	cd desktop && pnpm install && pnpm tauri:dev

# Run the AI node server
run-node: build-node
	$(BUILD_DIR)/lux-ai-node -port 9090

# Run the AI miner
run-miner: build-miner
	$(BUILD_DIR)/lux-ai-miner -wallet 0x0000000000000000000000000000000000000000

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
