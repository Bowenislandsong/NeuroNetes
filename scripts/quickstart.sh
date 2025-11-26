#!/bin/bash
# NeuroNetes Quick Start Script
# This script helps you quickly deploy NeuroNetes on your Kubernetes cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="neuronetes-system"
RELEASE_NAME="neuronetes"
INSTALL_MONITORING="false"
DEPLOY_SAMPLES="false"
USE_KIND="false"

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                    NeuroNetes Quick Start                      ║"
    echo "║        Agent-Native Kubernetes Framework for AI Workloads      ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -n, --namespace NAME     Namespace for installation (default: neuronetes-system)"
    echo "  -r, --release NAME       Helm release name (default: neuronetes)"
    echo "  -m, --monitoring         Install Prometheus/Grafana monitoring stack"
    echo "  -s, --samples            Deploy sample resources after installation"
    echo "  -k, --kind               Create a local Kind cluster for testing"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Basic installation"
    echo "  $0 -m -s                     # Install with monitoring and samples"
    echo "  $0 -k -m -s                  # Create Kind cluster with full setup"
    echo ""
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--release)
                RELEASE_NAME="$2"
                shift 2
                ;;
            -m|--monitoring)
                INSTALL_MONITORING="true"
                shift
                ;;
            -s|--samples)
                DEPLOY_SAMPLES="true"
                shift
                ;;
            -k|--kind)
                USE_KIND="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                usage
                exit 1
                ;;
        esac
    done
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}Error: kubectl is not installed${NC}"
        echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    echo -e "${GREEN}✓ kubectl found${NC}"
    
    # Check helm
    if ! command -v helm &> /dev/null; then
        echo -e "${RED}Error: helm is not installed${NC}"
        echo "Please install helm: https://helm.sh/docs/intro/install/"
        exit 1
    fi
    echo -e "${GREEN}✓ helm found${NC}"
    
    # Check kind if requested
    if [[ "$USE_KIND" == "true" ]]; then
        if ! command -v kind &> /dev/null; then
            echo -e "${RED}Error: kind is not installed${NC}"
            echo "Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
            exit 1
        fi
        echo -e "${GREEN}✓ kind found${NC}"
    fi
    
    # Check cluster connection (unless creating Kind cluster)
    if [[ "$USE_KIND" != "true" ]]; then
        if ! kubectl cluster-info &> /dev/null; then
            echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
            echo "Please configure kubectl to connect to your cluster"
            exit 1
        fi
        echo -e "${GREEN}✓ Connected to Kubernetes cluster${NC}"
    fi
    
    echo ""
}

# Create Kind cluster
create_kind_cluster() {
    echo -e "${BLUE}Creating Kind cluster...${NC}"
    
    CLUSTER_NAME="neuronetes-quickstart"
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        echo -e "${YELLOW}Kind cluster '${CLUSTER_NAME}' already exists${NC}"
        kind export kubeconfig --name "${CLUSTER_NAME}"
    else
        # Create cluster with custom config
        cat <<EOF | kind create cluster --name "${CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
- role: worker
EOF
    fi
    
    # Wait for cluster to be ready
    echo -e "${BLUE}Waiting for cluster to be ready...${NC}"
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
    
    echo -e "${GREEN}✓ Kind cluster created${NC}"
    echo ""
}

# Install NeuroNetes via Helm
install_neuronetes() {
    echo -e "${BLUE}Installing NeuroNetes...${NC}"
    
    # Get script directory
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    CHART_DIR="${SCRIPT_DIR}/../charts/neuronetes"
    
    # Check if chart exists locally
    if [[ -d "$CHART_DIR" ]]; then
        echo "Using local Helm chart from ${CHART_DIR}"
        CHART_SOURCE="$CHART_DIR"
    else
        echo "Using Helm chart from repository"
        helm repo add neuronetes https://bowenislandsong.github.io/NeuroNetes/charts || true
        helm repo update
        CHART_SOURCE="neuronetes/neuronetes"
    fi
    
    # Install or upgrade
    helm upgrade --install "${RELEASE_NAME}" "${CHART_SOURCE}" \
        --namespace "${NAMESPACE}" \
        --create-namespace \
        --wait \
        --timeout 5m
    
    echo -e "${GREEN}✓ NeuroNetes installed${NC}"
    echo ""
}

# Install monitoring stack
install_monitoring() {
    echo -e "${BLUE}Installing monitoring stack...${NC}"
    
    # Add Prometheus community repo
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo update
    
    # Install kube-prometheus-stack
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --create-namespace \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
        --set grafana.adminPassword=admin \
        --wait \
        --timeout 10m
    
    echo -e "${GREEN}✓ Monitoring stack installed${NC}"
    echo ""
    echo -e "${YELLOW}Access Grafana:${NC}"
    echo "  kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
    echo "  Open http://localhost:3000 (admin/admin)"
    echo ""
}

# Deploy sample resources
deploy_samples() {
    echo -e "${BLUE}Deploying sample resources...${NC}"
    
    # Get script directory
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SAMPLES_DIR="${SCRIPT_DIR}/../config/samples"
    
    if [[ -d "$SAMPLES_DIR" ]]; then
        kubectl apply -f "${SAMPLES_DIR}/"
        echo -e "${GREEN}✓ Sample resources deployed${NC}"
    else
        echo -e "${YELLOW}Sample files not found locally, deploying inline samples...${NC}"
        
        # Deploy inline samples
        cat <<EOF | kubectl apply -f -
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: llama-3-8b
  namespace: default
spec:
  weightsURI: s3://models/llama-3-8b/
  size: 16Gi
  quantization: int8
  cachePolicy:
    priority: medium
    pinDuration: 1h
---
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: sample-assistant
  namespace: default
spec:
  modelRef:
    name: llama-3-8b
  maxContextLength: 8192
  temperature: "0.7"
  maxTokens: 2048
  slo:
    ttft: 500ms
    tokensPerSecond: 50
---
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: sample-pool
  namespace: default
spec:
  agentClassRef:
    name: sample-assistant
  minReplicas: 1
  maxReplicas: 5
  prewarmPercent: 20
EOF
        echo -e "${GREEN}✓ Sample resources deployed${NC}"
    fi
    
    echo ""
}

# Print status
print_status() {
    echo -e "${BLUE}Checking installation status...${NC}"
    echo ""
    
    echo "=== Pods ==="
    kubectl get pods -n "${NAMESPACE}"
    echo ""
    
    echo "=== CRDs ==="
    kubectl get crds | grep neuronetes.io || echo "No NeuroNetes CRDs found"
    echo ""
    
    if [[ "$DEPLOY_SAMPLES" == "true" ]]; then
        echo "=== Sample Resources ==="
        kubectl get models,agentclasses,agentpools 2>/dev/null || echo "No sample resources found"
        echo ""
    fi
}

# Print next steps
print_next_steps() {
    echo -e "${GREEN}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                    Installation Complete!                      ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo ""
    echo "1. View the installed resources:"
    echo "   kubectl get pods -n ${NAMESPACE}"
    echo ""
    echo "2. Check the CRDs:"
    echo "   kubectl get crds | grep neuronetes.io"
    echo ""
    echo "3. View sample resources:"
    echo "   kubectl get models,agentclasses,agentpools"
    echo ""
    echo "4. Access metrics:"
    echo "   kubectl port-forward -n ${NAMESPACE} svc/${RELEASE_NAME}-metrics 8080:8080"
    echo "   curl http://localhost:8080/metrics"
    echo ""
    if [[ "$INSTALL_MONITORING" == "true" ]]; then
        echo "5. Access Grafana dashboard:"
        echo "   kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
        echo "   Open http://localhost:3000 (admin/admin)"
        echo ""
    fi
    echo "Documentation: https://github.com/Bowenislandsong/NeuroNetes"
    echo ""
}

# Cleanup function
cleanup() {
    if [[ "$USE_KIND" == "true" ]]; then
        echo -e "${YELLOW}To delete the Kind cluster, run:${NC}"
        echo "  kind delete cluster --name neuronetes-quickstart"
    fi
}

# Main function
main() {
    print_banner
    parse_args "$@"
    check_prerequisites
    
    if [[ "$USE_KIND" == "true" ]]; then
        create_kind_cluster
    fi
    
    install_neuronetes
    
    if [[ "$INSTALL_MONITORING" == "true" ]]; then
        install_monitoring
    fi
    
    if [[ "$DEPLOY_SAMPLES" == "true" ]]; then
        deploy_samples
    fi
    
    print_status
    print_next_steps
    cleanup
}

# Run main function
main "$@"
