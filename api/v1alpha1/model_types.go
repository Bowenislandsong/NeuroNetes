package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ModelSpec defines the desired state of Model
type ModelSpec struct {
	// WeightsURI is the location of the model weights (e.g., s3://bucket/path)
	// +kubebuilder:validation:Required
	WeightsURI string `json:"weightsURI"`

	// Size is the total size of the model weights
	// +kubebuilder:validation:Required
	Size resource.Quantity `json:"size"`

	// Quantization specifies the quantization format
	// +kubebuilder:validation:Enum=fp32;fp16;int8;int4;none
	// +optional
	Quantization string `json:"quantization,omitempty"`

	// ShardSpec defines how the model should be sharded across GPUs
	// +optional
	ShardSpec *ShardSpec `json:"shardSpec,omitempty"`

	// CachePolicy defines caching behavior for this model
	// +optional
	CachePolicy *CachePolicy `json:"cachePolicy,omitempty"`

	// Format specifies the model format (e.g., safetensors, pytorch, gguf)
	// +optional
	Format string `json:"format,omitempty"`

	// Architecture describes the model architecture
	// +optional
	Architecture string `json:"architecture,omitempty"`

	// ParameterCount is the number of parameters in the model
	// +optional
	ParameterCount string `json:"parameterCount,omitempty"`
}

// ShardSpec defines model sharding configuration
type ShardSpec struct {
	// Count is the number of shards
	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count"`

	// Strategy defines the sharding strategy
	// +kubebuilder:validation:Enum=tensor-parallel;pipeline-parallel;data-parallel
	Strategy string `json:"strategy"`

	// Topology specifies GPU topology requirements
	// +optional
	Topology *TopologyRequirement `json:"topology,omitempty"`
}

// TopologyRequirement specifies GPU topology constraints
type TopologyRequirement struct {
	// Locality specifies the locality requirement
	// +kubebuilder:validation:Enum=same-node;same-socket;nvlink;any
	Locality string `json:"locality"`

	// MinBandwidth is the minimum inter-GPU bandwidth required (GB/s)
	// +optional
	MinBandwidth *resource.Quantity `json:"minBandwidth,omitempty"`
}

// CachePolicy defines caching behavior
type CachePolicy struct {
	// Priority determines eviction order
	// +kubebuilder:validation:Enum=critical;high;medium;low
	Priority string `json:"priority"`

	// PinDuration is how long to keep the model pinned in cache
	// +optional
	PinDuration *metav1.Duration `json:"pinDuration,omitempty"`

	// PreloadNodes is a list of node selectors where model should be preloaded
	// +optional
	PreloadNodes []string `json:"preloadNodes,omitempty"`

	// EvictionPolicy defines when the model can be evicted
	// +kubebuilder:validation:Enum=never;idle;low-priority
	// +optional
	EvictionPolicy string `json:"evictionPolicy,omitempty"`
}

// ModelStatus defines the observed state of Model
type ModelStatus struct {
	// Phase represents the current phase of the model
	// +kubebuilder:validation:Enum=Pending;Loading;Ready;Failed
	Phase string `json:"phase"`

	// CachedNodes lists nodes where the model is currently cached
	// +optional
	CachedNodes []NodeCacheStatus `json:"cachedNodes,omitempty"`

	// LoadTime is the time it took to load the model
	// +optional
	LoadTime *metav1.Duration `json:"loadTime,omitempty"`

	// LastUsed is the timestamp of the last usage
	// +optional
	LastUsed *metav1.Time `json:"lastUsed,omitempty"`

	// Conditions represent the latest available observations of the model's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Version tracks the model version
	// +optional
	Version string `json:"version,omitempty"`
}

// NodeCacheStatus represents caching status on a specific node
type NodeCacheStatus struct {
	// NodeName is the name of the node
	NodeName string `json:"nodeName"`

	// Status is the cache status on this node
	// +kubebuilder:validation:Enum=loading;ready;evicting;failed
	Status string `json:"status"`

	// CachedAt is when the model was cached on this node
	// +optional
	CachedAt *metav1.Time `json:"cachedAt,omitempty"`

	// Size is the actual size cached on this node
	// +optional
	Size *resource.Quantity `json:"size,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mdl
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Quantization",type=string,JSONPath=`.spec.quantization`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Model is the Schema for the models API
type Model struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelSpec   `json:"spec,omitempty"`
	Status ModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ModelList contains a list of Model
type ModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Model `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Model{}, &ModelList{})
}
