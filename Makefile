# Makefile for KRR MCP Server

# Variables
BINARY_NAME=krr-mcp-server
BINARY_PATH=.
BUILD_DIR=./build
GO_VERSION=$(shell go version | awk '{print $$3}')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
VERSION=1.0.0

# Docker and K8s variables
IMAGE_NAME=krr-mcp
REGISTRY?=your-registry
FULL_IMAGE=$(REGISTRY)/$(IMAGE_NAME):$(VERSION)
NAMESPACE=krr-mcp

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT}"

# Default target
.PHONY: all
all: clean test build

# Build the MCP server binary
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${BINARY_PATH}

# Build for multiple platforms
.PHONY: build-cross
build-cross: clean
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	# Linux
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${BINARY_PATH}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${BINARY_PATH}
	# macOS
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${BINARY_PATH}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${BINARY_PATH}
	# Windows
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${BINARY_PATH}

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

# Install binary
.PHONY: install
install: build
	@echo "Installing ${BINARY_NAME}..."
	cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/

# Uninstall binary
.PHONY: uninstall
uninstall:
	@echo "Uninstalling ${BINARY_NAME}..."
	rm -f /usr/local/bin/${BINARY_NAME}

# Development server (with auto-reload using air if available)
.PHONY: dev
dev:
	@which air > /dev/null && air || go run main.go -log-level debug

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest -t $(FULL_IMAGE) .

.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image: $(FULL_IMAGE)"
	@if [ "$(REGISTRY)" = "your-registry" ]; then \
		echo "ERROR: Please set REGISTRY variable (e.g., make docker-push REGISTRY=myregistry.io)"; \
		exit 1; \
	fi
	docker push $(FULL_IMAGE)
	@echo "Docker image pushed: $(FULL_IMAGE)"

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it $(IMAGE_NAME):$(VERSION)

# Kubernetes targets
.PHONY: k8s-deploy
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	@if [ "$(REGISTRY)" = "your-registry" ]; then \
		echo "ERROR: Please set REGISTRY variable (e.g., make k8s-deploy REGISTRY=myregistry.io)"; \
		exit 1; \
	fi
	cd k8s && kustomize edit set image krr-mcp=$(FULL_IMAGE)
	kubectl apply -k k8s/
	@echo "Waiting for deployment to be ready..."
	kubectl wait --for=condition=available --timeout=60s deployment/krr-mcp -n $(NAMESPACE) || true
	@echo "Deployment complete!"

.PHONY: k8s-delete
k8s-delete:
	@echo "Deleting Kubernetes resources..."
	kubectl delete -k k8s/ || true
	@echo "Resources deleted"

.PHONY: k8s-logs
k8s-logs:
	@echo "Streaming logs from $(NAMESPACE)..."
	kubectl logs -n $(NAMESPACE) -l app=krr-mcp --tail=100 -f

.PHONY: k8s-restart
k8s-restart:
	@echo "Restarting deployment..."
	kubectl rollout restart deployment/krr-mcp -n $(NAMESPACE)
	kubectl rollout status deployment/krr-mcp -n $(NAMESPACE)

.PHONY: k8s-status
k8s-status:
	@echo "Deployment status:"
	kubectl get all -n $(NAMESPACE)
	@echo "\nPod details:"
	kubectl describe pod -n $(NAMESPACE) -l app=krr-mcp | head -50

.PHONY: k8s-port-forward
k8s-port-forward:
	@echo "Port forwarding to localhost:8080..."
	kubectl port-forward -n $(NAMESPACE) svc/krr-mcp 8080:8080

.PHONY: deploy-all
deploy-all: test docker-push k8s-deploy
	@echo "Full deployment pipeline complete!"

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
	for binary in krr-mcp-server-*; do \
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
	@echo "  build           - Build the MCP server binary"
	@echo "  build-cross     - Build for multiple platforms"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  bench           - Run benchmarks"
	@echo "  clean           - Clean build artifacts"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  tidy            - Tidy dependencies"
	@echo "  vendor          - Vendor dependencies"
	@echo "  install         - Install binary to /usr/local/bin"
	@echo "  uninstall       - Remove binary from /usr/local/bin"
	@echo "  dev             - Run development server"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-push     - Push Docker image to registry"
	@echo "  docker-run      - Run Docker container"
	@echo "  k8s-deploy      - Deploy to Kubernetes"
	@echo "  k8s-delete      - Delete Kubernetes resources"
	@echo "  k8s-logs        - Show logs from deployment"
	@echo "  k8s-restart     - Restart deployment"
	@echo "  k8s-status      - Show deployment status"
	@echo "  k8s-port-forward - Port forward service to localhost:8080"
	@echo "  deploy-all      - Run full deployment pipeline (test + push + deploy)"
	@echo "  docs            - Generate documentation"
	@echo "  audit           - Run security audit"
	@echo "  release         - Create release artifacts"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION         - Version tag (default: $(VERSION))"
	@echo "  REGISTRY        - Docker registry (default: $(REGISTRY))"
	@echo ""
	@echo "Example:"
	@echo "  make deploy-all REGISTRY=myregistry.io VERSION=v1.0.0"