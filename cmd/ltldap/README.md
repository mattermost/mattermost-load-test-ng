# ltldap

A utility for managing OpenLDAP data in Mattermost load test deployments.

## Overview

The `ltldap` tool helps generate and import user data into OpenLDAP servers deployed as part of Mattermost load test environments. It reads configuration from the deployment config and can automatically detect the LDAP server from Terraform output.

## Commands

### Generate Users

Generate LDIF files with test users:

```bash
ltldap generate users <start-index> <end-index> --deployer-config <config-file> [flags]
```

**Examples:**

```bash
# Generate users testuser-1 to testuser-100
ltldap generate users 1 100 --deployer-config deployer.json

# Generate users and import them to LDAP server
ltldap generate users 1 10 --deployer-config deployer.json --import

# Generate users with custom password and output file
ltldap generate users 1 20 --deployer-config deployer.json --user-password "mypass123" --output-file custom_users.ldif --import
```

**Flags:**
- `--deployer-config`: Path to the deployer configuration file (required)
- `--user-password`: Password for all generated users (default: "testPass123$")
- `--output-file`: Output LDIF file name (default: "users_<start>_<end>.ldif")
- `--import`: Import the generated LDIF to LDAP server

## Configuration

The tool reads LDAP connection settings from the deployment configuration file:

- `OpenLDAPSettings.BaseDN`: Base DN for LDAP operations
- `OpenLDAPSettings.BindUsername`: Username for LDAP authentication
- `OpenLDAPSettings.BindPassword`: Password for LDAP authentication

## Generated Users

Each generated user has the following structure:

```ldif
dn: uid=testuser-N,ou=users,dc=mm,dc=test,dc=com
objectClass: inetOrgPerson
objectClass: person
objectClass: top
uid: testuser-N
cn: testuser-N
sn: User
mail: testuser-N@mattermost.com
userPassword: testPass123$
```

All users are automatically added to a "developers" group:

```ldif
dn: cn=developers,ou=groups,dc=mm,dc=test,dc=com
objectClass: groupOfNames
cn: developers
description: Development team
member: uid=testuser-1,ou=users,dc=mm,dc=test,dc=com
member: uid=testuser-2,ou=users,dc=mm,dc=test,dc=com
# ... more members
```

## Prerequisites

For importing LDIF files to LDAP, the `ldapadd` command must be available on the system. This is typically provided by the `ldap-utils` package:

```bash
# Ubuntu/Debian
sudo apt-get install ldap-utils

# CentOS/RHEL
sudo yum install openldap-clients
```

## Integration with Load Test

The generated users can be used with Mattermost's LDAP authentication. The deployment automatically configures Mattermost to use the OpenLDAP server when `OpenLDAPSettings.Enabled = true` in the deployment config.