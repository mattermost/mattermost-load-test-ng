#!/bin/bash
set -e -o pipefail

source /tmp/common.sh

# Install OpenLDAP server and utilities
sudo dnf update -y
sudo dnf install -y \
    openldap-servers \
    openldap-clients \
    openldap \
    migrationtools

# Stop slapd if running
sudo systemctl stop slapd || true

# Set up basic configuration
LDAP_DOMAIN="mm.test.com"
LDAP_ORGANIZATION="Mattermost Load Test"
LDAP_ADMIN_PASSWORD="mostest"
LDAP_BASE_DN="dc=mm,dc=test,dc=com"
LDAP_ADMIN_DN="cn=admin,dc=mm,dc=test,dc=com"

# Generate password hash
LDAP_ADMIN_PASSWORD_HASH=$(slappasswd -s "${LDAP_ADMIN_PASSWORD}")

# Copy default database configuration
sudo cp /usr/share/openldap-servers/DB_CONFIG.example /var/lib/ldap/DB_CONFIG
sudo chown ldap:ldap /var/lib/ldap/DB_CONFIG

# Set file permissions
sudo chown -R ldap:ldap /var/lib/ldap/
sudo chmod 700 /var/lib/ldap/

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

# Configure firewall
sudo firewall-cmd --permanent --add-port=389/tcp
sudo firewall-cmd --permanent --add-port=636/tcp
sudo firewall-cmd --reload

echo "OpenLDAP installation and configuration completed successfully!"
echo "Configuration details saved to /home/${USER}/openldap_config.txt"
