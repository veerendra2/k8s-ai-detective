package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/veerendra2/gopackages/slogger"
	"github.com/veerendra2/gopackages/version"
	"github.com/veerendra2/k8s-ai-detective/internal/alertwebhook"
	"github.com/veerendra2/k8s-ai-detective/internal/processor"
	"github.com/veerendra2/k8s-ai-detective/pkg/httpserver"
	ai "github.com/veerendra2/k8s-ai-detective/pkg/kubectlai"
)

const AppName = "k8s-ai-detective"

var cli struct {
	Address string `env:"ADDRESS" default:":8080" help:"The address where the server should listen on."`

	KubectlAi ai.Config `embed:""`

	Processor processor.Config `embed:""`

	Log slogger.Config `embed:"" prefix:"log." envprefix:"LOG_"`
}

func main() {
	kongCtx := kong.Parse(&cli,
		kong.Name(AppName),
		kong.Description("Receives alerts, runs kubectl-ai for debugging, summarizes, and sends reports to Slack."),
	)

	kongCtx.FatalIfErrorf(kongCtx.Error)

	slog.SetDefault(slogger.New(cli.Log))

	slog.Info("Version information", version.Info()...)
	slog.Info("Build context", version.BuildContext()...)

	// Initialize AI client
	aiClient, err := ai.NewClient(cli.KubectlAi)
	if err != nil {
		slog.Error("Failed to create AI client", "error", err)
		kongCtx.Exit(1)
	}

	// Verify AI client is working
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	res, err := aiClient.RunQuietPrompt(ctx, "Ping test — short reply only, no emojis")
	if err != nil {
		slog.Error("kubectl-ai test failed", "error", err)
		kongCtx.Exit(1)
	}
	slog.Info("kubectl-ai is working...", "response", strings.TrimSpace(res))

	// Initialize processor
	processorClient, err := processor.NewClient(cli.Processor, aiClient)
	if err != nil {
		slog.Error("Failed to create processor client", "error", err)
		kongCtx.Exit(1)
	}

	// Start processor
	if err := processorClient.Start(ctx); err != nil {
		slog.Error("Failed to start processor", "error", err)
		kongCtx.Exit(1)
	}
	slog.Info("Processor started")

	// Initialize HTTP server
	mux := http.NewServeMux()
	alertHandler := alertwebhook.NewHandler(processorClient)
	http.HandleFunc("/alert", alertHandler.HandleAlerts)
	httpServer := httpserver.New(cli.Address, mux)

	// Start HTTP server
	httpServer.Start()

	// Graceful shutdown handling
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	slog.Debug("Start listening for SIGINT and SIGTERM signal.")
	<-done
	slog.Info("Shutdown started.")

	// Shutdown all components
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Stop(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server shutdown complete")
	}

	// Shutdown processor
	processorClient.Shutdown(shutdownCtx)
	slog.Info("Processor shutdown complete")

	slog.Info("Shutdown done.")
}
