# AWS Deployment Guide

This guide covers deploying NeuroNetes on Amazon Web Services (AWS) using EKS (Elastic Kubernetes Service).

## Prerequisites

- AWS CLI v2.x installed and configured
- kubectl v1.25+ installed
- eksctl v0.150+ installed
- Helm v3.x installed
- AWS account with appropriate permissions

## Architecture Overview

NeuroNetes on AWS leverages:
- **EKS**: Managed Kubernetes control plane
- **EC2 P4/P5 instances**: GPU nodes for LLM workloads
- **S3**: Model weights storage
- **EBS gp3/io2**: Persistent volumes for caching
- **EFS**: Shared model cache across nodes
- **Application Load Balancer**: Ingress with sticky sessions
- **CloudWatch**: Metrics and logs integration
- **Spot instances**: Cost optimization for burst capacity

## Quick Start

### 1. Create EKS Cluster

```bash
# Configure variables
export CLUSTER_NAME=neuronetes-prod
export REGION=us-west-2
export K8S_VERSION=1.28

# Create cluster with GPU node groups
eksctl create cluster \
  --name $CLUSTER_NAME \
  --region $REGION \
  --version $K8S_VERSION \
  --node-type m5.xlarge \
  --nodes 2 \
  --nodes-min 2 \
  --nodes-max 5 \
  --managed \
  --with-oidc

# Add GPU node group (P4d instances)
eksctl create nodegroup \
  --cluster $CLUSTER_NAME \
  --region $REGION \
  --name gpu-workers-p4d \
  --node-type p4d.24xlarge \
  --nodes 0 \
  --nodes-min 0 \
  --nodes-max 10 \
  --node-labels role=agent,gpu=nvidia-a100 \
  --node-taints neuronetes.ai/gpu=true:NoSchedule \
  --asg-access

# Add spot GPU node group for burst capacity
eksctl create nodegroup \
  --cluster $CLUSTER_NAME \
  --region $REGION \
  --name gpu-workers-spot \
  --node-type p4d.24xlarge \
  --nodes 0 \
  --nodes-min 0 \
  --nodes-max 20 \
  --spot \
  --instance-types p4d.24xlarge,p4de.24xlarge \
  --node-labels role=agent,gpu=nvidia-a100,capacity=spot \
  --node-taints neuronetes.ai/gpu=true:NoSchedule \
  --asg-access
```

### 2. Install GPU Device Plugin

```bash
# Install NVIDIA device plugin
kubectl create -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.0/nvidia-device-plugin.yml

# Verify GPU nodes
kubectl get nodes -l gpu=nvidia-a100
kubectl describe node <node-name> | grep nvidia.com/gpu
```

### 3. Configure Storage

```bash
# Create S3 bucket for model weights
aws s3 mb s3://neuronetes-models-$REGION --region $REGION

# Enable versioning and encryption
aws s3api put-bucket-versioning \
  --bucket neuronetes-models-$REGION \
  --versioning-configuration Status=Enabled

aws s3api put-bucket-encryption \
  --bucket neuronetes-models-$REGION \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "AES256"
      }
    }]
  }'

# Install EFS CSI driver for shared model cache
kubectl apply -k "github.com/kubernetes-sigs/aws-efs-csi-driver/deploy/kubernetes/overlays/stable/?ref=master"

# Create EFS filesystem
export EFS_ID=$(aws efs create-file-system \
  --region $REGION \
  --performance-mode maxIO \
  --throughput-mode provisioned \
  --provisioned-throughput-in-mibps 1024 \
  --encrypted \
  --tags Key=Name,Value=neuronetes-model-cache \
  --query 'FileSystemId' \
  --output text)

# Create mount targets in each AZ
for SUBNET_ID in $(aws ec2 describe-subnets \
  --region $REGION \
  --filters "Name=tag:kubernetes.io/cluster/$CLUSTER_NAME,Values=shared" \
  --query 'Subnets[*].SubnetId' \
  --output text); do
  
  SECURITY_GROUP=$(aws eks describe-cluster \
    --name $CLUSTER_NAME \
    --region $REGION \
    --query 'cluster.resourcesVpcConfig.clusterSecurityGroupId' \
    --output text)
  
  aws efs create-mount-target \
    --region $REGION \
    --file-system-id $EFS_ID \
    --subnet-id $SUBNET_ID \
    --security-groups $SECURITY_GROUP
done

# Create StorageClass for model cache
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: neuronetes-model-cache
provisioner: efs.csi.aws.com
parameters:
  provisioningMode: efs-ap
  fileSystemId: $EFS_ID
  directoryPerms: "700"
  gidRangeStart: "1000"
  gidRangeEnd: "2000"
EOF
```

### 4. Install NeuroNetes

```bash
# Clone repository
git clone https://github.com/Bowenislandsong/NeuroNetes.git
cd NeuroNetes

# Create IAM service account for S3 access
eksctl create iamserviceaccount \
  --name neuronetes-controller \
  --namespace neuronetes-system \
  --cluster $CLUSTER_NAME \
  --region $REGION \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess \
  --approve \
  --override-existing-serviceaccounts

# Install CRDs
kubectl create namespace neuronetes-system
kubectl apply -f config/crd/

# Deploy controllers
kubectl apply -f config/deploy/

# Verify installation
kubectl get pods -n neuronetes-system
kubectl get crds | grep neuronetes
```

### 5. Configure Observability

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false

# Install NeuroNetes ServiceMonitors
kubectl apply -f config/monitoring/

# Access Grafana
kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80
# Login with admin/prom-operator
```

### 6. Deploy Sample Agent

```bash
# Configure Model with S3 weights
cat <<EOF | kubectl apply -f -
apiVersion: neuronetes.ai/v1alpha1
kind: Model
metadata:
  name: llama-3-70b
  namespace: default
spec:
  weightsURI: s3://neuronetes-models-$REGION/llama-3-70b/
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
  status:
    phase: Pending
EOF

# Deploy AgentPool
kubectl apply -f config/samples/agentpool_sample.yaml

# Check status
kubectl get models,agentpools,pods
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
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                control-plane: controller-manager
            topologyKey: kubernetes.io/hostname
      serviceAccountName: neuronetes-controller
      containers:
      - name: manager
        image: ghcr.io/bowenislandsong/neuronetes:v0.1.0
        resources:
          limits:
            cpu: 2000m
            memory: 4Gi
          requests:
            cpu: 500m
            memory: 1Gi
```

### Cost Optimization

**Spot Instance Configuration**:
```yaml
apiVersion: neuronetes.ai/v1alpha1
kind: AgentPool
metadata:
  name: code-assistant-pool
spec:
  minReplicas: 2
  maxReplicas: 50
  prewarmPercent: 10
  
  # Cost optimization with spot instances
  costOptimization:
    enabled: true
    spotAllocation:
      maxSpotPercentage: 70
      fallbackToOnDemand: true
      sloHeadroomMs: 200
    
    # Scale to spot when headroom allows
    scaleToSpot:
      enabled: true
      minHeadroomMs: 500
  
  nodeSelector:
    capacity: spot
  
  tolerations:
  - key: neuronetes.ai/gpu
    operator: Equal
    value: "true"
    effect: NoSchedule
```

**Cluster Autoscaler Configuration**:
```bash
# Install cluster autoscaler
helm repo add autoscaler https://kubernetes.github.io/autoscaler
helm install cluster-autoscaler autoscaler/cluster-autoscaler \
  --namespace kube-system \
  --set autoDiscovery.clusterName=$CLUSTER_NAME \
  --set awsRegion=$REGION \
  --set rbac.serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=arn:aws:iam::ACCOUNT_ID:role/cluster-autoscaler
```

### Security

**Pod Security Standards**:
```bash
# Apply pod security standards
kubectl label namespace neuronetes-system \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

**Network Policies**:
```yaml
# config/security/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: agent-pool-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      neuronetes.ai/component: agent
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          neuronetes.ai/component: ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          neuronetes.ai/component: model-server
    ports:
    - protocol: TCP
      port: 8000
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

### Monitoring & Alerts

**CloudWatch Integration**:
{% raw %}
```bash
# Install CloudWatch Container Insights
curl https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/quickstart/cwagent-fluentd-quickstart.yaml | \
  sed "s/{{cluster_name}}/$CLUSTER_NAME/;s/{{region_name}}/$REGION/" | \
  kubectl apply -f -
```
{% endraw %}

**Custom Metrics**:
```yaml
# config/monitoring/servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
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
    path: /metrics
```

## Troubleshooting

### GPU Nodes Not Ready

```bash
# Check NVIDIA device plugin
kubectl logs -n kube-system -l name=nvidia-device-plugin-ds

# Verify GPU drivers
kubectl run gpu-test --image=nvidia/cuda:11.8.0-base-ubuntu22.04 --rm -it -- nvidia-smi

# Check node labels
kubectl get nodes --show-labels | grep gpu
```

### Model Loading Issues

```bash
# Check S3 access
kubectl run aws-cli --image=amazon/aws-cli:latest --rm -it -- \
  s3 ls s3://neuronetes-models-$REGION/

# Check EFS mount
kubectl exec -it <pod-name> -- df -h | grep efs

# View model controller logs
kubectl logs -n neuronetes-system -l app=neuronetes-controller --tail=100
```

### Performance Issues

```bash
# Check node metrics
kubectl top nodes

# Check pod resource usage
kubectl top pods -n default

# Analyze scheduler decisions
kubectl logs -n neuronetes-system -l app=neuronetes-scheduler | grep "Scheduling decision"
```

## Cost Estimation

**Monthly costs** (us-west-2, approximate):

| Component | Configuration | Monthly Cost |
|-----------|--------------|--------------|
| EKS Control Plane | Standard | $73 |
| m5.xlarge nodes (3) | On-demand | $365 |
| p4d.24xlarge (2) | On-demand | $65,000 |
| p4d.24xlarge (5) | Spot (avg) | $48,750 |
| S3 storage (5TB) | Standard | $115 |
| EFS (10TB) | Provisioned | $6,144 |
| Data transfer | 1TB/month | $90 |
| **Total** | | **~$120,537** |

**Cost optimization with spot**: ~40% savings on GPU compute

## Best Practices

1. **Use spot instances** for 60-80% of GPU capacity
2. **Enable cluster autoscaler** for dynamic scaling
3. **Configure node affinity** to keep warm pool on on-demand
4. **Use EFS for model cache** to minimize load times
5. **Monitor token costs** via CloudWatch custom metrics
6. **Set up budget alerts** in AWS Billing
7. **Use reserved instances** for baseline capacity
8. **Enable S3 lifecycle policies** for old model versions

## Next Steps

- Configure [observability stack](../observability.md)
- Set up [multi-region deployment](#multi-region)
- Implement [disaster recovery](#disaster-recovery)
- Review [security best practices](../security.md)

## References

- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [GPU Operator Documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)
- [Cluster Autoscaler on AWS](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/README.md)
