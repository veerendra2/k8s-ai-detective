# K8s AI Detective

> ⚠️ _\*CAUTION: This tool was created out of curiosity during a hackathon. It hasn’t been thoroughly tested and still requires improvements._

K8s AI Detective is a tool designed to automate debugging and summarizing issues when an alert is triggered. It leverages [`kubectl-ai`](https://github.com/GoogleCloudPlatform/kubectl-ai) to analyze the alert context, gather relevant information (such as logs, events, and resource states), and generate an initial summary.

<center>
  <img src="./assets/logo.png" alt="logo" width="90"/>
</center>

## Usage

```bash
Usage: k8s-ai-detective --api-key=STRING [flags]

K8s AI Detective automates debugging and summarizing alerts by leveraging `kubectl-ai` to analyze context, gather logs, events, and resource states, and generate an initial summary.

Flags:
  -h, --help                                   Show context-sensitive help.
      --address=":8080"                        The address where the server should listen on ($ADDRESS).
      --config-file-path="./config.yml"        Config file path ($CONFIG_FILE_PATH).
      --llm-provider="gemini"                  Language model provider ($LLM_PROVIDER)
      --llm-provider-model="gemini-2.5-pro"    LLM provider's model name ($LLM_PROVIDER_MODEL)
      --api-key=STRING                         API key of the llm-provider you set for authentication ($API_KEY)
      --kubeconfig=""                          Path to kubeconfig file (uses in-cluster config if not set) ($KUBECONFIG)
      --worker-count=3                         Number of alerts processed in parallel (Max 256) ($WORKER_COUNT).
      --alert-queue-size=10                    Queue size to hold alerts (Max 256) ($ALERT_QUEUE_SIZE).
      --slack-bot-token=""                     Slack bot token for authentication ($SLACK_BOT_TOKEN).
      --slack-channel-id=""                    Slack channel ID to send notifications ($SLACK_CHANNEL_ID).
      --log.format="json"                      Set the output format of the logs. Must be "console" or "json" ($LOG_FORMAT).
      --log.level=INFO                         Set the log level. Must be "DEBUG", "INFO", "WARN" or "ERROR" ($LOG_LEVEL).
      --log.add-source                         Whether to add source file and line number to log records ($LOG_ADD_SOURCE).
```

## How it Works?

![Diagram](./assets/workflow.png)

## Run Locally

1. Start the server

```bash
# Export envs
export API_KEY="REDACTED"
export KUBECONFIG="~/.kube/config"
export SLACK_BOT_TOKEN="REDACTED"
export SLACK_CHANNEL_ID="REDACTED"

# Run app locally
task run
time=2025-11-06T19:09:05+01:00 level=INFO source=/k8s-ai-detective/cmd/k8s-ai-detective/main.go:47 msg="Version information" version="" branch="" revision=""
time=2025-11-06T19:09:05+01:00 level=INFO source=/k8s-ai-detective/cmd/k8s-ai-detective/main.go:48 msg="Build context" go_version=go1.25.3 user="" date=""
time=2025-11-06T19:09:05+01:00 level=INFO source=/k8s-ai-detective/pkg/kubectlai/kubectlai.go:96 msg="Using kubeconfig" file=/Users/veerendra.kakumanu/.kube/config
time=2025-11-06T19:09:08+01:00 level=INFO source=/k8s-ai-detective/cmd/k8s-ai-detective/main.go:72 msg="kubectl-ai is working..." response=pong
time=2025-11-06T19:09:08+01:00 level=INFO source=/k8s-ai-detective/pkg/httpserver/httpserver.go:28 msg="Starting HTTP server" address=:8085
```

2. Send an example alert using `curl`

```bash
curl -X POST -H "Content-Type: application/json" -d @assets/example_alert.json http://localhost:8085/alert
```

3. It should able to send summarized info to the slack channel

## Alertmanager Config

Configure alertmanager to send alerts to `k8s-ai-detective` like below

```yaml
receivers:
  - name: "all-alerts"
    webhook_configs:
      - url: "https://k8s-ai-detective/alert"
        send_resolved: true
```

## Build & Test

- Using [Taskfile](https://taskfile.dev/)

_Install Taskfile: [Installation Guide](https://taskfile.dev/docs/installation)_

```bash
# List available tasks
task --list
task: Available tasks for this project:
* all:                   Run comprehensive checks: format, lint, security and test
* build:                 Build the application binary for the current platform
* build-docker:          Build Docker image
* build-platforms:       Build the application binaries for multiple platforms and architectures
* fmt:                   Formats all Go source files
* install:               Install required tools and dependencies
* lint:                  Run static analysis and code linting using golangci-lint
* run:                   Runs the main application
* security:              Run security vulnerability scan
* test:                  Runs all tests in the project      (aliases: tests)
* vet:                   Examines Go source code and reports suspicious constructs

# Build the application
task build

# Run tests
task test
```

- Build with [goreleaser](https://goreleaser.com/)

_Install GoReleaser: [Installation Guide](https://goreleaser.com/install/)_

```bash
# Build locally
goreleaser release --snapshot --clean
...
```

## Further Development

- [ ] Add contextual logging using `slog`
- [ ] Improve alert de-duplication with `fingerprint` during processing
- [ ] Expand configuration options
  - [ ] Support excluding or including specific alerts
  - [ ] Allow dedicated prompts for selected alerts
  - [ ] Enable exclusion of alert groups
  - [ ] Support excluding specific namespaces
- [ ] Add metrics collection and reporting

## References

- [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai)
- [Understanding the context package in golang](https://medium.com/@parikshit/understanding-the-context-package-in-golang-b1392c821d14)
- [Graceful Shutdown in Go: Practical Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/)
- [How to parse a JSON request body in Go](https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body)
- [slack-go -- Send message to Slack channel](https://github.com/slack-go/slack/blob/master/examples/messages/messages.go)
- [Stackoverflow -- different about withcancel and withtimeout in golang's context](https://stackoverflow.com/q/56721676/2200798)
- [Code Snippet -- How to use `kubectl-ai` natively in Go](https://gist.github.com/veerendra2/160533bfce722cf3d853bf500bc8f407)
