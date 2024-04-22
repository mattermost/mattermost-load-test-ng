# External authentication providers

## Introduction

External authentication providers are used to authenticate users against an external system. This is useful when you want to use an existing authentication system, such as LDAP, to authenticate users in your application and avoid the performance hit of managing (login in) users in the Mattermost server.

In the case of the load-test tool, a Keycloak server is used as the authentication provider. Keycloak is an open-source identity and access management solution that provides a way to authenticate users against an external system.

> **The load-test currently only supports OpenID Connect as an external authentication provider.**

## Configuration options

``` js
{
   // ...
  "ExternalAuthProviderSettings": {
    "Enabled": true,
    "KeycloakAdminUser": "mmadmin",
    "KeycloakAdminPassword": "mmpass",
    "KeycloakRealmFilePath": "",
    "KeycloakDBDumpURI": "",
    "GenerateUsersCount": 0,
    "InstanceType": "t3.medium",
  },
  // ...
}
```

See the [reference code in the deployment/config.go file](../deployment/config.go#L188).

- **Enabled**: Whether to enable the deployment of the Keycloak server.
- **KeycloakVersion**: The version of Keycloak to deploy.
- **KeycloakAdminUser**: The username of the Keycloak admin user.
- **KeycloakAdminPassword**: The password of the Keycloak admin user.
- **KeycloakRealmFilePath**: The path to a Keycloak realm file to use as import data.
  - If empty the load test will import a default one.
  -  See the [The keycloak realm](#the-keycloak-realm) section for more information.
- **KeycloakDBDumpURI**: The URI of a database dump to use as import data.
  - See the [Importing a database dump](#importing-a-database-dump) section for more information.
- **GenerateUsersCount**: The number of users to generate in the Keycloak server, if `0` no users will be generated.
  - See the [Generating users](#generating-users) section for more information.
- **InstanceType**: The instance type to use for the keycloak server.

## Enabling the Keycloak server

In order to enable the deployment of the Keycloak server (and configuration of the Mattermost instance to go along with it) you only need to set the `ExernalAuthProviderSettings.Enabled` setting to `true` in the deployer configuration.

## The keycloak realm

The Keycloak server uses a realm to manage users and applications. A realm is a container for users, applications, and groups. It is an isolated space where applications authenticate users and manage their security credentials, including password policies, user roles, and social logins.

- If you want to use a custom realm file, you can upload it to the Keycloak server by setting the `KeycloakRealmFilePath` configuration option to the path of the file.

- If this option is left empty, the load-test tool will use a default realm file with the following usable credentials:
  - To log in in mattermost: `keycloak-user-01`/`keycloak-user-01`.
  - To log in into the Keycloak admin interface: `mmadmin`/`mmpass`.

## Importing a database dump

The `KeycloakDBDumpURI` configuration option allows you to import a database dump into the Keycloak server. This is useful when you want to use a database dump from a previous Keycloak server deployment.

It should be a `.tgz` compressed file containing the database dump as a `.sql` file. The SQL file should be a full dump of the Keycloak database since no initialiaztion will be done when this parameter is set.

This option allows the use of an URI (can be `http://`, `https://`, or `file://`) to a database dump file.

## Generating users

> **WARNING**: Generating users is usually really slow, if you plan to use more than a couple hundred users you should consider using a custom realm file.

The `GenerateUsersCount` configuration option allows you to generate a number of users in the Keycloak server. This is useful when you want to test the load-test tool with a small number of users.

This option will override the `UsersConfiguration.UserFilePath` option with the path to a file containing the generated users.

## Development mode

The `DevelopmentMode` configuration option allows you to deploy the Keycloak server in development mode. This changes the command used to start the server from `start` (production) to `start-dev` (development) so Keycloak [disables several features](https://www.keycloak.org/server/configuration#_starting_keycloak_in_development_mode) to ease up the environment creation process.
