package config

import (
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	IncludeAlertGroups []string `yaml:"include_alert_groups" validate:"required_without=IncludeNamespace,min=1"`
	IncludeNamespace   []string `yaml:"include_namespace" validate:"required_without=IncludeAlertGroups,min=1"`
}

// LoadConfig loads and validates the configuration from the given YAML file.
func LoadConfig(filePath string) (*AppConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	config := &AppConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// Validate the configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, err
	}

	return config, nil
}
