package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IncludeAlertGroups []string `yaml:"include_alert_groups"`
	IncludeNamespace   []string `yaml:"include_namespace"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
