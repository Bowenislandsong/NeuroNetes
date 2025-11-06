# NeuroNetes: Agent-Native Kubernetes Framework

NeuroNetes is a comprehensive Kubernetes extension framework designed specifically for AI agent workloads, addressing the unique requirements of LLM-based applications that traditional Kubernetes wasn't optimized for.

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
- [Security Guide](docs/security.md) - Multi-tenancy and isolation
- [Operations Guide](docs/operations.md) - Deployment and maintenance

## Examples

- [Simple Chat Agent](examples/chat-agent/) - Basic conversational agent
- [Code Assistant](examples/code-assistant/) - Multi-tool code helper
- [RAG Pipeline](examples/rag-pipeline/) - Retrieval-augmented generation
- [Multi-Model Routing](examples/multi-model/) - Cost-optimized model selection
- [Spot Instance Integration](examples/spot-integration/) - SLO-aware spot usage

## Development

### Building from Source

```bash
# Build all components
make build

# Run tests
make test

# Run integration tests
make test-integration

# Run e2e tests
make test-e2e
```

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
