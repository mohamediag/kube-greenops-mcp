# KRR MCP Server - Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the KRR MCP Server.

## Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured to access your cluster
- Docker for building the container image
- (Optional) kustomize for easier deployment

## Quick Start

### 1. Build the Docker Image

```bash
# From the project root directory
docker build -t krr-mcp:latest .

# Tag and push to your registry
docker tag krr-mcp:latest your-registry/krr-mcp:latest
docker push your-registry/krr-mcp:latest
```

### 2. Update Image Reference

Edit `k8s/kustomization.yaml` or `k8s/deployment.yaml` to use your image:

```yaml
# In kustomization.yaml
images:
  - name: krr-mcp
    newName: your-registry/krr-mcp
    newTag: latest
```

### 3. Deploy Using Kustomize

```bash
kubectl apply -k k8s/
```

Or deploy individual manifests:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## Components

### Namespace
- Creates `krr-mcp` namespace for isolation

### RBAC
- **ServiceAccount**: `krr-mcp` - Service account for the MCP server
- **ClusterRole**: `krr-mcp-reader` - Read permissions for resources across all namespaces
- **ClusterRoleBinding**: Binds the ClusterRole to the ServiceAccount

The ClusterRole grants read access to:
- Pods, nodes, namespaces, PVCs
- Deployments, StatefulSets, DaemonSets, ReplicaSets
- Jobs, CronJobs
- Metrics (pods, nodes)
- HorizontalPodAutoscalers

### ConfigMap
- Stores the MCP server configuration
- Mounted at `/app/config/config.json`
- Can be customized via environment variables

### Deployment
- Runs a single replica of the MCP server
- Resource limits: 500m CPU, 512Mi memory
- Resource requests: 100m CPU, 128Mi memory
- Security context: runs as non-root user (UID 1000)

### Service
- ClusterIP service on port 8080
- For future HTTP/WebSocket transport support

## Configuration

### Environment Variables

The following environment variables can be set in the deployment:

- `KRR_PATH`: Path to KRR CLI (default: `/usr/local/bin/krr`)
- `KRR_OUTPUT_FORMAT`: Output format (fixed to `table`)
- `KRR_LOG_LEVEL`: Log level (debug, info, warn, error)
- `KRR_TIMEOUT`: Default timeout for scans
- `KRR_STRATEGY`: Default recommendation strategy

### ConfigMap

Edit `k8s/configmap.yaml` to customize the server configuration:

```yaml
data:
  config.json: |
    {
      "krr_path": "krr",
      "default_timeout": "5m",
      "default_strategy": "simple",
      "default_output_format": "table",
      ...
    }
```

After editing, reapply:

```bash
kubectl apply -f k8s/configmap.yaml
kubectl rollout restart deployment/krr-mcp -n krr-mcp
```

## Verification

Check if the deployment is running:

```bash
# Check pods
kubectl get pods -n krr-mcp

# Check logs
kubectl logs -n krr-mcp -l app=krr-mcp

# Describe pod for details
kubectl describe pod -n krr-mcp -l app=krr-mcp
```

## Usage

The MCP server is now running inside your cluster and can access all namespaces for resource analysis.

To use it, you would typically:
1. Expose it via a LoadBalancer or Ingress (for HTTP/WebSocket transport)
2. Or use kubectl port-forward for local access
3. Or integrate with Claude Code using the MCP protocol

## Cleanup

To remove all resources:

```bash
kubectl delete -k k8s/
```

Or delete individual resources:

```bash
kubectl delete -f k8s/service.yaml
kubectl delete -f k8s/deployment.yaml
kubectl delete -f k8s/configmap.yaml
kubectl delete -f k8s/rbac.yaml
kubectl delete -f k8s/namespace.yaml
```

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod -n krr-mcp -l app=krr-mcp
kubectl logs -n krr-mcp -l app=krr-mcp
```

### KRR not found

Check if KRR is installed in the container:

```bash
kubectl exec -n krr-mcp -it deployment/krr-mcp -- krr --version
```

### Permission errors

Verify RBAC is correctly set up:

```bash
kubectl get clusterrole krr-mcp-reader -o yaml
kubectl get clusterrolebinding krr-mcp-reader -o yaml
```

### Metrics not available

Ensure metrics-server is installed in your cluster:

```bash
kubectl top nodes
kubectl top pods -A
```
