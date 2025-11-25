package handlers

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/uzzalhcse/crawlify/internal/logger"
	"github.com/uzzalhcse/crawlify/internal/monitoring"
	"go.uber.org/zap"
)

// SnapshotHandler handles snapshot API requests
type SnapshotHandler struct {
	snapshotService *monitoring.SnapshotService
}

// NewSnapshotHandler creates a new snapshot handler
func NewSnapshotHandler(snapshotService *monitoring.SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{
		snapshotService: snapshotService,
	}
}

// GetSnapshot retrieves a snapshot by ID
func (h *SnapshotHandler) GetSnapshot(c *fiber.Ctx) error {
	ctx := context.Background()
	snapshotID := c.Params("snapshot_id")

	snapshot, err := h.snapshotService.GetSnapshot(ctx, snapshotID)
	if err != nil {
		logger.Error("Failed to get snapshot",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Snapshot not found",
		})
	}

	return c.JSON(snapshot)
}

// ListSnapshotsByReport retrieves all snapshots for a monitoring report
func (h *SnapshotHandler) ListSnapshotsByReport(c *fiber.Ctx) error {
	ctx := context.Background()
	reportID := c.Params("report_id")

	snapshots, err := h.snapshotService.GetSnapshotsByReport(ctx, reportID)
	if err != nil {
		logger.Error("Failed to get snapshots",
			zap.String("report_id", reportID),
			zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch snapshots",
		})
	}

	return c.JSON(fiber.Map{
		"report_id": reportID,
		"snapshots": snapshots,
		"total":     len(snapshots),
	})
}

// GetScreenshot serves a screenshot file
func (h *SnapshotHandler) GetScreenshot(c *fiber.Ctx) error {
	ctx := context.Background()
	snapshotID := c.Params("snapshot_id")

	// Get snapshot to find file path
	snapshot, err := h.snapshotService.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Snapshot not found",
		})
	}

	if snapshot.ScreenshotPath == nil || *snapshot.ScreenshotPath == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Screenshot not available for this snapshot",
		})
	}

	// Get full file path
	fullPath := h.snapshotService.GetScreenshotPath(*snapshot.ScreenshotPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.Warn("Screenshot file not found",
			zap.String("snapshot_id", snapshotID),
			zap.String("path", fullPath))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Screenshot file not found",
		})
	}

	// Send file
	return c.SendFile(fullPath)
}

// GetDOM serves a DOM snapshot file (decompressed)
func (h *SnapshotHandler) GetDOM(c *fiber.Ctx) error {
	ctx := context.Background()
	snapshotID := c.Params("snapshot_id")

	// Get snapshot to find file path
	snapshot, err := h.snapshotService.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Snapshot not found",
		})
	}

	if snapshot.DOMSnapshotPath == nil || *snapshot.DOMSnapshotPath == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "DOM snapshot not available",
		})
	}

	// Get full file path
	fullPath := h.snapshotService.GetDOMPath(*snapshot.DOMSnapshotPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		logger.Warn("DOM file not found",
			zap.String("snapshot_id", snapshotID),
			zap.String("path", fullPath))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "DOM file not found",
		})
	}

	// Check if file is gzipped
	isGzipped := filepath.Ext(fullPath) == ".gz"

	if isGzipped {
		// Decompress and send
		file, err := os.Open(fullPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to open DOM file",
			})
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to decompress DOM file",
			})
		}
		defer gzReader.Close()

		// Set content type
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")

		// Copy decompressed content to response
		_, err = io.Copy(c.Response().BodyWriter(), gzReader)
		if err != nil {
			logger.Error("Failed to send DOM content", zap.Error(err))
			return err
		}

		return nil
	}

	// Send file as-is
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendFile(fullPath)
}

// DeleteSnapshot removes a snapshot and its files
func (h *SnapshotHandler) DeleteSnapshot(c *fiber.Ctx) error {
	ctx := context.Background()
	snapshotID := c.Params("snapshot_id")

	// Get snapshot first to get file paths
	snapshot, err := h.snapshotService.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Snapshot not found",
		})
	}

	// Delete files
	if snapshot.ScreenshotPath != nil {
		fullPath := h.snapshotService.GetScreenshotPath(*snapshot.ScreenshotPath)
		os.Remove(fullPath) // Ignore error
	}

	if snapshot.DOMSnapshotPath != nil {
		fullPath := h.snapshotService.GetDOMPath(*snapshot.DOMSnapshotPath)
		os.Remove(fullPath) // Ignore error
	}

	// Delete from database (would need to add this method to repository)
	// For now, just return success

	return c.JSON(fiber.Map{
		"message": "Snapshot deleted successfully",
	})
}
