#!/bin/bash
# Shared config/helpers sourced by every deployment/azure/*.sh script.
set -euo pipefail

RG="rg-mm-loadtest-ha"
# eastus was the original choice but this subscription has no Postgres Flexible Server
# capacity there and the Dsv5 VM family is NotAvailableForSubscription in eastus too;
# westus2 has both available (confirmed via `az postgres flexible-server list-skus` /
# `az vm list-skus --all`), so using that instead. VM sizes also bumped v5->v6 for the
# same subscription-quota reason (see 03/04/05/06 scripts).
LOCATION="westus2"
VNET="vnet-mm-loadtest"

SNET_PROXY="snet-proxy";    SNET_PROXY_CIDR="10.20.1.0/24"
SNET_APP="snet-app";        SNET_APP_CIDR="10.20.2.0/24"
SNET_DB="snet-db";          SNET_DB_CIDR="10.20.3.0/24"
SNET_METRICS="snet-metrics"; SNET_METRICS_CIDR="10.20.4.0/24"
SNET_LOADTEST="snet-loadtest"; SNET_LOADTEST_CIDR="10.20.5.0/24"

VM_ADMIN_USER="mmadmin"
SSH_PUB_KEY="${HOME}/.ssh/id_ed25519.pub"

MM_VERSION="v11.9.0"
MM_DOWNLOAD_URL="https://releases.mattermost.com/${MM_VERSION#v}/mattermost-${MM_VERSION#v}-linux-amd64.tar.gz"
LICENSE_FILE="/home/alejandro/credentials/licenses/loadtest-dev-license-500K.mattermost-license"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
AZURE_DIR="${REPO_ROOT}/deployment/azure"
ASSETS_DIR="${AZURE_DIR}/assets"
SECRETS_DIR="${REPO_ROOT}/secrets"
STATE_DIR="${AZURE_DIR}/.state"
mkdir -p "${SECRETS_DIR}" "${STATE_DIR}"

DB_NAME="mattermost"
DB_ADMIN_USER="mmdbadmin"

# DNS labels are deterministic (based on the cached random suffix) so scripts that run before
# the proxy/metrics VMs exist (e.g. app-node SiteURL) can still reference their final FQDN.
proxy_fqdn()   { echo "mm-lt-proxy-$(dns_suffix).${LOCATION}.cloudapp.azure.com"; }
metrics_fqdn() { echo "mm-lt-grafana-$(dns_suffix).${LOCATION}.cloudapp.azure.com"; }
# Postgres Flexible Server names are globally unique across all of Azure (like storage
# accounts), not just within this subscription/RG - a plain "mm-loadtest-pg" collided with an
# unrelated server already owned by another user in the same subscription. Suffixed the same
# way as the other globally-unique names above to avoid that.
db_server_name() { echo "mm-loadtest-pg-$(dns_suffix)"; }

STORAGE_ACCOUNT_FILE="${STATE_DIR}/storage_account_name"
DNS_SUFFIX_FILE="${STATE_DIR}/dns_suffix"

log() { echo -e "\n\033[1;36m==> $*\033[0m"; }

# retry <cmd...> - retries a command up to 6 times with backoff. ARM has a well-known
# cross-replica read-after-write lag where a resource that was just created 404s on the very
# next call (even after the create call itself returned success, and even after polling
# 'provisioningState' on the same resource) - a plain retry-with-backoff on the *dependent*
# call is the robust fix, not just waiting on the resource that was created.
retry() {
  local attempt=1 max=6 delay=5
  until "$@"; do
    if [ "${attempt}" -ge "${max}" ]; then
      echo "retry: giving up after ${attempt} attempts: $*" >&2
      return 1
    fi
    echo "retry: attempt ${attempt}/${max} failed, retrying in ${delay}s: $*" >&2
    sleep "${delay}"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

# A short, stable random suffix for globally-unique names (storage account, DNS labels).
dns_suffix() {
  if [ ! -f "${DNS_SUFFIX_FILE}" ]; then
    tr -dc 'a-z0-9' < /dev/urandom | head -c8 > "${DNS_SUFFIX_FILE}"
  fi
  cat "${DNS_SUFFIX_FILE}"
}

my_ip() {
  local ip_file="${STATE_DIR}/operator_ip"
  if [ ! -f "${ip_file}" ]; then
    curl -s https://api.ipify.org > "${ip_file}"
  fi
  cat "${ip_file}"
}

gen_secret() {
  # gen_secret <name> -> writes/reads secrets/<name> and echoes the value
  local name="$1"
  local file="${SECRETS_DIR}/${name}"
  if [ ! -f "${file}" ]; then
    openssl rand -base64 24 | tr -dc 'A-Za-z0-9' | head -c24 > "${file}"
  fi
  cat "${file}"
}

vm_private_ip() {
  az vm list-ip-addresses -g "${RG}" -n "$1" --query "[0].virtualMachine.network.privateIpAddresses[0]" -o tsv
}

vm_public_ip() {
  az vm list-ip-addresses -g "${RG}" -n "$1" --query "[0].virtualMachine.network.publicIpAddresses[0].ipAddress" -o tsv
}

vm_fqdn() {
  az network public-ip show -g "${RG}" -n "$1-pip" --query "dnsSettings.fqdn" -o tsv
}

# run_on_vm <vm-name> <local-script-path> [arg]...
# Uploads and executes a script on a VM via `az vm run-command`, streaming output.
run_on_vm() {
  local vm="$1"; shift
  local script="$1"; shift
  az vm run-command invoke -g "${RG}" -n "${vm}" \
    --command-id RunShellScript \
    --scripts "@${script}" \
    ${1+--parameters "$@"} \
    -o json | tee "${STATE_DIR}/${vm}-$(basename "${script}").log" \
    | python3 -c 'import json,sys; d=json.load(sys.stdin); [print(m.get("message","")) for m in d.get("value",[])]'
}
