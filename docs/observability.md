# Observability Guide

## Overview

NeuroNetes provides comprehensive observability at the token and turn level, with built-in metrics, logging, and tracing tailored for agent workloads.

## Standard Metrics

### Token Metrics

#### neuronetes_ttft_seconds
Time to first token latency.

**Type**: Histogram
**Labels**:
- `agent_class`: AgentClass name
- `model`: Model name
- `route`: Request route/endpoint

**Example**:
```promql
# P95 TTFT
histogram_quantile(0.95,
  rate(neuronetes_ttft_seconds_bucket[5m])
)

# By agent class
histogram_quantile(0.95,
  rate(neuronetes_ttft_seconds_bucket{agent_class="code-assistant"}[5m])
)
```

#### neuronetes_tokens_per_second
Token generation throughput.

**Type**: Gauge
**Labels**:
- `agent_class`
- `model`
- `direction`: input/output

**Example**:
```promql
# Current output tokens/sec
neuronetes_tokens_per_second{direction="output"}

# Aggregate across all pools
sum(neuronetes_tokens_per_second{direction="output"})
```

#### neuronetes_tokens_total
Total token count.

**Type**: Counter
**Labels**:
- `agent_class`
- `model`
- `direction`: input/output

**Example**:
```promql
# Input tokens over time
rate(neuronetes_tokens_total{direction="input"}[5m])

# Total tokens processed
sum(neuronetes_tokens_total)
```

#### neuronetes_context_length_tokens
Context window utilization.

**Type**: Histogram
**Labels**:
- `agent_class`
- `model`

**Example**:
```promql
# Average context length
avg(neuronetes_context_length_tokens)

# P99 context length
histogram_quantile(0.99,
  rate(neuronetes_context_length_tokens_bucket[5m])
)
```

### Tool Metrics

#### neuronetes_tool_latency_seconds
Tool invocation latency.

**Type**: Histogram
**Labels**:
- `tool_name`
- `agent_class`
- `status`: success/failure

**Example**:
```promql
# P95 tool latency
histogram_quantile(0.95,
  rate(neuronetes_tool_latency_seconds_bucket[5m])
)

# By tool
histogram_quantile(0.95,
  rate(neuronetes_tool_latency_seconds_bucket{tool_name="code_search"}[5m])
)
```

#### neuronetes_tool_invocations_total
Total tool invocations.

**Type**: Counter
**Labels**:
- `tool_name`
- `agent_class`
- `status`

**Example**:
```promql
# Tool call rate
rate(neuronetes_tool_invocations_total[5m])

# Success rate
rate(neuronetes_tool_invocations_total{status="success"}[5m]) /
rate(neuronetes_tool_invocations_total[5m])
```

### Retrieval Metrics

#### neuronetes_retrieval_latency_seconds
RAG retrieval latency.

**Type**: Histogram
**Labels**:
- `agent_class`
- `vector_store`

**Example**:
```promql
# P95 retrieval latency
histogram_quantile(0.95,
  rate(neuronetes_retrieval_latency_seconds_bucket[5m])
)
```

#### neuronetes_retrieval_misses_total
Cache misses for retrieval.

**Type**: Counter
**Labels**:
- `agent_class`
- `cache_type`

**Example**:
```promql
# Miss rate
rate(neuronetes_retrieval_misses_total[5m]) /
rate(neuronetes_retrieval_requests_total[5m])
```

### Safety Metrics

#### neuronetes_safety_blocks_total
Safety guardrail blocks.

**Type**: Counter
**Labels**:
- `guardrail_type`: pii-detection, safety-check, etc.
- `action`: block, redact, warn
- `agent_class`

**Example**:
```promql
# Total blocks
sum(neuronetes_safety_blocks_total{action="block"})

# Block rate by type
rate(neuronetes_safety_blocks_total[5m])
```

#### neuronetes_guardrail_latency_seconds
Guardrail check latency.

**Type**: Histogram
**Labels**:
- `guardrail_type`
- `agent_class`

### Cost Metrics

#### neuronetes_cost_usd
Cost tracking in USD.

**Type**: Counter
**Labels**:
- `agent_class`
- `model`
- `cost_type`: compute, storage, network

**Example**:
```promql
# Hourly cost
rate(neuronetes_cost_usd[1h]) * 3600

# Cost by model
sum by (model) (rate(neuronetes_cost_usd[1h]) * 3600)
```

### Session Metrics

#### neuronetes_concurrent_sessions
Active conversation sessions.

**Type**: Gauge
**Labels**:
- `agent_class`
- `pool`

**Example**:
```promql
# Current concurrent sessions
neuronetes_concurrent_sessions

# Peak sessions
max_over_time(neuronetes_concurrent_sessions[1h])
```

#### neuronetes_session_duration_seconds
Session lifetime.

**Type**: Histogram
**Labels**:
- `agent_class`

**Example**:
```promql
# Average session duration
avg(neuronetes_session_duration_seconds)

# P95 duration
histogram_quantile(0.95,
  rate(neuronetes_session_duration_seconds_bucket[5m])
)
```

### Pool Metrics

#### neuronetes_agentpool_replicas
Current replica count.

**Type**: Gauge
**Labels**:
- `pool`
- `status`: ready, not_ready, warm

**Example**:
```promql
# Ready replicas
neuronetes_agentpool_replicas{status="ready"}

# Warm pool size
neuronetes_agentpool_replicas{status="warm"}
```

#### neuronetes_agentpool_scale_operations_total
Scaling operations.

**Type**: Counter
**Labels**:
- `pool`
- `direction`: up, down

**Example**:
```promql
# Scale operations per hour
sum by (pool) (
  rate(neuronetes_agentpool_scale_operations_total[1h]) * 3600
)
```

## Logging

### Structured Logging

All logs are JSON-structured:

```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "msg": "request completed",
  "agent_class": "code-assistant",
  "session_id": "sess-abc123",
  "request_id": "req-xyz789",
  "ttft_ms": 450,
  "total_tokens": 1523,
  "input_tokens": 342,
  "output_tokens": 1181,
  "tool_calls": 3,
  "duration_ms": 3420
}
```

### Log Levels

- **debug**: Detailed diagnostic information
- **info**: General informational messages
- **warn**: Warning messages (non-critical)
- **error**: Error messages (recoverable)
- **fatal**: Critical errors (unrecoverable)

### Automatic Redaction

PII is automatically redacted:

```json
{
  "msg": "processing request",
  "user_input": "My email is [REDACTED] and phone is [REDACTED]",
  "pii_detected": ["email", "phone"],
  "redaction_time_ms": 12
}
```

Configuration:
```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: secure-agent
spec:
  observability:
    logging:
      # Auto-redact PII
      redactPII: true
      
      # Patterns to redact
      redactionPatterns:
        - email
        - phone
        - ssn
        - credit_card
      
      # Sample rate (% of logs to keep)
      sampleRate: 100
```

## Tracing

### Distributed Tracing

NeuroNetes integrates with OpenTelemetry:

```
Request Span
├── Ingress Span
│   └── Auth Span
├── Router Span
│   └── Session Lookup Span
├── Agent Span
│   ├── Model Inference Span
│   │   ├── Context Fetch Span
│   │   ├── Prompt Processing Span
│   │   ├── Token Generation Span
│   │   └── Context Save Span
│   ├── Tool Call Span (code_search)
│   │   ├── Vector Search Span
│   │   └── Result Processing Span
│   └── Tool Call Span (file_read)
└── Response Span
```

### Span Attributes

Standard attributes:
- `agent_class`
- `model`
- `session_id`
- `request_id`
- `input_tokens`
- `output_tokens`
- `ttft_ms`
- `tool_calls`
- `guardrails_triggered`

Example trace:
```json
{
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "name": "agent.inference",
  "kind": "INTERNAL",
  "start_time": "2024-01-15T10:30:45.000Z",
  "end_time": "2024-01-15T10:30:48.420Z",
  "attributes": {
    "agent_class": "code-assistant",
    "model": "codellama-34b",
    "input_tokens": 342,
    "output_tokens": 1181,
    "ttft_ms": 450,
    "tool_calls": ["code_search", "file_read"],
    "cache_hits": 2
  }
}
```

### Jaeger Integration

Deploy Jaeger:
```bash
kubectl apply -f https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/main/all-in-one/jaeger-all-in-one-template.yml
```

Configure NeuroNetes:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: neuronetes-config
data:
  tracing.yaml: |
    enabled: true
    exporter: jaeger
    endpoint: jaeger-collector:14268
    sampleRate: 0.1  # 10% sampling
```

## Dashboards

### Grafana Dashboards

Pre-built dashboards are available:

#### 1. Agent Overview
- Active sessions
- Tokens/sec throughput
- TTFT P95
- Request rate
- Error rate

#### 2. Model Performance
- Inference latency
- Token generation speed
- Context utilization
- Cache hit rate

#### 3. Tool Metrics
- Tool invocation rate
- Tool latency by type
- Tool success rate
- Tool timeout rate

#### 4. Cost Analysis
- Cost per model
- Cost per agent class
- Cost trends
- Budget utilization

#### 5. Scaling Metrics
- Replica count
- Warm pool utilization
- Scaling events
- Queue depth

### Example Dashboard JSON

```json
{
  "dashboard": {
    "title": "NeuroNetes Agent Overview",
    "panels": [
      {
        "title": "TTFT P95",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(neuronetes_ttft_seconds_bucket[5m]))"
        }],
        "type": "graph"
      },
      {
        "title": "Tokens/Sec",
        "targets": [{
          "expr": "sum(neuronetes_tokens_per_second{direction='output'})"
        }],
        "type": "graph"
      },
      {
        "title": "Active Sessions",
        "targets": [{
          "expr": "neuronetes_concurrent_sessions"
        }],
        "type": "stat"
      }
    ]
  }
}
```

## Alerting

### Alert Rules

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-alerts
data:
  neuronetes.rules: |
    groups:
    - name: neuronetes
      interval: 30s
      rules:
      
      # High TTFT
      - alert: HighTTFT
        expr: |
          histogram_quantile(0.95,
            rate(neuronetes_ttft_seconds_bucket[5m])
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High time-to-first-token"
          description: "P95 TTFT is {{ $value }}s (threshold: 1s)"
      
      # Low throughput
      - alert: LowThroughput
        expr: |
          sum(neuronetes_tokens_per_second{direction="output"}) < 50
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low token throughput"
          description: "Throughput is {{ $value }} tokens/sec"
      
      # High error rate
      - alert: HighErrorRate
        expr: |
          rate(neuronetes_requests_failed_total[5m]) /
          rate(neuronetes_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate"
          description: "Error rate is {{ $value | humanizePercentage }}"
      
      # Safety blocks spike
      - alert: SafetyBlocksSpike
        expr: |
          rate(neuronetes_safety_blocks_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Spike in safety blocks"
          description: "{{ $value }} blocks/sec"
      
      # Cost budget exceeded
      - alert: BudgetExceeded
        expr: |
          sum(rate(neuronetes_cost_usd[1h]) * 3600) > 100
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Hourly cost budget exceeded"
          description: "Cost is ${{ $value }}/hour"
```

### Alert Channels

Configure notification channels:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
stringData:
  alertmanager.yml: |
    global:
      resolve_timeout: 5m
    
    route:
      group_by: ['alertname', 'agent_class']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 12h
      receiver: 'default'
      
      routes:
      - match:
          severity: critical
        receiver: 'pagerduty'
      
      - match:
          severity: warning
        receiver: 'slack'
    
    receivers:
    - name: 'default'
      webhook_configs:
      - url: 'http://webhook-service/alerts'
    
    - name: 'slack'
      slack_configs:
      - api_url: 'SLACK_WEBHOOK_URL'
        channel: '#neuronetes-alerts'
    
    - name: 'pagerduty'
      pagerduty_configs:
      - service_key: 'PAGERDUTY_KEY'
```

## Custom Metrics

### Adding Custom Metrics

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    customMetric = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "neuronetes_custom_metric_seconds",
            Help: "Custom metric description",
            Buckets: prometheus.DefBuckets,
        },
        []string{"label1", "label2"},
    )
)

func RecordCustomMetric(duration float64, labels ...string) {
    customMetric.WithLabelValues(labels...).Observe(duration)
}
```

### Exporting Custom Metrics

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-metrics
data:
  metrics.yaml: |
    metrics:
      - name: business_metric
        type: gauge
        help: "Business-specific metric"
        labels: ["tenant", "product"]
        
      - name: quality_score
        type: histogram
        help: "Generated content quality score"
        buckets: [0.1, 0.3, 0.5, 0.7, 0.9, 1.0]
```

## Best Practices

### 1. Use Labels Wisely

```yaml
# Good: Bounded cardinality
neuronetes_requests_total{agent_class="code-assistant", status="200"}

# Bad: Unbounded cardinality
neuronetes_requests_total{user_id="12345", session_id="abc..."}
```

### 2. Set Appropriate Sample Rates

```yaml
# High-traffic endpoints
sampleRate: 1  # 1%

# Low-traffic endpoints
sampleRate: 100  # 100%

# Errors
errorSampleRate: 100  # Always sample errors
```

### 3. Redact Sensitive Data

```yaml
redactPII: true
redactionPatterns:
  - email
  - phone
  - ssn
  - api_key
```

### 4. Use Structured Logging

```go
// Good
log.Info("request processed",
    "request_id", reqID,
    "duration_ms", duration,
    "tokens", tokens)

// Bad
log.Info(fmt.Sprintf("Request %s took %d ms with %d tokens",
    reqID, duration, tokens))
```

### 5. Monitor Key SLIs

Focus on:
- **Latency**: TTFT P95, P99
- **Throughput**: Tokens/sec
- **Errors**: Error rate, timeout rate
- **Saturation**: Queue depth, concurrent sessions
- **Cost**: $/hour, $/request

## Troubleshooting

### High TTFT

1. Check metrics:
```promql
histogram_quantile(0.95, rate(neuronetes_ttft_seconds_bucket[5m]))
```

2. Identify bottleneck:
```promql
# Model load time
neuronetes_model_load_seconds

# Queue wait time
neuronetes_queue_wait_seconds

# GPU availability
neuronetes_gpu_available
```

3. Solutions:
- Increase warm pool
- Add more replicas
- Use faster GPUs

### High Tool Latency

1. Check per-tool metrics:
```promql
histogram_quantile(0.95,
  rate(neuronetes_tool_latency_seconds_bucket{tool_name="code_search"}[5m])
)
```

2. Identify slow tools
3. Optimize or set timeouts

### Missing Metrics

1. Verify exporter:
```bash
kubectl port-forward svc/prometheus 9090
curl localhost:9090/api/v1/label/__name__/values | grep neuronetes
```

2. Check ServiceMonitor:
```bash
kubectl get servicemonitor -n neuronetes-system
```

3. Verify scrape config:
```yaml
kubectl get configmap prometheus-config -o yaml
```

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [OpenTelemetry](https://opentelemetry.io/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Jaeger Tracing](https://www.jaegertracing.io/docs/)
