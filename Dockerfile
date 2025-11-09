FROM golang:1.25.3 AS app_builder
WORKDIR /app
RUN curl -sL https://taskfile.dev/install.sh | sh
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN /app/bin/task build

# Build kubectl-ai and install kubectl via apt-get to make these
# binaries available in final "multiarch" docker image
FROM golang:1.25.3 AS kubectl_ai_builder
ARG KUBECTL_AI_VERSION=0.0.27
WORKDIR /app
RUN apt-get update && \
  apt-get install -y apt-transport-https ca-certificates curl gnupg && \
  curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.34/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg && \
  chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg && \
  echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.34/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list && \
  chmod 644 /etc/apt/sources.list.d/kubernetes.list && \
  apt-get update && \
  apt-get install -y kubectl
RUN wget https://github.com/GoogleCloudPlatform/kubectl-ai/archive/refs/tags/v${KUBECTL_AI_VERSION}.tar.gz && \
  tar -xf v${KUBECTL_AI_VERSION}.tar.gz && \
  cd kubectl-ai-${KUBECTL_AI_VERSION} && \
  CGO_ENABLED=0 go build -o /app/kubectl-ai ./cmd

FROM alpine:3.22.2
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /
COPY --from=app_builder /app/dist/k8s-ai-detective .
COPY --from=kubectl_ai_builder /app/kubectl-ai /bin/kubectl-ai
COPY --from=kubectl_ai_builder /usr/bin/kubectl /bin/kubectl
USER nobody
ENTRYPOINT ["/k8s-ai-detective"]