#!/bin/bash
# Load-test VM: build ltapi/ltcoordinator/ltagent from source, client-side sysctl tuning,
# ltapi as a systemd service, ltcoordinator run interactively (tmux) by the operator.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

PROXY_IP="$(vm_private_ip mm-proxy)"
METRICS_IP="$(vm_private_ip mm-metrics)"
MM_ADMIN_PASSWORD="$(gen_secret mm_admin_password)"
echo "${MM_ADMIN_PASSWORD}" > "${SECRETS_DIR}/mm_admin_password"

IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts-gen2:latest"

log "Creating public IP for load-test VM (SSH only)"
az network public-ip create -g "${RG}" -n mm-loadtest-pip --sku Standard --allocation-method Static -o none

log "Creating VM mm-loadtest"
az vm create -g "${RG}" -n mm-loadtest --image "${IMAGE}" --size Standard_B4ms \
  --vnet-name "${VNET}" --subnet "${SNET_LOADTEST}" --nsg "" \
  --public-ip-address mm-loadtest-pip \
  --admin-username "${VM_ADMIN_USER}" --ssh-key-values "${SSH_PUB_KEY}" \
  -o none

INSTANCE_NAME="loadtest-0" METRICS_IP="${METRICS_IP}" \
  envsubst '${INSTANCE_NAME} ${METRICS_IP}' \
  < "${ASSETS_DIR}/otelcol-loadtest.yaml.tmpl" > "${STATE_DIR}/otelcol-loadtest-rendered.yaml"

PROXY_IP="${PROXY_IP}" MM_ADMIN_PASSWORD="${MM_ADMIN_PASSWORD}" \
  envsubst '${PROXY_IP} ${MM_ADMIN_PASSWORD}' \
  < "${ASSETS_DIR}/config.json.tmpl" > "${STATE_DIR}/loadtest-config-rendered.json"

METRICS_IP="${METRICS_IP}" envsubst '${METRICS_IP}' \
  < "${ASSETS_DIR}/coordinator.json.tmpl" > "${STATE_DIR}/loadtest-coordinator-rendered.json"

export LIMITS_CONF SYSCTL_CONF GO_VERSION VM_ADMIN_USER LOADTEST_CONFIG_JSON LOADTEST_COORDINATOR_JSON \
  LOADTEST_SIMULCONTROLLER_JSON LTAPI_SERVICE OTELCOL_CONFIG
LIMITS_CONF="$(cat "${ASSETS_DIR}/limits.conf")"
SYSCTL_CONF="$(cat "${ASSETS_DIR}/sysctl-client.conf")"
GO_VERSION="1.26.3"
LOADTEST_CONFIG_JSON="$(cat "${STATE_DIR}/loadtest-config-rendered.json")"
LOADTEST_COORDINATOR_JSON="$(cat "${STATE_DIR}/loadtest-coordinator-rendered.json")"
LOADTEST_SIMULCONTROLLER_JSON="$(cat "${ASSETS_DIR}/simulcontroller.json")"
LTAPI_SERVICE="$(cat "${ASSETS_DIR}/ltapi.service")"
OTELCOL_CONFIG="$(cat "${STATE_DIR}/otelcol-loadtest-rendered.yaml")"

RENDERED="${STATE_DIR}/provision-loadtest.sh"
envsubst '${LIMITS_CONF} ${SYSCTL_CONF} ${GO_VERSION} ${VM_ADMIN_USER} ${LOADTEST_CONFIG_JSON} ${LOADTEST_COORDINATOR_JSON} ${LOADTEST_SIMULCONTROLLER_JSON} ${LTAPI_SERVICE} ${OTELCOL_CONFIG}' \
  < "${ASSETS_DIR}/provision-loadtest.sh.tmpl" > "${RENDERED}"

log "Provisioning mm-loadtest (this includes a from-source Go build, can take a few minutes)"
run_on_vm mm-loadtest "${RENDERED}"

log "Load-test VM ready: $(vm_private_ip mm-loadtest) (private), $(vm_public_ip mm-loadtest) (SSH only)"
