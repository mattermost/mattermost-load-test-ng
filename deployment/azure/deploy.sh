#!/bin/bash
# Runs the full Azure HA + metrics + load-test deployment end to end.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

for step in 00-network 01-database 02-storage 03-app-nodes 04-proxy 05-metrics 06-loadtest 06b-loadtest-docker 07-post-config; do
  log "=== ${step} ==="
  "./${step}.sh"
done

log "Configuring auto-shutdown (03:00 UTC) on all 5 VMs to save cost while idle"
for vm in mm-app-1 mm-app-2 mm-proxy mm-metrics mm-loadtest; do
  az vm auto-shutdown -g "${RG}" -n "${vm}" --time 0300 -o none
done

log "Deployment complete."
