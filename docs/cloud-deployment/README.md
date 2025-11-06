# Cloud Deployment Guides

NeuroNetes can be deployed on any major cloud platform. Choose your provider:

## Quick Links

- **[AWS Deployment Guide](aws.md)** - Deploy on Amazon EKS with EC2 GPU instances
- **[Google Cloud Deployment Guide](gcp.md)** - Deploy on GKE with A100/H100 GPUs  
- **[Azure Deployment Guide](azure.md)** - Deploy on AKS with NC-series VMs

## Comparison

| Feature | AWS | GCP | Azure |
|---------|-----|-----|-------|
| **Managed Kubernetes** | EKS | GKE | AKS |
| **GPU Options** | P4d (A100), P5 (H100) | A2 (A100), A3 (H100) | NC A100 v4 |
| **Model Storage** | S3 + EFS | Cloud Storage + Filestore | Blob + Azure Files |
| **Monitoring** | CloudWatch | Cloud Monitoring | Azure Monitor |
| **Spot Instances** | EC2 Spot | Preemptible VMs | Spot VMs |
| **Monthly Cost (2x A100)** | ~$65,000 | ~$23,000 | ~$27,000 |
| **Spot Savings** | ~40% | ~70% | ~60% |

## Cost Optimization Strategies

### 1. Spot/Preemptible Instances

Use spot instances for 60-80% of GPU capacity:

```yaml
apiVersion: neuronetes.ai/v1alpha1
kind: AgentPool
spec:
  costOptimization:
    enabled: true
    spotAllocation:
      maxSpotPercentage: 70
      fallbackToOnDemand: true
      sloHeadroomMs: 200
```

### 2. Regional Selection

**GPU availability by region**:

| Region | AWS | GCP | Azure |
|--------|-----|-----|-------|
| US East | ✅ P4d | ✅ A2 | ✅ NC A100 v4 |
| US West | ✅ P4d | ✅ A2 | ✅ NC A100 v4 |
| Europe | ✅ P4d | ✅ A2 | ✅ NC A100 v4 |
| Asia Pacific | Limited | ✅ A2 | Limited |

### 3. Reserved Capacity

Purchase 1-3 year commitments for baseline capacity:
- **AWS**: Savings Plans (up to 72% savings)
- **GCP**: Committed Use Discounts (up to 70% savings)
- **Azure**: Reserved Instances (up to 72% savings)

### 4. Autoscaling

Configure aggressive autoscaling with warm pools:

```yaml
spec:
  minReplicas: 2      # Baseline on-demand
  maxReplicas: 50     # Burst to spot
  prewarmPercent: 10  # Keep 10% warm
```

## Multi-Cloud Strategy

### Active-Active

Deploy across multiple clouds for maximum availability:

```yaml
# Global load balancer routes to closest healthy region
Regions:
  - aws-us-west-2    (Primary)
  - gcp-us-central1  (Secondary)
  - azure-eastus     (DR)
```

### Cost-Optimized

Use cheapest cloud for baseline, burst to others:

```yaml
Baseline: GCP (lowest GPU cost)
Burst: AWS Spot (good availability)
DR: Azure (different failure domain)
```

## Getting Started

1. **Choose your cloud provider** based on:
   - Existing infrastructure
   - GPU availability in your region
   - Cost requirements
   - Compliance needs

2. **Follow the deployment guide**:
   - [AWS Guide](aws.md) - Detailed EKS setup
   - [GCP Guide](gcp.md) - Detailed GKE setup
   - [Azure Guide](azure.md) - Detailed AKS setup

3. **Configure observability**:
   - Enable cloud-native monitoring
   - Install Prometheus + Grafana
   - Import NeuroNetes dashboards

4. **Optimize costs**:
   - Start with spot instances
   - Purchase reservations for baseline
   - Monitor with cost alerts

## Best Practices

### Security
- Use managed identities (Workload Identity/IRSA)
- Enable pod security standards
- Configure network policies
- Enable audit logging

### Reliability
- Deploy across availability zones
- Configure pod disruption budgets
- Set up health checks and readiness probes
- Implement graceful shutdown

### Performance
- Use local SSD for model cache
- Configure CPU pinning for GPU workloads
- Enable huge pages for large models
- Optimize network for multi-GPU

### Monitoring
- Track token costs per tenant
- Monitor GPU utilization
- Alert on SLO breaches
- Dashboard for business metrics

## Migration Between Clouds

Use these strategies to migrate workloads:

1. **Gradual traffic shift**: Route percentage of traffic to new cloud
2. **Session draining**: Wait for active sessions to complete
3. **Model sync**: Ensure model weights are replicated
4. **DNS cutover**: Update DNS to point to new cloud

## Support

For cloud-specific issues:
- **AWS**: [EKS Documentation](https://docs.aws.amazon.com/eks/)
- **GCP**: [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- **Azure**: [AKS Documentation](https://docs.microsoft.com/en-us/azure/aks/)

For NeuroNetes issues:
- [GitHub Issues](https://github.com/Bowenislandsong/NeuroNetes/issues)
- [Community Slack](#)
- [Documentation](../README.md)
