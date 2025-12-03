package plugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// PluginScaffoldConfig holds configuration for creating a new plugin
type PluginScaffoldConfig struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	PhaseType   string   `json:"phase_type"`
	AuthorName  string   `json:"author_name"`
	AuthorEmail string   `json:"author_email"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// TemplateGenerator generates plugin scaffolding
type TemplateGenerator struct {
	BasePath string
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator(basePath string) *TemplateGenerator {
	return &TemplateGenerator{BasePath: basePath}
}

// GeneratePlugin creates a new plugin from template
func (g *TemplateGenerator) GeneratePlugin(config *PluginScaffoldConfig) (*models.Plugin, error) {
	// Validate config
	if config.Name == "" || config.Slug == "" || config.PhaseType == "" {
		return nil, fmt.Errorf("name, slug, and phase_type are required")
	}

	// Create plugin directory
	pluginDir := filepath.Join(g.BasePath, config.Slug)
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin directory already exists: %s", config.Slug)
	}

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Generate files based on template
	var pluginCode, goMod, readme string

	switch config.PhaseType {
	case "discovery":
		pluginCode = g.generateDiscoveryTemplate(config)
	case "extraction":
		pluginCode = g.generateExtractionTemplate(config)
	case "processing":
		pluginCode = g.generateProcessingTemplate(config)
	default:
		return nil, fmt.Errorf("invalid phase_type: %s", config.PhaseType)
	}

	goMod = g.generateGoMod(config)
	readme = g.generateReadme(config)

	// Write files
	files := map[string]string{
		"plugin.go": pluginCode,
		"go.mod":    goMod,
		"README.md": readme,
	}

	for filename, content := range files {
		filePath := filepath.Join(pluginDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Run go mod tidy to download dependencies
	if err := g.runGoModTidy(pluginDir); err != nil {
		// Log warning but don't fail - the plugin files are created
		fmt.Printf("Warning: failed to run go mod tidy: %v\n", err)
	}

	// Create plugin model for database
	plugin := &models.Plugin{
		ID:          uuid.New().String(),
		Name:        config.Name,
		Slug:        config.Slug,
		Description: config.Description,
		AuthorName:  config.AuthorName,
		AuthorEmail: config.AuthorEmail,
		PhaseType:   models.PhaseType(config.PhaseType),
		Category:    config.Category,
		Tags:        config.Tags,
		PluginType:  "community",
		IsVerified:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return plugin, nil
}

func (g *TemplateGenerator) generateDiscoveryTemplate(config *PluginScaffoldConfig) string {
	className := toPascalCase(config.Slug)
	return fmt.Sprintf(`package main

import (
	"context"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// Plugin metadata
const (
	PluginName        = "%s"
	PluginVersion     = "1.0.0"
	PluginDescription = "%s"
)

// %s implements the DiscoveryPlugin interface
type %s struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *%s) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: PluginDescription,
		Author:      "%s",
		PhaseType:   "discovery",
	}
}

// Discover performs URL discovery
func (p *%s) Discover(ctx context.Context, input *plugins.DiscoveryInput) (*plugins.DiscoveryOutput, error) {
	p.logger.Info("Starting discovery", zap.String("url", input.URL))

	// TODO: Implement your discovery logic here
	// Example: Find product links
	var discoveredURLs []string
	
	// Use input.BrowserContext to interact with the page
	// page := input.BrowserContext.Page
	
	return &plugins.DiscoveryOutput{
		DiscoveredURLs: discoveredURLs,
		URLMarkers:     make(map[string]string),
		Metadata:       make(map[string]interface{}),
	}, nil
}

// Validate checks if the configuration is valid
func (p *%s) Validate(config map[string]interface{}) error {
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration
func (p *%s) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// NewDiscoveryPlugin is the required exported function for plugin loading
func NewDiscoveryPlugin(logger *zap.Logger) (plugins.DiscoveryPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &%s{
		logger: logger,
	}, nil
}

// Ensure the plugin implements the interface at compile time
var _ plugins.DiscoveryPlugin = (*%s)(nil)
`, config.Name, config.Description, className, className, className, config.AuthorName, className, className, className, className, className)
}

func (g *TemplateGenerator) generateExtractionTemplate(config *PluginScaffoldConfig) string {
	className := toPascalCase(config.Slug)
	return fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// Plugin metadata
const (
	PluginName        = "%s"
	PluginVersion     = "1.0.0"
	PluginDescription = "%s"
)

// %s implements the ExtractionPlugin interface
type %s struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *%s) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: PluginDescription,
		Author:      "%s",
		PhaseType:   "extraction",
	}
}

// Extract performs data extraction
func (p *%s) Extract(ctx context.Context, input *plugins.ExtractionInput) (*plugins.ExtractionOutput, error) {
	p.logger.Info("Starting extraction", zap.String("url", input.URL))

	// TODO: Implement your extraction logic here
	data := make(map[string]interface{})
	
	// Use input.BrowserContext to interact with the page
	// page := input.BrowserContext.Page
	
	return &plugins.ExtractionOutput{
		Data:       map[string]interface{}{"item": data},
		SchemaName: "default",
	}, nil
}

// Validate checks if the configuration is valid
func (p *%s) Validate(config map[string]interface{}) error {
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration
func (p *%s) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// NewExtractionPlugin is the required exported function for plugin loading
func NewExtractionPlugin(logger *zap.Logger) (plugins.ExtractionPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &%s{
		logger: logger,
	}, nil
}

// Ensure the plugin implements the interface at compile time
var _ plugins.ExtractionPlugin = (*%s)(nil)
`, config.Name, config.Description, className, className, className, config.AuthorName, className, className, className, className, className)
}

func (g *TemplateGenerator) generateProcessingTemplate(config *PluginScaffoldConfig) string {
	className := toPascalCase(config.Slug)
	return fmt.Sprintf(`package main

import (
	"context"

	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// Plugin metadata
const (
	PluginName        = "%s"
	PluginVersion     = "1.0.0"
	PluginDescription = "%s"
)

// %s implements the ProcessingPlugin interface
type %s struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *%s) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: PluginDescription,
		Author:      "%s",
		PhaseType:   "processing",
	}
}

// Process performs data processing
func (p *%s) Process(ctx context.Context, input *plugins.ProcessingInput) (*plugins.ProcessingOutput, error) {
	p.logger.Info("Starting processing")

	// TODO: Implement your processing logic here
	processedData := input.Data
	
	return &plugins.ProcessingOutput{
		Data: processedData,
	}, nil
}

// Validate checks if the configuration is valid
func (p *%s) Validate(config map[string]interface{}) error {
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration
func (p *%s) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// NewProcessingPlugin is the required exported function for plugin loading
func NewProcessingPlugin(logger *zap.Logger) (plugins.ProcessingPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &%s{
		logger: logger,
	}, nil
}

// Ensure the plugin implements the interface at compile time
var _ plugins.ProcessingPlugin = (*%s)(nil)
`, config.Name, config.Description, className, className, className, config.AuthorName, className, className, className, className, className)
}

func (g *TemplateGenerator) generateGoMod(config *PluginScaffoldConfig) string {
	return fmt.Sprintf(`module github.com/crawlify/plugins/%s

go 1.21

require (
	github.com/uzzalhcse/crawlify v0.0.0
	go.uber.org/zap v1.27.0
)

replace github.com/uzzalhcse/crawlify => ../../..
`, config.Slug)
}

func (g *TemplateGenerator) generateReadme(config *PluginScaffoldConfig) string {
	return fmt.Sprintf(`# %s

%s

## Author

%s <%s>

## Phase Type

%s

## Usage

Add this plugin to your workflow configuration.

## Development

Edit the plugin.go file to customize the plugin behavior.
`, config.Name, config.Description, config.AuthorName, config.AuthorEmail, config.PhaseType)
}

// Helper function to convert slug to PascalCase
func toPascalCase(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// runGoModTidy runs go mod tidy in the plugin directory
func (g *TemplateGenerator) runGoModTidy(pluginDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = pluginDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}
