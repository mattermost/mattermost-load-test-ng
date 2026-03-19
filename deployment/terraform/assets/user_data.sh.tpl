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
# Use runuser -l to execute the provisioner as the AMI user with a full
# login shell so that tools like nvm (sourced via ~/.bashrc) are available.
cd /tmp
rc=0
runuser -l "${ami_user}" -c /tmp/provisioner.sh || rc=$?
if [ $rc -eq 0 ]; then
  touch /var/lib/cloud/instance/provisioning-done
else
  echo "$rc" > /var/lib/cloud/instance/provisioning-exitcode
  exit 1
fi
