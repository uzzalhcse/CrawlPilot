package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// IncidentReporter handles creating and storing incident reports
// when automated recovery fails and human intervention is needed
type IncidentReporter struct {
	pool *pgxpool.Pool
}

// IncidentReport contains all details for human investigation
type IncidentReport struct {
	ID          string `json:"id"`
	ExecutionID string `json:"execution_id"`
	TaskID      string `json:"task_id"`
	WorkflowID  string `json:"workflow_id"`
	URL         string `json:"url"`
	Domain      string `json:"domain"`

	// Error Details
	ErrorPattern string `json:"error_pattern"`
	ErrorMessage string `json:"error_message"`
	StatusCode   int    `json:"status_code,omitempty"`

	// Recovery History - What we tried
	RecoveryAttempts []RecoveryAttemptSummary `json:"recovery_attempts"`
	TotalAttempts    int                      `json:"total_attempts"`

	// AI Agent Details
	AIEnabled       bool   `json:"ai_enabled"`
	AIProvider      string `json:"ai_provider,omitempty"`
	AIReasoning     string `json:"ai_reasoning,omitempty"`
	AIFailureReason string `json:"ai_failure_reason,omitempty"`

	// Snapshots for Investigation
	Screenshot  string `json:"screenshot,omitempty"`   // Base64 or GCS path
	DOMSnapshot string `json:"dom_snapshot,omitempty"` // Full HTML
	PageTitle   string `json:"page_title,omitempty"`
	PageURL     string `json:"page_url,omitempty"` // Final URL after redirects

	// Context
	BrowserProfile  string                 `json:"browser_profile,omitempty"`
	ProxyUsed       string                 `json:"proxy_used,omitempty"`
	Cookies         map[string]interface{} `json:"cookies,omitempty"`
	RequestHeaders  map[string]string      `json:"request_headers,omitempty"`
	ResponseHeaders map[string]string      `json:"response_headers,omitempty"`

	// Suggested Actions for Human
	SuggestedActions []string `json:"suggested_actions"`

	// Status Tracking
	Status     IncidentStatus   `json:"status"`
	Priority   IncidentPriority `json:"priority"`
	AssignedTo string           `json:"assigned_to,omitempty"`
	Resolution string           `json:"resolution,omitempty"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`

	// Timestamps
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	FirstErrorAt time.Time `json:"first_error_at"`
	LastErrorAt  time.Time `json:"last_error_at"`
}

// RecoveryAttemptSummary summarizes each recovery attempt
type RecoveryAttemptSummary struct {
	Attempt     int       `json:"attempt"`
	Action      string    `json:"action"`
	Source      string    `json:"source"` // rule, ai, default
	Reason      string    `json:"reason"`
	Success     bool      `json:"success"`
	Duration    string    `json:"duration"`
	Timestamp   time.Time `json:"timestamp"`
	RuleID      string    `json:"rule_id,omitempty"`
	AIReasoning string    `json:"ai_reasoning,omitempty"`
}

// IncidentStatus represents the status of an incident
type IncidentStatus string

const (
	IncidentStatusOpen       IncidentStatus = "open"
	IncidentStatusInProgress IncidentStatus = "in_progress"
	IncidentStatusResolved   IncidentStatus = "resolved"
	IncidentStatusIgnored    IncidentStatus = "ignored"
	IncidentStatusRecurring  IncidentStatus = "recurring"
)

// IncidentPriority represents the priority of an incident
type IncidentPriority string

const (
	PriorityCritical IncidentPriority = "critical" // Captcha, Auth - needs immediate attention
	PriorityHigh     IncidentPriority = "high"     // AI exhausted all options
	PriorityMedium   IncidentPriority = "medium"   // Multiple failures
	PriorityLow      IncidentPriority = "low"      // Single failure, might recover
)

// NewIncidentReporter creates a new incident reporter
func NewIncidentReporter(pool *pgxpool.Pool) *IncidentReporter {
	return &IncidentReporter{pool: pool}
}

// CreateIncident creates a new incident report
func (r *IncidentReporter) CreateIncident(ctx context.Context, incident *IncidentReport) error {
	if incident.ID == "" {
		incident.ID = uuid.New().String()
	}
	incident.CreatedAt = time.Now()
	incident.UpdatedAt = time.Now()
	if incident.Status == "" {
		incident.Status = IncidentStatusOpen
	}

	// Calculate priority based on pattern and attempts
	incident.Priority = r.calculatePriority(incident)

	// Generate suggested actions
	incident.SuggestedActions = r.generateSuggestedActions(incident)

	attemptsJSON, _ := json.Marshal(incident.RecoveryAttempts)
	suggestedJSON, _ := json.Marshal(incident.SuggestedActions)
	cookiesJSON, _ := json.Marshal(incident.Cookies)
	reqHeadersJSON, _ := json.Marshal(incident.RequestHeaders)
	respHeadersJSON, _ := json.Marshal(incident.ResponseHeaders)

	query := `
		INSERT INTO incidents (
			id, execution_id, task_id, workflow_id, url, domain,
			error_pattern, error_message, status_code,
			recovery_attempts, total_attempts,
			ai_enabled, ai_provider, ai_reasoning, ai_failure_reason,
			screenshot, dom_snapshot, page_title, page_url,
			browser_profile, proxy_used, cookies, request_headers, response_headers,
			suggested_actions, status, priority,
			first_error_at, last_error_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22, $23, $24,
			$25, $26, $27,
			$28, $29, $30, $31
		)`

	_, err := r.pool.Exec(ctx, query,
		incident.ID, incident.ExecutionID, incident.TaskID, incident.WorkflowID, incident.URL, incident.Domain,
		incident.ErrorPattern, incident.ErrorMessage, incident.StatusCode,
		attemptsJSON, incident.TotalAttempts,
		incident.AIEnabled, incident.AIProvider, incident.AIReasoning, incident.AIFailureReason,
		incident.Screenshot, incident.DOMSnapshot, incident.PageTitle, incident.PageURL,
		incident.BrowserProfile, incident.ProxyUsed, cookiesJSON, reqHeadersJSON, respHeadersJSON,
		suggestedJSON, incident.Status, incident.Priority,
		incident.FirstErrorAt, incident.LastErrorAt, incident.CreatedAt, incident.UpdatedAt,
	)

	if err != nil {
		logger.Error("Failed to create incident report", zap.Error(err))
		return err
	}

	logger.Info("Incident report created",
		zap.String("incident_id", incident.ID),
		zap.String("url", incident.URL),
		zap.String("pattern", incident.ErrorPattern),
		zap.String("priority", string(incident.Priority)),
		zap.Int("attempts", incident.TotalAttempts),
	)

	return nil
}

// CreateFromRecoveryFailure creates an incident from recovery manager context
func (r *IncidentReporter) CreateFromRecoveryFailure(
	ctx context.Context,
	taskID, executionID, workflowID, url string,
	detected *DetectedError,
	attempts []*RecoveryAttempt,
	aiReasoning, aiFailure string,
	snapshot *PageSnapshot,
) (*IncidentReport, error) {
	// Convert attempts to summary
	attemptSummaries := make([]RecoveryAttemptSummary, 0, len(attempts))
	for i, a := range attempts {
		summary := RecoveryAttemptSummary{
			Attempt:   i + 1,
			Action:    string(a.Plan.Action),
			Source:    a.Plan.Source,
			Reason:    a.Plan.Reason,
			Success:   a.Success,
			Timestamp: a.Timestamp,
		}
		if a.Plan.RuleID != "" {
			summary.RuleID = a.Plan.RuleID
		}
		attemptSummaries = append(attemptSummaries, summary)
	}

	incident := &IncidentReport{
		ExecutionID:      executionID,
		TaskID:           taskID,
		WorkflowID:       workflowID,
		URL:              url,
		Domain:           detected.Domain,
		ErrorPattern:     string(detected.Pattern),
		ErrorMessage:     detected.RawError,
		StatusCode:       detected.StatusCode,
		RecoveryAttempts: attemptSummaries,
		TotalAttempts:    len(attempts),
		AIEnabled:        true,
		AIReasoning:      aiReasoning,
		AIFailureReason:  aiFailure,
		FirstErrorAt:     time.Now(),
		LastErrorAt:      time.Now(),
	}

	// Add snapshot if available
	if snapshot != nil {
		incident.Screenshot = snapshot.ScreenshotPath
		incident.DOMSnapshot = snapshot.DOM
		incident.PageTitle = snapshot.Title
		incident.PageURL = snapshot.FinalURL
	}

	// Set first error time from attempts
	if len(attempts) > 0 {
		incident.FirstErrorAt = attempts[0].Timestamp
		incident.LastErrorAt = attempts[len(attempts)-1].Timestamp
	}

	return incident, r.CreateIncident(ctx, incident)
}

// PageSnapshot holds page state for investigation
type PageSnapshot struct {
	ScreenshotPath string // GCS path or base64
	ScreenshotData []byte // Raw screenshot data
	DOM            string // Full HTML
	Title          string
	FinalURL       string
	Cookies        map[string]interface{}
	Console        []string // Browser console logs
}

// UpdateStatus updates incident status
func (r *IncidentReporter) UpdateStatus(ctx context.Context, id string, status IncidentStatus, resolution string) error {
	query := `UPDATE incidents SET status = $1, resolution = $2, updated_at = NOW(), 
	          resolved_at = CASE WHEN $1 = 'resolved' THEN NOW() ELSE NULL END
	          WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, resolution, id)
	return err
}

// AssignTo assigns incident to a user
func (r *IncidentReporter) AssignTo(ctx context.Context, id, userID string) error {
	query := `UPDATE incidents SET assigned_to = $1, status = 'in_progress', updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, userID, id)
	return err
}

// GetOpenIncidents returns all open incidents
func (r *IncidentReporter) GetOpenIncidents(ctx context.Context) ([]IncidentReport, error) {
	query := `SELECT id, execution_id, task_id, workflow_id, url, domain,
	          error_pattern, error_message, total_attempts, ai_reasoning,
	          suggested_actions, status, priority, created_at
	          FROM incidents WHERE status IN ('open', 'in_progress')
	          ORDER BY 
	            CASE priority 
	              WHEN 'critical' THEN 1 
	              WHEN 'high' THEN 2 
	              WHEN 'medium' THEN 3 
	              ELSE 4 
	            END,
	            created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]IncidentReport, 0)
	for rows.Next() {
		var inc IncidentReport
		var suggestedJSON []byte
		if err := rows.Scan(&inc.ID, &inc.ExecutionID, &inc.TaskID, &inc.WorkflowID,
			&inc.URL, &inc.Domain, &inc.ErrorPattern, &inc.ErrorMessage,
			&inc.TotalAttempts, &inc.AIReasoning, &suggestedJSON,
			&inc.Status, &inc.Priority, &inc.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(suggestedJSON, &inc.SuggestedActions)
		incidents = append(incidents, inc)
	}
	return incidents, nil
}

// GetIncidentDetails returns full incident details
func (r *IncidentReporter) GetIncidentDetails(ctx context.Context, id string) (*IncidentReport, error) {
	query := `SELECT id, execution_id, task_id, workflow_id, url, domain,
	          error_pattern, error_message, status_code,
	          recovery_attempts, total_attempts,
	          ai_enabled, ai_provider, ai_reasoning, ai_failure_reason,
	          screenshot, dom_snapshot, page_title, page_url,
	          browser_profile, proxy_used,
	          suggested_actions, status, priority, assigned_to, resolution,
	          resolved_at, first_error_at, last_error_at, created_at, updated_at
	          FROM incidents WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	var inc IncidentReport
	var attemptsJSON, suggestedJSON []byte
	var assignedTo, resolution, screenshot, dom, pageTitle, pageURL, browserProfile, proxyUsed, aiProvider, aiReasoning, aiFailure *string
	var resolvedAt *time.Time
	var statusCode *int

	err := row.Scan(&inc.ID, &inc.ExecutionID, &inc.TaskID, &inc.WorkflowID, &inc.URL, &inc.Domain,
		&inc.ErrorPattern, &inc.ErrorMessage, &statusCode,
		&attemptsJSON, &inc.TotalAttempts,
		&inc.AIEnabled, &aiProvider, &aiReasoning, &aiFailure,
		&screenshot, &dom, &pageTitle, &pageURL,
		&browserProfile, &proxyUsed,
		&suggestedJSON, &inc.Status, &inc.Priority, &assignedTo, &resolution,
		&resolvedAt, &inc.FirstErrorAt, &inc.LastErrorAt, &inc.CreatedAt, &inc.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if statusCode != nil {
		inc.StatusCode = *statusCode
	}
	if aiProvider != nil {
		inc.AIProvider = *aiProvider
	}
	if aiReasoning != nil {
		inc.AIReasoning = *aiReasoning
	}
	if aiFailure != nil {
		inc.AIFailureReason = *aiFailure
	}
	if screenshot != nil {
		inc.Screenshot = *screenshot
	}
	if dom != nil {
		inc.DOMSnapshot = *dom
	}
	if pageTitle != nil {
		inc.PageTitle = *pageTitle
	}
	if pageURL != nil {
		inc.PageURL = *pageURL
	}
	if browserProfile != nil {
		inc.BrowserProfile = *browserProfile
	}
	if proxyUsed != nil {
		inc.ProxyUsed = *proxyUsed
	}
	if assignedTo != nil {
		inc.AssignedTo = *assignedTo
	}
	if resolution != nil {
		inc.Resolution = *resolution
	}
	if resolvedAt != nil {
		inc.ResolvedAt = resolvedAt
	}

	json.Unmarshal(attemptsJSON, &inc.RecoveryAttempts)
	json.Unmarshal(suggestedJSON, &inc.SuggestedActions)

	return &inc, nil
}

// GetIncidentStats returns incident statistics
func (r *IncidentReporter) GetIncidentStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE status = 'open') as open,
	          COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
	          COUNT(*) FILTER (WHERE status = 'resolved') as resolved,
	          COUNT(*) FILTER (WHERE priority = 'critical') as critical,
	          COUNT(*) FILTER (WHERE priority = 'high') as high,
	          COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as last_24h
	          FROM incidents`

	row := r.pool.QueryRow(ctx, query)

	var total, open, inProgress, resolved, critical, high, last24h int
	if err := row.Scan(&total, &open, &inProgress, &resolved, &critical, &high, &last24h); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":       total,
		"open":        open,
		"in_progress": inProgress,
		"resolved":    resolved,
		"critical":    critical,
		"high":        high,
		"last_24h":    last24h,
	}, nil
}

// calculatePriority determines incident priority
func (r *IncidentReporter) calculatePriority(incident *IncidentReport) IncidentPriority {
	pattern := ErrorPattern(incident.ErrorPattern)

	// Critical patterns
	if pattern == PatternCaptcha || pattern == PatternAuthRequired {
		return PriorityCritical
	}

	// High priority if AI exhausted or many attempts
	if incident.AIFailureReason != "" || incident.TotalAttempts >= 3 {
		return PriorityHigh
	}

	// Medium for blocked/rate limited
	if pattern == PatternBlocked || pattern == PatternRateLimited {
		return PriorityMedium
	}

	return PriorityLow
}

// generateSuggestedActions suggests actions for human investigation
func (r *IncidentReporter) generateSuggestedActions(incident *IncidentReport) []string {
	actions := make([]string, 0)
	pattern := ErrorPattern(incident.ErrorPattern)

	switch pattern {
	case PatternCaptcha:
		actions = append(actions,
			"Visit the URL manually and solve the CAPTCHA",
			"Consider adding a CAPTCHA solving service integration",
			"Check if the domain requires different IP/proxy",
			"Review if the scraping frequency is too aggressive",
		)
	case PatternAuthRequired:
		actions = append(actions,
			"Verify login credentials are correct",
			"Check if session/cookies have expired",
			"Update authentication workflow",
			"Check for 2FA requirements",
		)
	case PatternBlocked:
		actions = append(actions,
			"Try a different proxy or IP range",
			"Review user-agent and browser fingerprint",
			"Reduce scraping frequency for this domain",
			"Check if domain has new anti-bot measures",
			"Consider using residential proxies",
		)
	case PatternRateLimited:
		actions = append(actions,
			"Increase delay between requests",
			"Reduce concurrent workers for this domain",
			"Implement exponential backoff",
			"Check domain's rate limit documentation",
		)
	case PatternLayoutChanged:
		actions = append(actions,
			"Review the DOM snapshot for structural changes",
			"Update CSS selectors in workflow configuration",
			"Check if the website has a new version/redesign",
			"Consider using more robust selectors (XPath, data attributes)",
		)
	default:
		actions = append(actions,
			"Review the screenshot and DOM for clues",
			"Check server status for the domain",
			"Try accessing the URL manually",
			"Review network/connection issues",
		)
	}

	// Add AI-related suggestions if AI was involved
	if incident.AIFailureReason != "" {
		actions = append(actions,
			fmt.Sprintf("AI Analysis: %s", incident.AIReasoning),
			fmt.Sprintf("AI Failure: %s", incident.AIFailureReason),
		)
	}

	return actions
}
