#!/bin/bash
# Replaces ltagent init's synthetic seed data (a couple of teams/channels, no post history)
# with a real production-scale dump, automating the manual procedure documented in
# DEPLOYMENT.md's "Test data: DB dump restore" section. Destructive (DROP DATABASE), so this
# prompts for confirmation before touching anything - same pattern as teardown.sh.
#
# The dump ships its own sysadmin account (sysadmin@sample.mattermost.com / Sys@dmin-sample1),
# which replaces whatever sysadmin existed before the restore (see DEPLOYMENT.md's "Sysadmin
# account" section) - this script updates secrets/ and the load-test VM's local config.json to
# match. deployment/azure/operator/config.json (rendered by 08-operator-setup.sh) already
# hardcodes these dump credentials regardless of what order the two scripts run in.
#
# The Postgres Flexible Server has no public network access (private, VNet-only) - so both the
# DROP/CREATE DATABASE and the streamed restore itself have to run from inside the VNet, over
# SSH to mm-app-1, not from the operator's own machine.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source lib/common.sh

DB_SERVER_NAME="$(db_server_name)"

DUMP_URL="${1:-https://lt-public-data.s3.amazonaws.com/12M_610_fixed_psql.sql.gz}"
DUMP_ADMIN_EMAIL="sysadmin@sample.mattermost.com"
DUMP_ADMIN_PASSWORD="Sys@dmin-sample1"

if ! az group show -n "${RG}" &>/dev/null; then
  echo "Resource group ${RG} does not exist - nothing to restore into." >&2
  exit 1
fi

echo "This will DROP and recreate the '${DB_NAME}' database on ${DB_SERVER_NAME}, replacing"
echo "every team/channel/user currently on the server with the dataset from:"
echo "  ${DUMP_URL}"
echo "The dump's own sysadmin (${DUMP_ADMIN_EMAIL}) becomes the working admin account afterward."
read -r -p "Type 'restore' to confirm: " CONFIRM
if [ "${CONFIRM}" != "restore" ]; then
  echo "Confirmation did not match, aborting."
  exit 1
fi

DB_FQDN="$(cat "${STATE_DIR}/db_fqdn")"
DB_PASSWORD="$(cat "${SECRETS_DIR}/db_admin_password")"
APP1_PUB="$(vm_public_ip mm-app-1)"
APP2_PUB="$(vm_public_ip mm-app-2)"
LOADTEST_PUB="$(vm_public_ip mm-loadtest)"
PROXY_IP="$(vm_private_ip mm-proxy)"

ssh_app1() { ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 "${VM_ADMIN_USER}@${APP1_PUB}" "$@"; }
ssh_app2() { ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 "${VM_ADMIN_USER}@${APP2_PUB}" "$@"; }

wait_for_mattermost() {
  local host="$1" label="$2"
  for i in $(seq 1 30); do
    ssh -o ConnectTimeout=10 "${VM_ADMIN_USER}@${host}" "curl -sf http://localhost:8065/api/v4/system/ping >/dev/null" && return 0
    echo "  waiting for mattermost on ${label} (${i}/30)..."
    sleep 5
  done
  echo "ERROR: mattermost on ${label} never came up" >&2
  return 1
}

log "1/7 Stopping mattermost.service on app-1 and app-2"
ssh_app1 "sudo systemctl stop mattermost"
ssh_app2 "sudo systemctl stop mattermost"

log "2/7 Dropping and recreating database '${DB_NAME}' on ${DB_SERVER_NAME}"
cat > "${STATE_DIR}/restore-dump.sh" <<EOF
#!/bin/bash
set -euo pipefail
CONNINFO_MAINT="host=${DB_FQDN} port=5432 dbname=postgres user=${DB_ADMIN_USER} sslmode=require"
CONNINFO_DB="host=${DB_FQDN} port=5432 dbname=${DB_NAME} user=${DB_ADMIN_USER} sslmode=require"

echo "Terminating any lingering connections to ${DB_NAME}..."
PGPASSWORD='${DB_PASSWORD}' psql "\${CONNINFO_MAINT}" -v ON_ERROR_STOP=0 -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();"

echo "Dropping and recreating ${DB_NAME}..."
PGPASSWORD='${DB_PASSWORD}' psql "\${CONNINFO_MAINT}" -v ON_ERROR_STOP=1 \
  -c "DROP DATABASE IF EXISTS ${DB_NAME};" -c "CREATE DATABASE ${DB_NAME};"

echo "Streaming restore from ${DUMP_URL} (can take a while; errors about role \"rdsadmin\"/"
echo "\"mmuser\" not existing are expected and benign - those are ownership statements from the"
echo "dump's original AWS RDS source, not applicable here)..."
curl -sL '${DUMP_URL}' | gunzip | PGPASSWORD='${DB_PASSWORD}' psql "\${CONNINFO_DB}" -v ON_ERROR_STOP=0

echo "Row counts:"
PGPASSWORD='${DB_PASSWORD}' psql "\${CONNINFO_DB}" -c \
  "SELECT (SELECT count(*) FROM Posts) AS posts, (SELECT count(*) FROM Users) AS users, (SELECT count(*) FROM Teams) AS teams;"
EOF
chmod +x "${STATE_DIR}/restore-dump.sh"
scp -o StrictHostKeyChecking=accept-new "${STATE_DIR}/restore-dump.sh" "${VM_ADMIN_USER}@${APP1_PUB}:/tmp/restore-dump.sh"

log "3/7 Streaming the restore from mm-app-1 (avoids storing the multi-GB uncompressed SQL on disk; can take 15-30+ min)"
ssh_app1 "bash /tmp/restore-dump.sh; rm -f /tmp/restore-dump.sh"

log "4/7 Restarting mattermost.service - app-1 first (watch it migrate the dump's schema), then app-2"
ssh_app1 "sudo systemctl start mattermost"
wait_for_mattermost "${APP1_PUB}" "app-1"
ssh_app2 "sudo systemctl start mattermost"
wait_for_mattermost "${APP2_PUB}" "app-2"

log "5/7 Re-uploading the license (the restore wiped it - license state lives in the DB)"
LICENSE_B64="$(base64 -w0 "${LICENSE_FILE}")"
ssh_app1 "
  echo '${LICENSE_B64}' | base64 -d > /tmp/loadtest.mattermost-license
  TOKEN=\$(curl -sS -i -X POST http://localhost:8065/api/v4/users/login \
    -H 'Content-Type: application/json' \
    -d '{\"login_id\":\"${DUMP_ADMIN_EMAIL}\",\"password\":\"${DUMP_ADMIN_PASSWORD}\"}' \
    | grep -i '^Token:' | awk '{print \$2}' | tr -d '\r')
  if [ -z \"\$TOKEN\" ]; then
    echo 'ERROR: could not log in as the dump sysadmin to upload the license' >&2
    exit 1
  fi
  curl -sS -X POST http://localhost:8065/api/v4/license \
    -H \"Authorization: Bearer \$TOKEN\" \
    -F 'license=@/tmp/loadtest.mattermost-license'
  rm -f /tmp/loadtest.mattermost-license
  echo
"

log "6/7 Restarting both nodes once more so they pick up the license/cluster state from the DB cleanly"
ssh_app1 "sudo systemctl restart mattermost"
wait_for_mattermost "${APP1_PUB}" "app-1"
ssh_app2 "sudo systemctl restart mattermost"
wait_for_mattermost "${APP2_PUB}" "app-2"
echo "Cluster status (expect both app-1 and app-2):"
ssh_app1 "
  TOKEN=\$(curl -sS -i -X POST http://localhost:8065/api/v4/users/login \
    -H 'Content-Type: application/json' \
    -d '{\"login_id\":\"${DUMP_ADMIN_EMAIL}\",\"password\":\"${DUMP_ADMIN_PASSWORD}\"}' \
    | grep -i '^Token:' | awk '{print \$2}' | tr -d '\r')
  curl -s http://localhost:8065/api/v4/cluster/status -H \"Authorization: Bearer \$TOKEN\"
  echo
"

log "7/7 Switching secrets/ and the load-test VM's local config.json to the dump's sysadmin"
echo "${DUMP_ADMIN_PASSWORD}" > "${SECRETS_DIR}/mm_admin_password"
cat > "${SECRETS_DIR}/mm_admin_note.txt" <<NOTE
The DB dump (${DUMP_URL##*/}) ships its own sysadmin account, restored via
09-restore-dump.sh. Any sysadmin account created before the restore (e.g. by 07-post-config.sh's
signup-API bootstrap) was wiped along with the rest of the database.

Login: ${DUMP_ADMIN_EMAIL} / ${DUMP_ADMIN_PASSWORD} (username: sysadmin)
NOTE

PROXY_IP="${PROXY_IP}" MM_ADMIN_PASSWORD="${DUMP_ADMIN_PASSWORD}" \
  envsubst '${PROXY_IP} ${MM_ADMIN_PASSWORD}' < "${ASSETS_DIR}/config.json.tmpl" > "${STATE_DIR}/loadtest-config-restored.json"
python3 -c "
import json
with open('${STATE_DIR}/loadtest-config-restored.json') as f:
    d = json.load(f)
d['ConnectionConfiguration']['AdminEmail'] = '${DUMP_ADMIN_EMAIL}'
with open('${STATE_DIR}/loadtest-config-restored.json', 'w') as f:
    json.dump(d, f, indent=2)
"
# jq isn't installed on the load-test VM (a prior in-place-patch attempt via jq failed silently
# because of this) - so regenerate the whole file locally from the template instead and copy it
# over, rather than trying to patch it in place on the VM.
#
# The repo clone on that VM may be root-owned (provisioning runs via az vm run-command, i.e. as
# root) on deployments predating the fix in provision-loadtest.sh.tmpl - reassert ownership
# before scp'ing so this doesn't fail with "Permission denied" on older deployments.
ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 "${VM_ADMIN_USER}@${LOADTEST_PUB}" \
  "sudo chown -R ${VM_ADMIN_USER}:${VM_ADMIN_USER} mattermost-load-test-ng"
scp -o StrictHostKeyChecking=accept-new "${STATE_DIR}/loadtest-config-restored.json" \
  "${VM_ADMIN_USER}@${LOADTEST_PUB}:mattermost-load-test-ng/config/config.json"

log "Done. Real dataset restored; sysadmin is now ${DUMP_ADMIN_EMAIL} / ${DUMP_ADMIN_PASSWORD}"
echo "deployment/azure/operator/config.json already points at these credentials regardless of"
echo "whether 08-operator-setup.sh ran before or after this script."
