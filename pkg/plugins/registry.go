package plugins

import (
	"context"

	neuronetes "github.com/bowenislandsong/neuronetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// SchedulerPlugin is the interface for custom scheduling algorithms
type SchedulerPlugin interface {
	// Name returns the plugin name
	Name() string

	// Filter returns true if the node is suitable for the pod
	Filter(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) bool

	// Score returns a score for the node (0-100)
	Score(ctx context.Context, pod *corev1.Pod, node *corev1.Node, pool *neuronetes.AgentPool) int64

	// Priority returns the plugin priority (higher runs first)
	Priority() int
}

// AutoscalerPlugin is the interface for custom autoscaling algorithms
type AutoscalerPlugin interface {
	// Name returns the plugin name
	Name() string

	// CalculateReplicas calculates the desired replica count
	CalculateReplicas(ctx context.Context, pool *neuronetes.AgentPool, currentMetrics map[string]float64) (int32, error)

	// GetMetricNames returns the metric names this plugin needs
	GetMetricNames() []string

	// Priority returns the plugin priority (higher runs first)
	Priority() int
}

// ModelLoaderPlugin is the interface for custom model loading strategies
type ModelLoaderPlugin interface {
	// Name returns the plugin name
	Name() string

	// CanLoad returns true if this plugin can load the model
	CanLoad(ctx context.Context, model *neuronetes.Model) bool

	// Load loads the model and returns the loading time
	Load(ctx context.Context, model *neuronetes.Model, node string) error

	// Unload unloads the model from the node
	Unload(ctx context.Context, model *neuronetes.Model, node string) error

	// Priority returns the plugin priority (higher runs first)
	Priority() int
}

// MetricsProviderPlugin is the interface for custom metrics providers
type MetricsProviderPlugin interface {
	// Name returns the plugin name
	Name() string

	// GetMetric retrieves a metric value
	GetMetric(ctx context.Context, pool *neuronetes.AgentPool, metricType string) (float64, error)

	// ListMetrics returns available metric types
	ListMetrics() []string
}

// GuardrailPlugin is the interface for custom guardrails
type GuardrailPlugin interface {
	// Name returns the plugin name
	Name() string

	// Check evaluates the guardrail
	Check(ctx context.Context, request *GuardrailRequest) (*GuardrailResult, error)

	// GetType returns the guardrail type
	GetType() string
}

// GuardrailRequest represents a request to evaluate
type GuardrailRequest struct {
	Content     string
	Metadata    map[string]string
	AgentClass  string
	SessionID   string
	RequestID   string
}

// GuardrailResult represents the result of a guardrail check
type GuardrailResult struct {
	Passed    bool
	Action    string // block, redact, warn, log
	Reason    string
	Confidence float64
	Metadata  map[string]string
}

// PluginRegistry manages all plugins
type PluginRegistry struct {
	schedulers      []SchedulerPlugin
	autoscalers     []AutoscalerPlugin
	modelLoaders    []ModelLoaderPlugin
	metricsProviders []MetricsProviderPlugin
	guardrails      []GuardrailPlugin
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		schedulers:      make([]SchedulerPlugin, 0),
		autoscalers:     make([]AutoscalerPlugin, 0),
		modelLoaders:    make([]ModelLoaderPlugin, 0),
		metricsProviders: make([]MetricsProviderPlugin, 0),
		guardrails:      make([]GuardrailPlugin, 0),
	}
}

// RegisterScheduler registers a scheduler plugin
func (r *PluginRegistry) RegisterScheduler(plugin SchedulerPlugin) {
	r.schedulers = append(r.schedulers, plugin)
}

// RegisterAutoscaler registers an autoscaler plugin
func (r *PluginRegistry) RegisterAutoscaler(plugin AutoscalerPlugin) {
	r.autoscalers = append(r.autoscalers, plugin)
}

// RegisterModelLoader registers a model loader plugin
func (r *PluginRegistry) RegisterModelLoader(plugin ModelLoaderPlugin) {
	r.modelLoaders = append(r.modelLoaders, plugin)
}

// RegisterMetricsProvider registers a metrics provider plugin
func (r *PluginRegistry) RegisterMetricsProvider(plugin MetricsProviderPlugin) {
	r.metricsProviders = append(r.metricsProviders, plugin)
}

// RegisterGuardrail registers a guardrail plugin
func (r *PluginRegistry) RegisterGuardrail(plugin GuardrailPlugin) {
	r.guardrails = append(r.guardrails, plugin)
}

// GetSchedulers returns all registered scheduler plugins
func (r *PluginRegistry) GetSchedulers() []SchedulerPlugin {
	return r.schedulers
}

// GetAutoscalers returns all registered autoscaler plugins
func (r *PluginRegistry) GetAutoscalers() []AutoscalerPlugin {
	return r.autoscalers
}

// GetModelLoaders returns all registered model loader plugins
func (r *PluginRegistry) GetModelLoaders() []ModelLoaderPlugin {
	return r.modelLoaders
}

// GetMetricsProviders returns all registered metrics provider plugins
func (r *PluginRegistry) GetMetricsProviders() []MetricsProviderPlugin {
	return r.metricsProviders
}

// GetGuardrails returns all registered guardrail plugins
func (r *PluginRegistry) GetGuardrails() []GuardrailPlugin {
	return r.guardrails
}

// Global registry instance
var globalRegistry = NewPluginRegistry()

// RegisterScheduler registers a scheduler plugin globally
func RegisterScheduler(plugin SchedulerPlugin) {
	globalRegistry.RegisterScheduler(plugin)
}

// RegisterAutoscaler registers an autoscaler plugin globally
func RegisterAutoscaler(plugin AutoscalerPlugin) {
	globalRegistry.RegisterAutoscaler(plugin)
}

// RegisterModelLoader registers a model loader plugin globally
func RegisterModelLoader(plugin ModelLoaderPlugin) {
	globalRegistry.RegisterModelLoader(plugin)
}

// RegisterMetricsProvider registers a metrics provider plugin globally
func RegisterMetricsProvider(plugin MetricsProviderPlugin) {
	globalRegistry.RegisterMetricsProvider(plugin)
}

// RegisterGuardrail registers a guardrail plugin globally
func RegisterGuardrail(plugin GuardrailPlugin) {
	globalRegistry.RegisterGuardrail(plugin)
}

// GetGlobalRegistry returns the global plugin registry
func GetGlobalRegistry() *PluginRegistry {
	return globalRegistry
}
