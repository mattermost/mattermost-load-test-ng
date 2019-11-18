# Mattermost loadtest

Mattermost loadtest is a standalone tool written in [Golang](https://golang.org/) for profiling [Mattermost](https://github.com/mattermost/mattermost-server) under heavy load simulating real-world usage of a server installation at scale.

## Goals/Features 

- No external dependencies.
- Loosely coupled components.
- Short, *do one thing only* functions. 
- State handling out of main logic.
- Theoretically no need to bulkload.
- No need to synchronize state between multiple loadtesting instances.
- Easy to add/remove concurrent users at execution time.

## Running

`go run -v ./cmd/loadtest`

## Documentation

Documentation and implementation details can be found in the [docs](docs/) folder.

## Development

A sample implementation can be found in the [example](example/) folder.

## Help

If you need any help you can join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) channel and ask developers any question related to this project.
