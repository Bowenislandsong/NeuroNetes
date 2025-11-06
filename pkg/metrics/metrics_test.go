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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	require.NotNil(t, metrics)
	require.NotNil(t, metrics.TTFTHistogram)
	require.NotNil(t, metrics.LatencyHistogram)
	require.NotNil(t, metrics.InputTokens)
	require.NotNil(t, metrics.OutputTokens)
}

func TestRecordTTFT(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name  string
		ttft  time.Duration
		model string
		route string
	}{
		{
			name:  "fast ttft",
			ttft:  100 * time.Millisecond,
			model: "llama-3-70b",
			route: "/chat",
		},
		{
			name:  "slow ttft",
			ttft:  2 * time.Second,
			model: "llama-3-8b",
			route: "/complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordTTFT(ctx, tt.ttft, tt.model, tt.route)

			count := testutil.CollectAndCount(metrics.TTFTHistogram)
			assert.Greater(t, count, 0, "TTFT histogram should have observations")
		})
	}
}

func TestRecordLatency(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	ctx := context.Background()
	metrics.RecordLatency(ctx, 500*time.Millisecond, "llama-3-70b", "/chat")

	count := testutil.CollectAndCount(metrics.LatencyHistogram)
	assert.Greater(t, count, 0, "Latency histogram should have observations")
}

func TestRecordTokens(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		model        string
		expectedTotal int64
	}{
		{
			name:          "small conversation",
			inputTokens:   100,
			outputTokens:  50,
			model:         "llama-3-8b",
			expectedTotal: 150,
		},
		{
			name:          "large conversation",
			inputTokens:   10000,
			outputTokens:  5000,
			model:         "llama-3-70b",
			expectedTotal: 15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordTokens(ctx, tt.inputTokens, tt.outputTokens, tt.model)

			// Verify metrics were recorded
			inputVal := testutil.ToFloat64(metrics.InputTokens)
			outputVal := testutil.ToFloat64(metrics.OutputTokens)
			totalVal := testutil.ToFloat64(metrics.TotalTokens)

			assert.Greater(t, inputVal, float64(0))
			assert.Greater(t, outputVal, float64(0))
			assert.Greater(t, totalVal, float64(0))
		})
	}
}

func TestRecordToolCall(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name     string
		toolName string
		latency  time.Duration
		success  bool
	}{
		{
			name:     "successful tool call",
			toolName: "code_search",
			latency:  100 * time.Millisecond,
			success:  true,
		},
		{
			name:     "failed tool call",
			toolName: "web_search",
			latency:  5 * time.Second,
			success:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordToolCall(ctx, tt.toolName, tt.latency, tt.success)

			count := testutil.CollectAndCount(metrics.ToolLatency)
			assert.Greater(t, count, 0, "Tool latency should be recorded")
		})
	}
}

func TestRecordCost(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name            string
		costUSD         float64
		tokens          int64
		model           string
		tenant          string
		expectedCostPer1K float64
	}{
		{
			name:              "standard cost",
			costUSD:           0.10,
			tokens:            1000,
			model:             "llama-3-70b",
			tenant:            "tenant-1",
			expectedCostPer1K: 0.10, // $0.10 per 1K tokens
		},
		{
			name:              "high cost",
			costUSD:           1.00,
			tokens:            5000,
			model:             "gpt-4",
			tenant:            "tenant-2",
			expectedCostPer1K: 0.20, // $0.20 per 1K tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordCost(ctx, tt.costUSD, tt.tokens, tt.model, tt.tenant)

			costPer1K := testutil.ToFloat64(metrics.CostPer1KTokens)
			assert.InDelta(t, tt.expectedCostPer1K, costPer1K, 0.01)
		})
	}
}

func TestSetActiveSessions(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	metrics.SetActiveSessions(42)

	value := testutil.ToFloat64(metrics.ActiveSessions)
	assert.Equal(t, float64(42), value)

	metrics.SetActiveSessions(0)
	value = testutil.ToFloat64(metrics.ActiveSessions)
	assert.Equal(t, float64(0), value)
}

func TestSetQueueDepth(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	metrics.SetQueueDepth(100, "/chat")

	value := testutil.ToFloat64(metrics.QueueDepth)
	assert.Equal(t, float64(100), value)
}

func TestRecordGPUMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name       string
		node       string
		gpuUtil    float64
		vramUsed   float64
		vramTotal  float64
		expectedFrag float64
	}{
		{
			name:         "healthy GPU",
			node:         "node-1",
			gpuUtil:      85.0,
			vramUsed:     60.0,
			vramTotal:    80.0,
			expectedFrag: 25.0, // (80-60)/80 * 100
		},
		{
			name:         "full GPU",
			node:         "node-2",
			gpuUtil:      100.0,
			vramUsed:     80.0,
			vramTotal:    80.0,
			expectedFrag: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordGPUMetrics(ctx, tt.node, tt.gpuUtil, tt.vramUsed, tt.vramTotal)

			gpuUtil := testutil.ToFloat64(metrics.GPUUtilization)
			vramUsed := testutil.ToFloat64(metrics.VRAMUsed)
			vramFrag := testutil.ToFloat64(metrics.VRAMFragmentation)

			assert.Equal(t, tt.gpuUtil, gpuUtil)
			assert.Equal(t, tt.vramUsed, vramUsed)
			assert.InDelta(t, tt.expectedFrag, vramFrag, 0.01)
		})
	}
}

func TestRecordModelLoad(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	tests := []struct {
		name          string
		modelName     string
		loadTime      time.Duration
		fromCache     bool
		expectedCache float64
	}{
		{
			name:          "cache hit",
			modelName:     "llama-3-70b",
			loadTime:      1 * time.Second,
			fromCache:     true,
			expectedCache: 1.0,
		},
		{
			name:          "cache miss",
			modelName:     "llama-3-8b",
			loadTime:      60 * time.Second,
			fromCache:     false,
			expectedCache: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			metrics.RecordModelLoad(ctx, tt.modelName, tt.loadTime, tt.fromCache)

			count := testutil.CollectAndCount(metrics.ModelLoadTime)
			assert.Greater(t, count, 0)

			cacheHit := testutil.ToFloat64(metrics.NodeModelCacheHit)
			assert.Equal(t, tt.expectedCache, cacheHit)
		})
	}
}

func TestRecordScalingEvent(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	ctx := context.Background()
	metrics.RecordScalingEvent(ctx, "high_queue_depth", 15.5)

	decisions := testutil.ToFloat64(metrics.HPADecisions)
	assert.Greater(t, decisions, float64(0))

	count := testutil.CollectAndCount(metrics.ScalingLag)
	assert.Greater(t, count, 0)
}

func TestRecordPolicyBlock(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	ctx := context.Background()
	metrics.RecordPolicyBlock(ctx, "pii-detection", "credit_card_detected")

	blocks := testutil.ToFloat64(metrics.PolicyBlocks)
	assert.Greater(t, blocks, float64(0))
}

func TestRecordRedaction(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	ctx := context.Background()
	metrics.RecordRedaction(ctx, "email")

	redactions := testutil.ToFloat64(metrics.RedactionEvents)
	assert.Greater(t, redactions, float64(0))
}

func TestMetricsLabels(t *testing.T) {
	labels := &MetricsLabels{
		Model:      "llama-3-70b",
		Route:      "/chat",
		Tool:       "code_search",
		Node:       "node-1",
		Tenant:     "tenant-1",
		AgentClass: "code-assistant",
		AgentPool:  "main-pool",
	}

	attrSet := labels.WithLabels()
	assert.NotNil(t, attrSet)

	// Verify all labels are present
	assert.True(t, attrSet.HasValue("model"))
	assert.True(t, attrSet.HasValue("route"))
	assert.True(t, attrSet.HasValue("tool"))
	assert.True(t, attrSet.HasValue("node"))
	assert.True(t, attrSet.HasValue("tenant"))
	assert.True(t, attrSet.HasValue("agentclass"))
	assert.True(t, attrSet.HasValue("agentpool"))
}

func TestConcurrentMetricsRecording(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)

	ctx := context.Background()
	done := make(chan bool)

	// Simulate concurrent metric recording
	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics.RecordTTFT(ctx, time.Duration(id*100)*time.Millisecond, "llama-3-70b", "/chat")
			metrics.RecordTokens(ctx, int64(id*100), int64(id*50), "llama-3-70b")
			metrics.SetActiveSessions(id)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics were recorded
	count := testutil.CollectAndCount(metrics.TTFTHistogram)
	assert.Greater(t, count, 0)

	tokens := testutil.ToFloat64(metrics.TotalTokens)
	assert.Greater(t, tokens, float64(0))
}

func BenchmarkRecordTTFT(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
	}
}

func BenchmarkRecordTokens(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordTokens(ctx, 1000, 500, "llama-3-70b")
	}
}

func BenchmarkRecordGPUMetrics(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewAgentMetrics(registry)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordGPUMetrics(ctx, "node-1", 85.0, 60.0, 80.0)
	}
}
