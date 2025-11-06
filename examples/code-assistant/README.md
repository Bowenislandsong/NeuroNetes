# Complete Example: RAG-Powered Code Assistant

This example demonstrates a complete agent deployment with RAG (Retrieval-Augmented Generation) for code assistance.

## Overview

This setup creates:
- A code assistant agent with multiple tools
- RAG pipeline with vector store
- Queue-based task processing
- Session-based state management
- Cost-optimized GPU scheduling

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Client    │─────▶│   Ingress    │─────▶│   Agent     │
│  (VSCode)   │◀─────│   (HTTP)     │◀─────│    Pool     │
└─────────────┘      └──────────────┘      └─────┬───────┘
                                                   │
                     ┌──────────────┐             │
                     │  Task Queue  │◀────────────┤
                     │   (NATS)     │             │
                     └──────────────┘             │
                                                   │
                     ┌──────────────┐             │
                     │Vector Store  │◀────────────┤
                     │  (Weaviate)  │             │
                     └──────────────┘             │
                                                   │
                     ┌──────────────┐             │
                     │Session Store │◀────────────┘
                     │   (Redis)    │
                     └──────────────┘
```

See [README.md](README.md) for the complete example with full deployment instructions, configurations, and testing procedures.
