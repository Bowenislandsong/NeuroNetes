# NeuroNetes Examples - Quick Reference

This guide provides a quick overview of all examples with their key properties and expected results.

## Examples Overview

| Example | Model | Context | TTFT P95 | Throughput | Cost/1K | Use Case |
|---------|-------|---------|----------|------------|---------|----------|
| [Chat Agent](chat-agent/) | Llama-3-8B | 8K | 245ms | 82 tok/s | $0.02 | General conversation |
| [Code Assistant](code-assistant/) | Llama-3-70B | 128K | 450ms | 55 tok/s | $0.15 | Code help with tools |
| [RAG Pipeline](rag-pipeline/) | Llama-3-70B | 128K | 720ms | 48 tok/s | $0.18 | Knowledge Q&A |

## Quick Start

### Chat Agent
```bash
# Deploy in 3 commands
kubectl apply -f examples/chat-agent/model.yaml
kubectl apply -f examples/chat-agent/agentclass.yaml
kubectl apply -f examples/chat-agent/agentpool.yaml

# Test
curl -X POST http://$ENDPOINT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: test-123" \
  -d '{"messages": [{"role": "user", "content": "Hello!"}]}'
```

**Best For:** Customer support, virtual assistants, general Q&A

### Code Assistant
```bash
# Deploy with tools
kubectl apply -f examples/code-assistant/model.yaml
kubectl apply -f examples/code-assistant/agentclass.yaml
kubectl apply -f examples/code-assistant/agentpool.yaml

# Requires: Redis for session storage
```

**Best For:** Code review, debugging, documentation generation

### RAG Pipeline
```bash
# Deploy vector store first
kubectl create namespace rag-system
kubectl apply -f examples/rag-pipeline/weaviate.yaml

# Deploy models and agent
kubectl apply -f examples/rag-pipeline/models.yaml
kubectl apply -f examples/rag-pipeline/agentclass.yaml
kubectl apply -f examples/rag-pipeline/agentpool.yaml

# Requires: Weaviate (vector store), Redis (sessions)
```

**Best For:** Knowledge base Q&A, documentation search, semantic search

## Performance Comparison

### Latency
```
Chat Agent:     ████ 245ms ✓
Code Assistant: ███████ 450ms ✓
RAG Pipeline:   ████████████ 720ms ✓
Target SLO:     █████ 300-800ms
```

### Cost Efficiency
```
Chat Agent:     ██ $0.02/1K tokens (Most economical)
Code Assistant: ███████ $0.15/1K tokens
RAG Pipeline:   ████████ $0.18/1K tokens (Most capable)
```

### Resource Usage
```
Chat Agent:     █ 1 GPU (MIG 1g.5gb), 16GB VRAM
Code Assistant: ████ 2 GPUs (A100 40GB), 140GB VRAM
RAG Pipeline:   ████ 2 GPUs (A100 40GB), 140GB VRAM + Vector Store
```

## Key Features by Example

### Chat Agent ✨
- ✅ Fastest TTFT (245ms)
- ✅ Lowest cost ($0.02/1K)
- ✅ Simple deployment
- ✅ Guardrails (PII, safety)
- ✅ Session affinity
- ✅ Spot instance support

### Code Assistant 🛠️
- ✅ Multi-tool support (code-search, file-read)
- ✅ Large context (128K tokens)
- ✅ High accuracy (91%)
- ✅ Rate limiting per tool
- ✅ Redis session storage
- ✅ Code injection prevention

### RAG Pipeline 📚
- ✅ Vector search integration
- ✅ Embedding model included
- ✅ High cache hit rate (85%)
- ✅ Multi-source synthesis
- ✅ Data locality optimization
- ✅ Answer accuracy (94%)

## Scaling Configuration

### Chat Agent
```yaml
minReplicas: 3
maxReplicas: 30
prewarmPercent: 30%  # 9 warm replicas
triggers:
  - concurrent-sessions > 100
  - ttft-p95 > 300ms
```

### Code Assistant
```yaml
minReplicas: 2
maxReplicas: 10
prewarmPercent: 25%  # 2-3 warm replicas
triggers:
  - concurrent-sessions > 30
  - ttft-p95 > 500ms
  - tool-call-rate > 10/sec
```

### RAG Pipeline
```yaml
minReplicas: 3
maxReplicas: 15
prewarmPercent: 25%  # 3-4 warm replicas
triggers:
  - concurrent-sessions > 30
  - ttft-p95 > 800ms
  - tool-call-rate > 10/sec
```

## Cost Estimates

### Monthly Infrastructure (Typical Load)

**Chat Agent** (~10 replicas avg)
- LLM: $3,312/month
- With Spot: $1,160/month (65% savings)
- Cost per 1M requests: ~$4,200

**Code Assistant** (~4 replicas avg)
- LLM: $21,139/month
- Redis: $864/month
- With Spot: $8,263/month (61% savings)
- Cost per 100K requests: ~$23,450

**RAG Pipeline** (~5 replicas avg)
- LLM: $26,424/month
- Embedding: $1,728/month
- Weaviate: $6,912/month
- Redis: $864/month
- Total: $35,928/month
- With Spot: $18,752/month (48% savings)
- Cost per 80K queries: ~$39,200

## Optimization Tips

### For Lower Latency
1. Increase `prewarmPercent` (e.g., 40%)
2. Use snapshot/restore for faster cold starts
3. Co-locate with dependencies (Redis, Weaviate)
4. Enable request batching

### For Lower Cost
1. Enable spot instances
2. Reduce `minReplicas` during off-peak
3. Increase `scaleDown` stabilization window
4. Use aggressive caching (RAG)

### For Higher Quality
1. Increase model size (8B → 70B)
2. Fine-tune on domain data
3. Add domain-specific tools
4. Implement retrieval for context

## Monitoring

### Key Metrics to Watch
```promql
# TTFT P95
histogram_quantile(0.95, rate(neuronetes_ttft_seconds_bucket[5m]))

# Tokens per second
rate(neuronetes_tokens_generated_total[5m])

# Concurrent sessions
neuronetes_concurrent_sessions

# GPU utilization
neuronetes_gpu_utilization_percent

# Cost per request
rate(neuronetes_cost_total[5m]) / rate(neuronetes_requests_total[5m])
```

### Grafana Dashboards
```bash
# Import pre-built dashboard
kubectl create configmap neuronetes-dashboard \
  --from-file=config/grafana/neuronetes-dashboard.json \
  -n monitoring
```

## Common Issues & Solutions

### High Latency
**Problem:** TTFT P95 exceeds SLO
**Solutions:**
- Increase warm pool percentage
- Check GPU utilization
- Verify network latency to dependencies
- Review batch size configuration

### High Costs
**Problem:** Monthly bill higher than expected
**Solutions:**
- Enable spot instances
- Reduce minimum replicas
- Implement request caching
- Use smaller model for simple queries

### Low Accuracy
**Problem:** Responses not meeting quality standards
**Solutions:**
- Increase model size
- Add retrieval/RAG capabilities
- Fine-tune on domain data
- Adjust temperature (lower = more deterministic)

## Next Steps

1. Choose an example based on your use case
2. Review the full RESULTS.md for detailed metrics
3. Deploy using the provided YAML files
4. Monitor with Prometheus/Grafana
5. Optimize based on your workload

## Resources

- 📚 [Full Examples with Results](https://bowenislandsong.github.io/NeuroNetes/website/examples.html)
- 📊 [Performance Benchmarks](https://bowenislandsong.github.io/NeuroNetes/website/benchmarks.html)
- 🏠 [Main Website](https://bowenislandsong.github.io/NeuroNetes/website/)
- 📖 [Documentation](../docs/)
- 💬 [GitHub Discussions](https://github.com/Bowenislandsong/NeuroNetes/discussions)

## Support

- 🐛 [Report Issues](https://github.com/Bowenislandsong/NeuroNetes/issues)
- 💡 [Feature Requests](https://github.com/Bowenislandsong/NeuroNetes/issues/new)
- 📧 Community Support (coming soon)
