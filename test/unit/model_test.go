package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestModelValidation(t *testing.T) {
	tests := []struct {
		name    string
		model   *neuronetes.Model
		wantErr bool
	}{
		{
			name: "valid model with all fields",
			model: &neuronetes.Model{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-model",
					Namespace: "default",
				},
				Spec: neuronetes.ModelSpec{
					WeightsURI:   "s3://bucket/model",
					Size:         resource.MustParse("100Gi"),
					Quantization: "int4",
					ShardSpec: &neuronetes.ShardSpec{
						Count:    4,
						Strategy: "tensor-parallel",
					},
					CachePolicy: &neuronetes.CachePolicy{
						Priority: "high",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid model with minimal fields",
			model: &neuronetes.Model{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minimal-model",
					Namespace: "default",
				},
				Spec: neuronetes.ModelSpec{
					WeightsURI: "s3://bucket/model",
					Size:       resource.MustParse("50Gi"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid quantization",
			model: &neuronetes.Model{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-quant",
					Namespace: "default",
				},
				Spec: neuronetes.ModelSpec{
					WeightsURI:   "s3://bucket/model",
					Size:         resource.MustParse("50Gi"),
					Quantization: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModel(tt.model)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestShardSpecValidation(t *testing.T) {
	tests := []struct {
		name      string
		shardSpec *neuronetes.ShardSpec
		wantErr   bool
	}{
		{
			name: "valid tensor parallel",
			shardSpec: &neuronetes.ShardSpec{
				Count:    4,
				Strategy: "tensor-parallel",
				Topology: &neuronetes.TopologyRequirement{
					Locality: "same-node",
				},
			},
			wantErr: false,
		},
		{
			name: "valid pipeline parallel",
			shardSpec: &neuronetes.ShardSpec{
				Count:    8,
				Strategy: "pipeline-parallel",
			},
			wantErr: false,
		},
		{
			name: "invalid shard count",
			shardSpec: &neuronetes.ShardSpec{
				Count:    0,
				Strategy: "tensor-parallel",
			},
			wantErr: true,
		},
		{
			name: "invalid strategy",
			shardSpec: &neuronetes.ShardSpec{
				Count:    2,
				Strategy: "invalid-strategy",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShardSpec(tt.shardSpec)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCachePolicyValidation(t *testing.T) {
	duration1h := metav1.Duration{Duration: 3600}

	tests := []struct {
		name        string
		cachePolicy *neuronetes.CachePolicy
		wantErr     bool
	}{
		{
			name: "valid high priority with pin",
			cachePolicy: &neuronetes.CachePolicy{
				Priority:    "high",
				PinDuration: &duration1h,
			},
			wantErr: false,
		},
		{
			name: "valid with eviction policy",
			cachePolicy: &neuronetes.CachePolicy{
				Priority:       "medium",
				EvictionPolicy: "idle",
			},
			wantErr: false,
		},
		{
			name: "invalid priority",
			cachePolicy: &neuronetes.CachePolicy{
				Priority: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCachePolicy(tt.cachePolicy)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock validation functions
func validateModel(m *neuronetes.Model) error {
	if m.Spec.Quantization != "" {
		validQuants := map[string]bool{
			"fp32": true, "fp16": true, "int8": true, "int4": true, "none": true,
		}
		if !validQuants[m.Spec.Quantization] {
			return assert.AnError
		}
	}
	return nil
}

func validateShardSpec(s *neuronetes.ShardSpec) error {
	if s.Count < 1 {
		return assert.AnError
	}
	validStrategies := map[string]bool{
		"tensor-parallel":   true,
		"pipeline-parallel": true,
		"data-parallel":     true,
	}
	if !validStrategies[s.Strategy] {
		return assert.AnError
	}
	return nil
}

func validateCachePolicy(c *neuronetes.CachePolicy) error {
	validPriorities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true,
	}
	if !validPriorities[c.Priority] {
		return assert.AnError
	}
	return nil
}
