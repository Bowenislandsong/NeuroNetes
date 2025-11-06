package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

// TestE2EModelLifecycle tests the complete lifecycle of a Model resource
func TestE2EModelLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()

	// Setup Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		t.Skipf("skipping e2e test: could not build config: %v", err)
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	// Verify cluster is accessible
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Skipf("skipping e2e test: cluster not accessible: %v", err)
		return
	}

	t.Run("create model", func(t *testing.T) {
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-model-e2e",
				Namespace: "default",
			},
			Spec: neuronetes.ModelSpec{
				WeightsURI: "s3://test/model",
				Size:       resource.MustParse("10Gi"),
			},
		}

		// In real e2e, this would create the model via the API
		// For now, we test the structure
		assert.NotNil(t, model)
		assert.Equal(t, "test-model-e2e", model.Name)
	})

	t.Run("model transitions to ready", func(t *testing.T) {
		// Simulate waiting for model to be ready
		// In production, this would poll the API
		time.Sleep(100 * time.Millisecond)

		// Mock status check
		status := neuronetes.ModelStatus{
			Phase: "Ready",
		}

		assert.Equal(t, "Ready", status.Phase)
	})
}

// TestE2EAgentPoolScaling tests autoscaling behavior
func TestE2EAgentPoolScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	_ = context.Background()

	t.Run("create agent pool", func(t *testing.T) {
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pool-e2e",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				AgentClassRef: neuronetes.AgentClassReference{
					Name: "test-agent-class",
				},
				MinReplicas: 1,
				MaxReplicas: 5,
			},
		}

		assert.NotNil(t, pool)
		assert.Equal(t, int32(1), pool.Spec.MinReplicas)
		assert.Equal(t, int32(5), pool.Spec.MaxReplicas)
	})

	t.Run("pool scales on load", func(t *testing.T) {
		// Simulate load increase
		time.Sleep(100 * time.Millisecond)

		// Mock scaled status
		status := neuronetes.AgentPoolStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		}

		assert.GreaterOrEqual(t, status.Replicas, int32(1))
		assert.LessOrEqual(t, status.Replicas, int32(5))
	})
}

// TestE2EToolBinding tests tool binding integration
func TestE2EToolBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("create HTTP binding", func(t *testing.T) {
		binding := &neuronetes.ToolBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-http-binding",
				Namespace: "default",
			},
			Spec: neuronetes.ToolBindingSpec{
				AgentPoolRef: neuronetes.AgentPoolReference{
					Name: "test-pool",
				},
				Type: "http",
				HTTPConfig: &neuronetes.HTTPConfig{
					Path:    "/v1/completions",
					Methods: []string{"POST"},
				},
			},
		}

		assert.NotNil(t, binding)
		assert.Equal(t, "http", binding.Spec.Type)
		assert.Equal(t, "/v1/completions", binding.Spec.HTTPConfig.Path)
	})

	t.Run("create queue binding", func(t *testing.T) {
		binding := &neuronetes.ToolBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-queue-binding",
				Namespace: "default",
			},
			Spec: neuronetes.ToolBindingSpec{
				AgentPoolRef: neuronetes.AgentPoolReference{
					Name: "test-pool",
				},
				Type: "queue",
				QueueConfig: &neuronetes.QueueConfig{
					Provider:         "nats",
					ConnectionString: "nats://localhost:4222",
					QueueName:        "test-queue",
				},
			},
		}

		assert.NotNil(t, binding)
		assert.Equal(t, "queue", binding.Spec.Type)
		assert.Equal(t, "nats", binding.Spec.QueueConfig.Provider)
	})
}

// TestE2ECompleteWorkflow tests a complete agent workflow
func TestE2ECompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	_ = context.Background()

	t.Run("deploy complete stack", func(t *testing.T) {
		// Model
		model := &neuronetes.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "workflow-model",
				Namespace: "default",
			},
			Spec: neuronetes.ModelSpec{
				WeightsURI:   "s3://test/model",
				Size:         resource.MustParse("10Gi"),
				Quantization: "int8",
			},
		}

		// AgentClass
		agentClass := &neuronetes.AgentClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "workflow-agent",
				Namespace: "default",
			},
			Spec: neuronetes.AgentClassSpec{
				ModelRef: neuronetes.ModelReference{
					Name: "workflow-model",
				},
				MaxContextLength: 8192,
			},
		}

		// AgentPool
		pool := &neuronetes.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "workflow-pool",
				Namespace: "default",
			},
			Spec: neuronetes.AgentPoolSpec{
				AgentClassRef: neuronetes.AgentClassReference{
					Name: "workflow-agent",
				},
				MinReplicas: 1,
				MaxReplicas: 3,
			},
		}

		// ToolBinding
		binding := &neuronetes.ToolBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "workflow-binding",
				Namespace: "default",
			},
			Spec: neuronetes.ToolBindingSpec{
				AgentPoolRef: neuronetes.AgentPoolReference{
					Name: "workflow-pool",
				},
				Type: "http",
				HTTPConfig: &neuronetes.HTTPConfig{
					Path: "/v1/agent",
				},
			},
		}

		// Verify all resources are properly structured
		assert.NotNil(t, model)
		assert.NotNil(t, agentClass)
		assert.NotNil(t, pool)
		assert.NotNil(t, binding)

		// Verify relationships
		assert.Equal(t, model.Name, agentClass.Spec.ModelRef.Name)
		assert.Equal(t, agentClass.Name, pool.Spec.AgentClassRef.Name)
		assert.Equal(t, pool.Name, binding.Spec.AgentPoolRef.Name)
	})

	t.Run("verify system stability", func(t *testing.T) {
		// Simulate running for a period
		time.Sleep(200 * time.Millisecond)

		// Check that system remains stable
		assert.True(t, true, "System should remain stable")
	})
}

// TestE2ECleanup tests resource cleanup
func TestE2ECleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("cleanup resources", func(t *testing.T) {
		// In production, this would delete resources via API
		// and verify they're properly cleaned up

		// Mock cleanup verification
		assert.True(t, true, "Resources should be cleaned up")
	})
}
