# Azure Deployment Guide

This guide covers deploying NeuroNetes on Microsoft Azure using AKS (Azure Kubernetes Service).

## Prerequisites

- Azure CLI v2.50+ installed and configured
- kubectl v1.25+ installed
- Helm v3.x installed
- Active Azure subscription
- Required resource providers registered

## Architecture Overview

NeuroNetes on Azure leverages:
- **AKS**: Managed Kubernetes service
- **NC-series VMs**: GPU-enabled virtual machines (A100/V100)
- **Azure Blob Storage**: Model weights storage
- **Azure Managed Disks**: Premium SSD for caching
- **Azure Files**: Shared model cache across nodes
- **Azure Load Balancer**: Layer 4/7 load balancing
- **Azure Monitor**: Metrics, logs, and Application Insights
- **Spot VMs**: Cost optimization for burst capacity

## Quick Start

### 1. Set Up Azure Environment

```bash
# Configure variables
export RESOURCE_GROUP=neuronetes-prod
export LOCATION=eastus
export CLUSTER_NAME=neuronetes-aks
export ACR_NAME=neuronetes$RANDOM

# Login to Azure
az login

# Set default subscription
az account set --subscription "YOUR_SUBSCRIPTION_ID"

# Create resource group
az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION

# Register required providers
az provider register --namespace Microsoft.ContainerService
az provider register --namespace Microsoft.Compute
az provider register --namespace Microsoft.Storage
az provider register --namespace Microsoft.Network
az provider register --namespace Microsoft.OperationalInsights
```

### 2. Create AKS Cluster

```bash
# Create AKS cluster with system node pool
az aks create \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --location $LOCATION \
  --kubernetes-version 1.28.0 \
  --node-count 3 \
  --node-vm-size Standard_D8s_v3 \
  --enable-managed-identity \
  --enable-cluster-autoscaler \
  --min-count 2 \
  --max-count 10 \
  --network-plugin azure \
  --enable-azure-monitor \
  --enable-azure-defender \
  --enable-workload-identity \
  --enable-oidc-issuer \
  --generate-ssh-keys

# Get cluster credentials
az aks get-credentials \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --overwrite-existing

# Verify connection
kubectl get nodes
```

### 3. Add GPU Node Pools

**Standard NC A100 v4 Node Pool**:
```bash
# Add GPU node pool (A100 80GB)
az aks nodepool add \
  --resource-group $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name gpupool \
  --node-count 0 \
  --min-count 0 \
  --max-count 10 \
  --enable-cluster-autoscaler \
  --node-vm-size Standard_NC24ads_A100_v4 \
  --node-taints neuronetes.ai/gpu=true:NoSchedule \
  --labels gpu=nvidia-a100,role=agent \
  --aks-custom-headers UseGPUDedicatedVHD=true

# Install NVIDIA device plugin
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.0/nvidia-device-plugin.yml

# Verify GPU availability
kubectl get nodes -o json | \
  jq -r '.items[] | select(.status.allocatable."nvidia.com/gpu" != null) | .metadata.name'
```

**Spot VM Node Pool for Cost Optimization**:
```bash
# Add spot GPU node pool
az aks nodepool add \
  --resource-group $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name gpuspot \
  --priority Spot \
  --eviction-policy Delete \
  --spot-max-price -1 \
  --node-count 0 \
  --min-count 0 \
  --max-count 20 \
  --enable-cluster-autoscaler \
  --node-vm-size Standard_NC24ads_A100_v4 \
  --node-taints neuronetes.ai/gpu=true:NoSchedule,kubernetes.azure.com/scalesetpriority=spot:NoSchedule \
  --labels gpu=nvidia-a100,role=agent,capacity=spot \
  --aks-custom-headers UseGPUDedicatedVHD=true
```

### 4. Configure Storage

**Azure Blob Storage for Model Weights**:
```bash
# Create storage account
export STORAGE_ACCOUNT=neuronetes$RANDOM
az storage account create \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Premium_LRS \
  --kind BlockBlobStorage \
  --enable-hierarchical-namespace true \
  --allow-blob-public-access false

# Create container for models
az storage container create \
  --name models \
  --account-name $STORAGE_ACCOUNT \
  --auth-mode login

# Enable versioning
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT \
  --enable-versioning true

# Get storage account key
export STORAGE_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name $STORAGE_ACCOUNT \
  --query '[0].value' -o tsv)

# Create Kubernetes secret for storage access
kubectl create secret generic azure-storage-secret \
  --namespace neuronetes-system \
  --from-literal=azurestorageaccountname=$STORAGE_ACCOUNT \
  --from-literal=azurestorageaccountkey=$STORAGE_KEY
```

**Azure Files for Shared Model Cache**:
```bash
# Create premium file share
az storage share-rm create \
  --resource-group $RESOURCE_GROUP \
  --storage-account $STORAGE_ACCOUNT \
  --name modelcache \
  --quota 10240 \
  --enabled-protocols NFS \
  --root-squash NoRootSquash

# Create StorageClass
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: neuronetes-model-cache
provisioner: file.csi.azure.com
parameters:
  skuName: Premium_LRS
  protocol: nfs
reclaimPolicy: Retain
volumeBindingMode: Immediate
allowVolumeExpansion: true
mountOptions:
  - nfsvers=4.1
  - hard
  - timeo=600
  - retrans=2
EOF
```

**Azure Managed Identity for Storage Access**:
```bash
# Create managed identity
az identity create \
  --name neuronetes-storage-identity \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

# Get identity details
export IDENTITY_CLIENT_ID=$(az identity show \
  --name neuronetes-storage-identity \
  --resource-group $RESOURCE_GROUP \
  --query 'clientId' -o tsv)

export IDENTITY_RESOURCE_ID=$(az identity show \
  --name neuronetes-storage-identity \
  --resource-group $RESOURCE_GROUP \
  --query 'id' -o tsv)

# Assign Storage Blob Data Contributor role
export STORAGE_ACCOUNT_ID=$(az storage account show \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --query 'id' -o tsv)

az role assignment create \
  --assignee $IDENTITY_CLIENT_ID \
  --role "Storage Blob Data Contributor" \
  --scope $STORAGE_ACCOUNT_ID

# Create federated identity credential for Workload Identity
export AKS_OIDC_ISSUER=$(az aks show \
  --name $CLUSTER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "oidcIssuerProfile.issuerUrl" -o tsv)

az identity federated-credential create \
  --name neuronetes-federated-credential \
  --identity-name neuronetes-storage-identity \
  --resource-group $RESOURCE_GROUP \
  --issuer $AKS_OIDC_ISSUER \
  --subject system:serviceaccount:neuronetes-system:neuronetes-controller

# Create Kubernetes ServiceAccount
kubectl create namespace neuronetes-system
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: neuronetes-controller
  namespace: neuronetes-system
  annotations:
    azure.workload.identity/client-id: $IDENTITY_CLIENT_ID
  labels:
    azure.workload.identity/use: "true"
EOF
```

### 5. Install NeuroNetes

```bash
# Clone repository
git clone https://github.com/Bowenislandsong/NeuroNetes.git
cd NeuroNetes

# Install CRDs
kubectl apply -f config/crd/

# Deploy controllers
kubectl apply -f config/deploy/

# Verify installation
kubectl get pods -n neuronetes-system
kubectl get crds | grep neuronetes.ai
```

### 6. Configure Observability

**Azure Monitor Integration**:
```bash
# Enable Container Insights
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons monitoring

# Create Log Analytics workspace (if not exists)
az monitor log-analytics workspace create \
  --resource-group $RESOURCE_GROUP \
  --workspace-name neuronetes-logs \
  --location $LOCATION

# Get workspace ID
export WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group $RESOURCE_GROUP \
  --workspace-name neuronetes-logs \
  --query 'id' -o tsv)

# Configure diagnostic settings
az monitor diagnostic-settings create \
  --name neuronetes-diagnostics \
  --resource $CLUSTER_NAME \
  --resource-group $RESOURCE_GROUP \
  --resource-type Microsoft.ContainerService/managedClusters \
  --workspace $WORKSPACE_ID \
  --logs '[{"category":"kube-apiserver","enabled":true},{"category":"kube-controller-manager","enabled":true},{"category":"kube-scheduler","enabled":true}]' \
  --metrics '[{"category":"AllMetrics","enabled":true}]'
```

**Prometheus and Grafana**:
```bash
# Install kube-prometheus-stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set grafana.enabled=true \
  --set grafana.adminPassword=admin

# Access Grafana
kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80
```

### 7. Deploy Sample Agent

```bash
# Create Model with Azure Blob Storage
cat <<EOF | kubectl apply -f -
apiVersion: neuronetes.ai/v1alpha1
kind: Model
metadata:
  name: llama-3-70b
  namespace: default
spec:
  weightsURI: https://${STORAGE_ACCOUNT}.blob.core.windows.net/models/llama-3-70b/
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

**Multi-Zone Deployment**:
```bash
# Create cluster with availability zones
az aks create \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --location $LOCATION \
  --zones 1 2 3 \
  --node-count 3 \
  --enable-cluster-autoscaler \
  --min-count 3 \
  --max-count 15
```

**Pod Disruption Budgets**:
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: neuronetes-controller-pdb
  namespace: neuronetes-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      control-plane: controller-manager
```

### Cost Optimization

**Spot VM Configuration**:
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
    spotAllocation:
      maxSpotPercentage: 70
      fallbackToRegular: true
      sloHeadroomMs: 200
  
  nodeSelector:
    capacity: spot
  
  tolerations:
  - key: neuronetes.ai/gpu
    operator: Equal
    value: "true"
    effect: NoSchedule
  - key: kubernetes.azure.com/scalesetpriority
    operator: Equal
    value: "spot"
    effect: NoSchedule
```

**Azure Reservations**:
```bash
# Purchase 1-year reservation for baseline capacity
az reservations reservation-order purchase \
  --reservation-order-id ORDER_ID \
  --sku Standard_NC24ads_A100_v4 \
  --location $LOCATION \
  --quantity 2 \
  --term P1Y
```

### Security Best Practices

**Azure Policy Integration**:
```bash
# Enable Azure Policy for AKS
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons azure-policy

# Assign built-in policy initiative
az policy assignment create \
  --name 'aks-security-baseline' \
  --display-name 'AKS Security Baseline' \
  --scope "/subscriptions/SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP" \
  --policy-set-definition '/providers/Microsoft.Authorization/policySetDefinitions/a8640138-9b0a-4a28-b8cb-1666c838647d'
```

**Private Cluster**:
```bash
# Create private AKS cluster
az aks create \
  --resource-group $RESOURCE_GROUP \
  --name neuronetes-private \
  --location $LOCATION \
  --enable-private-cluster \
  --private-dns-zone system \
  --network-plugin azure
```

**Azure Key Vault Integration**:
```bash
# Enable Key Vault Secrets Provider
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons azure-keyvault-secrets-provider

# Create Key Vault
az keyvault create \
  --name neuronetes-kv-$RANDOM \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --enable-rbac-authorization

# Grant access to managed identity
az role assignment create \
  --assignee $IDENTITY_CLIENT_ID \
  --role "Key Vault Secrets User" \
  --scope $(az keyvault show --name neuronetes-kv-$RANDOM --query id -o tsv)
```

### Monitoring & Alerts

**Azure Monitor Alerts**:
```bash
# Create action group
az monitor action-group create \
  --name neuronetes-alerts \
  --resource-group $RESOURCE_GROUP \
  --short-name nn-alerts \
  --email-receiver name=admin email=admin@example.com

# Create metric alert for GPU utilization
az monitor metrics alert create \
  --name high-gpu-utilization \
  --resource-group $RESOURCE_GROUP \
  --scopes "/subscriptions/SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP" \
  --condition "avg Percentage GPU > 90" \
  --window-size 5m \
  --evaluation-frequency 1m \
  --action neuronetes-alerts

# Create log alert for errors
az monitor log-analytics query \
  --workspace $WORKSPACE_ID \
  --analytics-query 'ContainerLog | where LogEntry contains "error" | summarize count() by Computer' \
  --output table
```

**Application Insights**:
```bash
# Create Application Insights instance
az monitor app-insights component create \
  --app neuronetes-insights \
  --location $LOCATION \
  --resource-group $RESOURCE_GROUP \
  --workspace $WORKSPACE_ID

# Get instrumentation key
export APPINSIGHTS_KEY=$(az monitor app-insights component show \
  --app neuronetes-insights \
  --resource-group $RESOURCE_GROUP \
  --query 'instrumentationKey' -o tsv)

# Configure controllers to use App Insights
kubectl create secret generic appinsights-key \
  --namespace neuronetes-system \
  --from-literal=key=$APPINSIGHTS_KEY
```

## Troubleshooting

### GPU Nodes Not Ready

```bash
# Check NVIDIA device plugin
kubectl logs -n kube-system -l name=nvidia-device-plugin-ds

# Verify GPU drivers
kubectl run gpu-test --rm -it \
  --image=nvidia/cuda:11.8.0-base-ubuntu22.04 \
  --limits=nvidia.com/gpu=1 \
  -- nvidia-smi

# Check node labels
kubectl get nodes --show-labels | grep gpu
```

### Storage Access Issues

```bash
# Test Azure Blob access
kubectl run azure-cli --rm -it \
  --serviceaccount=neuronetes-controller \
  --image=mcr.microsoft.com/azure-cli:latest \
  -- az storage blob list --container-name models --account-name $STORAGE_ACCOUNT

# Check managed identity
az identity show \
  --name neuronetes-storage-identity \
  --resource-group $RESOURCE_GROUP
```

### High Network Egress Costs

```bash
# Use Azure CDN for model distribution
az cdn profile create \
  --name neuronetes-cdn \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_Microsoft

# Create CDN endpoint
az cdn endpoint create \
  --name neuronetes-models \
  --profile-name neuronetes-cdn \
  --resource-group $RESOURCE_GROUP \
  --origin ${STORAGE_ACCOUNT}.blob.core.windows.net \
  --origin-host-header ${STORAGE_ACCOUNT}.blob.core.windows.net
```

## Cost Estimation

**Monthly costs** (East US, approximate):

| Component | Configuration | Monthly Cost |
|-----------|--------------|--------------|
| AKS Control Plane | Free tier | $0 |
| Standard_D8s_v3 (3) | On-demand | $875 |
| NC24ads_A100_v4 (2) | On-demand | $27,000 |
| NC24ads_A100_v4 (5) | Spot (avg) | $6,750 |
| Blob Storage (5TB) | Premium | $683 |
| Azure Files (10TB) | Premium NFS | $15,360 |
| Load Balancer | Standard | $22 |
| Egress (1TB) | Inter-region | $87 |
| Azure Monitor | Standard | $200 |
| **Total** | | **~$50,977** |

**Savings with Spot VMs**: ~75% on GPU compute

## Best Practices

1. **Use Spot VMs** for 60-80% of GPU capacity
2. **Enable cluster autoscaler** for dynamic scaling
3. **Deploy across availability zones** for HA
4. **Use Workload Identity** for secure access
5. **Enable Azure Policy** for governance
6. **Configure NSGs** for network security
7. **Use Azure Monitor** for comprehensive observability
8. **Set up cost alerts** in Azure Cost Management
9. **Use Azure Reservations** for baseline capacity
10. **Enable Azure Defender** for security

## Next Steps

- Configure [observability stack](../observability.md)
- Set up [multi-region deployment](#multi-region)
- Implement [disaster recovery](../operations.md#disaster-recovery)
- Review [security best practices](../security.md)

## References

- [AKS Best Practices](https://learn.microsoft.com/en-us/azure/aks/best-practices)
- [GPU Support in AKS](https://learn.microsoft.com/en-us/azure/aks/gpu-cluster)
- [Workload Identity](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview)
- [Azure Monitor for Containers](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/container-insights-overview)
