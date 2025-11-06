package scheduler

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
)

// GPUTopologyScheduler implements GPU-aware scheduling
type GPUTopologyScheduler struct {
	clientset *kubernetes.Clientset
	config    *SchedulerConfig
}

// SchedulerConfig defines scheduler configuration
type SchedulerConfig struct {
	// Weight for GPU topology scoring (0.0-1.0)
	GPUTopologyWeight float64

	// Weight for model cache presence (0.0-1.0)
	ModelCacheWeight float64

	// Weight for cost efficiency (0.0-1.0)
	CostWeight float64

	// Weight for data locality (0.0-1.0)
	DataLocalityWeight float64

	// Scheduling timeout
	SchedulingTimeout time.Duration
}

// NewGPUTopologyScheduler creates a new scheduler
func NewGPUTopologyScheduler(clientset *kubernetes.Clientset, config *SchedulerConfig) *GPUTopologyScheduler {
	return &GPUTopologyScheduler{
		clientset: clientset,
		config:    config,
	}
}

// ScheduleResult represents a scheduling decision
type ScheduleResult struct {
	Node   string
	Score  int64
	Reason string
}

// Schedule finds the best node for a pod
func (s *GPUTopologyScheduler) Schedule(ctx context.Context, pod *corev1.Pod, agentPool *neuronetes.AgentPool) (*ScheduleResult, error) {
	// Get all nodes
	nodes, err := s.listNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Filter nodes
	feasibleNodes := s.filterNodes(ctx, pod, agentPool, nodes)
	if len(feasibleNodes) == 0 {
		return nil, fmt.Errorf("no feasible nodes found")
	}

	// Score nodes
	scored := s.scoreNodes(ctx, pod, agentPool, feasibleNodes)

	// Return best node
	if len(scored) == 0 {
		return nil, fmt.Errorf("no nodes scored")
	}

	return &scored[0], nil
}

func (s *GPUTopologyScheduler) listNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (s *GPUTopologyScheduler) filterNodes(ctx context.Context, pod *corev1.Pod, agentPool *neuronetes.AgentPool, nodes []corev1.Node) []corev1.Node {
	var feasible []corev1.Node

	for _, node := range nodes {
		if s.nodePassesFilters(ctx, &node, pod, agentPool) {
			feasible = append(feasible, node)
		}
	}

	return feasible
}

func (s *GPUTopologyScheduler) nodePassesFilters(ctx context.Context, node *corev1.Node, pod *corev1.Pod, agentPool *neuronetes.AgentPool) bool {
	// Check node readiness
	if !s.isNodeReady(node) {
		return false
	}

	// Check GPU availability
	if agentPool.Spec.GPURequirements != nil {
		if !s.hasRequiredGPUs(node, agentPool.Spec.GPURequirements) {
			return false
		}
	}

	// Check node selector
	if agentPool.Spec.Scheduling != nil && agentPool.Spec.Scheduling.NodeSelector != nil {
		if !s.matchesNodeSelector(node, agentPool.Spec.Scheduling.NodeSelector) {
			return false
		}
	}

	// Check MIG profile
	if agentPool.Spec.MIGProfile != "" {
		if !s.hasMIGProfile(node, agentPool.Spec.MIGProfile) {
			return false
		}
	}

	return true
}

func (s *GPUTopologyScheduler) isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (s *GPUTopologyScheduler) hasRequiredGPUs(node *corev1.Node, requirements *neuronetes.GPURequirements) bool {
	// Check GPU count
	gpuCount := node.Status.Capacity["nvidia.com/gpu"]
	if gpuCount.IsZero() || int32(gpuCount.Value()) < requirements.Count {
		return false
	}

	// Check GPU type
	if requirements.Type != "" {
		gpuType, ok := node.Labels["neuronetes.io/gpu-type"]
		if !ok || gpuType != requirements.Type {
			return false
		}
	}

	// Check GPU memory
	if requirements.Memory != "" {
		gpuMemory, ok := node.Labels["neuronetes.io/gpu-memory"]
		if !ok || gpuMemory < requirements.Memory {
			return false
		}
	}

	return true
}

func (s *GPUTopologyScheduler) matchesNodeSelector(node *corev1.Node, selector map[string]string) bool {
	return labels.SelectorFromSet(selector).Matches(labels.Set(node.Labels))
}

func (s *GPUTopologyScheduler) hasMIGProfile(node *corev1.Node, profile string) bool {
	migConfig, ok := node.Labels["neuronetes.io/mig-config"]
	if !ok {
		return false
	}

	// Simple check - in production, parse and verify availability
	return len(migConfig) > 0
}

func (s *GPUTopologyScheduler) scoreNodes(ctx context.Context, pod *corev1.Pod, agentPool *neuronetes.AgentPool, nodes []corev1.Node) []ScheduleResult {
	var results []ScheduleResult

	for _, node := range nodes {
		score := s.calculateScore(ctx, &node, pod, agentPool)
		results = append(results, ScheduleResult{
			Node:   node.Name,
			Score:  score,
			Reason: "scored",
		})
	}

	// Sort by score (descending)
	sortByScore(results)

	return results
}

func (s *GPUTopologyScheduler) calculateScore(ctx context.Context, node *corev1.Node, pod *corev1.Pod, agentPool *neuronetes.AgentPool) int64 {
	var totalScore float64

	// GPU topology score
	topologyScore := s.scoreGPUTopology(node, agentPool)
	totalScore += topologyScore * s.config.GPUTopologyWeight

	// Model cache score
	cacheScore := s.scoreModelCache(node, agentPool)
	totalScore += cacheScore * s.config.ModelCacheWeight

	// Cost efficiency score
	costScore := s.scoreCostEfficiency(node, agentPool)
	totalScore += costScore * s.config.CostWeight

	// Data locality score
	localityScore := s.scoreDataLocality(node, agentPool)
	totalScore += localityScore * s.config.DataLocalityWeight

	// Normalize to 0-100
	return int64(totalScore * 100)
}

func (s *GPUTopologyScheduler) scoreGPUTopology(node *corev1.Node, agentPool *neuronetes.AgentPool) float64 {
	// Score based on GPU topology
	if agentPool.Spec.GPURequirements == nil || agentPool.Spec.GPURequirements.Topology == nil {
		return 0.5 // Neutral score
	}

	topology := agentPool.Spec.GPURequirements.Topology
	nodeTopology, ok := node.Labels["neuronetes.io/gpu-topology"]
	if !ok {
		return 0.0
	}

	// Score based on locality match
	switch topology.Locality {
	case "nvlink":
		if nodeTopology == "nvlink" {
			return 1.0
		}
		return 0.3
	case "same-node":
		return 0.8
	case "any":
		return 0.5
	default:
		return 0.5
	}
}

func (s *GPUTopologyScheduler) scoreModelCache(node *corev1.Node, agentPool *neuronetes.AgentPool) float64 {
	// Check if model is cached on node
	// In production, query model cache controller
	modelCached, ok := node.Annotations["neuronetes.io/cached-models"]
	if !ok || len(modelCached) == 0 {
		return 0.0
	}

	// Simplified: return high score if any model cached
	return 0.9
}

func (s *GPUTopologyScheduler) scoreCostEfficiency(node *corev1.Node, agentPool *neuronetes.AgentPool) float64 {
	// Score based on cost
	if agentPool.Spec.Scheduling == nil || agentPool.Spec.Scheduling.CostOptimization == nil {
		return 0.5
	}

	// Check if spot instance
	_, ok := node.Labels["node.kubernetes.io/instance-type"]
	if !ok {
		return 0.5
	}

	// Prefer spot if enabled
	if agentPool.Spec.Scheduling.CostOptimization.SpotEnabled {
		if node.Labels["karpenter.sh/capacity-type"] == "spot" {
			return 1.0
		}
		return 0.6
	}

	return 0.7
}

func (s *GPUTopologyScheduler) scoreDataLocality(node *corev1.Node, agentPool *neuronetes.AgentPool) float64 {
	// Score based on data locality
	if agentPool.Spec.Scheduling == nil || agentPool.Spec.Scheduling.DataLocality == nil {
		return 0.5
	}

	locality := agentPool.Spec.Scheduling.DataLocality

	// Check vector store affinity
	if len(locality.VectorStoreAffinity) > 0 {
		// In production, query vector store locations
		return 0.8
	}

	return 0.5
}

func sortByScore(results []ScheduleResult) {
	// Simple bubble sort for now
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}
