# Autoscaling Guide

## Overview

NeuroNetes provides token-aware autoscaling that understands the unique characteristics of LLM workloads. This guide covers autoscaling strategies, metrics, and best practices.

## Why Token-Based Autoscaling?

Traditional CPU/memory-based autoscaling fails for LLM workloads because:

1. **GPU workloads have flat CPU usage** - Not correlated with load
2. **Memory is pre-allocated** - Model weights loaded regardless of traffic
3. **Load is token-based** - Real load is tokens/sec, not RPS
4. **Context matters** - Long contexts consume more resources
5. **TTFT is critical** - First token latency affects UX more than throughput

## Token-Aware Metrics

### Core Metrics

#### 1. Tokens In Queue

**Description**: Total tokens waiting to be processed

**Formula**:
```
tokens_in_queue = sum(input_tokens) for all queued requests
```

**Use case**: Primary scaling signal for queue-based workloads

**Example**:
```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: queue-pool
spec:
  autoscaling:
    metrics:
      - type: tokens-in-queue
        target: "1000"  # Scale when > 1000 tokens queued
        averagingWindow: 30s
```

#### 2. Time-To-First-Token (TTFT) P95

**Description**: 95th percentile latency for first token

**Formula**:
```
ttft_p95 = quantile(0.95, ttft_measurements)
```

**Use case**: Maintain user experience quality

**Example**:
```yaml
metrics:
  - type: ttft-p95
    target: "500ms"  # Scale if P95 > 500ms
    averagingWindow: 1m
```

#### 3. Concurrent Sessions

**Description**: Number of active conversations

**Formula**:
```
concurrent_sessions = count(sessions with activity in last 5m)
```

**Use case**: Capacity planning for stateful agents

**Example**:
```yaml
metrics:
  - type: concurrent-sessions
    target: "50"  # Scale when > 50 sessions
    averagingWindow: 1m
```

#### 4. Tokens Per Second

**Description**: Current token generation throughput

**Formula**:
```
tokens_per_second = rate(output_tokens_total[1m])
```

**Use case**: Maintain target throughput

**Example**:
```yaml
metrics:
  - type: tokens-per-second
    target: "100"  # Scale when < 100 tok/s capacity
    averagingWindow: 1m
```

#### 5. Queue Depth

**Description**: Number of requests waiting

**Formula**:
```
queue_depth = len(queue)
```

**Use case**: Simple queue length monitoring

**Example**:
```yaml
metrics:
  - type: queue-depth
    target: "20"  # Scale when > 20 requests
    averagingWindow: 30s
```

#### 6. Context Length

**Description**: Average context window size

**Formula**:
```
avg_context_length = avg(input_tokens + output_tokens) per request
```

**Use case**: Context-aware capacity planning

**Example**:
```yaml
metrics:
  - type: context-length
    target: "4000"  # Scale when avg > 4K tokens
    averagingWindow: 2m
```

#### 7. Tool Call Rate

**Description**: Frequency of tool invocations

**Formula**:
```
tool_call_rate = count(tool_calls) / duration
```

**Use case**: Resource-intensive tool workloads

**Example**:
```yaml
metrics:
  - type: tool-call-rate
    target: "10"  # Scale when > 10 calls/min
    averagingWindow: 1m
```

## Scaling Behavior

### Scaling Algorithm

```
1. Collect Metrics
   ├─ Fetch from Prometheus
   ├─ Apply averaging window
   └─ Calculate current values

2. Evaluate Targets
   for each metric:
     ├─ ratio = current / target
     └─ if ratio > 1.0: scale_up_signal
     
3. Determine Replicas
   ├─ desired = max(ratio * current_replicas)
   ├─ Apply scale-up limits
   ├─ Apply scale-down limits
   └─ Respect min/max bounds

4. Apply Stabilization
   ├─ Check stabilization window
   ├─ Apply cooldown period
   └─ Execute scaling
```

### Scaling Policies

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: controlled-pool
spec:
  minReplicas: 2
  maxReplicas: 20
  
  autoscaling:
    behavior:
      scaleUp:
        # Wait 60s before evaluating scale-up
        stabilizationWindow: 60s
        
        # Max 100% increase per scale operation
        maxChangePercent: 100
        
        # Max 5 pods added per scale operation
        maxChangeAbsolute: 5
        
        # Evaluate every 30s
        periodSeconds: 30
      
      scaleDown:
        # Wait 5m before scaling down
        stabilizationWindow: 300s
        
        # Max 50% decrease per scale operation
        maxChangePercent: 50
        
        # Max 2 pods removed per scale operation
        maxChangeAbsolute: 2
        
        # Evaluate every 2m
        periodSeconds: 120
    
    # Wait 5m between any scaling operations
    cooldownPeriod: 5m
```

### Multi-Metric Scaling

Use multiple metrics for robustness:

```yaml
autoscaling:
  metrics:
    # Primary: Queue-based
    - type: tokens-in-queue
      target: "500"
      averagingWindow: 30s
    
    # Secondary: Latency-based
    - type: ttft-p95
      target: "800ms"
      averagingWindow: 1m
    
    # Tertiary: Session-based
    - type: concurrent-sessions
      target: "40"
      averagingWindow: 1m
  
  # Scale if ANY metric exceeds target
  strategy: max  # max, min, average
```

## Warm Pool Integration

### Prewarming Strategy

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: warm-pool
spec:
  minReplicas: 5
  maxReplicas: 50
  
  # Keep 20% of max replicas warm
  prewarmPercent: 20  # = 10 warm replicas
  
  autoscaling:
    # When scaling up, use warm replicas first
    useWarmPool: true
    
    # Replenish warm pool after usage
    replenishStrategy:
      # Time to wait before replenishing
      delay: 30s
      
      # Batch size for replenishment
      batchSize: 2
```

### Scaling with Warm Pool

```
State Transition:

Idle → Warm → Serving → Idle
  ↑      ↑       ↓
  └──────┴───────┘
```

Benefits:
- **Fast scale-up**: < 1s from warm to serving
- **Better UX**: Reduced cold start latency
- **Cost efficiency**: Pay only when needed
- **Smooth scaling**: No traffic disruption

### Cost vs Performance

```yaml
# Aggressive prewarming (fast, expensive)
prewarmPercent: 50

# Balanced (good UX, reasonable cost)
prewarmPercent: 20

# Conservative (slower, cheaper)
prewarmPercent: 10

# No prewarming (cheapest, slowest)
prewarmPercent: 0
```

## Queue-Based Autoscaling

### NATS/Kafka Integration

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: queue-autoscale
spec:
  type: queue
  
  queueConfig:
    provider: nats
    queueName: agent-tasks
    
    # Enable automatic scaling on lag
    autoscaleOnLag: true
    
    # Scale when > 100 messages queued
    maxLagThreshold: 100
    
    # Each replica can handle 10 concurrent messages
    prefetchCount: 10
```

### Scaling Formula

```
desired_replicas = ceil(
    (queue_depth - (current_replicas * prefetch)) / 
    (prefetch * target_throughput)
)
```

Example:
- Queue depth: 150 messages
- Current replicas: 5
- Prefetch: 10
- Target throughput: 5 msg/s per replica

```
desired = ceil((150 - 50) / (10 * 5)) = ceil(2) = 2 additional replicas
```

## Predictive Autoscaling

### Traffic Pattern Learning

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: predictive-pool
spec:
  autoscaling:
    predictive:
      enabled: true
      
      # Learn from historical patterns
      lookbackWindow: 7d
      
      # Predict ahead
      horizonMinutes: 10
      
      # Confidence threshold
      minConfidence: 0.7
      
      # Schedule-based scaling
      schedules:
        - name: business-hours
          cron: "0 8 * * 1-5"  # 8 AM weekdays
          replicas: 20
        
        - name: off-hours
          cron: "0 18 * * *"   # 6 PM daily
          replicas: 5
```

### ML-Based Prediction

```python
# Example prediction model
import numpy as np
from sklearn.ensemble import RandomForestRegressor

def predict_load(historical_data, horizon_minutes):
    """
    Predict future load based on historical patterns.
    
    Features:
    - Time of day
    - Day of week
    - Recent trend
    - Seasonal patterns
    """
    features = extract_features(historical_data)
    model = train_model(features)
    
    future_load = model.predict(horizon=horizon_minutes)
    recommended_replicas = calculate_replicas(future_load)
    
    return recommended_replicas
```

## SLO-Aware Autoscaling

### SLO Definition

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: slo-agent
spec:
  slo:
    # Target TTFT
    ttft: 500ms
    
    # Target throughput
    tokensPerSecond: 50
    
    # Target P95 latency
    p95Latency: 2s
    
    # Availability target
    availabilityPercent: 99.5
```

### SLO-Based Scaling

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: slo-pool
spec:
  autoscaling:
    # Scale to meet SLO
    sloTarget:
      enabled: true
      
      # Headroom above SLO
      headroomPercent: 20
      
      # Fast scale on SLO breach
      breachResponse:
        scaleUpImmediately: true
        maxScaleUp: 10
        
      # Scale down only when safe
      scaleDownSafety:
        requireHeadroom: 50  # % above SLO
        checkDuration: 10m
```

## Cost-Aware Autoscaling

### Budget Constraints

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: budget-pool
spec:
  autoscaling:
    costConstraints:
      # Maximum hourly cost
      maxCostPerHour: 100.0
      
      # Budget allocation
      dailyBudget: 1000.0
      
      # Actions when budget exceeded
      onBudgetExceeded:
        - scale-to-min
        - use-fallback-model
        - enable-spot
      
      # Cost optimization
      optimization:
        # Prefer cheaper instances
        preferSpot: true
        spotMaxPrice: 5.0  # $/hour
        
        # Use smaller model when possible
        dynamicModelSelection:
          enabled: true
          costThreshold: 0.50  # Switch at 50% of budget
```

### Spot Instance Autoscaling

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: spot-pool
spec:
  autoscaling:
    spotStrategy:
      # Enable spot instances
      enabled: true
      
      # Maximum % on spot
      maxSpotPercent: 70
      
      # Fallback on interruption
      onInterruption:
        action: migrate-to-on-demand
        timeout: 30s
      
      # Diversification
      instanceTypes:
        - g4dn.12xlarge
        - g5.12xlarge
        - p3.8xlarge
```

## Monitoring

### Key Metrics

```promql
# Current replicas
neuronetes_agentpool_replicas{pool="my-pool"}

# Desired replicas
neuronetes_agentpool_desired_replicas{pool="my-pool"}

# Scaling operations
rate(neuronetes_agentpool_scale_operations_total[5m])

# Tokens in queue
neuronetes_tokens_in_queue{pool="my-pool"}

# TTFT P95
histogram_quantile(0.95, 
  rate(neuronetes_ttft_seconds_bucket[5m])
)

# Concurrent sessions
neuronetes_concurrent_sessions{pool="my-pool"}

# Warm pool size
neuronetes_warm_pool_replicas{pool="my-pool"}

# Warm pool utilization
neuronetes_warm_pool_activations_total /
neuronetes_warm_pool_replicas
```

### Grafana Dashboards

Example dashboard queries:

```yaml
# Panel: Replicas Over Time
- title: Agent Pool Replicas
  type: graph
  targets:
    - expr: neuronetes_agentpool_replicas
    - expr: neuronetes_agentpool_desired_replicas
    - expr: neuronetes_warm_pool_replicas

# Panel: Autoscaling Metrics
- title: Scaling Triggers
  type: graph
  targets:
    - expr: neuronetes_tokens_in_queue / 1000
    - expr: neuronetes_concurrent_sessions
    - expr: neuronetes_ttft_p95_milliseconds / 1000

# Panel: Scaling Events
- title: Scale Operations
  type: stat
  targets:
    - expr: sum(increase(neuronetes_scale_up_total[1h]))
    - expr: sum(increase(neuronetes_scale_down_total[1h]))
```

## Best Practices

### 1. Start Conservative

```yaml
# Begin with simple, conservative settings
autoscaling:
  metrics:
    - type: concurrent-sessions
      target: "50"
  
  behavior:
    scaleUp:
      stabilizationWindow: 120s
      maxChangeAbsolute: 2
    
    scaleDown:
      stabilizationWindow: 600s
      maxChangeAbsolute: 1
```

### 2. Use Multiple Metrics

```yaml
# Combine complementary metrics
metrics:
  - type: tokens-in-queue  # Immediate load
  - type: ttft-p95         # User experience
  - type: concurrent-sessions  # Capacity planning
```

### 3. Tune Stabilization Windows

```yaml
# Fast scale-up for responsiveness
scaleUp:
  stabilizationWindow: 60s

# Slow scale-down for stability
scaleDown:
  stabilizationWindow: 300s
```

### 4. Set Appropriate Min/Max

```yaml
# Ensure minimum availability
minReplicas: 2  # High availability

# Cap maximum for cost control
maxReplicas: 20  # Budget limit
```

### 5. Monitor and Iterate

- Track scaling events
- Measure SLO compliance
- Adjust thresholds based on data
- Test during peak load

## Troubleshooting

### Frequent Scaling

**Symptom**: Pods constantly scaling up and down

**Causes**:
- Stabilization window too short
- Averaging window too short
- Multiple conflicting metrics

**Solution**:
```yaml
# Increase stabilization
behavior:
  scaleUp:
    stabilizationWindow: 120s
  scaleDown:
    stabilizationWindow: 600s

# Increase averaging
metrics:
  - type: tokens-in-queue
    averagingWindow: 2m  # was 30s
```

### Slow Scale-Up

**Symptom**: Takes too long to scale up under load

**Causes**:
- Stabilization window too long
- maxChangeAbsolute too small
- No warm pool

**Solution**:
```yaml
# Faster scale-up
scaleUp:
  stabilizationWindow: 30s
  maxChangeAbsolute: 5

# Add warm pool
prewarmPercent: 20
```

### Not Scaling Down

**Symptom**: Replicas stay high when load decreases

**Causes**:
- Scale-down disabled
- Cooldown period too long
- Stabilization window too long

**Solution**:
```yaml
# Enable scale-down
scaleDown:
  stabilizationWindow: 300s
  maxChangeAbsolute: 2

# Reduce cooldown
cooldownPeriod: 3m
```

## Advanced Topics

### Custom Metrics

Implement custom metrics:

```go
type CustomMetricProvider struct{}

func (p *CustomMetricProvider) GetMetricValue(
    ctx context.Context,
    pool *neuronetes.AgentPool,
) (float64, error) {
    // Custom logic
    value := calculateCustomMetric(pool)
    return value, nil
}

func (p *CustomMetricProvider) GetMetricName() string {
    return "custom-metric"
}
```

### External Metrics

Use external data sources:

```yaml
autoscaling:
  externalMetrics:
    - name: datadog-queue-depth
      query: "avg:queue.depth{service:agents}"
      target: "100"
```

### Webhook Integration

Trigger scaling via webhook:

```bash
curl -X POST https://neuronetes-api/v1/scale \
  -H "Content-Type: application/json" \
  -d '{
    "pool": "my-pool",
    "replicas": 10,
    "reason": "traffic-spike-detected"
  }'
```

## References

- [Kubernetes HPA Documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [KEDA Scalers](https://keda.sh/docs/scalers/)
- [Prometheus Metrics](https://prometheus.io/docs/concepts/metric_types/)
