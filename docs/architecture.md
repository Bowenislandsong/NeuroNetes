# Architecture Guide

## Overview

NeuroNetes extends Kubernetes to provide agent-native orchestration for LLM workloads. This document describes the system architecture, design decisions, and component interactions.

## System Layers

### 1. Control Plane

The control plane manages the lifecycle of agent workloads through custom controllers.

#### CRD Controllers

**Model Controller**
- Manages model artifact lifecycle
- Coordinates model caching across nodes
- Handles weight loading and eviction
- Tracks model usage statistics
- Implements cache priority and pinning policies

**AgentClass Controller**
- Validates agent configurations
- Manages tool permissions and guardrails
- Monitors SLO compliance
- Updates agent specifications dynamically

**AgentPool Controller**
- Orchestrates agent replica lifecycle
- Manages warm pools and prewarming
- Coordinates with autoscalers
- Handles rolling updates and canary deployments

**ToolBinding Controller**
- Manages queue/topic bindings
- Configures ingress routes
- Monitors binding health
- Handles connection lifecycle

#### Schedulers

**GPU Topology Scheduler**
```
┌─────────────────────────────────────────┐
│         Scheduling Pipeline             │
├─────────────────────────────────────────┤
│ 1. Filter Nodes                         │
│    - GPU availability                   │
│    - MIG partition matching             │
│    - Topology requirements              │
│    - Data locality                      │
├─────────────────────────────────────────┤
│ 2. Score Nodes                          │
│    - Cost efficiency                    │
│    - SLO headroom                       │
│    - Model cache presence               │
│    - Network topology                   │
├─────────────────────────────────────────┤
│ 3. Gang Scheduling                      │
│    - Multi-GPU coordination             │
│    - All-or-nothing placement           │
│    - NUMA awareness                     │
├─────────────────────────────────────────┤
│ 4. Binding                              │
│    - Reserve GPU resources              │
│    - Update node state                  │
│    - Trigger model loading              │
└─────────────────────────────────────────┘
```

**Token-Aware HPA**
```
Metrics Collection → Metric Evaluation → Scaling Decision
      ↓                     ↓                    ↓
- tokens_in_queue    - Compare to target  - Scale up/down
- ttft_p95           - Apply smoothing     - Respect limits
- concurrent_sessions - Check stability   - Execute action
- tokens_per_second   window
```

**Cost/SLA Policy Engine**
- Multi-objective optimization (cost, latency, carbon)
- Dynamic model selection
- Spot instance integration with SLO guards
- Graceful degradation strategies

### 2. Data Plane

The data plane handles request routing and traffic management.

#### Sticky Session Router

```
Request → Session Key Extraction → Affinity Check → Route
   ↓             ↓                      ↓              ↓
Headers    conversation_id        Hash Table    Target Pod
           user_id                Cache TTL     Fallback
           custom_key                           Load Balance
```

Features:
- Session affinity by conversation/user ID
- Graceful session handoff during scaling
- TTL-based session expiration
- Fallback to load balancing

#### Stream-Aware Ingress

```
Connection → Protocol Detection → Stream Handler → Backpressure
    ↓              ↓                    ↓              ↓
WebSocket     gRPC/HTTP2          Token Streaming  Flow Control
SSE           HTTP/1.1            Cancellation     Rate Limiting
                                  Chunking         QoS
```

Capabilities:
- Native streaming support (gRPC, WebSocket, SSE)
- Per-stream rate limiting
- Cancellation propagation
- Backpressure handling
- Traffic shaping for long responses

#### Message Broker Integration

```
Queue/Topic → Consumer Group → Agent Pool → Response Queue
     ↓             ↓                ↓             ↓
NATS          Partition        Processing    Acknowledgment
Kafka         Assignment       Timeout       Dead Letter
SQS           Load Balance     Retry         Monitoring
```

Features:
- Autoscaling on topic lag
- Per-queue concurrency limits
- At-least-once delivery semantics
- Dead letter queue support

### 3. Runtime Layer

The runtime layer optimizes agent execution and resource utilization.

#### Warm Pool Controller

```
Pool State → Prewarm Strategy → Instance Management → Readiness
    ↓             ↓                    ↓                 ↓
Target Size   Model Loading      Container Start   Health Check
Current Size  Weight Caching     GPU Allocation    Ready/NotReady
Scaling       Lazy Init          Memory Map        Serve Traffic
```

Optimization strategies:
- Keep N% of max replicas warm
- Predictive prewarming based on traffic patterns
- Multi-stage initialization (container → model → ready)
- Fast swap between warm and serving states

#### Snapshot/Restore

```
Running Agent → Checkpoint → Storage → Restore → Running Agent
      ↓             ↓           ↓         ↓           ↓
Memory State   CRIU/CRAC    S3/PV    Fast Load   Resume
Model Weights  Metadata     Compression Decompress Context
Context        Incremental  Dedup      Memory Map Generation
```

Techniques:
- CRIU (Checkpoint/Restore In Userspace) for process state
- Memory-mapped model weights
- Lazy loading with on-demand paging
- Copy-on-write for weight sharing
- Incremental checkpointing

#### Sidecar Caches

**KV Cache (Short-term Memory)**
```
Agent ←→ Sidecar ←→ Redis/Memcached
        ↓
    Per-session storage
    Encrypted at rest
    TTL-based expiration
    Key: conversation_id
    Value: compressed JSON
```

**Vector Cache (RAG)**
```
Agent ←→ Sidecar ←→ Vector DB Shard
        ↓
    Node-local shard
    Embedding cache
    LRU eviction
    Co-located scheduling
```

### 4. Observability & Policy Layer

#### Token-Level Metrics

Standard metrics exported:
```
# Time-To-First-Token
neuronetes_ttft_seconds{agent_class, model, route}

# Tokens per second throughput
neuronetes_tokens_per_second{agent_class, model, direction}

# Input/output token counts
neuronetes_tokens_total{agent_class, model, direction}

# Tool invocation latency
neuronetes_tool_latency_seconds{tool_name, agent_class}

# Retrieval metrics
neuronetes_retrieval_misses_total{agent_class}
neuronetes_retrieval_latency_seconds{agent_class}

# Safety blocks
neuronetes_safety_blocks_total{guardrail_type, action}

# Cost tracking
neuronetes_cost_usd{agent_class, model}
```

#### Prompt-Level Canary Controller

```
Traffic → Route Matching → Split Logic → Variants → Compare
   ↓           ↓              ↓            ↓          ↓
Request    Intent/Prompt   % to A/B   System     Quality
Headers    Classification  Canary     Prompt     Metrics
                          Rules       Tools      Rollback
```

Features:
- A/B testing on prompts and tool sets
- Intent-aware traffic splitting
- Quality metrics comparison
- Automatic rollback on regressions
- Gradual rollout with safety gates

#### Guardrail Admission Controller

```
Request → Admission → Guardrails → Decision → Response
   ↓          ↓           ↓           ↓          ↓
Payload   Webhook    PII Detect   Allow      Forward
Headers   Policy     Safety       Block      or
                     Jailbreak    Redact     Reject
```

Guardrail types:
- PII detection and redaction
- Content safety filtering
- Jailbreak detection
- Prompt injection prevention
- Rate limiting and quotas

## Data Flow

### Request Flow (HTTP)

```
1. Client Request
   ↓
2. Ingress Controller
   ↓
3. Guardrail Admission (pre-processing)
   ↓
4. Session Router (affinity check)
   ↓
5. Agent Pod
   ↓
6. Sidecar Cache (context fetch)
   ↓
7. Model Inference
   ↓
8. Tool Invocations (if any)
   ↓
9. Token Streaming
   ↓
10. Metrics Collection
    ↓
11. Sidecar Cache (state save)
    ↓
12. Response to Client
```

### Request Flow (Queue)

```
1. Message Published to Queue
   ↓
2. ToolBinding Consumer
   ↓
3. Load Balancing across Pool
   ↓
4. Agent Pod Processing
   ↓
5. Model Inference
   ↓
6. Tool Invocations
   ↓
7. Result Publishing
   ↓
8. Message Acknowledgment
   ↓
9. Metrics Collection
```

### Autoscaling Flow

```
1. Metrics Aggregator
   ↓
2. Token-Aware HPA
   ↓
3. Scaling Decision
   ↓
4. AgentPool Controller
   ↓
5. Warm Pool Activation (if available)
   ↓  OR
   Pod Creation
   ↓
6. Scheduler Selection
   ↓
7. GPU Allocation
   ↓
8. Model Loading
   ↓
9. Readiness Check
   ↓
10. Traffic Routing Update
```

## Design Decisions

### Why Custom Schedulers?

Standard Kubernetes scheduler doesn't understand:
- GPU topology and NUMA
- MIG partitions
- Model cache locality
- Multi-GPU gang scheduling
- Cost/SLO tradeoffs

Custom schedulers enable:
- Bin-packing with GPU memory awareness
- NVLINK topology optimization
- Co-location with vector stores
- Spot instance integration with SLO guards

### Why Token-Based Autoscaling?

CPU/memory-based HPA fails because:
- GPU workloads have flat CPU utilization
- Memory is pre-allocated for models
- Load is determined by tokens/sec, not RPS

Token-based metrics enable:
- True load awareness
- Queue depth management
- TTFT and latency targeting
- Context length consideration

### Why Model as CRD?

Benefits:
- Declarative model management
- Version control and rollback
- Cache coordination across nodes
- Priority-based eviction
- Automatic preloading

### Why Session Affinity?

Agent workloads need:
- Conversation state persistence
- Tool execution context
- Short-term memory
- RAG cache locality

Session affinity provides:
- Lower latency (cached context)
- Better user experience
- Reduced redundant computation
- Efficient resource usage

## Security Considerations

### Multi-Tenant GPU Isolation

Challenges:
- GPU memory residue between tenants
- Performance interference
- Side-channel attacks

Solutions:
- Per-tenant GPU cgroups
- VRAM zeroization
- MIG-based hardware isolation
- Bandwidth shaping

### Encrypted Memory

Agent memory contains:
- User conversations
- API keys and credentials
- Tool invocation results
- PII and sensitive data

Protection:
- Encryption at rest (Redis/DB)
- Encryption in transit (TLS)
- Per-session keys
- Automatic expiration (TTL)

### Guardrail Defense-in-Depth

Layers:
1. Admission controller (pre-processing)
2. Runtime guardrails (during generation)
3. Post-processing filters
4. Audit logging

## Performance Optimizations

### Cold Start Mitigation

Techniques:
- Warm pools (20-30% of max)
- Snapshot/restore (< 1s)
- Lazy model loading
- Layer-wise streaming
- Memory-mapped weights

Target: < 500ms TTFT from cold

### GPU Utilization

Optimization strategies:
- MIG partitioning for smaller models
- Multi-instance serving (vLLM, TGI)
- Continuous batching
- Speculative decoding
- KV cache sharing

Target: > 80% GPU utilization

### Network Efficiency

Optimizations:
- gRPC for internal communication
- Protobuf for serialization
- Connection pooling
- Keep-alive and multiplexing
- Strategic co-location

## Extensibility

### Custom Metrics

Add custom autoscaling metrics:
```go
type CustomMetricProvider interface {
    GetMetricValue(ctx context.Context, pool *AgentPool) (float64, error)
    GetMetricName() string
}
```

### Custom Schedulers

Extend scheduling:
```go
type CustomSchedulerPlugin interface {
    Filter(ctx context.Context, pod *Pod, node *Node) bool
    Score(ctx context.Context, pod *Pod, node *Node) int64
}
```

### Custom Guardrails

Add guardrails:
```go
type GuardrailPlugin interface {
    Check(ctx context.Context, request *Request) (*GuardrailResult, error)
    GetType() string
}
```

## Future Enhancements

### Planned Features

- Multi-region federation
- Cross-cloud model sync
- Advanced RAG co-scheduling
- LoRA adapter management
- Speculative execution
- Multi-agent orchestration

### Research Areas

- Predictive autoscaling with ML
- Dynamic batching optimization
- Advanced GPU sharing
- Carbon-aware scheduling
- Federated learning integration
