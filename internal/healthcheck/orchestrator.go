package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/browser"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/workflow"
	"github.com/uzzalhcse/crawlify/internal/workflow/nodes"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// Orchestrator manages health check execution
type Orchestrator struct {
	browserPool *browser.BrowserPool
	registry    *workflow.NodeRegistry
	config      *models.HealthCheckConfig
}

// NewOrchestrator creates a new health check orchestrator
func NewOrchestrator(browserPool *browser.BrowserPool, registry *workflow.NodeRegistry, config *models.HealthCheckConfig) *Orchestrator {
	if config == nil {
		config = &models.HealthCheckConfig{
			MaxURLsPerPhase:    1,
			MaxPaginationPages: 2,
			MaxDepth:           2,
			TimeoutSeconds:     300,
			SkipDataStorage:    true,
		}
	}

	return &Orchestrator{
		browserPool: browserPool,
		registry:    registry,
		config:      config,
	}
}

// RunHealthCheck executes a health check for a workflow
func (o *Orchestrator) RunHealthCheck(ctx context.Context, wf *models.Workflow) (*models.HealthCheckReport, error) {
	report := &models.HealthCheckReport{
		ID:           uuid.New().String(),
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		Status:       models.HealthCheckStatusRunning,
		StartedAt:    time.Now(),
		Config:       o.config,
		Results:      make(map[string]*models.PhaseValidationResult),
	}

	logger.Info("Starting health check",
		zap.String("workflow_id", wf.ID),
		zap.String("workflow_name", wf.Name))

	// Track discovered URLs from each phase to use in next phase
	currentPhaseURLs := wf.Config.StartURLs

	// Validate each phase
	for _, phase := range wf.Config.Phases {
		phaseResult, discoveredURLs, err := o.validatePhase(ctx, wf, &phase, currentPhaseURLs)
		if err != nil && phaseResult.NavigationError == "" {
			phaseResult.NavigationError = err.Error()
		}
		report.Results[phase.ID] = phaseResult

		logger.Info("Phase validation completed",
			zap.String("phase_id", phase.ID),
			zap.Bool("has_critical_issues", phaseResult.HasCriticalIssues),
			zap.Int("discovered_urls", len(discoveredURLs)))

		// Use discovered URLs for next phase (limit to max configured)
		if len(discoveredURLs) > 0 {
			maxURLs := o.config.MaxURLsPerPhase
			if len(discoveredURLs) > maxURLs {
				currentPhaseURLs = discoveredURLs[:maxURLs]
			} else {
				currentPhaseURLs = discoveredURLs
			}
			logger.Debug("URLs for next phase",
				zap.Strings("urls", currentPhaseURLs))
		} else {
			logger.Warn("No URLs discovered, next phase will use current URLs",
				zap.String("phase_id", phase.ID),
				zap.Strings("current_urls", currentPhaseURLs))
		}
		// If no URLs discovered, keep using current URLs (will fail next phase)
	}

	// Generate summary
	report.Summary = o.generateSummary(report.Results)
	report.Status = o.determineOverallStatus(report.Summary)

	completedAt := time.Now()
	report.CompletedAt = &completedAt
	report.Duration = completedAt.Sub(report.StartedAt).Milliseconds()

	logger.Info("Health check completed",
		zap.String("workflow_id", wf.ID),
		zap.String("status", string(report.Status)),
		zap.Int64("duration_ms", report.Duration))

	return report, nil
}

// validatePhase validates all nodes in a workflow phase
func (o *Orchestrator) validatePhase(ctx context.Context, wf *models.Workflow, phase *models.WorkflowPhase, phaseURLs []string) (*models.PhaseValidationResult, []string, error) {
	result := &models.PhaseValidationResult{
		PhaseID:     phase.ID,
		PhaseName:   phase.Name,
		NodeResults: []models.NodeValidationResult{},
	}

	// Get test URL
	if len(phaseURLs) == 0 {
		return result, nil, fmt.Errorf("no URLs available for phase %s", phase.ID)
	}
	testURL := phaseURLs[0]

	logger.Debug("Validating phase",
		zap.String("phase_id", phase.ID),
		zap.String("test_url", testURL))

	// Acquire browser
	browserCtx, err := o.browserPool.Acquire(ctx)
	if err != nil {
		return result, nil, fmt.Errorf("failed to acquire browser: %w", err)
	}
	defer o.browserPool.Release(browserCtx)

	// Navigate
	_, err = browserCtx.Navigate(testURL)
	if err != nil {
		return result, nil, fmt.Errorf("failed to navigate to %s: %w", testURL, err)
	}

	// Wait a moment for page to settle
	time.Sleep(1 * time.Second)

	// Validate each node and collect discovered URLs
	execCtx := models.NewExecutionContext()
	discoveredURLs := []string{}

	for _, node := range phase.Nodes {
		nodeResult := o.validateNode(ctx, &node, browserCtx, &execCtx)
		result.NodeResults = append(result.NodeResults, nodeResult)

		// Check for critical issues
		if nodeResult.Status == models.ValidationStatusFail {
			result.HasCriticalIssues = true
		}

		// Collect URLs from any node that discovered them
		if urls, ok := nodeResult.Metrics["discovered_urls"].([]string); ok && len(urls) > 0 {
			logger.Debug("Collecting URLs from node",
				zap.String("node_id", node.ID),
				zap.String("node_type", string(node.Type)),
				zap.Int("url_count", len(urls)),
				zap.Strings("sample_urls", func() []string {
					if len(urls) > 3 {
						return urls[:3]
					}
					return urls
				}()))
			discoveredURLs = append(discoveredURLs, urls...)
		}
	}

	logger.Debug("Phase validation complete",
		zap.String("phase_id", phase.ID),
		zap.Int("total_discovered_urls", len(discoveredURLs)))

	return result, discoveredURLs, nil
}

// validateNode validates a single workflow node
func (o *Orchestrator) validateNode(ctx context.Context, node *models.Node, browserCtx *browser.BrowserContext, execCtx *models.ExecutionContext) models.NodeValidationResult {
	startTime := time.Now()

	logger.Debug("Validating node",
		zap.String("node_id", node.ID),
		zap.String("node_type", string(node.Type)))

	// Get node executor
	executor, err := o.registry.Get(node.Type)
	if err != nil {
		return models.NodeValidationResult{
			NodeID:   node.ID,
			NodeName: node.Name,
			NodeType: string(node.Type),
			Status:   models.ValidationStatusFail,
			Metrics:  make(map[string]interface{}),
			Issues: []models.ValidationIssue{{
				Severity: "critical",
				Code:     "NODE_TYPE_NOT_FOUND",
				Message:  fmt.Sprintf("No validator for node type: %s", node.Type),
			}},
		}
	}

	// Check if executor implements IHealthCheckValidator
	validator, ok := executor.(nodes.IHealthCheckValidator)
	if !ok {
		// Use generic validator as fallback
		validator = NewGenericValidator(node.Type)
	}

	// Run validation
	input := &nodes.ValidationInput{
		BrowserContext:   browserCtx,
		ExecutionContext: execCtx,
		Params:           node.Params,
		Config:           o.config,
	}

	result, err := validator.ValidateForHealthCheck(ctx, input)
	if err != nil {
		if result == nil {
			result = &models.NodeValidationResult{
				NodeType: string(node.Type),
				Status:   models.ValidationStatusFail,
				Metrics:  make(map[string]interface{}),
				Issues:   []models.ValidationIssue{},
			}
		}
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity: "critical",
			Code:     "VALIDATION_ERROR",
			Message:  err.Error(),
		})
	}

	result.NodeID = node.ID
	result.NodeName = node.Name
	result.Duration = time.Since(startTime).Milliseconds()

	return *result
}

// generateSummary aggregates validation results into a summary
func (o *Orchestrator) generateSummary(results map[string]*models.PhaseValidationResult) *models.HealthCheckSummary {
	summary := &models.HealthCheckSummary{
		TotalPhases:    len(results),
		CriticalIssues: []models.ValidationIssue{},
	}

	for _, phaseResult := range results {
		for _, nodeResult := range phaseResult.NodeResults {
			summary.TotalNodes++

			switch nodeResult.Status {
			case models.ValidationStatusPass:
				summary.PassedNodes++
			case models.ValidationStatusFail:
				summary.FailedNodes++
			case models.ValidationStatusWarning:
				summary.WarningNodes++
			}

			// Collect critical issues
			for _, issue := range nodeResult.Issues {
				if issue.Severity == "critical" {
					summary.CriticalIssues = append(summary.CriticalIssues, issue)
				}
			}
		}
	}

	return summary
}

// determineOverallStatus determines the overall health status
func (o *Orchestrator) determineOverallStatus(summary *models.HealthCheckSummary) models.HealthCheckStatus {
	if summary.FailedNodes > 0 {
		return models.HealthCheckStatusFailed
	}
	if summary.WarningNodes > 0 {
		return models.HealthCheckStatusDegraded
	}
	return models.HealthCheckStatusHealthy
}
