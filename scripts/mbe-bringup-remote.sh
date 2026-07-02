#!/bin/bash
# Runs as root on an MBE app node to bring the HSM + plugin + feature-flag environment up from a
# stock deployer.json deployment. Idempotent — safe to re-run.
#
# This exists because none of the "obvious" config-driven seams actually work (verified twice,
# M0 + M1 — see mbe-load-test-plan.md WS3): MattermostConfigPatchFile's FeatureFlags section is
# silently ignored, and deployer.json's MattermostPlugins map extracts the plugin to disk but
# never registers/enables it. Every step below is the manual workaround, scripted so it can't be
# partially skipped the way it was by hand in M1 (the SoftHSM usermod/chown gap recurred there).
#
# Expected to be invoked via mbe-bringup.sh, which prepends `export VAR=...` lines for the
# variables below before piping this file to `ltctl ssh <resource> --run "sudo bash -s"`. Can also
# be run directly on a node (as root) with the same env vars exported first.
set -euo pipefail

: "${TOKEN_LABEL:=pbe-dev}"
: "${PIN:=1234}"
: "${KEK_LABEL:=pbe-kek-dev}"
: "${PLUGIN_ID:=message-based-encryption}"
: "${PLUGIN_TARBALL:=/home/ubuntu/${PLUGIN_ID}.tar.gz}"
: "${SERVICE_USER:=ubuntu}"
: "${ADMIN_EMAIL:=sysadmin@sample.mattermost.com}"
: "${ADMIN_PASSWORD:=Sys@dmin-sample1}"

LIB_PATH="/usr/lib/softhsm/libsofthsm2.so"
SOFTHSM2_CONF_PATH="/etc/softhsm/softhsm2.conf"
BASE_URL="http://localhost:8065"
METRICS_URL="http://localhost:8067/metrics"
STEP="init"

on_error() {
	echo "MBE BRING-UP: FAILED at step: ${STEP}" >&2
	exit 1
}
trap on_error ERR

step() {
	STEP="$1"
	echo "--- ${STEP} ---"
}

# retry runs its argument list up to 3 times, matching the retry convention already used by
# deployment/terraform/assets/provisioners/debian/app.sh for the same reason: apt-get against the
# Ubuntu/PostgreSQL mirrors from these AWS nodes flakes transiently often enough to be worth it.
retry() {
	local n=0
	until "$@"; do
		n=$((n + 1))
		if [ "$n" -ge 3 ]; then
			return 1
		fi
		echo "attempt ${n} failed, retrying: $*" >&2
		sleep 2
	done
}

if [ "$(id -u)" -ne 0 ]; then
	echo "must be run as root (sudo)" >&2
	exit 1
fi

# ---------------------------------------------------------------------------
step "install SoftHSM2 + self-check tooling packages"
# ---------------------------------------------------------------------------
export DEBIAN_FRONTEND=noninteractive
retry apt-get -y update
# jq/curl aren't guaranteed present on the base AMI; softhsm2/libsofthsm2-dev/opensc are the HSM
# packages proper. All of this is required by later steps in this same script, not optional.
retry apt-get install -y softhsm2 libsofthsm2-dev opensc jq curl

# ---------------------------------------------------------------------------
step "write softhsm2.conf"
# ---------------------------------------------------------------------------
mkdir -p "$(dirname "$SOFTHSM2_CONF_PATH")"
cat >"$SOFTHSM2_CONF_PATH" <<EOF
directories.tokendir = /var/lib/softhsm/tokens/
objectstore.backend = file
EOF
export SOFTHSM2_CONF="$SOFTHSM2_CONF_PATH"
mkdir -p /var/lib/softhsm/tokens

# ---------------------------------------------------------------------------
step "init PKCS#11 token (idempotent)"
# ---------------------------------------------------------------------------
# No end-anchor: `--show-slots` pads the Label field with trailing spaces to a fixed column width,
# so a `$`-anchored match never fires and silently defeats idempotency (confirmed: caused a
# duplicate same-labeled token on a re-run). Matches ci/scripts/setup-hsm.sh's check, unanchored.
if softhsm2-util --show-slots 2>/dev/null | grep -qE "Label:[[:space:]]+${TOKEN_LABEL}([[:space:]]|$)"; then
	echo "token '${TOKEN_LABEL}' already exists, skipping init"
else
	softhsm2-util --init-token --free --label "$TOKEN_LABEL" --pin "$PIN" --so-pin "$PIN"
fi

# ---------------------------------------------------------------------------
step "generate AES-256 KEK (idempotent)"
# ---------------------------------------------------------------------------
# Note: --init-token above assigns the token to a new random slot ID every time it actually runs,
# so every pkcs11-tool call must address it by --token-label, never a cached --slot number.
if pkcs11-tool --module "$LIB_PATH" --token-label "$TOKEN_LABEL" --pin "$PIN" \
	--list-objects --type secrkey 2>/dev/null | grep -qE "label:[[:space:]]+${KEK_LABEL}([[:space:]]|$)"; then
	echo "KEK '${KEK_LABEL}' already exists, skipping generation"
else
	pkcs11-tool --module "$LIB_PATH" --token-label "$TOKEN_LABEL" --pin "$PIN" \
		--keygen --key-type aes:32 --label "$KEK_LABEL" \
		--usage-wrap --usage-decrypt --sensitive
fi

# ---------------------------------------------------------------------------
step "fix SoftHSM permissions (both usermod AND chown are required, see M0/M1 notes)"
# ---------------------------------------------------------------------------
# /etc/softhsm/ is root:softhsm mode 750 -- without group membership the service user can't even
# traverse into it (CKR_GENERAL_ERROR on module init). Group membership only takes effect for new
# process starts, hence the mattermost restart below is load-bearing, not optional.
usermod -aG softhsm "$SERVICE_USER"
# The token directory needs to be *writable* by the service user too -- a `generation` counter
# file is touched on every session open, not just read.
chown -R "${SERVICE_USER}:${SERVICE_USER}" /var/lib/softhsm/tokens/

# ---------------------------------------------------------------------------
step "write systemd drop-in (HSM env + feature flags)"
# ---------------------------------------------------------------------------
# MattermostConfigPatchFile's FeatureFlags section does NOT apply (confirmed M0 + M1) -- these
# must be env vars, in the same drop-in as the HSM library path so both land in one restart.
mkdir -p /etc/systemd/system/mattermost.service.d
cat >/etc/systemd/system/mattermost.service.d/mbe-env.conf <<EOF
[Service]
Environment="MBE_PKCS11_LIBRARY_PATH=${LIB_PATH}"
Environment="SOFTHSM2_CONF=${SOFTHSM2_CONF_PATH}"
Environment="MM_FEATUREFLAGS_CONSUMEPOSTHOOK=true"
Environment="MM_FEATUREFLAGS_AGGREGATEPLUGINMETRICS=true"
EOF

# ---------------------------------------------------------------------------
step "restart mattermost and wait for health"
# ---------------------------------------------------------------------------
systemctl daemon-reload
systemctl restart mattermost

WAITED=0
MAX_WAIT=120
until curl -sf "${BASE_URL}/api/v4/system/ping" >/dev/null 2>&1; do
	if [ "$WAITED" -ge "$MAX_WAIT" ]; then
		echo "mattermost did not become healthy within ${MAX_WAIT}s" >&2
		journalctl -u mattermost --no-pager -n 100 >&2 || true
		exit 1
	fi
	sleep 5
	WAITED=$((WAITED + 5))
done
echo "mattermost healthy after ${WAITED}s"

# ---------------------------------------------------------------------------
step "log in as sysadmin"
# ---------------------------------------------------------------------------
LOGIN_HEADERS="$(mktemp)"
LOGIN_BODY="$(mktemp)"
trap 'rm -f "$LOGIN_HEADERS" "$LOGIN_BODY"' EXIT
HTTP_CODE=$(curl -s -o "$LOGIN_BODY" -D "$LOGIN_HEADERS" -w '%{http_code}' \
	-X POST "${BASE_URL}/api/v4/users/login" \
	-H 'Content-Type: application/json' \
	-d "{\"login_id\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}")
if [ "$HTTP_CODE" != "200" ]; then
	echo "login failed (HTTP ${HTTP_CODE}): $(cat "$LOGIN_BODY")" >&2
	exit 1
fi
TOKEN=$(grep -i '^token:' "$LOGIN_HEADERS" | awk '{print $2}' | tr -d '\r')
ADMIN_ID=$(jq -r '.id' "$LOGIN_BODY")
if [ -z "$TOKEN" ] || [ -z "$ADMIN_ID" ] || [ "$ADMIN_ID" = "null" ]; then
	echo "failed to extract token/admin id from login response" >&2
	exit 1
fi
auth() { curl -s -H "Authorization: Bearer ${TOKEN}" "$@"; }

# ---------------------------------------------------------------------------
step "verify feature flags actually took effect"
# ---------------------------------------------------------------------------
CONFIG_JSON=$(auth "${BASE_URL}/api/v4/config")
CONSUME_POST_HOOK=$(echo "$CONFIG_JSON" | jq -r '.FeatureFlags.ConsumePostHook')
AGGREGATE_PLUGIN_METRICS=$(echo "$CONFIG_JSON" | jq -r '.FeatureFlags.AggregatePluginMetrics')
if [ "$CONSUME_POST_HOOK" != "true" ] || [ "$AGGREGATE_PLUGIN_METRICS" != "true" ]; then
	echo "feature flags did not take effect: ConsumePostHook=${CONSUME_POST_HOOK} AggregatePluginMetrics=${AGGREGATE_PLUGIN_METRICS}" >&2
	echo "expected both 'true' -- check the systemd drop-in actually loaded (systemctl show mattermost -p Environment)" >&2
	exit 1
fi
echo "FeatureFlags.ConsumePostHook=true, FeatureFlags.AggregatePluginMetrics=true"

# ---------------------------------------------------------------------------
step "install + enable plugin (idempotent)"
# ---------------------------------------------------------------------------
ACTIVE=$(auth "${BASE_URL}/api/v4/plugins" | jq -r --arg id "$PLUGIN_ID" '.active[]?.id' | grep -Fx "$PLUGIN_ID" || true)
if [ -n "$ACTIVE" ]; then
	echo "plugin '${PLUGIN_ID}' already active, skipping install"
else
	if [ ! -f "$PLUGIN_TARBALL" ]; then
		echo "plugin tarball not found at ${PLUGIN_TARBALL} -- did deployer.json's MattermostPlugins install it?" >&2
		exit 1
	fi
	INSTALL_BODY="$(mktemp)"
	HTTP_CODE=$(curl -s -o "$INSTALL_BODY" -w '%{http_code}' \
		-H "Authorization: Bearer ${TOKEN}" \
		-X POST "${BASE_URL}/api/v4/plugins" \
		-F "plugin=@${PLUGIN_TARBALL}")
	if [ "$HTTP_CODE" != "201" ]; then
		echo "plugin install failed (HTTP ${HTTP_CODE}): $(cat "$INSTALL_BODY")" >&2
		exit 1
	fi
	rm -f "$INSTALL_BODY"

	ENABLE_BODY="$(mktemp)"
	HTTP_CODE=$(curl -s -o "$ENABLE_BODY" -w '%{http_code}' \
		-H "Authorization: Bearer ${TOKEN}" \
		-X POST "${BASE_URL}/api/v4/plugins/${PLUGIN_ID}/enable")
	if [ "$HTTP_CODE" != "200" ]; then
		echo "plugin enable failed (HTTP ${HTTP_CODE}): $(cat "$ENABLE_BODY")" >&2
		exit 1
	fi
	rm -f "$ENABLE_BODY"

	WAITED=0
	MAX_WAIT=30
	while true; do
		ACTIVE=$(auth "${BASE_URL}/api/v4/plugins" | jq -r --arg id "$PLUGIN_ID" '.active[]?.id' | grep -Fx "$PLUGIN_ID" || true)
		[ -n "$ACTIVE" ] && break
		if [ "$WAITED" -ge "$MAX_WAIT" ]; then
			echo "plugin did not activate within ${MAX_WAIT}s" >&2
			exit 1
		fi
		sleep 1
		WAITED=$((WAITED + 1))
	done
	echo "plugin '${PLUGIN_ID}' active after ${WAITED}s"
fi

# ---------------------------------------------------------------------------
step "grant EM + test-connection (end-to-end HSM/KEK smoke test)"
# ---------------------------------------------------------------------------
# The single most authoritative check available: proves the plugin can actually reach the HSM and
# unwrap the KEK, not just that the process is running. Mirrors the same check the plugin repo's
# own e2e suite runs (ci/scripts/start.sh). The grant is idempotent (ON CONFLICT DO UPDATE per the
# plugin's own docs) and is also exactly what cmd/ltmbebootstrap needs from this same admin later,
# so leaving it granted is intentional, not a leftover.
GRANT_CODE=$(curl -s -o /dev/null -w '%{http_code}' \
	-H "Authorization: Bearer ${TOKEN}" \
	-X POST "${BASE_URL}/plugins/${PLUGIN_ID}/encryption-manager/${ADMIN_ID}")
if [ "$GRANT_CODE" != "200" ]; then
	echo "EM grant failed (HTTP ${GRANT_CODE})" >&2
	exit 1
fi

TEST_CONN_BODY="$(mktemp)"
HTTP_CODE=$(curl -s -o "$TEST_CONN_BODY" -w '%{http_code}' \
	-H "Authorization: Bearer ${TOKEN}" \
	-X POST "${BASE_URL}/plugins/${PLUGIN_ID}/test-connection")
if [ "$HTTP_CODE" != "200" ]; then
	echo "test-connection failed (HTTP ${HTTP_CODE}): $(cat "$TEST_CONN_BODY")" >&2
	echo "this means HSM/KEK access is broken even though the plugin process is active -- check the usermod/chown step and the mattermost service log for CKR_* errors" >&2
	exit 1
fi
echo "test-connection ok: $(cat "$TEST_CONN_BODY")"
rm -f "$TEST_CONN_BODY"

# ---------------------------------------------------------------------------
step "force the metrics subsystem to re-read FeatureFlags.AggregatePluginMetrics"
# ---------------------------------------------------------------------------
# Verified against server/channels/app/platform/service.go + metrics.go (release-11.9): the
# aggregation wrapper is decided ONCE, in PlatformService.HandleMetrics, at the moment
# resetMetrics() runs -- and resetMetrics() is only re-triggered by a live config-change listener
# that watches MetricsSettings.Enable/ListenAddress specifically, NOT FeatureFlags. At boot,
# resetMetrics() (Step 10 of platform.New()) runs before FeatureFlags' env-var overrides are
# resolved, so it locks in "don't wrap" even though `/api/v4/config` reports the flag as true
# moments later. A no-op MetricsSettings.Enable false->true patch re-triggers resetMetrics() with
# the now-current (true) flag value, wiring up the plugin-aggregation wrapper. Confirmed empirically:
# without this step, /metrics never carries plugin_id="..." series at all, even hours after boot.
for enable in false true; do
	PATCH_BODY="$(mktemp)"
	HTTP_CODE=$(curl -s -o "$PATCH_BODY" -w '%{http_code}' \
		-H "Authorization: Bearer ${TOKEN}" -H 'Content-Type: application/json' \
		-X PUT -d "{\"MetricsSettings\":{\"Enable\":${enable}}}" \
		"${BASE_URL}/api/v4/config/patch")
	if [ "$HTTP_CODE" != "200" ]; then
		echo "config/patch MetricsSettings.Enable=${enable} failed (HTTP ${HTTP_CODE}): $(cat "$PATCH_BODY")" >&2
		exit 1
	fi
	rm -f "$PATCH_BODY"
done
sleep 2

# ---------------------------------------------------------------------------
step "verify aggregated /metrics carries the MBE series"
# ---------------------------------------------------------------------------
METRICS_BODY=$(curl -sf "$METRICS_URL")

if ! echo "$METRICS_BODY" | grep -q 'plugin_id="'"${PLUGIN_ID}"'"'; then
	echo "no series with plugin_id=\"${PLUGIN_ID}\" found in ${METRICS_URL} -- AggregatePluginMetrics wiring is broken even though the flag reads true" >&2
	exit 1
fi

# These two are single (non-vec) metrics that are always registered at plugin activation, so they
# appear with a zero value even before any real post/read traffic -- unlike the per-hook/per-op
# counters (hook_invocations_total, crypto_operations_total, ...), which are label-vec metrics
# that only materialize once real traffic hits them. Those are verified separately, by the
# load-test run / ltmbebootstrap round-trip, not by this bring-up step.
# The aggregator (addPluginLabelToMetrics) injects plugin_id="..." as a label on every plugin
# series, so an originally-unlabeled metric like these two gains a `{plugin_id="..."}` block --
# e.g. `..._sum{plugin_id="message-based-encryption"} 0`, not a bare `..._sum 0`. Match either a
# label block or a plain space after the (optional) suffix, don't assume no labels.
for metric in mattermost_plugin_mbe_decrypt_item_failures_total mattermost_plugin_mbe_classification_store_duration_seconds; do
	if ! echo "$METRICS_BODY" | grep -qE "^${metric}(_bucket|_sum|_count)?(\{|[[:space:]])"; then
		echo "expected metric '${metric}' not found in ${METRICS_URL}" >&2
		exit 1
	fi
done

# Regression guard: M0 shipped a dashboard once querying the bare "mbe_*" prefix instead of the
# real "mattermost_plugin_mbe_*" one. The plugin itself was never wrong, but a bare-prefix series
# appearing here would mean something upstream (a stale plugin build) regressed it.
if echo "$METRICS_BODY" | grep -qE '^mbe_[a-z_]+(\{|[[:space:]])'; then
	echo "found an unprefixed mbe_* metric -- unexpected, check the plugin build's metrics namespace" >&2
	exit 1
fi
echo "aggregated /metrics carries plugin_id=\"${PLUGIN_ID}\" and the always-on MBE series"

echo "MBE BRING-UP: PASS"
