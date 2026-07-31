#!/bin/bash
# nginx reverse proxy / load balancer in front of the 2 app nodes.
# - Public 443 (TLS via certbot) + 80 (redirect) for real users.
# - Plain-HTTP 8065, NSG-restricted to snet-loadtest only, for the load-test VM.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

APP1_IP="$(vm_private_ip mm-app-1)"
APP2_IP="$(vm_private_ip mm-app-2)"
SUFFIX="$(dns_suffix)"
DNS_LABEL="mm-lt-proxy-${SUFFIX}"
PROXY_FQDN="$(proxy_fqdn)"

IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts-gen2:latest"

log "Creating public IP for proxy (${PROXY_FQDN})"
az network public-ip create -g "${RG}" -n mm-proxy-pip --sku Standard --allocation-method Static \
  --dns-name "${DNS_LABEL}" -o none

log "Creating VM mm-proxy"
az vm create -g "${RG}" -n mm-proxy --image "${IMAGE}" --size Standard_B1ms \
  --vnet-name "${VNET}" --subnet "${SNET_PROXY}" --nsg "" \
  --public-ip-address mm-proxy-pip \
  --admin-username "${VM_ADMIN_USER}" --ssh-key-values "${SSH_PUB_KEY}" \
  -o none

export LIMITS_CONF SYSCTL_CONF NGINX_CONF NGINX_PROXY_SNIPPET NGINX_CACHE_SNIPPET NGINX_SITE_CONF LE_EMAIL PROXY_FQDN
LIMITS_CONF="$(cat "${ASSETS_DIR}/limits.conf")"
SYSCTL_CONF="$(cat "${ASSETS_DIR}/sysctl-server.conf")"
NGINX_CONF="$(cat "${ASSETS_DIR}/nginx.conf")"
NGINX_PROXY_SNIPPET="$(cat "${ASSETS_DIR}/nginx-proxy-snippet.conf")"
NGINX_CACHE_SNIPPET="$(cat "${ASSETS_DIR}/nginx-cache-snippet.conf")"
LE_EMAIL="alejandro.garcia@mattermost.com"

APP1_IP="${APP1_IP}" APP2_IP="${APP2_IP}" PROXY_FQDN="${PROXY_FQDN}" \
  envsubst '${APP1_IP} ${APP2_IP} ${PROXY_FQDN}' \
  < "${ASSETS_DIR}/nginx-mattermost-site.conf.tmpl" > "${STATE_DIR}/nginx-site-rendered.conf"
NGINX_SITE_CONF="$(cat "${STATE_DIR}/nginx-site-rendered.conf")"

RENDERED="${STATE_DIR}/provision-proxy.sh"
envsubst '${LIMITS_CONF} ${SYSCTL_CONF} ${NGINX_CONF} ${NGINX_PROXY_SNIPPET} ${NGINX_CACHE_SNIPPET} ${NGINX_SITE_CONF} ${LE_EMAIL} ${PROXY_FQDN}' \
  < "${ASSETS_DIR}/provision-proxy.sh.tmpl" > "${RENDERED}"

log "Provisioning mm-proxy"
run_on_vm mm-proxy "${RENDERED}"

log "Proxy ready: https://${PROXY_FQDN} (public), http://$(vm_private_ip mm-proxy):8065 (load-test only)"
