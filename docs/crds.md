# CRD Reference

Complete reference for all NeuroNetes Custom Resource Definitions.

## Model

Represents an LLM model with weights, configuration, and caching policy.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `weightsURI` | string | Yes | URI to model weights (s3://, gs://, https://) |
| `size` | Quantity | Yes | Total size of model weights |
| `quantization` | enum | No | Quantization format: fp32, fp16, int8, int4, none |
| `shardSpec` | ShardSpec | No | Model sharding configuration |
| `cachePolicy` | CachePolicy | No | Caching behavior |
| `format` | string | No | Model format (safetensors, pytorch, gguf) |
| `architecture` | string | No | Model architecture (llama, gpt, etc.) |
| `parameterCount` | string | No | Number of parameters (e.g., "70B") |

### ShardSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `count` | int32 | Yes | Number of shards (min: 1) |
| `strategy` | enum | Yes | tensor-parallel, pipeline-parallel, data-parallel |
| `topology` | TopologyRequirement | No | GPU topology constraints |

### TopologyRequirement

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `locality` | enum | Yes | same-node, same-socket, nvlink, any |
| `minBandwidth` | Quantity | No | Minimum inter-GPU bandwidth |

### CachePolicy

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `priority` | enum | Yes | critical, high, medium, low |
| `pinDuration` | Duration | No | How long to pin in cache |
| `preloadNodes` | []string | No | Node selectors for preloading |
| `evictionPolicy` | enum | No | never, idle, low-priority |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | enum | Pending, Loading, Ready, Failed |
| `cachedNodes` | []NodeCacheStatus | Nodes where model is cached |
| `loadTime` | Duration | Time taken to load model |
| `lastUsed` | Time | Last usage timestamp |
| `conditions` | []Condition | Status conditions |
| `version` | string | Model version |

### Example

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: llama-3-70b
spec:
  weightsURI: s3://models/llama-3-70b/
  size: 140Gi
  quantization: int4
  architecture: llama
  parameterCount: "70B"
  shardSpec:
    count: 4
    strategy: tensor-parallel
    topology:
      locality: same-node
      minBandwidth: 600Gi
  cachePolicy:
    priority: high
    pinDuration: 2h
    evictionPolicy: idle
```

## AgentClass

Defines an agent configuration including model, tools, guardrails, and SLOs.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `modelRef` | ModelReference | Yes | Reference to Model resource |
| `maxContextLength` | int32 | No | Maximum context window in tokens |
| `toolPermissions` | []ToolPermission | No | Allowed tools and limits |
| `guardrails` | []Guardrail | No | Safety and policy checks |
| `slo` | ServiceLevelObjective | No | Performance targets |
| `systemPrompt` | string | No | Default system prompt |
| `temperature` | float32 | No | Generation randomness (0.0-2.0) |
| `maxTokens` | int32 | No | Maximum output tokens |
| `memoryConfig` | MemoryConfig | No | Memory/state configuration |

### ModelReference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Model name |
| `namespace` | string | No | Model namespace (defaults to same) |

### ToolPermission

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Tool identifier |
| `rateLimit` | string | No | Rate limit (e.g., "100/min") |
| `timeout` | Duration | No | Maximum execution time |
| `maxConcurrency` | int32 | No | Max concurrent invocations |
| `requiredScopes` | []string | No | Required permission scopes |

### Guardrail

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | enum | Yes | pii-detection, safety-check, content-filter, jailbreak-detection, prompt-injection |
| `action` | enum | Yes | block, redact, warn, log |
| `config` | map[string]string | No | Guardrail-specific config |
| `threshold` | float32 | No | Confidence threshold (0.0-1.0) |

### ServiceLevelObjective

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttft` | Duration | No | Target time-to-first-token |
| `tokensPerSecond` | int32 | No | Target throughput |
| `p95Latency` | Duration | No | Target P95 latency |
| `maxCostPerRequest` | float32 | No | Max cost in USD |
| `availabilityPercent` | float32 | No | Target availability (e.g., 99.9) |

### MemoryConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | enum | Yes | ephemeral, redis, memcached, postgres |
| `ttl` | Duration | No | Time-to-live for entries |
| `maxSize` | int32 | No | Maximum memory size per session |
| `encrypted` | bool | No | Encrypt memory at rest |
| `connectionString` | string | No | Connection string for external backend |

### Example

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: code-assistant
spec:
  modelRef:
    name: codellama-34b
  maxContextLength: 100000
  toolPermissions:
    - name: code_search
      rateLimit: "100/min"
      timeout: 10s
    - name: file_read
      rateLimit: "50/min"
  guardrails:
    - type: pii-detection
      action: redact
      threshold: 0.8
  slo:
    ttft: 500ms
    tokensPerSecond: 50
  memoryConfig:
    type: redis
    ttl: 2h
    encrypted: true
```

## AgentPool

Manages a pool of agent replicas with autoscaling and scheduling.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agentClassRef` | AgentClassReference | Yes | Reference to AgentClass |
| `minReplicas` | int32 | Yes | Minimum replicas (min: 0) |
| `maxReplicas` | int32 | Yes | Maximum replicas (min: 1) |
| `prewarmPercent` | int32 | No | Warm pool size (0-100) |
| `tokensPerSecondBudget` | int32 | No | Total tokens/sec capacity |
| `migProfile` | string | No | MIG configuration (e.g., "1g.5gb") |
| `autoscaling` | AutoscalingSpec | No | Autoscaling configuration |
| `gpuRequirements` | GPURequirements | No | GPU constraints |
| `sessionAffinity` | SessionAffinityConfig | No | Sticky session config |
| `scheduling` | SchedulingConfig | No | Scheduling hints |

### AutoscalingSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `metrics` | []AutoscalingMetric | Yes | Scaling metrics |
| `behavior` | ScalingBehavior | No | Scale-up/down rates |
| `cooldownPeriod` | Duration | No | Wait time between operations |

### AutoscalingMetric

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | enum | Yes | tokens-in-queue, ttft-p95, concurrent-sessions, tokens-per-second, queue-depth, context-length, tool-call-rate |
| `target` | string | Yes | Target value |
| `averagingWindow` | Duration | No | Metric averaging period |

### GPURequirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `count` | int32 | Yes | GPUs per replica (min: 1) |
| `memory` | string | No | Minimum GPU memory |
| `type` | string | No | GPU type (e.g., "A100") |
| `topology` | TopologyRequirement | No | Topology constraints |

### SessionAffinityConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Enable session affinity |
| `keyHeader` | string | No | HTTP header for session key |
| `ttl` | Duration | No | Affinity TTL |
| `type` | enum | No | conversation-id, user-id, custom |

### SchedulingConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `priority` | int32 | No | Scheduling priority |
| `costOptimization` | CostOptimizationConfig | No | Cost settings |
| `dataLocality` | DataLocalityConfig | No | Data locality hints |
| `nodeSelector` | map[string]string | No | Node label selector |

### Example

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
  migProfile: "2g.10gb"
  autoscaling:
    metrics:
      - type: tokens-in-queue
        target: "100"
      - type: ttft-p95
        target: "500ms"
  gpuRequirements:
    count: 2
    type: "A100"
  sessionAffinity:
    enabled: true
    type: conversation-id
```

## ToolBinding

Connects an AgentPool to ingress (HTTP, queue, topic).

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agentPoolRef` | AgentPoolReference | Yes | Reference to AgentPool |
| `type` | enum | Yes | queue, topic, webhook, grpc, http |
| `queueConfig` | QueueConfig | No | Queue configuration |
| `topicConfig` | TopicConfig | No | Topic configuration |
| `httpConfig` | HTTPConfig | No | HTTP configuration |
| `concurrency` | ConcurrencyConfig | No | Concurrency limits |
| `timeouts` | TimeoutConfig | No | Timeout settings |
| `retryPolicy` | RetryPolicy | No | Retry configuration |

### QueueConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | enum | Yes | nats, kafka, sqs, rabbitmq, redis |
| `connectionString` | string | Yes | Connection details |
| `queueName` | string | Yes | Queue name |
| `autoscaleOnLag` | bool | No | Enable lag-based autoscaling |
| `maxLagThreshold` | int32 | No | Lag threshold (messages) |
| `prefetchCount` | int32 | No | Messages to prefetch |
| `ackMode` | enum | No | auto, manual, client |

### HTTPConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | HTTP path |
| `methods` | []string | No | Allowed methods |
| `rateLimitPerIP` | string | No | Rate limit per IP |
| `streamingEnabled` | bool | No | Enable streaming |
| `corsConfig` | CORSConfig | No | CORS settings |

### TimeoutConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `requestTimeout` | Duration | No | Overall timeout |
| `toolTimeout` | Duration | No | Tool invocation timeout |
| `idleTimeout` | Duration | No | Idle connection timeout |

### RetryPolicy

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `maxAttempts` | int32 | Yes | Max retry attempts (min: 0) |
| `initialBackoff` | Duration | No | Initial backoff |
| `maxBackoff` | Duration | No | Maximum backoff |
| `backoffMultiplier` | float32 | No | Backoff multiplier |
| `retryableErrors` | []string | No | Error patterns to retry |

### Example

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: code-assistant-http
spec:
  agentPoolRef:
    name: code-assistant-pool
  type: http
  httpConfig:
    path: /v1/code/completions
    methods: [POST]
    streamingEnabled: true
  concurrency:
    maxConcurrentRequests: 50
  timeouts:
    requestTimeout: 5m
  retryPolicy:
    maxAttempts: 2
```

## Common Types

### Duration

String format: `30s`, `5m`, `2h`, `1d`

### Quantity

String format: `100Mi`, `10Gi`, `1Ti`, `600Gi`

### Condition

Standard Kubernetes condition:
- `type`: string
- `status`: True, False, Unknown
- `reason`: string
- `message`: string
- `lastTransitionTime`: timestamp

## Label Selectors

NeuroNetes adds standard labels:

- `neuronetes.io/agent-class`: AgentClass name
- `neuronetes.io/pool`: AgentPool name
- `neuronetes.io/model`: Model name
- `neuronetes.io/component`: Component type

## Annotations

Standard annotations:

- `neuronetes.io/version`: Resource version
- `neuronetes.io/last-updated`: Last update time
- `neuronetes.io/managed-by`: Management source

## Validation

All CRDs include:
- OpenAPI v3 validation
- Enum constraints
- Min/max constraints
- Required field validation
- Format validation

## Status Subresources

All CRDs have status subresources:
- Status updates don't trigger reconciliation
- Separate RBAC for status
- Generation tracking
