#!/bin/bash
# Switches the load-test VM's ltapi from a natively-built systemd binary (06-loadtest.sh) to
# the portable Docker image (Dockerfile.portable at repo root), per the "Portable Load Testing"
# Confluence spec (MM-69373) - this mimics the Crimson-1 scenario: build the image on an
# operator machine, transfer it across an air-gap boundary (here: scp, since we don't have a
# real air gap), `docker load` + `docker run` on the target VM. ltcoordinator/ltagent stay as
# the native binaries already built by 06-loadtest.sh - only ltapi needs to run persistently
# on the remote machine, per the spec.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

LOADTEST_IP="$(vm_public_ip mm-loadtest)"
IMAGE_TAR="${STATE_DIR}/ltapi-image.tar.gz"

log "Building ltapi:latest from ${REPO_ROOT}/Dockerfile.portable (operator machine)"
docker build -f "${REPO_ROOT}/Dockerfile.portable" -t ltapi:latest "${REPO_ROOT}"

log "Saving image to ${IMAGE_TAR} (simulates transfer across an air-gap boundary)"
docker save ltapi:latest | gzip > "${IMAGE_TAR}"
ls -lh "${IMAGE_TAR}"

log "Transferring image to mm-loadtest (${LOADTEST_IP})"
scp -o StrictHostKeyChecking=accept-new "${IMAGE_TAR}" "${VM_ADMIN_USER}@${LOADTEST_IP}:/tmp/ltapi-image.tar.gz"

log "Installing Docker on mm-loadtest (if not already present)"
ssh "${VM_ADMIN_USER}@${LOADTEST_IP}" '
  if ! command -v docker &>/dev/null; then
    sudo apt-get update -y -qq
    sudo apt-get install -y docker.io
    sudo systemctl enable --now docker
    sudo usermod -aG docker '"${VM_ADMIN_USER}"'
  fi
'

log "docker load + run (this SSH connection needs the group membership from a fresh login if docker was just installed)"
ssh "${VM_ADMIN_USER}@${LOADTEST_IP}" '
  docker load -i /tmp/ltapi-image.tar.gz
  rm -f /tmp/ltapi-image.tar.gz

  # Stop the native systemd-managed ltapi from 06-loadtest.sh - it would otherwise fight the
  # container for port 4000.
  sudo systemctl stop ltapi 2>/dev/null || true
  sudo systemctl disable ltapi 2>/dev/null || true

  docker rm -f ltapi 2>/dev/null || true
  docker run -d --name ltapi --restart always --ulimit nofile=100000:100000 -p 4000:4000 ltapi:latest ltapi
  sleep 2
  docker ps --filter name=ltapi
'

log "ltapi now running in Docker on mm-loadtest (port 4000, unchanged - Prometheus scrape target is unaffected)"
