package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// RecoveryReporter handles reporting recovery attempts to orchestrator
type RecoveryReporter struct {
	orchestratorURL string
	httpClient      *http.Client
}

// NewRecoveryReporter creates a new recovery reporter
func NewRecoveryReporter(orchestratorURL string) *RecoveryReporter {
	return &RecoveryReporter{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateAttemptRequest is the request body for creating a recovery attempt
type CreateAttemptRequest struct {
	ExecutionID  string `json:"execution_id"`
	TaskID       string `json:"task_id"`
	WorkflowID   string `json:"workflow_id"`
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	ErrorPattern string `json:"error_pattern"`
	ErrorMessage string `json:"error_message,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
}

// CreateAttemptResponse is the response from creating a recovery attempt
type CreateAttemptResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// CreateAttempt sends a new recovery attempt to orchestrator (called immediately on error detection)
func (r *RecoveryReporter) CreateAttempt(ctx context.Context, req *CreateAttemptRequest) (*CreateAttemptResponse, error) {
	url := fmt.Sprintf("%s/api/v1/internal/recovery/attempt", r.orchestratorURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		logger.Warn("Failed to report recovery attempt to orchestrator (will continue)",
			zap.String("execution_id", req.ExecutionID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	var result CreateAttemptResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.Debug("Recovery attempt created",
		zap.String("attempt_id", result.ID),
		zap.String("execution_id", req.ExecutionID),
		zap.String("error_pattern", req.ErrorPattern),
	)

	return &result, nil
}

// UpdateAttemptRequest is the request body for updating a recovery attempt
type UpdateAttemptRequest struct {
	Action       string `json:"action"`
	Source       string `json:"source"` // rule, ai, default, none
	Status       string `json:"status"` // success, failed
	RuleID       string `json:"rule_id,omitempty"`
	AIReasoning  string `json:"ai_reasoning,omitempty"`
	RetryDelayMs int    `json:"retry_delay_ms,omitempty"`
	DurationMs   int    `json:"duration_ms,omitempty"`
}

// UpdateAttempt updates a recovery attempt with outcome (called after recovery execution)
func (r *RecoveryReporter) UpdateAttempt(ctx context.Context, attemptID string, req *UpdateAttemptRequest) error {
	url := fmt.Sprintf("%s/api/v1/internal/recovery/attempt/%s", r.orchestratorURL, attemptID)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		logger.Warn("Failed to update recovery attempt in orchestrator (will continue)",
			zap.String("attempt_id", attemptID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	logger.Debug("Recovery attempt updated",
		zap.String("attempt_id", attemptID),
		zap.String("status", req.Status),
		zap.String("action", req.Action),
	)

	return nil
}
