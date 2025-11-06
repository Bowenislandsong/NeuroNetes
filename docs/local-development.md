# Local Development Guide

## Quick Start with Docker Compose

The fastest way to test NeuroNetes locally is using Docker Compose:

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f neuronetes-manager

# Stop services
docker-compose down
```

This starts:
- **Redis** (port 6379) - Session state
- **NATS** (port 4222) - Message queue
- **Weaviate** (port 8080) - Vector store (for RAG)
- **Prometheus** (port 9090) - Metrics
- **Grafana** (port 3000) - Visualization (admin/admin)
- **NeuroNetes Manager** (port 8081) - Controller in mock mode

## Local Kubernetes with Kind

### Prerequisites

```bash
# Install kind
go install sigs.k8s.io/kind@latest

# Install kubectl
# See: https://kubernetes.io/docs/tasks/tools/
```

### Start Local Cluster

```bash
# Create cluster with 3 nodes
make dev

# This runs:
# - kind create cluster --config hack/kind-config.yaml
# - make install (installs CRDs)
# - make deploy (deploys controllers)
```

### Verify Installation

```bash
# Check nodes
kubectl get nodes

# Should see:
# - 1 control-plane node
# - 2 worker nodes (labeled with mock GPU)

# Check NeuroNetes system
kubectl get pods -n neuronetes-system

# Check CRDs
kubectl get crds | grep neuronetes
```

### Deploy Sample Resources

```bash
# Deploy examples
make examples

# Check resources
kubectl get models,agentclasses,agentpools,toolbindings

# View specific resource
kubectl describe model llama-3-70b
```

### Access Services

```bash
# Port forward to access services
kubectl port-forward -n default svc/agent-service 8080:8080

# In another terminal, test
curl http://localhost:8080/health
```

### Cleanup

```bash
# Delete cluster
make dev-clean

# This runs:
# - kind delete cluster --name neuronetes-dev
```

## Running Tests

### Unit Tests

```bash
# Run all unit tests
make test

# Run specific test
go test -v ./test/unit/model_test.go

# With coverage
make coverage
open coverage.html
```

### Integration Tests

```bash
# Requires cluster running
make dev

# Run integration tests
make test-integration

# Tests autoscaler, lifecycle, etc.
```

### E2E Tests

```bash
# Requires cluster running
make dev

# Run E2E tests
make test-e2e

# Tests complete workflows
```

## Development Workflow

### 1. Make Code Changes

```bash
# Edit source files
vim pkg/scheduler/gpu_topology.go

# Format code
make fmt

# Run linters
make lint
```

### 2. Run Locally

```bash
# Option A: Run controller locally (connects to cluster)
make run-local

# Option B: Build and deploy to cluster
make build
make docker-build
kind load docker-image ghcr.io/bowenislandsong/neuronetes:v0.1.0 --name neuronetes-dev
kubectl rollout restart deployment/neuronetes-controller -n neuronetes-system
```

### 3. Test Changes

```bash
# Apply test resources
kubectl apply -f config/samples/

# Watch controller logs
kubectl logs -f -n neuronetes-system deployment/neuronetes-controller

# Check resource status
kubectl get agentpools -w
```

### 4. Debug

```bash
# Get pod logs
kubectl logs -n neuronetes-system <pod-name>

# Exec into pod
kubectl exec -it -n neuronetes-system <pod-name> -- /bin/sh

# Debug with delve
dlv debug ./cmd/manager/main.go
```

## Docker Development

### Build Image

```bash
# Build
make docker-build

# Or manually
docker build -t neuronetes:dev .
```

### Run Standalone

```bash
# Run manager
docker run -it --rm \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  -e NATS_URL=nats://host.docker.internal:4222 \
  -p 8080:8080 \
  neuronetes:dev
```

### Test with Docker Compose

```bash
# Start with custom build
docker-compose up --build

# Test endpoints
curl http://localhost:8081/metrics
curl http://localhost:8081/health
```

## Troubleshooting

### Cluster Won't Start

```bash
# Check kind version
kind version

# Delete and recreate
kind delete cluster --name neuronetes-dev
make dev
```

### CRDs Not Installing

```bash
# Manual install
kubectl apply -f config/crd/

# Check CRD status
kubectl get crds -o wide
```

### Controller Crash Loop

```bash
# Check logs
kubectl logs -n neuronetes-system <pod-name>

# Check events
kubectl get events -n neuronetes-system

# Describe deployment
kubectl describe deployment -n neuronetes-system neuronetes-controller
```

### Port Conflicts

```bash
# Check used ports
lsof -i :8080
lsof -i :6379

# Change ports in docker-compose.yml or kind-config.yaml
```

## Tips & Tricks

### Fast Iteration

```bash
# Watch and auto-rebuild
while true; do
  make build && ./bin/manager
  sleep 5
done
```

### Quick Test Cycle

```bash
# One-liner for test, build, deploy
make test && make build && make docker-build && kubectl rollout restart deployment/neuronetes-controller -n neuronetes-system
```

### Mock Mode

```bash
# Run with mock mode (no real GPU operations)
export MOCK_MODE=true
make run-local
```

### Metrics Access

```bash
# Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit http://localhost:9090

# Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Visit http://localhost:3000 (admin/admin)
```

## Next Steps

- See [Development Guide](development.md) for code generation
- See [Architecture Guide](architecture.md) for system design
- See [Plugin Guide](plugins.md) for creating custom algorithms
