# Adding functionality coverage

## Introduction

The goal of this document is to help Mattermost developers get acquainted with the relevant implementation details of  [`mattermost-load-test-ng`](https://github.com/mattermost/mattermost-load-test-ng) and to understand how to add load-test coverage for new or missing functionality.

## Implementation overview

This is a quick overview of the main components involved when it comes to adding new functionality to the load-test tool.

### High level diagram

![lt_dev](https://user-images.githubusercontent.com/1832946/112990833-c6a00680-9166-11eb-9442-4437e918a649.png)

### `UserController`

This is the component in charge of executing a user. It's an interface that permits different implementations based on the load-test needs. 

The default controller, and the one you should be usually concerned with when adding new functionality, is the [`SimulController`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/controller.go#L18) . Its implementation attempts to mimic a real user as closely as possible.

The controller's primary method is [`Run()`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/controller.go#L60) and it's a simple loop over some (user) actions that get executed based on a realistic set of frequencies.

Each action defines a user behaviour. These are some examples:

- [`login`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/actions.go#L124)
- [`createPost`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/actions.go#L514)
- [`switchChannel`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/actions.go#L413)
- [`viewChannel`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/actions.go#L283)

Depending on the functionality you are looking to add, you'd be either extending existing actions or creating new ones. 

## `User`

[`User`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/user/user.go#L20) is the interface that provides access to the underlying HTTP/WebSocket API. 

It mostly exports similar methods to the official Go driver but at the same time it handles saving API returned data into the store. Its main implementation is [`UserEntity`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/user/userentity/user.go#L21).

## `UserStore`

Each controlled user has its own state which holds all the data needed for the user to operate (teams, channels, posts, etc.). This can be accessed through an interface called [`UserStore`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/store/store.go#L28). Its primary implementation is [`MemStore`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/store/memstore/store.go#L18).

#### Note

From the controller's perspective the store is immutable. Any action altering the store needs to happen at the `User` layer.

## Development steps

This is a list of common steps needed when adding new or missing load-test coverage. Depending on the planned changes, some of these steps may not be needed.

### Update server dependency

If the added feature has new client (as in [`model/client4.go`](https://github.com/mattermost/mattermost-server/blob/master/model/client4.go)) additions, you should update the `mattermost-server` dependency so that the new methods can be used from within the load-test packages.

```sh
go get -u github.com/mattermost/mattermost-server/v6/@COMMIT_HASH
go mod tidy
```

### Update store

Add/extend required `MemStore` methods.

Most of the exported methods that can serve as good examples can be found in [`loadtest/store/memstore/store.go`](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/store/memstore/store.go)

If you implement new methods you should also update the [`UserStore`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/store/store.go#L28) interface accordingly.

### Update user

Add/extend required `UserEntity` methods.

Most of the exported methods that can serve as good examples can be found in [`loadtest/user/userentity/actions.go`](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/user/userentity/actions.go).

If you implement new methods you should also update the [`User`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/user/user.go#L20) interface accordingly.

### Update controller

Add/extend required `SimulController` actions.

Most of the existing actions that can serve as good examples can be found in [`loadtest/control/simulcontroller/actions.go`](https://github.com/mattermost/mattermost-load-test-ng/blob/master/loadtest/control/simulcontroller/actions.go).

If a new action is implemented make sure to add it to the [list of executed actions](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/controller.go#L106).

#### Note

If an added action only works on newer server versions make sure to set the `minVersion` field in [`userAction`](https://github.com/mattermost/mattermost-load-test-ng/blob/c76063def0b36d61c0467e18357cf4cca969fe8a/loadtest/control/simulcontroller/actions.go#L23) to the minimum supported server version.

## Testing

Currently we don't provide a way to test controller's actions. However tests are required when adding user agnostic logic (e.g. utility functions).

Before committing changes please make sure both code style/linting checks and tests are passing successfully.

```sh
make check-style && make test
```

## Code Review

When submitting your pull-request please make sure to include at least one reviewer among the code owners ([@agnivade](https://github.com/agnivade), [@isacikgoz](https://github.com/isacikgoz), [@streamer45](https://github.com/streamer45)).

## Help

If you still have questions, doubts or just need further help you reach out to us in the [~Developer: Performance](https://community.mattermost.com/core/channels/developers-performance) channel.
