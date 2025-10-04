# Kube GreenOps MCP

MCP server for Kubernetes Resource Recommender (KRR) - analyze and optimize Kubernetes resource usage.

## What is this?

This is a Model Context Protocol (MCP) server that integrates KRR with AI assistants like Claude. It allows you to:

- Scan Kubernetes clusters for resource recommendations
- Analyze CPU and memory usage
- Get optimization suggestions for workloads
- Improve resource efficiency and reduce carbon footprint

## Prerequisites

- [KRR](https://github.com/robusta-dev/krr) installed (`pip install krr`)
- Kubernetes cluster access
- Go 1.24+ (for building from source)

## Quick Start

### 1. Build the server

```bash
make build
```

### 2. Configure

Copy the example config:

```bash
cp config.example.json config.json
```

Edit `config.json` if needed (defaults work for most cases).

### 3. Add to Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "krr": {
      "command": "/path/to/greenops-mcp",
      "args": ["--config", "/path/to/config.json"]
    }
  }
}
```

### 4. Use with Claude

Ask Claude things like:
- "What are the resource recommendations for namespace nginx?"
- "Scan my cluster for optimization opportunities"
- "Show me CPU and memory recommendations"

## Example Usage

```
User: What are the resource recommendations for nginx namespace?

Claude: [scans cluster and shows recommendations for CPU/memory]
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `krr_path` | Path to KRR binary | `krr` |
| `default_strategy` | KRR strategy (simple/advanced) | `simple` |
| `default_namespace` | Default namespace to scan | `""` (all) |
| `log_level` | Logging level | `info` |

## Development

```bash
# Build
make build

# Run locally
./build/greenops-mcp

# Build Docker image
make docker-build
```
