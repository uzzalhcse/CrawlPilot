package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// SlackNotifier sends notifications to Slack via webhooks
type SlackNotifier struct{}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier() *SlackNotifier {
	return &SlackNotifier{}
}

// Send sends a monitoring notification to Slack
func (n *SlackNotifier) Send(config *models.SlackConfig, report *models.MonitoringReport, workflowName string) error {
	if config == nil || config.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL is required")
	}

	message := n.formatMessage(report, workflowName)
	payload := map[string]interface{}{
		"text":    message.Text,
		"blocks":  message.Blocks,
		"channel": config.Channel,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

type slackMessage struct {
	Text   string        `json:"text"`
	Blocks []interface{} `json:"blocks"`
}

func (n *SlackNotifier) formatMessage(report *models.MonitoringReport, workflowName string) *slackMessage {
	// Determine emoji based on status
	emoji := ":white_check_mark:"
	statusText := "Healthy"

	switch report.Status {
	case models.MonitoringStatusFailed:
		emoji = ":x:"
		statusText = "Failed"
	case models.MonitoringStatusDegraded:
		emoji = ":warning:"
		statusText = "Degraded"
	}

	// Create message blocks
	blocks := []interface{}{
		map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": fmt.Sprintf("%s Health Check: %s", emoji, statusText),
			},
		},
		map[string]interface{}{
			"type": "section",
			"fields": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Workflow:*\n%s", workflowName),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Status:*\n%s", statusText),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Duration:*\n%dms", report.Duration),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Report ID:*\n`%s`", report.ID),
				},
			},
		},
	}

	// Add summary if available
	if report.Summary != nil {
		summaryText := fmt.Sprintf(
			"✅ Passed: %d | ⚠️ Warnings: %d | ❌ Failed: %d",
			report.Summary.PassedNodes,
			report.Summary.WarningNodes,
			report.Summary.FailedNodes,
		)

		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Summary:*\n%s", summaryText),
			},
		})

		// Add critical issues if any
		if len(report.Summary.CriticalIssues) > 0 {
			issuesText := "*Critical Issues:*\n"
			for i, issue := range report.Summary.CriticalIssues {
				if i >= 3 {
					issuesText += fmt.Sprintf("_...and %d more_", len(report.Summary.CriticalIssues)-3)
					break
				}
				issuesText += fmt.Sprintf("• `%s`: %s\n", issue.Code, issue.Message)
			}

			blocks = append(blocks, map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": issuesText,
				},
			})
		}
	}

	// Add divider and timestamp
	blocks = append(blocks,
		map[string]interface{}{"type": "divider"},
		map[string]interface{}{
			"type": "context",
			"elements": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("Executed at %s", report.StartedAt.Format(time.RFC1123)),
				},
			},
		},
	)

	plainText := fmt.Sprintf("%s Health Check %s for workflow '%s'", emoji, statusText, workflowName)

	return &slackMessage{
		Text:   plainText,
		Blocks: blocks,
	}
}
