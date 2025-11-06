# E2E Testing Guide

## Overview

NeuroNetes provides comprehensive end-to-end tests that verify complete workflows across all components.

## Running E2E Tests

### Prerequisites

1. **Running Kubernetes cluster**:
```bash
# Create local cluster with Kind
make dev
```

2. **Install NeuroNetes**:
```bash
# Install CRDs and controllers
make install
make deploy
```

### Run Tests

```bash
# Run all E2E tests (requires cluster)
make test-e2e

# Or run directly with go test
go test -v ./test/e2e/...
```

## Test Coverage

### 1. Model Lifecycle (`TestE2EModelLifecycle`)

Tests the complete lifecycle of a Model resource:
- Model creation
- Weight loading
- Status transitions (Pending → Loading → Ready)
- Cache management
- Model cleanup

### 2. Agent Pool Scaling (`TestE2EAgentPoolScaling`)

Tests autoscaling behavior:
- Pool creation
- Scaling on load increase
- Min/max replica enforcement
- Warm pool management
- Scale-down behavior

### 3. Tool Binding (`TestE2EToolBinding`)

Tests integration with external systems:
- HTTP endpoint binding
- Queue binding (NATS)
- Topic binding (Kafka)
- Connection management

### 4. Complete Workflow (`TestE2ECompleteWorkflow`)

Tests full stack deployment:
- Model → AgentClass → AgentPool → ToolBinding
- Resource relationship verification
- System stability
- End-to-end request handling

### 5. Cleanup (`TestE2ECleanup`)

Tests resource cleanup:
- Resource deletion
- Finalizer handling
- Cache eviction
- State cleanup

## Writing E2E Tests

### Test Structure

```go
func TestE2EMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test in short mode")
    }

    ctx := context.Background()
    
    // Setup Kubernetes client
    config, err := clientcmd.BuildConfigFromFlags("", "")
    if err != nil {
        t.Skipf("skipping e2e test: could not build config: %v", err)
        return
    }

    clientset, err := kubernetes.NewForConfig(config)
    require.NoError(t, err)

    // Verify cluster is accessible
    _, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    if err != nil {
        t.Skipf("skipping e2e test: cluster not accessible: %v", err)
        return
    }

    // Test logic here
}
```

### Best Practices

1. **Skip when cluster unavailable**:
```go
if err != nil {
    t.Skipf("skipping e2e test: %v", err)
    return
}
```

2. **Use subtests for organization**:
```go
t.Run("setup", func(t *testing.T) {
    // Setup code
})

t.Run("verify", func(t *testing.T) {
    // Verification code
})
```

3. **Clean up resources**:
```go
defer func() {
    // Delete test resources
    clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
}()
```

4. **Wait for conditions**:
```go
// Wait for pod to be ready
err = wait.PollImmediate(1*time.Second, 5*time.Minute, func() (bool, error) {
    pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
    if err != nil {
        return false, err
    }
    return pod.Status.Phase == corev1.PodRunning, nil
})
```

## Debugging E2E Tests

### Enable Verbose Logging

```bash
go test -v -timeout 30m ./test/e2e/...
```

### Run Specific Test

```bash
go test -v -run TestE2EModelLifecycle ./test/e2e/...
```

### Check Cluster State

```bash
# View resources
kubectl get models,agentclasses,agentpools -A

# Check events
kubectl get events -A --sort-by='.lastTimestamp'

# View logs
kubectl logs -n neuronetes-system deployment/neuronetes-controller
```

### Common Issues

**Test fails with "cluster not accessible"**:
```bash
# Check kubeconfig
kubectl cluster-info

# Recreate cluster
make dev-clean
make dev
```

**Test times out**:
```bash
# Increase timeout
go test -v -timeout 60m ./test/e2e/...

# Check controller logs
kubectl logs -f -n neuronetes-system deployment/neuronetes-controller
```

**Resources not cleaning up**:
```bash
# Manual cleanup
kubectl delete models,agentclasses,agentpools --all -A

# Check finalizers
kubectl get models -o yaml | grep finalizers -A 5
```

## CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Kind
        run: |
          curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
          chmod +x ./kind
          sudo mv ./kind /usr/local/bin/kind
      
      - name: Create cluster
        run: make dev
      
      - name: Run E2E tests
        run: make test-e2e
      
      - name: Cleanup
        if: always()
        run: make dev-clean
```

### GitLab CI

```yaml
e2e-test:
  stage: test
  image: golang:1.21
  services:
    - docker:dind
  before_script:
    - curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
    - chmod +x ./kind && mv ./kind /usr/local/bin/
  script:
    - make dev
    - make test-e2e
  after_script:
    - make dev-clean
```

## Performance Testing

### Load Tests

```bash
# Run with increased load
export E2E_LOAD_FACTOR=10
go test -v ./test/e2e/...
```

### Benchmark Tests

```bash
# Run benchmark tests
go test -bench=. ./test/e2e/...
```

## Test Environments

### Local Development

```bash
# Quick test cycle
make dev
make test-e2e
```

### Staging

```bash
# Point to staging cluster
export KUBECONFIG=~/.kube/staging-config
make test-e2e
```

### Production

E2E tests should **not** run against production. Use:
- Smoke tests
- Canary deployments
- Feature flags

## Metrics & Monitoring

### Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./test/e2e/...
go tool cover -html=coverage.out
```

### Test Duration

```bash
# Profile test duration
go test -v -cpuprofile=cpu.prof ./test/e2e/...
go tool pprof cpu.prof
```

## Additional Resources

- [Kubernetes Testing Best Practices](https://kubernetes.io/docs/tasks/test/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Kind Documentation](https://kind.sigs.k8s.io/)
