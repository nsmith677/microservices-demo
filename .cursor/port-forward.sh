#!/usr/bin/env bash
#
# Long-running terminal helper: exposes the Online Boutique web frontend on
# http://localhost:8080. Waits for the frontend to be ready, then port-forwards
# and automatically reconnects if the forward drops (e.g. pod restart).
set -uo pipefail

CLUSTER_NAME="online-boutique"
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null 2>&1 || true

echo "[frontend] waiting for the frontend deployment..."
kubectl wait --for=condition=available deployment/frontend --timeout=600s || true

while true; do
  echo "[frontend] forwarding http://localhost:8080 -> deployment/frontend:8080"
  kubectl port-forward deployment/frontend 8080:8080 --address 0.0.0.0 || true
  echo "[frontend] port-forward exited; retrying in 3s..."
  sleep 3
done
