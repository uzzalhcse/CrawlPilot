package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/database"
)

// IncidentRepository handles incident CRUD operations
type IncidentRepository struct {
	db *database.DB
}

// Incident represents an incident from the incidents table
type Incident struct {
	ID               string                   `json:"id"`
	ExecutionID      string                   `json:"execution_id"`
	TaskID           string                   `json:"task_id"`
	WorkflowID       string                   `json:"workflow_id"`
	URL              string                   `json:"url"`
	Domain           string                   `json:"domain"`
	ErrorPattern     string                   `json:"error_pattern"`
	ErrorMessage     string                   `json:"error_message,omitempty"`
	StatusCode       int                      `json:"status_code,omitempty"`
	RecoveryAttempts []map[string]interface{} `json:"recovery_attempts"`
	TotalAttempts    int                      `json:"total_attempts"`
	AIEnabled        bool                     `json:"ai_enabled"`
	AIProvider       string                   `json:"ai_provider,omitempty"`
	AIReasoning      string                   `json:"ai_reasoning,omitempty"`
	AIFailureReason  string                   `json:"ai_failure_reason,omitempty"`
	Screenshot       string                   `json:"screenshot,omitempty"`
	DOMSnapshot      string                   `json:"dom_snapshot,omitempty"`
	PageTitle        string                   `json:"page_title,omitempty"`
	PageURL          string                   `json:"page_url,omitempty"`
	SuggestedActions []string                 `json:"suggested_actions"`
	Status           string                   `json:"status"`
	Priority         string                   `json:"priority"`
	AssignedTo       string                   `json:"assigned_to,omitempty"`
	Resolution       string                   `json:"resolution,omitempty"`
	ResolvedAt       *time.Time               `json:"resolved_at,omitempty"`
	FirstErrorAt     time.Time                `json:"first_error_at"`
	LastErrorAt      time.Time                `json:"last_error_at"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

// NewIncidentRepository creates a new incident repository
func NewIncidentRepository(db *database.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// GetAllIncidents returns all incidents with filtering
func (r *IncidentRepository) GetAllIncidents(ctx context.Context, status, priority string, limit, offset int) ([]Incident, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM incidents WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if status != "" {
		countQuery += ` AND status = $` + string(rune('0'+argNum))
		args = append(args, status)
		argNum++
	}
	if priority != "" {
		countQuery += ` AND priority = $` + string(rune('0'+argNum))
		args = append(args, priority)
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Main query
	query := `SELECT id, execution_id, task_id, workflow_id, url, domain,
	          error_pattern, error_message, total_attempts, ai_reasoning,
	          suggested_actions, status, priority, assigned_to, created_at
	          FROM incidents WHERE 1=1`

	args = []interface{}{}

	if status != "" {
		query += ` AND status = $1`
		args = append(args, status)
		if priority != "" {
			query += ` AND priority = $2`
			args = append(args, priority)
		}
	} else if priority != "" {
		query += ` AND priority = $1`
		args = append(args, priority)
	}

	query += ` ORDER BY 
	    CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
	    created_at DESC
	    LIMIT $` + string(rune('0'+len(args)+1)) + ` OFFSET $` + string(rune('0'+len(args)+2))

	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	incidents := make([]Incident, 0)
	for rows.Next() {
		var inc Incident
		var suggestedJSON []byte
		var aiReasoning, assignedTo *string
		if err := rows.Scan(&inc.ID, &inc.ExecutionID, &inc.TaskID, &inc.WorkflowID,
			&inc.URL, &inc.Domain, &inc.ErrorPattern, &inc.ErrorMessage,
			&inc.TotalAttempts, &aiReasoning, &suggestedJSON,
			&inc.Status, &inc.Priority, &assignedTo, &inc.CreatedAt); err != nil {
			continue
		}
		if aiReasoning != nil {
			inc.AIReasoning = *aiReasoning
		}
		if assignedTo != nil {
			inc.AssignedTo = *assignedTo
		}
		json.Unmarshal(suggestedJSON, &inc.SuggestedActions)
		incidents = append(incidents, inc)
	}

	return incidents, total, nil
}

// GetIncidentByID returns full incident details
func (r *IncidentRepository) GetIncidentByID(ctx context.Context, id string) (*Incident, error) {
	query := `SELECT id, execution_id, task_id, workflow_id, url, domain,
	          error_pattern, error_message, status_code,
	          recovery_attempts, total_attempts,
	          ai_enabled, ai_provider, ai_reasoning, ai_failure_reason,
	          screenshot, dom_snapshot, page_title, page_url,
	          suggested_actions, status, priority, assigned_to, resolution,
	          resolved_at, first_error_at, last_error_at, created_at, updated_at
	          FROM incidents WHERE id = $1`

	row := r.db.Pool.QueryRow(ctx, query, id)

	var inc Incident
	var attemptsJSON, suggestedJSON []byte
	var aiProvider, aiReasoning, aiFailure, screenshot, dom, pageTitle, pageURL, assignedTo, resolution *string
	var statusCode *int

	err := row.Scan(&inc.ID, &inc.ExecutionID, &inc.TaskID, &inc.WorkflowID, &inc.URL, &inc.Domain,
		&inc.ErrorPattern, &inc.ErrorMessage, &statusCode,
		&attemptsJSON, &inc.TotalAttempts,
		&inc.AIEnabled, &aiProvider, &aiReasoning, &aiFailure,
		&screenshot, &dom, &pageTitle, &pageURL,
		&suggestedJSON, &inc.Status, &inc.Priority, &assignedTo, &resolution,
		&inc.ResolvedAt, &inc.FirstErrorAt, &inc.LastErrorAt, &inc.CreatedAt, &inc.UpdatedAt)

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
	if assignedTo != nil {
		inc.AssignedTo = *assignedTo
	}
	if resolution != nil {
		inc.Resolution = *resolution
	}

	json.Unmarshal(attemptsJSON, &inc.RecoveryAttempts)
	json.Unmarshal(suggestedJSON, &inc.SuggestedActions)

	return &inc, nil
}

// UpdateIncidentStatus updates the status of an incident
func (r *IncidentRepository) UpdateIncidentStatus(ctx context.Context, id, status, resolution string) error {
	query := `UPDATE incidents SET status = $1, resolution = $2, updated_at = NOW(),
	          resolved_at = CASE WHEN $1 = 'resolved' THEN NOW() ELSE NULL END
	          WHERE id = $3`
	_, err := r.db.Pool.Exec(ctx, query, status, resolution, id)
	return err
}

// AssignIncident assigns an incident to a user
func (r *IncidentRepository) AssignIncident(ctx context.Context, id, userID string) error {
	query := `UPDATE incidents SET assigned_to = $1, status = 'in_progress', updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, userID, id)
	return err
}

// GetIncidentStats returns incident statistics
func (r *IncidentRepository) GetIncidentStats(ctx context.Context) (map[string]interface{}, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(*) FILTER (WHERE status = 'open') as open,
	          COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
	          COUNT(*) FILTER (WHERE status = 'resolved') as resolved,
	          COUNT(*) FILTER (WHERE priority = 'critical') as critical,
	          COUNT(*) FILTER (WHERE priority = 'high') as high,
	          COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as last_24h,
	          COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '7 days') as last_7days
	          FROM incidents`

	row := r.db.Pool.QueryRow(ctx, query)

	var total, open, inProgress, resolved, critical, high, last24h, last7days int
	if err := row.Scan(&total, &open, &inProgress, &resolved, &critical, &high, &last24h, &last7days); err != nil {
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
		"last_7days":  last7days,
	}, nil
}

// GetDomainIncidentStats returns incidents grouped by domain
func (r *IncidentRepository) GetDomainIncidentStats(ctx context.Context) ([]map[string]interface{}, error) {
	query := `SELECT domain, error_pattern, COUNT(*) as count,
	          COUNT(*) FILTER (WHERE status = 'open') as open_count
	          FROM incidents
	          GROUP BY domain, error_pattern
	          ORDER BY count DESC
	          LIMIT 20`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]map[string]interface{}, 0)
	for rows.Next() {
		var domain, pattern string
		var count, openCount int
		if err := rows.Scan(&domain, &pattern, &count, &openCount); err != nil {
			continue
		}
		stats = append(stats, map[string]interface{}{
			"domain":     domain,
			"pattern":    pattern,
			"count":      count,
			"open_count": openCount,
		})
	}
	return stats, nil
}
