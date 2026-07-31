#!/bin/bash
# Moves ltcoordinator to the operator's own machine, per the Confluence spec's actual framing
# ("ltcoordinator ... can run on the operator's laptop or any machine that can reach the agent
# VMs on port 4000") - a faithful simulation of Crimson-1, not something technically required
# in our environment (co-locating it on the load-test VM, as 06-loadtest.sh does, works fine
# and is more locked-down). This is deliberately extra friction, requested for fidelity.
#
# ltapi itself (the thing that actually connects to Mattermost and generates traffic) keeps
# running on mm-loadtest, unchanged - only the *control* plane (ltcoordinator) moves out here.
# ltagent is not touched by this script: with the DB dump loaded (see DEPLOYMENT.md), `ltagent
# init`'s synthetic team/channel seeding isn't needed.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

MYIP="$(my_ip)/32"
LOADTEST_PUBLIC_IP="$(vm_public_ip mm-loadtest)"
METRICS_PUBLIC_IP="$(vm_public_ip mm-metrics)"

log "Opening ltapi's control port (4000) and Prometheus (9090) to the operator's IP only"
retry az network nsg rule create -g "${RG}" --nsg-name nsg-loadtest -n Allow-Operator-4000 --priority 120 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 4000 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-Operator-Prometheus --priority 150 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 9090 -o none

log "Building ltcoordinator locally (this machine is 'the operator's laptop')"
( cd "${REPO_ROOT}" && go build -o bin/ltcoordinator ./cmd/ltcoordinator )

log "Rendering operator-side config in ${AZURE_DIR}/operator/"
mkdir -p operator

# config.json: ServerURL/WebSocketURL stay pointed at the proxy's PRIVATE IP. This is not a bug -
# ltcoordinator forwards this config to ltapi over the network (see coordinator/coordinator.go:
# "The ltConfig parameter is used to create and configure load-test agents") and it's ltapi,
# still running inside the VNet, that actually connects to Mattermost with it.
PROXY_IP="$(vm_private_ip mm-proxy)" MM_ADMIN_PASSWORD='Sys@dmin-sample1' \
  envsubst '${PROXY_IP} ${MM_ADMIN_PASSWORD}' < assets/config.json.tmpl > operator/config.json
python3 -c "
import json
with open('operator/config.json') as f:
    d = json.load(f)
d['ConnectionConfiguration']['AdminEmail'] = 'sysadmin@sample.mattermost.com'
with open('operator/config.json', 'w') as f:
    json.dump(d, f, indent=2)
"

# coordinator.json: Agents[].ApiURL and MonitorConfig.PrometheusURL DO need to change to public
# IPs now that ltcoordinator itself runs outside the VNet.
METRICS_IP="${METRICS_PUBLIC_IP}" envsubst '${METRICS_IP}' < assets/coordinator.json.tmpl > operator/coordinator.json
python3 -c "
import json
with open('operator/coordinator.json') as f:
    d = json.load(f)
d['ClusterConfig']['Agents'][0]['ApiURL'] = 'http://${LOADTEST_PUBLIC_IP}:4000'
with open('operator/coordinator.json', 'w') as f:
    json.dump(d, f, indent=2)
"

cp assets/simulcontroller.json operator/simulcontroller.json

log "Verifying reachability from here"
curl -s -o /dev/null -w "ltapi control (${LOADTEST_PUBLIC_IP}:4000): %{http_code}\n" "http://${LOADTEST_PUBLIC_IP}:4000/loadagent/" || true
curl -s -o /dev/null -w "prometheus (${METRICS_PUBLIC_IP}:9090): %{http_code}\n" "http://${METRICS_PUBLIC_IP}:9090/-/healthy" || true

log "Ready. Run the load test from here with:"
echo "  ${REPO_ROOT}/bin/ltcoordinator -c ${AZURE_DIR}/operator/coordinator.json -l ${AZURE_DIR}/operator/config.json"
