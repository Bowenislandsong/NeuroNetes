package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ToolBindingSpec defines the desired state of ToolBinding
type ToolBindingSpec struct {
	// AgentPoolRef references the AgentPool this binding applies to
	// +kubebuilder:validation:Required
	AgentPoolRef AgentPoolReference `json:"agentPoolRef"`

	// Type is the binding type
	// +kubebuilder:validation:Enum=queue;topic;webhook;grpc;http
	Type string `json:"type"`

	// QueueConfig for queue-based bindings
	// +optional
	QueueConfig *QueueConfig `json:"queueConfig,omitempty"`

	// TopicConfig for topic-based bindings
	// +optional
	TopicConfig *TopicConfig `json:"topicConfig,omitempty"`

	// HTTPConfig for HTTP-based bindings
	// +optional
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`

	// Concurrency limits
	// +optional
	Concurrency *ConcurrencyConfig `json:"concurrency,omitempty"`

	// Timeouts for tool operations
	// +optional
	Timeouts *TimeoutConfig `json:"timeouts,omitempty"`

	// RetryPolicy defines retry behavior
	// +optional
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// AgentPoolReference references an AgentPool resource
type AgentPoolReference struct {
	// Name is the name of the AgentPool
	Name string `json:"name"`

	// Namespace is the namespace of the AgentPool
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// QueueConfig defines queue-based binding configuration
type QueueConfig struct {
	// Provider is the queue provider
	// +kubebuilder:validation:Enum=nats;kafka;sqs;rabbitmq;redis
	Provider string `json:"provider"`

	// ConnectionString is the connection details
	// +kubebuilder:validation:Required
	ConnectionString string `json:"connectionString"`

	// QueueName is the name of the queue
	// +kubebuilder:validation:Required
	QueueName string `json:"queueName"`

	// AutoscaleOnLag enables autoscaling based on queue lag
	// +optional
	AutoscaleOnLag bool `json:"autoscaleOnLag,omitempty"`

	// MaxLagThreshold is the lag threshold for scaling (messages)
	// +optional
	MaxLagThreshold *int32 `json:"maxLagThreshold,omitempty"`

	// PrefetchCount is the number of messages to prefetch
	// +optional
	PrefetchCount *int32 `json:"prefetchCount,omitempty"`

	// AckMode defines acknowledgment behavior
	// +kubebuilder:validation:Enum=auto;manual;client
	// +optional
	AckMode string `json:"ackMode,omitempty"`
}

// TopicConfig defines topic-based binding configuration
type TopicConfig struct {
	// Provider is the topic provider
	// +kubebuilder:validation:Enum=nats;kafka;pubsub;sns
	Provider string `json:"provider"`

	// ConnectionString is the connection details
	// +kubebuilder:validation:Required
	ConnectionString string `json:"connectionString"`

	// TopicName is the name of the topic
	// +kubebuilder:validation:Required
	TopicName string `json:"topicName"`

	// ConsumerGroup is the consumer group ID
	// +optional
	ConsumerGroup string `json:"consumerGroup,omitempty"`

	// Partitions is the number of partitions to consume from
	// +optional
	Partitions []int32 `json:"partitions,omitempty"`

	// AutoscaleOnLag enables autoscaling based on topic lag
	// +optional
	AutoscaleOnLag bool `json:"autoscaleOnLag,omitempty"`
}

// HTTPConfig defines HTTP-based binding configuration
type HTTPConfig struct {
	// Path is the HTTP path for this binding
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// Methods are the allowed HTTP methods
	// +optional
	Methods []string `json:"methods,omitempty"`

	// RateLimitPerIP is the rate limit per IP address
	// +optional
	RateLimitPerIP string `json:"rateLimitPerIP,omitempty"`

	// StreamingEnabled enables streaming responses
	// +optional
	StreamingEnabled bool `json:"streamingEnabled,omitempty"`

	// CORSConfig defines CORS settings
	// +optional
	CORSConfig *CORSConfig `json:"corsConfig,omitempty"`
}

// CORSConfig defines CORS settings
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins
	AllowedOrigins []string `json:"allowedOrigins"`

	// AllowedMethods is the list of allowed methods
	// +optional
	AllowedMethods []string `json:"allowedMethods,omitempty"`

	// AllowedHeaders is the list of allowed headers
	// +optional
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`

	// MaxAge is the max age for preflight requests
	// +optional
	MaxAge *int32 `json:"maxAge,omitempty"`
}

// ConcurrencyConfig defines concurrency limits
type ConcurrencyConfig struct {
	// MaxConcurrentRequests is the max concurrent requests per replica
	// +optional
	MaxConcurrentRequests *int32 `json:"maxConcurrentRequests,omitempty"`

	// MaxQueuedRequests is the max queued requests per replica
	// +optional
	MaxQueuedRequests *int32 `json:"maxQueuedRequests,omitempty"`

	// PerSessionLimit is the max concurrent requests per session
	// +optional
	PerSessionLimit *int32 `json:"perSessionLimit,omitempty"`
}

// TimeoutConfig defines timeout settings
type TimeoutConfig struct {
	// RequestTimeout is the overall request timeout
	// +optional
	RequestTimeout *metav1.Duration `json:"requestTimeout,omitempty"`

	// ToolTimeout is the timeout for tool invocations
	// +optional
	ToolTimeout *metav1.Duration `json:"toolTimeout,omitempty"`

	// IdleTimeout is the idle connection timeout
	// +optional
	IdleTimeout *metav1.Duration `json:"idleTimeout,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts
	// +kubebuilder:validation:Minimum=0
	MaxAttempts int32 `json:"maxAttempts"`

	// InitialBackoff is the initial backoff duration
	// +optional
	InitialBackoff *metav1.Duration `json:"initialBackoff,omitempty"`

	// MaxBackoff is the maximum backoff duration
	// +optional
	MaxBackoff *metav1.Duration `json:"maxBackoff,omitempty"`

	// BackoffMultiplier is the backoff multiplier
	// +optional
	BackoffMultiplier *float32 `json:"backoffMultiplier,omitempty"`

	// RetryableErrors are error patterns that should trigger retries
	// +optional
	RetryableErrors []string `json:"retryableErrors,omitempty"`
}

// ToolBindingStatus defines the observed state of ToolBinding
type ToolBindingStatus struct {
	// Phase represents the current phase
	// +kubebuilder:validation:Enum=Pending;Active;Failed;Terminating
	Phase string `json:"phase"`

	// ActiveConnections is the number of active connections
	// +optional
	ActiveConnections *int32 `json:"activeConnections,omitempty"`

	// QueuedRequests is the current number of queued requests
	// +optional
	QueuedRequests *int32 `json:"queuedRequests,omitempty"`

	// ThroughputMetrics contains throughput information
	// +optional
	ThroughputMetrics *ThroughputMetrics `json:"throughputMetrics,omitempty"`

	// LastError is the last error encountered
	// +optional
	LastError string `json:"lastError,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ThroughputMetrics contains throughput statistics
type ThroughputMetrics struct {
	// RequestsPerSecond is the current RPS
	RequestsPerSecond float32 `json:"requestsPerSecond"`

	// TokensPerSecond is the current tokens/sec
	// +optional
	TokensPerSecond *float32 `json:"tokensPerSecond,omitempty"`

	// AverageLatency is the average latency
	// +optional
	AverageLatency *metav1.Duration `json:"averageLatency,omitempty"`

	// P95Latency is the p95 latency
	// +optional
	P95Latency *metav1.Duration `json:"p95Latency,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=tb
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="AgentPool",type=string,JSONPath=`.spec.agentPoolRef.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ToolBinding is the Schema for the toolbindings API
type ToolBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolBindingSpec   `json:"spec,omitempty"`
	Status ToolBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolBindingList contains a list of ToolBinding
type ToolBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolBinding{}, &ToolBindingList{})
}
