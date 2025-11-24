package handlers

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uzzalhcse/crawlify/internal/ai"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/storage"
	"github.com/uzzalhcse/crawlify/pkg/models"
	"go.uber.org/zap"
)

// AutoFixHandler handles AI auto-fix requests
type AutoFixHandler struct {
	snapshotRepo     *storage.SnapshotRepository
	suggestionRepo   *storage.FixSuggestionRepository
	workflowRepo     *storage.WorkflowRepository
	versionRepo      *storage.WorkflowVersionRepository
	healthCheckRepo  *storage.HealthCheckRepository // NEW
	autoFixService   *ai.AutoFixService
	snapshotBasePath string
}

// NewAutoFixHandler creates a new autofix handler
func NewAutoFixHandler(
	snapshotRepo *storage.SnapshotRepository,
	suggestionRepo *storage.FixSuggestionRepository,
	workflowRepo *storage.WorkflowRepository,
	versionRepo *storage.WorkflowVersionRepository,
	healthCheckRepo *storage.HealthCheckRepository, // NEW
	autoFixService *ai.AutoFixService,
	snapshotBasePath string,
) *AutoFixHandler {
	return &AutoFixHandler{
		snapshotRepo:     snapshotRepo,
		suggestionRepo:   suggestionRepo,
		workflowRepo:     workflowRepo,
		versionRepo:      versionRepo,
		healthCheckRepo:  healthCheckRepo,
		autoFixService:   autoFixService,
		snapshotBasePath: snapshotBasePath,
	}
}

// AnalyzeSnapshot triggers AI analysis for a snapshot
// POST /api/v1/snapshots/:id/analyze
func (h *AutoFixHandler) AnalyzeSnapshot(c *fiber.Ctx) error {
	snapshotID := c.Params("id")

	// Get snapshot
	snapshot, err := h.snapshotRepo.GetByID(c.Context(), snapshotID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Snapshot not found",
		})
	}

	// Get health check report to find the correct WorkflowID
	report, err := h.healthCheckRepo.GetByID(c.Context(), snapshot.ReportID)
	if err != nil {
		logger.Error("Failed to fetch health check report", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch health check report",
		})
	}

	// Get workflow to find the node and construct proper config update
	workflow, err := h.workflowRepo.GetByID(c.Context(), report.WorkflowID)
	if err != nil {
		logger.Error("Failed to fetch workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch workflow",
		})
	}

	// Build screenshot path
	var screenshotPath string
	if snapshot.ScreenshotPath != nil {
		screenshotPath = filepath.Join(h.snapshotBasePath, *snapshot.ScreenshotPath)
	}

	// Build DOM snapshot path
	if snapshot.DOMSnapshotPath != nil {
		fullDOMPath := filepath.Join(h.snapshotBasePath, *snapshot.DOMSnapshotPath)
		snapshot.DOMSnapshotPath = &fullDOMPath
	}

	// Try to get baseline preview data to help AI understand expected output
	var baselinePreview string
	baseline, err := h.healthCheckRepo.GetBaseline(c.Context(), report.WorkflowID)
	if err == nil && baseline != nil {
		// Get baseline snapshots for the same node
		baselineSnapshots, err := h.snapshotRepo.GetByReportID(c.Context(), baseline.ID)
		if err == nil {
			// Find snapshot for the same node
			for _, baselineSnap := range baselineSnapshots {
				if baselineSnap.NodeID == snapshot.NodeID && baselineSnap.ElementsFound > 0 {
					// This baseline snapshot worked! Extract preview using its selector
					if baselineSnap.DOMSnapshotPath != nil && baselineSnap.SelectorValue != nil {
						fullPath := filepath.Join(h.snapshotBasePath, *baselineSnap.DOMSnapshotPath)
						if domBytes, err := h.readAndExtractPreview(fullPath, *baselineSnap.SelectorValue); err == nil {
							baselinePreview = domBytes
							logger.Info("Loaded baseline preview for AI",
								zap.String("baseline_selector", *baselineSnap.SelectorValue),
								zap.Int("preview_length", len(baselinePreview)))
						}
					}
					break
				}
			}
		}
	}

	// Analyze with AI
	logger.Info("Analyzing snapshot with AI", zap.String("snapshot_id", snapshotID))
	aiSuggestion, err := h.autoFixService.AnalyzeSnapshot(c.Context(), snapshot, screenshotPath, baselinePreview)
	if err != nil {
		logger.Error("AI analysis failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "AI analysis failed: " + err.Error(),
		})
	}

	// Construct SuggestedNodeConfig by finding where the failed selector is used
	suggestedNodeConfig := make(map[string]interface{})

	// Find the node
	var targetNode *models.Node
	for _, phase := range workflow.Config.Phases {
		for _, node := range phase.Nodes {
			if node.ID == snapshot.NodeID {
				targetNode = &node
				break
			}
		}
		if targetNode != nil {
			break
		}
	}

	if targetNode != nil && snapshot.SelectorValue != nil {
		failedSelector := *snapshot.SelectorValue

		// Check top-level params
		if val, ok := targetNode.Params["selector"].(string); ok && val == failedSelector {
			suggestedNodeConfig["selector"] = aiSuggestion.SuggestedSelector
		} else if fields, ok := targetNode.Params["fields"].(map[string]interface{}); ok {
			// Check inside fields
			fieldsCopy := make(map[string]interface{})
			// Deep copy fields to avoid mutating original
			for k, v := range fields {
				fieldsCopy[k] = v
			}

			updated := false
			for fieldName, fieldConfig := range fields {
				if configMap, ok := fieldConfig.(map[string]interface{}); ok {
					if sel, ok := configMap["selector"].(string); ok && sel == failedSelector {
						// Found the field! Update its selector
						newConfigMap := make(map[string]interface{})
						for k, v := range configMap {
							newConfigMap[k] = v
						}
						newConfigMap["selector"] = aiSuggestion.SuggestedSelector
						fieldsCopy[fieldName] = newConfigMap
						updated = true
						break // Assuming only one field matches for now
					}
				}
			}

			if updated {
				suggestedNodeConfig["fields"] = fieldsCopy
			}
		}
	}

	// If we couldn't intelligently construct the config, fallback to just setting "selector"
	// if the AI returned a config (which it usually doesn't in current impl) or just the selector
	if len(suggestedNodeConfig) == 0 {
		if len(aiSuggestion.SuggestedNodeConfig) > 0 {
			suggestedNodeConfig = aiSuggestion.SuggestedNodeConfig
		} else {
			// Fallback: just try to set 'selector' param
			suggestedNodeConfig["selector"] = aiSuggestion.SuggestedSelector
		}
	}

	// Save suggestion to database
	suggestion := &models.FixSuggestion{
		ID:                   uuid.New().String(),
		SnapshotID:           snapshotID,
		WorkflowID:           report.WorkflowID, // Correctly use WorkflowID from report
		NodeID:               snapshot.NodeID,
		SuggestedSelector:    aiSuggestion.SuggestedSelector,
		AlternativeSelectors: aiSuggestion.AlternativeSelectors,
		SuggestedNodeConfig:  suggestedNodeConfig,
		FixExplanation:       aiSuggestion.Explanation,
		ConfidenceScore:      aiSuggestion.Confidence,
		VerificationResult:   aiSuggestion.VerificationResult, // Include verification results
		Status:               "pending",
		AIModel:              "gemini-2.0-flash-exp",
	}

	if err := h.suggestionRepo.Create(c.Context(), suggestion); err != nil {
		logger.Error("Failed to save suggestion", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save suggestion",
		})
	}

	logger.Info("AI suggestion created",
		zap.String("suggestion_id", suggestion.ID),
		zap.Float64("confidence", suggestion.ConfidenceScore))

	return c.JSON(suggestion)
}

// GetSuggestions retrieves all suggestions for a snapshot
// GET /api/v1/snapshots/:id/suggestions
func (h *AutoFixHandler) GetSuggestions(c *fiber.Ctx) error {
	snapshotID := c.Params("id")

	suggestions, err := h.suggestionRepo.GetBySnapshotID(c.Context(), snapshotID)
	if err != nil {
		logger.Error("Failed to fetch suggestions", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch suggestions",
		})
	}

	return c.JSON(suggestions)
}

// ApproveSuggestion approves a suggestion for implementation
// POST /api/v1/suggestions/:id/approve
func (h *AutoFixHandler) ApproveSuggestion(c *fiber.Ctx) error {
	suggestionID := c.Params("id")

	// TODO: Get reviewer from authenticated user context
	reviewedBy := "system"

	if err := h.suggestionRepo.UpdateStatus(c.Context(), suggestionID, "approved", reviewedBy); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to approve suggestion",
		})
	}

	suggestion, _ := h.suggestionRepo.GetByID(c.Context(), suggestionID)
	return c.JSON(suggestion)
}

// RejectSuggestion rejects a suggestion
// POST /api/v1/suggestions/:id/reject
func (h *AutoFixHandler) RejectSuggestion(c *fiber.Ctx) error {
	suggestionID := c.Params("id")

	// TODO: Get reviewer from authenticated user context
	reviewedBy := "system"

	if err := h.suggestionRepo.UpdateStatus(c.Context(), suggestionID, "rejected", reviewedBy); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject suggestion",
		})
	}

	suggestion, _ := h.suggestionRepo.GetByID(c.Context(), suggestionID)
	return c.JSON(suggestion)
}

// ApplySuggestion applies an approved suggestion to the workflow
// POST /api/v1/suggestions/:id/apply
func (h *AutoFixHandler) ApplySuggestion(c *fiber.Ctx) error {
	suggestionID := c.Params("id")

	suggestion, err := h.suggestionRepo.GetByID(c.Context(), suggestionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Suggestion not found",
		})
	}

	if suggestion.Status != "approved" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Suggestion must be approved before applying",
		})
	}

	// Get workflow
	logger.Info("Fetching workflow for suggestion",
		zap.String("suggestion_id", suggestionID),
		zap.String("workflow_id", suggestion.WorkflowID))

	workflow, err := h.workflowRepo.GetByID(c.Context(), suggestion.WorkflowID)
	if err != nil {
		logger.Error("Failed to fetch workflow",
			zap.String("workflow_id", suggestion.WorkflowID),
			zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// 1. Create a new version (Version N+1) with the APPLIED changes
	// We need to modify the workflow config first

	// Find the node to update
	nodeFound := false
	for i, phase := range workflow.Config.Phases {
		for j, node := range phase.Nodes {
			if node.ID == suggestion.NodeID {
				// Update node params with suggested config
				// We merge the suggested config into existing params
				for k, v := range suggestion.SuggestedNodeConfig {
					workflow.Config.Phases[i].Nodes[j].Params[k] = v
				}
				nodeFound = true
				break
			}
		}
		if nodeFound {
			break
		}
	}

	if !nodeFound {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Target node not found in workflow",
		})
	}

	// Create new version
	newVersionNum := workflow.Version + 1
	newVersion := &models.WorkflowVersion{
		WorkflowID:   workflow.ID,
		Version:      newVersionNum,
		Config:       workflow.Config,
		ChangeReason: "AI Auto-Fix: " + suggestion.FixExplanation,
	}

	if err := h.versionRepo.Create(c.Context(), newVersion); err != nil {
		logger.Error("Failed to create workflow version", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create workflow version",
		})
	}

	// Update workflow
	workflow.Version = newVersionNum
	if err := h.workflowRepo.Update(c.Context(), workflow); err != nil {
		logger.Error("Failed to update workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update workflow",
		})
	}

	// Mark suggestion as applied
	if err := h.suggestionRepo.MarkAsApplied(c.Context(), suggestionID); err != nil {
		logger.Error("Failed to mark suggestion as applied", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to apply suggestion",
		})
	}

	logger.Info("Suggestion applied and new version created",
		zap.String("suggestion_id", suggestionID),
		zap.Int("new_version", newVersionNum))

	suggestion, _ = h.suggestionRepo.GetByID(c.Context(), suggestionID)
	return c.JSON(suggestion)
}

// RevertSuggestion reverts an applied suggestion
// POST /api/v1/suggestions/:id/revert
func (h *AutoFixHandler) RevertSuggestion(c *fiber.Ctx) error {
	suggestionID := c.Params("id")

	suggestion, err := h.suggestionRepo.GetByID(c.Context(), suggestionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Suggestion not found",
		})
	}

	if suggestion.Status != "applied" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Can only revert applied suggestions",
		})
	}

	// To revert, we can simply rollback to the previous version?
	// Or we can try to undo the specific change?
	// Since we are using version control now, we should probably find the version BEFORE this change.
	// However, multiple changes might have happened since then.
	// Ideally, we should look at the workflow history.

	// For simplicity in this iteration:
	// We will just fetch the PREVIOUS version (N-1) relative to current workflow state?
	// No, that might revert other things if other changes happened.

	// Better approach:
	// We should have stored the `original_node_config` in the suggestion as per original plan,
	// OR we rely on the version history.
	// Since the user asked for version control, let's use that.
	// But finding the exact previous state might be tricky if we don't know which version it was applied in.
	// We didn't store "AppliedInVersion" in the suggestion.

	// Let's assume for now that we want to revert the *current* workflow state to the state *before* this suggestion was applied.
	// If this suggestion was the LAST change, then N-1 is correct.
	// If other changes happened, reverting might be complex.

	// Let's implement a simple "Undo" logic using the version history if possible,
	// or fall back to the `OriginalNodeConfig` idea if we had implemented it.
	// Wait, I didn't implement `OriginalNodeConfig` in the DB migration for version control plan.
	// The version control plan replaced the `OriginalNodeConfig` plan.

	// So, we should probably just rollback to the previous version of the workflow.
	// But we need to know which version.

	// Let's fetch the current workflow.
	workflow, err := h.workflowRepo.GetByID(c.Context(), suggestion.WorkflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// We will create a new version that restores the state of (CurrentVersion - 1).
	// This assumes the user wants to undo the last action.
	// If the user wants to revert an older suggestion, this logic is flawed without more tracking.
	// But typically "Revert" on a suggestion implies "Undo this specific change".

	// Given the constraints, let's implement a "Rollback to previous version" logic here,
	// assuming the user is reverting immediately or sequentially.

	if workflow.Version <= 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot revert: no previous version",
		})
	}

	prevVersionNum := workflow.Version - 1
	prevVersion, err := h.versionRepo.GetByVersion(c.Context(), workflow.ID, prevVersionNum)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch previous version",
		})
	}

	// Create new version (Revert)
	newVersionNum := workflow.Version + 1
	newVersion := &models.WorkflowVersion{
		WorkflowID:   workflow.ID,
		Version:      newVersionNum,
		Config:       prevVersion.Config,
		ChangeReason: "Revert: " + suggestion.FixExplanation,
	}

	if err := h.versionRepo.Create(c.Context(), newVersion); err != nil {
		logger.Error("Failed to create revert version", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create revert version",
		})
	}

	// Update workflow
	workflow.Config = prevVersion.Config
	workflow.Version = newVersionNum

	if err := h.workflowRepo.Update(c.Context(), workflow); err != nil {
		logger.Error("Failed to update workflow", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update workflow",
		})
	}

	if err := h.suggestionRepo.MarkAsReverted(c.Context(), suggestionID); err != nil {
		logger.Error("Failed to mark suggestion as reverted", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revert suggestion",
		})
	}

	return c.JSON(fiber.Map{"message": "Suggestion reverted successfully"})
}

// readAndExtractPreview reads a DOM snapshot and extracts preview data using a selector
func (h *AutoFixHandler) readAndExtractPreview(domPath string, selector string) (string, error) {
	// Open the DOM file
	f, err := os.Open(domPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var reader io.Reader = f

	// Check if gzipped
	if strings.HasSuffix(domPath, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		reader = gz
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return "", err
	}

	// Extract preview data (up to 3 items, truncated to 200 chars each)
	var previews []string
	doc.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i >= 3 {
			return false
		}
		text := strings.TrimSpace(s.Text())
		if len(text) > 200 {
			text = text[:200] + "..."
		}
		if text != "" {
			previews = append(previews, text)
		}
		return true
	})

	return strings.Join(previews, "\n"), nil
}
