#!/bin/bash
set -e -o pipefail

source /tmp/common.sh

# Install OpenLDAP server and utilities
sudo DEBIAN_FRONTEND=noninteractive apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
    slapd \
    ldap-utils \
    ldapscripts \
    ssl-cert \
    prometheus-node-exporter

# Configure OpenLDAP
sudo systemctl stop slapd

# Set up basic configuration
LDAP_DOMAIN="mm.test.com"
LDAP_ORGANIZATION="Mattermost Load Test"
LDAP_ADMIN_PASSWORD="mostest"
LDAP_BASE_DN="dc=mm,dc=test,dc=com"
LDAP_ADMIN_DN="cn=admin,dc=mm,dc=test,dc=com"

# Create slapd configuration using debconf-set-selections
# These settings will be used when reconfiguring slapd
cat << EOF | sudo debconf-set-selections
slapd slapd/internal/generated_adminpw password ${LDAP_ADMIN_PASSWORD}
slapd slapd/internal/adminpw password ${LDAP_ADMIN_PASSWORD}
slapd slapd/password2 password ${LDAP_ADMIN_PASSWORD}
slapd slapd/password1 password ${LDAP_ADMIN_PASSWORD}
slapd slapd/dump_database_destdir string /var/backups/slapd-VERSION
slapd slapd/domain string ${LDAP_DOMAIN}
slapd shared/organization string ${LDAP_ORGANIZATION}
slapd slapd/backend string MDB
slapd slapd/purge_database boolean true
slapd slapd/move_old_database boolean true
slapd slapd/allow_ldap_v2 boolean false
slapd slapd/no_configuration boolean false
slapd slapd/dump_database select when needed
EOF

# Reconfigure slapd with new settings
sudo dpkg-reconfigure -f noninteractive slapd

# Start and enable slapd
sudo systemctl start slapd
sudo systemctl enable slapd

# Create configuration file for easy reference
cat << EOF > /tmp/openldap_config.txt
LDAP Server Configuration:
- Domain: ${LDAP_DOMAIN}
- Base DN: ${LDAP_BASE_DN}
- Admin DN: ${LDAP_ADMIN_DN}
- Admin Password: ${LDAP_ADMIN_PASSWORD}
- LDAP URL: ldap://$(hostname -I | awk '{print $1}'):389
- LDAPS URL: ldaps://$(hostname -I | awk '{print $1}'):636

Test Users:
- testuser-1 / testPass123$

Test Commands:
- ldapsearch -x -H ldap://localhost -b "${LDAP_BASE_DN}" -D "${LDAP_ADMIN_DN}" -w "${LDAP_ADMIN_PASSWORD}"
- ldapsearch -x -H ldap://localhost -b "ou=users,${LDAP_BASE_DN}" "(objectClass=person)"
EOF

sudo mv /tmp/openldap_config.txt /home/${USER}/openldap_config.txt
sudo chown ${USER}:${USER} /home/${USER}/openldap_config.txt

echo "OpenLDAP installation and configuration completed successfully!"
echo "Configuration details saved to /home/${USER}/openldap_config.txt"
