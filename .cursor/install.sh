#!/usr/bin/env bash
#
# Idempotent setup for the Online Boutique (microservices-demo) Cloud Agent
# environment. Installs the container tooling needed to build and run the app
# on a local Kubernetes cluster: Docker Engine, kubectl, kind and skaffold.
#
# The Docker daemon itself is NOT started here (that belongs in start.sh); this
# script only installs durable dependencies and warms the base image cache so a
# fresh boot can bring the cluster up quickly.
set -euo pipefail

KIND_VERSION="v0.30.0"
KIND_NODE_IMAGE="kindest/node:v1.34.0"
KUBECTL_VERSION="v1.37.0"
SKAFFOLD_VERSION="v2.24.0"

log() { echo "[install] $*"; }

# --- System packages -------------------------------------------------------
log "Installing system packages (docker.io, fuse-overlayfs, iptables, ...)"
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
# The fuse3 package prompts about the pre-existing /etc/fuse.conf conffile,
# which would abort a non-interactive install; force-conf* answers it so every
# package configures cleanly.
sudo apt-get install -y -qq \
  -o Dpkg::Options::=--force-confdef \
  -o Dpkg::Options::=--force-confold \
  docker.io \
  fuse-overlayfs \
  uidmap \
  iptables \
  curl \
  ca-certificates \
  git
command -v fuse-overlayfs >/dev/null || { echo "fuse-overlayfs missing" >&2; exit 1; }

# --- Docker daemon configuration -------------------------------------------
# Nested Cloud Agent VMs cannot mount the native overlay2 driver, so use the
# classic fuse-overlayfs graph driver. The nftables/legacy iptables split is
# handled by pointing the alternatives at the legacy backend, which the Docker
# daemon and kind expect.
log "Configuring Docker daemon (fuse-overlayfs storage driver)"
sudo update-alternatives --set iptables /usr/sbin/iptables-legacy || true
sudo update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy || true
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json >/dev/null <<'JSON'
{
  "features": { "containerd-snapshotter": false },
  "storage-driver": "fuse-overlayfs"
}
JSON

# Allow the unprivileged user to talk to the Docker socket without sudo.
sudo groupadd -f docker
sudo usermod -aG docker "$(id -un)"

# --- CLI tooling -----------------------------------------------------------
install_bin() {
  # install_bin <name> <url> <version-check-substring>
  local name="$1" url="$2" want="$3"
  if command -v "$name" >/dev/null 2>&1 && "$name" version 2>/dev/null | grep -q "$want"; then
    log "$name $want already installed"
    return
  fi
  log "Installing $name ($want)"
  local tmp
  tmp="$(mktemp)"
  curl -fsSL -o "$tmp" "$url"
  chmod +x "$tmp"
  sudo mv "$tmp" "/usr/local/bin/$name"
}

install_bin kubectl \
  "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
  "${KUBECTL_VERSION}"
install_bin kind \
  "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64" \
  "${KIND_VERSION}"
install_bin skaffold \
  "https://storage.googleapis.com/skaffold/releases/${SKAFFOLD_VERSION}/skaffold-linux-amd64" \
  "${SKAFFOLD_VERSION}"

# --- Warm the image cache --------------------------------------------------
# Pre-pull the kind node image so cluster creation on a fresh boot is fast.
# This needs the daemon briefly; start it, pull, and leave the pulled layers
# in /var/lib/docker (durable), then stop the daemon we started here.
DOCKERD_PID=""
# Use `sudo docker` here: the docker group membership added above is not active
# within this already-running process.
if ! sudo docker info >/dev/null 2>&1; then
  log "Temporarily starting dockerd to pre-pull ${KIND_NODE_IMAGE}"
  sudo bash -c 'nohup dockerd >/tmp/dockerd-install.log 2>&1 & echo $! >/tmp/dockerd-install.pid'
  DOCKERD_PID="$(cat /tmp/dockerd-install.pid 2>/dev/null || true)"
  for _ in $(seq 1 30); do sudo docker info >/dev/null 2>&1 && break; sleep 1; done
fi
if sudo docker info >/dev/null 2>&1; then
  sudo docker pull "${KIND_NODE_IMAGE}" || log "warn: could not pre-pull ${KIND_NODE_IMAGE}"
fi
if [ -n "${DOCKERD_PID}" ]; then
  log "Stopping the temporary dockerd (pid ${DOCKERD_PID}) started for pre-pull"
  sudo kill "${DOCKERD_PID}" 2>/dev/null || true
fi

log "install.sh complete"
