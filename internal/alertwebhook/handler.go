package alertwebhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// HandleAlerts processes Alertmanager webhook POSTs
func HandleAlerts(w http.ResponseWriter, r *http.Request) {
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

	var msg WebhookMessage

	if err := dec.Decode(&msg); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	fmt.Println(msg)
	// TODO Push the the alert from alerts array in for loop
}
