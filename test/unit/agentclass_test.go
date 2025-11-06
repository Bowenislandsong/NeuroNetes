package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestAgentClassValidation(t *testing.T) {
	tests := []struct {
		name       string
		agentClass *neuronetes.AgentClass
		wantErr    bool
	}{
		{
			name: "valid agent class",
			agentClass: &neuronetes.AgentClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: "default",
				},
				Spec: neuronetes.AgentClassSpec{
					ModelRef: neuronetes.ModelReference{
						Name: "test-model",
					},
					MaxContextLength: 8192,
					ToolPermissions: []neuronetes.ToolPermission{
						{
							Name:      "search",
							RateLimit: "100/min",
						},
					},
					SLO: &neuronetes.ServiceLevelObjective{
						TokensPerSecond: int32Ptr(50),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid guardrail type",
			agentClass: &neuronetes.AgentClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-guardrail",
					Namespace: "default",
				},
				Spec: neuronetes.AgentClassSpec{
					ModelRef: neuronetes.ModelReference{
						Name: "test-model",
					},
					Guardrails: []neuronetes.Guardrail{
						{
							Type:   "invalid-type",
							Action: "block",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentClass(tt.agentClass)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToolPermissionValidation(t *testing.T) {
	tests := []struct {
		name       string
		permission neuronetes.ToolPermission
		wantErr    bool
	}{
		{
			name: "valid permission",
			permission: neuronetes.ToolPermission{
				Name:      "search",
				RateLimit: "100/min",
			},
			wantErr: false,
		},
		{
			name: "valid with timeout",
			permission: neuronetes.ToolPermission{
				Name:           "expensive-tool",
				RateLimit:      "10/min",
				Timeout:        &metav1.Duration{Duration: 30},
				MaxConcurrency: int32Ptr(5),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolPermission(&tt.permission)
			assert.NoError(t, err)
		})
	}
}

func TestGuardrailValidation(t *testing.T) {
	tests := []struct {
		name      string
		guardrail neuronetes.Guardrail
		wantErr   bool
	}{
		{
			name: "valid pii detection",
			guardrail: neuronetes.Guardrail{
				Type:      "pii-detection",
				Action:    "redact",
				Threshold: float32Ptr(0.8),
			},
			wantErr: false,
		},
		{
			name: "valid safety check",
			guardrail: neuronetes.Guardrail{
				Type:   "safety-check",
				Action: "block",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			guardrail: neuronetes.Guardrail{
				Type:   "invalid-type",
				Action: "block",
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			guardrail: neuronetes.Guardrail{
				Type:   "pii-detection",
				Action: "invalid-action",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGuardrail(&tt.guardrail)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSLOValidation(t *testing.T) {
	tests := []struct {
		name    string
		slo     *neuronetes.ServiceLevelObjective
		wantErr bool
	}{
		{
			name: "valid slo",
			slo: &neuronetes.ServiceLevelObjective{
				TTFT:                &metav1.Duration{Duration: 500},
				TokensPerSecond:     int32Ptr(50),
				P95Latency:          &metav1.Duration{Duration: 2000},
				AvailabilityPercent: float32Ptr(99.9),
			},
			wantErr: false,
		},
		{
			name: "valid minimal slo",
			slo: &neuronetes.ServiceLevelObjective{
				TTFT: &metav1.Duration{Duration: 1000},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSLO(tt.slo)
			assert.NoError(t, err)
		})
	}
}

// Mock validation functions
func validateAgentClass(ac *neuronetes.AgentClass) error {
	for _, g := range ac.Spec.Guardrails {
		if err := validateGuardrail(&g); err != nil {
			return err
		}
	}
	return nil
}

func validateToolPermission(tp *neuronetes.ToolPermission) error {
	return nil
}

func validateGuardrail(g *neuronetes.Guardrail) error {
	validTypes := map[string]bool{
		"pii-detection":       true,
		"safety-check":        true,
		"content-filter":      true,
		"jailbreak-detection": true,
		"prompt-injection":    true,
	}
	if !validTypes[g.Type] {
		return assert.AnError
	}

	validActions := map[string]bool{
		"block": true, "redact": true, "warn": true, "log": true,
	}
	if !validActions[g.Action] {
		return assert.AnError
	}

	return nil
}

func validateSLO(slo *neuronetes.ServiceLevelObjective) error {
	return nil
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func float32Ptr(f float32) *float32 {
	return &f
}
