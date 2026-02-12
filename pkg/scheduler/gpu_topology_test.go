package scheduler

import (
"context"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
corev1 "k8s.io/api/core/v1"
"k8s.io/apimachinery/pkg/api/resource"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/client-go/kubernetes/fake"

neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

func TestSchedule_NoFeasibleNodes(t *testing.T) {
clientset := fake.NewSimpleClientset()
config := &SchedulerConfig{
GPUTopologyWeight:  0.4,
ModelCacheWeight:   0.3,
CostWeight:         0.2,
DataLocalityWeight: 0.1,
SchedulingTimeout:  30 * time.Second,
}
scheduler := &GPUTopologyScheduler{
clientset: clientset,
config:    config,
}

ctx := context.Background()
pod := &corev1.Pod{
ObjectMeta: metav1.ObjectMeta{
Name:      "test-pod",
Namespace: "default",
},
}
agentPool := &neuronetes.AgentPool{
Spec: neuronetes.AgentPoolSpec{
GPURequirements: &neuronetes.GPURequirements{
Count: 4,
},
},
}

// Create a node that doesn't meet GPU requirements
node := &corev1.Node{
ObjectMeta: metav1.ObjectMeta{
Name: "node-1",
},
Status: corev1.NodeStatus{
Conditions: []corev1.NodeCondition{
{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
},
Capacity: corev1.ResourceList{
"nvidia.com/gpu": resource.MustParse("2"), // Less than required
},
},
}
_, err := clientset.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
require.NoError(t, err)

_, err = scheduler.Schedule(ctx, pod, agentPool)
assert.Error(t, err, "Should have no feasible nodes")
}

func TestSchedule_WithFeasibleNodes(t *testing.T) {
clientset := fake.NewSimpleClientset()
config := &SchedulerConfig{
GPUTopologyWeight:  0.4,
ModelCacheWeight:   0.3,
CostWeight:         0.2,
DataLocalityWeight: 0.1,
SchedulingTimeout:  30 * time.Second,
}
scheduler := &GPUTopologyScheduler{
clientset: clientset,
config:    config,
}

ctx := context.Background()
pod := &corev1.Pod{
ObjectMeta: metav1.ObjectMeta{
Name:      "test-pod",
Namespace: "default",
},
}
agentPool := &neuronetes.AgentPool{
Spec: neuronetes.AgentPoolSpec{
GPURequirements: &neuronetes.GPURequirements{
Count: 2,
},
},
}

// Create nodes with different configurations
nodes := []*corev1.Node{
{
ObjectMeta: metav1.ObjectMeta{
Name: "node-1",
Labels: map[string]string{
"neuronetes.io/gpu-topology": "nvlink",
},
},
Status: corev1.NodeStatus{
Conditions: []corev1.NodeCondition{
{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
},
Capacity: corev1.ResourceList{
"nvidia.com/gpu": resource.MustParse("4"),
},
},
},
{
ObjectMeta: metav1.ObjectMeta{
Name: "node-2",
Labels: map[string]string{
"neuronetes.io/gpu-topology": "pcie",
},
},
Status: corev1.NodeStatus{
Conditions: []corev1.NodeCondition{
{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
},
Capacity: corev1.ResourceList{
"nvidia.com/gpu": resource.MustParse("2"),
},
},
},
}

for _, node := range nodes {
_, err := clientset.CoreV1().Nodes().Create(ctx, node, metav1.CreateOptions{})
require.NoError(t, err)
}

result, err := scheduler.Schedule(ctx, pod, agentPool)
require.NoError(t, err)
assert.NotNil(t, result)
assert.NotEmpty(t, result.Node)
assert.GreaterOrEqual(t, result.Score, int64(0))
}

func TestSortByScore(t *testing.T) {
tests := []struct {
name     string
input    []ScheduleResult
expected []string // Expected node names in order
}{
{
name: "already sorted",
input: []ScheduleResult{
{Node: "node-1", Score: 100},
{Node: "node-2", Score: 80},
{Node: "node-3", Score: 60},
},
expected: []string{"node-1", "node-2", "node-3"},
},
{
name: "reverse order",
input: []ScheduleResult{
{Node: "node-1", Score: 60},
{Node: "node-2", Score: 80},
{Node: "node-3", Score: 100},
},
expected: []string{"node-3", "node-2", "node-1"},
},
{
name: "random order",
input: []ScheduleResult{
{Node: "node-1", Score: 80},
{Node: "node-2", Score: 100},
{Node: "node-3", Score: 60},
},
expected: []string{"node-2", "node-1", "node-3"},
},
{
name:     "empty list",
input:    []ScheduleResult{},
expected: []string{},
},
{
name: "single element",
input: []ScheduleResult{
{Node: "node-1", Score: 100},
},
expected: []string{"node-1"},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
sortByScore(tt.input)

actual := make([]string, len(tt.input))
for i, result := range tt.input {
actual[i] = result.Node
}

assert.Equal(t, tt.expected, actual)
})
}
}
