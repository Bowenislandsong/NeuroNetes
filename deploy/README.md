# NeuroNetes Deployment Configurations

This directory contains ready-to-use deployment configurations for different environments.

## Quick Start

### Using Helm (Recommended)

```bash
# Install using Helm chart
helm install neuronetes ./charts/neuronetes \
  --namespace neuronetes-system \
  --create-namespace

# With monitoring enabled
helm install neuronetes ./charts/neuronetes \
  --namespace neuronetes-system \
  --create-namespace \
  --set metrics.serviceMonitor.enabled=true
```

### Using Quickstart Script

```bash
# Basic installation
./scripts/quickstart.sh

# With Kind cluster, monitoring, and samples
./scripts/quickstart.sh -k -m -s
```

### Using kubectl with Kustomize

```bash
# For AWS EKS
kubectl apply -k deploy/eks/

# For Google GKE
kubectl apply -k deploy/gke/

# For Azure AKS
kubectl apply -k deploy/aks/

# For on-premises
kubectl apply -k deploy/onprem/
```

## Cloud Provider Configurations

### AWS EKS (`deploy/eks/`)

Features:
- Multi-AZ deployment with pod anti-affinity
- IRSA (IAM Roles for Service Accounts) support
- EBS gp3 storage integration
- Spot instance tolerations

Prerequisites:
- EKS cluster with GPU node group
- NVIDIA device plugin installed
- (Optional) S3 bucket for model storage
- (Optional) EFS for shared model cache

```bash
# Deploy to EKS
kubectl apply -k deploy/eks/

# Configure IRSA for S3 access
kubectl annotate serviceaccount neuronetes-controller \
  -n neuronetes-system \
  eks.amazonaws.com/role-arn=arn:aws:iam::ACCOUNT_ID:role/neuronetes-controller
```

### Google GKE (`deploy/gke/`)

Features:
- Regional deployment with zone spread
- Workload Identity support
- GCS storage integration
- Preemptible VM tolerations

Prerequisites:
- GKE cluster with GPU node pool
- NVIDIA GPU drivers installed
- (Optional) Cloud Storage bucket for models
- (Optional) Filestore for shared cache

```bash
# Deploy to GKE
kubectl apply -k deploy/gke/

# Configure Workload Identity
kubectl annotate serviceaccount neuronetes-controller \
  -n neuronetes-system \
  iam.gke.io/gcp-service-account=neuronetes-storage@PROJECT_ID.iam.gserviceaccount.com
```

### Azure AKS (`deploy/aks/`)

Features:
- Availability zone deployment
- Azure AD Workload Identity support
- Azure Blob storage integration
- Spot VM tolerations

Prerequisites:
- AKS cluster with GPU node pool
- NVIDIA device plugin installed
- (Optional) Storage account for models
- (Optional) Azure Files for shared cache

```bash
# Deploy to AKS
kubectl apply -k deploy/aks/

# Configure Workload Identity
kubectl annotate serviceaccount neuronetes-controller \
  -n neuronetes-system \
  azure.workload.identity/client-id=CLIENT_ID
```

### On-Premises (`deploy/onprem/`)

Features:
- Lower resource requirements
- Local storage configuration
- No cloud-specific features
- GPU topology scheduling

Prerequisites:
- Kubernetes cluster (v1.25+)
- GPU nodes with NVIDIA drivers
- Local storage for model cache

```bash
# Deploy on-premises
kubectl apply -k deploy/onprem/

# Update storage configuration first
# Edit deploy/onprem/storage.yaml with your paths and node names
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENABLE_TOKEN_AUTOSCALING` | Token-aware HPA | `true` |
| `ENABLE_GPU_TOPOLOGY_SCHEDULING` | GPU topology-aware scheduling | `true` |
| `ENABLE_SESSION_AFFINITY` | Sticky session routing | `true` |
| `ENABLE_WARM_POOLS` | Warm pool management | `true` |
| `ENABLE_COST_OPTIMIZATION` | Cost optimization (cloud only) | `true` |

### Customizing Deployments

Create a `kustomization.yaml` in your own directory:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - github.com/Bowenislandsong/NeuroNetes/deploy/eks?ref=v0.1.0

namespace: my-namespace

patches:
  - patch: |-
      - op: replace
        path: /spec/replicas
        value: 3
    target:
      kind: Deployment
      name: neuronetes-controller-manager
```

## Monitoring

All deployments include Prometheus ServiceMonitor resources. To enable:

1. Install Prometheus Operator:
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace
```

2. Deploy NeuroNetes with monitoring:
```bash
kubectl apply -k deploy/eks/
```

3. Access Grafana:
```bash
kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80
# Login: admin/prom-operator
```

4. Import the NeuroNetes dashboard:
```bash
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring
```

## Benchmarking

Run benchmarks against your deployment:

```bash
# Basic benchmark
./scripts/benchmark.sh -p my-agent-pool

# Extended benchmark
./scripts/benchmark.sh -p my-agent-pool -c 50 -d 300
```

## Troubleshooting

### Pods not starting

```bash
# Check events
kubectl describe pod -n neuronetes-system -l app.kubernetes.io/name=neuronetes

# Check logs
kubectl logs -n neuronetes-system -l app.kubernetes.io/name=neuronetes --tail=100
```

### CRDs not installed

```bash
# Manually install CRDs
kubectl apply -f config/crd/

# Verify
kubectl get crds | grep neuronetes.io
```

### GPU scheduling issues

```bash
# Check GPU node labels
kubectl get nodes -l gpu=nvidia-a100 -o wide

# Check NVIDIA device plugin
kubectl get pods -n kube-system -l name=nvidia-device-plugin-ds

# Verify GPU resources
kubectl describe node <gpu-node> | grep nvidia.com/gpu
```

## Support

- [Documentation](https://github.com/Bowenislandsong/NeuroNetes)
- [Issues](https://github.com/Bowenislandsong/NeuroNetes/issues)
- [Cloud Deployment Guides](../docs/cloud-deployment/)
