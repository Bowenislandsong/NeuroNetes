# Google Cloud Platform (GCP) Deployment Guide

This guide covers deploying NeuroNetes on Google Cloud Platform using GKE (Google Kubernetes Engine).

## Prerequisites

- gcloud CLI installed and configured
- kubectl v1.25+ installed
- Helm v3.x installed
- GCP project with billing enabled
- Compute Engine, GKE, and Cloud Storage APIs enabled

## Architecture Overview

NeuroNetes on GCP leverages:
- **GKE**: Managed Kubernetes with autopilot or standard mode
- **A100/H100 GPUs**: Compute Engine GPU instances
- **Cloud Storage**: Model weights storage
- **Persistent Disk**: SSD persistent volumes for caching
- **Filestore**: Shared model cache across nodes
- **Cloud Load Balancer**: Global load balancing with session affinity
- **Cloud Monitoring**: Metrics and logs integration
- **Preemptible VMs**: Cost optimization for burst capacity

## Quick Start

### 1. Set Up GCP Environment

```bash
# Configure variables
export PROJECT_ID=neuronetes-prod
export REGION=us-central1
export ZONE=us-central1-a
export CLUSTER_NAME=neuronetes-cluster

# Set default project and region
gcloud config set project $PROJECT_ID
gcloud config set compute/region $REGION
gcloud config set compute/zone $ZONE

# Enable required APIs
gcloud services enable container.googleapis.com
gcloud services enable compute.googleapis.com
gcloud services enable storage-api.googleapis.com
gcloud services enable file.googleapis.com
gcloud services enable monitoring.googleapis.com
gcloud services enable logging.googleapis.com
```

### 2. Create GKE Cluster

**Option A: Standard Cluster (Recommended for GPU workloads)**

```bash
# Create standard cluster
gcloud container clusters create $CLUSTER_NAME \
  --region $REGION \
  --cluster-version 1.28 \
  --machine-type n1-standard-8 \
  --num-nodes 2 \
  --min-nodes 2 \
  --max-nodes 10 \
  --enable-autoscaling \
  --enable-stackdriver-kubernetes \
  --enable-ip-alias \
  --enable-autorepair \
  --enable-autoupgrade \
  --maintenance-window-start "2024-01-01T00:00:00Z" \
  --maintenance-window-duration 4h \
  --workload-pool=$PROJECT_ID.svc.id.goog \
  --enable-shielded-nodes

# Get cluster credentials
gcloud container clusters get-credentials $CLUSTER_NAME --region $REGION

# Add GPU node pool (A100 80GB)
gcloud container node-pools create gpu-pool-a100 \
  --cluster $CLUSTER_NAME \
  --region $REGION \
  --machine-type a2-highgpu-8g \
  --accelerator type=nvidia-tesla-a100,count=8 \
  --num-nodes 0 \
  --min-nodes 0 \
  --max-nodes 10 \
  --enable-autoscaling \
  --enable-autorepair \
  --node-labels gpu=nvidia-a100,role=agent \
  --node-taints neuronetes.ai/gpu=true:NoSchedule \
  --disk-type pd-ssd \
  --disk-size 200

# Add preemptible GPU pool for cost savings
gcloud container node-pools create gpu-pool-preemptible \
  --cluster $CLUSTER_NAME \
  --region $REGION \
  --machine-type a2-highgpu-4g \
  --accelerator type=nvidia-tesla-a100,count=4 \
  --preemptible \
  --num-nodes 0 \
  --min-nodes 0 \
  --max-nodes 20 \
  --enable-autoscaling \
  --enable-autorepair \
  --node-labels gpu=nvidia-a100,role=agent,capacity=preemptible \
  --node-taints neuronetes.ai/gpu=true:NoSchedule \
  --disk-type pd-ssd \
  --disk-size 200
```

**Option B: GKE Autopilot (Serverless)**

```bash
# Create autopilot cluster
gcloud container clusters create-auto $CLUSTER_NAME \
  --region $REGION \
  --cluster-version 1.28

# Note: GPU support in Autopilot is limited
# Autopilot automatically provisions nodes based on workload requirements
```

### 3. Install GPU Drivers

```bash
# Install NVIDIA GPU device plugin
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml

# Verify GPU nodes
kubectl get nodes -o json | jq '.items[] | {name:.metadata.name, gpus:.status.capacity."nvidia.com/gpu"}'

# Test GPU access
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvidia/cuda:11.8.0-base-ubuntu22.04 \
  --limits=nvidia.com/gpu=1 \
  -- nvidia-smi
```

### 4. Configure Storage

**Create Cloud Storage Bucket**:
```bash
# Create bucket for model weights
export BUCKET_NAME=neuronetes-models-${PROJECT_ID}
gsutil mb -l $REGION gs://$BUCKET_NAME

# Enable versioning
gsutil versioning set on gs://$BUCKET_NAME

# Set lifecycle policy for old versions
cat <<EOF > lifecycle.json
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {
          "numNewerVersions": 3,
          "isLive": false
        }
      }
    ]
  }
}
EOF
gsutil lifecycle set lifecycle.json gs://$BUCKET_NAME

# Create service account for GCS access
gcloud iam service-accounts create neuronetes-storage \
  --display-name "NeuroNetes Storage Access"

# Grant storage permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member "serviceAccount:neuronetes-storage@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role "roles/storage.objectAdmin"

# Create Workload Identity binding
kubectl create serviceaccount neuronetes-controller -n neuronetes-system

gcloud iam service-accounts add-iam-policy-binding \
  neuronetes-storage@${PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[neuronetes-system/neuronetes-controller]"

kubectl annotate serviceaccount neuronetes-controller -n neuronetes-system \
  iam.gke.io/gcp-service-account=neuronetes-storage@${PROJECT_ID}.iam.gserviceaccount.com
```

**Create Filestore for Model Cache**:
```bash
# Create Filestore instance (high-scale tier for better performance)
gcloud filestore instances create neuronetes-cache \
  --zone=$ZONE \
  --tier=HIGH_SCALE_SSD \
  --file-share=name=modelcache,capacity=10Ti \
  --network=name=default

# Get Filestore IP
export FILESTORE_IP=$(gcloud filestore instances describe neuronetes-cache \
  --zone=$ZONE \
  --format="value(networks[0].ipAddresses[0])")

# Create StorageClass
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: neuronetes-model-cache
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: neuronetes-cache-pv
spec:
  capacity:
    storage: 10Ti
  accessModes:
  - ReadWriteMany
  nfs:
    server: $FILESTORE_IP
    path: /modelcache
  storageClassName: neuronetes-model-cache
  mountOptions:
  - hard
  - nfsvers=3
EOF
```

### 5. Install NeuroNetes

```bash
# Clone repository
git clone https://github.com/Bowenislandsong/NeuroNetes.git
cd NeuroNetes

# Create namespace
kubectl create namespace neuronetes-system

# Install CRDs
kubectl apply -f config/crd/

# Deploy controllers
kubectl apply -f config/deploy/

# Verify installation
kubectl get pods -n neuronetes-system
kubectl get crds | grep neuronetes.ai
```

### 6. Configure Observability

```bash
# Install Google Cloud Managed Prometheus
gcloud container clusters update $CLUSTER_NAME \
  --region $REGION \
  --enable-managed-prometheus

# Create PodMonitoring for NeuroNetes
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.googleapis.com/v1
kind: PodMonitoring
metadata:
  name: neuronetes-metrics
  namespace: neuronetes-system
spec:
  selector:
    matchLabels:
      app: neuronetes-controller
  endpoints:
  - port: metrics
    interval: 30s
EOF

# Install Grafana (optional, for local visualization)
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana grafana/grafana \
  --namespace monitoring \
  --create-namespace \
  --set persistence.enabled=true \
  --set persistence.size=10Gi

# Get Grafana admin password
kubectl get secret --namespace monitoring grafana -o jsonpath="{.data.admin-password}" | base64 --decode

# Access Grafana
kubectl port-forward -n monitoring svc/grafana 3000:80
```

### 7. Deploy Sample Agent

```bash
# Create Model with GCS weights
cat <<EOF | kubectl apply -f -
apiVersion: neuronetes.ai/v1alpha1
kind: Model
metadata:
  name: llama-3-70b
  namespace: default
spec:
  weightsURI: gs://$BUCKET_NAME/llama-3-70b/
  size: 140Gi
  quantization: int4
  framework: vllm
  shardSpec:
    count: 8
    strategy: tensor-parallel
    topology:
      locality: same-node
      minBandwidth: 600Gi
  cachePolicy:
    storageClass: neuronetes-model-cache
    priority: high
    pinDuration: 2h
EOF

# Deploy AgentPool
kubectl apply -f config/samples/agentpool_sample.yaml

# Check status
kubectl get models,agentpools,pods -o wide
```

## Production Configuration

### High Availability

```yaml
# config/deploy/manager-ha.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neuronetes-controller-manager
  namespace: neuronetes-system
spec:
  replicas: 3
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                control-plane: controller-manager
            topologyKey: topology.kubernetes.io/zone
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: DoNotSchedule
        labelSelector:
          matchLabels:
            control-plane: controller-manager
```

### Multi-Region Deployment

```bash
# Create clusters in multiple regions
REGIONS=("us-central1" "us-east1" "europe-west1")

for REGION in "${REGIONS[@]}"; do
  gcloud container clusters create neuronetes-$REGION \
    --region $REGION \
    --cluster-version 1.28 \
    --machine-type n1-standard-8 \
    --num-nodes 2
done

# Configure multi-cluster ingress
gcloud compute addresses create neuronetes-global-ip \
  --ip-version=IPV4 \
  --global

# Set up Cloud DNS for global load balancing
gcloud dns managed-zones create neuronetes-zone \
  --dns-name="neuronetes.example.com." \
  --description="NeuroNetes multi-region"
```

### Cost Optimization

**Use Preemptible VMs**:
```yaml
apiVersion: neuronetes.ai/v1alpha1
kind: AgentPool
metadata:
  name: cost-optimized-pool
spec:
  minReplicas: 2
  maxReplicas: 50
  prewarmPercent: 10
  
  costOptimization:
    enabled: true
    preemptibleAllocation:
      maxPreemptiblePercentage: 70
      fallbackToRegular: true
      sloHeadroomMs: 200
  
  nodeSelector:
    capacity: preemptible
  
  tolerations:
  - key: neuronetes.ai/gpu
    operator: Equal
    value: "true"
    effect: NoSchedule
```

**Committed Use Discounts**:
```bash
# Purchase 1-year commitment for baseline capacity
gcloud compute commitments create neuronetes-commitment \
  --region $REGION \
  --resources-accelerator type=nvidia-tesla-a100,count=16 \
  --plan 12-month
```

### Security Best Practices

**Binary Authorization**:
```bash
# Enable Binary Authorization
gcloud services enable binaryauthorization.googleapis.com

# Create policy
cat <<EOF > binauth-policy.yaml
admissionWhitelistPatterns:
- namePattern: gcr.io/${PROJECT_ID}/*
- namePattern: ghcr.io/bowenislandsong/*
defaultAdmissionRule:
  evaluationMode: REQUIRE_ATTESTATION
  enforcementMode: ENFORCED_BLOCK_AND_AUDIT_LOG
  requireAttestationsBy:
  - projects/${PROJECT_ID}/attestors/neuronetes-attestor
globalPolicyEvaluationMode: ENABLE
EOF

gcloud container binauthz policy import binauth-policy.yaml

# Update cluster to enforce policy
gcloud container clusters update $CLUSTER_NAME \
  --region $REGION \
  --enable-binauthz
```

**Workload Identity** (already configured above):
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: neuronetes-controller
  namespace: neuronetes-system
  annotations:
    iam.gke.io/gcp-service-account: neuronetes-storage@${PROJECT_ID}.iam.gserviceaccount.com
```

**Private GKE Cluster**:
```bash
# Create private cluster
gcloud container clusters create neuronetes-private \
  --region $REGION \
  --enable-private-nodes \
  --enable-private-endpoint \
  --master-ipv4-cidr 172.16.0.0/28 \
  --enable-ip-alias \
  --create-subnetwork=""
```

### Monitoring & Alerts

**Cloud Monitoring Alerts**:
```bash
# Create alert policy for high GPU utilization
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="High GPU Utilization" \
  --condition-display-name="GPU > 90%" \
  --condition-threshold-value=0.9 \
  --condition-threshold-duration=300s \
  --condition-threshold-comparison=COMPARISON_GT \
  --condition-threshold-aggregations-alignment-period=60s \
  --condition-threshold-aggregations-per-series-aligner=ALIGN_MEAN \
  --condition-threshold-filter='resource.type="k8s_node" AND metric.type="custom.googleapis.com/agent/gpu_utilization"'

# Create alert for token cost
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="High Token Cost" \
  --condition-display-name="Cost > $100/hour" \
  --condition-threshold-value=100 \
  --condition-threshold-duration=3600s \
  --condition-threshold-comparison=COMPARISON_GT \
  --condition-threshold-filter='metric.type="custom.googleapis.com/agent/cost_per_hour"'
```

**Custom Metrics**:
```go
// Export custom metrics to Cloud Monitoring
import (
    monitoring "cloud.google.com/go/monitoring/apiv3"
    "google.golang.org/genproto/googleapis/api/metric"
    monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

{% raw %}
// Example: Report token throughput
func reportTokenThroughput(tokensPerSecond float64) error {
    ctx := context.Background()
    client, _ := monitoring.NewMetricClient(ctx)
    
    req := &monitoringpb.CreateTimeSeriesRequest{
        Name: "projects/" + projectID,
        TimeSeries: []*monitoringpb.TimeSeries{{
            Metric: &metric.Metric{
                Type: "custom.googleapis.com/agent/tokens_per_second",
                Labels: map[string]string{
                    "model": "llama-3-70b",
                },
            },
            Points: []*monitoringpb.Point{{
                Interval: &monitoringpb.TimeInterval{
                    EndTime: timestamppb.Now(),
                },
                Value: &monitoringpb.TypedValue{
                    Value: &monitoringpb.TypedValue_DoubleValue{
                        DoubleValue: tokensPerSecond,
                    },
                },
            }},
        }},
    }
    
    return client.CreateTimeSeries(ctx, req)
}
```
{% endraw %}

## Troubleshooting

### GPU Not Available

```bash
# Check GPU driver installation
kubectl describe nodes | grep nvidia.com/gpu

# View GPU installer logs
kubectl logs -n kube-system -l name=nvidia-driver-installer

# Restart GPU driver installer
kubectl delete pods -n kube-system -l name=nvidia-driver-installer
```

### Model Loading from GCS Fails

```bash
# Test GCS access from pod
kubectl run gcs-test --rm -it \
  --serviceaccount=neuronetes-controller \
  --image=google/cloud-sdk:slim \
  -- gsutil ls gs://$BUCKET_NAME/

# Check Workload Identity binding
gcloud iam service-accounts get-iam-policy \
  neuronetes-storage@${PROJECT_ID}.iam.gserviceaccount.com
```

### High Network Costs

```bash
# Use GCS regional buckets in same region as cluster
gsutil mb -l $REGION gs://$BUCKET_NAME

# Enable Cloud CDN for model distribution
gcloud compute backend-services update BACKEND_NAME \
  --enable-cdn \
  --global
```

## Cost Estimation

**Monthly costs** (us-central1, approximate):

| Component | Configuration | Monthly Cost |
|-----------|--------------|--------------|
| GKE Cluster | Standard | Free |
| n1-standard-8 (3) | On-demand | $730 |
| a2-highgpu-8g (2) | On-demand | $23,000 |
| a2-highgpu-4g (5) | Preemptible | $3,450 |
| Cloud Storage (5TB) | Standard | $100 |
| Filestore High-Scale (10TB) | SSD | $20,480 |
| Load Balancer | Global | $18 |
| Egress (1TB) | Inter-region | $120 |
| **Total** | | **~$47,898** |

**Savings with preemptible VMs**: ~70% on compute

## Best Practices

1. **Use preemptible VMs** for 60-80% of GPU capacity
2. **Enable cluster autoscaler** and node auto-provisioning
3. **Use regional clusters** for high availability
4. **Configure Workload Identity** for secure service access
5. **Enable Binary Authorization** for image security
6. **Use Filestore** for shared model cache
7. **Set up budget alerts** in Cloud Billing
8. **Use committed use discounts** for baseline capacity
9. **Monitor with Cloud Monitoring** and custom metrics
10. **Use Cloud Armor** for DDoS protection on ingress

## Next Steps

- Configure [observability stack](../observability.md)
- Set up [multi-region deployment](#multi-region)
- Implement [cost optimization](#cost-optimization)
- Review [security best practices](../security.md)

## References

- [GKE Best Practices](https://cloud.google.com/kubernetes-engine/docs/best-practices)
- [GPU on GKE](https://cloud.google.com/kubernetes-engine/docs/how-to/gpus)
- [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [GKE Cost Optimization](https://cloud.google.com/architecture/best-practices-for-running-cost-effective-kubernetes-applications-on-gke)
