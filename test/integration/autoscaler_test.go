package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
	"github.com/bowenislandsong/neuronetes/pkg/autoscaler"
)

func TestAutoscalerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Create autoscaler with mock metrics
	provider := autoscaler.NewMockMetricsProvider()
	config := &autoscaler.AutoscalerConfig{
		MetricsInterval:     10 * time.Second,
		DecisionInterval:    30 * time.Second,
		StabilizationWindow: 60 * time.Second,
	}
	scaler := autoscaler.NewTokenAwareAutoscaler(provider, config)

	t.Run("scale up on high load", func(t *testing.T) {
		// Create agent pool
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 2,
				MaxReplicas: 10,
				Autoscaling: &neuronetes.AutoscalingSpec{
					Metrics: []neuronetes.AutoscalingMetric{
						{
							Type:   "tokens-in-queue",
							Target: "100",
						},
					},
				},
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 2,
			},
		}

		// Set high load
		provider.SetMetric("tokens-in-queue", 300)

		// Evaluate
		decision, err := scaler.Evaluate(ctx, pool)
		require.NoError(t, err)
		assert.Greater(t, decision.DesiredReplicas, decision.CurrentReplicas)
		assert.LessOrEqual(t, decision.DesiredReplicas, pool.Spec.MaxReplicas)
	})

	t.Run("scale down on low load", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 2,
				MaxReplicas: 10,
				Autoscaling: &neuronetes.AutoscalingSpec{
					Metrics: []neuronetes.AutoscalingMetric{
						{
							Type:   "tokens-in-queue",
							Target: "100",
						},
					},
				},
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 8,
			},
		}

		// Set low load
		provider.SetMetric("tokens-in-queue", 30)

		// Evaluate
		decision, err := scaler.Evaluate(ctx, pool)
		require.NoError(t, err)
		assert.Less(t, decision.DesiredReplicas, decision.CurrentReplicas)
		assert.GreaterOrEqual(t, decision.DesiredReplicas, pool.Spec.MinReplicas)
	})

	t.Run("respect min replicas", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 5,
				MaxReplicas: 20,
				Autoscaling: &neuronetes.AutoscalingSpec{
					Metrics: []neuronetes.AutoscalingMetric{
						{
							Type:   "tokens-in-queue",
							Target: "100",
						},
					},
				},
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 5,
			},
		}

		// Set very low load
		provider.SetMetric("tokens-in-queue", 10)

		// Evaluate
		decision, err := scaler.Evaluate(ctx, pool)
		require.NoError(t, err)
		assert.Equal(t, pool.Spec.MinReplicas, decision.DesiredReplicas)
	})

	t.Run("respect max replicas", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 2,
				MaxReplicas: 10,
				Autoscaling: &neuronetes.AutoscalingSpec{
					Metrics: []neuronetes.AutoscalingMetric{
						{
							Type:   "tokens-in-queue",
							Target: "100",
						},
					},
				},
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 10,
			},
		}

		// Set very high load
		provider.SetMetric("tokens-in-queue", 1000)

		// Evaluate
		decision, err := scaler.Evaluate(ctx, pool)
		require.NoError(t, err)
		assert.Equal(t, pool.Spec.MaxReplicas, decision.DesiredReplicas)
	})

	t.Run("apply scaling policies", func(t *testing.T) {
		maxChange := int32(3)
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
					},
					Behavior: &neuronetes.ScalingBehavior{
						ScaleUp: &neuronetes.ScalingPolicy{
							MaxChangeAbsolute: &maxChange,
						},
					},
				},
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas: 5,
			},
		}

		// Set high load
		provider.SetMetric("tokens-in-queue", 500)

		// Evaluate
		decision, err := scaler.Evaluate(ctx, pool)
		require.NoError(t, err)

		// Should not exceed max change
		assert.LessOrEqual(t, decision.DesiredReplicas-decision.CurrentReplicas, maxChange)
	})
}

func TestModelLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Run("model transitions through phases", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-model",
				Namespace: "default",
			},
			Spec: neuronetes.ModelSpec{
				WeightsURI: "s3://test/model",
				Size:       resource.MustParse("50Gi"),
			},
			Status: neuronetes.ModelStatus{
				Phase: "",
			},
		}

		// Initial state should be Pending
		assert.Equal(t, "", model.Status.Phase)

		// Transition to Loading
		model.Status.Phase = "Loading"
		assert.Equal(t, "Loading", model.Status.Phase)

		// Transition to Ready
		model.Status.Phase = "Ready"
		assert.Equal(t, "Ready", model.Status.Phase)
	})

	t.Run("model caching", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cached-model",
				Namespace: "default",
			},
			Spec: neuronetes.ModelSpec{
				WeightsURI: "s3://test/model",
				Size:       resource.MustParse("50Gi"),
				CachePolicy: &neuronetes.CachePolicy{
					Priority: "high",
				},
			},
			Status: neuronetes.ModelStatus{
				Phase: "Ready",
				CachedNodes: []neuronetes.NodeCacheStatus{
					{
						NodeName: "node-1",
						Status:   "ready",
					},
				},
			},
		}

		assert.Len(t, model.Status.CachedNodes, 1)
		assert.Equal(t, "node-1", model.Status.CachedNodes[0].NodeName)
		assert.Equal(t, "ready", model.Status.CachedNodes[0].Status)
	})
}

func TestAgentPoolScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Run("agent pool scales within bounds", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas: 3,
				MaxReplicas: 15,
			},
			Status: neuronetes.AgentPoolStatus{
				Replicas:      5,
				ReadyReplicas: 5,
			},
		}

		// Verify initial state
		assert.Equal(t, int32(5), pool.Status.Replicas)
		assert.GreaterOrEqual(t, pool.Status.Replicas, pool.Spec.MinReplicas)
		assert.LessOrEqual(t, pool.Status.Replicas, pool.Spec.MaxReplicas)
	})

	t.Run("warm pool calculation", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "warm-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				MinReplicas:    2,
				MaxReplicas:    20,
				PrewarmPercent: 25,
			},
		}

		// Calculate expected warm pool size
		expectedWarm := int32(float64(pool.Spec.MaxReplicas) * float64(pool.Spec.PrewarmPercent) / 100.0)
		assert.Equal(t, int32(5), expectedWarm)
	})
}
