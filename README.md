# Mattermost load-test-ng

Mattermost load-test-ng provides a set of tools written in [Go](https://golang.org/) to help profiling [Mattermost](https://github.com/mattermost/mattermost) under heavy load, simulating real-world usage of a server installation at scale.

It's a complete rewrite of the [previous](https://github.com/mattermost/mattermost-load-test) load-test tool which served as inspiration.

## Goals

- Give an estimate on the maximum number of concurrently active users the target system supports.
- Enable more control over the load to generate through the use of [Controllers](docs/controllers.md).
- Provide extensive documentation from lower level code details to higher level guides and walk-throughs.

## Documentation

Documentation and implementation details can be found in the [docs](docs/) folder.
Code specific documentation can be found on [GoDoc](https://godoc.org/github.com/mattermost/mattermost-load-test-ng).

## Help

If you need any help you can join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) channel and ask developers any question related to this project.

