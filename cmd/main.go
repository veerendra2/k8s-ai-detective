package main

import (
	"context"
	"errors"
	"log"
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
	ai "github.com/veerendra2/k8s-ai-detective/pkg/kubectlai"
)

const AppName = "k8s-ai-detective"

var cli struct {
	Address string `env:"ADDRESS" default:":8080" help:"The address where the server should listen on."`

	KubectlAi ai.Config `embed:""`

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

	aiClient, err := ai.NewClient(cli.KubectlAi)
	if err != nil {
		slog.Error("Failed to create AI client", "error", err)
		kongCtx.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Verify AI is working, fail-fast in case we can't run kubectl-ai
	res, err := aiClient.RunQuietPrompt(ctx, "Ping test — short reply only, no emojis")
	if err != nil {
		slog.Error("kubectl-ai test failed", "error", err)
		kongCtx.Exit(1)
	}
	slog.Info("kubectl-ai is working...", "response", strings.TrimSpace(res))

	// ------------------------ HTTP SERVER ------------------------
	mux := http.NewServeMux()
	mux.HandleFunc("/alert", alertwebhook.HandleAlerts)

	server := &http.Server{
		Addr:         cli.Address,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server died unexpected.", slog.Any("error", err))
		}
		slog.Error("Server stopped.")
	}()
	// ------------------------------------------------------

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	slog.Debug("Start listening for SIGINT and SIGTERM signal.")
	<-done
	slog.Info("Shutdown started.")

	sdCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(sdCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}

	slog.Info("Shutdown done.")
}
