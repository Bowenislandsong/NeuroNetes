# Development Guide

## Setup

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- kind (for local testing)
- controller-gen
- kustomize

### Install Tools

```bash
# Install controller-gen
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

# Install kustomize
go install sigs.k8s.io/kustomize/kustomize/v5@latest

# Install kind
go install sigs.k8s.io/kind@latest
```

## Building

### Generate Code

Before building or running tests, generate required code:

```bash
# Generate DeepCopy methods and other boilerplate
make generate

# This runs:
# controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
```

### Build Binaries

```bash
# Build all binaries
make build

# Build specific component
go build -o bin/manager ./cmd/manager/main.go
```

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# Run specific test
go test -v ./test/unit/model_test.go

# With coverage
make coverage
```

### Integration Tests

```bash
# Run integration tests (requires cluster)
make test-integration

# Skip in short mode
go test -short ./test/integration/...
```

### E2E Tests

```bash
# Run end-to-end tests
make test-e2e
```

## Code Generation

NeuroNetes uses code generation for:

1. **DeepCopy methods** - Required by Kubernetes runtime
2. **Client code** - For programmatic access
3. **CRD manifests** - Kubernetes Custom Resource Definitions
4. **RBAC manifests** - Role-based access control

### Generate All

```bash
make generate
```

### Generate Specific

```bash
# Just DeepCopy methods
controller-gen object paths="./api/..."

# Just CRDs
controller-gen crd paths="./api/..." output:crd:artifacts:config=config/crd

# Just RBAC
controller-gen rbac:roleName=manager-role paths="./..." output:rbac:artifacts:config=config/rbac
```

## Local Development

### Run Controller Locally

```bash
# Create local cluster
make dev

# Run controller against local cluster
make run-local

# Clean up
make dev-clean
```

### Debug with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug controller
dlv debug ./cmd/manager/main.go
```

## Project Structure

```
neuronetes/
├── api/
│   └── v1alpha1/          # API definitions (CRDs)
├── cmd/
│   ├── manager/           # Main controller
│   ├── scheduler/         # GPU scheduler
│   └── autoscaler/        # Autoscaler
├── config/
│   ├── crd/               # CRD manifests
│   ├── samples/           # Example resources
│   └── deploy/            # Deployment manifests
├── controllers/           # Controller implementations
├── pkg/
│   ├── scheduler/         # Scheduling logic
│   ├── autoscaler/        # Autoscaling logic
│   ├── dataplane/         # Routing, ingress
│   ├── runtime/           # Runtime management
│   ├── observability/     # Metrics, tracing
│   └── policy/            # Policy enforcement
├── docs/                  # Documentation
├── examples/              # Usage examples
├── test/
│   ├── unit/             # Unit tests
│   ├── integration/      # Integration tests
│   └── e2e/              # End-to-end tests
├── Makefile              # Build automation
└── go.mod                # Go dependencies
```

## Adding a New CRD

1. Define types in `api/v1alpha1/`:

```go
// myresource_types.go
package v1alpha1

type MyResourceSpec struct {
    // Your spec fields
}

type MyResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   MyResourceSpec `json:"spec,omitempty"`
}

func init() {
    SchemeBuilder.Register(&MyResource{}, &MyResourceList{})
}
```

2. Generate code:

```bash
make generate
```

3. Create controller in `controllers/`:

```go
// myresource_controller.go
package controllers

type MyResourceReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Your reconciliation logic
}
```

4. Add tests in `test/unit/`:

```go
// myresource_test.go
func TestMyResource(t *testing.T) {
    // Your tests
}
```

5. Add examples in `config/samples/`

6. Update documentation

## Code Standards

### Go Style

Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines:

- Use `gofmt` for formatting
- Use meaningful variable names
- Keep functions small and focused
- Add comments for exported functions
- Handle errors explicitly

### Kubebuilder Markers

Use kubebuilder markers for code generation:

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mdl
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.size`
type Model struct {
    // ...
}
```

### Testing

- Write tests for all new code
- Aim for >80% coverage
- Use table-driven tests
- Mock external dependencies
- Test error paths

Example:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "TEST",
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

### Code Comments

```go
// MyFunction does something important.
// It takes input and returns output.
// Returns an error if something goes wrong.
func MyFunction(input string) (string, error) {
    // ...
}
```

### API Documentation

Update docs when changing APIs:

```bash
# Generate API reference
make docs
```

### Examples

Add examples for new features:

```yaml
# examples/my-feature/README.md
# My Feature Example

This example demonstrates...

## Quick Start

```bash
kubectl apply -f example.yaml
```
```

## Troubleshooting

### Build Issues

```bash
# Clean and rebuild
make clean
make deps
make build
```

### Test Failures

```bash
# Run specific test with verbose output
go test -v -run TestMyFunction ./pkg/mypackage

# Check for race conditions
go test -race ./...
```

### Generated Code Issues

```bash
# Regenerate all
rm -rf api/v1alpha1/zz_generated.*.go
make generate
```

## Performance

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=.

# Memory profile
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof cpu.prof
```

### Benchmarking

```bash
# Run benchmarks
make benchmark

# Compare results
benchstat old.txt new.txt
```

## Release Process

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag
4. Build and push images
5. Generate release notes
6. Publish release

```bash
# Tag release
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

# Build images
make docker-build-push VERSION=v0.2.0
```

## Additional Resources

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Controller Runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
