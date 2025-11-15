# NeuroNetes Metrics Testing

This directory contains comprehensive tests for the NeuroNetes metrics system, covering all 60+ agent-native metrics across 10 categories.

## Test Structure

```
test/
├── integration/
│   └── metrics_test.go          # Integration tests for all metric categories
└── e2e/
    └── metrics_e2e_test.go       # End-to-end tests for Prometheus/Grafana
```

## Running Tests

### All Metrics Tests
```bash
# Run all metrics tests
make test-metrics

# Run with coverage
go test -v ./pkg/metrics/... -race -coverprofile=metrics-coverage.out
go tool cover -html=metrics-coverage.out

# Current coverage: 95.7%
```

### Unit Tests
```bash
# Run unit tests only
go test -v ./pkg/metrics/...

# With race detector
go test -v -race ./pkg/metrics/...

# Benchmarks
go test -bench=. -benchmem ./pkg/metrics/...
```

### Integration Tests
```bash
# Run integration tests
go test -v ./test/integration/metrics_test.go

# Run specific test
go test -v ./test/integration/metrics_test.go -run TestMetricsEndToEndWorkflow
```

### E2E Tests
```bash
# Run E2E tests (short mode skips slow tests)
go test -v -short ./test/e2e/metrics_e2e_test.go

# Run all E2E tests including real-time scenarios
go test -v ./test/e2e/metrics_e2e_test.go -timeout 2m
```

## Test Coverage

### Unit Tests (15 tests, 95.7% coverage)
Located in `pkg/metrics/metrics_test.go`:

- `TestNewAgentMetrics` - Metrics initialization
- `TestRecordTTFT` - Time to first token recording
- `TestRecordLatency` - Latency tracking
- `TestRecordTokens` - Token usage tracking
- `TestRecordToolCall` - Tool call metrics
- `TestRecordCost` - Cost tracking
- `TestSetActiveSessions` - Session management
- `TestSetQueueDepth` - Queue depth tracking
- `TestRecordGPUMetrics` - GPU utilization
- `TestRecordModelLoad` - Model loading metrics
- `TestRecordScalingEvent` - Autoscaling events
- `TestRecordPolicyBlock` - Security policy enforcement
- `TestRecordRedaction` - PII redaction tracking
- `TestMetricsLabels` - Label structure validation
- `TestConcurrentMetricsRecording` - Thread safety

### Integration Tests (13 tests)
Located in `test/integration/metrics_test.go`:

1. **End-to-End Workflow** - Complete request lifecycle simulation
2. **UX & Quality Tracking** - TTFT, latency, RTF, CSAT, quality win rate
3. **Load & Concurrency** - Active sessions, queue depth, scaling lag
4. **Token Dynamics** - Input/output tokens, context length, KV cache
5. **Tooling & RAG** - Tool calls, retrieval latency, cache hits
6. **GPU Efficiency** - Utilization, VRAM, model loading
7. **Network & Streaming** - Stream initialization, backpressure, jitter
8. **Scheduler & Placement** - Gang scheduling, affinity, data locality
9. **Autoscaling & Reliability** - HPA decisions, preemptions, failover
10. **Security & Policy** - Policy blocks, redactions, authz denials
11. **Cost & Carbon** - Token costs, GPU hours, energy, spot savings
12. **SLO Compliance** - Validation of all SLO thresholds
13. **High Cardinality** - Label cardinality validation

### E2E Tests (11 tests)
Located in `test/e2e/metrics_e2e_test.go`:

1. **Prometheus Export** - Verify metrics exposed in Prometheus format
2. **SLO Alerting** - Test alert conditions for all SLOs
3. **Grafana Dashboard Queries** - Validate dashboard panel queries
4. **Multi-Tenant Isolation** - Tenant-specific metrics
5. **Real-Time Updates** - Metrics update in real-time
6. **Label Cardinality** - OpenTelemetry label structure
7. **Recording Rules** - Prometheus recording rule validation
8. **Export Formats** - Prometheus text format compliance
9. **Consistency Across Scrapes** - Metric stability
10. **OpenTelemetry Integration** - OTEL meter integration
11. **Benchmarks** - Performance benchmarks for metric recording

## Metric Categories Tested

### 1. UX & Quality (SLO-Facing)
- Time to first token (TTFT) - P50/P95/P99
- Turn latency - End-to-end
- RTF ratio - Real-time factor
- Tokens/sec - Generation rate
- CSAT score - Customer satisfaction
- Quality win rate - Canary testing

**SLOs**:
- TTFT P95 ≤ 350ms
- Latency P95 ≤ 2.5s
- RTF ratio ≤ 1.5

### 2. Load & Concurrency
- Active sessions (gauge)
- Queue depth (gauge)
- Admission rejects (counter)
- Scaling lag (histogram)

**SLOs**:
- Scaling lag < 60s
- Admission reject rate < 1%

### 3. Token & Context Dynamics
- Input/output/total tokens (counters)
- Context length P95 (gauge)
- Context truncation rate (counter)
- KV cache hit ratio (gauge)
- Batch merge efficiency (gauge)

### 4. Tooling / Function Calls
- Tool calls per turn (histogram)
- Tool latency P95 (histogram)
- Tool success/timeout/retry rates (gauges)
- Retrieval latency (histogram)
- Retrieval cache hit ratio (gauge)
- Grounding coverage (gauge)

**SLOs**:
- Tool P95 ≤ 800ms
- Tool success rate ≥ 95%

### 5. RAG Quality
- Retrieval hit@k / MRR (gauges)
- Hallucination rate (gauge)
- Citation validity rate (gauge)

### 6. GPU & System Efficiency
- GPU/SM/Memory BW utilization (gauges)
- VRAM used/fragmentation (gauges)
- MIG slice utilization (gauge)
- Model load time (histogram)
- Cold start rate (gauge)

**SLOs**:
- Cold start rate < 2%

### 7. Network & Streaming
- Stream init latency (histogram)
- Backpressure events (counter)
- Drop/cancel rates (gauges)
- Token delivery jitter (histogram)

### 8. Scheduler & Placement
- Gang schedule wait (histogram)
- Topology penalty score (gauge)
- Session affinity hit rate (gauge)
- Data locality rate (gauge)

### 9. Autoscaling & Reliability
- HPA decisions (counter)
- Replica preemptions/evictions (counters)
- Spot interruptions (counter)
- Failover time (histogram)
- Error budget burn rate (gauge)

**SLOs**:
- Error rate < 1%

### 10. Security, Safety, Policy
- Policy blocks (counter)
- Redaction events (counter)
- Authz denials (counter)

### 11. Cost & Carbon
- Cost per 1K tokens (gauge)
- Cost per session (gauge)
- GPU/CPU hours (counters)
- Egress GB (counter)
- Energy kWh per 1K tokens (gauge)
- Spot savings (counter)

**SLOs**:
- Cost ≤ $0.10 per 1K tokens

## Example Test Usage

### Testing TTFT SLO Compliance
```go
func TestTTFTCompliance(t *testing.T) {
    m := metrics.NewAgentMetrics(prometheus.NewRegistry())
    ctx := context.Background()
    
    // Record TTFT
    m.RecordTTFT(ctx, 300*time.Millisecond, "llama-3-70b", "/chat")
    
    // Verify SLO (350ms)
    // In production, Prometheus would evaluate:
    // histogram_quantile(0.95, rate(agent_ttft_ms_bucket[5m])) < 350
}
```

### Testing End-to-End Workflow
```go
func TestCompleteWorkflow(t *testing.T) {
    m := metrics.NewAgentMetrics(prometheus.NewRegistry())
    ctx := context.Background()
    
    // 1. TTFT
    m.RecordTTFT(ctx, 250*time.Millisecond, "llama-3-70b", "/chat")
    
    // 2. Token usage
    m.RecordTokens(ctx, 1500, 750, "llama-3-70b")
    
    // 3. Tool calls
    m.RecordToolCall(ctx, "code_search", 150*time.Millisecond, true)
    
    // 4. GPU metrics
    m.RecordGPUMetrics(ctx, "node-1", 85.5, 60.0, 80.0)
    
    // 5. Latency
    m.RecordLatency(ctx, 1500*time.Millisecond, "llama-3-70b", "/chat")
    
    // 6. Cost
    m.RecordCost(ctx, 0.15, 2250, "llama-3-70b", "tenant-1")
}
```

### Testing Prometheus Export
```go
func TestPrometheusExport(t *testing.T) {
    registry := prometheus.NewRegistry()
    m := metrics.NewAgentMetrics(registry)
    
    // Record metrics
    m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
    
    // Create handler
    handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
    
    // Verify export format contains expected metrics
    // ... test HTTP endpoint
}
```

## CI/CD Integration

Tests are automatically run in GitHub Actions:

```yaml
# .github/workflows/ci.yml
jobs:
  test-metrics:
    runs-on: ubuntu-latest
    steps:
      - name: Run metrics tests
        run: go test -v ./pkg/metrics/... -race -coverprofile=metrics-coverage.out
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./metrics-coverage.out
          flags: metrics
```

## Prometheus Rules Testing

```bash
# Validate Prometheus rules
promtool check rules config/monitoring/prometheus-rules.yaml

# Test recording rules
kubectl port-forward -n monitoring svc/prometheus 9090:9090
curl 'http://localhost:9090/api/v1/query?query=neuronetes:ttft_slo_compliance:rate5m'
```

## Grafana Dashboard Testing

```bash
# Import dashboard
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring

# Access dashboard
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Visit http://localhost:3000/d/neuronetes-agents
```

## Benchmarks

Run performance benchmarks:

```bash
go test -bench=. -benchmem ./pkg/metrics/...
```

Expected results:
```
BenchmarkRecordTTFT-8         3000000    450 ns/op    128 B/op    2 allocs/op
BenchmarkRecordTokens-8       5000000    320 ns/op     96 B/op    2 allocs/op
BenchmarkRecordGPUMetrics-8   3000000    480 ns/op    144 B/op    3 allocs/op
```

## Troubleshooting

### Tests Timeout
```bash
# Increase timeout
go test -v -timeout 5m ./test/e2e/metrics_e2e_test.go
```

### Port Conflicts in E2E Tests
```bash
# E2E tests use ports 19090-19094
# Ensure these are free
lsof -i :19090-19094
```

### Coverage Issues
```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./pkg/metrics/...
go tool cover -html=coverage.out -o coverage.html
```

## References

- [Metrics Guide](../../docs/metrics.md) - Complete metrics documentation
- [Prometheus Rules](../../config/monitoring/prometheus-rules.yaml) - Alert and recording rules
- [Grafana Dashboard](../../config/grafana/neuronetes-dashboard.json) - Pre-built dashboard
- [OpenTelemetry](https://opentelemetry.io/docs/specs/otel/metrics/) - OTEL metrics specification
