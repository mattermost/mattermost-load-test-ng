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
    "InstanceCount": 1,
    "DevelopmentMode": true,
    "KeycloakAdminUser": "mmadmin",
    "KeycloakAdminPassword": "mmpass",
    "KeycloakRealmFilePath": "",
    "GenerateUsersCount": 0,
    "InstanceType": "t3.medium",
    "DatabaseInstanceCount": 0,
    "DatabaseInstanceType": "db.t3.medium",
    "DatabaseInstanceEngine": "aurora-postgresql",
    "DatabaseUsername": "mmuser",
    "DatabasePassword": "mmpassword",
    "DatabaseParameters": []
  },
  // ...
}
```

See the [reference code in the deployment/config.go file](../deployment/config.go#L188).

- **InstanceCount**: The number of instances to deploy. (`0` or `1`, `0` disables the deployment of the Keycloak server)
- **DevelopmentMode**: Whether to deploy the server in development mode. This changes the command used to start the server from `start` (production) to `start-dev` (development) and disables the usage of an external database.
- **KeycloakVersion**: The version of Keycloak to deploy.
- **KeycloakAdminUser**: The username of the Keycloak admin user.
- **KeycloakAdminPassword**: The password of the Keycloak admin user.
- **KeycloakRealmFilePath**: The path to a Keycloak realm file to use as import data.
  - If empty the load test will import a default one.
- **GenerateUsersCount**: The number of users to generate in the Keycloak server, if `0` no users will be generated.
- **InstanceType**: The instance type to use for the keycloak server.
- **DatabaseInstanceCount**: The number of database instances to deploy. This defaults to `0` if `DevelopmentMode` is set to `true`.
- **DatabaseInstanceType**: The instance type to use for the database.
- **DatabaseInstanceEngine**: The database engine to use.
- **DatabaseUsername**: The username to use for the database.
- **DatabasePassword**: The password to use for the database.
- **DatabaseParameters**: Additional parameters to use for the database.

## Enabling the Keycloak server

In order to enable the deployment of the Keycloak server (and configuration of the Mattermost instance to go along with it) you only need to set the `ExernalAuthProviderSettings.InstanceCount` section to `1` in the deployer configuration.

## The keycloak realm

The Keycloak server uses a realm to manage users and applications. A realm is a container for users, applications, and groups. It is an isolated space where applications authenticate users and manage their security credentials, including password policies, user roles, and social logins.

- If you want to use a custom realm file, you can upload it to the Keycloak server by setting the `KeycloakRealmFilePath` configuration option to the path of the file.

- If this option is left empty, the load-test tool will use a default realm file with the following usable credentials:
  - To log in in mattermost: `keycloak-user-01` as username and password.
  - To log in into the Keycloak admin interface: `mmadmin`/`mmpass`.

## Generating users

> **WARNING**: Generating users is usually really slow, if you plan to use more than a couple hundred users you should consider using a custom realm file.

The `GenerateUsersCount` configuration option allows you to generate a number of users in the Keycloak server. This is useful when you want to test the load-test tool with a large number of users.

This option will override the `UsersConfiguration.UserFilePath` option with the path to a file containing the generated users.

## Development mode

The `DevelopmentMode` configuration option allows you to deploy the Keycloak server in development mode. This changes the command used to start the server from `start` (production) to `start-dev` (development) and disables the usage of an external database.

This is useful when you want to test the load-test tool with a small number of users and don't want to deploy a database.

## Production mode

Not supported yet.
