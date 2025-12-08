package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// VisualSelectorService manages browser sessions for visual selector
type VisualSelectorService struct {
	pw               *playwright.Playwright
	selectFlowScript string
	sessions         sync.Map // sessionID -> *BrowserSession
	callbackBaseURL  string   // Base URL for SelectFlow callbacks (e.g., http://localhost:8080/api/v1)
}

// BrowserSession represents an active browser session
type BrowserSession struct {
	Browser   playwright.Browser
	Page      playwright.Page
	Fields    map[string]interface{} // Collected fields from SelectFlow
	Completed bool                   // True when user clicked Done
}

// NewVisualSelectorService creates a new visual selector service
func NewVisualSelectorService(selectFlowScriptPath string, callbackBaseURL string) (*VisualSelectorService, error) {
	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	// Load SelectFlow script
	var selectFlowScript string
	if selectFlowScriptPath != "" {
		scriptBytes, err := os.ReadFile(selectFlowScriptPath)
		if err != nil {
			logger.Warn("Failed to load SelectFlow script, visual selector will not work",
				zap.String("path", selectFlowScriptPath),
				zap.Error(err),
			)
		} else {
			selectFlowScript = string(scriptBytes)
			logger.Info("Loaded SelectFlow script",
				zap.String("path", selectFlowScriptPath),
				zap.Int("bytes", len(selectFlowScript)),
			)
		}
	}

	return &VisualSelectorService{
		pw:               pw,
		selectFlowScript: selectFlowScript,
		callbackBaseURL:  callbackBaseURL,
	}, nil
}

// LaunchSession opens a non-headless browser and injects SelectFlow
func (s *VisualSelectorService) LaunchSession(sessionID, url string, existingFields map[string]interface{}) error {
	if s.selectFlowScript == "" {
		return fmt.Errorf("SelectFlow script not loaded")
	}

	// Launch browser in non-headless mode (visible to user) with fullscreen
	browser, err := s.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--start-maximized",
			"--start-fullscreen",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Create context with no viewport restrictions (allows fullscreen)
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		NoViewport:        playwright.Bool(true), // Allow browser to use full window size
		IgnoreHttpsErrors: playwright.Bool(true),
	})
	if err != nil {
		browser.Close()
		return fmt.Errorf("failed to create context: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		browser.Close()
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Store session BEFORE navigation so we can update it from callback
	s.sessions.Store(sessionID, &BrowserSession{
		Browser: browser,
		Page:    page,
	})

	// Expose a function that SelectFlow can call directly (no HTTP needed - avoids HTTPS mixed content issues)
	err = page.ExposeFunction("__selectFlowSaveFields", func(args ...interface{}) interface{} {
		if len(args) == 0 {
			return "error: no arguments"
		}
		fieldsJSON, ok := args[0].(string)
		if !ok {
			return "error: invalid argument type"
		}

		// Parse fields
		var fields map[string]interface{}
		if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
			logger.Error("Failed to parse fields JSON", zap.Error(err))
			return "error: " + err.Error()
		}

		// Store fields in session - handler will read this
		if sessionData, ok := s.sessions.Load(sessionID); ok {
			session := sessionData.(*BrowserSession)
			session.Fields = fields
			session.Completed = true
			s.sessions.Store(sessionID, session)
		}

		logger.Info("SelectFlow fields received via exposed function",
			zap.String("session_id", sessionID),
			zap.Int("field_count", len(fields)),
		)
		return "ok"
	})
	if err != nil {
		logger.Warn("Failed to expose save function", zap.Error(err))
	}

	// Expose function to close the browser (called by countdown timer)
	err = page.ExposeFunction("__selectFlowCloseBrowser", func(args ...interface{}) interface{} {
		logger.Info("Browser close requested via exposed function",
			zap.String("session_id", sessionID),
		)
		// Close browser in a goroutine to avoid blocking
		go func() {
			time.Sleep(500 * time.Millisecond) // Small delay for UX
			s.CloseSession(sessionID)
		}()
		return "closing"
	})
	if err != nil {
		logger.Warn("Failed to expose close function", zap.Error(err))
	}

	// Navigate to URL (no timeout - user controls when they're done)
	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60000), // 60 seconds for initial load
	}); err != nil {
		s.CloseSession(sessionID)
		return fmt.Errorf("failed to navigate to URL: %w", err)
	}

	// Wait a bit for page to settle
	time.Sleep(500 * time.Millisecond)

	// Build existing fields JSON for pre-population
	existingFieldsJSON := "{}"
	if len(existingFields) > 0 {
		if jsonBytes, err := json.Marshal(existingFields); err == nil {
			existingFieldsJSON = string(jsonBytes)
		}
	}

	// Inject callback that uses exposed function (works on HTTPS pages!)
	callbackScript := fmt.Sprintf(`
		// Store callback info for SelectFlow - uses Playwright exposed function (no HTTP needed)
		window.__selectFlowCallback = async function(fields) {
			try {
				// Call the Playwright-exposed function directly
				await window.__selectFlowSaveFields(JSON.stringify(fields));
				
				console.log('[SelectFlow] Fields saved successfully via Playwright bridge');
				
				const fieldCount = Object.keys(fields).length;
				
				// Create professional overlay
				const overlay = document.createElement('div');
				overlay.setAttribute('data-selectflow-ui', 'overlay');
				overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,0.85);z-index:99999999;display:flex;flex-direction:column;align-items:center;justify-content:center;font-family:-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,sans-serif;';
				
				// Success icon
				const icon = document.createElement('div');
				icon.innerHTML = '✓';
				icon.style.cssText = 'width:80px;height:80px;border-radius:50%%;background:linear-gradient(135deg,#10b981,#059669);color:white;font-size:40px;display:flex;align-items:center;justify-content:center;margin-bottom:24px;box-shadow:0 10px 40px rgba(16,185,129,0.4);animation:pulse 1s ease-in-out infinite;';
				overlay.appendChild(icon);
				
				// Title
				const title = document.createElement('h1');
				title.textContent = 'Fields Saved Successfully!';
				title.style.cssText = 'color:white;font-size:28px;font-weight:600;margin:0 0 8px 0;';
				overlay.appendChild(title);
				
				// Field count
				const subtitle = document.createElement('p');
				subtitle.textContent = fieldCount + ' field' + (fieldCount !== 1 ? 's' : '') + ' collected';
				subtitle.style.cssText = 'color:rgba(255,255,255,0.7);font-size:16px;margin:0 0 32px 0;';
				overlay.appendChild(subtitle);
				
				// Progress container
				const progressContainer = document.createElement('div');
				progressContainer.style.cssText = 'width:300px;height:6px;background:rgba(255,255,255,0.2);border-radius:3px;overflow:hidden;margin-bottom:16px;';
				overlay.appendChild(progressContainer);
				
				// Progress bar
				const progressBar = document.createElement('div');
				progressBar.style.cssText = 'width:100%%;height:100%%;background:linear-gradient(90deg,#10b981,#34d399);border-radius:3px;transition:width 1s linear;';
				progressContainer.appendChild(progressBar);
				
				// Countdown text
				const countdown = document.createElement('p');
				countdown.textContent = 'Closing in 10 seconds...';
				countdown.style.cssText = 'color:rgba(255,255,255,0.5);font-size:14px;margin:0;';
				overlay.appendChild(countdown);
				
				// Add keyframe animation
				const style = document.createElement('style');
				style.textContent = '@keyframes pulse{0%%,100%%{transform:scale(1)}50%%{transform:scale(1.05)}}';
				document.head.appendChild(style);
				
				document.body.appendChild(overlay);
				
				// Animate countdown
				let seconds = 10;
				progressBar.style.width = '100%%';
				
				const timer = setInterval(async function() {
					seconds--;
					const pct = (seconds / 10) * 100;
					progressBar.style.width = pct + '%%';
					
					if (seconds <= 0) {
						clearInterval(timer);
						countdown.textContent = 'Closing...';
						await window.__selectFlowCloseBrowser();
					} else {
						countdown.textContent = 'Closing in ' + seconds + ' second' + (seconds !== 1 ? 's' : '') + '...';
					}
				}, 1000);
				
			} catch(err) {
				console.error('[SelectFlow] Error saving fields:', err);
				alert('Error saving fields: ' + err.message);
			}
		};
		
		// Store existing fields for pre-population
		window.__selectFlowExistingFields = %s;
		
		console.log('[SelectFlow] Callback configured for session: %s (using Playwright bridge)');
	`, existingFieldsJSON, sessionID)

	// Inject callback configuration
	if _, err := page.Evaluate(callbackScript); err != nil {
		logger.Warn("Failed to inject callback script", zap.Error(err))
	}

	// Inject SelectFlow script
	if _, err := page.Evaluate(s.selectFlowScript); err != nil {
		logger.Warn("Failed to inject SelectFlow script", zap.Error(err))
		s.CloseSession(sessionID)
		return fmt.Errorf("failed to inject SelectFlow: %w", err)
	}

	logger.Info("Visual selector session launched",
		zap.String("session_id", sessionID),
		zap.String("url", url),
	)

	return nil
}

// CloseSession closes a browser session
func (s *VisualSelectorService) CloseSession(sessionID string) {
	if sessionData, ok := s.sessions.Load(sessionID); ok {
		session := sessionData.(*BrowserSession)
		if err := session.Browser.Close(); err != nil {
			logger.Warn("Failed to close browser",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
		}
		s.sessions.Delete(sessionID)
		logger.Info("Closed visual selector session", zap.String("session_id", sessionID))
	}
}

// Close shuts down the Playwright instance
func (s *VisualSelectorService) Close() error {
	// Close all active sessions
	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*BrowserSession)
		session.Browser.Close()
		return true
	})

	if s.pw != nil {
		return s.pw.Stop()
	}
	return nil
}

// GetSessionFields returns the collected fields if the session is completed
// This is used by the handler to poll for completion
func (s *VisualSelectorService) GetSessionFields(sessionID string) (map[string]interface{}, bool) {
	if sessionData, ok := s.sessions.Load(sessionID); ok {
		session := sessionData.(*BrowserSession)
		if session.Completed && session.Fields != nil {
			return session.Fields, true
		}
	}
	return nil, false
}
