package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentPoolSpec defines the desired state of AgentPool
type AgentPoolSpec struct {
	// AgentClassRef references the AgentClass to use
	// +kubebuilder:validation:Required
	AgentClassRef AgentClassReference `json:"agentClassRef"`

	// MinReplicas is the minimum number of agent replicas
	// +kubebuilder:validation:Minimum=0
	MinReplicas int32 `json:"minReplicas"`

	// MaxReplicas is the maximum number of agent replicas
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// PrewarmPercent is the percentage of replicas to keep warm (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	PrewarmPercent int32 `json:"prewarmPercent,omitempty"`

	// TokensPerSecondBudget is the total tokens/sec capacity budget
	// +optional
	TokensPerSecondBudget *int32 `json:"tokensPerSecondBudget,omitempty"`

	// MIGProfile specifies MIG configuration (e.g., "1g.5gb", "2g.10gb")
	// +optional
	MIGProfile string `json:"migProfile,omitempty"`

	// Autoscaling defines autoscaling behavior
	// +optional
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`

	// GPURequirements specifies GPU constraints
	// +optional
	GPURequirements *GPURequirements `json:"gpuRequirements,omitempty"`

	// SessionAffinity enables sticky session routing
	// +optional
	SessionAffinity *SessionAffinityConfig `json:"sessionAffinity,omitempty"`

	// Scheduling provides scheduling hints
	// +optional
	Scheduling *SchedulingConfig `json:"scheduling,omitempty"`
}

// AgentClassReference references an AgentClass resource
type AgentClassReference struct {
	// Name is the name of the AgentClass
	Name string `json:"name"`

	// Namespace is the namespace of the AgentClass
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// AutoscalingSpec defines autoscaling configuration
type AutoscalingSpec struct {
	// Metrics are the metrics to use for autoscaling
	Metrics []AutoscalingMetric `json:"metrics"`

	// Behavior defines scaling behavior (scale up/down rates)
	// +optional
	Behavior *ScalingBehavior `json:"behavior,omitempty"`

	// CooldownPeriod is the time to wait between scaling operations
	// +optional
	CooldownPeriod *metav1.Duration `json:"cooldownPeriod,omitempty"`
}

// AutoscalingMetric defines a single autoscaling metric
type AutoscalingMetric struct {
	// Type is the metric type
	// +kubebuilder:validation:Enum=tokens-in-queue;ttft-p95;concurrent-sessions;tokens-per-second;queue-depth;context-length;tool-call-rate
	Type string `json:"type"`

	// Target is the target value for this metric
	// +kubebuilder:validation:Required
	Target string `json:"target"`

	// AveragingWindow is the time window for averaging the metric
	// +optional
	AveragingWindow *metav1.Duration `json:"averagingWindow,omitempty"`
}

// ScalingBehavior controls scaling velocity
type ScalingBehavior struct {
	// ScaleUp defines scale-up behavior
	// +optional
	ScaleUp *ScalingPolicy `json:"scaleUp,omitempty"`

	// ScaleDown defines scale-down behavior
	// +optional
	ScaleDown *ScalingPolicy `json:"scaleDown,omitempty"`
}

// ScalingPolicy defines scaling rate limits
type ScalingPolicy struct {
	// StabilizationWindow is the time to consider past recommendations
	// +optional
	StabilizationWindow *metav1.Duration `json:"stabilizationWindow,omitempty"`

	// MaxChangePercent is the maximum change as a percentage
	// +optional
	MaxChangePercent *int32 `json:"maxChangePercent,omitempty"`

	// MaxChangeAbsolute is the maximum absolute change
	// +optional
	MaxChangeAbsolute *int32 `json:"maxChangeAbsolute,omitempty"`

	// PeriodSeconds is how often to evaluate
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
}

// GPURequirements specifies GPU constraints
type GPURequirements struct {
	// Count is the number of GPUs per replica
	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count"`

	// Memory is the minimum GPU memory per GPU
	// +optional
	Memory string `json:"memory,omitempty"`

	// Type is the GPU type (e.g., "A100", "H100")
	// +optional
	Type string `json:"type,omitempty"`

	// Topology specifies GPU topology requirements
	// +optional
	Topology *TopologyRequirement `json:"topology,omitempty"`
}

// SessionAffinityConfig defines sticky session behavior
type SessionAffinityConfig struct {
	// Enabled turns on session affinity
	Enabled bool `json:"enabled"`

	// KeyHeader is the HTTP header containing the session key
	// +optional
	KeyHeader string `json:"keyHeader,omitempty"`

	// TTL is how long to maintain affinity after last use
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`

	// Type is the affinity type
	// +kubebuilder:validation:Enum=conversation-id;user-id;custom
	// +optional
	Type string `json:"type,omitempty"`
}

// SchedulingConfig provides scheduling hints
type SchedulingConfig struct {
	// Priority is the scheduling priority
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// CostOptimization enables cost-aware scheduling
	// +optional
	CostOptimization *CostOptimizationConfig `json:"costOptimization,omitempty"`

	// DataLocality specifies data locality requirements
	// +optional
	DataLocality *DataLocalityConfig `json:"dataLocality,omitempty"`

	// NodeSelector is a label selector for nodes
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// CostOptimizationConfig defines cost optimization behavior
type CostOptimizationConfig struct {
	// Enabled turns on cost optimization
	Enabled bool `json:"enabled"`

	// MaxCostPerHour is the maximum hourly cost
	// +optional
	MaxCostPerHour *float32 `json:"maxCostPerHour,omitempty"`

	// SpotEnabled allows use of spot instances
	// +optional
	SpotEnabled bool `json:"spotEnabled,omitempty"`

	// SLOHeadroomMs is the minimum SLO headroom to use spot (ms)
	// +optional
	SLOHeadroomMs *int32 `json:"sloHeadroomMs,omitempty"`

	// FallbackModel is a cheaper model to fall back to
	// +optional
	FallbackModel string `json:"fallbackModel,omitempty"`
}

// DataLocalityConfig specifies data locality requirements
type DataLocalityConfig struct {
	// VectorStoreAffinity enables co-location with vector stores
	// +optional
	VectorStoreAffinity []string `json:"vectorStoreAffinity,omitempty"`

	// CacheAffinity specifies required cache co-location
	// +optional
	CacheAffinity []string `json:"cacheAffinity,omitempty"`

	// AntiAffinity prevents co-location with specific components
	// +optional
	AntiAffinity []string `json:"antiAffinity,omitempty"`
}

// AgentPoolStatus defines the observed state of AgentPool
type AgentPoolStatus struct {
	// Replicas is the current number of replicas
	Replicas int32 `json:"replicas"`

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas"`

	// PrewarmedReplicas is the number of prewarmed replicas
	// +optional
	PrewarmedReplicas int32 `json:"prewarmedReplicas,omitempty"`

	// CurrentTokensPerSecond is the current throughput
	// +optional
	CurrentTokensPerSecond *int32 `json:"currentTokensPerSecond,omitempty"`

	// CurrentMetrics contains the current autoscaling metrics
	// +optional
	CurrentMetrics []CurrentMetric `json:"currentMetrics,omitempty"`

	// LastScaleTime is the last time the pool was scaled
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// CurrentMetric represents a current metric value
type CurrentMetric struct {
	// Type is the metric type
	Type string `json:"type"`

	// Current is the current value
	Current string `json:"current"`

	// Target is the target value
	Target string `json:"target"`

	// Timestamp is when this was measured
	// +optional
	Timestamp *metav1.Time `json:"timestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.minReplicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:scope=Namespaced,shortName=ap
// +kubebuilder:printcolumn:name="AgentClass",type=string,JSONPath=`.spec.agentClassRef.name`
// +kubebuilder:printcolumn:name="Min",type=integer,JSONPath=`.spec.minReplicas`
// +kubebuilder:printcolumn:name="Max",type=integer,JSONPath=`.spec.maxReplicas`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentPool is the Schema for the agentpools API
type AgentPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentPoolSpec   `json:"spec,omitempty"`
	Status AgentPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentPoolList contains a list of AgentPool
type AgentPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentPool{}, &AgentPoolList{})
}
