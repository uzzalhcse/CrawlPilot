package workflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/error_recovery"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// tryRecoverFromError attempts to recover from an error using the error recovery system
func (e *Executor) tryRecoverFromError(ctx context.Context, err error, item *models.URLQueueItem, response *ResponseInfo) error {
	startTime := time.Now()

	logger.Info("🚨 Error Recovery: Attempting to recover from error",
		zap.String("url", item.URL),
		zap.String("error", err.Error()))

	if e.errorRecoverySystem == nil {
		logger.Debug("⚠️  Error recovery system not initialized, skipping recovery")
		return err
	}

	// Cast to error recovery system
	system, ok := e.errorRecoverySystem.(*error_recovery.ErrorRecoverySystem)
	if !ok {
		logger.Warn("⚠️  Error recovery system type assertion failed")
		return err
	}

	// Create execution context for error recovery
	execCtx := e.createErrorRecoveryContext(item, err, response)

	// Initialize history record
	historyRecord := e.createHistoryRecord(item, err, response, startTime)

	// Save initial record immediately
	e.saveHistoryRecord(ctx, historyRecord)

	// Try to get a solution
	solution, recoveryErr := system.HandleError(ctx, err, execCtx)

	// Update history with pattern detection info
	if analyzer := system.GetAnalyzer(); analyzer != nil {
		if decision := analyzer.ShouldActivate(err, execCtx); decision.ShouldActivate {
			historyRecord.PatternDetected = true
			historyRecord.ActivationReason = decision.Reason
			// Determine pattern type from reason
			if contains(decision.Reason, "rate") {
				historyRecord.PatternType = "rate_spike"
			} else if contains(decision.Reason, "consecutive") {
				historyRecord.PatternType = "consecutive"
			} else {
				historyRecord.PatternType = "systematic"
			}
		}
	}

	if recoveryErr != nil {
		logger.Debug("❌ No error recovery solution available",
			zap.Error(recoveryErr),
			zap.String("url", item.URL))

		// Log failed recovery attempt
		historyRecord.SolutionType = "none"
		historyRecord.RecoveryAttempted = false
		historyRecord.RecoverySuccessful = false
		e.updateHistoryRecord(ctx, historyRecord)

		return err
	}

	logger.Info("✅ Recovery solution found - applying...",
		zap.String("rule", solution.RuleName),
		zap.String("type", solution.Type),
		zap.Float64("confidence", solution.Confidence),
		zap.Int("actions", len(solution.Actions)))

	// Update history record with solution info
	historyRecord.RuleName = solution.RuleName
	historyRecord.SolutionType = solution.Type
	historyRecord.Confidence = &solution.Confidence
	historyRecord.ActionsApplied = solution.Actions
	if solution.RuleID != nil {
		historyRecord.RuleID = solution.RuleID
	}

	// Update record with solution details before applying actions
	e.updateHistoryRecord(ctx, historyRecord)

	// Apply the solution
	logger.Debug("🔧 Applying recovery actions...", zap.Int("action_count", len(solution.Actions)))
	if applyErr := e.applyRecoverySolution(ctx, solution, item); applyErr != nil {
		logger.Error("❌ Failed to apply recovery solution",
			zap.Error(applyErr),
			zap.String("rule", solution.RuleName))
		system.TrackFailure(solution)

		// Log failed application
		historyRecord.RecoveryAttempted = true
		historyRecord.RecoverySuccessful = false
		recoveredAt := time.Now()
		historyRecord.RecoveredAt = &recoveredAt
		historyRecord.TimeToRecoveryMs = int(time.Since(startTime).Milliseconds())
		e.updateHistoryRecord(ctx, historyRecord)

		return err
	}

	// Track success
	system.TrackSuccess(solution, execCtx)
	logger.Info("✅ Recovery successful - tracking for learning",
		zap.String("rule", solution.RuleName),
		zap.String("type", solution.Type))

	// Log successful recovery
	historyRecord.RecoveryAttempted = true
	historyRecord.RecoverySuccessful = true
	recoveredAt := time.Now()
	historyRecord.RecoveredAt = &recoveredAt
	historyRecord.TimeToRecoveryMs = int(time.Since(startTime).Milliseconds())
	e.updateHistoryRecord(ctx, historyRecord)

	// Return nil to indicate recovery was successful (caller should retry)
	return nil
}

// createErrorRecoveryContext creates an ExecutionContext for error recovery
func (e *Executor) createErrorRecoveryContext(item *models.URLQueueItem, err error, response *ResponseInfo) *error_recovery.ExecutionContext {
	execCtx := error_recovery.NewExecutionContext()
	execCtx.URL = item.URL

	// Extract domain from URL
	if parsedURL, parseErr := url.Parse(item.URL); parseErr == nil {
		execCtx.Domain = parsedURL.Host
	}

	// Set error info
	execCtx.Error.Message = err.Error()
	execCtx.Error.Type = fmt.Sprintf("%T", err)

	// Set response info if available
	if response != nil {
		execCtx.Response.StatusCode = response.StatusCode
		execCtx.Response.Body = response.Body
		execCtx.Response.Header = response.Headers
	}

	return execCtx
}

// ResponseInfo captures HTTP response details
type ResponseInfo struct {
	StatusCode int
	Headers    map[string][]string
	Body       string
}

// applyRecoverySolution applies the actions from a recovery solution
func (e *Executor) applyRecoverySolution(ctx context.Context, solution *error_recovery.Solution, item *models.URLQueueItem) error {
	for _, action := range solution.Actions {
		logger.Debug("Applying recovery action",
			zap.String("action", action.Type),
			zap.Any("parameters", action.Parameters))

		switch action.Type {
		case "wait":
			if duration, ok := action.Parameters["duration"].(float64); ok {
				waitTime := time.Duration(duration) * time.Second
				logger.Info("⏱️  Waiting before retry", zap.Duration("duration", waitTime))
				time.Sleep(waitTime)
			}

		case "enable_stealth":
			// This would be handled by browser context in actual navigation
			logger.Info("🥷 Stealth mode would be enabled",
				zap.Any("level", action.Parameters["level"]))

		case "rotate_proxy":
			// Proxy rotation would be handled during browser acquisition
			logger.Info("🔄 Proxy rotation flagged for next request")

		case "reduce_workers":
			// Worker reduction would be handled at executor level
			if count, ok := action.Parameters["count"].(float64); ok {
				logger.Info("⬇️  Reducing concurrent workers", zap.Int("new_count", int(count)))
			}

		case "add_delay":
			if duration, ok := action.Parameters["duration"].(float64); ok {
				delayTime := time.Duration(duration) * time.Millisecond
				logger.Info("⏸️  Adding delay between requests", zap.Duration("delay", delayTime))
				time.Sleep(delayTime)
			}

		case "adjust_timeout":
			if multiplier, ok := action.Parameters["multiplier"].(float64); ok {
				logger.Info("⏲️  Adjusting timeout", zap.Float64("multiplier", multiplier))
			}

		case "pause_execution":
			logger.Info("⏸️  Pausing execution")

		case "resume_execution":
			logger.Info("▶️  Resuming execution")

		default:
			logger.Warn("Unknown recovery action", zap.String("action", action.Type))
		}
	}

	return nil
}

// createHistoryRecord initializes a history record for the current recovery attempt
func (e *Executor) createHistoryRecord(item *models.URLQueueItem, err error, response *ResponseInfo, startTime time.Time) *storage.RecoveryHistoryRecord {
	// Parse execution ID
	executionUUID, _ := uuid.Parse(item.ExecutionID)
	workflowUUID := uuid.UUID{} // Zero value for now

	record := &storage.RecoveryHistoryRecord{
		ExecutionID:  executionUUID,
		WorkflowID:   workflowUUID,
		ErrorType:    fmt.Sprintf("%T", err),
		ErrorMessage: err.Error(),
		URL:          item.URL,
		NodeID:       item.URL, // Use URL as node context for now
		PhaseID:      item.PhaseID,
		DetectedAt:   startTime,
		RetryCount:   1,
	}

	// Extract domain from URL
	if parsedURL, parseErr := url.Parse(item.URL); parseErr == nil {
		record.Domain = parsedURL.Host
	}

	// Add status code if available
	if response != nil && response.StatusCode > 0 {
		record.StatusCode = &response.StatusCode
	}

	// Add request context
	record.RequestContext = map[string]interface{}{
		"depth":    item.Depth,
		"marker":   item.Marker,
		"phase_id": item.PhaseID,
	}

	return record
}

// saveHistoryRecord saves a history record to the database
func (e *Executor) saveHistoryRecord(ctx context.Context, record *storage.RecoveryHistoryRecord) {
	if e.recoveryHistoryRepo == nil {
		return
	}

	if err := e.recoveryHistoryRepo.Create(ctx, record); err != nil {
		logger.Error("Failed to save recovery history record",
			zap.Error(err),
			zap.String("url", record.URL))
	}
}

// updateHistoryRecord updates an existing history record in the database
func (e *Executor) updateHistoryRecord(ctx context.Context, record *storage.RecoveryHistoryRecord) {
	if e.recoveryHistoryRepo == nil {
		return
	}

	if err := e.recoveryHistoryRepo.Update(ctx, record); err != nil {
		logger.Error("Failed to update recovery history record",
			zap.Error(err),
			zap.String("url", record.URL))
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
