.PHONY: all build test clean dev bundle dist check-style help

# Default target
all: build

# Build the plugin for the current platform
build: check-go-version check-node-version
	@echo "Building plugin..."
	@mkdir -p server/dist
	cd server && go build -o dist/plugin-$(GOOS)-$(GOARCH)$(GOEXE) .
	cd webapp && npm run build

# Build for all platforms
build-all: check-go-version check-node-version
	@echo "Building for all platforms..."
	@mkdir -p server/dist
	cd server && GOOS=linux GOARCH=amd64 go build -o dist/plugin-linux-amd64 .
	cd server && GOOS=darwin GOARCH=amd64 go build -o dist/plugin-darwin-amd64 .
	cd server && GOOS=windows GOARCH=amd64 go build -o dist/plugin-windows-amd64.exe .
	cd webapp && npm run build

# Run tests
test:
	@echo "Running backend tests..."
	cd server && go test -v -race ./...
	@echo "Running frontend tests..."
	cd webapp && npm test

# Run tests with coverage
test-coverage:
	@echo "Running backend tests with coverage..."
	cd server && go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Running frontend tests with coverage..."
	cd webapp && npm run test:coverage

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf server/dist
	rm -rf webapp/dist
	rm -rf dist

# Create plugin bundle (for deployment, builds linux binary)
bundle: check-go-version check-node-version
	@echo "Building and creating plugin bundle..."
	@mkdir -p server/dist
	cd server && GOOS=linux GOARCH=amd64 go build -o dist/plugin-linux-amd64 .
	cd webapp && npm run build
	@mkdir -p dist
	@rm -rf dist/bundle-tmp && mkdir -p dist/bundle-tmp/server/dist dist/bundle-tmp/webapp/dist
	@cp plugin.json dist/bundle-tmp/
	@cp server/dist/plugin-linux-amd64 dist/bundle-tmp/server/dist/
	@cp -r webapp/dist/* dist/bundle-tmp/webapp/dist/
	cd dist/bundle-tmp && tar -czf ../com.appsome.claudecode.tar.gz .
	@rm -rf dist/bundle-tmp

# Create all platform bundles
bundle-all: build-all
	@echo "Creating platform-specific bundles..."
	@mkdir -p dist
	tar -czf dist/com.appsome.claudecode-linux-amd64.tar.gz plugin.json server/dist/plugin-linux-amd64 webapp/dist
	tar -czf dist/com.appsome.claudecode-darwin-amd64.tar.gz plugin.json server/dist/plugin-darwin-amd64 webapp/dist
	tar -czf dist/com.appsome.claudecode-windows-amd64.tar.gz plugin.json server/dist/plugin-windows-amd64.exe webapp/dist

# Start development environment
dev:
	@echo "Starting development environment..."
	docker-compose up -d
	@echo "Mattermost is running at http://localhost:8065"
	@echo "Default credentials: admin@example.com / admin123"

# Stop development environment
dev-down:
	docker-compose down

# Format code
fmt:
	@echo "Formatting code..."
	cd server && go fmt ./...
	cd webapp && npm run format

# Lint code
lint:
	@echo "Linting code..."
	cd server && go vet ./...
	cd webapp && npm run lint

# Check code style
check-style: lint fmt
	@echo "Code style check complete"

# Check Go version (1.21+)
check-go-version:
	@go version | grep -qE 'go1\.(2[1-9]|[3-9][0-9])' || (echo "Go 1.21+ required" && exit 1)

# Check Node version
check-node-version:
	@node --version | grep -q 'v22' || echo "Warning: Node.js 22+ recommended"

# Help
help:
	@echo "Available targets:"
	@echo "  make build        - Build plugin for current platform"
	@echo "  make build-all    - Build plugin for all platforms"
	@echo "  make test         - Run all tests"
	@echo "  make test-coverage- Run tests with coverage"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make bundle       - Create plugin bundle"
	@echo "  make bundle-all   - Create all platform bundles"
	@echo "  make dev          - Start development environment"
	@echo "  make dev-down     - Stop development environment"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make check-style  - Check code style"
	@echo "  make help         - Show this help message"

# Platform detection
ifeq ($(OS),Windows_NT)
    GOOS := windows
    GOEXE := .exe
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        GOOS := linux
    endif
    ifeq ($(UNAME_S),Darwin)
        GOOS := darwin
    endif
endif

GOARCH ?= amd64
