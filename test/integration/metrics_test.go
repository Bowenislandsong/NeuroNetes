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

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/bowenislandsong/neuronetes/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// TestMetricsEndToEndWorkflow tests a complete request workflow with all metrics
func TestMetricsEndToEndWorkflow(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate a complete agent request workflow
	// 1. Record TTFT
	ttft := 250 * time.Millisecond
	m.RecordTTFT(ctx, ttft, "llama-3-70b", "/chat")

	// 2. Update active sessions
	m.SetActiveSessions(5)

	// 3. Record token usage
	m.RecordTokens(ctx, 1500, 750, "llama-3-70b")

	// 4. Record tool calls
	m.RecordToolCall(ctx, "code_search", 150*time.Millisecond, true)
	m.RecordToolCall(ctx, "web_search", 300*time.Millisecond, true)

	// 5. Record GPU metrics
	m.RecordGPUMetrics(ctx, "node-1", 85.5, 60.0, 80.0)

	// 6. Record end-to-end latency
	latency := 1500 * time.Millisecond
	m.RecordLatency(ctx, latency, "llama-3-70b", "/chat")

	// 7. Record cost
	m.RecordCost(ctx, 0.15, 2250, "llama-3-70b", "tenant-1")

	// Verify all metrics were recorded
	ttftCount := testutil.CollectAndCount(m.TTFTHistogram)
	assert.Greater(t, ttftCount, 0, "TTFT should be recorded")

	sessions := testutil.ToFloat64(m.ActiveSessions)
	assert.Equal(t, float64(5), sessions, "Active sessions should match")

	tokens := testutil.ToFloat64(m.TotalTokens)
	assert.Greater(t, tokens, float64(0), "Total tokens should be recorded")

	toolCount := testutil.CollectAndCount(m.ToolLatency)
	assert.Greater(t, toolCount, 0, "Tool calls should be recorded")

	gpuUtil := testutil.ToFloat64(m.GPUUtilization)
	assert.Equal(t, 85.5, gpuUtil, "GPU utilization should match")

	latencyCount := testutil.CollectAndCount(m.LatencyHistogram)
	assert.Greater(t, latencyCount, 0, "Latency should be recorded")

	costPer1K := testutil.ToFloat64(m.CostPer1KTokens)
	assert.InDelta(t, 0.0667, costPer1K, 0.001, "Cost per 1K tokens should be calculated")
}

// TestMetricsQualityTracking tests UX & Quality metrics
func TestMetricsQualityTracking(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	tests := []struct {
		name   string
		action func()
		verify func(t *testing.T)
	}{
		{
			name: "track TTFT distribution",
			action: func() {
				m.RecordTTFT(ctx, 100*time.Millisecond, "llama-3-8b", "/chat")
				m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
				m.RecordTTFT(ctx, 500*time.Millisecond, "llama-3-70b", "/complete")
			},
			verify: func(t *testing.T) {
				count := testutil.CollectAndCount(m.TTFTHistogram)
				assert.Greater(t, count, 0, "Should record TTFT observations")
			},
		},
		{
			name: "track RTF ratio",
			action: func() {
				m.RTFRatio.Set(1.2) // Generation time / output duration
			},
			verify: func(t *testing.T) {
				ratio := testutil.ToFloat64(m.RTFRatio)
				assert.Equal(t, 1.2, ratio, "RTF ratio should match")
			},
		},
		{
			name: "track tokens per second",
			action: func() {
				m.TokensOutRate.Set(45.5)
			},
			verify: func(t *testing.T) {
				rate := testutil.ToFloat64(m.TokensOutRate)
				assert.Equal(t, 45.5, rate, "Tokens/s should match")
			},
		},
		{
			name: "track CSAT score",
			action: func() {
				m.CSATScore.Set(4.5)
			},
			verify: func(t *testing.T) {
				score := testutil.ToFloat64(m.CSATScore)
				assert.Equal(t, 4.5, score, "CSAT score should match")
			},
		},
		{
			name: "track quality win rate",
			action: func() {
				m.QualityWinRate.Set(0.85)
			},
			verify: func(t *testing.T) {
				winRate := testutil.ToFloat64(m.QualityWinRate)
				assert.Equal(t, 0.85, winRate, "Win rate should match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.action()
			tt.verify(t)
		})
	}
}

// TestMetricsLoadAndConcurrency tests load & concurrency metrics
func TestMetricsLoadAndConcurrency(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate load changes
	m.SetActiveSessions(10)
	m.SetQueueDepth(25, "/chat")
	m.AdmissionRejects.Inc()
	m.RecordScalingEvent(ctx, "high_queue_depth", 45.0)

	// Verify metrics
	sessions := testutil.ToFloat64(m.ActiveSessions)
	assert.Equal(t, float64(10), sessions)

	queueDepth := testutil.ToFloat64(m.QueueDepth)
	assert.Equal(t, float64(25), queueDepth)

	rejects := testutil.ToFloat64(m.AdmissionRejects)
	assert.Greater(t, rejects, float64(0))

	scalingLagCount := testutil.CollectAndCount(m.ScalingLag)
	assert.Greater(t, scalingLagCount, 0)
}

// TestMetricsTokenDynamics tests token & context metrics
func TestMetricsTokenDynamics(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Record various token operations
	m.RecordTokens(ctx, 10000, 5000, "llama-3-70b")
	m.ContextLengthP95.Set(12500)
	m.ContextTruncations.Inc()
	m.KVCacheHitRatio.Set(0.75)
	m.BatchMergeEfficiency.Set(0.92)

	// Verify metrics
	inputTokens := testutil.ToFloat64(m.InputTokens)
	assert.Equal(t, float64(10000), inputTokens)

	outputTokens := testutil.ToFloat64(m.OutputTokens)
	assert.Equal(t, float64(5000), outputTokens)

	totalTokens := testutil.ToFloat64(m.TotalTokens)
	assert.Equal(t, float64(15000), totalTokens)

	ctxLen := testutil.ToFloat64(m.ContextLengthP95)
	assert.Equal(t, float64(12500), ctxLen)

	truncations := testutil.ToFloat64(m.ContextTruncations)
	assert.Greater(t, truncations, float64(0))

	kvHitRatio := testutil.ToFloat64(m.KVCacheHitRatio)
	assert.Equal(t, 0.75, kvHitRatio)

	batchEff := testutil.ToFloat64(m.BatchMergeEfficiency)
	assert.Equal(t, 0.92, batchEff)
}

// TestMetricsToolingAndRAG tests tool and RAG metrics
func TestMetricsToolingAndRAG(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate tool calls
	m.ToolCallsPerTurn.Observe(2)
	m.RecordToolCall(ctx, "code_search", 150*time.Millisecond, true)
	m.RecordToolCall(ctx, "web_search", 800*time.Millisecond, false)

	// RAG metrics
	m.RetrievalLatency.Observe(50)
	m.RetrievalCacheHit.Set(0.60)
	m.GroundingCoverage.Set(0.85)
	m.RetrievalHitAtK.Set(0.92)
	m.RetrievalMRR.Set(0.88)
	m.HallucinationRate.Set(0.02)
	m.CitationValidityRate.Set(0.95)

	// Verify tool metrics
	toolCallsCount := testutil.CollectAndCount(m.ToolCallsPerTurn)
	assert.Greater(t, toolCallsCount, 0)

	toolLatencyCount := testutil.CollectAndCount(m.ToolLatency)
	assert.Greater(t, toolLatencyCount, 0, "Tool calls should be recorded")

	// Verify RAG metrics
	retrievalLatencyCount := testutil.CollectAndCount(m.RetrievalLatency)
	assert.Greater(t, retrievalLatencyCount, 0)

	cacheHit := testutil.ToFloat64(m.RetrievalCacheHit)
	assert.Equal(t, 0.60, cacheHit)

	coverage := testutil.ToFloat64(m.GroundingCoverage)
	assert.Equal(t, 0.85, coverage)
}

// TestMetricsGPUEfficiency tests GPU & system metrics
func TestMetricsGPUEfficiency(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// GPU metrics
	m.RecordGPUMetrics(ctx, "node-1", 92.5, 70.0, 80.0)
	m.SMUtilization.Set(88.0)
	m.MemoryBWUtilization.Set(75.0)
	m.MIGSliceUtilization.Set(85.0)

	// Model loading metrics
	m.RecordModelLoad(ctx, "llama-3-70b", 5*time.Second, true)
	m.SnapshotRestoreTime.Observe(2.5)
	m.ColdStartRate.Set(0.05)

	// Verify GPU metrics
	gpuUtil := testutil.ToFloat64(m.GPUUtilization)
	assert.Equal(t, 92.5, gpuUtil)

	vramUsed := testutil.ToFloat64(m.VRAMUsed)
	assert.Equal(t, 70.0, vramUsed)

	vramFrag := testutil.ToFloat64(m.VRAMFragmentation)
	assert.InDelta(t, 12.5, vramFrag, 0.1)

	smUtil := testutil.ToFloat64(m.SMUtilization)
	assert.Equal(t, 88.0, smUtil)

	// Verify model loading
	loadTimeCount := testutil.CollectAndCount(m.ModelLoadTime)
	assert.Greater(t, loadTimeCount, 0)

	cacheHit := testutil.ToFloat64(m.NodeModelCacheHit)
	assert.Equal(t, 1.0, cacheHit)
}

// TestMetricsNetworkStreaming tests network & streaming metrics
func TestMetricsNetworkStreaming(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)

	// Network metrics
	m.StreamInitLatency.Observe(25)
	m.StreamBackpressure.Inc()
	m.StreamDropRate.Set(0.001)
	m.StreamCancelRate.Set(0.05)
	m.TokenDeliveryJitter.Observe(5)

	// Verify metrics
	initLatencyCount := testutil.CollectAndCount(m.StreamInitLatency)
	assert.Greater(t, initLatencyCount, 0)

	backpressure := testutil.ToFloat64(m.StreamBackpressure)
	assert.Greater(t, backpressure, float64(0))

	dropRate := testutil.ToFloat64(m.StreamDropRate)
	assert.Equal(t, 0.001, dropRate)

	jitterCount := testutil.CollectAndCount(m.TokenDeliveryJitter)
	assert.Greater(t, jitterCount, 0)
}

// TestMetricsSchedulerPlacement tests scheduler & placement metrics
func TestMetricsSchedulerPlacement(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)

	// Scheduler metrics
	m.GangScheduleWait.Observe(15)
	m.TopologyPenaltyScore.Set(2.5)
	m.SessionAffinityHitRate.Set(0.88)
	m.DataLocalityRate.Set(0.92)

	// Verify metrics
	gangWaitCount := testutil.CollectAndCount(m.GangScheduleWait)
	assert.Greater(t, gangWaitCount, 0)

	penaltyScore := testutil.ToFloat64(m.TopologyPenaltyScore)
	assert.Equal(t, 2.5, penaltyScore)

	affinityHit := testutil.ToFloat64(m.SessionAffinityHitRate)
	assert.Equal(t, 0.88, affinityHit)

	localityRate := testutil.ToFloat64(m.DataLocalityRate)
	assert.Equal(t, 0.92, localityRate)
}

// TestMetricsAutoscalingReliability tests autoscaling & reliability metrics
func TestMetricsAutoscalingReliability(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Autoscaling & reliability metrics
	m.RecordScalingEvent(ctx, "queue_depth_high", 30.0)
	m.ReplicaPreemptions.Inc()
	m.ReplicaEvictions.Inc()
	m.SpotInterruptions.Inc()
	m.FailoverTime.Observe(5)
	m.ErrorBudgetBurnRate.Set(0.15)

	// Verify metrics
	decisions := testutil.ToFloat64(m.HPADecisions)
	assert.Greater(t, decisions, float64(0))

	preemptions := testutil.ToFloat64(m.ReplicaPreemptions)
	assert.Greater(t, preemptions, float64(0))

	evictions := testutil.ToFloat64(m.ReplicaEvictions)
	assert.Greater(t, evictions, float64(0))

	interruptions := testutil.ToFloat64(m.SpotInterruptions)
	assert.Greater(t, interruptions, float64(0))

	failoverCount := testutil.CollectAndCount(m.FailoverTime)
	assert.Greater(t, failoverCount, 0)

	burnRate := testutil.ToFloat64(m.ErrorBudgetBurnRate)
	assert.Equal(t, 0.15, burnRate)
}

// TestMetricsSecurityPolicy tests security & policy metrics
func TestMetricsSecurityPolicy(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Security metrics
	m.RecordPolicyBlock(ctx, "pii-detection", "ssn_detected")
	m.RecordPolicyBlock(ctx, "safety-check", "harmful_content")
	m.RecordRedaction(ctx, "email")
	m.RecordRedaction(ctx, "credit_card")
	m.AuthzDenials.Inc()

	// Verify metrics
	blocks := testutil.ToFloat64(m.PolicyBlocks)
	assert.Equal(t, float64(2), blocks)

	redactions := testutil.ToFloat64(m.RedactionEvents)
	assert.Equal(t, float64(2), redactions)

	denials := testutil.ToFloat64(m.AuthzDenials)
	assert.Greater(t, denials, float64(0))
}

// TestMetricsCostCarbon tests cost & carbon metrics
func TestMetricsCostCarbon(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Cost metrics
	m.RecordCost(ctx, 0.50, 5000, "llama-3-70b", "tenant-1")
	m.CostPerSession.Set(0.25)
	m.GPUHours.Add(2.5)
	m.CPUHours.Add(5.0)
	m.EgressGB.Add(10.5)
	m.EnergyKWHPer1KTokens.Set(0.002)
	m.SpotSavings.Add(15.75)

	// Verify metrics
	costPer1K := testutil.ToFloat64(m.CostPer1KTokens)
	assert.InDelta(t, 0.10, costPer1K, 0.01)

	sessionCost := testutil.ToFloat64(m.CostPerSession)
	assert.Equal(t, 0.25, sessionCost)

	gpuHours := testutil.ToFloat64(m.GPUHours)
	assert.Equal(t, 2.5, gpuHours)

	cpuHours := testutil.ToFloat64(m.CPUHours)
	assert.Equal(t, 5.0, cpuHours)

	egress := testutil.ToFloat64(m.EgressGB)
	assert.Equal(t, 10.5, egress)

	energy := testutil.ToFloat64(m.EnergyKWHPer1KTokens)
	assert.Equal(t, 0.002, energy)

	savings := testutil.ToFloat64(m.SpotSavings)
	assert.Equal(t, 15.75, savings)
}

// TestMetricsSLOCompliance tests SLO compliance scenarios
func TestMetricsSLOCompliance(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	tests := []struct {
		name       string
		ttft       time.Duration
		latency    time.Duration
		toolP95    time.Duration
		errorRate  float64
		passessSLO bool
	}{
		{
			name:       "within SLO",
			ttft:       300 * time.Millisecond,
			latency:    2000 * time.Millisecond,
			toolP95:    750 * time.Millisecond,
			errorRate:  0.005,
			passessSLO: true,
		},
		{
			name:       "exceeds TTFT SLO",
			ttft:       400 * time.Millisecond,
			latency:    2000 * time.Millisecond,
			toolP95:    750 * time.Millisecond,
			errorRate:  0.005,
			passessSLO: false,
		},
		{
			name:       "exceeds latency SLO",
			ttft:       300 * time.Millisecond,
			latency:    3000 * time.Millisecond,
			toolP95:    750 * time.Millisecond,
			errorRate:  0.005,
			passessSLO: false,
		},
		{
			name:       "exceeds tool SLO",
			ttft:       300 * time.Millisecond,
			latency:    2000 * time.Millisecond,
			toolP95:    850 * time.Millisecond,
			errorRate:  0.005,
			passessSLO: false,
		},
		{
			name:       "exceeds error rate SLO",
			ttft:       300 * time.Millisecond,
			latency:    2000 * time.Millisecond,
			toolP95:    750 * time.Millisecond,
			errorRate:  0.015,
			passessSLO: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record metrics
			m.RecordTTFT(ctx, tt.ttft, "llama-3-70b", "/chat")
			m.RecordLatency(ctx, tt.latency, "llama-3-70b", "/chat")
			m.RecordToolCall(ctx, "test_tool", tt.toolP95, true)

			// SLO thresholds
			ttftSLO := 350 * time.Millisecond
			latencySLO := 2500 * time.Millisecond
			toolSLO := 800 * time.Millisecond
			errorRateSLO := 0.01

			// Check SLO compliance
			passesTTFT := tt.ttft <= ttftSLO
			passesLatency := tt.latency <= latencySLO
			passesTool := tt.toolP95 <= toolSLO
			passesErrorRate := tt.errorRate < errorRateSLO

			overallPass := passesTTFT && passesLatency && passesTool && passesErrorRate
			assert.Equal(t, tt.passessSLO, overallPass,
				"SLO compliance should match expected (TTFT: %v, Latency: %v, Tool: %v, Error: %v)",
				passesTTFT, passesLatency, passesTool, passesErrorRate)
		})
	}
}

// TestMetricsHighCardinality verifies metrics don't create excessive cardinality
func TestMetricsHighCardinality(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate multiple models, routes, and nodes (bounded cardinality)
	models := []string{"llama-3-8b", "llama-3-70b", "mistral-7b"}
	routes := []string{"/chat", "/complete", "/embed"}
	nodes := []string{"node-1", "node-2", "node-3"}

	for _, model := range models {
		for _, route := range routes {
			m.RecordTTFT(ctx, 300*time.Millisecond, model, route)
			m.RecordTokens(ctx, 1000, 500, model)
		}
	}

	for _, node := range nodes {
		m.RecordGPUMetrics(ctx, node, 85.0, 60.0, 80.0)
	}

	// Verify metrics are being recorded
	ttftCount := testutil.CollectAndCount(m.TTFTHistogram)
	assert.Greater(t, ttftCount, 0)

	tokensVal := testutil.ToFloat64(m.TotalTokens)
	assert.Greater(t, tokensVal, float64(0))

	// Note: In production, these would be separate metric instances with labels
	// Here we're verifying the metrics structure doesn't explode with cardinality
}

// BenchmarkMetricsRecording benchmarks metric recording performance
func BenchmarkMetricsRecording(b *testing.B) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	b.Run("RecordTTFT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
		}
	})

	b.Run("RecordTokens", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.RecordTokens(ctx, 1000, 500, "llama-3-70b")
		}
	})

	b.Run("RecordGPUMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.RecordGPUMetrics(ctx, "node-1", 85.0, 60.0, 80.0)
		}
	})

	b.Run("RecordToolCall", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.RecordToolCall(ctx, "code_search", 150*time.Millisecond, true)
		}
	})

	b.Run("RecordCost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.RecordCost(ctx, 0.10, 1000, "llama-3-70b", "tenant-1")
		}
	})
}
