package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestPluginRegistry(t *testing.T) {
	// Create a fresh registry for testing
	registry := &PluginRegistry{
		schedulers:       make(map[string]SchedulerPlugin),
		autoscalers:      make(map[string]AutoscalerPlugin),
		modelLoaders:     make(map[string]ModelLoaderPlugin),
		metricsProviders: make(map[string]MetricsProviderPlugin),
		guardrails:       make(map[string]GuardrailPlugin),
	}

	t.Run("register and get scheduler", func(t *testing.T) {
		plugin := NewExampleSchedulerPlugin()
		registry.RegisterScheduler(plugin)
		
		retrieved := registry.GetScheduler("example-scheduler")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "example-scheduler", retrieved.GetName())
	})

	t.Run("register and get autoscaler", func(t *testing.T) {
		plugin := NewExampleAutoscalerPlugin()
		registry.RegisterAutoscaler(plugin)
		
		retrieved := registry.GetAutoscaler("example-autoscaler")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "example-autoscaler", retrieved.GetName())
	})

	t.Run("register and get model loader", func(t *testing.T) {
		plugin := NewExampleModelLoaderPlugin()
		registry.RegisterModelLoader(plugin)
		
		retrieved := registry.GetModelLoader("example-loader")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "example-loader", retrieved.GetName())
	})

	t.Run("register and get guardrail", func(t *testing.T) {
		plugin := NewExampleGuardrailPlugin()
		registry.RegisterGuardrail(plugin)
		
		retrieved := registry.GetGuardrail("example-guardrail")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "example-guardrail", retrieved.GetName())
	})

	t.Run("get non-existent plugin", func(t *testing.T) {
		retrieved := registry.GetScheduler("non-existent")
		assert.Nil(t, retrieved)
	})

	t.Run("list all plugins", func(t *testing.T) {
		plugin1 := NewExampleSchedulerPlugin()
		plugin2 := NewExampleAutoscalerPlugin()
		
		registry.RegisterScheduler(plugin1)
		registry.RegisterAutoscaler(plugin2)
		
		schedulers := registry.ListSchedulers()
		assert.Contains(t, schedulers, "example-scheduler")
		
		autoscalers := registry.ListAutoscalers()
		assert.Contains(t, autoscalers, "example-autoscaler")
	})
}

func TestGlobalRegistry(t *testing.T) {
	t.Run("global register functions", func(t *testing.T) {
		// Note: These tests use the global registry, so they may affect each other
		scheduler := NewExampleSchedulerPlugin()
		RegisterScheduler(scheduler)
		
		retrieved := GetScheduler("example-scheduler")
		assert.NotNil(t, retrieved)
	})
}

func TestExampleSchedulerPlugin(t *testing.T) {
	plugin := NewExampleSchedulerPlugin()
	ctx := context.Background()

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, "example-scheduler", plugin.GetName())
	})

	t.Run("score nodes", func(t *testing.T) {
		nodes := []*corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
					Labels: map[string]string{
						"custom-score": "high",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-2",
					Labels: map[string]string{
						"custom-score": "low",
					},
				},
			},
		}
		pool := &neuronetes.AgentPool{}

		results, err := plugin.ScoreNodes(ctx, nodes, pool)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		
		// Verify all scores are in valid range
		for _, result := range results {
			assert.GreaterOrEqual(t, result.Score, int64(0))
			assert.LessOrEqual(t, result.Score, int64(100))
		}
	})
}

func TestExampleAutoscalerPlugin(t *testing.T) {
	plugin := NewExampleAutoscalerPlugin()
	ctx := context.Background()

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, "example-autoscaler", plugin.GetName())
	})

	t.Run("calculate desired replicas", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 2,
				MaxReplicas: 10,
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 5,
			},
		}

		replicas, reason, err := plugin.CalculateDesiredReplicas(ctx, pool, nil)
		require.NoError(t, err)
		
		// Should return a value within bounds
		assert.GreaterOrEqual(t, replicas, pool.Spec.MinReplicas)
		assert.LessOrEqual(t, replicas, pool.Spec.MaxReplicas)
		assert.NotEmpty(t, reason)
	})
}

func TestExampleModelLoaderPlugin(t *testing.T) {
	plugin := NewExampleModelLoaderPlugin()
	ctx := context.Background()

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, "example-loader", plugin.GetName())
	})

	t.Run("load model", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-model",
				Namespace: "default",
			},
			Spec: neuronetes.ModelSpec{
				WeightsURI: "s3://test-bucket/model",
			},
		}

		err := plugin.LoadModel(ctx, model, "node-1")
		assert.NoError(t, err)
	})

	t.Run("unload model", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-model",
				Namespace: "default",
			},
		}

		err := plugin.UnloadModel(ctx, model, "node-1")
		assert.NoError(t, err)
	})

	t.Run("get load status", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-model",
				Namespace: "default",
			},
		}

		status, err := plugin.GetLoadStatus(ctx, model, "node-1")
		require.NoError(t, err)
		
		// Status should be one of: "loading", "ready", "failed", "not-found"
		validStatuses := []string{"loading", "ready", "failed", "not-found"}
		assert.Contains(t, validStatuses, status)
	})
}

func TestExampleMetricsProviderPlugin(t *testing.T) {
	plugin := NewExampleMetricsProviderPlugin()
	ctx := context.Background()

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, "example-metrics", plugin.GetName())
	})

	t.Run("get metric", func(t *testing.T) {
		value, err := plugin.GetMetric(ctx, "test-metric", "default", "test-pool")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, value, 0.0)
	})
}

func TestExampleGuardrailPlugin(t *testing.T) {
	plugin := NewExampleGuardrailPlugin()
	ctx := context.Background()

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, "example-guardrail", plugin.GetName())
	})

	t.Run("get type", func(t *testing.T) {
		assert.Equal(t, "custom-keyword-filter", plugin.GetType())
	})

	t.Run("validate - clean content", func(t *testing.T) {
		result, err := plugin.Validate(ctx, "This is clean content", nil)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Empty(t, result.Reason)
	})

	t.Run("validate - blocked content", func(t *testing.T) {
		result, err := plugin.Validate(ctx, "This contains badword1 which is blocked", nil)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "blocked keyword")
	})

	t.Run("validate - another blocked keyword", func(t *testing.T) {
		result, err := plugin.Validate(ctx, "This has badword2 in it", nil)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Contains(t, result.Reason, "blocked keyword")
	})

	t.Run("validate - case sensitivity", func(t *testing.T) {
		// The implementation converts to lowercase, so uppercase should also be blocked
		result, err := plugin.Validate(ctx, "This has BADWORD1 in uppercase", nil)
		require.NoError(t, err)
		// Note: Depending on implementation, this might or might not be blocked
		// The example implementation converts to lowercase, so it should be blocked
		assert.False(t, result.Allowed)
	})
}

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at start",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "substring in middle",
			s:        "hello world",
			substr:   "lo wo",
			expected: true,
		},
		{
			name:     "not found",
			s:        "hello world",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "both empty",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
