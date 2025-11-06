# Security Guide

## Overview

NeuroNetes includes comprehensive security features for multi-tenant AI agent workloads. This guide covers security best practices, configurations, and threat models.

## Threat Model

### Threats to Agent Workloads

1. **Prompt Injection**
   - Malicious prompts that override system instructions
   - Tool misuse through crafted inputs
   - Context poisoning

2. **Data Leakage**
   - PII exposure in responses
   - Training data extraction
   - Cross-tenant data access

3. **Resource Abuse**
   - Token exhaustion attacks
   - Computational denial of service
   - Cost exploitation

4. **Model Security**
   - Model weights theft
   - Model inversion attacks
   - Backdoor triggers

5. **GPU Isolation**
   - Memory residue between tenants
   - Side-channel attacks
   - Performance interference

## Security Features

### 1. Guardrails

#### PII Detection and Redaction

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: secure-agent
spec:
  guardrails:
    - type: pii-detection
      action: redact
      threshold: 0.8
      config:
        patterns: "email,phone,ssn,credit_card,api_key"
        replacement: "[REDACTED]"
```

#### Content Safety

```yaml
guardrails:
  - type: safety-check
    action: block
    threshold: 0.9
    config:
      categories: "violence,hate,sexual,self-harm"
      severity: "medium"
```

#### Jailbreak Detection

```yaml
guardrails:
  - type: jailbreak-detection
    action: block
    threshold: 0.85
    config:
      techniques: "role-play,ignore-previous,injection"
```

#### Prompt Injection Prevention

```yaml
guardrails:
  - type: prompt-injection
    action: warn
    threshold: 0.7
    config:
      detection_methods: "similarity,classifier,rule-based"
```

### 2. Tool Permissions

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: restricted-agent
spec:
  toolPermissions:
    - name: file-read
      rateLimit: "10/min"
      timeout: 5s
      maxConcurrency: 2
      requiredScopes:
        - read:files
        - read:metadata
      
      # Restrict to specific paths
      restrictions:
        allowedPaths:
          - /data/public/*
          - /data/user/{user_id}/*
        deniedPaths:
          - /data/private/*
          - /etc/*
          - /root/*
```

### 3. Memory Encryption

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: encrypted-agent
spec:
  memoryConfig:
    type: redis
    ttl: 1h
    maxSize: 10000
    encrypted: true
    
    # Encryption settings
    encryption:
      algorithm: AES-256-GCM
      keyRotation: 24h
      keyProvider: vault
      vaultPath: secret/neuronetes/keys
```

### 4. Multi-Tenant GPU Isolation

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gpu-isolation-config
data:
  config.yaml: |
    # MIG-based isolation
    mig:
      enabled: true
      profiles:
        small: "1g.5gb"
        medium: "2g.10gb"
        large: "3g.20gb"
    
    # VRAM zeroization
    vram:
      zeroize: true
      zeroizeOnEvict: true
      pattern: random
    
    # Compute isolation
    compute:
      enableCgroups: true
      cpuQuota: 80
      priorityClass: tenant-isolation
```

### 5. Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: agent-network-policy
  namespace: agents
spec:
  podSelector:
    matchLabels:
      neuronetes.io/component: agent
  
  policyTypes:
  - Ingress
  - Egress
  
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress
    ports:
    - protocol: TCP
      port: 8080
  
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  
  # Allow specific external services
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

## RBAC Configuration

### User Roles

```yaml
# Read-only user
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: neuronetes-viewer
  namespace: agents
rules:
- apiGroups: ["neuronetes.io"]
  resources: ["models", "agentclasses", "agentpools", "toolbindings"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["neuronetes.io"]
  resources: ["*/status"]
  verbs: ["get"]
---
# Developer role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: neuronetes-developer
  namespace: agents
rules:
- apiGroups: ["neuronetes.io"]
  resources: ["agentclasses", "agentpools", "toolbindings"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["neuronetes.io"]
  resources: ["models"]
  verbs: ["get", "list", "watch"]
---
# Admin role
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: neuronetes-admin
rules:
- apiGroups: ["neuronetes.io"]
  resources: ["*"]
  verbs: ["*"]
```

## Audit Logging

### Enable Audit Logs

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: neuronetes-audit-config
data:
  audit-policy.yaml: |
    apiVersion: audit.k8s.io/v1
    kind: Policy
    rules:
    # Log all requests to NeuroNetes resources
    - level: RequestResponse
      resources:
      - group: neuronetes.io
        resources: ["*"]
    
    # Log tool invocations
    - level: Metadata
      omitStages: ["RequestReceived"]
      verbs: ["invoke"]
    
    # Log guardrail triggers
    - level: Request
      resources:
      - group: neuronetes.io
        resources: ["guardrails"]
```

### Query Audit Logs

```bash
# Search for guardrail blocks
kubectl logs -n neuronetes-system \
  -l app=audit-logger \
  | jq 'select(.guardrail.action == "block")'

# Find tool invocations
kubectl logs -n neuronetes-system \
  -l app=audit-logger \
  | jq 'select(.tool_name != null)'

# Track specific user actions
kubectl logs -n neuronetes-system \
  -l app=audit-logger \
  | jq 'select(.user.username == "specific-user")'
```

## Secrets Management

### Vault Integration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vault-config
data:
  config.yaml: |
    vault:
      address: https://vault.company.com
      role: neuronetes
      authPath: kubernetes
      
    secrets:
      # Model weights access
      - path: secret/models/credentials
        keys:
          - aws_access_key
          - aws_secret_key
      
      # Memory store credentials
      - path: secret/redis/password
        keys:
          - password
      
      # Tool API keys
      - path: secret/tools/api-keys
        keys:
          - openai_key
          - search_key
```

### Secret Injection

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: agent-pod
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "neuronetes"
    vault.hashicorp.com/agent-inject-secret-aws: "secret/models/credentials"
spec:
  serviceAccountName: neuronetes-agent
  containers:
  - name: agent
    image: neuronetes/agent:latest
    env:
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: aws-credentials
          key: access_key_id
```

## Security Best Practices

### 1. Least Privilege

```yaml
# Minimal tool permissions
toolPermissions:
  - name: code-search
    rateLimit: "10/min"  # Conservative limit
    timeout: 5s          # Short timeout
    maxConcurrency: 1    # Single invocation
    requiredScopes:
      - read:code        # Minimal scope
```

### 2. Input Validation

```yaml
# Strict input validation
validation:
  maxInputTokens: 4000
  maxOutputTokens: 2000
  allowedContentTypes:
    - text/plain
    - application/json
  bannedPatterns:
    - <script>
    - javascript:
    - data:text/html
```

### 3. Rate Limiting

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: ToolBinding
metadata:
  name: rate-limited-endpoint
spec:
  httpConfig:
    rateLimitPerIP: "30/min"
    rateLimitPerUser: "100/min"
    rateLimitGlobal: "10000/min"
    
    # Burst allowance
    burst:
      size: 10
      replenishRate: "1/sec"
```

### 4. Monitoring and Alerting

```yaml
# Prometheus alerts for security events
groups:
- name: security
  rules:
  - alert: HighGuardrailBlocks
    expr: rate(neuronetes_safety_blocks_total[5m]) > 10
    for: 5m
    annotations:
      summary: "High rate of guardrail blocks"
  
  - alert: SuspiciousToolUsage
    expr: |
      rate(neuronetes_tool_invocations_total{status="forbidden"}[5m]) > 5
    for: 5m
    annotations:
      summary: "Suspicious tool usage detected"
```

## Compliance

### GDPR Compliance

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: gdpr-compliant
spec:
  # PII handling
  guardrails:
    - type: pii-detection
      action: redact
  
  # Data retention
  memoryConfig:
    ttl: 24h  # Max retention
    encrypted: true
  
  # Right to be forgotten
  dataRetention:
    enableDeletion: true
    deletionPolicy: immediate
  
  # Audit trail
  auditLog:
    enabled: true
    retention: 90d
```

### HIPAA Compliance

```yaml
apiVersion: neuronetes.io/v1alpha1
kind: AgentClass
metadata:
  name: hipaa-compliant
spec:
  # PHI protection
  guardrails:
    - type: phi-detection
      action: block
  
  # Encryption
  memoryConfig:
    encrypted: true
    encryption:
      algorithm: FIPS-140-2
  
  # Access controls
  accessControl:
    mfa: required
    sessionTimeout: 15m
  
  # Audit
  auditLog:
    enabled: true
    immutable: true
    retention: 7y
```

## Incident Response

### Security Incident Playbook

1. **Detection**
   - Monitor security alerts
   - Review audit logs
   - Check guardrail metrics

2. **Containment**
   ```bash
   # Disable compromised agent
   kubectl scale agentpool <pool> --replicas=0
   
   # Revoke credentials
   kubectl delete secret <secret>
   ```

3. **Investigation**
   ```bash
   # Export audit logs
   kubectl logs -n neuronetes-system \
     -l app=audit-logger \
     --since=1h > incident-logs.json
   
   # Check recent changes
   kubectl get events --all-namespaces \
     --field-selector involvedObject.apiVersion=neuronetes.io/v1alpha1
   ```

4. **Recovery**
   - Rotate all secrets
   - Update guardrails
   - Patch vulnerabilities
   - Restore from clean backup

5. **Post-Mortem**
   - Document timeline
   - Identify root cause
   - Update procedures
   - Improve monitoring

## Security Checklist

- [ ] Enable all guardrails
- [ ] Configure tool permissions
- [ ] Enable memory encryption
- [ ] Set up GPU isolation
- [ ] Configure network policies
- [ ] Enable RBAC
- [ ] Set up audit logging
- [ ] Integrate secrets management
- [ ] Configure rate limiting
- [ ] Set up security monitoring
- [ ] Document incident response
- [ ] Regular security audits
- [ ] Compliance verification
- [ ] Pen testing schedule

## References

- [OWASP Top 10 for LLMs](https://owasp.org/www-project-top-10-for-large-language-model-applications/)
- [NIST AI Risk Management Framework](https://www.nist.gov/itl/ai-risk-management-framework)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
