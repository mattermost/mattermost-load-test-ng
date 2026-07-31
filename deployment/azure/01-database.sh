#!/bin/bash
# Azure Database for PostgreSQL Flexible Server, VNet-integrated (private access only,
# no public network access), on snet-db.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

DB_SERVER_NAME="$(db_server_name)"

DB_PASSWORD="$(gen_secret db_admin_password)"
echo "${DB_PASSWORD}" > "${SECRETS_DIR}/db_admin_password"

# `az postgres flexible-server create --private-dns-zone <name>` has a broken code path when
# the zone doesn't already exist: it rejects EVERY name (tested several unrelated ones) with
# "(PrivateDnsZoneNameNotValid) ... can not be server name plus zone suffix", regardless of
# what the name actually is. Pre-creating the zone + VNet link ourselves and pointing
# --private-dns-zone at the now-existing zone avoids that broken auto-create path entirely.
PRIVATE_DNS_ZONE="testzone123.private.postgres.database.azure.com"

log "Pre-creating private DNS zone ${PRIVATE_DNS_ZONE} (works around a broken CLI auto-create path)"
if az network private-dns zone show -g "${RG}" -n "${PRIVATE_DNS_ZONE}" &>/dev/null; then
  log "Zone ${PRIVATE_DNS_ZONE} already exists, skipping create (unlike most az create calls here, this one isn't idempotent)"
else
  az network private-dns zone create -g "${RG}" -n "${PRIVATE_DNS_ZONE}" -o none
fi
# A zone can only have ONE link to a given VNet, regardless of link name - so check by VNet,
# not by our expected link name (a leftover link from an earlier failed/renamed attempt still
# blocks a new one under a different name with a Conflict error).
EXISTING_LINK="$(az network private-dns link vnet list -g "${RG}" -z "${PRIVATE_DNS_ZONE}" \
  --query "[?ends_with(virtualNetwork.id, '/${VNET}')].name" -o tsv)"
if [ -z "${EXISTING_LINK}" ]; then
  retry az network private-dns link vnet create -g "${RG}" \
    -n "${DB_SERVER_NAME}-dns-link" -z "${PRIVATE_DNS_ZONE}" -v "${VNET}" -e false -o none
else
  log "VNet ${VNET} already linked to ${PRIVATE_DNS_ZONE} as '${EXISTING_LINK}', skipping create"
fi

log "Creating Postgres Flexible Server ${DB_SERVER_NAME} (this takes ~10-15 min)"
retry az postgres flexible-server create \
  -g "${RG}" -n "${DB_SERVER_NAME}" -l "${LOCATION}" \
  --admin-user "${DB_ADMIN_USER}" --admin-password "${DB_PASSWORD}" \
  --sku-name Standard_D4ds_v5 --tier GeneralPurpose \
  --storage-size 128 --version 16 \
  --vnet "${VNET}" --subnet "${SNET_DB}" \
  --private-dns-zone "${PRIVATE_DNS_ZONE}" \
  --yes -o none

log "Creating database ${DB_NAME}"
retry az postgres flexible-server db create -g "${RG}" -s "${DB_SERVER_NAME}" -n "${DB_NAME}" -o none

DB_FQDN="$(az postgres flexible-server show -g "${RG}" -n "${DB_SERVER_NAME}" --query fullyQualifiedDomainName -o tsv)"
echo "${DB_FQDN}" > "${STATE_DIR}/db_fqdn"

log "Postgres Flexible Server ready: ${DB_FQDN} (private access only, VNet: ${VNET}/${SNET_DB})"
