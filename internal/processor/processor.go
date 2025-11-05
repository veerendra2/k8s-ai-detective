package processor

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/slack-go/slack"
	"github.com/veerendra2/k8s-ai-detective/internal/config"
	"github.com/veerendra2/k8s-ai-detective/pkg/kubectlai"
	"github.com/veerendra2/k8s-ai-detective/pkg/models"
)

type Config struct {
	WorkerCount    uint8 `name:"worker-count" help:"Number of alerts processed in parallel (Max 256)." env:"WORKER_COUNT" default:"3"`
	AlertQueueSize uint8 `name:"alert-queue-size" help:"Queue size to hold alerts (Max 256)." env:"ALERT_QUEUE_SIZE" default:"10"`

	SlackBotToken  string `name:"slack-bot-token" help:"Slack bot token for authentication." env:"SLACK_BOT_TOKEN" default:""`
	SlackChannelId string `name:"slack-channel-id" help:"Slack channel ID to send notifications." env:"SLACK_CHANNEL_ID" default:""`

	ConfigFilePath string `name:"config-file-path" help:"Config file path." env:"CONFIG_FILE_PATH" default:"./config.yml"`
}

// AlertQueue wraps alertmanager.Alert for future metadata (timestamps, retries, etc.)
type AlertQueue struct {
	Alert models.Alert
}

type Client interface {
	Start(ctx context.Context) error
	Push(webhookMsg models.WebhookMessage) error
	Shutdown(ctx context.Context)
}

type client struct {
	workerCount uint8

	queue  chan AlertQueue
	wg     sync.WaitGroup
	cancel func()

	slackClient    *slack.Client
	slackChannelId string

	aiClient kubectlai.Client

	appConfig *config.Config
}

// Start launches the worker pool
func (c *client) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	for i := 0; i < int(c.workerCount); i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}
	return nil
}

// Shutdown gracefully stops workers
func (c *client) Shutdown(ctx context.Context) {
	if c.cancel != nil {
		c.cancel()
	}
	close(c.queue)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Push adds a new alert to the queue
func (c *client) Push(webhookMsg models.WebhookMessage) error {
	for _, alert := range webhookMsg.Alerts {
		// Ensure "alertgroup" is a string and retrieve its value
		alertGroup, ok := alert.Labels["alertgroup"].(string)
		if !ok {
			slog.Info("Discarding the alert: invalid or missing alertgroup")
			continue
		}

		// Ensure "namespace" is a string and retrieve its value
		namespace, ok := alert.Labels["namespace"].(string)
		if !ok {
			slog.Info("Discarding the alert: invalid or missing namespace")
			continue
		}

		// Discard the alert if the namespace or alert group is not in the config
		if !slices.Contains(c.appConfig.IncludeAlertGroups, alertGroup) || !slices.Contains(c.appConfig.IncludeNamespace, namespace) {
			slog.Info("Discarding the alert: not included in config", "alertgroup", alertGroup, "namespace", namespace)
			continue
		}

		select {
		case c.queue <- AlertQueue{Alert: alert}:
		default:
			return fmt.Errorf("alert queue full")
		}
	}

	return nil
}

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
	// TODO set context timeout!
	c.aiClient.RunQuietPrompt(ctx, "This is a test!")

	// Send Message to slack channel
	// Example: https://github.com/slack-go/slack/blob/master/examples/messages/messages.go
	attachment := slack.Attachment{
		Pretext: "some pretext",
		Text:    "some text",
	}

	respChannelId, timestamp, err := c.slackClient.PostMessage(
		c.slackChannelId,
		slack.MsgOptionText("Some text", false),
		slack.MsgOptionAttachments(attachment),
	)
	if err != nil {
		slog.Error("Failed to set message to slack", any(err))
		return
	}

	slog.Info("Sent message to slack", "channel_id", respChannelId, "sent_timestamp", timestamp)
}

func NewClient(cfg Config, aiClient kubectlai.Client) (Client, error) {
	slackClient := slack.New(cfg.SlackBotToken)

	appCfg, err := config.LoadConfig(cfg.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	return &client{
		workerCount: cfg.WorkerCount,
		queue:       make(chan AlertQueue, cfg.AlertQueueSize),

		slackClient:    slackClient,
		slackChannelId: cfg.SlackChannelId,

		aiClient: aiClient,

		appConfig: appCfg,
	}, nil
}
