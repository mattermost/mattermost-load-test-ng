#!/bin/bash
# Wrapper for scripts/mbe-bringup-remote.sh: drives the MBE HSM/plugin/feature-flag bring-up (and
# its self-check) on a deployed app node over `ltctl ssh`, without needing a second SCP step or
# any reachability into the deployment beyond what `ltctl` itself already has.
#
# Usage:
#   scripts/mbe-bringup.sh -c mbe-m1/deployer-a.json [-r <cluster>-app-0] [options...]
#
# All configuration is piped to the remote script over stdin as `export VAR='value'` lines, never
# interpolated into the `ltctl ssh --run` string itself -- that string does its own naive
# space-splitting (see cmd/ltctl/main.go), which would silently mangle any value containing a
# space or shell metacharacter. Values here are single-quoted for the remote shell, so anything
# (including the default admin password's '@') survives intact.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REMOTE_SCRIPT="$SCRIPT_DIR/mbe-bringup-remote.sh"

# ltctl has no `go install`/PATH convention in this repo -- on ltserver it's a manually built
# binary, rebuilt on every cmd/ltctl change via `go build -o ~/cp/ltctl ./cmd/ltctl`, per Chris.
# Resolution order: explicit --ltctl-bin/$LTCTL_BIN override > ~/cp/ltctl (the ltserver
# convention) > `ltctl` on PATH > `go run ./cmd/ltctl` as a last resort (works anywhere but is
# slow and, since it doesn't rebuild a stale binary for you, is not a substitute for rebuilding
# after a cmd/ltctl change on a host that already has ~/cp/ltctl).
LTCTL_BIN="${LTCTL_BIN:-}"
CONFIG=""
RESOURCE=""
TOKEN_LABEL="pbe-dev"
PIN="1234"
KEK_LABEL="pbe-kek-dev"
PLUGIN_ID="message-based-encryption"
PLUGIN_TARBALL=""
SERVICE_USER="ubuntu"
ADMIN_EMAIL="sysadmin@sample.mattermost.com"
ADMIN_PASSWORD="Sys@dmin-sample1"

usage() {
	cat <<EOF
Usage: $0 -c <deployer.json> [-r <resource>] [options]

  -c, --config <path>          deployer.json for the target deployment (required, forwarded to ltctl -c)
  -r, --resource <name>        ltctl ssh resource name (default: <ClusterName>-app-0, derived from -c;
                                ltctl ssh resource names are cluster-prefixed, e.g. 'cpmbem0-app-0', NOT
                                bare 'app-0' -- run 'ltctl ssh -c <config>' with no args to list them.
                                Use <ClusterName>-app-1, -app-2, ... for other app nodes)
      --ltctl-bin <path>       path to the ltctl binary (default: \$LTCTL_BIN, else ~/cp/ltctl if present,
                                else 'ltctl' on PATH, else 'go run ./cmd/ltctl'). If you've changed
                                cmd/ltctl, rebuild first: go build -o ~/cp/ltctl ./cmd/ltctl -- this
                                script will NOT rebuild a stale binary for you.
      --token-label <label>    SoftHSM token label (default: pbe-dev)
      --pin <pin>               SoftHSM PIN (default: 1234)
      --kek-label <label>      SoftHSM KEK label (default: pbe-kek-dev)
      --plugin-id <id>         MBE plugin id (default: message-based-encryption)
      --plugin-tarball <path>  plugin tarball path *on the app node* (default: /home/<service-user>/<plugin-id>.tar.gz,
                                which is where deployer.json's MattermostPlugins already puts it)
      --service-user <user>    OS user mattermost runs as (default: ubuntu)
      --admin-email <email>    sysadmin login used for plugin install/enable/EM-grant (default: sysadmin@sample.mattermost.com)
      --admin-password <pw>    sysadmin password (default: Sys@dmin-sample1)
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	-c | --config)
		CONFIG="$2"
		shift 2
		;;
	-r | --resource)
		RESOURCE="$2"
		shift 2
		;;
	--ltctl-bin)
		LTCTL_BIN="$2"
		shift 2
		;;
	--token-label)
		TOKEN_LABEL="$2"
		shift 2
		;;
	--pin)
		PIN="$2"
		shift 2
		;;
	--kek-label)
		KEK_LABEL="$2"
		shift 2
		;;
	--plugin-id)
		PLUGIN_ID="$2"
		shift 2
		;;
	--plugin-tarball)
		PLUGIN_TARBALL="$2"
		shift 2
		;;
	--service-user)
		SERVICE_USER="$2"
		shift 2
		;;
	--admin-email)
		ADMIN_EMAIL="$2"
		shift 2
		;;
	--admin-password)
		ADMIN_PASSWORD="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

if [ -z "$CONFIG" ]; then
	echo "error: -c/--config is required" >&2
	usage >&2
	exit 2
fi
if [ ! -f "$CONFIG" ]; then
	echo "error: config file not found: $CONFIG" >&2
	exit 2
fi
if [ -z "$PLUGIN_TARBALL" ]; then
	PLUGIN_TARBALL="/home/${SERVICE_USER}/${PLUGIN_ID}.tar.gz"
fi
if [ -z "$RESOURCE" ]; then
	CLUSTER_NAME="$(jq -r '.ClusterName' "$CONFIG")"
	if [ -z "$CLUSTER_NAME" ] || [ "$CLUSTER_NAME" = "null" ]; then
		echo "error: could not read .ClusterName from $CONFIG to derive a default -r; pass -r explicitly" >&2
		exit 2
	fi
	RESOURCE="${CLUSTER_NAME}-app-0"
fi

if [ -z "$LTCTL_BIN" ] && [ -x "${HOME}/cp/ltctl" ]; then
	LTCTL_BIN="${HOME}/cp/ltctl"
fi
if [ -n "$LTCTL_BIN" ]; then
	if [ ! -x "$LTCTL_BIN" ]; then
		echo "error: ltctl binary not found or not executable: $LTCTL_BIN" >&2
		exit 2
	fi
	LTCTL=("$LTCTL_BIN")
elif command -v ltctl >/dev/null 2>&1; then
	LTCTL=(ltctl)
else
	echo "warning: no ltctl binary found (checked \$LTCTL_BIN, ~/cp/ltctl, PATH) -- falling back to" \
		"'go run ./cmd/ltctl'. This will NOT pick up an unbuilt local change the way a rebuilt" \
		"~/cp/ltctl would; if cmd/ltctl changed, prefer: go build -o ~/cp/ltctl ./cmd/ltctl" >&2
	LTCTL=(go run "${REPO_ROOT}/cmd/ltctl")
fi

echo "=== MBE bring-up: resource=${RESOURCE} config=${CONFIG} ltctl=${LTCTL[*]} ==="

# `ltctl ssh --run` shells out to the real `ssh` binary (unlike ltctl's own internal SSH client,
# which skips host-key checking), so it fails outright against a freshly created node whose host
# key isn't already in known_hosts. Auto-accept it here: the node was just created seconds ago by
# this same operator/AWS account, so TOFU risk is negligible, and this is what a human running the
# equivalent interactive `ltctl ssh <resource>` once would do anyway (type "yes").
APP_IP=$("${LTCTL[@]}" deployment info -c "$CONFIG" 2>/dev/null | grep -F "${RESOURCE}:" | awk '{print $NF}')
if [ -n "$APP_IP" ]; then
	ssh-keyscan -H "$APP_IP" >>"${HOME}/.ssh/known_hosts" 2>/dev/null || true
fi

{
	printf "export TOKEN_LABEL=%q\n" "$TOKEN_LABEL"
	printf "export PIN=%q\n" "$PIN"
	printf "export KEK_LABEL=%q\n" "$KEK_LABEL"
	printf "export PLUGIN_ID=%q\n" "$PLUGIN_ID"
	printf "export PLUGIN_TARBALL=%q\n" "$PLUGIN_TARBALL"
	printf "export SERVICE_USER=%q\n" "$SERVICE_USER"
	printf "export ADMIN_EMAIL=%q\n" "$ADMIN_EMAIL"
	printf "export ADMIN_PASSWORD=%q\n" "$ADMIN_PASSWORD"
	cat "$REMOTE_SCRIPT"
} | "${LTCTL[@]}" ssh "$RESOURCE" -c "$CONFIG" --run "sudo bash -s"
