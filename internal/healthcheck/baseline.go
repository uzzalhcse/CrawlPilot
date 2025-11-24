package healthcheck

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

// BaselineService manages baseline health checks and comparisons
type BaselineService struct {
	repo *storage.HealthCheckRepository
}

// NewBaselineService creates a new baseline service
func NewBaselineService(repo *storage.HealthCheckRepository) *BaselineService {
	return &BaselineService{repo: repo}
}

// SetAsBaseline marks a health check report as the baseline for its workflow
func (s *BaselineService) SetAsBaseline(ctx context.Context, reportID string) error {
	// Get the report
	report, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("failed to get report: %w", err)
	}

	// Unset any existing baseline for this workflow
	err = s.repo.UnsetBaseline(ctx, report.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to unset existing baseline: %w", err)
	}

	// Set this report as baseline
	err = s.repo.SetAsBaseline(ctx, reportID)
	if err != nil {
		return fmt.Errorf("failed to set as baseline: %w", err)
	}

	return nil
}

// GetBaseline retrieves the baseline report for a workflow
func (s *BaselineService) GetBaseline(ctx context.Context, workflowID string) (*models.HealthCheckReport, error) {
	return s.repo.GetBaseline(ctx, workflowID)
}

// CompareWithBaseline compares a health check report with the baseline
func (s *BaselineService) CompareWithBaseline(current, baseline *models.HealthCheckReport) []models.BaselineComparison {
	if baseline == nil || baseline.Summary == nil || current.Summary == nil {
		return []models.BaselineComparison{}
	}

	comparisons := []models.BaselineComparison{}

	// Compare total nodes
	comparisons = append(comparisons, models.BaselineComparison{
		Metric:        "total_nodes",
		Baseline:      baseline.Summary.TotalNodes,
		Current:       current.Summary.TotalNodes,
		ChangePercent: calculatePercentChange(baseline.Summary.TotalNodes, current.Summary.TotalNodes),
		Status:        determineComparisonStatus(baseline.Summary.TotalNodes, current.Summary.TotalNodes, false),
	})

	// Compare passed nodes
	comparisons = append(comparisons, models.BaselineComparison{
		Metric:        "passed_nodes",
		Baseline:      baseline.Summary.PassedNodes,
		Current:       current.Summary.PassedNodes,
		ChangePercent: calculatePercentChange(baseline.Summary.PassedNodes, current.Summary.PassedNodes),
		Status:        determineComparisonStatus(baseline.Summary.PassedNodes, current.Summary.PassedNodes, true),
	})

	// Compare failed nodes
	comparisons = append(comparisons, models.BaselineComparison{
		Metric:        "failed_nodes",
		Baseline:      baseline.Summary.FailedNodes,
		Current:       current.Summary.FailedNodes,
		ChangePercent: calculatePercentChange(baseline.Summary.FailedNodes, current.Summary.FailedNodes),
		Status:        determineComparisonStatus(baseline.Summary.FailedNodes, current.Summary.FailedNodes, false), // Lower is better
	})

	// Compare warning nodes
	comparisons = append(comparisons, models.BaselineComparison{
		Metric:        "warning_nodes",
		Baseline:      baseline.Summary.WarningNodes,
		Current:       current.Summary.WarningNodes,
		ChangePercent: calculatePercentChange(baseline.Summary.WarningNodes, current.Summary.WarningNodes),
		Status:        determineComparisonStatus(baseline.Summary.WarningNodes, current.Summary.WarningNodes, false),
	})

	// Compare duration
	comparisons = append(comparisons, models.BaselineComparison{
		Metric:        "duration_ms",
		Baseline:      baseline.Duration,
		Current:       current.Duration,
		ChangePercent: calculatePercentChange(int(baseline.Duration), int(current.Duration)),
		Status:        determineComparisonStatus(int(baseline.Duration), int(current.Duration), false),
	})

	return comparisons
}

func calculatePercentChange(baseline, current int) float64 {
	if baseline == 0 {
		if current == 0 {
			return 0
		}
		return 100 // or could return infinity
	}
	return float64(current-baseline) / float64(baseline) * 100
}

func determineComparisonStatus(baseline, current int, higherIsBetter bool) models.ComparisonStatus {
	if current == baseline {
		return models.ComparisonUnchanged
	}

	if higherIsBetter {
		if current > baseline {
			return models.ComparisonImproved
		}
		return models.ComparisonDegraded
	}

	// Lower is better
	if current < baseline {
		return models.ComparisonImproved
	}
	return models.ComparisonDegraded
}
