FROM golang:1.25.3 AS app_builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN curl -sL https://taskfile.dev/install.sh | sh
RUN ls -lah
RUN /app/bin/task build

# Build kubectl-ai, to make docker image multiarch
FROM golang:1.25.3 AS kubectl_ai_builder
ARG KUBECTL_AI_VERSION=0.0.27
WORKDIR /app
RUN wget https://github.com/GoogleCloudPlatform/kubectl-ai/archive/refs/tags/v${KUBECTL_AI_VERSION}.tar.gz && \
  tar -xf v${KUBECTL_AI_VERSION}.tar.gz && \
  cd kubectl-ai-${KUBECTL_AI_VERSION} && \
  CGO_ENABLED=0 go build -o /app/kubectl-ai ./cmd

FROM alpine:3.22.2
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /
COPY --from=app_builder /app/dist/k8s-ai-detective .
COPY --from=kubectl_ai_builder /app/kubectl-ai /bin/kubectl-ai
USER nobody
ENTRYPOINT ["/k8s-ai-detective"]