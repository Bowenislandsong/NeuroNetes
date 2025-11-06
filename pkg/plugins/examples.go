package plugins

import (
	"context"
	"fmt"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ExampleSchedulerPlugin demonstrates how to create a custom scheduler plugin
type ExampleSchedulerPlugin struct {
	name string
}

// NewExampleSchedulerPlugin creates a new example scheduler plugin
func NewExampleSchedulerPlugin() *ExampleSchedulerPlugin {
	return &ExampleSchedulerPlugin{
		name: "example-scheduler",
	}
}

func (p *ExampleSchedulerPlugin) Name() string {
	return p.name
}

func (p *ExampleSchedulerPlugin) Filter(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) bool {
	// Example: Filter nodes based on custom label
	if pool.Spec.Scheduling != nil && pool.Spec.Scheduling.NodeSelector != nil {
		for key, value := range pool.Spec.Scheduling.NodeSelector {
			if nodeValue, ok := node.Labels[key]; !ok || nodeValue != value {
				return false
			}
		}
	}
	return true
}

func (p *ExampleSchedulerPlugin) Score(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) int64 {
	// Example: Score based on available resources
	var score int64 = 50 // Base score
	
	// Bonus for nodes with specific labels
	if _, ok := node.Labels["neuronetes.io/gpu-type"]; ok {
		score += 20
	}
	
	if _, ok := node.Labels["neuronetes.io/high-bandwidth"]; ok {
		score += 15
	}
	
	return score
}

func (p *ExampleSchedulerPlugin) Priority() int {
	return 100 // Medium priority
}

// ExampleAutoscalerPlugin demonstrates how to create a custom autoscaler plugin
type ExampleAutoscalerPlugin struct {
	name string
}

// NewExampleAutoscalerPlugin creates a new example autoscaler plugin
func NewExampleAutoscalerPlugin() *ExampleAutoscalerPlugin {
	return &ExampleAutoscalerPlugin{
		name: "example-autoscaler",
	}
}

func (p *ExampleAutoscalerPlugin) Name() string {
	return p.name
}

func (p *ExampleAutoscalerPlugin) CalculateReplicas(ctx context.Context, pool *neuronetes.AgentPool, currentMetrics map[string]float64) (int32, error) {
	// Example: Calculate replicas based on custom metric
	customMetric, ok := currentMetrics["custom-load"]
	if !ok {
		return pool.Status.Replicas, nil
	}
	
	// Simple scaling logic
	targetLoad := 70.0
	currentReplicas := float64(pool.Status.Replicas)
	
	if currentReplicas == 0 {
		return pool.Spec.MinReplicas, nil
	}
	
	desiredReplicas := int32(currentReplicas * (customMetric / targetLoad))
	
	// Apply bounds
	if desiredReplicas < pool.Spec.MinReplicas {
		desiredReplicas = pool.Spec.MinReplicas
	}
	if desiredReplicas > pool.Spec.MaxReplicas {
		desiredReplicas = pool.Spec.MaxReplicas
	}
	
	return desiredReplicas, nil
}

func (p *ExampleAutoscalerPlugin) GetMetricNames() []string {
	return []string{"custom-load"}
}

func (p *ExampleAutoscalerPlugin) Priority() int {
	return 100
}

// ExampleModelLoaderPlugin demonstrates how to create a custom model loader plugin
type ExampleModelLoaderPlugin struct {
	name string
}

// NewExampleModelLoaderPlugin creates a new example model loader plugin
func NewExampleModelLoaderPlugin() *ExampleModelLoaderPlugin {
	return &ExampleModelLoaderPlugin{
		name: "example-loader",
	}
}

func (p *ExampleModelLoaderPlugin) Name() string {
	return p.name
}

func (p *ExampleModelLoaderPlugin) CanLoad(ctx context.Context, model *neuronetes.Model) bool {
	// Example: Check if we can handle this model format
	return model.Spec.Format == "custom-format"
}

func (p *ExampleModelLoaderPlugin) Load(ctx context.Context, model *neuronetes.Model, node string) error {
	// Example: Custom loading logic
	fmt.Printf("Loading model %s on node %s\n", model.Name, node)
	// Implement actual loading logic here
	return nil
}

func (p *ExampleModelLoaderPlugin) Unload(ctx context.Context, model *neuronetes.Model, node string) error {
	// Example: Custom unloading logic
	fmt.Printf("Unloading model %s from node %s\n", model.Name, node)
	// Implement actual unloading logic here
	return nil
}

func (p *ExampleModelLoaderPlugin) Priority() int {
	return 50
}

// ExampleGuardrailPlugin demonstrates how to create a custom guardrail plugin
type ExampleGuardrailPlugin struct {
	name string
}

// NewExampleGuardrailPlugin creates a new example guardrail plugin
func NewExampleGuardrailPlugin() *ExampleGuardrailPlugin {
	return &ExampleGuardrailPlugin{
		name: "example-guardrail",
	}
}

func (p *ExampleGuardrailPlugin) Name() string {
	return p.name
}

func (p *ExampleGuardrailPlugin) Check(ctx context.Context, request *GuardrailRequest) (*GuardrailResult, error) {
	// Example: Simple keyword-based guardrail
	blockedKeywords := []string{"forbidden", "blocked"}
	
	for _, keyword := range blockedKeywords {
		if contains(request.Content, keyword) {
			return &GuardrailResult{
				Passed:     false,
				Action:     "block",
				Reason:     fmt.Sprintf("Content contains blocked keyword: %s", keyword),
				Confidence: 1.0,
			}, nil
		}
	}
	
	return &GuardrailResult{
		Passed:     true,
		Action:     "allow",
		Confidence: 1.0,
	}, nil
}

func (p *ExampleGuardrailPlugin) GetType() string {
	return "custom-keyword-filter"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

// init registers the example plugins
func init() {
	// Uncomment to auto-register example plugins
	// RegisterScheduler(NewExampleSchedulerPlugin())
	// RegisterAutoscaler(NewExampleAutoscalerPlugin())
	// RegisterModelLoader(NewExampleModelLoaderPlugin())
	// RegisterGuardrail(NewExampleGuardrailPlugin())
}
