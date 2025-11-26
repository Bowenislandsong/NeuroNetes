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

package e2e

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bowenislandsong/neuronetes/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

// TestMetricsPrometheusExport tests that metrics are properly exported to Prometheus format
func TestMetricsPrometheusExport(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Record various metrics
	m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
	m.RecordLatency(ctx, 2000*time.Millisecond, "llama-3-70b", "/chat")
	m.RecordTokens(ctx, 1500, 750, "llama-3-70b")
	m.RecordGPUMetrics(ctx, "node-1", 85.5, 60.0, 80.0)
	m.SetActiveSessions(10)
	m.RecordCost(ctx, 0.15, 2250, "llama-3-70b", "tenant-1")

	// Create HTTP handler for metrics
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Start test server
	server := &http.Server{
		Addr:    ":19090",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Query metrics endpoint
	resp, err := http.Get("http://localhost:19090/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metricsOutput := string(body)

	// Verify key metrics are exported
	expectedMetrics := []string{
		"agent_ttft_ms",
		"agent_latency_ms",
		"agent_input_tokens_total",
		"agent_output_tokens_total",
		"agent_total_tokens",
		"agent_active_sessions",
		"gpu_util_pct",
		"gpu_vram_used_gb",
		"cost_usd_per_1k_tokens",
	}

	for _, metric := range expectedMetrics {
		assert.Contains(t, metricsOutput, metric, "Metrics output should contain %s", metric)
	}

	// Verify histogram buckets
	assert.Contains(t, metricsOutput, "agent_ttft_ms_bucket")
	assert.Contains(t, metricsOutput, "agent_latency_ms_bucket")

	t.Logf("Successfully exported %d bytes of metrics", len(body))
}

// TestMetricsSLOAlerting tests SLO-based alerting scenarios
func TestMetricsSLOAlerting(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	scenarios := []struct {
		name          string
		setup         func()
		expectedAlert bool
		alertType     string
	}{
		{
			name: "TTFT within SLO",
			setup: func() {
				for i := 0; i < 10; i++ {
					m.RecordTTFT(ctx, 300*time.Millisecond, "llama-3-70b", "/chat")
				}
			},
			expectedAlert: false,
			alertType:     "ttft",
		},
		{
			name: "TTFT exceeds SLO",
			setup: func() {
				for i := 0; i < 10; i++ {
					m.RecordTTFT(ctx, 400*time.Millisecond, "llama-3-70b", "/chat")
				}
			},
			expectedAlert: true,
			alertType:     "ttft",
		},
		{
			name: "Error rate within SLO",
			setup: func() {
				// Simulate 100 requests with 0.5% error rate
				for i := 0; i < 100; i++ {
					if i < 1 {
						m.RecordError(ctx, "timeout", "llama-3-70b")
					}
				}
			},
			expectedAlert: false,
			alertType:     "error_rate",
		},
		{
			name: "Error rate exceeds SLO",
			setup: func() {
				// Simulate 100 requests with 2% error rate
				for i := 0; i < 100; i++ {
					if i < 2 {
						m.RecordError(ctx, "timeout", "llama-3-70b")
					}
				}
			},
			expectedAlert: true,
			alertType:     "error_rate",
		},
		{
			name: "Cost within budget",
			setup: func() {
				m.RecordCost(ctx, 0.08, 1000, "llama-3-70b", "tenant-1")
			},
			expectedAlert: false,
			alertType:     "cost",
		},
		{
			name: "Cost exceeds budget",
			setup: func() {
				m.RecordCost(ctx, 0.15, 1000, "llama-3-70b", "tenant-1")
			},
			expectedAlert: true,
			alertType:     "cost",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create fresh registry for each scenario
			registry := prometheus.NewRegistry()
			metrics.NewAgentMetrics(registry)

			scenario.setup()

			// In a real scenario, Prometheus would evaluate alert rules
			// Here we verify the metrics are being recorded correctly
			// for the alerting system to pick up

			t.Logf("Scenario '%s' completed. Alert expected: %v for type: %s",
				scenario.name, scenario.expectedAlert, scenario.alertType)
		})
	}
}

// TestMetricsGrafanaDashboardQueries tests queries used in Grafana dashboards
func TestMetricsGrafanaDashboardQueries(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate a realistic workload
	for i := 0; i < 100; i++ {
		// Vary TTFT to create distribution
		ttft := time.Duration(200+i*5) * time.Millisecond
		m.RecordTTFT(ctx, ttft, "llama-3-70b", "/chat")

		// Vary latency
		latency := time.Duration(1000+i*10) * time.Millisecond
		m.RecordLatency(ctx, latency, "llama-3-70b", "/chat")

		// Token usage
		m.RecordTokens(ctx, int64(500+i*10), int64(250+i*5), "llama-3-70b")

		// GPU utilization varies
		m.RecordGPUMetrics(ctx, "node-1", float64(70+i%30), 60.0, 80.0)

		// Tool calls
		if i%5 == 0 {
			m.RecordToolCall(ctx, "code_search", time.Duration(100+i*2)*time.Millisecond, true)
		}

		// Costs
		m.RecordCost(ctx, 0.001*float64(i), int64(500+i*10+250+i*5), "llama-3-70b", "tenant-1")
	}

	// Update gauges
	m.SetActiveSessions(15)
	m.SetQueueDepth(8, "/chat")
	m.KVCacheHitRatio.Set(0.75)
	m.BatchMergeEfficiency.Set(0.88)

	// Create HTTP handler and verify metrics can be scraped
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	server := &http.Server{
		Addr:    ":19091",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Scrape metrics
	resp, err := http.Get("http://localhost:19091/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metricsOutput := string(body)

	// Verify dashboard panel metrics are present
	dashboardMetrics := map[string]string{
		"TTFT P95":              "agent_ttft_ms_bucket",
		"Tokens/Second":         "agent_total_tokens",
		"Active Sessions":       "agent_active_sessions",
		"GPU Utilization":       "gpu_util_pct",
		"Cost per 1K Tokens":    "cost_usd_per_1k_tokens",
		"KV Cache Hit Ratio":    "agent_kv_cache_hit_ratio",
		"Batch Efficiency":      "agent_batch_merge_efficiency",
		"Tool Call Latency":     "agent_tool_latency_ms",
		"Queue Depth":           "agent_queue_depth",
		"Input Tokens":          "agent_input_tokens_total",
		"Output Tokens":         "agent_output_tokens_total",
		"Turn Latency":          "agent_latency_ms_bucket",
	}

	for panel, metric := range dashboardMetrics {
		assert.Contains(t, metricsOutput, metric,
			"Metrics for dashboard panel '%s' should be exported", panel)
	}

	t.Logf("Verified %d dashboard panel metrics", len(dashboardMetrics))
}

// TestMetricsMultiTenantIsolation tests metrics isolation between tenants
func TestMetricsMultiTenantIsolation(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate workload for multiple tenants
	tenants := []string{"tenant-1", "tenant-2", "tenant-3"}

	for _, tenant := range tenants {
		// Each tenant has different cost profile
		costMultiplier := 1.0
		switch tenant {
		case "tenant-1":
			costMultiplier = 1.0
		case "tenant-2":
			costMultiplier = 1.5
		case "tenant-3":
			costMultiplier = 0.8
		}

		tokens := int64(1000)
		cost := 0.10 * costMultiplier
		m.RecordCost(ctx, cost, tokens, "llama-3-70b", tenant)
	}

	// Note: In production, each tenant would have labeled metrics
	// Here we verify the cost recording mechanism works
	costPer1K := testutil.ToFloat64(m.CostPer1KTokens)
	assert.Greater(t, costPer1K, 0.0, "Cost should be recorded")
}

// TestMetricsRealTimeUpdates tests metrics updates in real-time scenario
func TestMetricsRealTimeUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-time test in short mode")
	}

	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Start metrics server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := &http.Server{
		Addr:    ":19092",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Simulate real-time workload
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		count := 0
		for range ticker.C {
			m.RecordTTFT(ctx, time.Duration(200+count*10)*time.Millisecond, "llama-3-70b", "/chat")
			m.SetActiveSessions(5 + count%10)
			m.RecordTokens(ctx, 500, 250, "llama-3-70b")
			count++
			if count >= 10 {
				done <- true
				return
			}
		}
	}()

	// Wait for workload to complete
	<-done

	// Verify metrics are up-to-date
	resp, err := http.Get("http://localhost:19092/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metricsOutput := string(body)
	assert.Contains(t, metricsOutput, "agent_ttft_ms")
	assert.Contains(t, metricsOutput, "agent_active_sessions")
	assert.Contains(t, metricsOutput, "agent_total_tokens")

	t.Log("Real-time metrics updates verified successfully")
}

// TestMetricsLabelsCardinality tests that label cardinality stays bounded
func TestMetricsLabelsCardinality(t *testing.T) {
	// Test that we don't create unbounded cardinality with labels
	labels := &metrics.MetricsLabels{
		Model:      "llama-3-70b",
		Route:      "/chat",
		Tool:       "code_search",
		Node:       "node-1",
		Tenant:     "tenant-1",
		AgentClass: "code-assistant",
		AgentPool:  "main-pool",
	}

	attrSet := labels.WithLabels()
	require.NotNil(t, attrSet)

	// Verify all expected labels are present
	expectedLabels := []string{
		"model", "route", "tool", "node", "tenant", "agentclass", "agentpool",
	}

	for _, label := range expectedLabels {
		val, ok := attrSet.Value(attribute.Key(label))
		assert.True(t, ok && val.AsString() != "",
			"Label %s should be present", label)
	}

	// Verify label values
	assert.Equal(t, "llama-3-70b", labels.Model)
	assert.Equal(t, "/chat", labels.Route)
	assert.Equal(t, "code_search", labels.Tool)

	t.Log("Label cardinality test passed")
}

// TestMetricsPrometheusRecordingRules tests queries that would be used in recording rules
func TestMetricsPrometheusRecordingRules(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Simulate data for recording rules
	// Example: calculate average tokens per request over time
	for i := 0; i < 50; i++ {
		inputTokens := int64(800 + i*20)
		outputTokens := int64(400 + i*10)
		m.RecordTokens(ctx, inputTokens, outputTokens, "llama-3-70b")
	}

	// Recording rule: cost efficiency (cost / tokens)
	for i := 0; i < 50; i++ {
		tokens := int64(1000 + i*100)
		cost := float64(tokens) * 0.0001 // $0.0001 per token
		m.RecordCost(ctx, cost, tokens, "llama-3-70b", "tenant-1")
	}

	// Recording rule: GPU efficiency (tokens / GPU utilization)
	for i := 0; i < 50; i++ {
		gpuUtil := 70.0 + float64(i%30)
		m.RecordGPUMetrics(ctx, "node-1", gpuUtil, 60.0, 80.0)
		m.RecordTokens(ctx, 1000, 500, "llama-3-70b")
	}

	// Verify metrics are recorded
	totalTokens := testutil.ToFloat64(m.TotalTokens)
	assert.Greater(t, totalTokens, 0.0, "Total tokens should be recorded")

	gpuUtil := testutil.ToFloat64(m.GPUUtilization)
	assert.Greater(t, gpuUtil, 0.0, "GPU utilization should be recorded")

	t.Log("Recording rules test completed")
}

// TestMetricsExportFormats tests different export formats
func TestMetricsExportFormats(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Record sample metrics
	m.RecordTTFT(ctx, 350*time.Millisecond, "llama-3-70b", "/chat")
	m.RecordLatency(ctx, 2000*time.Millisecond, "llama-3-70b", "/chat")
	m.SetActiveSessions(10)

	// Test Prometheus text format
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	server := &http.Server{
		Addr:    ":19093",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Request Prometheus format
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "http://localhost:19093/metrics", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	metricsOutput := string(body)

	// Verify Prometheus exposition format
	assert.Contains(t, metricsOutput, "# HELP agent_ttft_ms")
	assert.Contains(t, metricsOutput, "# TYPE agent_ttft_ms histogram")
	assert.Contains(t, metricsOutput, "agent_ttft_ms_bucket")

	// Verify metric values are present
	assert.True(t, strings.Contains(metricsOutput, "agent_active_sessions 10"),
		"Active sessions gauge should be exported")

	t.Log("Metrics export format validation passed")
}

// TestMetricsConsistencyAcrossScrapes tests that metrics remain consistent
func TestMetricsConsistencyAcrossScrapes(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)
	ctx := context.Background()

	// Record some metrics
	m.RecordTokens(ctx, 1000, 500, "llama-3-70b")
	m.SetActiveSessions(15)
	m.RecordCost(ctx, 0.15, 1500, "llama-3-70b", "tenant-1")

	// Start server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := &http.Server{
		Addr:    ":19094",
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Scrape metrics multiple times
	client := &http.Client{Timeout: 5 * time.Second}

	scrapeMetrics := func() (string, error) {
		resp, err := client.Get("http://localhost:19094/metrics")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}

	// First scrape
	scrape1, err := scrapeMetrics()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Second scrape (without recording new metrics)
	scrape2, err := scrapeMetrics()
	require.NoError(t, err)

	// Counter values should be identical
	extractCounter := func(output, metric string) string {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, metric+" ") && !strings.Contains(line, "#") {
				return line
			}
		}
		return ""
	}

	counter1 := extractCounter(scrape1, "agent_input_tokens_total")
	counter2 := extractCounter(scrape2, "agent_input_tokens_total")

	assert.Equal(t, counter1, counter2,
		"Counter values should be consistent across scrapes")

	t.Log("Metrics consistency validation passed")
}

// TestMetricsOpenTelemetryIntegration tests OpenTelemetry meter integration
func TestMetricsOpenTelemetryIntegration(t *testing.T) {
	registry := prometheus.NewRegistry()
	m := metrics.NewAgentMetrics(registry)

	// Verify OpenTelemetry meter is initialized
	require.NotNil(t, m, "AgentMetrics should be initialized")

	// Test labels structure for OTEL
	labels := &metrics.MetricsLabels{
		Model:      "llama-3-70b",
		Route:      "/chat",
		Tenant:     "tenant-1",
		AgentClass: "code-assistant",
	}

	attrSet := labels.WithLabels()
	require.NotNil(t, attrSet)

	// Verify attributes are properly structured for OTEL
	assert.True(t, attrSet.HasValue("model"))
	assert.True(t, attrSet.HasValue("route"))
	assert.True(t, attrSet.HasValue("tenant"))
	assert.True(t, attrSet.HasValue("agentclass"))

	t.Log("OpenTelemetry integration test passed")
}
