#!/bin/bash
# Deletes every resource created by deploy.sh (they all live in one resource group).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

if ! az group show -n "${RG}" &>/dev/null; then
  echo "Resource group ${RG} does not exist, nothing to tear down."
  exit 0
fi

echo "This will permanently delete resource group '${RG}' and everything in it"
echo "(2 app VMs, proxy VM, metrics VM, load-test VM, Postgres Flexible Server, storage account, VNet)."
read -r -p "Type the resource group name to confirm deletion: " CONFIRM
if [ "${CONFIRM}" != "${RG}" ]; then
  echo "Confirmation did not match '${RG}', aborting."
  exit 1
fi

log "Deleting resource group ${RG} (running in background, this takes several minutes)"
az group delete -n "${RG}" --yes --no-wait

log "Deletion started. Check status with: az group exists -n ${RG}"
