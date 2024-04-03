# External authentication providers

## Introduction

External authentication providers are used to authenticate users against an external system. This is useful when you want to use an existing authentication system, such as LDAP, to authenticate users in your application.

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

## Enabling the OpenID Connect provider

In order to enable the deployment of the Keycloak server (and configuration of the Mattermost instance to go along with it) you only need to provide the raise the `ExernalAuthProviderSettings.InstanceCount` section to `1` in the deployer configuration.
