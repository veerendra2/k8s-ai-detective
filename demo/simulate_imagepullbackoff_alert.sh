#!/usr/bin/env bash
# Simulate an ImagePullBackOff pod and generate a mock Alertmanager JSON

set -e

NAMESPACE="default"
DEPLOYMENT="my-app"
IMAGE="myusername/myimage:latest"   # intentionally invalid image
ALERT_FILE="alert.json"
APP_PORT=8085

echo "[*] Checking if deployment $DEPLOYMENT exists in namespace $NAMESPACE..."
if kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" >/dev/null 2>&1; then
  echo "[*] Deployment '$DEPLOYMENT' already exists, skipping creation."
else
  echo "[*] Creating deployment $DEPLOYMENT..."
  kubectl create deployment "$DEPLOYMENT" --image="$IMAGE" -n "$NAMESPACE"
fi

echo "[*] Waiting for pod to be created..."
for i in {1..30}; do
  POD=$(kubectl get pods -n "$NAMESPACE" -l app="$DEPLOYMENT" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  if [ -n "$POD" ]; then
    echo "[*] Found pod: $POD"
    break
  fi
  echo "[*] Pod not yet available, retrying ($i/30)..."
  sleep 2
done

if [ -z "$POD" ]; then
  echo "[*] Timed out waiting for pod from deployment $DEPLOYMENT"
  exit 1
fi

cat <<EOF > "$ALERT_FILE"
{
  "receiver": "all-alerts",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertgroup": "kubernetes-apps",
        "alertname": "KubePodImagePullBackOff",
        "namespace": "$NAMESPACE",
        "pod": "$POD",
        "container": "$DEPLOYMENT",
        "instance": "10.244.12.56:8080",
        "job": "kube-state-metrics",
        "prometheus": "monitoring/vmagent",
        "service": "kube-state-metrics",
        "severity": "critical",
        "staffbase_cluster": "de1",
        "staffbase_env": "dev"
      },
      "annotations": {
        "summary": "Pod $POD in namespace $NAMESPACE is failing to pull its container image.",
        "message": "Pod $NAMESPACE/$POD is in state ImagePullBackOff. Check if the image exists and registry credentials are valid.",
        "runbook": "Check image name and registry credentials:\n1. Run 'kubectl describe pod $POD -n $NAMESPACE'\n2. Verify image URL and pull secrets.\n3. Check node network or registry availability."
      },
      "startsAt": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://localhost:$APP_PORT/vmalert/alert?group_id=1&alert_id=1",
      "fingerprint": "$(date +%s%N | cut -c1-16)"
    }
  ],
  "groupLabels": {
    "alertname": "KubePodImagePullBackOff",
    "namespace": "$NAMESPACE",
    "staffbase_cluster": "de1",
    "staffbase_env": "dev"
  },
  "commonLabels": {
    "alertgroup": "kubernetes-apps",
    "alertname": "KubePodImagePullBackOff",
    "namespace": "$NAMESPACE",
    "pod": "$POD",
    "container": "$DEPLOYMENT",
    "severity": "critical",
    "staffbase_cluster": "de1",
    "staffbase_env": "dev"
  },
  "commonAnnotations": {
    "runbook": "Check image name and registry credentials:\n1. Run 'kubectl describe pod $POD -n $NAMESPACE'\n2. Verify image URL and pull secrets.\n3. Check node network or registry availability."
  },
  "externalURL": "http://localhost:$APP_PORT",
  "version": "4",
  "groupKey": "{}:{alertname=\"KubePodImagePullBackOff\", namespace=\"$NAMESPACE\", staffbase_cluster=\"de1\", staffbase_env=\"dev\"}",
  "truncatedAlerts": 0
}
EOF

echo "[*] Alert JSON generated: $ALERT_FILE"
echo "[*] To test your app, run:"
echo "curl -X POST -H \"Content-Type: application/json\" -d @$ALERT_FILE http://localhost:$APP_PORT/alert"

read -p "[*] Send the alert now? (y/N): " confirm
if [[ "$confirm" =~ ^[Yy]$ ]]; then
  echo "[*] Sending alert to http://localhost:$APP_PORT/alert ..."
  curl -s -o /dev/null -w "%{http_code}\n" -X POST -H "Content-Type: application/json" -d @"$ALERT_FILE" http://localhost:$APP_PORT/alert
  echo "[*] Alert sent."
else
  echo "[*] Skipped sending alert."
fi
