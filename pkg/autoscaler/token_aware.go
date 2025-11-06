package autoscaler

import (
	"context"
	"fmt"
	"time"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

// TokenAwareAutoscaler implements token-based autoscaling
type TokenAwareAutoscaler struct {
	metricsProvider MetricsProvider
	config          *AutoscalerConfig
}

// AutoscalerConfig defines autoscaler configuration
type AutoscalerConfig struct {
	// Metrics collection interval
	MetricsInterval time.Duration

	// Decision interval
	DecisionInterval time.Duration

	// Stabilization window
	StabilizationWindow time.Duration
}

// MetricsProvider interface for fetching metrics
type MetricsProvider interface {
	GetMetric(ctx context.Context, pool *neuronetes.AgentPool, metricType string) (float64, error)
}

// NewTokenAwareAutoscaler creates a new autoscaler
func NewTokenAwareAutoscaler(provider MetricsProvider, config *AutoscalerConfig) *TokenAwareAutoscaler {
	return &TokenAwareAutoscaler{
		metricsProvider: provider,
		config:          config,
	}
}

// ScalingDecision represents an autoscaling decision
type ScalingDecision struct {
	CurrentReplicas int32
	DesiredReplicas int32
	Reason          string
	Metrics         map[string]float64
}

// Evaluate calculates desired replicas for an AgentPool
func (a *TokenAwareAutoscaler) Evaluate(ctx context.Context, pool *neuronetes.AgentPool) (*ScalingDecision, error) {
	if pool.Spec.Autoscaling == nil || len(pool.Spec.Autoscaling.Metrics) == 0 {
		return &ScalingDecision{
			CurrentReplicas: pool.Status.Replicas,
			DesiredReplicas: pool.Status.Replicas,
			Reason:          "no autoscaling configured",
		}, nil
	}

	// Collect metrics
	metrics := make(map[string]float64)
	var maxRatio float64
	var primaryMetric string

	for _, metric := range pool.Spec.Autoscaling.Metrics {
		value, err := a.metricsProvider.GetMetric(ctx, pool, metric.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric %s: %w", metric.Type, err)
		}

		metrics[metric.Type] = value

		// Parse target
		target, err := parseMetricTarget(metric.Target)
		if err != nil {
			return nil, fmt.Errorf("invalid target for %s: %w", metric.Type, err)
		}

		// Calculate ratio
		ratio := value / target
		if ratio > maxRatio {
			maxRatio = ratio
			primaryMetric = metric.Type
		}
	}

	// Calculate desired replicas
	currentReplicas := pool.Status.Replicas
	desiredReplicas := int32(float64(currentReplicas) * maxRatio)

	// Apply min/max bounds
	if desiredReplicas < pool.Spec.MinReplicas {
		desiredReplicas = pool.Spec.MinReplicas
	}
	if desiredReplicas > pool.Spec.MaxReplicas {
		desiredReplicas = pool.Spec.MaxReplicas
	}

	// Apply scaling policies
	desiredReplicas = a.applyScalingPolicies(pool, currentReplicas, desiredReplicas)

	reason := fmt.Sprintf("scaled based on %s (ratio: %.2f)", primaryMetric, maxRatio)

	return &ScalingDecision{
		CurrentReplicas: currentReplicas,
		DesiredReplicas: desiredReplicas,
		Reason:          reason,
		Metrics:         metrics,
	}, nil
}

func (a *TokenAwareAutoscaler) applyScalingPolicies(pool *neuronetes.AgentPool, current, desired int32) int32 {
	if pool.Spec.Autoscaling.Behavior == nil {
		return desired
	}

	behavior := pool.Spec.Autoscaling.Behavior

	// Scale up
	if desired > current {
		if behavior.ScaleUp != nil {
			// Apply max change limits
			if behavior.ScaleUp.MaxChangeAbsolute != nil {
				maxIncrease := current + *behavior.ScaleUp.MaxChangeAbsolute
				if desired > maxIncrease {
					desired = maxIncrease
				}
			}

			if behavior.ScaleUp.MaxChangePercent != nil {
				maxIncrease := int32(float64(current) * (1.0 + float64(*behavior.ScaleUp.MaxChangePercent)/100.0))
				if desired > maxIncrease {
					desired = maxIncrease
				}
			}
		}
	}

	// Scale down
	if desired < current {
		if behavior.ScaleDown != nil {
			// Apply max change limits
			if behavior.ScaleDown.MaxChangeAbsolute != nil {
				maxDecrease := current - *behavior.ScaleDown.MaxChangeAbsolute
				if desired < maxDecrease {
					desired = maxDecrease
				}
			}

			if behavior.ScaleDown.MaxChangePercent != nil {
				maxDecrease := int32(float64(current) * (1.0 - float64(*behavior.ScaleDown.MaxChangePercent)/100.0))
				if desired < maxDecrease {
					desired = maxDecrease
				}
			}
		}
	}

	return desired
}

func parseMetricTarget(target string) (float64, error) {
	// Simple parser - in production, handle units properly
	var value float64
	_, err := fmt.Sscanf(target, "%f", &value)
	return value, err
}

// MockMetricsProvider for testing
type MockMetricsProvider struct {
	metrics map[string]float64
}

func NewMockMetricsProvider() *MockMetricsProvider {
	return &MockMetricsProvider{
		metrics: make(map[string]float64),
	}
}

func (m *MockMetricsProvider) SetMetric(metricType string, value float64) {
	m.metrics[metricType] = value
}

func (m *MockMetricsProvider) GetMetric(ctx context.Context, pool *neuronetes.AgentPool, metricType string) (float64, error) {
	value, ok := m.metrics[metricType]
	if !ok {
		return 0, fmt.Errorf("metric %s not found", metricType)
	}
	return value, nil
}
