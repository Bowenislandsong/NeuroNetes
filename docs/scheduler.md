# Scheduler Guide

## Overview

NeuroNetes includes specialized schedulers optimized for agent workloads. This guide covers the GPU topology-aware scheduler, token-based scheduling, and cost/SLO optimization.

## GPU Topology-Aware Scheduler

### Why Custom GPU Scheduling?

The default Kubernetes scheduler treats all resources uniformly. For GPU workloads, this approach fails because:

1. **GPU Memory**: Not all GPUs have equal memory
2. **Topology**: Inter-GPU bandwidth varies (PCIe vs NVLink)
3. **NUMA**: GPU-CPU affinity affects performance
4. **MIG**: Multi-Instance GPU partitioning needs special handling
5. **Gang Scheduling**: Multi-GPU jobs need all-or-nothing placement

### Architecture

```
┌────────────────────────────────────────────────┐
│          GPU Topology Scheduler                │
├────────────────────────────────────────────────┤
│                                                 │
│  ┌──────────────┐  ┌──────────────┐           │
│  │   Filter     │  │    Score     │           │
│  │   Phase      │──│    Phase     │           │
│  └──────────────┘  └──────────────┘           │
│         │                  │                    │
│         ▼                  ▼                    │
│  ┌──────────────┐  ┌──────────────┐           │
│  │  Topology    │  │ Bin-Packing  │           │
│  │  Filtering   │  │   Scoring    │           │
│  └──────────────┘  └──────────────┘           │
│         │                  │                    │
│         ▼                  ▼                    │
│  ┌──────────────────────────────┐             │
│  │      Gang Reservation        │             │
│  └──────────────────────────────┘             │
└────────────────────────────────────────────────┘
```

### Scheduling Policies

#### 1. Topology-Aware Placement

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: tensor-parallel-pool
spec:
  gpuRequirements:
    count: 8
    type: A100
    topology:
      # Requires all GPUs on same node with NVLink
      locality: same-node
      minBandwidth: 600Gi  # NVLink bandwidth
```

Locality options:
- `same-node`: All GPUs on one node
- `same-socket`: GPUs on same CPU socket
- `nvlink`: GPUs connected via NVLink
- `any`: No topology requirement

#### 2. MIG Partition Scheduling

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: small-model-pool
spec:
  migProfile: "1g.5gb"  # 1 GPU slice, 5GB memory
  gpuRequirements:
    count: 1
    type: A100
```

Supported MIG profiles (A100):
- `1g.5gb`: 7 instances per GPU
- `2g.10gb`: 3 instances per GPU
- `3g.20gb`: 2 instances per GPU
- `4g.20gb`: 1 instance per GPU
- `7g.40gb`: Full GPU

#### 3. Gang Scheduling

For tensor/pipeline parallel workloads:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: pipeline-parallel-pool
spec:
  minReplicas: 4  # Will only schedule if 4 GPUs available
  gpuRequirements:
    count: 2  # Each replica needs 2 GPUs
    topology:
      locality: same-node
  scheduling:
    # Gang scheduling: all replicas scheduled together
    gangScheduling:
      enabled: true
      minMembers: 4
```

### Scoring Algorithm

The scheduler scores nodes based on:

1. **GPU Availability** (weight: 30%)
   - Available GPU count
   - Available GPU memory
   - MIG partition availability

2. **Topology Fitness** (weight: 25%)
   - Inter-GPU bandwidth
   - NUMA locality
   - PCIe generation

3. **Model Cache Presence** (weight: 20%)
   - Is model already cached on node?
   - Cache hit rate on node

4. **Cost Efficiency** (weight: 15%)
   - $/hour for the node
   - Spot vs on-demand pricing
   - Power consumption

5. **Data Locality** (weight: 10%)
   - Co-location with vector stores
   - Co-location with caches
   - Network distance

### Node Labeling

Label GPU nodes for scheduling:

```bash
# Label GPU type
kubectl label node gpu-node-1 neuronetes.io/gpu-type=A100

# Label GPU memory
kubectl label node gpu-node-1 neuronetes.io/gpu-memory=40Gi

# Label GPU count
kubectl label node gpu-node-1 neuronetes.io/gpu-count=8

# Label topology
kubectl label node gpu-node-1 neuronetes.io/gpu-topology=nvlink

# Label MIG capability
kubectl label node gpu-node-1 neuronetes.io/mig-capable=true

# Label MIG configuration
kubectl label node gpu-node-1 neuronetes.io/mig-config=1g.5gb:7,2g.10gb:3
```

## Token-Based Scheduling

### Metrics

The scheduler considers token-based metrics:

```
# Tokens waiting in queue
tokens_in_queue = sum(tokens) for messages in queue

# Expected time-to-first-token
expected_ttft = (tokens_in_queue / tokens_per_second) + model_load_time

# Concurrent sessions
concurrent_sessions = count(active_sessions)

# Context length distribution
avg_context_length = avg(context_length) for sessions
```

### Scheduling Decisions

```
if expected_ttft > slo.ttft:
    # Scale up needed
    new_replicas = ceil(tokens_in_queue / (tokens_per_second * slo.ttft))
    scale_to(min(new_replicas, max_replicas))

if tokens_in_queue < (tokens_per_second * 0.5):
    # Scale down opportunity
    if time_since_last_request > cooldown:
        scale_to(max(current - 1, min_replicas))
```

### Queue-Aware Scheduling

With message queues:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: queue-binding
spec:
  queueConfig:
    autoscaleOnLag: true
    maxLagThreshold: 100  # messages
    
    # Scheduler calculates:
    # lag = queue_depth - (replicas * prefetch * throughput)
    # if lag > maxLagThreshold: scale up
```

## Cost/SLO Optimization

### Multi-Objective Scheduling

The scheduler balances multiple objectives:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: cost-optimized-pool
spec:
  scheduling:
    priority: 100
    
    costOptimization:
      enabled: true
      
      # Primary constraint: don't exceed SLO
      sloHeadroomMs: 1000  # Must have 1s headroom
      
      # Cost constraint
      maxCostPerHour: 50.0
      
      # Enable spot instances
      spotEnabled: true
      spotMaxInterruptions: 2  # per hour
      
      # Fallback strategy
      fallbackModel: smaller-model
      fallbackTrigger:
        costExceeded: true
        sloAtRisk: false
```

### Decision Tree

```
1. Check SLO compliance
   ├─ SLO met? → Try cost reduction
   │  ├─ Spot available? → Use spot
   │  ├─ Smaller model viable? → Degrade model
   │  └─ Current setup → Keep as-is
   │
   └─ SLO at risk? → Improve performance
      ├─ Scale up replicas
      ├─ Use on-demand instances
      └─ Use larger/faster model
```

### Spot Instance Integration

Safely use spot instances:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: spot-pool
spec:
  scheduling:
    costOptimization:
      spotEnabled: true
      sloHeadroomMs: 2000  # 2s buffer for interruptions
      
      # Spot strategy
      spotStrategy:
        # Diversification
        instanceTypes:
          - "p4d.24xlarge"
          - "p3dn.24xlarge"
        
        # Fallback on interruption
        onInterruption: migrate-to-on-demand
        
        # Max % of replicas on spot
        maxSpotPercent: 80
        
      # Monitor interruption rate
      interruptionThreshold:
        count: 3
        window: 1h
        action: disable-spot
```

### Carbon-Aware Scheduling

Minimize carbon footprint:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: green-pool
spec:
  scheduling:
    carbonAware:
      enabled: true
      
      # Prefer regions with clean energy
      preferredRegions:
        - us-west-2  # Hydroelectric
        - eu-north-1  # Renewable
      
      # Time-shift non-urgent work
      timeShifting:
        enabled: true
        cleanEnergyHours: "09:00-17:00"  # Peak solar
        
      # Maximum carbon intensity (gCO2/kWh)
      maxCarbonIntensity: 100
```

## Data Locality Scheduling

### Co-Scheduling with Vector Stores

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: rag-pool
spec:
  scheduling:
    dataLocality:
      # Co-locate with vector store
      vectorStoreAffinity:
        - weaviate-shard-0
        - weaviate-shard-1
      
      # Scoring bonus for locality
      localityWeight: high  # low, medium, high
      
      # Anti-affinity with other workloads
      antiAffinity:
        - training-jobs
        - batch-inference
```

### Cache-Aware Scheduling

Prefer nodes with cached models:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: cached-pool
spec:
  scheduling:
    cacheAware:
      enabled: true
      
      # Prefer nodes with model cached
      cacheHitWeight: 0.3  # 30% of score
      
      # Preload model on specific nodes
      preloadStrategy:
        nodes:
          labelSelector:
            matchLabels:
              node-pool: gpu-primary
        minCachedReplicas: 2
```

## Monitoring

### Scheduler Metrics

```promql
# Scheduling latency
neuronetes_scheduler_latency_seconds

# Scheduling throughput
rate(neuronetes_scheduler_attempts_total[5m])

# Success rate
rate(neuronetes_scheduler_success_total[5m]) / 
rate(neuronetes_scheduler_attempts_total[5m])

# Pending pods
neuronetes_scheduler_pending_pods

# GPU utilization
neuronetes_gpu_utilization_percent

# Topology violations
neuronetes_scheduler_topology_violations_total
```

### Debugging

```bash
# View scheduler logs
kubectl logs -n neuronetes-system deployment/gpu-scheduler -f

# Check scheduling events
kubectl get events --field-selector involvedObject.kind=Pod

# View node GPU resources
kubectl get nodes -o custom-columns=\
NAME:.metadata.name,\
GPU-TYPE:.metadata.labels.neuronetes\.io/gpu-type,\
GPU-COUNT:.metadata.labels.neuronetes\.io/gpu-count,\
GPU-MEMORY:.metadata.labels.neuronetes\.io/gpu-memory

# Describe pod scheduling
kubectl describe pod <pod-name>
```

## Best Practices

### 1. Right-Size GPU Requests

```yaml
# Bad: Over-requesting
gpuRequirements:
  count: 4  # Only using 2
  memory: "80Gi"  # Only need 40Gi

# Good: Match actual needs
gpuRequirements:
  count: 2
  memory: "40Gi"
```

### 2. Use MIG for Small Models

```yaml
# Instead of full GPU for small model
migProfile: "1g.5gb"  # 7x better utilization
```

### 3. Specify Topology Requirements

```yaml
# For tensor parallel
topology:
  locality: same-node
  minBandwidth: 600Gi  # NVLink

# For pipeline parallel
topology:
  locality: any  # Can span nodes
```

### 4. Enable Cost Optimization

```yaml
costOptimization:
  enabled: true
  spotEnabled: true
  sloHeadroomMs: 1000
  fallbackModel: smaller-model
```

### 5. Monitor and Tune

- Track GPU utilization
- Monitor scheduling latency
- Adjust topology requirements
- Optimize for your workload

## Advanced Topics

### Custom Scheduler Plugins

Extend the scheduler:

```go
type CustomPlugin struct{}

func (p *CustomPlugin) Name() string {
    return "custom-scorer"
}

func (p *CustomPlugin) Score(ctx context.Context, 
    pod *v1.Pod, node *v1.Node) (int64, error) {
    // Custom scoring logic
    score := calculateCustomScore(pod, node)
    return score, nil
}

// Register plugin
scheduler.RegisterPlugin(&CustomPlugin{})
```

### Preemption

Handle resource contention:

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: high-priority-pool
spec:
  scheduling:
    priority: 1000  # Higher = more important
    
    preemption:
      enabled: true
      policy: lowest-priority-first
```

## Troubleshooting

### Pods Not Scheduling

**Symptom**: Pods stuck in Pending

**Checks**:
1. GPU availability: `kubectl describe nodes | grep -A5 nvidia.com/gpu`
2. Topology constraints: Check if requirements too strict
3. MIG configuration: Verify MIG profiles match
4. Resource requests: Check if requests exceed capacity

**Solution**:
```bash
# Relax topology requirements or add more nodes
kubectl scale deployment gpu-pool --replicas=+1
```

### Poor GPU Utilization

**Symptom**: GPUs underutilized

**Checks**:
1. MIG opportunities: Can smaller models use MIG?
2. Bin-packing: Are pods too scattered?
3. Prewarming: Is warm pool too large?

**Solution**:
```yaml
# Enable MIG for better packing
migProfile: "1g.5gb"

# Reduce warm pool
prewarmPercent: 10
```

### High Scheduling Latency

**Symptom**: Slow pod placement

**Checks**:
1. Node count: Too many nodes to score?
2. Plugin complexity: Custom plugins too slow?
3. Cache misses: Model cache not warmed?

**Solution**:
```yaml
# Reduce scoring complexity
scheduling:
  fastScheduling:
    enabled: true
    maxNodesToScore: 50
```

## References

- [Kubernetes Scheduler Framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/)
- [GPU Operator Documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)
- [MIG User Guide](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/)
