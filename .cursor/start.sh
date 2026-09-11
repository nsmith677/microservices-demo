#!/usr/bin/env bash
#
# Per-boot startup for the Online Boutique Cloud Agent environment.
#
# Brings up the Docker daemon, (re)creates a local kind Kubernetes cluster and
# deploys the Online Boutique release manifests. It is idempotent: re-running it
# reconciles state instead of duplicating it, and it returns once the frontend
# is available.
set -euo pipefail

CLUSTER_NAME="online-boutique"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KIND_CONFIG="${REPO_ROOT}/.cursor/kind-cluster.yaml"
MANIFESTS="${REPO_ROOT}/release/kubernetes-manifests.yaml"

log() { echo "[start] $*"; }

# docker/kind need the docker group; run those through `sg` so this works even
# when start.sh is not launched from a fresh login shell.
dockerg() { sg docker -c "$*"; }

# --- 1. Docker daemon ------------------------------------------------------
if ! dockerg "docker info" >/dev/null 2>&1; then
  log "Starting dockerd"
  sudo rm -f /tmp/dockerd.log /tmp/dockerd.pid
  sudo bash -c 'nohup dockerd >/tmp/dockerd.log 2>&1 & echo $! >/tmp/dockerd.pid'
  for _ in $(seq 1 60); do
    dockerg "docker info" >/dev/null 2>&1 && break
    sleep 1
  done
fi
dockerg "docker info" >/dev/null 2>&1 || { echo "dockerd failed to start; see /tmp/dockerd.log" >&2; exit 1; }
# Make the socket group-accessible for tools run in this session.
sudo chown root:docker /var/run/docker.sock 2>/dev/null || true
sudo chmod 660 /var/run/docker.sock 2>/dev/null || true
log "Docker is up: $(dockerg "docker version --format '{{.Server.Version}} ({{.Server.Os}})'" 2>/dev/null)"

# --- 2. kind cluster -------------------------------------------------------
if dockerg "kind get clusters" 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "kind cluster '${CLUSTER_NAME}' already exists"
else
  log "Creating kind cluster '${CLUSTER_NAME}'"
  dockerg "kind create cluster --name ${CLUSTER_NAME} --config ${KIND_CONFIG} --wait 120s"
fi
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# --- 3. Deploy Online Boutique --------------------------------------------
log "Applying release manifests"
kubectl apply -f "${MANIFESTS}"

# --- 4. Wait for the frontend to be ready ----------------------------------
log "Waiting for the frontend deployment to become available"
kubectl wait --for=condition=available deployment/frontend --timeout=300s || {
  echo "frontend did not become ready in time; recent pods:" >&2
  kubectl get pods >&2 || true
  exit 1
}

log "start.sh complete — cluster is up. Open the 'frontend' terminal for the web UI on http://localhost:8080"
