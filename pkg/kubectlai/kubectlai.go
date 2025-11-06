package kubectlai

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	LlmProvider      string `name:"llm-provider" help:"Language model provider" env:"LLM_PROVIDER" default:"gemini"`
	LlmProviderModel string `name:"llm-provider-model" help:"LLM provider's model name" env:"LLM_PROVIDER_MODEL" default:"gemini-2.5-pro"`
	APIKey           string `name:"api-key" help:"API key of the llm-provider you set for authentication" env:"API_KEY" required:""`
	Kubeconfig       string `name:"kubeconfig" help:"Path to kubeconfig file (uses in-cluster config if not set)" env:"KUBECONFIG" default:""`
}

type Client interface {
	RunQuietPrompt(ctx context.Context, prompt string) (string, error)
}

type client struct {
	kubeconfigPath   string
	llmProvider      string
	llmProviderModel string
	apiKey           string
}

func (c *client) RunQuietPrompt(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"kubectl-ai",
		"--quiet",
		"--skip-permissions",
		"--llm-provider", c.llmProvider,
		"--model", c.llmProviderModel,
		"--kubeconfig", c.kubeconfigPath,
		prompt,
	)

	// Create minimal environment
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/tmp",
		fmt.Sprintf("%s_API_KEY=%s", strings.ToUpper(c.llmProvider), c.apiKey),
		"KUBECONFIG=" + c.kubeconfigPath,
	}

	// Set working directory to /tmp
	cmd.Dir = "/tmp"

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	slog.Debug("Executing...", "command", strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		slog.Error("kubectl-ai execution failed", "error", err.Error(), "stderr", stderr.String())
		return "", fmt.Errorf("kubectl-ai failed: %w", err)
	}

	return out.String(), nil
}

func NewClient(config Config) (Client, error) {
	var kubeconfigPath string

	// Check kubeconfig file exists, if KUBECONFIG is set
	if config.Kubeconfig != "" {
		// Expand tilde to home directory
		expandedPath := config.Kubeconfig
		if strings.HasPrefix(config.Kubeconfig, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			expandedPath = filepath.Join(homeDir, config.Kubeconfig[2:])
		}

		info, err := os.Stat(expandedPath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, fmt.Errorf("kubeconfig file path should not be a directory")
		}

		slog.Info("Using kubeconfig", "file", expandedPath)
		kubeconfigPath = expandedPath

	} else {
		// Else, create kubeconfig file from in-cluster kubeconfig
		// More Info: https://stackoverflow.com/a/73461820/2200798
		// Because, kubectl-ai binary only accepts kubeconfig file
		restCfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error getting in-cluster config: %w", err)
		}

		apiCfg := api.NewConfig()
		apiCfg.Clusters["in-cluster"] = &api.Cluster{
			Server:                   restCfg.Host,
			CertificateAuthority:     restCfg.CAFile,
			CertificateAuthorityData: restCfg.CAData,
		}
		apiCfg.AuthInfos["in-cluster-user"] = &api.AuthInfo{
			Token: restCfg.BearerToken,
		}
		apiCfg.Contexts["in-cluster-context"] = &api.Context{
			Cluster:  "in-cluster",
			AuthInfo: "in-cluster-user",
		}
		apiCfg.CurrentContext = "in-cluster-context"

		// NOTE: This '/cache' dir can be mounted as emptyDir volume in the pod
		kubeconfigPath = filepath.Join("/cache", "incluster.kubeconfig")
		if err := clientcmd.WriteToFile(*apiCfg, kubeconfigPath); err != nil {
			return nil, fmt.Errorf("error writing kubeconfig: %w", err)
		}

		slog.Info("Created in-cluster kubeconfig", "file", kubeconfigPath)
	}

	return &client{
		kubeconfigPath:   kubeconfigPath,
		llmProvider:      config.LlmProvider,
		llmProviderModel: config.LlmProviderModel,
		apiKey:           config.APIKey,
	}, nil
}
