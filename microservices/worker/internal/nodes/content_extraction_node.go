package nodes

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ExtractContentNode extracts full page content for Universal Scraper
// Supports HTML, Markdown, and Screenshot output formats
type ExtractContentNode struct{}

func NewExtractContentNode() *ExtractContentNode {
	return &ExtractContentNode{}
}

func (n *ExtractContentNode) Type() string {
	return "extract_content"
}

func (n *ExtractContentNode) Execute(ctx context.Context, execCtx *ExecutionContext, node models.Node) error {
	// Get output format (default: html)
	outputFormat := "html"
	if format, ok := node.Params["output_format"].(string); ok && format != "" {
		outputFormat = format
	}

	logger.Info("Extracting page content",
		zap.String("format", outputFormat),
		zap.String("url", execCtx.Task.URL),
	)

	startTime := time.Now()
	var extractedData map[string]interface{}

	switch outputFormat {
	case "html":
		content, err := execCtx.Page.Content()
		if err != nil {
			return fmt.Errorf("failed to get HTML content: %w", err)
		}
		extractedData = map[string]interface{}{
			"type":         "html",
			"content":      content,
			"content_type": "text/html",
			"url":          execCtx.Task.URL,
			"timestamp":    time.Now().Format(time.RFC3339),
		}

	case "markdown":
		content, err := execCtx.Page.Content()
		if err != nil {
			return fmt.Errorf("failed to get HTML content for markdown: %w", err)
		}
		markdown := htmlToMarkdown(content)
		extractedData = map[string]interface{}{
			"type":         "markdown",
			"content":      markdown,
			"content_type": "text/markdown",
			"url":          execCtx.Task.URL,
			"timestamp":    time.Now().Format(time.RFC3339),
		}

	case "screenshot":
		screenshot, err := execCtx.Page.Screenshot()
		if err != nil {
			return fmt.Errorf("failed to capture screenshot: %w", err)
		}
		extractedData = map[string]interface{}{
			"type":         "screenshot",
			"screenshot":   base64.StdEncoding.EncodeToString(screenshot),
			"content_type": "image/png",
			"url":          execCtx.Task.URL,
			"timestamp":    time.Now().Format(time.RFC3339),
		}

	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	// Store in ExtractedItems for TaskExecutor to process
	execCtx.ExtractedItems = append(execCtx.ExtractedItems, extractedData)

	duration := time.Since(startTime)
	logger.Info("Page content extracted",
		zap.String("format", outputFormat),
		zap.Duration("duration", duration),
	)

	return nil
}

// htmlToMarkdown converts HTML to Markdown format (Zenrows-style clean output)
// Removes: scripts, styles, nav, footer, ads - keeps only main content
func htmlToMarkdown(html string) string {
	result := html

	// Remove scripts, styles, and head section
	result = removeTagContent(result, "script")
	result = removeTagContent(result, "style")
	result = removeTagContent(result, "head")

	// Remove boilerplate/navigation elements (Zenrows-style)
	result = removeTagContent(result, "nav")
	result = removeTagContent(result, "footer")
	result = removeTagContent(result, "header")
	result = removeTagContent(result, "aside")
	result = removeTagContent(result, "form")
	result = removeTagContent(result, "iframe")
	result = removeTagContent(result, "noscript")
	result = removeTagContent(result, "svg")

	// Convert headings
	for i := 6; i >= 1; i-- {
		prefix := strings.Repeat("#", i) + " "
		result = replaceTag(result, fmt.Sprintf("h%d", i), prefix, "\n\n")
	}

	// Convert links: <a href="url">text</a> -> [text](url)
	linkRegex := regexp.MustCompile(`<a[^>]*href=["']([^"']*)["'][^>]*>(.*?)</a>`)
	result = linkRegex.ReplaceAllString(result, "[$2]($1)")

	// Convert images: <img src="url" alt="text"> -> ![text](url)
	imgRegex := regexp.MustCompile(`<img[^>]*src=["']([^"']*)["'][^>]*alt=["']([^"']*)["'][^>]*/?>`)
	result = imgRegex.ReplaceAllString(result, "![$2]($1)")
	imgRegex2 := regexp.MustCompile(`<img[^>]*alt=["']([^"']*)["'][^>]*src=["']([^"']*)["'][^>]*/?>`)
	result = imgRegex2.ReplaceAllString(result, "![$1]($2)")

	// Convert bold
	result = replaceTag(result, "strong", "**", "**")
	result = replaceTag(result, "b", "**", "**")

	// Convert italic
	result = replaceTag(result, "em", "*", "*")
	result = replaceTag(result, "i", "*", "*")

	// Convert code
	result = replaceTag(result, "code", "`", "`")

	// Convert pre/code blocks
	preRegex := regexp.MustCompile(`<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	result = preRegex.ReplaceAllString(result, "\n```\n$1\n```\n")
	result = replaceTag(result, "pre", "\n```\n", "\n```\n")

	// Convert lists
	ulRegex := regexp.MustCompile(`<li[^>]*>(.*?)</li>`)
	result = ulRegex.ReplaceAllString(result, "- $1\n")
	result = replaceTag(result, "ul", "\n", "\n")
	result = replaceTag(result, "ol", "\n", "\n")

	// Convert paragraphs and divs
	result = replaceTag(result, "p", "", "\n\n")
	result = replaceTag(result, "div", "", "\n")
	result = replaceTag(result, "br", "", "\n")

	// Remove remaining HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	result = tagRegex.ReplaceAllString(result, "")

	// Decode common HTML entities
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&apos;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")

	// Clean up whitespace
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	result = multipleNewlines.ReplaceAllString(result, "\n\n")
	result = strings.TrimSpace(result)

	return result
}

// removeTagContent removes a tag and its content (including multi-line)
func removeTagContent(html, tag string) string {
	// (?s) makes . match newlines for multi-line content like <script>...</script>
	regex := regexp.MustCompile(fmt.Sprintf(`(?si)<%s[^>]*>.*?</%s>`, tag, tag))
	return regex.ReplaceAllString(html, "")
}

// replaceTag replaces an HTML tag with markdown equivalents
func replaceTag(html, tag, prefix, suffix string) string {
	// Handle self-closing and regular tags
	openRegex := regexp.MustCompile(fmt.Sprintf(`<%s[^>]*>`, tag))
	closeRegex := regexp.MustCompile(fmt.Sprintf(`</%s>`, tag))

	result := openRegex.ReplaceAllString(html, prefix)
	result = closeRegex.ReplaceAllString(result, suffix)
	return result
}
