package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
	"go.uber.org/zap"
)

// ProbeReporter handles reporting probe results to orchestrator
type ProbeReporter struct {
	orchestratorURL string
	httpClient      *http.Client
}

// NewProbeReporter creates a new probe reporter
func NewProbeReporter(orchestratorURL string) *ProbeReporter {
	return &ProbeReporter{
		orchestratorURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProbeResultRequest is the request body for saving probe results
type ProbeResultRequest struct {
	ExecutionID string                    `json:"execution_id"`
	WorkflowID  string                    `json:"workflow_id"`
	Status      string                    `json:"status"`
	DurationMs  int                       `json:"duration_ms"`
	Phases      []models.PhaseProbeResult `json:"phases"`
}

// ReportProbeResult sends probe result to orchestrator
func (r *ProbeReporter) ReportProbeResult(ctx context.Context, result *ProbeResultRequest) error {
	url := fmt.Sprintf("%s/api/v1/internal/probes/result", r.orchestratorURL)

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal probe result: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send probe result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	logger.Info("Probe result reported to orchestrator",
		zap.String("execution_id", result.ExecutionID),
		zap.String("status", result.Status),
	)

	return nil
}

// SampleURLsRequest is the request body for saving sample URLs
type SampleURLsRequest struct {
	WorkflowID string            `json:"workflow_id"`
	SampleURLs map[string]string `json:"sample_urls"`
}

// ReportSampleURLs sends sample URLs to orchestrator
func (r *ProbeReporter) ReportSampleURLs(ctx context.Context, workflowID string, sampleURLs map[string]string) error {
	if len(sampleURLs) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/internal/probes/sample-urls", r.orchestratorURL)

	reqBody := SampleURLsRequest{
		WorkflowID: workflowID,
		SampleURLs: sampleURLs,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal sample URLs: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send sample URLs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	logger.Info("Sample URLs reported to orchestrator",
		zap.String("workflow_id", workflowID),
		zap.Int("count", len(sampleURLs)),
	)

	return nil
}

// DetermineProbeStatus determines overall probe status based on phase results
func DetermineProbeStatus(phases []models.PhaseProbeResult) string {
	failedCount := 0
	degradedCount := 0

	for _, phase := range phases {
		if phase.Status == "failed" {
			failedCount++
			continue
		}

		// Check for degraded conditions: nodes that "passed" but found nothing or missing required fields
		for _, node := range phase.Nodes {
			if node.Status == "passed" {
				// extract_links finding 0 links is a problem
				if node.NodeType == "extract_links" && node.LinksFound == 0 {
					degradedCount++
				}
				// extract finding 0 elements is a problem
				if node.NodeType == "extract" && node.ElementCount == 0 {
					degradedCount++
				}
				// extract missing required fields is a problem
				if node.NodeType == "extract" && len(node.MissingRequiredFields) > 0 {
					degradedCount++
				}
			}
		}
	}

	if failedCount > 0 {
		if failedCount == len(phases) {
			return "broken"
		}
		return "degraded"
	}

	if degradedCount > 0 {
		return "degraded"
	}

	return "healthy"
}

// GetBaseline fetches the last successful probe result for comparison
func (r *ProbeReporter) GetBaseline(ctx context.Context, workflowID string) (*models.ProbeResult, error) {
	url := fmt.Sprintf("%s/api/v1/internal/probes/baseline/%s", r.orchestratorURL, workflowID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch baseline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No baseline exists yet
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	var result models.ProbeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode baseline: %w", err)
	}

	logger.Info("Baseline fetched from orchestrator",
		zap.String("workflow_id", workflowID),
		zap.String("status", result.Status),
	)

	return &result, nil
}

// WorkflowFixRequest is the request body for applying workflow fixes
type WorkflowFixRequest struct {
	WorkflowID  string  `json:"workflow_id"`
	ExecutionID string  `json:"execution_id"`
	NodeID      string  `json:"node_id"`
	FieldName   string  `json:"field_name,omitempty"` // For update_field_selector
	FixType     string  `json:"fix_type"`             // "update_selector", "update_field_selector", "skip_node"
	OldSelector string  `json:"old_selector,omitempty"`
	NewSelector string  `json:"new_selector,omitempty"`
	Reasoning   string  `json:"reasoning"`
	Confidence  float64 `json:"confidence"`
	AutoApplied bool    `json:"auto_applied"`
}

// ApplyWorkflowFix sends a workflow fix to orchestrator for application
func (r *ProbeReporter) ApplyWorkflowFix(ctx context.Context, fix *WorkflowFixRequest) error {
	url := fmt.Sprintf("%s/api/v1/internal/probes/fix", r.orchestratorURL)

	body, err := json.Marshal(fix)
	if err != nil {
		return fmt.Errorf("failed to marshal fix request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to apply workflow fix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("orchestrator returned error: %d", resp.StatusCode)
	}

	logger.Info("Workflow fix applied via orchestrator",
		zap.String("workflow_id", fix.WorkflowID),
		zap.String("node_id", fix.NodeID),
		zap.String("fix_type", fix.FixType),
		zap.Float64("confidence", fix.Confidence),
	)

	return nil
}
