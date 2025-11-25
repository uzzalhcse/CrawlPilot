package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/uzzalhcse/crawlify/pkg/plugins"
	"go.uber.org/zap"
)

// Plugin metadata
const (
	PluginName        = "Aqua Product Extractor"
	PluginVersion     = "1.0.0"
	PluginDescription = "Extracts comprehensive product details from Aqua e-commerce product pages"
)

// AquaProductExtractor implements the ExtractionPlugin interface
type AquaProductExtractor struct {
	logger *zap.Logger
}

// Info returns plugin metadata
func (p *AquaProductExtractor) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: PluginDescription,
		Author:      "Crawlify Team",
		PhaseType:   "extraction",
	}
}

// Extract performs data extraction from Aqua product pages
func (p *AquaProductExtractor) Extract(ctx context.Context, input *plugins.ExtractionInput) (*plugins.ExtractionOutput, error) {
	p.logger.Info("Starting Aqua product extraction",
		zap.String("url", input.URL),
	)

	// Get page HTML from browser context
	pageHTML, err := input.BrowserContext.Page.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to get page content: %w", err)
	}

	// Get schema from config with default
	schema := "aqua_product"
	if schemaVal, ok := input.Config["schema"].(string); ok && schemaVal != "" {
		schema = schemaVal
	}

	// Parse the page HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract product data
	productData := make(map[string]interface{})

	// Page title
	productData["page_title"] = strings.TrimSpace(doc.Find("title").Text())

	// Product name
	productName := doc.Find(".ProductInfo_Head_Main_ProductName").Text()
	productData["product_name"] = strings.TrimSpace(productName)

	// Images - extract all product images
	images := []string{}
	doc.Find(".productImageList img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if strings.HasPrefix(src, "//") {
				src = "https:" + src
			} else if !strings.HasPrefix(src, "http") {
				src = "https://aqua-has.com" + src
			}
			images = append(images, src)
		}
	})
	productData["images"] = images

	// Maker/Brand
	maker := "AQUA"
	if metaContent, exists := doc.Find("meta[property='og:site_name']").Attr("content"); exists {
		maker = strings.TrimSpace(metaContent)
	}
	productData["maker"] = maker
	productData["brand"] = maker

	// Category
	category := doc.Find(".ProductDetail_Section_Headline_Sub").Text()
	productData["category"] = strings.TrimSpace(category)

	// Description
	description := doc.Find("div.ProductDetail_Section_Text_Group").Text()
	productData["description"] = strings.TrimSpace(description)

	// Price (Open Price for Aqua products)
	priceText := doc.Find(".ProductInfo_PriceArea_OpenPrice").Text()
	if priceText == "" {
		priceText = "Open Price"
	}
	productData["list_price"] = strings.TrimSpace(priceText)

	// Specifications/Attributes - extract key-value pairs
	attributes := make(map[string]interface{})
	doc.Find(".ProductDetail_Section_Spec_Item").Each(func(i int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find(".ProductDetail_Section_Spec_Item_Head").Text())
		value := strings.TrimSpace(s.Find(".ProductDetail_Section_Spec_Item_Body").Text())
		if key != "" {
			attributes[key] = value
		}
	})
	productData["attributes"] = attributes

	// Additional metadata
	productData["url"] = input.URL

	// Extract model number from product name if present
	if modelNum := extractModelNumber(productName); modelNum != "" {
		productData["model_number"] = modelNum
	}

	p.logger.Info("Successfully extracted Aqua product data",
		zap.String("product_name", productData["product_name"].(string)),
		zap.Int("image_count", len(images)),
		zap.Int("attribute_count", len(attributes)),
	)

	return &plugins.ExtractionOutput{
		Data: map[string]interface{}{
			schema: productData,
		},
		SchemaName: schema,
	}, nil
}

// Validate checks if the configuration is valid
func (p *AquaProductExtractor) Validate(config map[string]interface{}) error {
	// All configuration is optional for this plugin
	return nil
}

// ConfigSchema returns the JSON schema for plugin configuration
func (p *AquaProductExtractor) ConfigSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"schema": map[string]interface{}{
				"type":        "string",
				"description": "Schema name for extracted data",
				"default":     "aqua_product",
			},
		},
	}
}

// Helper function to extract model number from product name
func extractModelNumber(productName string) string {
	// Match patterns like "AQR-TZ42M", "AQW-GV100M", etc.
	re := regexp.MustCompile(`[A-Z]{3}-[A-Z0-9]+`)
	if match := re.FindString(productName); match != "" {
		return match
	}
	return ""
}

// NewExtractionPlugin is the required exported function for plugin loading
func NewExtractionPlugin(logger *zap.Logger) (plugins.ExtractionPlugin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AquaProductExtractor{
		logger: logger,
	}, nil
}

// Ensure the plugin implements the interface at compile time
var _ plugins.ExtractionPlugin = (*AquaProductExtractor)(nil)
