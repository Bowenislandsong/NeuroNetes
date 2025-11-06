# Plugin Development Guide

## Overview

NeuroNetes provides a plugin framework for extending its functionality with custom algorithms. You can create plugins for:

- **Schedulers** - Custom node selection and scoring
- **Autoscalers** - Custom scaling algorithms
- **Model Loaders** - Custom model loading strategies
- **Metrics Providers** - Custom metrics collection
- **Guardrails** - Custom safety checks

## Plugin Types

### 1. Scheduler Plugin

Custom scheduling logic for pod placement.

```go
package myplugins

import (
    "context"
    "github.com/bowenislandsong/neuronetes/pkg/plugins"
    neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
    corev1 "k8s.io/api/core/v1"
)

type CustomScheduler struct {
    name string
}

func NewCustomScheduler() *CustomScheduler {
    return &CustomScheduler{name: "custom-scheduler"}
}

func (s *CustomScheduler) Name() string {
    return s.name
}

func (s *CustomScheduler) Filter(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) bool {
    // Return true if node is suitable
    // Example: Check custom requirements
    customLabel, ok := node.Labels["custom-requirement"]
    return ok && customLabel == "met"
}

func (s *CustomScheduler) Score(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) int64 {
    // Return score 0-100
    var score int64 = 50
    
    // Add custom scoring logic
    if node.Labels["performance-tier"] == "high" {
        score += 30
    }
    
    return score
}

func (s *CustomScheduler) Priority() int {
    return 200 // Higher priority runs first
}

// Register plugin
func init() {
    plugins.RegisterScheduler(NewCustomScheduler())
}
```

### 2. Autoscaler Plugin

Custom autoscaling algorithms.

```go
type CustomAutoscaler struct {
    name string
}

func NewCustomAutoscaler() *CustomAutoscaler {
    return &CustomAutoscaler{name: "custom-autoscaler"}
}

func (a *CustomAutoscaler) Name() string {
    return a.name
}

func (a *CustomAutoscaler) CalculateReplicas(
    ctx context.Context,
    pool *neuronetes.AgentPool,
    currentMetrics map[string]float64,
) (int32, error) {
    // Custom scaling logic
    queueDepth := currentMetrics["queue-depth"]
    targetDepthPerReplica := 10.0
    
    desiredReplicas := int32(math.Ceil(queueDepth / targetDepthPerReplica))
    
    // Apply bounds
    if desiredReplicas < pool.Spec.MinReplicas {
        desiredReplicas = pool.Spec.MinReplicas
    }
    if desiredReplicas > pool.Spec.MaxReplicas {
        desiredReplicas = pool.Spec.MaxReplicas
    }
    
    return desiredReplicas, nil
}

func (a *CustomAutoscaler) GetMetricNames() []string {
    return []string{"queue-depth", "custom-metric"}
}

func (a *CustomAutoscaler) Priority() int {
    return 150
}

func init() {
    plugins.RegisterAutoscaler(NewCustomAutoscaler())
}
```

### 3. Model Loader Plugin

Custom model loading strategies.

```go
type S3ModelLoader struct {
    name string
    s3Client *s3.Client
}

func NewS3ModelLoader() *S3ModelLoader {
    return &S3ModelLoader{
        name: "s3-loader",
        s3Client: createS3Client(),
    }
}

func (l *S3ModelLoader) Name() string {
    return l.name
}

func (l *S3ModelLoader) CanLoad(ctx context.Context, model *neuronetes.Model) bool {
    // Check if we can handle this URI
    return strings.HasPrefix(model.Spec.WeightsURI, "s3://")
}

func (l *S3ModelLoader) Load(ctx context.Context, model *neuronetes.Model, node string) error {
    // Download from S3
    bucket, key := parseS3URI(model.Spec.WeightsURI)
    
    // Download to node
    cachePath := fmt.Sprintf("/var/lib/neuronetes/models/%s", model.Name)
    return l.s3Client.DownloadToPath(ctx, bucket, key, cachePath)
}

func (l *S3ModelLoader) Unload(ctx context.Context, model *neuronetes.Model, node string) error {
    // Clean up cached files
    cachePath := fmt.Sprintf("/var/lib/neuronetes/models/%s", model.Name)
    return os.RemoveAll(cachePath)
}

func (l *S3ModelLoader) Priority() int {
    return 100
}

func init() {
    plugins.RegisterModelLoader(NewS3ModelLoader())
}
```

### 4. Guardrail Plugin

Custom safety checks.

```go
type PIIDetectionGuardrail struct {
    name string
    detector *pii.Detector
}

func NewPIIDetectionGuardrail() *PIIDetectionGuardrail {
    return &PIIDetectionGuardrail{
        name: "pii-detection",
        detector: pii.NewDetector(),
    }
}

func (g *PIIDetectionGuardrail) Name() string {
    return g.name
}

func (g *PIIDetectionGuardrail) Check(
    ctx context.Context,
    request *plugins.GuardrailRequest,
) (*plugins.GuardrailResult, error) {
    // Detect PII
    findings := g.detector.Scan(request.Content)
    
    if len(findings) > 0 {
        // Redact PII
        redacted := g.detector.Redact(request.Content, findings)
        
        return &plugins.GuardrailResult{
            Passed: false,
            Action: "redact",
            Reason: fmt.Sprintf("Found %d PII instances", len(findings)),
            Confidence: 0.95,
            Metadata: map[string]string{
                "redacted_content": redacted,
                "pii_types": strings.Join(findings, ","),
            },
        }, nil
    }
    
    return &plugins.GuardrailResult{
        Passed: true,
        Action: "allow",
        Confidence: 1.0,
    }, nil
}

func (g *PIIDetectionGuardrail) GetType() string {
    return "pii-detection"
}

func init() {
    plugins.RegisterGuardrail(NewPIIDetectionGuardrail())
}
```

### 5. Metrics Provider Plugin

Custom metrics collection.

```go
type PrometheusMetricsProvider struct {
    name string
    client *prometheus.Client
}

func NewPrometheusMetricsProvider() *PrometheusMetricsProvider {
    return &PrometheusMetricsProvider{
        name: "prometheus",
        client: createPrometheusClient(),
    }
}

func (p *PrometheusMetricsProvider) Name() string {
    return p.name
}

func (p *PrometheusMetricsProvider) GetMetric(
    ctx context.Context,
    pool *neuronetes.AgentPool,
    metricType string,
) (float64, error) {
    // Query Prometheus
    query := fmt.Sprintf(
        `neuronetes_%s{pool="%s"}`,
        metricType,
        pool.Name,
    )
    
    result, err := p.client.Query(ctx, query)
    if err != nil {
        return 0, err
    }
    
    return parseResult(result), nil
}

func (p *PrometheusMetricsProvider) ListMetrics() []string {
    return []string{
        "tokens_per_second",
        "ttft_p95",
        "concurrent_sessions",
        "queue_depth",
    }
}

func init() {
    plugins.RegisterMetricsProvider(NewPrometheusMetricsProvider())
}
```

## Using Plugins

### 1. Build Your Plugin

Create a Go package with your plugin:

```bash
mkdir -p plugins/myplugin
cd plugins/myplugin

# Create plugin.go with your implementation
cat > plugin.go <<EOF
package myplugin

import "github.com/bowenislandsong/neuronetes/pkg/plugins"

// Your plugin code here

func init() {
    plugins.RegisterScheduler(NewMyScheduler())
}
EOF
```

### 2. Import in Main

```go
// cmd/manager/main.go
package main

import (
    _ "github.com/bowenislandsong/neuronetes/plugins/myplugin" // Import to register
)

func main() {
    // Your main code
}
```

### 3. Build with Plugin

```bash
go build -o bin/manager ./cmd/manager/main.go
```

### 4. Configure Plugin

Some plugins support configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-config
data:
  scheduler-plugins: |
    - name: custom-scheduler
      enabled: true
      priority: 200
      config:
        custom-param: value
```

## Testing Plugins

```go
package myplugin_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/bowenislandsong/neuronetes/plugins/myplugin"
)

func TestCustomScheduler(t *testing.T) {
    scheduler := myplugin.NewCustomScheduler()
    
    // Test filter
    passed := scheduler.Filter(ctx, pod, node, pool)
    assert.True(t, passed)
    
    // Test score
    score := scheduler.Score(ctx, pod, node, pool)
    assert.GreaterOrEqual(t, score, int64(0))
    assert.LessOrEqual(t, score, int64(100))
}
```

## Best Practices

### 1. Error Handling

```go
func (p *Plugin) Process(ctx context.Context) error {
    if err := p.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Process with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    return p.doWork(ctx)
}
```

### 2. Logging

```go
import "sigs.k8s.io/controller-runtime/pkg/log"

func (p *Plugin) Process(ctx context.Context) error {
    log := log.FromContext(ctx)
    
    log.Info("processing", "plugin", p.Name())
    
    if err := p.doWork(ctx); err != nil {
        log.Error(err, "processing failed")
        return err
    }
    
    log.Info("processing complete")
    return nil
}
```

### 3. Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    pluginDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "plugin_duration_seconds",
            Help: "Plugin execution duration",
        },
        []string{"plugin_name"},
    )
)

func (p *Plugin) Process(ctx context.Context) error {
    start := time.Now()
    defer func() {
        pluginDuration.WithLabelValues(p.Name()).Observe(
            time.Since(start).Seconds(),
        )
    }()
    
    return p.doWork(ctx)
}
```

### 4. Thread Safety

```go
type Plugin struct {
    mu sync.RWMutex
    cache map[string]interface{}
}

func (p *Plugin) Get(key string) interface{} {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.cache[key]
}

func (p *Plugin) Set(key string, value interface{}) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.cache[key] = value
}
```

## Examples

See `pkg/plugins/examples.go` for complete working examples of all plugin types.

## Plugin Registry

Access registered plugins:

```go
registry := plugins.GetGlobalRegistry()

// Get all schedulers
schedulers := registry.GetSchedulers()
for _, s := range schedulers {
    fmt.Printf("Scheduler: %s (priority: %d)\n", s.Name(), s.Priority())
}

// Get all autoscalers
autoscalers := registry.GetAutoscalers()

// Get specific plugin
for _, a := range autoscalers {
    if a.Name() == "custom-autoscaler" {
        // Use this autoscaler
    }
}
```

## Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# List loaded plugins
curl http://localhost:8080/debug/plugins

# Check plugin execution
curl http://localhost:8080/debug/plugins/scheduler/trace
```

## Contributing Plugins

To contribute a plugin to NeuroNetes:

1. Create plugin in `pkg/plugins/contrib/`
2. Add tests
3. Add documentation
4. Submit PR

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.
