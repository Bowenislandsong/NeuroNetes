# Operations Guide

Deployment, configuration, and maintenance guide for NeuroNetes.

## Prerequisites

### Kubernetes Cluster

- Kubernetes 1.25+
- NVIDIA GPU Operator
- Metrics Server
- Cert Manager (for webhooks)

### Tools

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install kustomize
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
```

## Installation

### Method 1: kubectl

```bash
# Install CRDs
kubectl apply -f https://github.com/bowenislandsong/neuronetes/releases/latest/download/crds.yaml

# Install controllers
kubectl apply -f https://github.com/bowenislandsong/neuronetes/releases/latest/download/neuronetes.yaml

# Verify installation
kubectl get pods -n neuronetes-system
```

### Method 2: Helm

```bash
# Add helm repository
helm repo add neuronetes https://bowenislandsong.github.io/neuronetes
helm repo update

# Install
helm install neuronetes neuronetes/neuronetes \
  --namespace neuronetes-system \
  --create-namespace

# Verify
helm status neuronetes -n neuronetes-system
```

### Method 3: From Source

```bash
# Clone repository
git clone https://github.com/bowenislandsong/neuronetes.git
cd neuronetes

# Install dependencies
make deps

# Generate manifests
make manifests

# Install CRDs
make install

# Deploy controllers
make deploy
```

## Configuration

### Controller Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: neuronetes-config
  namespace: neuronetes-system
data:
  # Controller settings
  controller.yaml: |
    leaderElection: true
    metricsBindAddress: ":8080"
    healthProbeBindAddress: ":8081"
    
    # Reconcile intervals
    syncPeriod: 30s
    
    # Concurrent reconciles
    maxConcurrentReconciles: 5
  
  # Scheduler settings
  scheduler.yaml: |
    # GPU scheduling
    gpuTopologyWeight: 0.25
    modelCacheWeight: 0.20
    costWeight: 0.15
    dataLocalityWeight: 0.10
    
    # Scheduling intervals
    schedulingInterval: 10s
    
    # Node scoring timeout
    scoringTimeout: 5s
  
  # Autoscaler settings
  autoscaler.yaml: |
    # Metrics collection
    metricsInterval: 15s
    metricsRetention: 5m
    
    # Scaling decisions
    decisionInterval: 30s
    stabilizationWindow: 60s
    
    # Warm pool
    warmPoolCheckInterval: 10s
```

### Resource Limits

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: neuronetes-quota
  namespace: neuronetes-system
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
```

## Monitoring

### Prometheus

```bash
# Install Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Configure ServiceMonitor
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: neuronetes
  namespace: neuronetes-system
spec:
  selector:
    matchLabels:
      app: neuronetes
  endpoints:
  - port: metrics
    interval: 15s
EOF
```

### Grafana Dashboards

```bash
# Import dashboards
kubectl create configmap neuronetes-dashboards \
  --from-file=dashboards/ \
  -n monitoring

# Label for auto-import
kubectl label configmap neuronetes-dashboards \
  grafana_dashboard=1 \
  -n monitoring
```

## Backup and Recovery

### CRD Backup

```bash
# Backup all CRDs
kubectl get models,agentclasses,agentpools,toolbindings \
  --all-namespaces -o yaml > neuronetes-backup.yaml

# Restore
kubectl apply -f neuronetes-backup.yaml
```

### State Backup

```bash
# Backup controller state
kubectl get configmap -n neuronetes-system -o yaml > state-backup.yaml

# Backup metrics
# Configure Prometheus long-term storage
```

## Upgrades

### Rolling Upgrade

```bash
# Check current version
kubectl get deployment -n neuronetes-system \
  -o jsonpath='{.items[0].spec.template.spec.containers[0].image}'

# Update to new version
kubectl set image deployment/neuronetes-controller \
  manager=ghcr.io/bowenislandsong/neuronetes:v0.2.0 \
  -n neuronetes-system

# Watch rollout
kubectl rollout status deployment/neuronetes-controller \
  -n neuronetes-system
```

### Helm Upgrade

```bash
# Update helm repo
helm repo update

# Upgrade
helm upgrade neuronetes neuronetes/neuronetes \
  --namespace neuronetes-system \
  --version 0.2.0

# Verify
helm history neuronetes -n neuronetes-system
```

### Rollback

```bash
# Kubectl rollback
kubectl rollout undo deployment/neuronetes-controller \
  -n neuronetes-system

# Helm rollback
helm rollback neuronetes -n neuronetes-system
```

## Security

### RBAC

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: neuronetes-user
rules:
- apiGroups: ["neuronetes.io"]
  resources: ["models", "agentclasses", "agentpools", "toolbindings"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: neuronetes-controller
  namespace: neuronetes-system
spec:
  podSelector:
    matchLabels:
      app: neuronetes
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
```

### Pod Security

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: neuronetes-controller
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 2000
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: manager
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      readOnlyRootFilesystem: true
```

## Troubleshooting

### Controller Issues

```bash
# Check controller logs
kubectl logs -n neuronetes-system \
  deployment/neuronetes-controller -f

# Check controller status
kubectl get deployment -n neuronetes-system

# Describe pod
kubectl describe pod -n neuronetes-system \
  -l app=neuronetes
```

### CRD Issues

```bash
# Validate CRD
kubectl get crd models.neuronetes.io -o yaml

# Check CRD status
kubectl get models --all-namespaces

# Describe resource
kubectl describe model llama-3-70b
```

### Performance Issues

```bash
# Check metrics
kubectl top pods -n neuronetes-system

# Check resource limits
kubectl describe resourcequota -n neuronetes-system

# Increase resources
kubectl patch deployment neuronetes-controller \
  -n neuronetes-system \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"manager","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

## High Availability

### Multiple Replicas

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neuronetes-controller
  namespace: neuronetes-system
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: neuronetes
            topologyKey: kubernetes.io/hostname
```

### Leader Election

Leader election is enabled by default:

```yaml
# Check leader
kubectl get configmap -n neuronetes-system \
  neuronetes-leader -o yaml

# Current leader
kubectl get endpoints -n neuronetes-system
```

## Disaster Recovery

### Disaster Recovery Plan

1. **Backup regularly**
   - CRDs and resources
   - Controller configuration
   - Metrics and logs

2. **Document infrastructure**
   - Cluster configuration
   - Network setup
   - Storage configuration

3. **Test recovery procedures**
   - Regular DR drills
   - Recovery time objectives
   - Recovery point objectives

4. **Monitor continuously**
   - Health checks
   - Performance metrics
   - Alerting

### Recovery Procedure

```bash
# 1. Restore cluster
kubectl apply -f cluster-backup.yaml

# 2. Install NeuroNetes
kubectl apply -f neuronetes.yaml

# 3. Restore CRDs
kubectl apply -f neuronetes-backup.yaml

# 4. Verify
kubectl get all -n neuronetes-system
```

## Best Practices

### Resource Management

1. Set appropriate resource limits
2. Use quotas per namespace
3. Monitor resource usage
4. Scale based on metrics

### Security

1. Enable RBAC
2. Use network policies
3. Enable pod security policies
4. Regular security audits

### Monitoring

1. Deploy Prometheus
2. Configure alerting
3. Use Grafana dashboards
4. Log aggregation

### Maintenance

1. Regular backups
2. Test upgrades in staging
3. Monitor deprecations
4. Keep documentation updated

## Support

### Logs

```bash
# Controller logs
kubectl logs -n neuronetes-system deployment/neuronetes-controller

# Scheduler logs
kubectl logs -n neuronetes-system deployment/neuronetes-scheduler

# Autoscaler logs
kubectl logs -n neuronetes-system deployment/neuronetes-autoscaler
```

### Debugging

```bash
# Enable debug logging
kubectl set env deployment/neuronetes-controller \
  LOG_LEVEL=debug \
  -n neuronetes-system

# Get events
kubectl get events --all-namespaces \
  --field-selector involvedObject.apiVersion=neuronetes.io/v1alpha1

# Exec into controller
kubectl exec -it -n neuronetes-system \
  deployment/neuronetes-controller -- /bin/sh
```

### Community Support

- GitHub Issues
- Slack Channel
- Documentation
- Stack Overflow (tag: neuronetes)
