# Build stage
FROM --platform=linux/amd64 golang:1.24.2-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o krr-mcp ./main.go

# Runtime stage
FROM --platform=linux/amd64 python:3.11-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Install KRR CLI from source
RUN git clone https://github.com/robusta-dev/krr /opt/krr \
    && pip install --no-cache-dir -r /opt/krr/requirements.txt \
    && echo '#!/bin/sh\npython /opt/krr/krr.py "$@"' > /usr/local/bin/krr \
    && chmod +x /usr/local/bin/krr

# Copy binary from builder
COPY --from=builder /build/krr-mcp /app/krr-mcp

# Copy default config
COPY --from=builder /build/config.example.json /app/config.json

# Create non-root user
RUN useradd -m -u 1000 mcp && chown -R mcp:mcp /app
USER mcp

# Set environment variables
ENV KRR_PATH=/usr/local/bin/krr
ENV KRR_OUTPUT_FORMAT=table

# Expose port (if needed for future HTTP/WebSocket transport)
EXPOSE 8080

# Run the MCP server
ENTRYPOINT ["/app/krr-mcp"]
