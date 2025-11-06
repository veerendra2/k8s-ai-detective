package alertwebhook

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veerendra2/k8s-ai-detective/internal/processor"
	"github.com/veerendra2/k8s-ai-detective/pkg/models"
)

type Handler struct {
	Processor processor.Client
}

func NewHandler(processor processor.Client) *Handler {
	return &Handler{Processor: processor}
}

// HandleAlerts processes Alertmanager webhook POSTs
func (h *Handler) AlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return
		}
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var msg models.WebhookMessage

	if err := dec.Decode(&msg); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	slog.Debug("Received webhook message", "alert", msg)
	if err := h.Processor.Push(msg); err != nil {
		slog.Error("Failed to push the alerts to queue", "error", err)
	}
}

// HealthHandler returns the health status of the service
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}
