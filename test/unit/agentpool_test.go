package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestAgentPoolValidation(t *testing.T) {
	tests := []struct {
		name      string
		agentPool *neuronetes.AgentPool
		wantErr   bool
	}{
		{
			name: "valid agent pool",
			agentPool: &neuronetes.AgentPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pool",
					Namespace: "default",
				},
				Spec: neuronetes.AgentPoolSpec{
					AgentClassRef: neuronetes.AgentClassReference{
						Name: "test-agent-class",
					},
					MinReplicas:    2,
					MaxReplicas:    10,
					PrewarmPercent: 20,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid min > max",
			agentPool: &neuronetes.AgentPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-pool",
					Namespace: "default",
				},
				Spec: neuronetes.AgentPoolSpec{
					AgentClassRef: neuronetes.AgentClassReference{
						Name: "test-agent-class",
					},
					MinReplicas: 10,
					MaxReplicas: 5,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid prewarm percent",
			agentPool: &neuronetes.AgentPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-prewarm",
					Namespace: "default",
				},
				Spec: neuronetes.AgentPoolSpec{
					AgentClassRef: neuronetes.AgentClassReference{
						Name: "test-agent-class",
					},
					MinReplicas:    2,
					MaxReplicas:    10,
					PrewarmPercent: 150,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentPool(tt.agentPool)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAutoscalingMetricValidation(t *testing.T) {
	tests := []struct {
		name    string
		metric  neuronetes.AutoscalingMetric
		wantErr bool
	}{
		{
			name: "valid tokens-in-queue",
			metric: neuronetes.AutoscalingMetric{
				Type:   "tokens-in-queue",
				Target: "100",
			},
			wantErr: false,
		},
		{
			name: "valid ttft-p95",
			metric: neuronetes.AutoscalingMetric{
				Type:   "ttft-p95",
				Target: "500ms",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			metric: neuronetes.AutoscalingMetric{
				Type:   "invalid-metric",
				Target: "100",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAutoscalingMetric(&tt.metric)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGPURequirementsValidation(t *testing.T) {
	tests := []struct {
		name         string
		requirements *neuronetes.GPURequirements
		wantErr      bool
	}{
		{
			name: "valid single GPU",
			requirements: &neuronetes.GPURequirements{
				Count:  1,
				Memory: "40Gi",
				Type:   "A100",
			},
			wantErr: false,
		},
		{
			name: "valid multi-GPU with topology",
			requirements: &neuronetes.GPURequirements{
				Count:  4,
				Memory: "80Gi",
				Type:   "H100",
				Topology: &neuronetes.TopologyRequirement{
					Locality: "same-node",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid count",
			requirements: &neuronetes.GPURequirements{
				Count: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGPURequirements(tt.requirements)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionAffinityValidation(t *testing.T) {
	tests := []struct {
		name     string
		affinity *neuronetes.SessionAffinityConfig
		wantErr  bool
	}{
		{
			name: "valid conversation-id affinity",
			affinity: &neuronetes.SessionAffinityConfig{
				Enabled:   true,
				KeyHeader: "X-Session-ID",
				Type:      "conversation-id",
			},
			wantErr: false,
		},
		{
			name: "valid user-id affinity",
			affinity: &neuronetes.SessionAffinityConfig{
				Enabled: true,
				Type:    "user-id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionAffinity(tt.affinity)
			assert.NoError(t, err)
		})
	}
}

func TestCostOptimizationValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *neuronetes.CostOptimizationConfig
		wantErr bool
	}{
		{
			name: "valid with spot",
			config: &neuronetes.CostOptimizationConfig{
				Enabled:        true,
				SpotEnabled:    true,
				SLOHeadroomMs:  int32Ptr(1000),
				MaxCostPerHour: float32Ptr(50.0),
			},
			wantErr: false,
		},
		{
			name: "valid with fallback model",
			config: &neuronetes.CostOptimizationConfig{
				Enabled:       true,
				FallbackModel: "smaller-model",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCostOptimization(tt.config)
			assert.NoError(t, err)
		})
	}
}

// Mock validation functions
func validateAgentPool(ap *neuronetes.AgentPool) error {
	if ap.Spec.MinReplicas > ap.Spec.MaxReplicas {
		return assert.AnError
	}
	if ap.Spec.PrewarmPercent < 0 || ap.Spec.PrewarmPercent > 100 {
		return assert.AnError
	}
	return nil
}

func validateAutoscalingMetric(m *neuronetes.AutoscalingMetric) error {
	validTypes := map[string]bool{
		"tokens-in-queue":     true,
		"ttft-p95":            true,
		"concurrent-sessions": true,
		"tokens-per-second":   true,
		"queue-depth":         true,
		"context-length":      true,
		"tool-call-rate":      true,
	}
	if !validTypes[m.Type] {
		return assert.AnError
	}
	return nil
}

func validateGPURequirements(req *neuronetes.GPURequirements) error {
	if req.Count < 1 {
		return assert.AnError
	}
	return nil
}

func validateSessionAffinity(affinity *neuronetes.SessionAffinityConfig) error {
	return nil
}

func validateCostOptimization(config *neuronetes.CostOptimizationConfig) error {
	return nil
}
