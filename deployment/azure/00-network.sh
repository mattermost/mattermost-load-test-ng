#!/bin/bash
# Resource group, VNet + 5 subnets, and one NSG per subnet with the minimum rules needed.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

MYIP="$(my_ip)/32"
log "Operator IP for SSH rules: ${MYIP}"

log "Resource group ${RG} in ${LOCATION}"
az group create -n "${RG}" -l "${LOCATION}" -o none

log "VNet ${VNET} + subnets"
az network vnet create -g "${RG}" -n "${VNET}" --address-prefix 10.20.0.0/16 -o none \
  --subnet-name "${SNET_PROXY}" --subnet-prefix "${SNET_PROXY_CIDR}"

# ARM read-after-write lag: creating a subnet immediately after the vnet can 404 even though
# the vnet create above already returned success, and this can happen even after polling
# 'provisioningState' on the vnet itself (cross-replica lag, not a provisioning issue) -
# retry-with-backoff on the dependent calls is the robust fix.
az network vnet wait -g "${RG}" -n "${VNET}" --created

retry az network vnet subnet create -g "${RG}" --vnet-name "${VNET}" -n "${SNET_APP}" --address-prefix "${SNET_APP_CIDR}" -o none
retry az network vnet subnet create -g "${RG}" --vnet-name "${VNET}" -n "${SNET_METRICS}" --address-prefix "${SNET_METRICS_CIDR}" -o none
retry az network vnet subnet create -g "${RG}" --vnet-name "${VNET}" -n "${SNET_LOADTEST}" --address-prefix "${SNET_LOADTEST_CIDR}" -o none

retry az network vnet subnet create -g "${RG}" --vnet-name "${VNET}" -n "${SNET_DB}" --address-prefix "${SNET_DB_CIDR}" -o none \
  --delegations "Microsoft.DBforPostgreSQL/flexibleServers"

log "NSG: nsg-proxy"
retry az network nsg create -g "${RG}" -n nsg-proxy -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-proxy -n Allow-SSH-Operator --priority 100 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 22 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-proxy -n Allow-HTTP-Internet --priority 110 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes Internet --destination-port-ranges 80 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-proxy -n Allow-HTTPS-Internet --priority 120 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes Internet --destination-port-ranges 443 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-proxy -n Allow-LoadTest-8065 --priority 130 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_LOADTEST_CIDR}" --destination-port-ranges 8065 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-proxy -n Allow-NodeExporter-Metrics --priority 140 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_METRICS_CIDR}" --destination-port-ranges 9100 -o none

log "NSG: nsg-app"
retry az network nsg create -g "${RG}" -n nsg-app -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-app -n Allow-SSH-Operator --priority 100 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 22 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-app -n Allow-MM-FromProxy --priority 110 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_PROXY_CIDR}" --destination-port-ranges 8065 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-app -n Allow-Metrics-FromMetricsVM --priority 120 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_METRICS_CIDR}" --destination-port-ranges 8067 9100 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-app -n Allow-Gossip-Intracluster --priority 130 \
  --access Allow --protocol "*" --direction Inbound --source-address-prefixes "${SNET_APP_CIDR}" --destination-port-ranges 8074 -o none

log "NSG: nsg-db"
retry az network nsg create -g "${RG}" -n nsg-db -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-db -n Allow-Postgres-FromApp --priority 100 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_APP_CIDR}" --destination-port-ranges 5432 -o none

log "NSG: nsg-metrics"
retry az network nsg create -g "${RG}" -n nsg-metrics -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-SSH-Operator --priority 100 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 22 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-HTTP-Internet --priority 110 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes Internet --destination-port-ranges 80 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-HTTPS-Internet --priority 120 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes Internet --destination-port-ranges 443 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-Loki-Push --priority 130 \
  --access Allow --protocol Tcp --direction Inbound \
  --source-address-prefixes "${SNET_APP_CIDR}" "${SNET_PROXY_CIDR}" "${SNET_LOADTEST_CIDR}" --destination-port-ranges 3100 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-metrics -n Allow-Prometheus-FromLoadtest --priority 140 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_LOADTEST_CIDR}" --destination-port-ranges 9090 -o none

log "NSG: nsg-loadtest"
retry az network nsg create -g "${RG}" -n nsg-loadtest -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-loadtest -n Allow-SSH-Operator --priority 100 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${MYIP}" --destination-port-ranges 22 -o none
retry az network nsg rule create -g "${RG}" --nsg-name nsg-loadtest -n Allow-Metrics-FromMetricsVM --priority 110 \
  --access Allow --protocol Tcp --direction Inbound --source-address-prefixes "${SNET_METRICS_CIDR}" --destination-port-ranges 4000 9100 -o none

log "Associating NSGs with subnets"
retry az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_PROXY}" --network-security-group nsg-proxy -o none
retry az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_APP}" --network-security-group nsg-app -o none
retry az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_DB}" --network-security-group nsg-db -o none
retry az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_METRICS}" --network-security-group nsg-metrics -o none
retry az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_LOADTEST}" --network-security-group nsg-loadtest -o none

log "Network setup complete"
