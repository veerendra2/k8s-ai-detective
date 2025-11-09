package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/veerendra2/k8s-ai-detective/pkg/models"
)

const promptTpl1 = `The following JSON, {{ .AlertJSON }}, contains details of an alert from Alertmanager.
	Use the information in the "labels" field such as "namespace", "pod", "container", and other relevant
	identifiers to investigate the issue. Perform only basic diagnostic reasoning: review logs, events, and
	errors related to the affected resources. Do not make any modifications, write any data, or ask for user
	confirmation. Identify the affected resources, check recent logs and Kubernetes events for errors,
	warnings, or anomalies, and determine the likely cause of the alert.

	Output only a short summary of your findings, no more than three to five lines. The summary must start with
	the delimiter "-----" (exactly five hyphens) to clearly indicate the beginning of the summary. Do not include
	any reasoning, explanation of your thought process, or markdown formatting. The output must be concise,
	factual, and directly actionable, describing only the key issues or anomalies related to the alert.
`

func (c *client) worker(ctx context.Context, id int) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker shutting down", "worker_id", id)
			return
		case item, ok := <-c.queue:
			if !ok {
				slog.Info("Alert queue closed, worker exiting", "worker_id", id)
				return
			}

			// Ensure alert processing completes within the specified timeout
			workerCtx, cancel := context.WithTimeout(ctx, c.workerTimeout)
			err := c.processAlert(workerCtx, item.Alert, id)
			if err != nil {
				slog.Error("Errors during processing the alert", "worker_id", id, "error", err)
			}
			cancel()
		}
	}
}

func (c *client) processAlert(ctx context.Context, alert models.Alert, id int) error {
	// Serialize the alert to JSON
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to parse alert json: %w", err)
	}

	// Parse the prompt template from constants
	tmpl, err := template.New("prompt").Parse(promptTpl1)
	if err != nil {
		return fmt.Errorf("failed to parse prompt template: %w", err)
	}

	// Fill the template with the alert JSON
	var promptBuffer bytes.Buffer
	err = tmpl.Execute(&promptBuffer, map[string]string{
		"AlertJSON": string(alertJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to execute prompt template: %w", err)
	}

	// Run the AI prompt
	prompt := promptBuffer.String()
	output, err := c.aiClient.RunQuietPrompt(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to run kubectl-ai: %w", err)
	}

	// Send the output to the Slack channel
	attachment := slack.Attachment{
		Pretext: "Alert Summary",
		Text:    output,
	}

	respChannelId, timestamp, err := c.slackClient.PostMessage(
		c.slackChannelId,
		slack.MsgOptionText("AI-Generated Alert Summary", false),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack channel: %w", err)
	}

	slog.Info("Post message to Slack", "worker_id", id, "channel_id", respChannelId, "timestamp", timestamp)

	return nil
}
