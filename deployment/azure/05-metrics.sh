#!/bin/bash
# Metrics VM: Prometheus + Loki + Grafana (public, behind nginx+certbot).
# The 'loadtest' Prometheus scrape target isn't known yet (VM doesn't exist until 06) -
# it's wired up for real in 07-post-config.sh, which also reloads Prometheus.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

APP1_IP="$(vm_private_ip mm-app-1)"
APP2_IP="$(vm_private_ip mm-app-2)"
PROXY_IP="$(vm_private_ip mm-proxy)"
LOADTEST_IP="127.0.0.1"   # placeholder; fixed up in 07-post-config.sh
SUFFIX="$(dns_suffix)"
DNS_LABEL="mm-lt-grafana-${SUFFIX}"
METRICS_FQDN="$(metrics_fqdn)"

GRAFANA_ADMIN_PASSWORD="$(gen_secret grafana_admin_password)"
echo "${GRAFANA_ADMIN_PASSWORD}" > "${SECRETS_DIR}/grafana_admin_password"

IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts-gen2:latest"

log "Creating public IP for metrics VM (${METRICS_FQDN})"
az network public-ip create -g "${RG}" -n mm-metrics-pip --sku Standard --allocation-method Static \
  --dns-name "${DNS_LABEL}" -o none

log "Creating VM mm-metrics"
az vm create -g "${RG}" -n mm-metrics --image "${IMAGE}" --size Standard_B1ms \
  --vnet-name "${VNET}" --subnet "${SNET_METRICS}" --nsg "" \
  --public-ip-address mm-metrics-pip \
  --admin-username "${VM_ADMIN_USER}" --ssh-key-values "${SSH_PUB_KEY}" \
  -o none

APP1_IP="${APP1_IP}" APP2_IP="${APP2_IP}" PROXY_IP="${PROXY_IP}" LOADTEST_IP="${LOADTEST_IP}" \
  envsubst '${APP1_IP} ${APP2_IP} ${PROXY_IP} ${LOADTEST_IP}' \
  < "${ASSETS_DIR}/prometheus.yml.tmpl" > "${STATE_DIR}/prometheus-rendered.yml"

export PROMETHEUS_YML GRAFANA_DATASOURCE GRAFANA_DASHBOARD_PROVIDER GRAFANA_AUTH_INI \
  GRAFANA_ADMIN_PASSWORD NGINX_GRAFANA_MAIN NGINX_GRAFANA_SITE LE_EMAIL METRICS_FQDN
PROMETHEUS_YML="$(cat "${STATE_DIR}/prometheus-rendered.yml")"
GRAFANA_DATASOURCE="$(cat "${ASSETS_DIR}/grafana-datasource.yaml")"
GRAFANA_DASHBOARD_PROVIDER="$(cat "${ASSETS_DIR}/grafana-dashboard-provider.yaml")"
GRAFANA_AUTH_INI="$(cat "${ASSETS_DIR}/grafana-auth.ini")"
NGINX_GRAFANA_MAIN="$(cat "${ASSETS_DIR}/nginx-grafana-main.conf")"
LE_EMAIL="alejandro.garcia@mattermost.com"

METRICS_FQDN="${METRICS_FQDN}" envsubst '${METRICS_FQDN}' \
  < "${ASSETS_DIR}/nginx-grafana-site.conf.tmpl" > "${STATE_DIR}/nginx-grafana-site-rendered.conf"
NGINX_GRAFANA_SITE="$(cat "${STATE_DIR}/nginx-grafana-site-rendered.conf")"

RENDERED="${STATE_DIR}/provision-metrics.sh"
envsubst '${PROMETHEUS_YML} ${GRAFANA_DATASOURCE} ${GRAFANA_DASHBOARD_PROVIDER} ${GRAFANA_AUTH_INI} ${GRAFANA_ADMIN_PASSWORD} ${NGINX_GRAFANA_MAIN} ${NGINX_GRAFANA_SITE} ${LE_EMAIL} ${METRICS_FQDN}' \
  < "${ASSETS_DIR}/provision-metrics.sh.tmpl" > "${RENDERED}"

log "Provisioning mm-metrics"
run_on_vm mm-metrics "${RENDERED}"

log "Grafana ready: https://${METRICS_FQDN} (admin / see secrets/grafana_admin_password)"
log "NOTE: default dashboards (default_dashboard_tmpl.json / coordinator_dashboard_tmpl.json) were NOT imported -"
log "they're Go-templated/generated in deployment/terraform/metrics.go and not straightforward to reuse outside that Go code."
log "Grafana has working Prometheus + Loki datasources; use Explore or build/import a dashboard manually."
