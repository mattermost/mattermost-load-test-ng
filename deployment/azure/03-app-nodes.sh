#!/bin/bash
# 2x Mattermost app node VMs: Postgres DSN, HA cluster settings, shared file storage,
# server-side sysctl tuning, systemd unit. Public IP exists only for SSH (NSG-restricted).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

DB_FQDN="$(cat "${STATE_DIR}/db_fqdn")"
DB_PASSWORD="$(cat "${SECRETS_DIR}/db_admin_password")"
STORAGE_ACCOUNT="$(cat "${STORAGE_ACCOUNT_FILE}")"
STORAGE_KEY="$(cat "${SECRETS_DIR}/storage_account_key")"
SITE_URL="https://$(proxy_fqdn)"

IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts-gen2:latest"

for vm in mm-app-1 mm-app-2; do
  log "Creating VM ${vm}"
  az vm create -g "${RG}" -n "${vm}" --image "${IMAGE}" --size Standard_D2s_v6 \
    --vnet-name "${VNET}" --subnet "${SNET_APP}" --nsg "" \
    --public-ip-sku Standard \
    --admin-username "${VM_ADMIN_USER}" --ssh-key-values "${SSH_PUB_KEY}" \
    -o none
done

export LIMITS_CONF SYSCTL_CONF MATTERMOST_SERVICE DB_FQDN DB_ADMIN_USER DB_PASSWORD DB_NAME STORAGE_ACCOUNT STORAGE_KEY SITE_URL MM_DOWNLOAD_URL
LIMITS_CONF="$(cat "${ASSETS_DIR}/limits.conf")"
SYSCTL_CONF="$(cat "${ASSETS_DIR}/sysctl-server.conf")"
MATTERMOST_SERVICE="$(cat "${ASSETS_DIR}/mattermost.service")"

RENDERED="${STATE_DIR}/provision-app.sh"
envsubst '${LIMITS_CONF} ${SYSCTL_CONF} ${MATTERMOST_SERVICE} ${DB_FQDN} ${DB_ADMIN_USER} ${DB_PASSWORD} ${DB_NAME} ${STORAGE_ACCOUNT} ${STORAGE_KEY} ${SITE_URL} ${MM_DOWNLOAD_URL}' \
  < "${ASSETS_DIR}/provision-app.sh.tmpl" > "${RENDERED}"

for vm in mm-app-1 mm-app-2; do
  log "Provisioning ${vm}"
  run_on_vm "${vm}" "${RENDERED}"
done

log "App nodes: $(vm_private_ip mm-app-1) $(vm_private_ip mm-app-2) (private IPs)"
