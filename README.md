# NeuroNetes: Agent-Native Kubernetes Framework

NeuroNetes is a comprehensive Kubernetes extension framework designed specifically for AI agent workloads, addressing the unique requirements of LLM-based applications that traditional Kubernetes wasn't optimized for.

**📚 [View Project Website](https://bowenislandsong.github.io/NeuroNetes/)** | **🎯 [See Examples](https://bowenislandsong.github.io/NeuroNetes/website/examples.html)** | **📊 [Performance Benchmarks](https://bowenislandsong.github.io/NeuroNetes/website/benchmarks.html)**

## Overview

While Kubernetes excels at managing traditional microservices ("pods + services"), agent-style, LLM-heavy workloads stress dimensions K8s was never designed to handle. NeuroNetes fills these gaps with agent-aware scheduling, token-based autoscaling, GPU-first orchestration, and conversation-level routing.

## Key Features

### 🚀 Ultra-Fast Scale-from-Zero
- TTFT (Time-To-First-Token) aware autoscaling
- Built-in warm pools for instant response
- Snapshot/restore of model weights
- Lazy container initialization

### 🎮 GPU-First Scheduling
- Topology-aware GPU bin-packing
- MIG (Multi-Instance GPU) partition orchestration
- Gang scheduling for tensor/pipeline parallel jobs
- NUMA-aware placement with memory-pressure preemption

### 🎯 Token-Aware Autoscaling
- Native metrics: tokens/sec, TTFT, queue depth, context length
- Tool-call rate monitoring
- Function-of-input-length scaling
- Session-aware concurrency management

### 💬 Session & Conversation Affinity
- Sticky routing by conversation ID
- Graceful session handoff during scaling
- State-aware load balancing
- Short-lived state management (tools, scratchpads, vector caches)

### 📦 Model Lifecycle Management
- Model as a first-class Kubernetes object (CRD)
- Versioning and quantization profiles
- Sharding plans and node-local caching
- Priority-based pin/evict strategies

### 🔄 Queue-Native Ingress
- Built-in message broker integration (NATS/Kafka/SQS)
- Autoscaling on topic lag
- Per-queue concurrency limits
- Low-latency, message-driven concurrency

### 💰 Cost/SLA-Aware Placement
- Multi-objective scheduling (SLO, $/token, carbon footprint)
- Graceful degradation to smaller models
- Spot instance integration with SLO guards
- Dynamic cost optimization

### 🛡️ Agent Governance
- Tool permission scopes and rate limits
- Per-session encrypted memory stores with TTL
- Audit trails for all tool invocations
- Policy-driven tool access control

### 📊 Token-Level Observability
- Standardized metrics: TTFT, tokens/sec, tool-call p95
- Automatic sampling and PII redaction
- Safety block tracking
- Retrieval miss monitoring

### 🌊 Streaming-First Networking
- gRPC/WebSocket streaming optimizations
- Backpressure handling
- Cancellation propagation
- Traffic shaping for long responses

### 🔒 Multi-Tenant GPU Isolation
- Per-tenant GPU cgroups
- VRAM zeroization between tenants
- MIG-based tenancy
- Bandwidth shaping across tenants

### 🧪 Prompt-Level Canary Deployments
- A/B testing on system prompts and tool sets
- Guardrail gates with quality metrics
- Rollback based on user-perceived regressions
- Intent-aware traffic splitting

### 📍 Data-Locality for Retrieval
- Co-scheduling of agent + RAG cache
- Node-local vector shards
- Anti-affinity for replica-shard pairs
- Embedding store locality awareness

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Control Plane                             │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │   Model    │  │ AgentClass │  │ AgentPool  │  CRDs      │
│  │ Controller │  │ Controller │  │ Controller │            │
│  └────────────┘  └────────────┘  └────────────┘            │
│                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │   Token    │  │    GPU     │  │ Cost/SLA   │ Schedulers │
│  │   HPA      │  │ Topology   │  │  Policy    │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Data Plane                              │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │  Sticky    │  │  Stream    │  │  Message   │            │
│  │  Session   │  │  Aware     │  │  Broker    │            │
│  │  Router    │  │  Ingress   │  │  Class     │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Runtime Layer                             │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │  Warm Pool │  │ Snapshot/  │  │  Sidecar   │            │
│  │ Controller │  │  Restore   │  │  Caches    │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│              Observability & Policy                          │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │   Token    │  │  Prompt    │  │ Guardrail  │            │
│  │  Metrics   │  │  Canary    │  │ Admission  │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

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

## Documentation

- [Architecture Guide](docs/architecture.md) - Deep dive into system design
- [CRD Reference](docs/crds.md) - Complete CRD specifications
- [Scheduler Guide](docs/scheduler.md) - GPU and token-aware scheduling
- [Autoscaling Guide](docs/autoscaling.md) - Token-based autoscaling strategies
- [Observability Guide](docs/observability.md) - Metrics, logging, and tracing
- [Metrics Guide](docs/metrics.md) - 60+ agent-native metrics with Prometheus & Grafana
- [Security Guide](docs/security.md) - Multi-tenancy and isolation
- [Operations Guide](docs/operations.md) - Deployment and maintenance
- [Local Development Guide](docs/local-development.md) - Docker & Kind setup
- [Plugin Guide](docs/plugins.md) - Creating custom algorithms

### Cloud Deployment

- [AWS Deployment Guide](docs/cloud-deployment/aws.md) - Deploy on Amazon EKS
- [GCP Deployment Guide](docs/cloud-deployment/gcp.md) - Deploy on Google GKE
- [Azure Deployment Guide](docs/cloud-deployment/azure.md) - Deploy on Microsoft AKS
- [Cloud Deployment Overview](docs/cloud-deployment/README.md) - Multi-cloud strategies

## Metrics & Observability

NeuroNetes provides **60+ specialized metrics** designed for LLM agent workloads:

### Key Metric Categories

- **UX & Quality**: TTFT P95, latency, RTF ratio, CSAT scores
- **Token Economics**: Tokens/sec, cost per 1K tokens, context length
- **GPU Efficiency**: GPU utilization, VRAM usage, MIG slice usage
- **Tool Performance**: Tool call latency, success rate, RAG retrieval time
- **Cost & Carbon**: $/session, spot savings, energy per 1K tokens

### Ready-to-Use Dashboards

```bash
# Import Grafana dashboard
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring

# Access at http://localhost:3000/d/neuronetes-agents
```

**Key Panels**:
- Time to First Token (TTFT) with SLO alerts
- Active sessions and queue depth
- GPU utilization and VRAM usage
- Cost per 1K tokens by model
- Tool call latency P95

See [Metrics Guide](docs/metrics.md) for complete documentation.

## Examples

**🌐 [View Complete Examples with Results](https://bowenislandsong.github.io/NeuroNetes/website/examples.html)**

- [Simple Chat Agent](examples/chat-agent/) - Basic conversational agent ([Performance Results](examples/chat-agent/RESULTS.md))
- [Code Assistant](examples/code-assistant/) - Multi-tool code helper ([Performance Results](examples/code-assistant/RESULTS.md))
- [RAG Pipeline](examples/rag-pipeline/) - Retrieval-augmented generation ([Performance Results](examples/rag-pipeline/RESULTS.md))

Each example includes:
- ✅ Complete YAML configurations
- ✅ Expected performance metrics (TTFT, throughput, cost)
- ✅ Sample outputs and quality metrics
- ✅ Cost analysis and optimization tips
- ✅ Comparison with alternatives

## Development

### Quick Start with Docker Compose

```bash
# Start local services (Redis, NATS, Weaviate, Prometheus, Grafana)
make docker-compose-up

# Access services:
# - Grafana: http://localhost:3000 (admin/admin)
# - Prometheus: http://localhost:9090
# - Weaviate: http://localhost:8080
# - NeuroNetes: http://localhost:8081

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
```

### Local Kubernetes with Kind

```bash
# Create local cluster with 3 nodes
make dev

# Deploy example resources
kubectl apply -f config/samples/

# Check status
kubectl get models,agentclasses,agentpools

# Clean up
make dev-clean
```

### Building from Source

```bash
# Build all components
make build

# Run all tests
make test-all

# Run specific test suites
make test              # Unit tests
make test-metrics      # Metrics tests (95.7% coverage)
make test-plugins      # Plugin framework tests
make test-scheduler    # Scheduler tests
make test-autoscaler   # Autoscaler tests
make test-integration  # Integration tests
make test-e2e          # End-to-end tests
```

### Continuous Integration

All code changes are automatically tested via GitHub Actions:

**CI Pipeline** (`.github/workflows/ci.yml`):
- ✅ Lint with golangci-lint
- ✅ Unit tests with race detector
- ✅ Integration tests with docker-compose
- ✅ E2E tests with Kind cluster
- ✅ Component-specific tests (metrics, plugins, scheduler, autoscaler)
- ✅ Docker build
- ✅ Code coverage upload to Codecov

**Deployment Tests** (`.github/workflows/deployment.yml`):
- ✅ Kind cluster deployment
- ✅ Docker Compose stack validation
- ✅ Monitoring stack integration

See test results on [GitHub Actions](https://github.com/Bowenislandsong/NeuroNetes/actions).

### Developing Custom Plugins

NeuroNetes provides a plugin framework for custom algorithms:

```go
// Create custom scheduler plugin
type MyScheduler struct{}

func (s *MyScheduler) Filter(ctx context.Context, pod *corev1.Pod, 
    node *corev1.Node, pool *neuronetes.AgentPool) bool {
    // Custom filtering logic
    return true
}

func (s *MyScheduler) Score(ctx context.Context, pod *corev1.Pod, 
    node *corev1.Node, pool *neuronetes.AgentPool) int64 {
    // Custom scoring logic
    return 80
}

// Register plugin
func init() {
    plugins.RegisterScheduler(&MyScheduler{})
}
```

See [Plugin Guide](docs/plugins.md) for details on creating:
- Scheduler plugins
- Autoscaler plugins
- Model loader plugins
- Guardrail plugins
- Metrics provider plugins

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Community

- GitHub Issues: Bug reports and feature requests
- Discussions: Architecture and design discussions
- Slack: Real-time community support (coming soon)

## Roadmap

### Q1 2024
- [x] Core CRD definitions
- [x] Basic controllers implementation
- [x] Token-aware HPA
- [ ] GPU topology scheduler

### Q2 2024
- [ ] Streaming ingress controller
- [ ] Warm pool optimization
- [ ] Cost/SLA policy engine
- [ ] Prompt-level canaries

### Q3 2024
- [ ] Multi-cloud support
- [ ] Advanced RAG locality
- [ ] Enterprise security features
- [ ] Performance benchmarks

### Q4 2024
- [ ] Ecosystem integrations
- [ ] Production hardening
- [ ] GA release
- [ ] Certification program
