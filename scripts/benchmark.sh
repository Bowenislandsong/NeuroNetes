#!/bin/bash
# NeuroNetes Benchmark Script
# This script runs benchmarks against a NeuroNetes deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="default"
POOL_NAME=""
ENDPOINT=""
CONCURRENCY=10
DURATION=60
OUTPUT_DIR="/tmp/neuronetes-benchmark"

# Print usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -n, --namespace NAME     Namespace of the AgentPool (default: default)"
    echo "  -p, --pool NAME          Name of the AgentPool to benchmark (required)"
    echo "  -e, --endpoint URL       Direct endpoint URL (optional, auto-detected)"
    echo "  -c, --concurrency NUM    Number of concurrent requests (default: 10)"
    echo "  -d, --duration SECS      Duration of benchmark in seconds (default: 60)"
    echo "  -o, --output DIR         Output directory for results (default: /tmp/neuronetes-benchmark)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -p chat-pool"
    echo "  $0 -p chat-pool -c 50 -d 120"
    echo "  $0 -p code-assistant-pool -e http://localhost:8080/v1/chat/completions"
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
            -p|--pool)
                POOL_NAME="$2"
                shift 2
                ;;
            -e|--endpoint)
                ENDPOINT="$2"
                shift 2
                ;;
            -c|--concurrency)
                CONCURRENCY="$2"
                shift 2
                ;;
            -d|--duration)
                DURATION="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
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
    
    if [[ -z "$POOL_NAME" ]]; then
        echo -e "${RED}Error: Pool name is required (-p/--pool)${NC}"
        usage
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}Error: kubectl is not installed${NC}"
        exit 1
    fi
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}Error: jq is not installed${NC}"
        echo "Please install jq manually:"
        echo "  - Ubuntu/Debian: sudo apt-get install jq"
        echo "  - macOS: brew install jq"
        echo "  - RHEL/CentOS: sudo yum install jq"
        exit 1
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Prerequisites met${NC}"
    echo ""
}

# Get pool information
get_pool_info() {
    echo -e "${BLUE}Getting AgentPool information...${NC}"
    
    POOL_INFO=$(kubectl get agentpool "$POOL_NAME" -n "$NAMESPACE" -o json 2>/dev/null)
    
    if [[ -z "$POOL_INFO" ]]; then
        echo -e "${RED}Error: AgentPool '$POOL_NAME' not found in namespace '$NAMESPACE'${NC}"
        exit 1
    fi
    
    AGENT_CLASS=$(echo "$POOL_INFO" | jq -r '.spec.agentClassRef.name')
    MIN_REPLICAS=$(echo "$POOL_INFO" | jq -r '.spec.minReplicas')
    MAX_REPLICAS=$(echo "$POOL_INFO" | jq -r '.spec.maxReplicas')
    READY_REPLICAS=$(echo "$POOL_INFO" | jq -r '.status.readyReplicas // 0')
    
    echo "  Pool: $POOL_NAME"
    echo "  Agent Class: $AGENT_CLASS"
    echo "  Replicas: $READY_REPLICAS ready ($MIN_REPLICAS min, $MAX_REPLICAS max)"
    echo ""
}

# Run warmup
run_warmup() {
    echo -e "${BLUE}Running warmup requests...${NC}"
    
    for i in {1..5}; do
        curl -s -X POST "$ENDPOINT" \
            -H "Content-Type: application/json" \
            -d '{"messages":[{"role":"user","content":"Hello"}],"max_tokens":10}' > /dev/null
    done
    
    echo -e "${GREEN}✓ Warmup complete${NC}"
    echo ""
}

# Run benchmark
run_benchmark() {
    echo -e "${BLUE}Running benchmark...${NC}"
    echo "  Concurrency: $CONCURRENCY"
    echo "  Duration: ${DURATION}s"
    echo ""
    
    mkdir -p "$OUTPUT_DIR"
    RESULTS_FILE="$OUTPUT_DIR/results-$(date +%Y%m%d-%H%M%S).json"
    
    START_TIME=$(date +%s)
    END_TIME=$((START_TIME + DURATION))
    
    REQUEST_COUNT=0
    SUCCESS_COUNT=0
    ERROR_COUNT=0
    TOTAL_LATENCY=0
    TTFT_TOTAL=0
    TOKENS_TOTAL=0
    
    declare -a LATENCIES
    declare -a TTFTS
    
    echo "Progress:"
    
    while [[ $(date +%s) -lt $END_TIME ]]; do
        # Run concurrent requests
        for ((i=0; i<CONCURRENCY; i++)); do
            (
                REQ_START=$(date +%s%3N)
                
                RESPONSE=$(curl -s -w "\n%{time_total}" -X POST "$ENDPOINT" \
                    -H "Content-Type: application/json" \
                    -d '{"messages":[{"role":"user","content":"Write a short poem about technology."}],"max_tokens":100}')
                
                REQ_END=$(date +%s%3N)
                LATENCY=$((REQ_END - REQ_START))
                
                # Check if response is valid
                if echo "$RESPONSE" | head -n -1 | jq -e '.choices' > /dev/null 2>&1; then
                    echo "success,$LATENCY" >> "$OUTPUT_DIR/raw_results.csv"
                else
                    echo "error,$LATENCY" >> "$OUTPUT_DIR/raw_results.csv"
                fi
            ) &
        done
        
        wait
        
        REQUEST_COUNT=$((REQUEST_COUNT + CONCURRENCY))
        echo -ne "\r  Requests sent: $REQUEST_COUNT"
    done
    
    echo ""
    echo ""
    
    # Calculate statistics
    if [[ -f "$OUTPUT_DIR/raw_results.csv" ]]; then
        SUCCESS_COUNT=$(grep "^success" "$OUTPUT_DIR/raw_results.csv" | wc -l)
        ERROR_COUNT=$(grep "^error" "$OUTPUT_DIR/raw_results.csv" | wc -l)
        
        if [[ $SUCCESS_COUNT -gt 0 ]]; then
            LATENCIES=$(grep "^success" "$OUTPUT_DIR/raw_results.csv" | cut -d',' -f2 | sort -n)
            
            AVG_LATENCY=$(echo "$LATENCIES" | awk '{ sum += $1 } END { print int(sum/NR) }')
            MIN_LATENCY=$(echo "$LATENCIES" | head -1)
            MAX_LATENCY=$(echo "$LATENCIES" | tail -1)
            
            P50_IDX=$((SUCCESS_COUNT / 2))
            P95_IDX=$((SUCCESS_COUNT * 95 / 100))
            P99_IDX=$((SUCCESS_COUNT * 99 / 100))
            
            P50_LATENCY=$(echo "$LATENCIES" | sed -n "${P50_IDX}p")
            P95_LATENCY=$(echo "$LATENCIES" | sed -n "${P95_IDX}p")
            P99_LATENCY=$(echo "$LATENCIES" | sed -n "${P99_IDX}p")
        fi
    fi
    
    # Calculate throughput
    ACTUAL_DURATION=$(($(date +%s) - START_TIME))
    if [[ $ACTUAL_DURATION -eq 0 ]]; then
        ACTUAL_DURATION=1
    fi
    if [[ $REQUEST_COUNT -eq 0 ]]; then
        RPS=0
    else
        RPS=$(awk "BEGIN {printf \"%.2f\", $SUCCESS_COUNT / $ACTUAL_DURATION}")
    fi
    
    # Calculate success rate safely
    if [[ $REQUEST_COUNT -eq 0 ]]; then
        SUCCESS_RATE=0
    else
        SUCCESS_RATE=$(awk "BEGIN {printf \"%.2f\", $SUCCESS_COUNT * 100 / $REQUEST_COUNT}")
    fi
    
    # Generate results
    cat > "$RESULTS_FILE" <<EOF
{
  "benchmark": {
    "pool": "$POOL_NAME",
    "namespace": "$NAMESPACE",
    "endpoint": "$ENDPOINT",
    "concurrency": $CONCURRENCY,
    "duration_seconds": $ACTUAL_DURATION,
    "timestamp": "$(date -Iseconds)"
  },
  "results": {
    "total_requests": $REQUEST_COUNT,
    "successful_requests": $SUCCESS_COUNT,
    "failed_requests": $ERROR_COUNT,
    "success_rate": $SUCCESS_RATE,
    "requests_per_second": $RPS
  },
  "latency_ms": {
    "avg": ${AVG_LATENCY:-0},
    "min": ${MIN_LATENCY:-0},
    "max": ${MAX_LATENCY:-0},
    "p50": ${P50_LATENCY:-0},
    "p95": ${P95_LATENCY:-0},
    "p99": ${P99_LATENCY:-0}
  }
}
EOF
    
    echo -e "${GREEN}✓ Benchmark complete${NC}"
    echo ""
}

# Print results
print_results() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                       Benchmark Results                         ${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    
    echo "Pool: $POOL_NAME"
    echo "Duration: ${ACTUAL_DURATION}s"
    echo "Concurrency: $CONCURRENCY"
    echo ""
    
    echo -e "${YELLOW}Throughput:${NC}"
    echo "  Total Requests: $REQUEST_COUNT"
    echo "  Successful: $SUCCESS_COUNT"
    echo "  Failed: $ERROR_COUNT"
    echo "  Success Rate: ${SUCCESS_RATE}%"
    echo "  Requests/sec: $RPS"
    echo ""
    
    echo -e "${YELLOW}Latency (ms):${NC}"
    echo "  Average: ${AVG_LATENCY:-N/A}"
    echo "  Min: ${MIN_LATENCY:-N/A}"
    echo "  Max: ${MAX_LATENCY:-N/A}"
    echo "  P50: ${P50_LATENCY:-N/A}"
    echo "  P95: ${P95_LATENCY:-N/A}"
    echo "  P99: ${P99_LATENCY:-N/A}"
    echo ""
    
    echo -e "${GREEN}Results saved to: $RESULTS_FILE${NC}"
    echo ""
}

# Collect metrics
collect_metrics() {
    echo -e "${BLUE}Collecting metrics from Prometheus...${NC}"
    
    # Try to get Prometheus endpoint
    PROM_SVC=$(kubectl get svc -A | grep prometheus-server | head -1 | awk '{print $1,$2}')
    
    if [[ -n "$PROM_SVC" ]]; then
        NS=$(echo "$PROM_SVC" | awk '{print $1}')
        SVC=$(echo "$PROM_SVC" | awk '{print $2}')
        
        echo "  Found Prometheus at $NS/$SVC"
        
        # Port-forward in background
        kubectl port-forward -n "$NS" "svc/$SVC" 9090:9090 &
        PF_PID=$!
        sleep 2
        
        # Query metrics
        METRICS_FILE="$OUTPUT_DIR/metrics-$(date +%Y%m%d-%H%M%S).json"
        
        # Get token metrics
        curl -s "http://localhost:9090/api/v1/query?query=neuronetes_tokens_per_second{pool=\"$POOL_NAME\"}" | jq '.' > "$METRICS_FILE"
        
        # Kill port-forward
        kill $PF_PID 2>/dev/null || true
        
        echo -e "${GREEN}✓ Metrics collected to $METRICS_FILE${NC}"
    else
        echo -e "${YELLOW}Prometheus not found, skipping metrics collection${NC}"
    fi
    
    echo ""
}

# Main function
main() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║                    NeuroNetes Benchmark                        ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    
    parse_args "$@"
    check_prerequisites
    get_pool_info
    
    # Auto-detect endpoint if not provided
    if [[ -z "$ENDPOINT" ]]; then
        # Look for ToolBinding
        TB=$(kubectl get toolbinding -n "$NAMESPACE" -o json | jq -r ".items[] | select(.spec.agentPoolRef.name == \"$POOL_NAME\") | .status.endpoint" | head -1)
        
        if [[ -n "$TB" && "$TB" != "null" ]]; then
            ENDPOINT="$TB"
            echo "Auto-detected endpoint: $ENDPOINT"
        else
            echo -e "${YELLOW}No endpoint found. Please provide one with -e/--endpoint${NC}"
            echo "Or set up port-forwarding and provide localhost endpoint"
            exit 1
        fi
    fi
    
    run_warmup
    run_benchmark
    print_results
    collect_metrics
    
    # Cleanup
    rm -f "$OUTPUT_DIR/raw_results.csv"
}

# Run main function
main "$@"
