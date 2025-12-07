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
	for _, phase := range phases {
		if phase.Status == "failed" {
			failedCount++
		}
	}

	if failedCount == 0 {
		return "healthy"
	} else if failedCount < len(phases) {
		return "degraded"
	}
	return "broken"
}
