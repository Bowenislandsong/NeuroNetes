/*
Copyright 2024 NeuroNetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// AgentMetrics defines all agent-native metrics for NeuroNetes
type AgentMetrics struct {
	// UX & Quality (SLO-facing)
	TTFTHistogram        prometheus.Histogram
	LatencyHistogram     prometheus.Histogram
	RTFRatio             prometheus.Gauge
	TokensOutRate        prometheus.Gauge
	CSATScore            prometheus.Gauge
	ThumbsUpRate         prometheus.Gauge
	TurnErrorRate        prometheus.Counter
	QualityWinRate       prometheus.Gauge

	// Load & Concurrency
	ActiveSessions     prometheus.Gauge
	QueueDepth         prometheus.Gauge
	AdmissionRejects   prometheus.Counter
	ScalingLag         prometheus.Histogram

	// Token & Context Dynamics
	InputTokens         prometheus.Counter
	OutputTokens        prometheus.Counter
	TotalTokens         prometheus.Counter
	ContextLengthP95    prometheus.Gauge
	ContextTruncations  prometheus.Counter
	KVCacheHitRatio     prometheus.Gauge
	BatchMergeEfficiency prometheus.Gauge

	// Tooling / Function Calls
	ToolCallsPerTurn    prometheus.Histogram
	ToolLatency         prometheus.Histogram
	ToolSuccessRate     prometheus.Gauge
	ToolTimeoutRate     prometheus.Gauge
	ToolRetryRate       prometheus.Gauge
	RetrievalLatency    prometheus.Histogram
	RetrievalCacheHit   prometheus.Gauge
	GroundingCoverage   prometheus.Gauge

	// RAG Quality
	RetrievalHitAtK      prometheus.Gauge
	RetrievalMRR         prometheus.Gauge
	HallucinationRate    prometheus.Gauge
	CitationValidityRate prometheus.Gauge

	// GPU & System Efficiency
	GPUUtilization        prometheus.Gauge
	SMUtilization         prometheus.Gauge
	MemoryBWUtilization   prometheus.Gauge
	VRAMUsed              prometheus.Gauge
	VRAMFragmentation     prometheus.Gauge
	MIGSliceUtilization   prometheus.Gauge
	NodeModelCacheHit     prometheus.Gauge
	ModelLoadTime         prometheus.Histogram
	SnapshotRestoreTime   prometheus.Histogram
	ColdStartRate         prometheus.Gauge

	// Network & Streaming
	StreamInitLatency      prometheus.Histogram
	StreamBackpressure     prometheus.Counter
	StreamDropRate         prometheus.Gauge
	StreamCancelRate       prometheus.Gauge
	TokenDeliveryJitter    prometheus.Histogram

	// Scheduler & Placement
	GangScheduleWait      prometheus.Histogram
	TopologyPenaltyScore  prometheus.Gauge
	SessionAffinityHitRate prometheus.Gauge
	DataLocalityRate      prometheus.Gauge

	// Autoscaling & Reliability
	HPADecisions          prometheus.Counter
	ReplicaPreemptions    prometheus.Counter
	ReplicaEvictions      prometheus.Counter
	SpotInterruptions     prometheus.Counter
	FailoverTime          prometheus.Histogram
	ErrorBudgetBurnRate   prometheus.Gauge

	// Security, Safety, Policy
	PolicyBlocks     prometheus.Counter
	RedactionEvents  prometheus.Counter
	AuthzDenials     prometheus.Counter

	// Cost & Carbon
	CostPer1KTokens      prometheus.Gauge
	CostPerSession       prometheus.Gauge
	GPUHours             prometheus.Counter
	CPUHours             prometheus.Counter
	EgressGB             prometheus.Counter
	EnergyKWHPer1KTokens prometheus.Gauge
	SpotSavings          prometheus.Counter

	// OpenTelemetry metrics
	otelMeter metric.Meter
}

// NewAgentMetrics creates and registers all Prometheus metrics
func NewAgentMetrics(registry prometheus.Registerer) *AgentMetrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	m := &AgentMetrics{
		// UX & Quality metrics
		TTFTHistogram: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_ttft_ms",
			Help:    "Time to first token in milliseconds",
			Buckets: []float64{50, 100, 200, 350, 500, 750, 1000, 2000, 5000},
		}),
		LatencyHistogram: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_latency_ms",
			Help:    "End-to-end turn latency in milliseconds",
			Buckets: []float64{100, 250, 500, 1000, 2500, 5000, 10000, 30000},
		}),
		RTFRatio: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_rtf_ratio",
			Help: "Real-time factor (generation time / output seconds)",
		}),
		TokensOutRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_tokens_out_per_s",
			Help: "Token generation rate (tokens/second)",
		}),
		CSATScore: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_csat_score",
			Help: "Customer satisfaction score (0-5)",
		}),
		ThumbsUpRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_thumbs_up_rate",
			Help: "Thumbs up rate (0-1)",
		}),
		TurnErrorRate: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_turn_errors_total",
			Help: "Total number of turn errors (5xx + aborted)",
		}),
		QualityWinRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_quality_winrate",
			Help: "Quality win rate for canary vs baseline",
		}),

		// Load & Concurrency
		ActiveSessions: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_active_sessions",
			Help: "Number of active sessions",
		}),
		QueueDepth: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_queue_depth",
			Help: "Current queue depth per route/topic",
		}),
		AdmissionRejects: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_admission_rejects_total",
			Help: "Total admission rejections due to SLO/capacity",
		}),
		ScalingLag: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_scaling_lag_seconds",
			Help:    "Time from load spike to replica ready",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		}),

		// Token & Context Dynamics
		InputTokens: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_input_tokens_total",
			Help: "Total input tokens processed",
		}),
		OutputTokens: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_output_tokens_total",
			Help: "Total output tokens generated",
		}),
		TotalTokens: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_total_tokens",
			Help: "Total tokens (input + output)",
		}),
		ContextLengthP95: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_ctx_len_p95",
			Help: "95th percentile context length",
		}),
		ContextTruncations: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "agent_ctx_truncations_total",
			Help: "Total context truncations",
		}),
		KVCacheHitRatio: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_kv_cache_hit_ratio",
			Help: "KV cache hit ratio",
		}),
		BatchMergeEfficiency: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_batch_merge_efficiency",
			Help: "Batch merge efficiency (effective / ideal)",
		}),

		// Tooling / Function Calls
		ToolCallsPerTurn: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_tool_calls_per_turn",
			Help:    "Number of tool calls per turn",
			Buckets: []float64{0, 1, 2, 3, 5, 10, 20},
		}),
		ToolLatency: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "agent_tool_latency_ms",
			Help:    "Tool call latency in milliseconds",
			Buckets: []float64{10, 50, 100, 200, 500, 800, 1000, 2000, 5000},
		}),
		ToolSuccessRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_tool_success_rate",
			Help: "Tool call success rate",
		}),
		ToolTimeoutRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_tool_timeout_rate",
			Help: "Tool call timeout rate",
		}),
		ToolRetryRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_tool_retry_rate",
			Help: "Tool call retry rate",
		}),
		RetrievalLatency: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "rag_retrieval_latency_ms",
			Help:    "RAG retrieval latency in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 200, 500, 1000},
		}),
		RetrievalCacheHit: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "rag_retrieval_cache_hit_ratio",
			Help: "RAG retrieval cache hit ratio",
		}),
		GroundingCoverage: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_grounding_coverage",
			Help: "Percentage of turns with citations",
		}),

		// RAG Quality
		RetrievalHitAtK: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "rag_hit_at_k",
			Help: "Retrieval hit@k metric",
		}),
		RetrievalMRR: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "rag_mrr",
			Help: "Retrieval Mean Reciprocal Rank",
		}),
		HallucinationRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_hallucination_rate",
			Help: "Hallucination proxy rate (no-source spans)",
		}),
		CitationValidityRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_citation_validity_rate",
			Help: "Citation validity rate (post-hoc check)",
		}),

		// GPU & System Efficiency
		GPUUtilization: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_util_pct",
			Help: "GPU utilization percentage",
		}),
		SMUtilization: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_sm_util_pct",
			Help: "GPU SM utilization percentage",
		}),
		MemoryBWUtilization: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_mem_bw_util_pct",
			Help: "GPU memory bandwidth utilization percentage",
		}),
		VRAMUsed: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_vram_used_gb",
			Help: "GPU VRAM used in GB",
		}),
		VRAMFragmentation: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_vram_frag_pct",
			Help: "GPU VRAM fragmentation percentage",
		}),
		MIGSliceUtilization: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "gpu_mig_slice_util_pct",
			Help: "MIG slice utilization percentage",
		}),
		NodeModelCacheHit: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "model_cache_hit_ratio",
			Help: "Node model cache hit ratio",
		}),
		ModelLoadTime: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "model_load_time_seconds",
			Help:    "Model loading time in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		}),
		SnapshotRestoreTime: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "model_snapshot_restore_seconds",
			Help:    "Model snapshot restore time in seconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
		}),
		ColdStartRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "agent_cold_start_rate",
			Help: "Replica cold start rate",
		}),

		// Network & Streaming
		StreamInitLatency: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "stream_init_ms",
			Help:    "Stream initialization latency in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 200, 500},
		}),
		StreamBackpressure: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "stream_backpressure_events_total",
			Help: "Total stream backpressure events",
		}),
		StreamDropRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "stream_drop_rate",
			Help: "Stream drop rate",
		}),
		StreamCancelRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "stream_cancel_rate",
			Help: "Stream cancellation rate",
		}),
		TokenDeliveryJitter: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "token_delivery_jitter_ms",
			Help:    "Token delivery jitter in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 200},
		}),

		// Scheduler & Placement
		GangScheduleWait: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "gang_schedule_wait_seconds",
			Help:    "Gang scheduling wait time in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
		}),
		TopologyPenaltyScore: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "topology_penalty_score",
			Help: "Topology penalty score for suboptimal placement",
		}),
		SessionAffinityHitRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "session_affinity_hit_ratio",
			Help: "Session affinity hit ratio",
		}),
		DataLocalityRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "data_locality_rate",
			Help: "Data locality rate (agent colocated with shard)",
		}),

		// Autoscaling & Reliability
		HPADecisions: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "hpa_decisions_total",
			Help: "Total HPA/KEDA decisions",
		}),
		ReplicaPreemptions: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "replica_preemptions_total",
			Help: "Total replica preemptions",
		}),
		ReplicaEvictions: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "replica_evictions_total",
			Help: "Total replica evictions",
		}),
		SpotInterruptions: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "spot_interruptions_total",
			Help: "Total spot instance interruptions",
		}),
		FailoverTime: promauto.With(registry).NewHistogram(prometheus.HistogramOpts{
			Name:    "failover_time_seconds",
			Help:    "Failover time in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120},
		}),
		ErrorBudgetBurnRate: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "error_budget_burn_rate",
			Help: "Error budget burn rate per SLO",
		}),

		// Security, Safety, Policy
		PolicyBlocks: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "policy_blocks_total",
			Help: "Total policy blocks (safety/PII filters)",
		}),
		RedactionEvents: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "redaction_events_total",
			Help: "Total redaction events",
		}),
		AuthzDenials: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "authz_denials_total",
			Help: "Total authorization denials (tool scope violations)",
		}),

		// Cost & Carbon
		CostPer1KTokens: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "cost_usd_per_1k_tokens",
			Help: "Cost per 1000 tokens in USD",
		}),
		CostPerSession: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "cost_usd_per_session",
			Help: "Cost per session in USD",
		}),
		GPUHours: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "gpu_hours_total",
			Help: "Total GPU hours consumed",
		}),
		CPUHours: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "cpu_hours_total",
			Help: "Total CPU hours consumed",
		}),
		EgressGB: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "egress_gb_total",
			Help: "Total egress in GB",
		}),
		EnergyKWHPer1KTokens: promauto.With(registry).NewGauge(prometheus.GaugeOpts{
			Name: "energy_kwh_per_1k_tokens",
			Help: "Energy consumption per 1000 tokens in kWh",
		}),
		SpotSavings: promauto.With(registry).NewCounter(prometheus.CounterOpts{
			Name: "spot_savings_usd_total",
			Help: "Total spot instance savings in USD (vs on-demand)",
		}),
	}

	// Initialize OpenTelemetry meter
	m.otelMeter = otel.Meter("neuronetes.ai/metrics")

	return m
}

// RecordTTFT records time-to-first-token metric
func (m *AgentMetrics) RecordTTFT(ctx context.Context, ttft time.Duration, model, route string) {
	m.TTFTHistogram.Observe(float64(ttft.Milliseconds()))
}

// RecordLatency records end-to-end latency
func (m *AgentMetrics) RecordLatency(ctx context.Context, latency time.Duration, model, route string) {
	m.LatencyHistogram.Observe(float64(latency.Milliseconds()))
}

// RecordTokens records token usage
func (m *AgentMetrics) RecordTokens(ctx context.Context, inputTokens, outputTokens int64, model string) {
	m.InputTokens.Add(float64(inputTokens))
	m.OutputTokens.Add(float64(outputTokens))
	m.TotalTokens.Add(float64(inputTokens + outputTokens))
}

// RecordToolCall records tool call metrics
func (m *AgentMetrics) RecordToolCall(ctx context.Context, toolName string, latency time.Duration, success bool) {
	m.ToolLatency.Observe(float64(latency.Milliseconds()))
	if !success {
		m.ToolTimeoutRate.Inc()
	}
}

// RecordError records error metrics
func (m *AgentMetrics) RecordError(ctx context.Context, errorType, model string) {
	m.TurnErrorRate.Inc()
}

// RecordCost records cost metrics
func (m *AgentMetrics) RecordCost(ctx context.Context, costUSD float64, tokens int64, model, tenant string) {
	if tokens > 0 {
		costPer1K := (costUSD / float64(tokens)) * 1000
		m.CostPer1KTokens.Set(costPer1K)
	}
}

// SetActiveSessions updates active session count
func (m *AgentMetrics) SetActiveSessions(count int) {
	m.ActiveSessions.Set(float64(count))
}

// SetQueueDepth updates queue depth
func (m *AgentMetrics) SetQueueDepth(depth int, route string) {
	m.QueueDepth.Set(float64(depth))
}

// RecordGPUMetrics records GPU utilization metrics
func (m *AgentMetrics) RecordGPUMetrics(ctx context.Context, node string, gpuUtil, vramUsed, vramTotal float64) {
	m.GPUUtilization.Set(gpuUtil)
	m.VRAMUsed.Set(vramUsed)
	if vramTotal > 0 {
		m.VRAMFragmentation.Set((vramTotal - vramUsed) / vramTotal * 100)
	}
}

// RecordModelLoad records model loading time
func (m *AgentMetrics) RecordModelLoad(ctx context.Context, modelName string, loadTime time.Duration, fromCache bool) {
	m.ModelLoadTime.Observe(loadTime.Seconds())
	if fromCache {
		m.NodeModelCacheHit.Set(1.0)
	} else {
		m.NodeModelCacheHit.Set(0.0)
	}
}

// RecordScalingEvent records autoscaling event
func (m *AgentMetrics) RecordScalingEvent(ctx context.Context, reason string, lagSeconds float64) {
	m.HPADecisions.Inc()
	m.ScalingLag.Observe(lagSeconds)
}

// RecordPolicyBlock records policy enforcement
func (m *AgentMetrics) RecordPolicyBlock(ctx context.Context, policyType, reason string) {
	m.PolicyBlocks.Inc()
}

// RecordRedaction records PII redaction
func (m *AgentMetrics) RecordRedaction(ctx context.Context, fieldType string) {
	m.RedactionEvents.Inc()
}

// MetricsLabels defines common label structure
type MetricsLabels struct {
	Model      string
	Route      string
	Tool       string
	Node       string
	Tenant     string
	AgentClass string
	AgentPool  string
}

// WithLabels returns attribute.Set for OpenTelemetry
func (l *MetricsLabels) WithLabels() attribute.Set {
	attrs := []attribute.KeyValue{}
	if l.Model != "" {
		attrs = append(attrs, attribute.String("model", l.Model))
	}
	if l.Route != "" {
		attrs = append(attrs, attribute.String("route", l.Route))
	}
	if l.Tool != "" {
		attrs = append(attrs, attribute.String("tool", l.Tool))
	}
	if l.Node != "" {
		attrs = append(attrs, attribute.String("node", l.Node))
	}
	if l.Tenant != "" {
		attrs = append(attrs, attribute.String("tenant", l.Tenant))
	}
	if l.AgentClass != "" {
		attrs = append(attrs, attribute.String("agentclass", l.AgentClass))
	}
	if l.AgentPool != "" {
		attrs = append(attrs, attribute.String("agentpool", l.AgentPool))
	}
	return attribute.NewSet(attrs...)
}
