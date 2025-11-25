package nodes

import (
	"context"
	"fmt"
	"plugin"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// PluginNodeExecutor executes plugin-based nodes
type PluginNodeExecutor struct {
	logger *zap.Logger
}

// NewPluginNodeExecutor creates a new plugin node executor
func NewPluginNodeExecutor(logger *zap.Logger) *PluginNodeExecutor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PluginNodeExecutor{
		logger: logger,
	}
}

// Type returns the node type
func (e *PluginNodeExecutor) Type() models.NodeType {
	return models.NodeTypePlugin
}

// Validate validates node parameters
func (e *PluginNodeExecutor) Validate(params map[string]interface{}) error {
	// plugin_slug is required
	if _, ok := params["plugin_slug"]; !ok {
		return fmt.Errorf("plugin_slug is required")
	}
	if slug, ok := params["plugin_slug"].(string); !ok || slug == "" {
		return fmt.Errorf("plugin_slug must be a non-empty string")
	}
	return nil
}

// Execute executes the plugin node
func (e *PluginNodeExecutor) Execute(ctx context.Context, input *ExecutionInput) (*ExecutionOutput, error) {
	// Get plugin slug from params
	pluginSlug, ok := input.Params["plugin_slug"].(string)
	if !ok {
		return nil, fmt.Errorf("plugin_slug not found in params")
	}

	// Get plugin config (optional)
	pluginConfig, _ := input.Params["config"].(map[string]interface{})
	if pluginConfig == nil {
		pluginConfig = make(map[string]interface{})
	}

	e.logger.Info("Executing plugin node",
		zap.String("plugin_slug", pluginSlug),
		zap.String("url", input.URLItem.URL),
	)

	// Load plugin by slug - construct the path from the slug
	pluginPath := fmt.Sprintf("./plugins/%s.so", pluginSlug)

	// Load the shared object
	plug, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Look for the plugin constructor functions
	// Try extraction plugin first
	if newExtractionFunc, err := plug.Lookup("NewExtractionPlugin"); err == nil {
		// Call the constructor
		constructor, ok := newExtractionFunc.(func(*zap.Logger) (plugins.ExtractionPlugin, error))
		if !ok {
			return nil, fmt.Errorf("NewExtractionPlugin has invalid signature")
		}

		extractionPlugin, err := constructor(e.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction plugin: %w", err)
		}
		// Create extraction input
		extractionInput := &plugins.ExtractionInput{
			BrowserContext:   input.BrowserContext,
			URL:              input.URLItem.URL,
			URLItem:          input.URLItem,
			ExecutionContext: input.ExecutionContext,
			Config:           pluginConfig,
			ExecutionID:      input.ExecutionID,
		}

		// Execute extraction
		e.logger.Debug("Calling plugin Extract method",
			zap.String("plugin_slug", pluginSlug),
			zap.String("url", input.URLItem.URL),
		)

		output, err := extractionPlugin.Extract(ctx, extractionInput)
		if err != nil {
			e.logger.Error("Plugin extraction failed",
				zap.String("plugin_slug", pluginSlug),
				zap.Error(err),
			)
			return nil, fmt.Errorf("plugin extraction failed: %w", err)
		}

		e.logger.Info("Plugin extraction completed",
			zap.String("plugin_slug", pluginSlug),
			zap.String("schema", output.SchemaName),
			zap.Int("data_fields", len(output.Data)),
			zap.Any("data_keys", func() []string {
				keys := make([]string, 0, len(output.Data))
				for k := range output.Data {
					keys = append(keys, k)
				}
				return keys
			}()),
		)

		// Store extracted data in execution context and set marker
		// This is critical - the collectExtractedData function needs this marker
		extractedFieldNames := make([]string, 0, len(output.Data))
		for schemaKey, data := range output.Data {
			e.logger.Debug("Storing plugin data in context",
				zap.String("schema_key", schemaKey),
				zap.String("data_type", fmt.Sprintf("%T", data)),
			)
			// Store the data under the schema key
			input.ExecutionContext.Set(schemaKey, data)
			extractedFieldNames = append(extractedFieldNames, schemaKey)
		}

		// CRITICAL: Set the marker that tells collectExtractedData which fields to save
		input.ExecutionContext.Set("__extracted_fields__", extractedFieldNames)

		e.logger.Debug("Set extracted fields marker",
			zap.Strings("field_names", extractedFieldNames),
		)

		// Return result for node execution tracking
		var resultData interface{}
		if len(output.Data) > 0 {
			for _, data := range output.Data {
				resultData = data
				break
			}
		} else {
			e.logger.Warn("Plugin returned no data",
				zap.String("plugin_slug", pluginSlug),
			)
		}

		e.logger.Debug("Returning plugin execution output",
			zap.String("plugin_slug", pluginSlug),
			zap.Bool("has_result", resultData != nil),
			zap.Int("discovered_urls", len(output.DiscoveredURLs)),
		)

		return &ExecutionOutput{
			Result:         resultData,
			DiscoveredURLs: output.DiscoveredURLs,
		}, nil
	}

	// Try discovery plugin
	if newDiscoveryFunc, err := plug.Lookup("NewDiscoveryPlugin"); err == nil {
		constructor, ok := newDiscoveryFunc.(func(*zap.Logger) (plugins.DiscoveryPlugin, error))
		if !ok {
			return nil, fmt.Errorf("NewDiscoveryPlugin has invalid signature")
		}

		discoveryPlugin, err := constructor(e.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create discovery plugin: %w", err)
		}
		// Create discovery input
		discoveryInput := &plugins.DiscoveryInput{
			BrowserContext:   input.BrowserContext,
			URL:              input.URLItem.URL,
			URLItem:          input.URLItem,
			ExecutionContext: input.ExecutionContext,
			Config:           pluginConfig,
			ExecutionID:      input.ExecutionID,
		}

		// Execute discovery
		output, err := discoveryPlugin.Discover(ctx, discoveryInput)
		if err != nil {
			return nil, fmt.Errorf("plugin discovery failed: %w", err)
		}

		e.logger.Info("Plugin discovery completed",
			zap.String("plugin_slug", pluginSlug),
			zap.Int("urls_discovered", len(output.DiscoveredURLs)),
		)

		return &ExecutionOutput{
			Result:         output.Metadata,
			DiscoveredURLs: output.DiscoveredURLs,
		}, nil
	}

	return nil, fmt.Errorf("plugin does not export NewDiscoveryPlugin or NewExtractionPlugin function")
}
