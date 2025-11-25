package main

import (
	"context"
	"testing"

	"github.com/uzzalhcse/crawlify/pkg/models"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
)

func TestPluginInfo(t *testing.T) {
	plugin := NewDiscoveryPlugin()
	info := plugin.Info()

	if info.ID == "" {
		t.Error("Plugin ID should not be empty")
	}

	if info.Name == "" {
		t.Error("Plugin name should not be empty")
	}

	if info.PhaseType != models.PhaseTypeDiscovery {
		t.Errorf("Expected phase type %s, got %s", models.PhaseTypeDiscovery, info.PhaseType)
	}
}

func TestValidate(t *testing.T) {
	plugin := NewDiscoveryPlugin()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"link_selector": "a",
				"max_links":     100,
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  map[string]interface{}{},
			wantErr: false, // Should use defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSchema(t *testing.T) {
	plugin := NewDiscoveryPlugin()
	schema := plugin.ConfigSchema()

	if schema == nil {
		t.Error("ConfigSchema should not be nil")
	}

	if schema["type"] != "object" {
		t.Error("Schema type should be 'object'")
	}
}

// Note: Full integration tests would require a browser context
// This is just a basic structure test
func TestDiscoverStructure(t *testing.T) {
	plugin := NewDiscoveryPlugin()

	// Verify the plugin implements the interface correctly
	var _ plugins.DiscoveryPlugin = plugin

	// Test with nil input should handle gracefully
	_, err := plugin.Discover(context.Background(), nil)
	if err == nil {
		t.Log("Plugin should handle nil input gracefully")
	}
}
