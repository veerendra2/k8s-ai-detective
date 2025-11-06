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

const promptTpl1 = `The following JSON, {{ .AlertJSON }}, contains details of an alert from Alertmanager.
Use the information in the "labels" field such as "namespace", "pod", "container", and other relevant
identifiers to investigate the issue. Perform only basic diagnostic reasoning: review logs, events, and
errors related to the affected resources. Do not make any modifications, write any data, or ask for user
confirmation. Identify the affected resources, check recent logs and Kubernetes events for errors,
warnings, or anomalies, and determine the likely cause of the alert. Output only a short summary of your
findings, no more than three to five lines. Do not include any reasoning, explanation of your thought
process, or markdown formatting. The output must be concise, factual, and directly actionable, describing
only the key issues or anomalies related to the alert.
`

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

	// Parse the prompt template from constants
	tmpl, err := template.New("prompt").Parse(promptTpl1)
	if err != nil {
		slog.Error("Failed to parse prompt template", "worker_id", id, "error", err)
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
