# Makefile for KRR MCP Server

# Variables
BINARY_NAME=krr-mcp-server
CLI_BINARY_NAME=krr-cli
BINARY_PATH=./cmd
CLI_BINARY_PATH=./cmd/krr-cli
BUILD_DIR=./build
GO_VERSION=$(shell go version | awk '{print $$3}')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
VERSION=1.0.0

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT}"

# Default target
.PHONY: all
all: clean test build-all-local

# Build the MCP server binary
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${BINARY_PATH}

# Build the CLI tool binary
.PHONY: build-cli
build-cli:
	@echo "Building ${CLI_BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME} ${CLI_BINARY_PATH}

# Build both binaries
.PHONY: build-all-local
build-all-local: build build-cli

# Build for multiple platforms
.PHONY: build-cross
build-cross: clean
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	# Linux
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${BINARY_PATH}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${BINARY_PATH}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME}-linux-amd64 ${CLI_BINARY_PATH}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME}-linux-arm64 ${CLI_BINARY_PATH}
	# macOS
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${BINARY_PATH}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${BINARY_PATH}
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME}-darwin-amd64 ${CLI_BINARY_PATH}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME}-darwin-arm64 ${CLI_BINARY_PATH}
	# Windows
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BINARY_PATH}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${CLI_BINARY_NAME}-windows-amd64.exe ${CLI_BINARY_PATH}

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ${BUILD_DIR}
	rm -f coverage.out coverage.html

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Vendor dependencies
.PHONY: vendor
vendor:
	@echo "Vendoring dependencies..."
	go mod vendor

# Install binaries
.PHONY: install
install: build-all-local
	@echo "Installing ${BINARY_NAME} and ${CLI_BINARY_NAME}..."
	cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	cp ${BUILD_DIR}/${CLI_BINARY_NAME} /usr/local/bin/

# Install just the CLI tool
.PHONY: install-cli
install-cli: build-cli
	@echo "Installing ${CLI_BINARY_NAME}..."
	cp ${BUILD_DIR}/${CLI_BINARY_NAME} /usr/local/bin/

# Uninstall binaries
.PHONY: uninstall
uninstall:
	@echo "Uninstalling ${BINARY_NAME} and ${CLI_BINARY_NAME}..."
	rm -f /usr/local/bin/${BINARY_NAME}
	rm -f /usr/local/bin/${CLI_BINARY_NAME}

# Development server (with auto-reload using air if available)
.PHONY: dev
dev:
	@which air > /dev/null && air || go run ${BINARY_PATH}/main.go -log-level debug

# Validate KRR installation using the CLI tool
.PHONY: validate-krr
validate-krr: build-cli
	@echo "Validating KRR installation..."
	${BUILD_DIR}/${CLI_BINARY_NAME} validate

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t krr-mcp-server:${VERSION} .

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it krr-mcp-server:${VERSION}

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	godoc -http=:6060
	@echo "Documentation server started at http://localhost:6060"

# Security audit
.PHONY: audit
audit:
	@echo "Running security audit..."
	go list -json -m all | nancy sleuth

# Generate release
.PHONY: release
release: clean test build-cross
	@echo "Creating release ${VERSION}..."
	@mkdir -p ${BUILD_DIR}/release
	cd ${BUILD_DIR} && \
	for binary in krr-mcp-server-* krr-cli-*; do \
		if [[ "$$binary" == *.exe ]]; then \
			zip "release/$${binary%.exe}.zip" "$$binary"; \
		else \
			tar czf "release/$$binary.tar.gz" "$$binary"; \
		fi; \
	done
	@echo "Release files created in ${BUILD_DIR}/release/"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the MCP server binary"
	@echo "  build-cli     - Build the CLI tool binary"
	@echo "  build-all-local - Build both binaries locally"
	@echo "  build-cross   - Build for multiple platforms"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  bench         - Run benchmarks"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  tidy          - Tidy dependencies"
	@echo "  vendor        - Vendor dependencies"
	@echo "  install       - Install both binaries to /usr/local/bin"
	@echo "  install-cli   - Install only CLI tool to /usr/local/bin"
	@echo "  uninstall     - Remove both binaries from /usr/local/bin"
	@echo "  dev           - Run development server"
	@echo "  validate-krr  - Validate KRR installation using CLI tool"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docs          - Generate documentation"
	@echo "  audit         - Run security audit"
	@echo "  release       - Create release artifacts"
	@echo "  help          - Show this help message"