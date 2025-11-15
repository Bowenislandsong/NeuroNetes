# NeuroNetes Metrics Guide

Complete guide to agent-native metrics, monitoring, and observability in NeuroNetes.

## Overview

NeuroNetes provides **60+ specialized metrics** across 10 categories designed for LLM agent workloads. Unlike traditional Kubernetes metrics (CPU, memory), these focus on token-level performance, model efficiency, and agent-specific SLOs.

## Quick Start

```bash
# Deploy Prometheus rules
kubectl apply -f config/monitoring/prometheus-rules.yaml

# Import Grafana dashboard
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring

# Access dashboard at http://localhost:3000/d/neuronetes-agents
```

## Test Coverage

NeuroNetes includes comprehensive test coverage for all metrics:

- **Unit Tests**: 15 tests covering basic metric recording (95.7% coverage)
- **Integration Tests**: 13 tests covering all metric categories
- **E2E Tests**: 11 tests for Prometheus export and dashboards

```bash
# Run all metrics tests
make test-metrics

# Run with coverage report
go test -v ./pkg/metrics/... -race -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run integration tests
go test -v ./test/integration/metrics_test.go

# Run E2E tests
go test -v ./test/e2e/metrics_e2e_test.go
```

## Metric Categories

### 1. UX & Quality (SLO-Facing)

**Metrics**:
- `agent_ttft_ms` - Time to first token (histogram)
- `agent_latency_ms` - End-to-end turn latency (histogram)
- `agent_rtf_ratio` - Real-time factor (gauge)
- `agent_tokens_out_per_s` - Token generation rate (gauge)
- `agent_csat_score` - Customer satisfaction (gauge)
- `agent_thumbs_up_rate` - Positive feedback rate (gauge)
- `agent_turn_errors_total` - Turn errors (counter)
- `agent_quality_winrate` - Canary win rate (gauge)

**Time to First Token (TTFT)**:
```promql
# P50, P95, P99 latencies
histogram_quantile(0.50, rate(agent_ttft_ms_bucket[5m]))
histogram_quantile(0.95, rate(agent_ttft_ms_bucket[5m]))
histogram_quantile(0.99, rate(agent_ttft_ms_bucket[5m]))

# SLO: TTFT P95 ≤ 350ms
```

**Turn Latency**:
```promql
# End-to-end turn completion
histogram_quantile(0.95, rate(agent_latency_ms_bucket[5m]))

# SLO: Latency P95 ≤ 2.5s
```

**Quality Metrics**:
- `agent_rtf_ratio` - Real-time factor (generation time / output duration), target ≤ 1.5
- `agent_tokens_out_per_s` - Token generation rate (tokens/sec)
- `agent_csat_score` - Customer satisfaction (0-5)
- `agent_quality_winrate` - A/B test win rate (0-1)

**Testing**:
```go
// Unit test example
m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
m.RecordLatency(ctx, 2000*time.Millisecond, "llama-3-70b", "/chat")

// Verify SLO compliance
ttft := 300 * time.Millisecond
assert.Less(t, ttft, 350*time.Millisecond, "TTFT should meet SLO")
```

### 2. Load & Concurrency

**Active Load**:
```promql
# Current active sessions
agent_active_sessions

# Queue depth per route
agent_queue_depth{route="/chat"}
```

**Admission Control**:
```promql
# Rejections due to capacity/SLO
rate(agent_admission_rejects_total[5m])
```

**Scaling Performance**:
```promql
# Time from load spike to replica ready
histogram_quantile(0.95, rate(agent_scaling_lag_seconds_bucket[5m]))

# SLO: Scaling lag < 60s
```

### 3. Token & Context Dynamics

**Token Counters**:
```promql
# Total tokens by model
rate(agent_total_tokens{model="llama-3-70b"}[5m])

# Input vs output ratio
rate(agent_input_tokens_total[5m]) / rate(agent_output_tokens_total[5m])
```

**Context Management**:
```promql
# P95 context length
agent_ctx_len_p95

# Truncation rate
rate(agent_ctx_truncations_total[5m])
```

**Efficiency**:
```promql
# KV cache hit ratio
agent_kv_cache_hit_ratio

# Batch merge efficiency
agent_batch_merge_efficiency
```

### 4. Tool & RAG Performance

**Tool Calls**:
```promql
# Tool latency P95
histogram_quantile(0.95, rate(agent_tool_latency_ms_bucket{tool="code_search"}[5m]))

# Tool success rate
agent_tool_success_rate{tool="web_search"}

# SLO: Tool P95 ≤ 800ms
```

**RAG Retrieval**:
```promql
# Retrieval latency
histogram_quantile(0.95, rate(rag_retrieval_latency_ms_bucket[5m]))

# Cache hit ratio
rag_retrieval_cache_hit_ratio

# Quality metrics
rag_hit_at_k
rag_mrr
```

### 5. GPU & System Efficiency

**GPU Utilization**:
```promql
# GPU utilization by node
gpu_util_pct{node="gpu-node-1"}

# VRAM usage
gpu_vram_used_gb

# MIG slice utilization
gpu_mig_slice_util_pct{slice="3g.40gb"}
```

**Model Loading**:
```promql
# Load time distribution
histogram_quantile(0.95, rate(model_load_time_seconds_bucket[5m]))

# Cache effectiveness
model_cache_hit_ratio

# Cold start rate
agent_cold_start_rate
```

### 6. Scheduler & Placement

**Scheduling Quality**:
```promql
# Gang scheduling wait time
histogram_quantile(0.95, rate(gang_schedule_wait_seconds_bucket[5m]))

# Topology penalty (suboptimal placement)
topology_penalty_score

# Session affinity effectiveness
session_affinity_hit_ratio

# Data locality rate
data_locality_rate
```

### 7. Cost & Carbon

**Token Economics**:
```promql
# Cost per 1K tokens
cost_usd_per_1k_tokens{model="llama-3-70b",tenant="prod"}

# Session cost
cost_usd_per_session

# Hourly burn rate
sum(rate(cost_usd_per_session[1h])) * 3600
```

**Resource Consumption**:
```promql
# GPU hours
rate(gpu_hours_total[1h])

# Spot savings
increase(spot_savings_usd_total[24h])

# Energy efficiency
energy_kwh_per_1k_tokens
```

### 8. Security & Policy

**Guardrails**:
```promql
# Policy blocks (PII, safety)
rate(policy_blocks_total{type="pii-detection"}[5m])

# Redaction events
rate(redaction_events_total{field="email"}[5m])

# Authorization denials
rate(authz_denials_total[5m])
```

## Grafana Dashboards

### Import Pre-Built Dashboard

```bash
# Import NeuroNetes dashboard
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring

# Add to Grafana
kubectl label configmap neuronetes-dashboard \
  grafana_dashboard=1 \
  -n monitoring
```

Access at: `http://localhost:3000/d/neuronetes-agents`

### Key Panels

1. **TTFT P95** - Time to first token (with SLO alert)
2. **Tokens/Second** - Generation throughput
3. **Active Sessions** - Current load
4. **GPU Utilization** - Resource usage
5. **Cost per 1K Tokens** - Economics
6. **Error Rate** - Reliability
7. **Tool Call Latency** - Integration performance
8. **Model Load Time** - Warm pool effectiveness

## Prometheus Rules

NeuroNetes includes comprehensive Prometheus alerting and recording rules in `config/monitoring/prometheus-rules.yaml`.

### SLO Alerts

Deploy with:
```bash
kubectl apply -f config/monitoring/prometheus-rules.yaml
```

**Key Alerts**:

| Alert | Threshold | Severity | Description |
|-------|-----------|----------|-------------|
| `TTFTSLOBreach` | P95 > 350ms | warning | Time to first token exceeds SLO |
| `LatencySLOBreach` | P95 > 2.5s | warning | Turn latency exceeds SLO |
| `HighErrorRate` | > 1% | critical | Error rate above threshold |
| `ToolLatencySLOBreach` | P95 > 800ms | warning | Tool calls too slow |
| `HighCostPer1KTokens` | > $0.10 | warning | Cost exceeds budget |
| `LowGPUUtilization` | < 50% | info | Underutilized GPUs |
| `HighGPUUtilization` | > 95% | warning | GPU throttling risk |
| `HighColdStartRate` | > 2% | warning | Too many cold starts |
| `HighPolicyBlockRate` | > 5/sec | warning | Unusual policy blocks |
| `ErrorBudgetBurnRateHigh` | > 1.0 | critical | SLO at risk |

### Recording Rules

Recording rules pre-compute expensive queries for efficient dashboards:

```promql
# Token efficiency
neuronetes:tokens_per_second:rate5m
neuronetes:tokens_per_request:avg5m

# Cost metrics
neuronetes:cost_efficiency:usd_per_token
neuronetes:gpu_cost_per_hour:by_node

# GPU efficiency
neuronetes:tokens_per_gpu_second:rate5m
neuronetes:vram_efficiency:pct

# SLO compliance
neuronetes:ttft_slo_compliance:rate5m
neuronetes:latency_slo_compliance:rate5m
neuronetes:error_rate:rate5m

# Capacity planning
neuronetes:request_rate:rate5m
neuronetes:avg_active_sessions:5m
neuronetes:peak_queue_depth:5m
```

**Testing Recording Rules**:
```bash
# Test rule validity
promtool check rules config/monitoring/prometheus-rules.yaml

# Query recording rules
curl -s 'http://localhost:9090/api/v1/query?query=neuronetes:ttft_slo_compliance:rate5m'
```

## OpenTelemetry Integration

### Configure OTLP Export

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func initMetrics() {
    exporter, _ := otlpmetricgrpc.New(context.Background())
    provider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
    )
    otel.SetMeterProvider(provider)
}
```

### Tracing Integration

Link metrics to traces:

```go
metrics.RecordTTFT(ctx, ttft, "llama-3-70b", "/chat")
// ctx carries trace context automatically
```

## Usage Examples

### Record TTFT

```go
import "github.com/Bowenislandsong/NeuroNetes/pkg/metrics"

m := metrics.NewAgentMetrics(prometheus.DefaultRegisterer)

start := time.Now()
// ... generate first token ...
ttft := time.Since(start)

m.RecordTTFT(ctx, ttft, "llama-3-70b", "/chat")
```

### Record Token Usage

```go
m.RecordTokens(ctx, inputTokens, outputTokens, "llama-3-70b")
```

### Record Cost

```go
costUSD := calculateCost(inputTokens, outputTokens)
m.RecordCost(ctx, costUSD, inputTokens+outputTokens, "llama-3-70b", "tenant-1")
```

### Record GPU Metrics

```go
gpuUtil, vramUsed, vramTotal := getGPUStats()
m.RecordGPUMetrics(ctx, "node-1", gpuUtil, vramUsed, vramTotal)
```

## Testing Metrics

```bash
# Run metrics tests
go test ./pkg/metrics/... -v -race -coverprofile=coverage.out

# Benchmark
go test ./pkg/metrics/... -bench=. -benchmem
```

## Best Practices

1. **Use Labels Sparingly**: High-cardinality labels (user IDs) cause memory issues
2. **Aggregate Before Query**: Use recording rules for expensive queries
3. **Set Retention Policies**: Keep detailed metrics for 7 days, aggregated for 90 days
4. **Monitor Monitoring**: Alert on Prometheus scrape failures
5. **Cost Attribution**: Always include `tenant` label for chargebacks
6. **SLO-Based Alerts**: Alert on user-visible symptoms, not internal metrics
7. **Context in Traces**: Link metrics to traces for deep debugging

## Cloud-Specific Integration

### AWS CloudWatch

```go
import "github.com/aws/aws-sdk-go-v2/service/cloudwatch"

// Export metrics to CloudWatch
putMetricData(&cloudwatch.PutMetricDataInput{
    Namespace: aws.String("NeuroNetes"),
    MetricData: []types.MetricDatum{{
        MetricName: aws.String("TTFT_P95"),
        Value: aws.Float64(ttftP95),
        Unit: types.StandardUnitMilliseconds,
    }},
})
```

### GCP Cloud Monitoring

```go
import monitoring "cloud.google.com/go/monitoring/apiv3/v2"

// Export custom metrics
client.CreateTimeSeries(ctx, &monitoringpb.CreateTimeSeriesRequest{
    Name: "projects/" + projectID,
    TimeSeries: []*monitoringpb.TimeSeries{{
        Metric: &metricpb.Metric{
            Type: "custom.googleapis.com/agent/ttft",
        },
    }},
})
```

### Azure Monitor

```go
import "github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"

// Query metrics
client.QueryResource(ctx, resourceID, &azquery.MetricsQueryOptions{
    MetricNames: []string{"agent_ttft_ms"},
})
```

## Troubleshooting

### Missing Metrics

```bash
# Check ServiceMonitor
kubectl get servicemonitor -n neuronetes-system

# Verify Prometheus targets
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit http://localhost:9090/targets

# Check scrape configs
kubectl get prometheus -n monitoring -o yaml
```

### High Cardinality

```promql
# Find high-cardinality metrics
topk(10, count by (__name__)({__name__=~".+"}))

# Check label cardinality
count by (model)(agent_ttft_ms_bucket)
```

### Performance Issues

```bash
# Check Prometheus resource usage
kubectl top pod -n monitoring -l app=prometheus

# Optimize queries with recording rules
# See config/monitoring/prometheus-rules.yaml
```

## References

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Grafana Dashboard Design](https://grafana.com/docs/grafana/latest/dashboards/)
- [OpenTelemetry Metrics](https://opentelemetry.io/docs/specs/otel/metrics/)
- [SLO Framework](https://sre.google/workbook/implementing-slos/)
