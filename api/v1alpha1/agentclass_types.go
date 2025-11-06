package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentClassSpec defines the desired state of AgentClass
type AgentClassSpec struct {
	// ModelRef references the Model to use for this agent class
	// +kubebuilder:validation:Required
	ModelRef ModelReference `json:"modelRef"`

	// MaxContextLength is the maximum context window size in tokens
	// +kubebuilder:validation:Minimum=1
	// +optional
	MaxContextLength int32 `json:"maxContextLength,omitempty"`

	// ToolPermissions defines which tools this agent class can access
	// +optional
	ToolPermissions []ToolPermission `json:"toolPermissions,omitempty"`

	// Guardrails defines safety and policy checks
	// +optional
	Guardrails []Guardrail `json:"guardrails,omitempty"`

	// SLO defines service level objectives for this agent class
	// +optional
	SLO *ServiceLevelObjective `json:"slo,omitempty"`

	// SystemPrompt is the default system prompt for this agent class
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`

	// Temperature controls randomness in generation
	// +optional
	Temperature *float32 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	// +optional
	MaxTokens *int32 `json:"maxTokens,omitempty"`

	// MemoryConfig defines memory/state management
	// +optional
	MemoryConfig *MemoryConfig `json:"memoryConfig,omitempty"`
}

// ModelReference references a Model resource
type ModelReference struct {
	// Name is the name of the Model resource
	Name string `json:"name"`

	// Namespace is the namespace of the Model (defaults to same namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ToolPermission defines access to a specific tool
type ToolPermission struct {
	// Name is the tool identifier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// RateLimit defines rate limiting for this tool (e.g., "100/min", "10/sec")
	// +optional
	RateLimit string `json:"rateLimit,omitempty"`

	// Timeout is the maximum execution time for this tool
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// MaxConcurrency is the maximum concurrent invocations
	// +optional
	MaxConcurrency *int32 `json:"maxConcurrency,omitempty"`

	// RequiredScopes are the permission scopes required
	// +optional
	RequiredScopes []string `json:"requiredScopes,omitempty"`
}

// Guardrail defines a safety or policy check
type Guardrail struct {
	// Type is the guardrail type
	// +kubebuilder:validation:Enum=pii-detection;safety-check;content-filter;jailbreak-detection;prompt-injection
	Type string `json:"type"`

	// Action defines what to do when guardrail triggers
	// +kubebuilder:validation:Enum=block;redact;warn;log
	Action string `json:"action"`

	// Config provides guardrail-specific configuration
	// +optional
	Config map[string]string `json:"config,omitempty"`

	// Threshold is the confidence threshold for triggering (0.0-1.0)
	// +optional
	Threshold *float32 `json:"threshold,omitempty"`
}

// ServiceLevelObjective defines performance targets
type ServiceLevelObjective struct {
	// TTFT is the target time-to-first-token
	// +optional
	TTFT *metav1.Duration `json:"ttft,omitempty"`

	// TokensPerSecond is the target generation throughput
	// +optional
	TokensPerSecond *int32 `json:"tokensPerSecond,omitempty"`

	// P95Latency is the target p95 end-to-end latency
	// +optional
	P95Latency *metav1.Duration `json:"p95Latency,omitempty"`

	// MaxCostPerRequest is the maximum cost per request in dollars
	// +optional
	MaxCostPerRequest *float32 `json:"maxCostPerRequest,omitempty"`

	// AvailabilityPercent is the target availability (e.g., 99.9)
	// +optional
	AvailabilityPercent *float32 `json:"availabilityPercent,omitempty"`
}

// MemoryConfig defines agent memory/state management
type MemoryConfig struct {
	// Type is the memory backend type
	// +kubebuilder:validation:Enum=ephemeral;redis;memcached;postgres
	Type string `json:"type"`

	// TTL is the time-to-live for memory entries
	// +optional
	TTL *metav1.Duration `json:"ttl,omitempty"`

	// MaxSize is the maximum memory size per session
	// +optional
	MaxSize *int32 `json:"maxSize,omitempty"`

	// Encrypted indicates if memory should be encrypted at rest
	// +optional
	Encrypted bool `json:"encrypted,omitempty"`

	// ConnectionString for external memory backend
	// +optional
	ConnectionString string `json:"connectionString,omitempty"`
}

// AgentClassStatus defines the observed state of AgentClass
type AgentClassStatus struct {
	// ActivePools lists pools using this agent class
	// +optional
	ActivePools []string `json:"activePools,omitempty"`

	// TotalInstances is the total number of agent instances
	// +optional
	TotalInstances int32 `json:"totalInstances,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ac
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelRef.name`
// +kubebuilder:printcolumn:name="MaxContext",type=integer,JSONPath=`.spec.maxContextLength`
// +kubebuilder:printcolumn:name="Instances",type=integer,JSONPath=`.status.totalInstances`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgentClass is the Schema for the agentclasses API
type AgentClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentClassSpec   `json:"spec,omitempty"`
	Status AgentClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentClassList contains a list of AgentClass
type AgentClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgentClass{}, &AgentClassList{})
}
