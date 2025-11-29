package browser

import (
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

// GetResponseStatus returns the HTTP status code of the last navigation
func (bc *BrowserContext) GetResponseStatus() (int, error) {
	if bc.lastResponse == nil {
		return 0, fmt.Errorf("no response available")
	}
	return bc.lastResponse.Status(), nil
}

// GetResponseHeaders returns the headers of the last navigation
func (bc *BrowserContext) GetResponseHeaders() (map[string]string, error) {
	if bc.lastResponse == nil {
		return nil, fmt.Errorf("no response available")
	}

	headers := make(map[string]string)
	headersArray, err := bc.lastResponse.HeadersArray()
	if err != nil {
		return nil, err
	}

	for _, header := range headersArray {
		headers[header.Name] = header.Value
	}
	return headers, nil
}

// GetPageBody returns the current page content
func (bc *BrowserContext) GetPageBody() (string, error) {
	return bc.Page.Content()
}

// CheckHTTPStatus checks if the last navigation resulted in an HTTP error status (4xx/5xx)
// Returns an error if status >= 400, nil otherwise
func (bc *BrowserContext) CheckHTTPStatus() error {
	statusCode, err := bc.GetResponseStatus()
	if err != nil {
		// No response means no error to check
		return nil
	}

	if statusCode >= 400 {
		logger.Warn("⚠️ HTTP error status detected",
			zap.Int("status_code", statusCode),
			zap.String("url", bc.lastResponse.URL()))

		switch {
		case statusCode == 429:
			return fmt.Errorf("rate limit exceeded (HTTP 429)")
		case statusCode >= 500:
			return fmt.Errorf("server error (HTTP %d)", statusCode)
		case statusCode == 403:
			return fmt.Errorf("forbidden (HTTP 403)")
		case statusCode == 401:
			return fmt.Errorf("unauthorized (HTTP 401)")
		case statusCode >= 400:
			return fmt.Errorf("client error (HTTP %d)", statusCode)
		}
	}

	logger.Debug("✅ HTTP status OK", zap.Int("status_code", statusCode))
	return nil
}
