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

cd /tmp
/tmp/provisioner.sh

touch /var/lib/cloud/instance/provisioning-done
