# Mattermost load-test-ng

Mattermost load-test-ng provides a set of tools written in [Go](https://golang.org/) to help profiling [Mattermost](https://github.com/mattermost/mattermost-server) under heavy load, simulating real-world usage of a server installation at scale.

It's a complete rewrite of the [previous](https://github.com/mattermost/mattermost-load-test) load-test tool which served as inspiration.

## Goals

- Give an estimate on the maximum number of concurrently active users the target system supports.
- Enable more control over the load to generate through the use of [Controllers](docs/controllers.md).
- Provide extensive documentation from lower level code details to higher level guides and walk-throughs.

## How to use

There are mainly two ways to run a load-test:

- On a Terraform deployment. This is the recommended way to start a load-test for production. [Link to the guide](docs/terraform_loadtest.md)
- Locally. This is a good way to getting started and better understand the inner mechanics. [Link to the guide](docs/local_loadtest.md)

See the [user workflow guide](docs/load-test-how-to-use.md) for details.

## FAQ

Answers to frequently asked questions and troubleshooting for common issues can be found in the [FAQ](docs/faq.md) section.

## Documentation

Documentation and implementation details can be found in the [docs](docs/) folder.  
Code specific documentation can be found on [GoDoc](https://godoc.org/github.com/mattermost/mattermost-load-test-ng).

## Development

Information about the development workflow and release process can be found in [Developer's workflow](docs/developing.md).  
A guide on how to add load-test coverage for new or missing functionality can be found in [Adding functionality](docs/coverage.md).  

## Help

If you need any help you can join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) channel and ask developers any question related to this project.

