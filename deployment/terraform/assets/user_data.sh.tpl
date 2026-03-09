#!/bin/bash
set -euo pipefail

cat > /tmp/common.sh << 'COMMONEOF'
${common_sh}
COMMONEOF
chmod +x /tmp/common.sh

cat > /tmp/provisioner.sh << 'PROVEOF'
${provisioner_sh}
PROVEOF
chmod +x /tmp/provisioner.sh

# Cloud-init runs user_data as root with a minimal environment.
# Set HOME to the AMI user's home directory since provisioner scripts
# (e.g. nvm) install to $HOME and other code expects files there.
export HOME="/home/${ami_user}"
export USER="${ami_user}"

cd /tmp
if /tmp/provisioner.sh; then
  touch /var/lib/cloud/instance/provisioning-done
else
  echo "$?" > /var/lib/cloud/instance/provisioning-exitcode
  exit 1
fi
