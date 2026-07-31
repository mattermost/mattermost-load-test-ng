#!/bin/bash
# Storage account + SMB file share for /opt/mattermost/data, shared between both app nodes.
# Public network access disabled; only snet-app is allowed in via a VNet service-endpoint rule.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

SUFFIX="$(dns_suffix)"
STORAGE_ACCOUNT="mmlt${SUFFIX}"
echo "${STORAGE_ACCOUNT}" > "${STORAGE_ACCOUNT_FILE}"

log "Enabling Microsoft.Storage service endpoint on ${SNET_APP}"
az network vnet subnet update -g "${RG}" --vnet-name "${VNET}" -n "${SNET_APP}" \
  --service-endpoints Microsoft.Storage -o none

log "Creating storage account ${STORAGE_ACCOUNT}"
az storage account create -g "${RG}" -n "${STORAGE_ACCOUNT}" -l "${LOCATION}" \
  --sku Standard_LRS --kind StorageV2 \
  --default-action Deny --bypass AzureServices \
  -o none

retry az storage account network-rule add -g "${RG}" --account-name "${STORAGE_ACCOUNT}" \
  --vnet-name "${VNET}" --subnet "${SNET_APP}" -o none

STORAGE_KEY="$(az storage account keys list -g "${RG}" -n "${STORAGE_ACCOUNT}" --query "[0].value" -o tsv)"
echo "${STORAGE_KEY}" > "${SECRETS_DIR}/storage_account_key"

log "Creating file share 'mattermost-data' (via ARM, bypasses the data-plane network rule)"
retry az storage share-rm create -g "${RG}" --storage-account "${STORAGE_ACCOUNT}" \
  -n mattermost-data --quota 256 -o none

log "Storage ready: account=${STORAGE_ACCOUNT} share=mattermost-data (network-restricted to ${SNET_APP})"
