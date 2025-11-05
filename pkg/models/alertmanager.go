package models

import (
	"time"
)

// https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
type WebhookMessage struct {
	Version           string         `json:"version"`
	GroupKey          string         `json:"groupKey"`
	TruncatedAlerts   int            `json:"truncatedAlerts"`
	Status            string         `json:"status"`
	Receiver          string         `json:"receiver"`
	GroupLabels       map[string]any `json:"groupLabels"`
	CommonLabels      map[string]any `json:"commonLabels"`
	CommonAnnotations map[string]any `json:"commonAnnotations"`
	ExternalURL       string         `json:"externalURL"`
	Alerts            []Alert        `json:"alerts"`
}

type Alert struct {
	Status       string         `json:"status"`
	Labels       map[string]any `json:"labels"`
	Annotations  map[string]any `json:"annotations"`
	StartsAt     time.Time      `json:"startsAt"`
	EndsAt       time.Time      `json:"endsAt"`
	GeneratorURL string         `json:"generatorURL"`
	Fingerprint  string         `json:"fingerprint"`
}
