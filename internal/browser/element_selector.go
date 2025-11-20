package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

// SelectorSession represents an active element selection session
type SelectorSession struct {
	ID             string
	URL            string
	BrowserContext *BrowserContext
	SelectedFields []SelectedField
	CreatedAt      time.Time
	LastActivity   time.Time
	mu             sync.RWMutex
}

// SelectedField represents a field selected by the user
type SelectedField struct {
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Type      string `json:"type"` // text, attribute, html
	Attribute string `json:"attribute,omitempty"`
	Multiple  bool   `json:"multiple"`
	XPath     string `json:"xpath,omitempty"`
	Preview   string `json:"preview"`
}

// ElementSelectorManager manages selector sessions
type ElementSelectorManager struct {
	pool     *BrowserPool
	sessions map[string]*SelectorSession
	mu       sync.RWMutex
}

// NewElementSelectorManager creates a new element selector manager
func NewElementSelectorManager(pool *BrowserPool) *ElementSelectorManager {
	manager := &ElementSelectorManager{
		pool:     pool,
		sessions: make(map[string]*SelectorSession),
	}

	// Start cleanup goroutine for inactive sessions
	go manager.cleanupInactiveSessions()

	return manager
}

// CreateSession creates a new selector session and launches a browser
func (m *ElementSelectorManager) CreateSession(ctx context.Context, url string) (*SelectorSession, error) {
	sessionID := uuid.New().String()

	logger.Info("Creating selector session",
		zap.String("session_id", sessionID),
		zap.String("url", url),
	)

	// Acquire browser context with headed mode
	browserCtx, err := m.pool.Acquire(ctx, true) // Pass true for headed mode
	if err != nil {
		return nil, fmt.Errorf("failed to acquire browser: %w", err)
	}

	// Navigate to the URL
	_, err = browserCtx.Navigate(url)
	if err != nil {
		m.pool.Release(browserCtx)
		return nil, fmt.Errorf("failed to navigate to URL: %w", err)
	}

	// Wait for page to load
	if err := browserCtx.Page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		logger.Warn("Page load state warning", zap.Error(err))
	}

	// Inject the selector overlay UI
	if err := m.injectSelectorOverlay(browserCtx); err != nil {
		m.pool.Release(browserCtx)
		return nil, fmt.Errorf("failed to inject selector overlay: %w", err)
	}

	session := &SelectorSession{
		ID:             sessionID,
		URL:            url,
		BrowserContext: browserCtx,
		SelectedFields: make([]SelectedField, 0),
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// Start listening for selection events
	go m.listenForSelections(session)

	return session, nil
}

// GetSession retrieves a session by ID
func (m *ElementSelectorManager) GetSession(sessionID string) (*SelectorSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastActivity = time.Now()
	return session, nil
}

// GetSelectedFields returns the fields selected in a session
func (m *ElementSelectorManager) GetSelectedFields(sessionID string) ([]SelectedField, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	return session.SelectedFields, nil
}

// CloseSession closes a selector session and releases the browser
func (m *ElementSelectorManager) CloseSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	logger.Info("Closing selector session", zap.String("session_id", sessionID))

	// Release browser context
	if session.BrowserContext != nil {
		m.pool.Release(session.BrowserContext)
	}

	delete(m.sessions, sessionID)
	return nil
}

// injectSelectorOverlay injects the Vue.js-based selector UI into the page
func (m *ElementSelectorManager) injectSelectorOverlay(browserCtx *BrowserContext) error {
	// Inject Vue 3 from CDN
	_, err := browserCtx.Page.Evaluate(`
		// Add Vue 3 from CDN
		const vueScript = document.createElement('script');
		vueScript.src = 'https://unpkg.com/vue@3/dist/vue.global.js';
		document.head.appendChild(vueScript);
		
		// Wait for Vue to load
		new Promise((resolve) => {
			vueScript.onload = resolve;
		});
	`)
	if err != nil {
		return fmt.Errorf("failed to inject Vue.js: %w", err)
	}

	// Wait a bit for Vue to load
	time.Sleep(1 * time.Second)

	// Inject the selector overlay application
	overlayJS := m.getSelectorOverlayJS()
	_, err = browserCtx.Page.Evaluate(overlayJS)
	if err != nil {
		return fmt.Errorf("failed to inject selector overlay: %w", err)
	}

	return nil
}

// listenForSelections listens for element selection events from the injected UI
func (m *ElementSelectorManager) listenForSelections(session *SelectorSession) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if session still exists
			m.mu.RLock()
			_, exists := m.sessions[session.ID]
			m.mu.RUnlock()

			if !exists {
				return
			}

			// Poll for selections from the page
			result, err := session.BrowserContext.Page.Evaluate(`
				window.__crawlifyGetSelections ? window.__crawlifyGetSelections() : null
			`)

			if err != nil || result == nil {
				continue
			}

			// Parse the selections
			var selections []SelectedField
			jsonData, err := json.Marshal(result)
			if err != nil {
				continue
			}

			if err := json.Unmarshal(jsonData, &selections); err != nil {
				continue
			}

			// Update session with new selections
			session.mu.Lock()
			session.SelectedFields = selections
			session.mu.Unlock()
		}
	}
}

// cleanupInactiveSessions removes sessions that have been inactive for too long
func (m *ElementSelectorManager) cleanupInactiveSessions() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for id, session := range m.sessions {
			if time.Since(session.LastActivity) > 30*time.Minute {
				logger.Info("Cleaning up inactive session", zap.String("session_id", id))
				if session.BrowserContext != nil {
					m.pool.Release(session.BrowserContext)
				}
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
	}
}

// getSelectorOverlayJS returns the JavaScript code for the selector overlay
func (m *ElementSelectorManager) getSelectorOverlayJS() string {
	return getSelectorOverlayJS()
}

// GenerateOptimalSelector generates the best CSS selector for an element
func GenerateOptimalSelector(element playwright.ElementHandle) (string, error) {
	// This is a placeholder - we'll implement smart selector generation
	// For now, return a basic approach
	return "", fmt.Errorf("not implemented yet")
}
