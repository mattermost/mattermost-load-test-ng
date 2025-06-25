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

# Create base configuration LDIF
cat << EOF > /tmp/base_config.ldif
dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcSuffix
olcSuffix: ${LDAP_BASE_DN}

dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcRootDN
olcRootDN: ${LDAP_ADMIN_DN}

dn: olcDatabase={2}mdb,cn=config
changetype: modify
replace: olcRootPW
olcRootPW: ${LDAP_ADMIN_PASSWORD_HASH}
EOF

# Apply base configuration
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/base_config.ldif

# Create base DN structure
cat << EOF > /tmp/base_structure.ldif
dn: ${LDAP_BASE_DN}
objectClass: top
objectClass: dcObject
objectClass: organization
o: ${LDAP_ORGANIZATION}
dc: mm

dn: ou=users,${LDAP_BASE_DN}
objectClass: organizationalUnit
ou: users

dn: ou=groups,${LDAP_BASE_DN}
objectClass: organizationalUnit
ou: groups
description: Container for group accounts
EOF

# Add base structure
sudo ldapadd -x -D "${LDAP_ADMIN_DN}" -w "${LDAP_ADMIN_PASSWORD}" -f /tmp/base_structure.ldif

# Create sample LDIF file for test users
cat << EOF > /tmp/test_users.ldif
# Mattermost Users LDIF Export
# Generated on Mon Jun 23 08:02:05 PM IST 2025

dn: uid=testuser-1,ou=users,${LDAP_BASE_DN}
objectClass: inetOrgPerson
objectClass: person
objectClass: top
uid: testuser-1
cn: testuser-1
sn: User
mail: testuser-1@mattermost.com
userPassword: testPass123$

dn: cn=developers,ou=groups,${LDAP_BASE_DN}
objectClass: groupOfNames
cn: developers
description: Development team
member: uid=testuser-1,ou=users,${LDAP_BASE_DN}
EOF

# Add test data to LDAP
sudo ldapadd -x -D "${LDAP_ADMIN_DN}" -w "${LDAP_ADMIN_PASSWORD}" -f /tmp/test_users.ldif

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