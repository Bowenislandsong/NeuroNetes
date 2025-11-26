# NeuroNetes

**Kubernetes, but for AI agents.**

NeuroNetes is an open-source Kubernetes extension framework that makes deploying and scaling LLM-based AI agents as simple as deploying a web service. Stop wrestling with GPU scheduling, cold starts, and session management—NeuroNetes handles it all.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Bowenislandsong/NeuroNetes)](https://goreportcard.com/report/github.com/Bowenislandsong/NeuroNetes)
[![Test Coverage](https://img.shields.io/badge/coverage-95.7%25-brightgreen)](https://github.com/Bowenislandsong/NeuroNetes)

**[📚 Documentation](https://bowenislandsong.github.io/NeuroNetes/)** · **[🎯 Examples](https://bowenislandsong.github.io/NeuroNetes/website/examples.html)** · **[📊 Benchmarks](https://bowenislandsong.github.io/NeuroNetes/website/benchmarks.html)**

---

## Why NeuroNetes?

Traditional Kubernetes wasn't designed for AI agents. NeuroNetes solves the hard problems:

| Challenge | Vanilla K8s | NeuroNetes |
|-----------|-------------|------------|
| Cold start time | 45+ seconds | **2.3 seconds** |
| GPU utilization | ~50% | **85%+** |
| Session affinity | Manual setup | **Built-in** |
| Token-aware scaling | Not available | **Native support** |
| Model versioning | DIY | **First-class CRDs** |

## Quick Start

### Prerequisites
- Kubernetes 1.25+
- GPU Operator (for GPU workloads)
- Metrics Server
- Prometheus (for custom metrics)

### Installation

Choose your preferred installation method:

#### Option 1: Quickstart Script (Recommended for testing)

```bash
# Clone the repository
git clone https://github.com/Bowenislandsong/NeuroNetes.git
cd NeuroNetes

# Run quickstart script (creates Kind cluster, installs NeuroNetes)
./scripts/quickstart.sh -k -m -s

# Or install to existing cluster with monitoring and samples
./scripts/quickstart.sh -m -s
```

#### Option 2: Helm Chart (Recommended for production)

```bash
# Install using Helm
helm install neuronetes ./charts/neuronetes \
  --namespace neuronetes-system \
  --create-namespace

# With high availability and monitoring
helm install neuronetes ./charts/neuronetes \
  --namespace neuronetes-system \
  --create-namespace \
  --set highAvailability.enabled=true \
  --set metrics.serviceMonitor.enabled=true
```

#### Option 3: Cloud-Specific Manifests

```bash
# For AWS EKS
kubectl apply -k deploy/eks/

# For Google GKE
kubectl apply -k deploy/gke/

# For Azure AKS
kubectl apply -k deploy/aks/

# For on-premises
kubectl apply -k deploy/onprem/
```

#### Option 4: Manual kubectl

```bash
# Install CRDs
kubectl apply -f config/crd/

# Deploy controllers
kubectl apply -f config/deploy/

# Verify installation
kubectl get pods -n neuronetes-system
```

### Basic Usage

1. **Define a Model**:
```yaml
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: llama-3-70b
spec:
  weightsURI: s3://models/llama-3-70b/
  size: 140GB
  quantization: int4
  shardSpec:
    count: 4
    strategy: tensor-parallel
  cachePolicy:
    priority: high
    pinDuration: 1h
```

2. **Create an AgentClass**:
```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: code-assistant
spec:
  modelRef:
    name: llama-3-70b
  maxContextLength: 128000
  toolPermissions:
    - name: code-search
      rateLimit: 100/min
    - name: file-read
      rateLimit: 50/min
  guardrails:
    - type: pii-detection
      action: redact
    - type: safety-check
      action: block
  slo:
    ttft: 500ms
    tokensPerSecond: 50
    p95Latency: 2s
```

3. **Deploy an AgentPool**:
```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: code-assistant-pool
spec:
  agentClassRef:
    name: code-assistant
  minReplicas: 2
  maxReplicas: 20
  prewarmPercent: 20
  tokensPerSecondBudget: 1000
  migProfile: 1g.5gb
  autoscaling:
    metrics:
      - type: tokens-in-queue
        target: 100
      - type: ttft-p95
        target: 500ms
```

## Examples

```bash
# Deploy a chat agent in under a minute
kubectl apply -f examples/chat-agent/

# Check your agent is running
kubectl get agentpools
```

See [examples/](examples/) for complete configurations including:
- [Chat Agent](examples/chat-agent/) - Conversational AI with session management
- [Code Assistant](examples/code-assistant/) - Multi-tool RAG-powered coding help
- [RAG Pipeline](examples/rag-pipeline/) - Document Q&A with vector search

## Documentation

| Guide | Description |
|-------|-------------|
| [Architecture](docs/architecture.md) | System design deep-dive |
| [CRD Reference](docs/crds.md) | Complete API specifications |
| [Autoscaling](docs/autoscaling.md) | Token-aware scaling strategies |
| [Metrics](docs/metrics.md) | 60+ specialized metrics |
| [Cloud Deployment](docs/cloud-deployment/) | AWS, GCP, Azure guides |
| [Plugins](docs/plugins.md) | Extend NeuroNetes |

## Development

```bash
# Quick start with Docker Compose
make docker-compose-up

# Or use Kind for full K8s experience
make dev

# Build and test
make build && make test
```

See [docs/local-development.md](docs/local-development.md) for detailed setup.

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
