package autoscaler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestNewTokenAwareAutoscaler(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}

	scaler := NewTokenAwareAutoscaler(provider, config)
	
	assert.NotNil(t, scaler)
	assert.Equal(t, provider, scaler.provider)
	assert.Equal(t, config, scaler.config)
}

func TestEvaluate_BasicScaling(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)
	ctx := context.Background()

	tests := []struct {
		name            string
		currentReplicas int32
		metricValue     float64
		metricTarget    string
		minReplicas     int32
		maxReplicas     int32
		expectedMin     int32
		expectedMax     int32
	}{
		{
			name:            "scale up - high load",
			currentReplicas: 5,
			metricValue:     300,
			metricTarget:    "100",
			minReplicas:     2,
			maxReplicas:     20,
			expectedMin:     10, // 5 * (300/100) = 15, but checking it scales up
			expectedMax:     20,
		},
		{
			name:            "scale down - low load",
			currentReplicas: 10,
			metricValue:     50,
			metricTarget:    "100",
			minReplicas:     2,
			maxReplicas:     20,
			expectedMin:     2,
			expectedMax:     8, // 10 * (50/100) = 5
		},
		{
			name:            "at min replicas",
			currentReplicas: 2,
			metricValue:     10,
			metricTarget:    "100",
			minReplicas:     2,
			maxReplicas:     20,
			expectedMin:     2,
			expectedMax:     2,
		},
		{
			name:            "at max replicas",
			currentReplicas: 20,
			metricValue:     500,
			metricTarget:    "100",
			minReplicas:     2,
			maxReplicas:     20,
			expectedMin:     20,
			expectedMax:     20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &neuronetes.AgentPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pool",
					Namespace: "default",
				},
				Spec: neuronetes.AgentPoolSpec{
					MinReplicas: tt.minReplicas,
					MaxReplicas: tt.maxReplicas,
					Autoscaling: &neuronetes.AutoscalingSpec{
						Metrics: []neuronetes.AutoscalingMetric{
							{
								Type:   "tokens-in-queue",
								Target: tt.metricTarget,
							},
						},
					},
				},
				Status: neuronetes.AgentPoolStatus{
					Replicas: tt.currentReplicas,
				},
			}

			provider.SetMetric("tokens-in-queue", tt.metricValue)

			decision, err := scaler.Evaluate(ctx, pool)
			require.NoError(t, err)
			
			assert.GreaterOrEqual(t, decision.DesiredReplicas, tt.expectedMin)
			assert.LessOrEqual(t, decision.DesiredReplicas, tt.expectedMax)
			assert.GreaterOrEqual(t, decision.DesiredReplicas, tt.minReplicas)
			assert.LessOrEqual(t, decision.DesiredReplicas, tt.maxReplicas)
		})
	}
}

func TestEvaluate_NoAutoscalingSpec(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)
	ctx := context.Background()

	pool := &neuronetes.AgentPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pool",
			Namespace: "default",
		},
		Spec: neuronetes.AgentPoolSpec{
			MinReplicas: 2,
			MaxReplicas: 10,
			// No Autoscaling spec
		},
		Status: neuronetes.AgentPoolStatus{
			Replicas: 5,
		},
	}

	decision, err := scaler.Evaluate(ctx, pool)
	require.NoError(t, err)
	assert.Equal(t, int32(5), decision.DesiredReplicas, "Should maintain current replicas")
}

func TestEvaluate_MultipleMetrics(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)
	ctx := context.Background()

	pool := &neuronetes.AgentPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pool",
			Namespace: "default",
		},
		Spec: neuronetes.AgentPoolSpec{
			MinReplicas: 2,
			MaxReplicas: 20,
			Autoscaling: &neuronetes.AutoscalingSpec{
				Metrics: []neuronetes.AutoscalingMetric{
					{
						Type:   "tokens-in-queue",
						Target: "100",
					},
					{
						Type:   "ttft-p95",
						Target: "500ms",
					},
				},
			},
		},
		Status: neuronetes.AgentPoolStatus{
			Replicas: 5,
		},
	}

	// Set one metric high, one low - should scale based on the highest ratio
	provider.SetMetric("tokens-in-queue", 300) // ratio: 3.0
	provider.SetMetric("ttft-p95", 400)        // ratio: 0.8 (400/500)

	decision, err := scaler.Evaluate(ctx, pool)
	require.NoError(t, err)
	
	// Should scale based on the highest ratio (tokens-in-queue)
	assert.Greater(t, decision.DesiredReplicas, int32(5))
	assert.Equal(t, "tokens-in-queue", decision.Reason)
}

func TestApplyScalingPolicies_MaxChangePercent(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)

	maxPercent := int32(50) // 50%
	pool := &neuronetes.AgentPool{
		Spec: neuronetes.AgentPoolSpec{
			Autoscaling: &neuronetes.AutoscalingSpec{
				Behavior: &neuronetes.ScalingBehavior{
					ScaleUp: &neuronetes.ScalingPolicy{
						MaxChangePercent: &maxPercent,
					},
				},
			},
		},
	}

	currentReplicas := int32(10)
	desiredReplicas := int32(20) // Would scale up by 100%

	result := scaler.applyScalingPolicies(pool, currentReplicas, desiredReplicas)
	
	// Should be limited to 50% increase: 10 + (10 * 0.5) = 15
	assert.Equal(t, int32(15), result)
}

func TestApplyScalingPolicies_MaxChangeAbsolute(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)

	maxChange := int32(3)
	pool := &neuronetes.AgentPool{
		Spec: neuronetes.AgentPoolSpec{
			Autoscaling: &neuronetes.AutoscalingSpec{
				Behavior: &neuronetes.ScalingBehavior{
					ScaleUp: &neuronetes.ScalingPolicy{
						MaxChangeAbsolute: &maxChange,
					},
				},
			},
		},
	}

	currentReplicas := int32(5)
	desiredReplicas := int32(15) // Would scale up by 10

	result := scaler.applyScalingPolicies(pool, currentReplicas, desiredReplicas)
	
	// Should be limited to absolute increase of 3: 5 + 3 = 8
	assert.Equal(t, int32(8), result)
}

func TestApplyScalingPolicies_ScaleDown(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)

	maxPercent := int32(30) // 30%
	pool := &neuronetes.AgentPool{
		Spec: neuronetes.AgentPoolSpec{
			Autoscaling: &neuronetes.AutoscalingSpec{
				Behavior: &neuronetes.ScalingBehavior{
					ScaleDown: &neuronetes.ScalingPolicy{
						MaxChangePercent: &maxPercent,
					},
				},
			},
		},
	}

	currentReplicas := int32(10)
	desiredReplicas := int32(2) // Would scale down by 80%

	result := scaler.applyScalingPolicies(pool, currentReplicas, desiredReplicas)
	
	// Should be limited to 30% decrease: 10 - (10 * 0.3) = 7
	assert.Equal(t, int32(7), result)
}

func TestApplyScalingPolicies_NoPolicy(t *testing.T) {
	provider := NewMockMetricsProvider()
	config := &AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := NewTokenAwareAutoscaler(provider, config)

	pool := &neuronetes.AgentPool{
		Spec: neuronetes.AgentPoolSpec{
			// No scaling behavior policies
		},
	}

	currentReplicas := int32(5)
	desiredReplicas := int32(15)

	result := scaler.applyScalingPolicies(pool, currentReplicas, desiredReplicas)
	
	// Should return desired replicas unchanged
	assert.Equal(t, desiredReplicas, result)
}

func TestParseMetricTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected float64
		hasError bool
	}{
		{
			name:     "integer",
			target:   "100",
			expected: 100.0,
			hasError: false,
		},
		{
			name:     "float",
			target:   "99.5",
			expected: 99.5,
			hasError: false,
		},
		{
			name:     "with milliseconds",
			target:   "500ms",
			expected: 500.0,
			hasError: false,
		},
		{
			name:     "invalid",
			target:   "invalid",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty",
			target:   "",
			expected: 0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMetricTarget(tt.target)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.01)
			}
		})
	}
}

func TestMockMetricsProvider(t *testing.T) {
	provider := NewMockMetricsProvider()

	t.Run("set and get metric", func(t *testing.T) {
		provider.SetMetric("test-metric", 123.45)
		
		value, err := provider.GetMetric(context.Background(), "test-metric", "default", "test-pool")
		require.NoError(t, err)
		assert.Equal(t, 123.45, value)
	})

	t.Run("get non-existent metric", func(t *testing.T) {
		value, err := provider.GetMetric(context.Background(), "non-existent", "default", "test-pool")
		require.NoError(t, err)
		assert.Equal(t, 0.0, value)
	})

	t.Run("update metric", func(t *testing.T) {
		provider.SetMetric("update-test", 100.0)
		provider.SetMetric("update-test", 200.0)
		
		value, err := provider.GetMetric(context.Background(), "update-test", "default", "test-pool")
		require.NoError(t, err)
		assert.Equal(t, 200.0, value)
	})
}

func TestScalingDecision(t *testing.T) {
	decision := &ScalingDecision{
		CurrentReplicas: 5,
		DesiredReplicas: 10,
		Reason:          "high_load",
		Timestamp:       time.Now(),
	}

	assert.Equal(t, int32(5), decision.CurrentReplicas)
	assert.Equal(t, int32(10), decision.DesiredReplicas)
	assert.Equal(t, "high_load", decision.Reason)
	assert.False(t, decision.Timestamp.IsZero())
}
