package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/veerendra2/k8s-ai-detective/pkg/models"
)

// worker processes alerts sequentially
func (c *client) worker(ctx context.Context, id int) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-c.queue:
			if !ok {
				return
			}
			c.processAlert(ctx, item.Alert, id)
		}
	}
}

func (c *client) processAlert(ctx context.Context, alert models.Alert, id int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered from panic in processAlert", "worker_id", id, "error", r)
		}
	}()

	// Serialize the alert to JSON
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		slog.Error("Failed to marshal alert to JSON", "worker_id", id, "error", err)
		return
	}

	// Load the prompt template
	tmpl, err := template.ParseFiles("templates/prompt.tmpl")
	if err != nil {
		slog.Error("Failed to load prompt template", "worker_id", id, "error", err)
		return
	}

	// Fill the template with the alert JSON
	var promptBuffer bytes.Buffer
	err = tmpl.Execute(&promptBuffer, map[string]string{
		"AlertJSON": string(alertJSON),
	})
	if err != nil {
		slog.Error("Failed to execute prompt template", "worker_id", id, "error", err)
		return
	}

	// Run the AI prompt
	prompt := promptBuffer.String()
	output, err := c.aiClient.RunQuietPrompt(ctx, prompt)
	if err != nil {
		slog.Error("Failed to run AI prompt", "worker_id", id, "error", err)
		return
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
		slog.Error("Failed to send message to Slack", "worker_id", id, "error", err)
		return
	}

	slog.Info("Sent message to Slack", "worker_id", id, "channel_id", respChannelId, "timestamp", timestamp)
}
