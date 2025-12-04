package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// PluginExecutor wraps a compiled plugin to implement NodeExecutor interface
type PluginExecutor struct {
	pluginID   string
	pluginInfo plugins.PluginInfo
	plugin     interface{} // DiscoveryPlugin or ExtractionPlugin
	logger     *zap.Logger
	metrics    *plugins.PluginMetrics
	health     *plugins.PluginHealth
}

// NewPluginExecutor creates a new plugin executor wrapper
func NewPluginExecutor(loaded *LoadedPlugin, logger *zap.Logger) *PluginExecutor {
	return &PluginExecutor{
		pluginID:   loaded.Info.ID,
		pluginInfo: loaded.Info,
		plugin:     loaded.Instance,
		logger:     logger.With(zap.String("plugin_id", loaded.Info.ID)),
		metrics: &plugins.PluginMetrics{
			ExecutionCount: 0,
			TotalDuration:  0,
			SuccessCount:   0,
			FailureCount:   0,
		},
		health: &plugins.PluginHealth{
			IsHealthy:        true,
			ConsecutiveFails: 0,
		},
	}
}

// Execute executes the plugin
func (pe *PluginExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	startTime := time.Now()
	pe.metrics.ExecutionCount++

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			pe.logger.Error("Plugin panic recovered",
				zap.Any("panic", r),
				zap.String("plugin", pe.pluginInfo.Name))
			pe.recordFailure(fmt.Errorf("plugin panic: %v", r))
		}
	}()

	var output *nodes.ExecutionOutput
	var err error

	// Execute based on plugin type
	switch p := pe.plugin.(type) {
	case plugins.DiscoveryPlugin:
		output, err = pe.executeDiscovery(ctx, p, input)
	case plugins.ExtractionPlugin:
		output, err = pe.executeExtraction(ctx, p, input)
	default:
		err = fmt.Errorf("unknown plugin type: %T", pe.plugin)
	}

	// Update metrics
	duration := time.Since(startTime)
	pe.metrics.TotalDuration += duration
	pe.metrics.AverageDuration = pe.metrics.TotalDuration / time.Duration(pe.metrics.ExecutionCount)
	pe.metrics.LastExecutedAt = time.Now()

	if err != nil {
		pe.recordFailure(err)
		return nil, err
	}

	pe.recordSuccess()
	return output, nil
}

// executeDiscovery executes a discovery plugin
func (pe *PluginExecutor) executeDiscovery(ctx context.Context, plugin plugins.DiscoveryPlugin, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	// Convert ExecutionInput to DiscoveryInput
	discoveryInput := &plugins.DiscoveryInput{
		BrowserContext:   input.BrowserContext,
		URL:              input.URLItem.URL,
		URLItem:          input.URLItem,
		ExecutionContext: input.ExecutionContext,
		Config:           input.Params,
		ExecutionID:      input.ExecutionID,
	}

	// Execute plugin
	result, err := plugin.Discover(ctx, discoveryInput)
	if err != nil {
		return nil, fmt.Errorf("discovery plugin error: %w", err)
	}

	// Update metrics
	pe.metrics.URLsDiscovered += int64(len(result.DiscoveredURLs))

	// Convert DiscoveryOutput to ExecutionOutput
	return &nodes.ExecutionOutput{
		Result:         result,
		Metadata:       result.Metadata,
		DiscoveredURLs: result.DiscoveredURLs,
	}, nil
}

// executeExtraction executes an extraction plugin
func (pe *PluginExecutor) executeExtraction(ctx context.Context, plugin plugins.ExtractionPlugin, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
	// Convert ExecutionInput to ExtractionInput
	extractionInput := &plugins.ExtractionInput{
		BrowserContext:   input.BrowserContext,
		URL:              input.URLItem.URL,
		URLItem:          input.URLItem,
		ExecutionContext: input.ExecutionContext,
		Config:           input.Params,
		ExecutionID:      input.ExecutionID,
	}

	// Execute plugin
	result, err := plugin.Extract(ctx, extractionInput)
	if err != nil {
		return nil, fmt.Errorf("extraction plugin error: %w", err)
	}

	// Update metrics
	pe.metrics.ItemsExtracted++

	// Convert ExtractionOutput to ExecutionOutput
	return &nodes.ExecutionOutput{
		Result:         result.Data,
		Metadata:       result.Metadata,
		DiscoveredURLs: result.DiscoveredURLs,
	}, nil
}

// Validate validates plugin configuration
func (pe *PluginExecutor) Validate(params map[string]interface{}) error {
	switch p := pe.plugin.(type) {
	case plugins.DiscoveryPlugin:
		return p.Validate(params)
	case plugins.ExtractionPlugin:
		return p.Validate(params)
	default:
		return fmt.Errorf("unknown plugin type")
	}
}

// Type returns the node type
func (pe *PluginExecutor) Type() models.NodeType {
	// Use plugin ID as node type for tracking
	return models.NodeType(fmt.Sprintf("plugin_%s", pe.pluginID))
}

// GetPluginInfo returns plugin information
func (pe *PluginExecutor) GetPluginInfo() plugins.PluginInfo {
	return pe.pluginInfo
}

// GetMetrics returns plugin execution metrics
func (pe *PluginExecutor) GetMetrics() *plugins.PluginMetrics {
	return pe.metrics
}

// GetHealth returns plugin health status
func (pe *PluginExecutor) GetHealth() *plugins.PluginHealth {
	return pe.health
}

// recordSuccess updates metrics and health for successful execution
func (pe *PluginExecutor) recordSuccess() {
	pe.metrics.SuccessCount++
	pe.health.IsHealthy = true
	pe.health.ConsecutiveFails = 0
}

// recordFailure updates metrics and health for failed execution
func (pe *PluginExecutor) recordFailure(err error) {
	pe.metrics.FailureCount++
	pe.health.ConsecutiveFails++
	pe.health.LastError = err.Error()
	pe.health.LastErrorAt = time.Now()

	// Mark unhealthy after 3 consecutive failures
	if pe.health.ConsecutiveFails >= 3 {
		pe.health.IsHealthy = false
		pe.logger.Warn("Plugin marked unhealthy",
			zap.String("plugin", pe.pluginInfo.Name),
			zap.Int("consecutive_fails", pe.health.ConsecutiveFails))
	}
}
