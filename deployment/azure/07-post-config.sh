#!/bin/bash
# Final cross-VM wiring now that every IP is known:
#   1. Prometheus scrape config gets the real load-test target + reload.
#   2. otelcol-contrib on app-1/app-2/proxy gets the real metrics IP + starts.
#   3. License upload + sysadmin creation on mm-app-1.
#   4. ltagent init (seed teams/channels) on mm-loadtest.
#   5. Print final summary of URLs/IPs/credentials.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

APP1_IP="$(vm_private_ip mm-app-1)"
APP2_IP="$(vm_private_ip mm-app-2)"
PROXY_IP="$(vm_private_ip mm-proxy)"
METRICS_IP="$(vm_private_ip mm-metrics)"
LOADTEST_IP="$(vm_private_ip mm-loadtest)"
PROXY_FQDN="$(proxy_fqdn)"
METRICS_FQDN="$(metrics_fqdn)"
MM_ADMIN_PASSWORD="$(cat "${SECRETS_DIR}/mm_admin_password")"
GRAFANA_ADMIN_PASSWORD="$(cat "${SECRETS_DIR}/grafana_admin_password")"

log "1/4 Final Prometheus scrape config (real loadtest target: ${LOADTEST_IP})"
APP1_IP="${APP1_IP}" APP2_IP="${APP2_IP}" PROXY_IP="${PROXY_IP}" LOADTEST_IP="${LOADTEST_IP}" \
  envsubst '${APP1_IP} ${APP2_IP} ${PROXY_IP} ${LOADTEST_IP}' \
  < "${ASSETS_DIR}/prometheus.yml.tmpl" > "${STATE_DIR}/prometheus-final.yml"
export PROMETHEUS_YML="$(cat "${STATE_DIR}/prometheus-final.yml")"
envsubst '${PROMETHEUS_YML}' < "${ASSETS_DIR}/reload-prometheus.sh.tmpl" > "${STATE_DIR}/reload-prometheus.sh"
run_on_vm mm-metrics "${STATE_DIR}/reload-prometheus.sh"

log "2/4 Enabling log shipping (otelcol-contrib) on app-1, app-2, proxy"
declare -A ROLE_TMPL=( [mm-app-1]=otelcol-app [mm-app-2]=otelcol-app [mm-proxy]=otelcol-proxy )
declare -A ROLE_INSTANCE=( [mm-app-1]=app-0 [mm-app-2]=app-1 [mm-proxy]=proxy )
for vm in mm-app-1 mm-app-2 mm-proxy; do
  tmpl="${ROLE_TMPL[$vm]}"
  INSTANCE_NAME="${ROLE_INSTANCE[$vm]}" METRICS_IP="${METRICS_IP}" \
    envsubst '${INSTANCE_NAME} ${METRICS_IP}' \
    < "${ASSETS_DIR}/${tmpl}.yaml.tmpl" > "${STATE_DIR}/${vm}-otelcol.yaml"
  export OTELCOL_CONFIG="$(cat "${STATE_DIR}/${vm}-otelcol.yaml")"
  export VM_ADMIN_USER
  envsubst '${OTELCOL_CONFIG} ${VM_ADMIN_USER}' < "${ASSETS_DIR}/apply-otelcol.sh.tmpl" > "${STATE_DIR}/${vm}-apply-otelcol.sh"
  run_on_vm "${vm}" "${STATE_DIR}/${vm}-apply-otelcol.sh"
done

log "3/4 Uploading license + creating sysadmin on mm-app-1"
LICENSE_B64="$(base64 -w0 "${LICENSE_FILE}")"
export MM_ADMIN_PASSWORD LICENSE_B64
envsubst '${MM_ADMIN_PASSWORD} ${LICENSE_B64}' \
  < "${ASSETS_DIR}/provision-postconfig-app1.sh.tmpl" > "${STATE_DIR}/postconfig-app1.sh"
run_on_vm mm-app-1 "${STATE_DIR}/postconfig-app1.sh"

log "4/4 Seeding test data (ltagent init -n 2000) on mm-loadtest"
export VM_ADMIN_USER
envsubst '${VM_ADMIN_USER}' < "${ASSETS_DIR}/ltagent-init.sh.tmpl" > "${STATE_DIR}/ltagent-init.sh"
run_on_vm mm-loadtest "${STATE_DIR}/ltagent-init.sh"

cat <<SUMMARY

================================================================================
 Mattermost HA load-test environment ready
================================================================================
 Mattermost (public):      https://${PROXY_FQDN}
   sysadmin login:         sysadmin@loadtest.mattermost.com / ${MM_ADMIN_PASSWORD}
 Mattermost (load-test):   http://${PROXY_IP}:8065  (only reachable from mm-loadtest)

 Grafana (public):         https://${METRICS_FQDN}
   admin login:            admin / ${GRAFANA_ADMIN_PASSWORD}
   (Prometheus + Loki datasources pre-provisioned; no default dashboards imported -
    default_dashboard_tmpl.json/coordinator_dashboard_tmpl.json are Go-templated/generated
    in deployment/terraform/metrics.go and weren't ported to this bash-based deployment)

 SSH (restricted to your IP $(my_ip)):
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-app-1)       # app node 1
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-app-2)       # app node 2
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-proxy)       # nginx proxy
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-metrics)     # prometheus/loki/grafana
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-loadtest)    # ltapi/ltcoordinator

 To run the load test:
   ssh ${VM_ADMIN_USER}@$(vm_public_ip mm-loadtest)
   cd mattermost-load-test-ng
   tmux new -s loadtest
   ./bin/ltcoordinator -c config/coordinator.json -l config/config.json
   # Ctrl-b d to detach, tmux attach -t loadtest to reattach

 All secrets are also in: ${SECRETS_DIR}/
 Teardown everything:      ./deployment/azure/teardown.sh
================================================================================
SUMMARY
