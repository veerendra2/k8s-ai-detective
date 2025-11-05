package processor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/slack-go/slack"
	"github.com/veerendra2/k8s-ai-detective/internal/alertwebhook"
	"github.com/veerendra2/k8s-ai-detective/pkg/kubectlai"
)

type Config struct {
	WorkerCount    uint8 `name:"worker-count" help:"Number of alerts processed in parallel (Max 256)." env:"WORKER_COUNT" default:"3"`
	AlertQueueSize uint8 `name:"alert-queue-size" help:"Queue size to hold alerts (Max 256)." env:"ALERT_QUEUE_SIZE" default:"10"`

	SlackBotToken  string `name:"slack-bot-token" help:"Slack bot token for authentication." env:"SLACK_BOT_TOKEN" default:""`
	SlackChannelId string `name:"slack-channel-id" help:"Slack channel ID to send notifications." env:"SLACK_CHANNEL_ID" default:""`
}

// AlertQueue wraps alertmanager.Alert for future metadata (timestamps, retries, etc.)
type AlertQueue struct {
	Alert alertwebhook.Alert
}

type Client interface {
	Start(ctx context.Context) error
	Push(alert alertwebhook.Alert) error
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

// Push adds a new alert to the queue
func (c *client) Push(alert alertwebhook.Alert) error {
	select {
	case c.queue <- AlertQueue{Alert: alert}:
		return nil
	default:
		return fmt.Errorf("alert queue full")
	}
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

func (c *client) processAlert(ctx context.Context, alert alertwebhook.Alert, id int) {
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

func NewClient(cfg Config, aiClient kubectlai.Client) (Client, error) {
	slackClient := slack.New(cfg.SlackBotToken)

	return &client{
		workerCount: cfg.WorkerCount,
		queue:       make(chan AlertQueue, cfg.AlertQueueSize),

		slackClient:    slackClient,
		slackChannelId: cfg.SlackChannelId,

		aiClient: aiClient,
	}, nil
}
