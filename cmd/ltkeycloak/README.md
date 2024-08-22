# ltkeycloak

Allows operations to interact between a load test environment and a Keycloak server.

## Usage

```bash
ltkeycloak [command] [flags]
```

## Commands

### `sync from_mattermost`

This command will sync users from a Mattermost server to a Keycloak server, setting the same password for all users and creating a txt file to be used with the loadtest.

```
Flags:
      --keycloak-realm string         The Keycloak realm to migrate users to (default "master")
      --set-user-password-to string   Set's the user password to the provided value (default "testpassword")

Global Flags:
  -c, --config string            path to the deployer configuration file to use
      --dry-run                  perform a dry run without making any changes
      --keycloak-host string     keycloak host (default "http://localhost:8484")
      --mattermost-host string   The Mattermost host to migrate users from
```

#### Example

Migrating users from a local Mattermost server to a local Keycloak server:

> **NOTE**: This still makes use of the `config/deployer.json` file to get the Mattermost system admin credentials.

```bash
$ ltkeycloak sync \
    -c config_local/config.json \
    from_mattermost \
    --keycloak-realm mattermost \
    --keycloak-host http://localhost:8484 \
    --mattermost-host localhost:8065
{"level":"info","msg":"fetching mattermost users","fields":{"page":"0","per_page":"100"}}
{"level":"info","msg":"migrated user","fields":{"username":"fmartingr"}}
{"level":"info","msg":"migrated user","fields":{"username":"sysadmin"}}
...
{"level":"info","msg":"migrated user","fields":{"username":"testuser-999"}}
{"level":"info","msg":"migration finished","fields":{"duration":"28.323263334s"}}
```
