#!/bin/bash
# Helm Chart Tests
# This script validates the NeuroNetes Helm chart

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/../../charts/neuronetes"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "Running Helm Chart Tests..."
echo ""

# Test 1: Lint the chart
echo "Test 1: Helm lint..."
if helm lint "$CHART_DIR" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Helm lint passed${NC}"
else
    echo -e "${RED}✗ Helm lint failed${NC}"
    helm lint "$CHART_DIR"
    exit 1
fi

# Test 2: Template with default values
echo "Test 2: Template with default values..."
if helm template test-release "$CHART_DIR" --namespace neuronetes-system > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Template with defaults passed${NC}"
else
    echo -e "${RED}✗ Template with defaults failed${NC}"
    helm template test-release "$CHART_DIR" --namespace neuronetes-system
    exit 1
fi

# Test 3: Template with HA enabled
echo "Test 3: Template with HA enabled..."
if helm template test-release "$CHART_DIR" --namespace neuronetes-system \
    --set highAvailability.enabled=true > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Template with HA enabled passed${NC}"
else
    echo -e "${RED}✗ Template with HA enabled failed${NC}"
    exit 1
fi

# Test 4: Template with ServiceMonitor enabled
echo "Test 4: Template with ServiceMonitor enabled..."
if helm template test-release "$CHART_DIR" --namespace neuronetes-system \
    --set metrics.serviceMonitor.enabled=true > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Template with ServiceMonitor passed${NC}"
else
    echo -e "${RED}✗ Template with ServiceMonitor failed${NC}"
    exit 1
fi

# Test 5: Verify required resources are generated
echo "Test 5: Verify required resources..."
TEMPLATE_OUTPUT=$(helm template test-release "$CHART_DIR" --namespace neuronetes-system)

check_resource() {
    local kind=$1
    local name=$2
    if echo "$TEMPLATE_OUTPUT" | grep -q "kind: $kind"; then
        echo -e "${GREEN}  ✓ $kind resource found${NC}"
        return 0
    else
        echo -e "${RED}  ✗ $kind resource not found${NC}"
        return 1
    fi
}

FAILED=0
check_resource "Namespace" "namespace" || FAILED=1
check_resource "ServiceAccount" "serviceaccount" || FAILED=1
check_resource "ClusterRole" "clusterrole" || FAILED=1
check_resource "ClusterRoleBinding" "clusterrolebinding" || FAILED=1
check_resource "Deployment" "controller" || FAILED=1
check_resource "Service" "metrics" || FAILED=1

if [[ $FAILED -eq 1 ]]; then
    echo -e "${RED}✗ Required resources check failed${NC}"
    exit 1
fi

# Test 6: Verify CRDs in crds directory
echo "Test 6: Verify CRDs..."
CRDS_DIR="${CHART_DIR}/crds"
if [[ -d "$CRDS_DIR" ]]; then
    CRD_COUNT=$(ls -1 "$CRDS_DIR"/*.yaml 2>/dev/null | wc -l)
    if [[ $CRD_COUNT -ge 4 ]]; then
        echo -e "${GREEN}✓ Found $CRD_COUNT CRDs${NC}"
    else
        echo -e "${RED}✗ Expected at least 4 CRDs, found $CRD_COUNT${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ CRDs directory not found${NC}"
    exit 1
fi

# Test 7: Verify values schema (basic validation)
echo "Test 7: Verify values.yaml structure..."
VALUES_FILE="${CHART_DIR}/values.yaml"
if [[ -f "$VALUES_FILE" ]]; then
    # Check for required keys
    REQUIRED_KEYS=("image" "controller" "scheduler" "autoscaler" "metrics" "rbac" "serviceAccount")
    for key in "${REQUIRED_KEYS[@]}"; do
        if grep -q "^${key}:" "$VALUES_FILE"; then
            echo -e "${GREEN}  ✓ Found '$key' section${NC}"
        else
            echo -e "${RED}  ✗ Missing '$key' section${NC}"
            exit 1
        fi
    done
else
    echo -e "${RED}✗ values.yaml not found${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}All Helm chart tests passed!${NC}"
