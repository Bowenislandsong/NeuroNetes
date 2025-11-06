# RAG Pipeline Example

This example demonstrates a complete Retrieval-Augmented Generation (RAG) pipeline with agent orchestration.

## Overview

Deploy a RAG-powered Q&A system with:
- Document ingestion and embedding
- Vector store for semantic search
- Agent with retrieval tool
- Session management
- Cost optimization

## Architecture

```
Documents → Embedder → Vector Store → Agent → Response
                          ↑             ↓
                          └─── Retrieval Tool
```

## Setup

### 1. Deploy Vector Store

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: weaviate
  namespace: rag-system
spec:
  serviceName: weaviate
  replicas: 3
  selector:
    matchLabels:
      app: weaviate
  template:
    metadata:
      labels:
        app: weaviate
    spec:
      containers:
      - name: weaviate
        image: semitechnologies/weaviate:1.23.0
        ports:
        - containerPort: 8080
        env:
        - name: PERSISTENCE_DATA_PATH
          value: "/var/lib/weaviate"
        - name: CLUSTER_HOSTNAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        volumeMounts:
        - name: data
          mountPath: /var/lib/weaviate
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
---
apiVersion: v1
kind: Service
metadata:
  name: weaviate
  namespace: rag-system
spec:
  clusterIP: None
  selector:
    app: weaviate
  ports:
  - port: 8080
    targetPort: 8080
EOF
```

### 2. Deploy Embedding Model

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: embedding-model
  namespace: rag-system
spec:
  weightsURI: s3://models/bge-large-en-v1.5/
  size: 1.3Gi
  quantization: fp16
  architecture: bert
  format: safetensors
  cachePolicy:
    priority: critical
    pinDuration: 24h
    evictionPolicy: never
EOF
```

### 3. Deploy LLM Model

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: Model
metadata:
  name: llama-3-70b-rag
  namespace: rag-system
spec:
  weightsURI: s3://models/llama-3-70b-instruct/
  size: 140Gi
  quantization: int4
  architecture: llama
  parameterCount: "70B"
  shardSpec:
    count: 4
    strategy: tensor-parallel
    topology:
      locality: same-node
  cachePolicy:
    priority: high
    pinDuration: 4h
EOF
```

### 4. Create RAG AgentClass

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: rag-agent
  namespace: rag-system
spec:
  modelRef:
    name: llama-3-70b-rag
  
  maxContextLength: 128000
  
  systemPrompt: |
    You are a helpful assistant with access to a knowledge base.
    
    When answering questions:
    1. Use the retrieve tool to find relevant information
    2. Cite sources from the retrieved documents
    3. Be accurate and admit when you don't know
    4. Synthesize information from multiple sources
    
    Format citations as [Source: document_id].
  
  temperature: 0.3
  maxTokens: 4096
  
  toolPermissions:
    - name: retrieve
      rateLimit: "50/min"
      timeout: 5s
      maxConcurrency: 3
      requiredScopes:
        - read:documents
    
    - name: rerank
      rateLimit: "30/min"
      timeout: 2s
      maxConcurrency: 2
  
  guardrails:
    - type: pii-detection
      action: redact
    
    - type: content-filter
      action: block
  
  slo:
    ttft: 800ms
    tokensPerSecond: 45
    p95Latency: 3s
  
  memoryConfig:
    type: redis
    ttl: 1h
    maxSize: 20000
    encrypted: true
    connectionString: redis://redis.rag-system:6379
EOF
```

### 5. Create AgentPool with Data Locality

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: AgentPool
metadata:
  name: rag-pool
  namespace: rag-system
spec:
  agentClassRef:
    name: rag-agent
  
  minReplicas: 3
  maxReplicas: 15
  prewarmPercent: 25
  
  migProfile: "3g.20gb"
  
  autoscaling:
    metrics:
      - type: concurrent-sessions
        target: "30"
      - type: ttft-p95
        target: "800ms"
      - type: tool-call-rate
        target: "10"
  
  gpuRequirements:
    count: 2
    memory: "40Gi"
    type: "A100"
  
  sessionAffinity:
    enabled: true
    keyHeader: "X-Session-ID"
    ttl: 1h
    type: conversation-id
  
  scheduling:
    priority: 90
    
    # Co-locate with vector store
    dataLocality:
      vectorStoreAffinity:
        - weaviate
      cacheAffinity:
        - redis
    
    costOptimization:
      enabled: true
      spotEnabled: true
      sloHeadroomMs: 1500
    
    nodeSelector:
      node-type: gpu
      region: us-west-2
EOF
```

### 6. Create HTTP Endpoint

```bash
kubectl apply -f - <<EOF
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: rag-http
  namespace: rag-system
spec:
  agentPoolRef:
    name: rag-pool
  
  type: http
  
  httpConfig:
    path: /v1/rag/query
    methods: [POST]
    streamingEnabled: true
    rateLimitPerIP: "30/min"
  
  concurrency:
    maxConcurrentRequests: 30
    perSessionLimit: 2
  
  timeouts:
    requestTimeout: 3m
    toolTimeout: 5s
  
  retryPolicy:
    maxAttempts: 2
    initialBackoff: 1s
EOF
```

## Document Ingestion

### Ingest Script

```python
import weaviate
import openai

# Connect to Weaviate
client = weaviate.Client("http://weaviate.rag-system:8080")

# Create schema
schema = {
    "class": "Document",
    "vectorizer": "none",  # We'll provide embeddings
    "properties": [
        {"name": "content", "dataType": ["text"]},
        {"name": "source", "dataType": ["string"]},
        {"name": "timestamp", "dataType": ["date"]},
    ]
}
client.schema.create_class(schema)

# Embed and ingest
def ingest_document(content, source):
    # Get embedding (use your embedding model)
    embedding = get_embedding(content)
    
    # Store in Weaviate
    client.data_object.create(
        data_object={
            "content": content,
            "source": source,
            "timestamp": datetime.now().isoformat()
        },
        class_name="Document",
        vector=embedding
    )

def get_embedding(text):
    # Call embedding model endpoint
    response = requests.post(
        "http://embedding-service:8080/embed",
        json={"text": text}
    )
    return response.json()["embedding"]
```

## Query the RAG System

### Python Client

```python
import requests
import json

def rag_query(question, session_id=None):
    """Query the RAG system"""
    
    response = requests.post(
        "http://rag-http.rag-system/v1/rag/query",
        headers={
            "Content-Type": "application/json",
            "X-Session-ID": session_id or "default"
        },
        json={
            "messages": [
                {
                    "role": "user",
                    "content": question
                }
            ],
            "stream": False,
            "retrieval_config": {
                "top_k": 5,
                "rerank": True
            }
        }
    )
    
    return response.json()

# Example usage
result = rag_query(
    "What are the best practices for Kubernetes security?",
    session_id="user-123"
)

print(result["choices"][0]["message"]["content"])
print(f"Sources used: {result['metadata']['sources_used']}")
```

### Streaming Response

```python
import requests
import json

def rag_query_stream(question, session_id=None):
    """Query with streaming response"""
    
    response = requests.post(
        "http://rag-http.rag-system/v1/rag/query",
        headers={
            "Content-Type": "application/json",
            "X-Session-ID": session_id or "default"
        },
        json={
            "messages": [{"role": "user", "content": question}],
            "stream": True
        },
        stream=True
    )
    
    for line in response.iter_lines():
        if line:
            data = json.loads(line)
            if "content" in data:
                yield data["content"]

# Example
for chunk in rag_query_stream("Explain Docker containers"):
    print(chunk, end="", flush=True)
```

## Monitoring

```bash
# Check pool status
kubectl get agentpool rag-pool -n rag-system

# View metrics
kubectl port-forward -n neuronetes-system svc/prometheus 9090

# Query retrieval latency
neuronetes_retrieval_latency_seconds{agent_class="rag-agent"}

# Query tool invocation rate
rate(neuronetes_tool_invocations_total{tool_name="retrieve"}[5m])

# Check session count
neuronetes_concurrent_sessions{pool="rag-pool"}
```

## Performance Tuning

### Optimize Retrieval

```yaml
# In AgentClass spec
toolPermissions:
  - name: retrieve
    config:
      top_k: 5          # Retrieve top 5 documents
      rerank: true      # Re-rank results
      cache_ttl: 300    # Cache for 5 minutes
```

### Optimize Context

```yaml
# Chunk documents appropriately
chunk_size: 512         # Tokens per chunk
chunk_overlap: 50       # Overlap between chunks
max_chunks_per_query: 5 # Limit context size
```

### Scale Vector Store

```bash
# Scale Weaviate replicas
kubectl scale statefulset weaviate \
  --replicas=5 \
  -n rag-system
```

## Best Practices

1. **Document Chunking**
   - Chunk size: 256-512 tokens
   - Overlap: 10-20%
   - Include metadata

2. **Embedding Strategy**
   - Use domain-specific embeddings
   - Batch embed for efficiency
   - Cache frequently accessed embeddings

3. **Retrieval Tuning**
   - Start with top_k=5
   - Enable reranking
   - Filter by metadata when possible

4. **Context Management**
   - Limit context size
   - Prioritize recent messages
   - Summarize old context

5. **Cost Optimization**
   - Cache embeddings
   - Use spot instances
   - Batch queries when possible

## Cleanup

```bash
kubectl delete namespace rag-system
```
