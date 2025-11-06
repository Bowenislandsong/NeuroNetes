# Simple Chat Agent Example

This example demonstrates a basic conversational agent deployment.

## Overview

Deploy a simple chat assistant with:
- Basic conversation capabilities
- Session management
- HTTP endpoint
- Cost optimization

## Quick Start

### 1. Define the Model

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: llama-3-8b-chat
  namespace: default
spec:
  weightsURI: s3://models/llama-3-8b-instruct/
  size: 16Gi
  quantization: int8
  architecture: llama
  parameterCount: "8B"
  format: safetensors
  cachePolicy:
    priority: medium
    pinDuration: 1h
EOF
```

### 2. Create AgentClass

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: chat-assistant
  namespace: default
spec:
  modelRef:
    name: llama-3-8b-chat
  
  maxContextLength: 8192
  
  systemPrompt: |
    You are a helpful, friendly AI assistant. Provide clear,
    concise, and accurate responses. Be conversational but professional.
  
  temperature: 0.8
  maxTokens: 2048
  
  guardrails:
    - type: pii-detection
      action: redact
      threshold: 0.8
    
    - type: safety-check
      action: block
      threshold: 0.9
  
  slo:
    ttft: 300ms
    tokensPerSecond: 75
    p95Latency: 1s
  
  memoryConfig:
    type: ephemeral
    ttl: 30m
    maxSize: 5000
EOF
```

### 3. Deploy AgentPool

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: chat-pool
  namespace: default
spec:
  agentClassRef:
    name: chat-assistant
  
  minReplicas: 3
  maxReplicas: 30
  prewarmPercent: 30
  
  migProfile: "1g.5gb"
  
  autoscaling:
    metrics:
      - type: concurrent-sessions
        target: "100"
      - type: ttft-p95
        target: "300ms"
    
    behavior:
      scaleUp:
        stabilizationWindow: 30s
        maxChangeAbsolute: 5
      scaleDown:
        stabilizationWindow: 180s
        maxChangeAbsolute: 2
  
  gpuRequirements:
    count: 1
    memory: "20Gi"
    type: "A100"
  
  sessionAffinity:
    enabled: true
    keyHeader: "X-Session-ID"
    ttl: 30m
    type: conversation-id
  
  scheduling:
    costOptimization:
      enabled: true
      spotEnabled: true
      maxCostPerHour: 30.0
EOF
```

### 4. Create HTTP Endpoint

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: chat-http
  namespace: default
spec:
  agentPoolRef:
    name: chat-pool
  
  type: http
  
  httpConfig:
    path: /v1/chat/completions
    methods: [POST, OPTIONS]
    streamingEnabled: true
    
    corsConfig:
      allowedOrigins: ["*"]
      allowedMethods: [POST, OPTIONS]
      allowedHeaders: [Content-Type, Authorization, X-Session-ID]
  
  concurrency:
    maxConcurrentRequests: 100
    perSessionLimit: 3
  
  timeouts:
    requestTimeout: 1m
    idleTimeout: 30s
  
  retryPolicy:
    maxAttempts: 2
    initialBackoff: 500ms
EOF
```

### 5. Test the Agent

```bash
# Get the endpoint
ENDPOINT=$(kubectl get svc neuronetes-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Send a request
curl -X POST http://$ENDPOINT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: test-session-123" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Hello! Can you help me understand quantum computing?"
      }
    ],
    "stream": false
  }'
```

### 6. Test Streaming

```bash
curl -X POST http://$ENDPOINT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: test-session-123" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "Tell me a short story about a robot."
      }
    ],
    "stream": true
  }'
```

## Monitoring

```bash
# Check agent pool status
kubectl get agentpool chat-pool

# View replicas
kubectl get pods -l neuronetes.io/pool=chat-pool

# Monitor metrics
kubectl port-forward -n neuronetes-system svc/prometheus 9090

# View logs
kubectl logs -l neuronetes.io/pool=chat-pool --tail=100 -f
```

## Scaling Behavior

The chat pool automatically scales based on:

- **Concurrent sessions**: Scales when > 100 active conversations
- **TTFT P95**: Scales when latency > 300ms
- **30% warm pool**: Always keeps 9 warm replicas (30% of 30 max)

## Cost Optimization

The pool uses spot instances when:
- Available in the region
- Cost < $30/hour budget
- SLO not at risk

Fallback to on-demand if:
- Spot interrupted
- Latency exceeds SLO
- No spot availability

## Client Libraries

### Python

```python
import requests
import json

def chat(message, session_id="default"):
    response = requests.post(
        "http://ENDPOINT/v1/chat/completions",
        headers={
            "Content-Type": "application/json",
            "X-Session-ID": session_id
        },
        json={
            "messages": [{"role": "user", "content": message}],
            "stream": False
        }
    )
    return response.json()

# Usage
result = chat("What is the weather like today?", "user-123")
print(result["choices"][0]["message"]["content"])
```

### JavaScript

```javascript
async function chat(message, sessionId = "default") {
  const response = await fetch("http://ENDPOINT/v1/chat/completions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Session-ID": sessionId
    },
    body: JSON.stringify({
      messages: [{ role: "user", content: message }],
      stream: false
    })
  });
  
  return await response.json();
}

// Usage
const result = await chat("Tell me a joke", "user-456");
console.log(result.choices[0].message.content);
```

### Streaming in Python

```python
import requests
import json

def chat_stream(message, session_id="default"):
    response = requests.post(
        "http://ENDPOINT/v1/chat/completions",
        headers={
            "Content-Type": "application/json",
            "X-Session-ID": session_id
        },
        json={
            "messages": [{"role": "user", "content": message}],
            "stream": True
        },
        stream=True
    )
    
    for line in response.iter_lines():
        if line:
            data = json.loads(line.decode('utf-8'))
            if "content" in data:
                yield data["content"]

# Usage
for chunk in chat_stream("Write a poem about AI", "user-789"):
    print(chunk, end="", flush=True)
```

## Cleanup

```bash
kubectl delete toolbinding chat-http
kubectl delete agentpool chat-pool
kubectl delete agentclass chat-assistant
kubectl delete model llama-3-8b-chat
```

## Next Steps

- Add custom tools
- Integrate with RAG
- Enable multi-language support
- Add conversation history
- Implement rate limiting
